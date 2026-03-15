# First-Run Onboarding

> Add a guided, LLM-safe auth setup flow so a new user can reach a successful `auth check` without hand-writing config.

## Why

The current onboarding path is too manual for a first-time user. A new operator must discover the config format from the README, choose between env vars and settings files, create `.twenty/settings` or `~/.twenty/settings` by hand, and then infer whether setup worked by retrying `auth check`.

The new onboarding manual tests make that weakness explicit:
- no-config errors are machine-readable, but not enough to complete setup unaided
- there is no `auth login`, `auth set`, or `config init` command
- README quickstart still assumes too much repo context

For an LLM-first CLI, the right first-run experience is:
- one obvious command to persist credentials
- deterministic config scope
- validation before writing
- clear recovery paths for malformed config and bad credentials

## What Changes

This change introduces a guided onboarding/auth-setup feature centered on `twenty auth login`, supported by better first-run guidance and docs.

High-level changes:
- add a non-interactive-first `twenty auth login` command that can write `~/.twenty/settings` or `./.twenty/settings`
- add safe config writing semantics, including directory creation and overwrite behavior
- improve missing-auth and malformed-config error messages so they point users at the setup command
- add a compact README quickstart for first install and first successful `auth check`
- extend tests to cover install/onboarding flows, config scope behavior, and recovery paths

## Scope

In scope:
- auth setup flow
- config file creation/update
- first-run guidance
- docs and tests

Out of scope:
- OS keychain integration
- browser-based OAuth or hosted login flow
- multi-profile config management
- encrypted credential storage

## Key Decisions

- The primary setup command should be `twenty auth login`, not `twenty config init`.
  Reason: the user goal is “make auth work”, not “manage config files”.
- The command should be non-interactive-first but support prompts when attached to a TTY.
  Reason: LLM callers need flag-driven behavior; humans benefit from prompts.
- The default persistence scope should be `home`.
  Reason: first-time install should work from arbitrary directories, not only inside one repo.

## Open Questions

- Whether to add `twenty auth logout` in the same change or defer it.
- Whether to support `--no-verify` on first release or require credential verification before write.
