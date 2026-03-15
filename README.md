# twenty-crm-cli

Go CLI foundation for interacting with Twenty CRM using an LLM-friendly command surface.

## Current Commands

```bash
twenty auth check
twenty people search
twenty person get|create|update
twenty companies search
twenty company get|create|update
twenty deals search
twenty deal get|create|update
twenty note add
twenty task create
twenty meeting log
twenty call capture
twenty prospect import
twenty version
```

## Output Contract

JSON is the primary machine-readable mode. Every command emits a stable envelope on `stdout`:

```json
{
  "ok": true,
  "command": "auth.check",
  "data": {
    "status_code": 200,
    "endpoint": "/metadata"
  }
}
```

Failures keep the same top-level shape and normalize error classification:

```json
{
  "ok": false,
  "command": "auth.check",
  "error": {
    "kind": "auth",
    "code": "auth.invalid_credentials",
    "message": "api request failed with status 401",
    "details": {
      "status_code": 401
    }
  }
}
```

Optional metadata lives under `meta` and is omitted when empty. The contract currently reserves:

- `meta.page_info` for paging state (`limit`, `returned`, `total`, `next_cursor`, `prev_cursor`)
- `meta.warnings` for non-fatal machine-readable warnings

Exit codes are stable across commands:

- `0` success
- `2` usage or parse failures
- `3` auth or credential failures
- `4` API failures after a valid request
- `10` internal/runtime failures

`--format text` remains available for humans, but it is just a presentation layer over the same underlying result/error model.

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

Settings files are JSON. Example:

```json
{
  "api_key": "twenty_api_key_here",
  "base_url": "https://api.twenty.com"
}
```

The current working directory config overrides the home directory config.

## Local Twenty dev server

This repo includes a local Twenty CRM checkout as the `twenty/` git submodule. The root `./dev` wrapper runs Docker Compose from `twenty/packages/twenty-docker`.

```bash
./dev up -d
./dev down
./dev logs -f server
./dev ps
./dev reset
```

The local app is exposed on:

- `http://localhost:3000`
- `http://localhost:3000/graphql`

Postgres and Redis remain internal to the Docker network.

## Reset For Integration Tests

`./dev reset` does all of the setup needed for local integration testing:

1. Wipes the Docker volumes
2. Starts a fresh Twenty stack
3. Seeds the upstream demo workspace
4. Generates a fresh API key
5. Writes CLI-compatible local settings into `./.twenty/settings`

It also writes a few convenience artifacts:

- `./.twenty/api-key`
- `./.twenty/dev-server.env`

The generated CLI settings file looks like this:

```json
{
  "api_key": "<generated-api-key>",
  "base_url": "http://localhost:3000"
}
```

Because the Go CLI checks `./.twenty/settings` before `~/.twenty/settings`, running commands from this repo will automatically target the local dev server after `./dev reset`.

## Demo Account

The reset flow seeds the `Apple` demo workspace with:

- Email: `tim@apple.dev`
- Password: `tim@apple.dev`

## Browserplex Flow

`./dev reset` is the fastest path because it generates the API key directly inside the server container. If you want to validate the browser path too, use Browserplex against `http://localhost:3000`, log in with the demo account above, then go to `Settings` -> `API & Webhooks` and create a key there.

For repeatable browser automation, save the Browserplex storage state after login so later runs can skip the sign-in form.

## Notes

- Twenty's upstream demo seeder is not cleanly idempotent. Re-running it against an already-seeded workspace can produce duplicate-key errors.
- `./dev reset` avoids seed conflicts by wiping Docker volumes first, then seeding into a clean database.
- API key generation uses Twenty's built-in `workspace:generate-api-key` command with `NODE_ENV=development` forced for that one-shot command run.

## Verification

```bash
go test ./...
go vet ./...
```

## Integration Test Harness

The end-to-end CLI harness lives under `./test/e2e`. These tests build the real
`twenty` binary, run it in a temp workspace with explicit env/config, and compare
JSON stdout against golden files in `./test/e2e/testdata`.

Run the full suite:

```bash
go test ./...
```

Run only the integration harness:

```bash
go test ./test/e2e
```

Update golden snapshots intentionally:

```bash
UPDATE_GOLDEN=1 go test ./test/e2e
```
