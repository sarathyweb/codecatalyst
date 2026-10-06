package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAzureModels(t *testing.T) {
	for _, tt := range []struct {
		name          string
		deployment    string
		responseModel string
		settings      string
		wantModel     string
		wantMode      string
		wantEffort    string
		wantMulti     bool
	}{
		{
			name: "astra defaults", deployment: "gpt-6-astra", responseModel: "gpt-6-astra",
			wantModel: "gpt-6-astra", wantMode: "pro", wantEffort: "xhigh", wantMulti: true,
		},
		{
			name: "astra custom deployment", deployment: "coding-production", responseModel: "gpt-6-astra",
			wantModel: "gpt-6-astra", wantMode: "pro", wantEffort: "xhigh", wantMulti: true,
		},
		{
			name: "astra standard single agent", deployment: "gpt-6-astra", responseModel: "gpt-6-astra",
			settings:  "azure_openai_reasoning_mode: standard\nazure_openai_multi_agent: false\n",
			wantModel: "gpt-6-astra", wantMode: "standard", wantEffort: "xhigh",
		},
		{
			name: "astra effort override", deployment: "gpt-6-astra", responseModel: "gpt-6-astra",
			settings:  "azure_openai_reasoning_effort: high\n",
			wantModel: "gpt-6-astra", wantMode: "pro", wantEffort: "high", wantMulti: true,
		},
		{
			name: "legacy sol", deployment: "gpt-5.6-sol", responseModel: "gpt-5.6-sol",
			wantModel: "gpt-5.6-sol", wantMode: "pro", wantEffort: "xhigh", wantMulti: true,
		},
		{
			name: "missing response model", deployment: "gpt-6-astra",
			wantModel: "gpt-6-astra", wantMode: "pro", wantEffort: "xhigh", wantMulti: true,
		},
		{
			name: "unrecognized response model", deployment: "custom-deployment", responseModel: "unknown-model",
			wantModel: "unknown-model", wantMode: "pro", wantEffort: "xhigh", wantMulti: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			const input = "User:\nReview this code.\n"
			const output = "Here is the review."
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/openai/v1/responses" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("Api-Key") != "test-api-key" {
					t.Error("missing Azure API key")
				}
				var body struct {
					Model     string `json:"model"`
					Input     string `json:"input"`
					Reasoning struct {
						Mode   string `json:"mode"`
						Effort string `json:"effort"`
					} `json:"reasoning"`
					MultiAgent *struct {
						Enabled bool `json:"enabled"`
					} `json:"multi_agent"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode request: %v", err)
					http.Error(w, "bad request", http.StatusBadRequest)
					return
				}
				if body.Model != tt.deployment || body.Input != input || body.Reasoning.Mode != tt.wantMode {
					t.Errorf("unexpected request body: %+v", body)
				}
				if body.Reasoning.Effort != tt.wantEffort {
					t.Errorf("reasoning effort = %q, want %q", body.Reasoning.Effort, tt.wantEffort)
				}
				if tt.wantMulti {
					if body.MultiAgent == nil || !body.MultiAgent.Enabled {
						t.Error("multi-agent orchestration was not enabled")
					}
				} else if body.MultiAgent != nil {
					t.Error("disabled multi-agent field should be omitted")
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{
					"id": "resp_test", "object": "response", "status": "completed", "model": %q,
					"output": [{"type": "message", "role": "assistant", "status": "completed",
						"content": [{"type": "output_text", "text": %q, "annotations": []}]}],
					"usage": {"input_tokens": 100, "output_tokens": 20, "total_tokens": 120,
						"input_tokens_details": {"cached_tokens": 30, "cache_write_tokens": 40},
						"output_tokens_details": {"reasoning_tokens": 10}}
				}`, tt.responseModel, output)
			}))
			t.Cleanup(server.Close)
			useTestTransport(t, server)

			configPath, chatPath := writeRunFiles(t, server.URL, tt.deployment, tt.settings, input)
			result, err := run(configPath, chatPath)
			if err != nil {
				t.Fatal(err)
			}
			if result.Model != tt.wantModel {
				t.Errorf("model = %q, want %q", result.Model, tt.wantModel)
			}
			if result.Usage.InputTokens != 100 || result.Usage.OutputTokens != 20 || result.Usage.TotalTokens != 120 ||
				result.Usage.InputTokensDetails.CachedTokens != 30 || result.Usage.InputTokensDetails.CacheWriteTokens != 40 {
				t.Errorf("unexpected usage: %+v", result.Usage)
			}
			content, err := os.ReadFile(chatPath)
			if err != nil {
				t.Fatal(err)
			}
			if want := input + "\nAI Assistant:\n" + output + "\n"; string(content) != want {
				t.Errorf("chat log = %q, want %q", content, want)
			}
		})
	}
}

func TestLoadConfigRejectsInvalidReasoningEffort(t *testing.T) {
	for _, effort := range []string{"pro", "none", ""} {
		t.Run(effort, func(t *testing.T) {
			settings := fmt.Sprintf("azure_openai_reasoning_effort: %q\n", effort)
			configPath, _ := writeRunFiles(t, "https://example.openai.azure.com", "gpt-6-astra", settings, "")
			if _, err := loadConfig(configPath); err == nil || !strings.Contains(err.Error(), "azure_openai_reasoning_effort") {
				t.Fatalf("expected reasoning effort validation error, got %v", err)
			}
		})
	}
}

func TestRunAPIErrorPreservesChatLog(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"message":"Deployment not available","type":"invalid_request_error"}}`)
	}))
	t.Cleanup(server.Close)
	useTestTransport(t, server)

	const input = "User:\nReview this code.\n"
	configPath, chatPath := writeRunFiles(t, server.URL, "gpt-6-astra", "", input)
	if _, err := run(configPath, chatPath); err == nil {
		t.Fatal("expected API error")
	}
	content, err := os.ReadFile(chatPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != input {
		t.Fatalf("failed request changed chat log: %q", content)
	}
}

// run clones the default transport; trust only the local TLS test server here.
// These tests must remain sequential because the default transport is global.
func useTestTransport(t *testing.T, server *httptest.Server) {
	t.Helper()
	original := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	t.Cleanup(func() { http.DefaultTransport = original })
}

func writeRunFiles(t *testing.T, endpoint, deployment, settings, input string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	chatPath := filepath.Join(dir, "chat.log")
	content := fmt.Sprintf("azure_openai_api_key: test-api-key\nazure_openai_endpoint: %q\nazure_openai_model: %q\n%s",
		endpoint, deployment, settings)
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chatPath, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	return configPath, chatPath
}
