package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/azure"
	"github.com/openai/openai-go/v3/responses"
)

const (
	inputTokenPricePerMillionUSD  = 30.00
	outputTokenPricePerMillionUSD = 180.00
)

type config struct {
	APIKey     string
	EndPoint   string
	APIVersion string
	Model      string
}

func main() {
	startedAt := time.Now()

	usage, err := run()
	elapsed := time.Since(startedAt)
	if err != nil {
		log.Printf("failed after %s: %v", formatDuration(elapsed), err)
		os.Exit(1)
	}

	printRunSummary(elapsed, usage)
}

func run() (responses.ResponseUsage, error) {
	if len(os.Args) < 2 {
		return responses.ResponseUsage{}, fmt.Errorf("usage: codecatalyst.exe <chat-log-file>")
	}

	cfg, err := loadConfig()
	if err != nil {
		return responses.ResponseUsage{}, err
	}

	chatLogFile := os.Args[1]

	// e := echo.New()
	// e.Use(middleware.RequestLogger())

	// e.GET("/", func(c *echo.Context) error {
	// 	return c.String(http.StatusOK, "Hello")
	// })
	// if err := e.Start(":5063"); err != nil {
	// 	e.Logger.Error("Failed to start server", "error", err)
	// }
	// 1. Read chat log
	content, err := os.ReadFile(chatLogFile)
	if err != nil {
		return responses.ResponseUsage{}, err
	}

	client := openai.NewClient(
		azure.WithAPIKey(cfg.APIKey),
		azure.WithEndpoint(cfg.EndPoint, cfg.APIVersion),
	)

	// 2. Call model
	resp, err := client.Responses.New(context.Background(), responses.ResponseNewParams{
		Model: openai.ChatModel(cfg.Model),
		Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(string(content))},
		// Instructions: openai.String(sm),
	})
	if err != nil {
		return responses.ResponseUsage{}, err
	}

	// 3. Append AI output
	f, err := os.OpenFile(chatLogFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return responses.ResponseUsage{}, err
	}
	defer f.Close()

	_, err = f.WriteString("\nAI Assistant:\n" + resp.OutputText() + "\n")
	if err != nil {
		return responses.ResponseUsage{}, err
	}

	return resp.Usage, nil
}

func printRunSummary(elapsed time.Duration, usage responses.ResponseUsage) {
	inputCost := tokenCostUSD(usage.InputTokens, inputTokenPricePerMillionUSD)
	outputCost := tokenCostUSD(usage.OutputTokens, outputTokenPricePerMillionUSD)
	totalCost := inputCost + outputCost

	fmt.Printf("Completed in %s\n", formatDuration(elapsed))
	fmt.Println("Usage summary:")
	fmt.Printf("  Input tokens: %d\n", usage.InputTokens)
	fmt.Printf("  Cached input tokens: %d\n", usage.InputTokensDetails.CachedTokens)
	fmt.Printf("  Output tokens: %d\n", usage.OutputTokens)
	fmt.Printf("  Reasoning tokens: %d\n", usage.OutputTokensDetails.ReasoningTokens)
	fmt.Printf("  Total tokens: %d\n", usage.TotalTokens)
	fmt.Println("Cost breakdown:")
	fmt.Printf("  Input: %d tokens x $%.2f / 1M = $%.6f\n", usage.InputTokens, inputTokenPricePerMillionUSD, inputCost)
	fmt.Printf("  Output: %d tokens x $%.2f / 1M = $%.6f\n", usage.OutputTokens, outputTokenPricePerMillionUSD, outputCost)
	fmt.Printf("  Total cost: $%.6f\n", totalCost)
}

func tokenCostUSD(tokens int64, pricePerMillionUSD float64) float64 {
	return (float64(tokens) / 1_000_000) * pricePerMillionUSD
}

func formatDuration(duration time.Duration) time.Duration {
	if duration < time.Millisecond {
		return duration
	}

	return duration.Round(time.Millisecond)
}

func loadConfig() (config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return config{}, fmt.Errorf("failed to load .env: %w", err)
	}

	apiKey, err := requiredEnv("AZURE_OPENAI_API_KEY")
	if err != nil {
		return config{}, err
	}

	endPoint, err := requiredEnv("AZURE_OPENAI_ENDPOINT")
	if err != nil {
		return config{}, err
	}

	apiVersion, err := requiredEnv("AZURE_OPENAI_API_VERSION")
	if err != nil {
		return config{}, err
	}

	model, err := requiredEnv("AZURE_OPENAI_MODEL")
	if err != nil {
		return config{}, err
	}

	return config{
		APIKey:     apiKey,
		EndPoint:   endPoint,
		APIVersion: apiVersion,
		Model:      model,
	}, nil
}

func requiredEnv(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", key)
	}

	return value, nil
}
