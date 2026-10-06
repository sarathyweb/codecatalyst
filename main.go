package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/azure"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const azureOpenAIRequestTimeout = 24 * time.Hour

type config struct {
	APIKey          string                 `yaml:"azure_openai_api_key"`
	EndPoint        string                 `yaml:"azure_openai_endpoint"`
	Model           string                 `yaml:"azure_openai_model"`
	ReasoningMode   shared.ReasoningMode   `yaml:"azure_openai_reasoning_mode"`
	ReasoningEffort shared.ReasoningEffort `yaml:"azure_openai_reasoning_effort"`
	MultiAgent      bool                   `yaml:"azure_openai_multi_agent"`
	EmbeddingModel  string                 `yaml:"azure_openai_embedding_model"`
	DatabaseURL     string                 `yaml:"database_url"`
}

type runResult struct {
	Model string
	Usage responses.ResponseUsage
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

			result, err := run(configPath, args[0])
			elapsed := time.Since(startedAt)
			if err != nil {
				return fmt.Errorf("failed after %s: %w", formatDuration(elapsed), err)
			}

			printRunSummary(elapsed, result)
			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", defaultConfigPath, "Path to the config file")
	return cmd, nil
}

func run(configPath string, chatLogFile string) (runResult, error) {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return runResult{}, err
	}

	content, err := os.ReadFile(chatLogFile)
	if err != nil {
		return runResult{}, err
	}

	baseURL, err := azureOpenAIBaseURL(cfg.EndPoint)
	if err != nil {
		return runResult{}, err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = azureOpenAIRequestTimeout

	client := openai.NewClient(
		azure.WithAPIKey(cfg.APIKey),
		option.WithBaseURL(baseURL),
		option.WithHTTPClient(&http.Client{Transport: transport}),
	)

	requestOptions := make([]option.RequestOption, 0, 1)
	if cfg.MultiAgent {
		requestOptions = append(requestOptions, option.WithJSONSet("multi_agent.enabled", true))
	}

	ctx, cancel := context.WithTimeout(context.Background(), azureOpenAIRequestTimeout)
	defer cancel()

	resp, err := client.Responses.New(ctx, responses.ResponseNewParams{
		Model:     openai.ChatModel(cfg.Model),
		Input:     responses.ResponseNewParamsInputUnion{OfString: openai.String(string(content))},
		Reasoning: shared.ReasoningParam{Mode: cfg.ReasoningMode, Effort: cfg.ReasoningEffort},
	}, requestOptions...)
	if err != nil {
		return runResult{}, err
	}

	f, err := os.OpenFile(chatLogFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return runResult{}, err
	}
	defer f.Close()

	if _, err := f.WriteString("\nAI Assistant:\n" + resp.OutputText() + "\n"); err != nil {
		return runResult{}, err
	}

	model := string(resp.Model)
	if model == "" {
		model = cfg.Model
	}
	return runResult{Model: model, Usage: resp.Usage}, nil
}

func printRunSummary(elapsed time.Duration, result runResult) {
	cost := "cost unavailable"
	if totalCost, ok := estimateTokenCostUSD(result.Model, result.Usage); ok {
		cost = fmt.Sprintf("$%.4f", totalCost)
	}

	fmt.Printf("Done in %s | %d tokens | %s\n",
		formatDuration(elapsed), result.Usage.TotalTokens, cost)
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

	cfg := config{
		ReasoningMode:   shared.ReasoningModePro,
		ReasoningEffort: shared.ReasoningEffortXhigh,
		MultiAgent:      true,
	}
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

	if _, err := azureOpenAIBaseURL(cfg.EndPoint); err != nil {
		return fmt.Errorf("azure_openai_endpoint is invalid: %w", err)
	}

	if strings.TrimSpace(cfg.Model) == "" {
		return errors.New("azure_openai_model is required")
	}

	switch cfg.ReasoningMode {
	case shared.ReasoningModeStandard, shared.ReasoningModePro:
	default:
		return fmt.Errorf("azure_openai_reasoning_mode must be %q or %q", shared.ReasoningModeStandard, shared.ReasoningModePro)
	}

	switch cfg.ReasoningEffort {
	case shared.ReasoningEffortLow, shared.ReasoningEffortMedium, shared.ReasoningEffortHigh,
		shared.ReasoningEffortXhigh, shared.ReasoningEffortMax:
	default:
		return errors.New("azure_openai_reasoning_effort must be low, medium, high, xhigh, or max")
	}

	return nil
}

func azureOpenAIBaseURL(endpoint string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil {
		return "", err
	}

	if parsed.Scheme != "https" || parsed.Host == "" {
		return "", errors.New("must be an absolute HTTPS URL")
	}

	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("must not contain a query string or fragment")
	}

	path := strings.TrimRight(parsed.Path, "/")
	if !strings.HasSuffix(strings.ToLower(path), "/openai/v1") {
		path += "/openai/v1"
	}

	parsed.Path = path + "/"
	parsed.RawPath = ""
	return parsed.String(), nil
}

func defaultConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".codecatalyst.yaml"), nil
}
