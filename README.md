# codecatalyst

`codecatalyst` is a small Go CLI built with Cobra. It reads a chat log file, sends its contents to Azure OpenAI using the Responses API, and appends the model output back to the same file.

After each run, it prints the total elapsed time, token usage, and an estimated USD cost to the terminal.

## Requirements

- Go 1.25+
- An Azure OpenAI resource with a GPT-6 Astra or GPT-5.6 Sol deployment
- SSH access to `git@github.com:sarathyweb/codecatalyst.git` if you plan to push

## Configuration

The CLI reads a global YAML config file. By default it expects `~/.codecatalyst.yaml`. You can override the path with `--config`.

Example config file:

```yaml
azure_openai_api_key: "your-azure-openai-api-key"
azure_openai_endpoint: "https://your-resource.openai.azure.com/"
azure_openai_model: "gpt-6-astra"
azure_openai_reasoning_mode: "pro"
azure_openai_reasoning_effort: "xhigh"
azure_openai_multi_agent: true
azure_openai_embedding_model: "text-embedding-3-large"
database_url: "postgres://postgres:postgres@localhost:5432/codemigo?sslmode=disable"
```

The CLI uses Azure OpenAI's v1 API and automatically appends `/openai/v1/` to `azure_openai_endpoint` when needed. A dated `azure_openai_api_version` is no longer used.

`azure_openai_model` must match your Azure deployment name. To use GPT-6 Astra, select a deployment backed by `gpt-6-astra`; the example assumes the deployment has that name. Existing GPT-5.6 Sol deployments and configuration files continue to work. Model and feature availability depend on your Azure resource.

`azure_openai_reasoning_mode` accepts `pro` or `standard` and defaults to `pro` when omitted. Both models support Pro reasoning.

`azure_openai_reasoning_effort` accepts `low`, `medium`, `high`, `xhigh`, or `max` and defaults to `xhigh` when omitted, including in existing config files. Mode and effort are independent: the default request uses Pro mode with `xhigh` effort. Set the effort explicitly to override it.

`azure_openai_multi_agent` defaults to `true`; set it to `false` to disable beta multi-agent orchestration.

## Build

```powershell
go build -o codecatalyst.exe .
```

## Usage

```powershell
.\codecatalyst.exe .\chat.log
```

```powershell
.\codecatalyst.exe --config C:\path\to\codecatalyst.yaml .\chat.log
```

Behavior:

1. Reads the full contents of the chat log file.
2. Sends that content to the configured Azure OpenAI model using Pro mode, `xhigh` reasoning effort, and multi-agent orchestration by default.
3. Appends the returned text as a new `AI Assistant:` block at the end of the same file.
4. Prints elapsed time, token usage, and an estimated token cost for the model returned by Azure.

## Cost estimates

Estimates use standard OpenAI USD prices per million tokens, checked October 6, 2026:

| Model | Input | Cached input | Cache write | Output |
| --- | ---: | ---: | ---: | ---: |
| [GPT-6 Astra](https://developers.openai.com/api/docs/models/gpt-6-astra) | $10 | $1 | $12.50 | $50 |
| [GPT-5.6 Sol](https://developers.openai.com/api/docs/models/gpt-5.6-sol) | $4 | $0.40 | $5 | $20 |

The estimate uses the reported cache reads and writes. For more than 272,000 input tokens, it applies 2x input/cache rates and 1.5x output rates to the full request. GPT-5.6 Sol's listed promotional prices are available at least through November 21, 2026.

Pricing is selected from the response's model ID, including dated snapshots and the `gpt-5.6` alias. If the response omits the model, the configured deployment name is used. Unrecognized model IDs show `cost unavailable` while still appending the response and reporting token usage.

These are estimates, not Azure billing totals: Azure region, deployment pricing, and service tier may differ. Pro and multi-agent usage can aggregate several model calls, so applying the long-input threshold to aggregate usage is approximate.

## Development

Install dependencies and verify the project builds:

```powershell
go mod tidy
go test ./...
go build ./...
```

## Files

- `main.go`: CLI entry point and Azure OpenAI request flow
- `pricing.go`: model-specific token cost estimates
- `codecatalyst.example.yaml`: safe template for the global config file
- `.gitignore`: excludes binaries and local environment files from git
