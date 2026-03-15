# Manual Test Results: Workflow And Prospect Commands

Date: 2026-03-15
Server: http://localhost:3000
Scope: local dev server only

## Passed

- `note add`
- `task create`
- `meeting log --create-followups`
- `call capture --deal-stage --create-followups`
- `prospect import --lookup-first`

## Notes

- `meeting log` and `call capture` are implemented as:
  - create a `note`
  - optionally create linked `tasks`
  - optionally update the linked deal stage for `call capture`
- Local Twenty does not expose first-class `meetings` or `calls` REST objects in this workspace, so these workflows intentionally translate into note/task/target operations.
- Prospect import now uses lookup-first server-side search before creating people or companies.
