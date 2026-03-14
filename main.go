package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/azure"
	"github.com/openai/openai-go/v3/responses"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	inputTokenPricePerMillionUSD  = 30.00
	outputTokenPricePerMillionUSD = 180.00
)

type config struct {
	APIKey         string `yaml:"azure_openai_api_key"`
	EndPoint       string `yaml:"azure_openai_endpoint"`
	APIVersion     string `yaml:"azure_openai_api_version"`
	Model          string `yaml:"azure_openai_model"`
	EmbeddingModel string `yaml:"azure_openai_embedding_model"`
	DatabaseURL    string `yaml:"database_url"`
}

func main() {
	cmd, err := newRootCmd()
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	if err := cmd.Execute(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func newRootCmd() (*cobra.Command, error) {
	defaultConfigPath, err := defaultConfigPath()
	if err != nil {
		return nil, fmt.Errorf("resolve default config path: %w", err)
	}

	var configPath string

	cmd := &cobra.Command{
		Use:           "codecatalyst <chat-log-file>",
		Short:         "Send a chat log to Azure OpenAI and append the response",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			startedAt := time.Now()

			usage, err := run(configPath, args[0])
			elapsed := time.Since(startedAt)
			if err != nil {
				return fmt.Errorf("failed after %s: %w", formatDuration(elapsed), err)
			}

			printRunSummary(elapsed, usage)
			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", defaultConfigPath, "Path to the config file")
	return cmd, nil
}

func run(configPath string, chatLogFile string) (responses.ResponseUsage, error) {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return responses.ResponseUsage{}, err
	}

	content, err := os.ReadFile(chatLogFile)
	if err != nil {
		return responses.ResponseUsage{}, err
	}

	client := openai.NewClient(
		azure.WithAPIKey(cfg.APIKey),
		azure.WithEndpoint(cfg.EndPoint, cfg.APIVersion),
	)

	resp, err := client.Responses.New(context.Background(), responses.ResponseNewParams{
		Model: openai.ChatModel(cfg.Model),
		Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(string(content))},
	})
	if err != nil {
		return responses.ResponseUsage{}, err
	}

	f, err := os.OpenFile(chatLogFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return responses.ResponseUsage{}, err
	}
	defer f.Close()

	if _, err := f.WriteString("\nAI Assistant:\n" + resp.OutputText() + "\n"); err != nil {
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

func loadConfig(configPath string) (config, error) {
	configPath = filepath.Clean(strings.TrimSpace(configPath))
	if configPath == "" {
		return config{}, errors.New("config path is required")
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config{}, fmt.Errorf("config file not found: %s", configPath)
		}

		return config{}, err
	}

	var cfg config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return config{}, fmt.Errorf("decode config file %s: %w", configPath, err)
	}

	if err := cfg.validate(); err != nil {
		return config{}, fmt.Errorf("invalid config file %s: %w", configPath, err)
	}

	return cfg, nil
}

func (cfg config) validate() error {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return errors.New("azure_openai_api_key is required")
	}

	if strings.TrimSpace(cfg.EndPoint) == "" {
		return errors.New("azure_openai_endpoint is required")
	}

	if strings.TrimSpace(cfg.APIVersion) == "" {
		return errors.New("azure_openai_api_version is required")
	}

	if strings.TrimSpace(cfg.Model) == "" {
		return errors.New("azure_openai_model is required")
	}

	return nil
}

func defaultConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".codecatalyst.yaml"), nil
}
