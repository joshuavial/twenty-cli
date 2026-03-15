---
tests:
  - id: MT-ENT-001
    name: People search returns stable JSON and text output
    category: entities
  - id: MT-ENT-002
    name: Person create, get, and update work end to end
    category: entities
  - id: MT-ENT-003
    name: Company search, create, get, and update work end to end
    category: entities
  - id: MT-ENT-004
    name: Deal search, create, get, and update work end to end
    category: entities
  - id: MT-ENT-005
    name: Entity commands fail cleanly on missing required flags
    category: entities
---

# Manual Test Plan: Entity Commands Against Local Twenty

## Scope
Manual verification of the shipped entity command surface against the local dev server:
- `people search`
- `person get|create|update`
- `companies search`
- `company get|create|update`
- `deals search`
- `deal get|create|update`

## Prerequisites
- Run `./dev reset`
- Confirm `go run ./cmd/twenty auth check` succeeds
- Work from the repo root so `./.twenty/settings` is picked up

## MT-ENT-001: People search returns stable JSON and text output

## Steps
1. Run `go run ./cmd/twenty people search --limit 3`
2. Run `go run ./cmd/twenty people search --query tim --limit 3`
3. Run `go run ./cmd/twenty --format text people search --query tim --limit 3`

## Expected Results
- JSON mode returns:
  - `ok: true`
  - `command: "people.search"`
  - `data` is an array
  - `meta.page_info.limit` is present
- Query mode returns an array result and should usually return one or more relevant records for seeded demo data
- Text mode prints a short human summary like `<n> people`

## MT-ENT-002: Person create, get, and update work end to end

## Steps
1. Run:
   `go run ./cmd/twenty person create --first-name Manual --last-name Person --email manual.person.$(date +%s)@example.test --job-title "QA Lead" --city Auckland`
2. Capture the returned `data.id` as `PERSON_ID`
3. Run `go run ./cmd/twenty person get --id "$PERSON_ID"`
4. Run `go run ./cmd/twenty person update --id "$PERSON_ID" --job-title "Senior QA Lead" --city Wellington`
5. Run `go run ./cmd/twenty person get --id "$PERSON_ID"`

## Expected Results
- Create returns:
  - `ok: true`
  - `command: "person.create"`
  - `data.id` is non-empty
- First get returns the created person
- Update returns:
  - `ok: true`
  - `command: "person.update"`
- Final get shows updated `jobTitle` and `city`

## MT-ENT-003: Company search, create, get, and update work end to end

## Steps
1. Run `go run ./cmd/twenty companies search --limit 3`
2. Run:
   `go run ./cmd/twenty company create --name "Manual Test Co $(date +%s)" --domain manual-test-$(date +%s).example.test --employees 42 --tagline "QA account"`
3. Capture the returned `data.id` as `COMPANY_ID`
4. Run `go run ./cmd/twenty companies search --query "Manual Test Co" --limit 5`
5. Run `go run ./cmd/twenty company get --id "$COMPANY_ID"`
6. Run `go run ./cmd/twenty company update --id "$COMPANY_ID" --employees 75 --tagline "Updated QA account"`
7. Run `go run ./cmd/twenty company get --id "$COMPANY_ID"`

## Expected Results
- Search returns a stable JSON envelope with `data` as an array, even when empty
- Create returns `command: "company.create"` and a non-empty `data.id`
- Search by the created company name returns the new company
- Get returns the created company
- Update returns `command: "company.update"`
- Final get shows updated `employees` and `tagline`

## MT-ENT-004: Deal search, create, get, and update work end to end

## Steps
1. Create a company and capture `COMPANY_ID` if you do not already have one
2. Create a person linked to that company and capture `PERSON_ID` if you do not already have one
3. Run:
   `go run ./cmd/twenty deal create --name "Manual Deal $(date +%s)" --company-id "$COMPANY_ID" --person-id "$PERSON_ID"`
4. Capture the returned `data.id` as `DEAL_ID`
5. Run `go run ./cmd/twenty deal get --id "$DEAL_ID"`
6. Run `go run ./cmd/twenty deals search --query "Manual Deal" --limit 5`
7. Run `go run ./cmd/twenty deal update --id "$DEAL_ID" --name "Manual Deal Updated" --stage NEW`
8. Run `go run ./cmd/twenty deal get --id "$DEAL_ID"`

## Expected Results
- Create returns `command: "deal.create"` and a non-empty `data.id`
- Get returns the created deal
- Search returns the deal in the result set
- Update returns `command: "deal.update"`
- Final get shows the new name and updated stage

## MT-ENT-005: Entity commands fail cleanly on missing required flags

## Steps
1. Run `go run ./cmd/twenty person get`
2. Run `go run ./cmd/twenty person create --email nobody@example.test`
3. Run `go run ./cmd/twenty company update --tagline "missing id"`
4. Run `go run ./cmd/twenty deal update --id fake-id`

## Expected Results
- Each command exits non-zero
- JSON mode returns:
  - `ok: false`
  - `error.kind: "usage"`
- Error codes are command-specific:
  - `person.missing_id`
  - `person.missing_name`
  - `company.missing_id`
  - `deal.missing_changes`

## Suggested Execution Order
1. MT-ENT-001
2. MT-ENT-002
3. MT-ENT-003
4. MT-ENT-004
5. MT-ENT-005
