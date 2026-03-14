# Manual Test Results: Entity Commands Against Local Twenty

Date: 2026-03-15
Server: http://localhost:3000
Scope: local dev server only

## Summary

- Passed: `people search`
- Passed: `companies search`
- Passed: `deals search`
- Passed: `person get`
- Passed: `company get`
- Passed: `deal get`
- Passed: `person create`
- Passed: `company create`
- Passed: `deal create`
- Passed: `person update`
- Passed: `company update`
- Passed: `deal update`
- Passed: text mode spot checks for `people search`, `company create`, and `deal update`

## Bug Found And Fixed

- Before server-side query translation, `people search --query ...` and `companies search --query ...` could miss freshly created records because the CLI fetched a single page and filtered locally.
- This caused follow-on update commands to fail when a shell pipeline resolved `null` as the record ID.
- Fixed by translating queries into Twenty REST `filter=` parameters and issuing server-side searches on domain-specific fields.

## Translation Notes

- `people search --query ...` now probes:
  - `name.firstName[ilike]`
  - `name.lastName[ilike]`
  - `emails.primaryEmail[ilike]`
- `companies search --query ...` now probes:
  - `name[ilike]`
  - `domainName.primaryLinkUrl[ilike]`
- `deals search --query ...` now probes:
  - `name[ilike]`

Results are merged and deduplicated by record ID.
