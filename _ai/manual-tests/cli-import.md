---
tests:
  - id: MT-IMP-001
    name: Prospect import dry-run previews work without creating records
    category: import
  - id: MT-IMP-002
    name: Prospect import creates people and companies from JSON array input
    category: import
  - id: MT-IMP-003
    name: Prospect import lookup-first skips existing records
    category: import
  - id: MT-IMP-004
    name: Prospect import handles JSONL and stdin input
    category: import
  - id: MT-IMP-005
    name: Prospect import fails cleanly for missing or invalid files
    category: import
---

# Manual Test Plan: Prospect Import Against Local Twenty

## Scope
Manual verification of `prospect import` against the local dev server:
- JSON array input
- JSONL input
- stdin input
- `--dry-run`
- `--lookup-first`
- invalid usage and malformed files

## Prerequisites
- Run `./dev reset`
- Confirm `go run ./cmd/twenty auth check` succeeds
- Create a temp directory for import fixtures

## Test Data
Use unique values per run. Recommended sample JSON array:

```json
[
  {
    "first_name": "Import",
    "last_name": "One",
    "email": "import.one.RUN_ID@example.test",
    "company": "Import Co RUN_ID",
    "company_domain": "import-RUN_ID.example.test",
    "job_title": "Buyer",
    "city": "Auckland"
  },
  {
    "first_name": "Import",
    "last_name": "Two",
    "email": "import.two.RUN_ID@example.test",
    "company": "Import Co RUN_ID",
    "company_domain": "import-RUN_ID.example.test",
    "job_title": "Champion",
    "city": "Wellington"
  }
]
```

Replace `RUN_ID` with a timestamp or other unique suffix before use.

## MT-IMP-001: Prospect import dry-run previews work without creating records

## Steps
1. Save the sample JSON array to `/tmp/prospects.json`
2. Run `go run ./cmd/twenty prospect import --file /tmp/prospects.json --dry-run`
3. Search for one of the prospect emails with `people search --query ...`

## Expected Results
- Import returns:
  - `ok: true`
  - `command: "prospect.import"`
  - `data.dry_run: true`
  - `data.processed: 2`
- Result items show planned actions
- Follow-up search does not show created records

## MT-IMP-002: Prospect import creates people and companies from JSON array input

## Steps
1. Run `go run ./cmd/twenty prospect import --file /tmp/prospects.json`
2. Search for the imported people by email
3. Search for the imported company by name or domain

## Expected Results
- Import returns:
  - `ok: true`
  - `data.processed: 2`
  - `data.created_people` is at least `2`
  - `data.created_companies` is at least `1`
- Search confirms the imported records exist

## MT-IMP-003: Prospect import lookup-first skips existing records

## Steps
1. Re-run the same file with `go run ./cmd/twenty prospect import --file /tmp/prospects.json --lookup-first`
2. Inspect the summary and per-record results

## Expected Results
- Import returns:
  - `ok: true`
  - `data.processed: 2`
- Existing people are counted under `skipped_people`
- Existing company is counted under `skipped_companies`
- No duplicate records are created

## MT-IMP-004: Prospect import handles JSONL and stdin input

## Steps
1. Create `/tmp/prospects.jsonl` with one valid JSON object per line
2. Run `go run ./cmd/twenty prospect import --file /tmp/prospects.jsonl --dry-run`
3. Run `cat /tmp/prospects.jsonl | go run ./cmd/twenty prospect import --file - --dry-run`

## Expected Results
- Both commands succeed
- Each returns `command: "prospect.import"` and `data.processed` matching the number of JSONL lines

## MT-IMP-005: Prospect import fails cleanly for missing or invalid files

## Steps
1. Run `go run ./cmd/twenty prospect import`
2. Run `go run ./cmd/twenty prospect import --file /tmp/does-not-exist.json`
3. Create `/tmp/bad-prospects.json` with invalid JSON and run `go run ./cmd/twenty prospect import --file /tmp/bad-prospects.json`

## Expected Results
- All commands exit non-zero
- Missing flag returns:
  - `error.kind: "usage"`
  - `error.code: "prospect.import.missing_file"`
- Missing or malformed files return:
  - `error.kind: "usage"`
  - `error.code: "prospect.import.file"`

## Suggested Execution Order
1. MT-IMP-001
2. MT-IMP-002
3. MT-IMP-003
4. MT-IMP-004
5. MT-IMP-005
