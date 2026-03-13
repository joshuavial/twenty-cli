# Research Status: Twenty CRM API

## Objective
Build a practical understanding of the Twenty CRM API surface, including core record APIs, metadata APIs, auth, rate limits, webhook behavior, and implementation-level route/query conventions that matter for a CLI integration.

## Requester
Current repo work in `crew/conduit`; output should be reusable notes under `_ai`.

## Timeline
- 2026-03-13T10:31:41Z - Research initiated.
- 2026-03-13T10:31:41Z - Confirmed official public docs for API overview, import guidance, and webhooks.
- 2026-03-13T10:31:41Z - Pulled implementation details from the official `twentyhq/twenty` repository, including REST integration tests and metadata GraphQL test helpers.
- 2026-03-13T10:31:41Z - Wrote durable notes and source summaries.

## Open Questions
- [ ] Whether the Metadata REST API has stable public documentation equivalent to the GraphQL metadata surface.
- [ ] Whether `packages/twenty-sdk` is an internal/generated SDK only or a supported public client package.
- [ ] Whether batch upsert is exposed publicly in REST as strongly as the docs imply, since the clearest implementation evidence found here was for GraphQL and generic docs.

## Follow-Up Ideas
- Pull a live workspace schema from the API playground once credentials are available.
- Validate the exact error payloads and pagination behavior against `api.twenty.com` rather than repo tests alone.
- Prototype a minimal CLI flow against `/rest/people`, `/rest/companies`, and `/metadata` to confirm auth and role constraints.
