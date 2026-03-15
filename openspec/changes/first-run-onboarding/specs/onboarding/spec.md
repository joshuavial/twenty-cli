# Onboarding

## ADDED Requirements

### Requirement: Guided auth setup command

The CLI SHALL provide a `twenty auth login` command that lets a first-time user persist credentials without manually creating JSON settings files.

#### Scenario: Non-interactive auth setup to home scope

- **Given**: the user has no existing `~/.twenty/settings`
- **When**: they run `twenty auth login --api-key <key> --base-url <url> --scope home`
- **Then**: the CLI verifies the credentials
- **And**: the CLI creates `~/.twenty/settings`
- **And**: the CLI writes valid JSON settings with the provided API key and base URL
- **And**: the command returns a success envelope identifying the written scope and path

#### Scenario: Non-interactive auth setup to project scope

- **Given**: the user is inside a project directory with no `./.twenty/settings`
- **When**: they run `twenty auth login --api-key <key> --base-url <url> --scope project`
- **Then**: the CLI verifies the credentials
- **And**: the CLI creates `./.twenty/settings`
- **And**: later commands in that project resolve the project settings before home settings

#### Scenario: Interactive auth setup on a TTY

- **Given**: the user runs `twenty auth login` from a TTY with required flags omitted
- **When**: the CLI detects interactive input is available
- **Then**: it prompts for missing values
- **And**: it still writes the same settings format as the non-interactive flow

### Requirement: Safe and explicit config persistence

The CLI MUST write config files safely and predictably so onboarding does not silently destroy existing setup.

#### Scenario: Existing settings file without overwrite permission

- **Given**: the target settings file already exists
- **When**: the user runs `twenty auth login` without an explicit overwrite flag
- **Then**: the CLI refuses to overwrite the file
- **And**: it returns a machine-readable failure explaining how to proceed

#### Scenario: Existing settings file with overwrite permission

- **Given**: the target settings file already exists
- **When**: the user runs `twenty auth login` with `--overwrite`
- **Then**: the CLI replaces the settings file atomically
- **And**: the resulting file contains only the newly supplied credentials and base URL

#### Scenario: Missing config directory

- **Given**: the target `.twenty/` directory does not exist
- **When**: the user runs `twenty auth login`
- **Then**: the CLI creates the directory automatically with user-writable permissions

### Requirement: Actionable first-run failures

When auth-dependent commands fail because onboarding has not happened yet, the CLI MUST tell the user the next action to take.

#### Scenario: Auth check with no credentials

- **Given**: there are no flags, no auth env vars, and no settings files
- **When**: the user runs `twenty auth check`
- **Then**: the CLI returns `auth.missing_api_key`
- **And**: the message tells the user to run `twenty auth login` or provide `--api-key`

#### Scenario: Malformed settings file

- **Given**: a settings file exists but contains invalid JSON
- **When**: the user runs an auth-dependent command
- **Then**: the CLI returns a machine-readable failure
- **And**: the error clearly identifies the settings file as malformed
- **And**: the message tells the user how to repair or overwrite it

### Requirement: First-install documentation

The repo documentation SHALL include a short first-install path that gets a user from zero config to a successful `auth check`.

#### Scenario: New user follows README quickstart

- **Given**: a user has the binary or a `go run` path but no prior config
- **When**: they follow the README onboarding section only
- **Then**: they can supply credentials, persist them, and run a successful `twenty auth check`
- **And**: the docs do not require source inspection to discover the settings file shape

### Requirement: LLM-safe onboarding contract

The onboarding flow MUST remain easy for LLM callers to automate.

#### Scenario: LLM caller uses flags only

- **Given**: an LLM agent runs `twenty auth login --api-key <key> --base-url <url> --scope home`
- **When**: stdin is not a TTY
- **Then**: the command does not prompt
- **And**: it succeeds or fails entirely through stable stdout/stderr and exit codes

#### Scenario: LLM caller requests text mode

- **Given**: an LLM or human caller runs `twenty --format text auth login ...`
- **When**: the setup succeeds
- **Then**: the text output is concise and confirms what path was written
