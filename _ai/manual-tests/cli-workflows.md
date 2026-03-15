---
tests:
  - id: MT-WF-001
    name: Note add supports linked targets
    category: workflows
  - id: MT-WF-002
    name: Task create supports due dates and linked targets
    category: workflows
  - id: MT-WF-003
    name: Meeting log creates note and follow-up tasks
    category: workflows
  - id: MT-WF-004
    name: Call capture logs notes, tasks, and deal stage updates
    category: workflows
  - id: MT-WF-005
    name: Workflow commands fail cleanly on invalid usage
    category: workflows
---

# Manual Test Plan: Workflow Commands Against Local Twenty

## Scope
Manual verification of the higher-level workflow commands against the local dev server:
- `note add`
- `task create`
- `meeting log`
- `call capture`

## Prerequisites
- Run `./dev reset`
- Confirm `go run ./cmd/twenty auth check` succeeds
- Create or locate a valid `PERSON_ID`, `COMPANY_ID`, and `DEAL_ID`

## MT-WF-001: Note add supports linked targets

## Steps
1. Run:
   `go run ./cmd/twenty note add --title "Manual note" --body "Met and captured notes" --person-id "$PERSON_ID" --company-id "$COMPANY_ID" --deal-id "$DEAL_ID"`
2. Optionally rerun with `--format text`

## Expected Results
- JSON mode returns:
  - `ok: true`
  - `command: "note.add"`
  - `data.id` is non-empty
- Text mode prints `note added`

## MT-WF-002: Task create supports due dates and linked targets

## Steps
1. Run:
   `go run ./cmd/twenty task create --title "Send follow-up" --body "Email notes and next steps" --due-at 2026-03-31T09:00:00Z --person-id "$PERSON_ID" --company-id "$COMPANY_ID" --deal-id "$DEAL_ID"`
2. Optionally rerun using `--body-file` with a temp markdown file

## Expected Results
- JSON mode returns:
  - `ok: true`
  - `command: "task.create"`
  - `data.id` is non-empty
  - `data.dueAt` matches the supplied RFC3339 value

## MT-WF-003: Meeting log creates note and follow-up tasks

## Steps
1. Run:
   `go run ./cmd/twenty meeting log --title "Weekly sync" --body "Discussed blockers and action items" --person-id "$PERSON_ID" --company-id "$COMPANY_ID" --deal-id "$DEAL_ID" --create-followups --next-step "Send revised proposal" --next-step "Book technical review"`
2. Inspect the returned JSON

## Expected Results
- JSON mode returns:
  - `ok: true`
  - `command: "meeting.log"`
  - `data.note.id` is non-empty
  - `data.tasks` exists with 2 created tasks

## MT-WF-004: Call capture logs notes, tasks, and deal stage updates

## Steps
1. Run:
   `go run ./cmd/twenty call capture --title "Discovery call" --body "Qualified budget and timeline" --person-id "$PERSON_ID" --company-id "$COMPANY_ID" --deal-id "$DEAL_ID" --deal-stage NEW --create-followups --next-step "Send pricing" --next-step "Schedule demo"`
2. Inspect the returned JSON
3. Run `go run ./cmd/twenty deal get --id "$DEAL_ID"`

## Expected Results
- JSON mode returns:
  - `ok: true`
  - `command: "call.capture"`
  - `data.note.id` is non-empty
  - `data.tasks` exists with 2 created tasks
  - `data.deal.stage` is updated
- Final `deal get` shows the updated stage

## MT-WF-005: Workflow commands fail cleanly on invalid usage

## Steps
1. Run `go run ./cmd/twenty note add --title "Bad note"`
2. Run `go run ./cmd/twenty task create --title "Bad due" --due-at tomorrow`
3. Run `go run ./cmd/twenty meeting log --body "Notes" --create-followups`
4. Run `go run ./cmd/twenty call capture --body "Notes" --deal-stage closed --deal-id "$DEAL_ID"`

## Expected Results
- First three commands exit non-zero with `error.kind: "usage"`
- Expected error codes:
  - `note.add.missing_body`
  - `task.create.due_at`
  - `meeting.log.missing_next_steps`
- The invalid deal stage command exits non-zero with either:
  - `error.kind: "api"` if the server rejects the stage value
  - or `error.kind: "usage"` if client-side validation is added later

## Suggested Execution Order
1. MT-WF-001
2. MT-WF-002
3. MT-WF-003
4. MT-WF-004
5. MT-WF-005
