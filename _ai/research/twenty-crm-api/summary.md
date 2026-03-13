# Twenty CRM API Summary

## Objective
Document the Twenty CRM API in enough detail to support a practical CLI or integration build.

## Top Findings

1. Twenty exposes two API families: a core data API for records and a metadata API for workspace schema/configuration. Both are available in REST and GraphQL forms. Core paths are `/rest/` and `/graphql`; metadata paths are `/rest/metadata/` and `/metadata`. [S1] [S5]
2. The public auth model is simple bearer-token auth using API keys. The cloud base URL is `https://api.twenty.com/`. API keys can be assigned roles, so effective permissions are role-scoped rather than all-powerful by default. [S1]
3. The REST API is more capable than the high-level docs suggest. Official integration tests confirm collection CRUD, batch create via `/rest/batch/{object}`, cursor pagination, filters, ordering, duplicate detection, and group-by endpoints. [S4]
4. The metadata API is clearly GraphQL-first in the evidence reviewed. Official repo tests show mutations such as `createOneObject` and `createOneField`, plus connection-style queries like `fields(...) { edges { node { ... }}}`. [S5]
5. Webhooks are outbound-only notifications for record create/update/delete events, signed with HMAC SHA256 headers `X-Twenty-Webhook-Signature` and `X-Twenty-Webhook-Timestamp`. A receiving service must answer with a `2xx` status to acknowledge delivery. [S3]

## Recommendations

- Prefer REST for a first-pass CLI if the goal is straightforward object CRUD and import/export flows.
- Use metadata GraphQL early to discover object and field names because Twenty generates APIs around the workspace schema, including custom objects.
- Design import tooling around the documented limits: 100 requests/minute and 60 records/batch.
- Treat docs and repo behavior together: docs explain intent, while repo integration tests reveal concrete route/query contracts.

## Confidence
Medium-high. Core claims are supported by official docs plus official repo code/tests. Confidence is lower on undocumented or weakly documented surfaces such as Metadata REST and any public SDK status.

## Open Questions

- Public stability of Metadata REST.
- Public support status of `twenty-sdk`.
- Full public contract for REST upsert operations.
