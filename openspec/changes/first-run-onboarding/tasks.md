# Tasks

- [ ] Add `twenty auth login` command with `--api-key`, `--base-url`, and `--scope home|project`
- [ ] Add optional interactive prompting when flags are omitted and stdin is a TTY
- [ ] Verify credentials with the current auth probe before persisting settings
- [ ] Create `.twenty/` or `~/.twenty/` automatically when missing
- [ ] Add overwrite behavior and a clear refusal mode when config already exists
- [ ] Improve missing-auth and malformed-settings failure messages to point at `twenty auth login`
- [ ] Update usage output and README quickstart for first-time users
- [ ] Add unit, e2e, and manual coverage for onboarding flows
