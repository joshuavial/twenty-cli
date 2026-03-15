---
tests:
  - id: MT-ONB-001
    name: First run with no config fails with a clear next step
    category: onboarding
  - id: MT-ONB-002
    name: Version and usage remain available before auth setup
    category: onboarding
  - id: MT-ONB-003
    name: Environment-only setup works without .twenty files
    category: onboarding
  - id: MT-ONB-004
    name: Home-directory settings work outside the repo
    category: onboarding
  - id: MT-ONB-005
    name: Broken settings file fails in an understandable way
    category: onboarding
  - id: MT-ONB-006
    name: README quickstart is sufficient for first successful auth check
    category: onboarding
---

# Manual Test Plan: First-Run And No-Config Onboarding

## Scope
Manual verification of the experience for a user who does not already have a local `.twenty` config:
- first run with no config
- environment-only setup
- home-directory config setup
- broken config handling
- install and quickstart guidance quality

## Why This Matters
The CLI is intended to be easy for LLMs to call, but a first-time human operator still needs a low-friction path to:
- install or build the binary
- understand why auth-dependent commands fail
- provide credentials once
- confirm success quickly

This plan is meant to expose friction in that path and verify that `twenty auth login` is enough to remove the need for manual JSON editing in the common case.

## Prerequisites
- Local Twenty dev server is running and healthy
- A valid API key is available from `./.twenty/settings` or `./.twenty/dev-server.env`
- Build a disposable binary before starting:
  `go build -o /tmp/twenty-manual ./cmd/twenty`

## MT-ONB-001: First run with no config fails with a clear next step

## Steps
1. Create a clean temp home and temp working directory:
   `export TMP_HOME="$(mktemp -d)"`
   `export TMP_WORK="$(mktemp -d)"`
2. Ensure no auth env vars are present:
   `unset TWENTY_API_KEY TWENTY_BASE_URL`
3. Run:
   `cd "$TMP_WORK" && HOME="$TMP_HOME" /tmp/twenty-manual auth check`

## Expected Results
- Command exits non-zero
- JSON output returns:
  - `ok: false`
  - `command: "auth.check"`
  - `error.kind: "auth"`
  - `error.code: "auth.missing_api_key"`
- The message should tell the user at least one concrete next step:
  - run `twenty auth login`
  - set `TWENTY_API_KEY`
  - or pass `--api-key`

## UX Notes
- Record whether the message is good enough for a first-time user without reading the source.
- If not, note the exact missing instruction.

## MT-ONB-002: Version and usage remain available before auth setup

## Steps
1. From the same clean temp environment, run:
   `HOME="$TMP_HOME" /tmp/twenty-manual version`
2. Run:
   `HOME="$TMP_HOME" /tmp/twenty-manual`
3. Optionally run:
   `HOME="$TMP_HOME" /tmp/twenty-manual --format text`

## Expected Results
- `version` succeeds without requiring auth
- bare CLI invocation returns usage output rather than an auth error
- usage lists the major command groups so a new user can see what the tool does

## UX Notes
- Record whether the usage output is readable enough to orient a new user.
- Record whether the command list implies how auth should be configured.

## MT-ONB-003: Environment-only setup works without .twenty files

## Steps
1. Extract the dev-server credentials:
   `export TWENTY_API_KEY="$(jq -r '.api_key' ./.twenty/settings)"`
   `export TWENTY_BASE_URL="$(jq -r '.base_url' ./.twenty/settings)"`
2. Run from a clean temp working directory with a clean temp home:
   `cd "$TMP_WORK" && HOME="$TMP_HOME" /tmp/twenty-manual auth check`

## Expected Results
- Command succeeds
- JSON output returns:
  - `ok: true`
  - `command: "auth.check"`
  - `data.status_code: 200`

## UX Notes
- Record how much setup was required for this to work.
- Record whether env-only setup feels acceptable as the first documented path.

## MT-ONB-004: Home-directory settings work outside the repo

## Steps
1. Run from the temp working directory:
   `cd "$TMP_WORK" && HOME="$TMP_HOME" /tmp/twenty-manual auth login --api-key "$(jq -r '.api_key' ./.twenty/settings)" --base-url "$(jq -r '.base_url' ./.twenty/settings)"`
2. Unset `TWENTY_API_KEY` and `TWENTY_BASE_URL`
3. Run from the temp working directory:
   `cd "$TMP_WORK" && HOME="$TMP_HOME" /tmp/twenty-manual auth check`

## Expected Results
- `auth login` succeeds and writes `$TMP_HOME/.twenty/settings`
- Command succeeds without requiring repo-local config
- This proves a user can install the binary once and use it from arbitrary directories

## UX Notes
- Record whether `auth login` is simple enough that manual JSON editing can stay undocumented in the quickstart.

## MT-ONB-005: Broken settings file fails in an understandable way

## Steps
1. Write invalid JSON to `$TMP_HOME/.twenty/settings`
2. Ensure auth env vars are unset
3. Run:
   `cd "$TMP_WORK" && HOME="$TMP_HOME" /tmp/twenty-manual auth check`

## Expected Results
- Command exits non-zero
- Output should make it obvious the settings file is malformed
- Failure should still be machine-readable JSON in default mode
- The error should tell the user to repair the file or rerun `twenty auth login --overwrite`

## UX Notes
- Record whether the error names the likely source of the problem clearly enough.
- Record whether a first-time user would know how to recover.

## MT-ONB-006: README quickstart is sufficient for first successful auth check

## Steps
1. Read [README.md](/Users/jv/gt/twenty_crm_cli/crew/conduit/README.md) as if you are a new user with no prior repo context
2. Follow only the documented setup path needed to get to a successful `auth check`
3. Do not rely on tribal knowledge or source inspection

## Expected Results
- The README makes it obvious:
  - how to build or run the CLI
  - how to provide credentials with `auth login`
  - how to verify success with `auth check`
- A first successful auth check should be achievable from the docs alone

## UX Notes
- Record every point where the README assumes insider knowledge.
- Record the minimum doc changes needed to make first install feel obvious.

## Suggested Execution Order
1. MT-ONB-001
2. MT-ONB-002
3. MT-ONB-003
4. MT-ONB-004
5. MT-ONB-005
6. MT-ONB-006

## Exit Criteria
- A first-time user can get from zero config to a successful `auth check`
- Failure paths remain clear and actionable
- The docs and CLI messages provide enough guidance without code reading
