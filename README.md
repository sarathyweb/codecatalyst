# codecatalyst

`codecatalyst` is a small Go CLI built with Cobra. It reads a chat log file, sends its contents to Azure OpenAI using the Responses API, and appends the model output back to the same file.

After each run, it prints the total elapsed time, token usage, and an estimated USD cost to the terminal.

## Requirements

- Go 1.25+
- An Azure OpenAI resource with a GPT-5.6 Sol deployment
- SSH access to `git@github.com:sarathyweb/codecatalyst.git` if you plan to push

## Configuration

The CLI reads a global YAML config file. By default it expects `~/.codecatalyst.yaml`. You can override the path with `--config`.

Example config file:

```yaml
azure_openai_api_key: "your-azure-openai-api-key"
azure_openai_endpoint: "https://your-resource.openai.azure.com/"
azure_openai_model: "gpt-5.6-sol"
azure_openai_reasoning_mode: "pro"
azure_openai_multi_agent: true
azure_openai_embedding_model: "text-embedding-3-large"
database_url: "postgres://postgres:postgres@localhost:5432/codemigo?sslmode=disable"
```

The CLI uses Azure OpenAI's v1 API and automatically appends `/openai/v1/` to `azure_openai_endpoint` when needed. A dated `azure_openai_api_version` is no longer used.

`azure_openai_model` must match the name of your Azure GPT-5.6 Sol deployment. `azure_openai_reasoning_mode` accepts `pro` or `standard` and defaults to `pro` when omitted. `azure_openai_multi_agent` defaults to `true`; set it to `false` to disable beta multi-agent orchestration.

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
2. Sends that content to the configured Azure OpenAI model using Pro reasoning and multi-agent orchestration by default.
3. Appends the returned text as a new `AI Assistant:` block at the end of the same file.
4. Prints elapsed time, token usage, and an estimated GPT-5.6 Sol token cost.

## Development

Install dependencies and verify the project builds:

```powershell
go mod tidy
go build ./...
```

## Files

- `main.go`: CLI entry point and Azure OpenAI request flow
- `codecatalyst.example.yaml`: safe template for the global config file
- `.gitignore`: excludes binaries and local environment files from git
