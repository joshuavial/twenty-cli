# Twenty CRM API Details

## Scope

In scope:
- Core API shape
- Metadata API shape
- Auth and permissions
- Rate limits and batching
- Practical REST conventions from official tests
- Webhook behavior

Out of scope:
- Exhaustive workspace-specific schema docs, because Twenty generates those per workspace
- Live credentialed validation against a real tenant

## API Families

Twenty splits its API into two conceptual layers:

- Core API: record data such as people, companies, opportunities, notes, tasks.
- Metadata API: workspace schema and configuration such as objects, fields, and related admin structures.

The official docs state these endpoint roots:

- Core REST: `/rest/`
- Core GraphQL: `/graphql`
- Metadata REST: `/rest/metadata/`
- Metadata GraphQL: `/metadata`

On cloud, the base host is `https://api.twenty.com/`. For self-hosted deployments, the API sits under the instance domain. [S1] [S5]

## Authentication And Permissions

All reviewed API surfaces use bearer auth:

```http
Authorization: Bearer YOUR_API_KEY
```

The docs also show that API keys can be assigned a role. That matters for a CLI because access is not just key presence; the role constrains what the key can read or mutate. [S1]

Repo REST authentication tests add useful edge behavior:

- Missing token can yield `403` with a missing-token message.
- Invalid or expired tokens can yield `401` with `UNAUTHENTICATED`.

That means client code should not assume all auth failures normalize to one status code. [S4]

## Rate Limits, Batch Size, And Import Planning

The official docs consistently describe:

- 100 API calls per minute
- 60 records per batch call

The import guide converts that into an approximate maximum throughput of about 6,000 records/minute and recommends API-based imports especially for recurring syncs, real-time integrations, and large data sets above 50,000 records. [S1] [S2]

Practical implication for a CLI:

- Use batch endpoints wherever possible.
- Implement pacing of roughly 600 ms/request for sustained imports.
- Back off on `429`.
- Log record IDs and failures so imports can resume safely.

## Core REST API: Practical Contract

The official docs describe REST broadly, but the repo integration tests make the concrete contract much clearer.

### Route Prefix And Object Naming

All tested REST requests are rooted at `/rest`. Example object routes:

- `GET /rest/people`
- `POST /rest/people`
- `PATCH /rest/people/{id}`
- `DELETE /rest/people/{id}`
- `POST /rest/batch/people`
- `POST /rest/people/duplicates`
- `GET /rest/opportunities/groupBy`

Objects use human-readable plural names like `people`, `companies`, `opportunities`, not opaque numeric route IDs. This aligns with the docs claim that Twenty generates APIs around workspace object names. [S1] [S4]

### Response Shape

Observed REST responses are action-oriented rather than raw-resource oriented:

- Find many: `data.{pluralObjectName}`, plus `pageInfo` and `totalCount`
- Create one: `data.createPerson`
- Update one: `data.updatePerson`
- Delete one: `data.deletePerson`

So if you are building a generic CLI, response parsing should be object-operation aware, not just status-code aware. [S4]

### Query Parameters Confirmed In Tests

Confirmed collection query parameters:

- `limit`
- `filter`
- `order_by`
- `starting_after`
- `ending_before`
- `depth`

Confirmed examples:

- `GET /rest/people?limit=10`
- `GET /rest/people?filter=position[lte]:1`
- `GET /rest/people?filter=companyId[in]:["<uuid>"]`
- `GET /rest/people?order_by=position[AscNullsLast]`
- `GET /rest/people?starting_after=<cursor>&limit=1`
- `GET /rest/people?ending_before=<cursor>&limit=2`

Cursor pagination is real, and the response includes `pageInfo.startCursor`, `pageInfo.endCursor`, and `pageInfo.hasNextPage`. [S4]

### Relation Expansion With `depth`

Official REST tests confirm:

- `depth=0`: return relation IDs only
- `depth=1`: expand one relation level
- `depth=2`: rejected with `400`

That is useful for CLI design:

- Use `depth=0` for bulk sync and compact payloads.
- Use `depth=1` for convenience reads.
- Do not assume arbitrary recursive expansion. [S4]

### Batch Operations

Confirmed REST batch create route:

- `POST /rest/batch/people`

Docs say batch operations support up to 60 records and cover create, update, delete, and upserts broadly, but the clearest implementation evidence gathered here was batch create in REST plus docs-level statements for the rest. Treat REST batch create as confirmed and the remaining batch verbs as likely but worth live validation before relying on them in production. [S1] [S4]

### Duplicate Detection

There is a dedicated duplicate-finding route:

- `POST /rest/people/duplicates`

Tested request modes:

- `{ "data": [ ... ] }`
- `{ "ids": [ ... ] }`

This is not standard CRUD and is worth remembering for migration tooling that wants pre-insert duplicate checks. [S4]

### Group By

There is a tested group-by route:

