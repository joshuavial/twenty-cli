# twenty-crm-cli

Go CLI foundation for interacting with Twenty CRM using an LLM-friendly command surface.

## Current Commands

```bash
twenty auth check
twenty version
```

## Config

The CLI looks for credentials and base URL in this order:

1. Command flags
2. Environment variables
3. `./.twenty/settings`
4. `~/.twenty/settings`
5. Default base URL: `https://api.twenty.com`

Supported environment variables:

- `TWENTY_API_KEY`
- `TWENTY_BASE_URL`

Supported flags:

- `--api-key`
- `--base-url`
- `--format json|text`

Settings files live inside a `.twenty` config directory and are JSON. Example:

```json
{
  "api_key": "twenty_api_key_here",
  "base_url": "https://api.twenty.com"
}
```

The current working directory config overrides the home directory config.

## Verification

```bash
go test ./...
go vet ./...
```
