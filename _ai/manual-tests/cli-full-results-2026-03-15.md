# Manual Test Results: Full Local CLI Pass

Date: 2026-03-15
Server: http://localhost:3000
Scope: local dev server only

## Summary

- Passed: foundation and auth checks
- Passed: entity search/get/create/update flows
- Passed: workflow note/task/meeting/call flows
- Passed: prospect import JSON, JSONL, stdin, dry-run, and lookup-first flows
- Passed: negative-path checks for usage and API failures

## Bugs Found And Fixed

### Prospect import failed on repeated companies within one input file

- Symptom: `prospect import` created the company for the first row, then failed the second row with a `400` when the same company appeared again in the same batch.
- Fix: cache same-batch company resolution and reuse the created company ID for later rows.

### Prospect import dry-run overstated company creation counts

- Symptom: dry-run responses counted planned companies under `created_companies`.
- Fix: stop treating planned actions as created in summary accounting.

### Empty search results returned `null` instead of `[]`

- Symptom: `companies search --query <missing>` returned `"data": null`.
- Fix: normalize empty entity search results to an empty array for a stable machine-facing contract.

### Manual auth test procedure was not runnable as written

- Symptom: the prior `MT-AUTH-002` temp-directory step used `go run`, which fails outside the module root.
- Fix: update the manual plan to build a disposable binary first, then run it from a temp directory.

## Verification

- `go test ./...`
- `go vet ./...`
- Live local checks confirmed:
  - empty company search returns `data: []`
  - duplicate-company import creates one company and two people
  - `--lookup-first` skips both duplicate people and duplicate company reuse
  - missing API key from a temp directory returns `auth.missing_api_key`
