# Source: Twenty REST Integration Tests And Utilities

**URL**: Multiple official repo files under `packages/twenty-server/test/integration/rest/` and `packages/twenty-front/src/modules/apollo/constant/rest-api-base-url.ts`
**Retrieved**: 2026-03-13T10:31:41Z
**Relevance**: Best concrete evidence for route prefixes, query parameters, response shapes, and special endpoints.

## Summary

- REST requests are sent to `/rest`.
- Object routes are pluralized, e.g. `/people`, `/companies`, `/opportunities`.
- Confirmed operations include:
  - collection listing
  - create one
  - create many via `/batch/{plural}`
  - update one
  - delete one
  - duplicate detection via `/{plural}/duplicates`
  - aggregation via `/{plural}/groupBy`
- Confirmed query parameters include `limit`, `filter`, `order_by`, `starting_after`, `ending_before`, and `depth`.
- `depth=0` and `depth=1` work; `depth=2` is rejected.
- Bulk delete/destroy operations require filters.
- Response bodies are action-oriented, e.g. `data.createPerson`, `data.updatePerson`, `data.deletePerson`, or `data.people`.

## Reliability Notes

- Official integration tests are strong evidence of implemented behavior.
- They reflect code behavior in the repo state retrieved on 2026-03-13, not necessarily every cloud deployment/version.
