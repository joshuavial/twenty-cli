---
tests:
  - id: MT-FOUND-001
    name: Dev server reset artifacts exist
    category: foundation
  - id: MT-FOUND-002
    name: CLI version returns stable envelope
    category: foundation
  - id: MT-AUTH-001
    name: Auth check succeeds from local settings
    category: auth
  - id: MT-AUTH-002
    name: Auth check fails cleanly with missing API key
    category: auth
  - id: MT-AUTH-003
    name: Flags override local settings
    category: auth
  - id: MT-AUTH-004
    name: Invalid base URL fails with usage envelope
    category: auth
---

# Manual Test Plan: CLI Foundation Against Local Twenty

## Scope
Manual verification of the current implemented surface:
- local Twenty dev server lifecycle assumptions
- generated local config artifacts from `./dev reset`
- `twenty version`
- `twenty auth check`
- config precedence and failure handling

## Out Of Scope
- people/company/deal commands
- schema inspection commands exposed via CLI
- meeting/call/prospect workflows
- hosted production smoke tests

## MT-FOUND-001: Dev server reset artifacts exist

## Prerequisites
- Run `./dev reset`

## Steps
1. Run `./dev ps`
2. Run `curl -fsS http://localhost:3000/healthz`
3. Run `ls -la .twenty`
4. Open `.twenty/settings`
5. Open `.twenty/dev-server.env`

## Expected Results
- `db`, `redis`, `server`, and `worker` containers are up
- `healthz` returns JSON with `"status":"ok"`
- `.twenty/settings` exists and contains `api_key` plus `base_url`
- `.twenty/dev-server.env` exists and contains `TWENTY_API_KEY`, `TWENTY_BASE_URL`, and demo workspace values

## Notes
- This test validates the environment required for all later manual CLI checks.

## MT-FOUND-002: CLI version returns stable envelope

## Prerequisites
- None beyond repo setup

## Steps
1. Run `go run ./cmd/twenty version`
2. Run `go run ./cmd/twenty --format text version`

## Expected Results
- JSON mode returns:
  - `ok: true`
  - `command: "version"`
  - `data.version`
- Text mode prints `dev`

## MT-AUTH-001: Auth check succeeds from local settings

## Prerequisites
- `./dev reset` completed successfully
- `.twenty/settings` exists in repo root

## Steps
1. Ensure `TWENTY_API_KEY` and `TWENTY_BASE_URL` are unset in the shell running the command
2. Run `go run ./cmd/twenty auth check`
3. Optionally run `go run ./cmd/twenty --format text auth check`

## Expected Results
- JSON mode returns:
  - `ok: true`
  - `command: "auth.check"`
  - `data.status_code: 200`
  - `data.endpoint: "/metadata"`
- Text mode prints `auth ok`

## MT-AUTH-002: Auth check fails cleanly with missing API key

## Prerequisites
- None

## Steps
1. Temporarily move `./.twenty/settings` out of the way or run from a temp directory without local settings
2. Ensure `TWENTY_API_KEY` is unset
3. Run `go run ./cmd/twenty auth check`

## Expected Results
- Command exits non-zero
- JSON envelope returns:
  - `ok: false`
  - `command: "auth.check"`
  - `error.kind: "auth"`
  - `error.code: "auth.missing_api_key"`

## MT-AUTH-003: Flags override local settings

## Prerequisites
- Local stack is running

## Steps
1. Run `go run ./cmd/twenty --api-key bad-key --base-url http://localhost:3000 auth check`
2. Run `go run ./cmd/twenty --api-key "$(jq -r '.api_key' .twenty/settings)" --base-url http://localhost:3000 auth check`

## Expected Results
- First command fails with an auth or API error envelope, proving the explicit bad flag value overrides settings
- Second command succeeds, proving explicit valid flags also override settings correctly

## MT-AUTH-004: Invalid base URL fails with usage envelope

## Prerequisites
- None

## Steps
1. Run `go run ./cmd/twenty --base-url not-a-url version`
2. Run `go run ./cmd/twenty --format text --base-url not-a-url version`

## Expected Results
- JSON mode returns:
  - `ok: false`
  - `command: "cli"`
  - `error.kind: "usage"`
  - `error.code: "cli.parse"`
- Text mode prints the validation error plainly

## Suggested Execution Order
1. MT-FOUND-001
2. MT-FOUND-002
3. MT-AUTH-001
4. MT-AUTH-002
5. MT-AUTH-003
6. MT-AUTH-004

## Exit Criteria
- All foundation tests pass
- Auth success works against the reset local workspace
- Failure modes remain machine-readable and stable
