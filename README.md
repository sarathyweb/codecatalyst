# codecatalyst

`codecatalyst` is a small Go CLI that reads a chat log file, sends its contents to Azure OpenAI using the Responses API, and appends the model output back to the same file.

## Requirements

- Go 1.25+
- An Azure OpenAI resource
- SSH access to `git@github.com:sarathyweb/codecatalyst.git` if you plan to push

## Configuration

Copy the example env file and fill in your real values:

```powershell
Copy-Item .env.example .env
```

Required environment variables:

- `AZURE_OPENAI_API_KEY`
- `AZURE_OPENAI_ENDPOINT`
- `AZURE_OPENAI_API_VERSION`
- `AZURE_OPENAI_MODEL`

The app loads variables from `.env` automatically via `godotenv`.

## Build

```powershell
go build -o codecatalyst.exe .
```

## Usage

```powershell
.\codecatalyst.exe .\chat.log
```

Behavior:

1. Reads the full contents of the chat log file.
2. Sends that content to the configured Azure OpenAI model.
3. Appends the returned text as a new `AI Assistant:` block at the end of the same file.

## Development

Install dependencies and verify the project builds:

```powershell
go mod tidy
go build ./...
```

## Files

- `main.go`: CLI entry point and Azure OpenAI request flow
- `.env.example`: safe template for local configuration
- `.gitignore`: excludes binaries and local env files from git