- `GET /rest/opportunities/groupBy`

Observed parameters include:

- `group_by`
- `aggregate`
- `filter`
- `include_records_sample`
- `limit`

This suggests the REST surface includes reporting/aggregation helpers, not just row-level CRUD. [S4]

### Bulk Delete Guardrails

Delete-many and destroy-many operations require filters. The integration tests explicitly check that bulk delete requests without filters fail. That is a sensible safety feature and means a CLI should require an explicit filter expression for destructive bulk actions. [S4]

## Data Modeling Conventions

The REST tests also reveal that record payloads are not always flat:

- Composite/nested fields exist, such as `name.firstName`, `name.lastName`, `emails.primaryEmail`, and `domainName.primaryLinkUrl`.
- Relations are commonly set through foreign-key style fields like `companyId`.

This matters for import tooling: flattening everything to primitive columns will miss actual field structure. [S4]

## GraphQL Core API

The docs state the core GraphQL endpoint is `/graphql`. Official repo test utilities confirm that path directly. [S1] [S4]

The docs also make two important GraphQL claims:

- GraphQL can fetch related data in one call.
- GraphQL supports batch upsert, which is presented as a differentiator versus REST.

Because Twenty generates the schema from the workspace model, a GraphQL-first integration can be more adaptable if custom objects and relation traversal are central requirements. [S1]

## Metadata GraphQL API

The strongest metadata evidence gathered here is GraphQL.

Official repo test utilities confirm:

- Metadata GraphQL endpoint: `POST /metadata`
- Object creation mutation name: `createOneObject`
- Field creation mutation name: `createOneField`
- Field listing query shape: `fields(filter: ..., paging: ...) { edges { node { ... }}}`

Observed object creation input shape includes fields like:

- `nameSingular`
- `namePlural`
- `labelSingular`
- `labelPlural`
- `description`
- `icon`
- `isLabelSyncedWithName`

Observed field creation input shape includes:

- `name`
- `label`
- `type`
- `objectMetadataId`
- `isLabelSyncedWithName`
- `options` for select fields

Observed field type usage includes at least `TEXT` and `SELECT`. [S5]

For a CLI, this means metadata automation is viable, but the implementation should probably target GraphQL first unless a public Metadata REST contract is documented in the workspace playground.

## Webhooks

Twenty webhooks are outbound notifications triggered by record changes.

Confirmed characteristics:

- Delivery method: `POST`
- Payload format: JSON
- Event categories: created, updated, deleted
- Event names follow object-qualified patterns such as `person.created`
- Authentication/verification headers:
  - `X-Twenty-Webhook-Signature`
  - `X-Twenty-Webhook-Timestamp`
- Signature method: HMAC SHA256 over `{timestamp}:{JSON payload}`

Operational implications:

- The receiver must preserve the exact payload bytes or carefully reconstruct the JSON before validating.
- The receiver should reject stale timestamps to limit replay risk.
- The receiver should return `2xx` to acknowledge receipt; non-2xx responses are treated as delivery failures. [S3]

## Documentation Gaps And Inferences

### Workspace-Specific Docs Matter

The official docs repeatedly note that the most accurate API documentation is generated inside each workspace under Settings. That means public docs are intentionally generic. Any serious implementation should inspect the live workspace API playground before hard-coding object and field expectations. [S1]

### Metadata REST Exists But Was Not Well Exposed Publicly

The overview docs state `/rest/metadata/` exists, but the evidence gathered here was much stronger for metadata GraphQL than for Metadata REST. I would treat Metadata REST as possible but under-documented until validated in a live workspace. [S1] [S5]

### SDK Status Is Ambiguous

Public docs still say there is no Python or Node.js SDK. However, the official repo contains `packages/twenty-sdk` with a generated metadata GraphQL schema. The safest interpretation is:

- There may be an internal or emerging SDK artifact in the repo.
- Publicly supported SDK guidance is still effectively "use HTTP clients directly" unless confirmed otherwise. [S2] [S6]

## Recommended CLI Strategy

If the goal is to build `twenty_crm_cli`, the lowest-risk sequence is:

1. Use core REST for standard object CRUD and imports.
2. Use metadata GraphQL to inspect or manage schema when object names and fields are not known ahead of time.
3. Design a generic query builder around:
   - object plural route names
   - `filter`
   - `order_by`
   - cursor pagination
   - `depth`
4. Add safety constraints for destructive operations because bulk delete requires explicit filters.
5. Reserve GraphQL core operations for cases where relation-rich reads or upsert semantics are more important than REST simplicity.

## Source References

- [S1](./sources/s1-api-overview.md)
- [S2](./sources/s2-import-guide.md)
- [S3](./sources/s3-webhooks.md)
- [S4](./sources/s4-rest-tests.md)
- [S5](./sources/s5-metadata-graphql.md)
- [S6](./sources/s6-generated-schema.md)
