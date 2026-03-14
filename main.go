package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/azure"
	"github.com/openai/openai-go/v3/responses"
)

type config struct {
	APIKey     string
	EndPoint   string
	APIVersion string
	Model      string
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: codecatalyst.exe <chat-log-file>")
	}

	cfg := loadConfig()
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
		log.Fatal(err)
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
		log.Fatal(err)
	}

	// 3. Append AI output
	f, err := os.OpenFile(chatLogFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, err = f.WriteString("\nAI Assistant:\n" + resp.OutputText() + "\n")
	if err != nil {
		log.Fatal(err)
	}
}

func loadConfig() config {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatalf("failed to load .env: %v", err)
	}

	return config{
		APIKey:     requiredEnv("AZURE_OPENAI_API_KEY"),
		EndPoint:   requiredEnv("AZURE_OPENAI_ENDPOINT"),
		APIVersion: requiredEnv("AZURE_OPENAI_API_VERSION"),
		Model:      requiredEnv("AZURE_OPENAI_MODEL"),
	}
}

func requiredEnv(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		log.Fatalf("%s is required", key)
	}

	return value
}
