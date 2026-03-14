# codecatalyst

`codecatalyst` is a small Go CLI built with Cobra. It reads a chat log file, sends its contents to Azure OpenAI using the Responses API, and appends the model output back to the same file.

After each run, it prints the total elapsed time, token usage, and a USD cost breakdown to the terminal.

## Requirements

- Go 1.25+
- An Azure OpenAI resource
- SSH access to `git@github.com:sarathyweb/codecatalyst.git` if you plan to push

## Configuration

The CLI reads a global YAML config file. By default it expects:

- `~/.codemigo.yaml`

You can override the path with `--config`.

Example config file:

```yaml
azure_openai_api_key: "your-azure-openai-api-key"
azure_openai_endpoint: "https://your-resource.openai.azure.com/"
azure_openai_api_version: "2025-03-01-preview"
azure_openai_model: "gpt-5"
azure_openai_embedding_model: "text-embedding-3-large"
database_url: "postgres://postgres:postgres@localhost:5432/codemigo?sslmode=disable"
```

## Build

```powershell
go build -o codecatalyst.exe .
```

## Usage

```powershell
.\codecatalyst.exe .\chat.log
```

```powershell
.\codecatalyst.exe --config C:\path\to\codemigo.yaml .\chat.log
```

Behavior:

1. Reads the full contents of the chat log file.
2. Sends that content to the configured Azure OpenAI model.
3. Appends the returned text as a new `AI Assistant:` block at the end of the same file.
4. Prints elapsed time and token-cost summary information.

## Development

Install dependencies and verify the project builds:

```powershell
go mod tidy
go build ./...
```

## Files

- `main.go`: CLI entry point and Azure OpenAI request flow
- `codemigo.example.yaml`: safe template for the global config file
- `.gitignore`: excludes binaries and local env files from git
