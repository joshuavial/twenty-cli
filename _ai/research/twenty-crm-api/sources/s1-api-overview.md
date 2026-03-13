# Source: Twenty API Overview

**URL**: https://raw.githubusercontent.com/twentyhq/twenty/main/packages/twenty-docs/developers/extend/api.mdx
**Retrieved**: 2026-03-13T10:31:41Z
**Relevance**: Primary public documentation for API families, endpoints, auth, rate limits, playground, and batching.

## Summary

- Twenty documents four API surfaces conceptually: core REST, core GraphQL, metadata REST, and metadata GraphQL.
- Core endpoints are documented under `/rest/` and `/graphql`; metadata under `/rest/metadata/` and `/metadata/`.
- Cloud traffic is directed to `https://api.twenty.com/`.
- Auth is bearer-token API key auth.
- API keys can be assigned roles.
- Public docs state batch size is 60 records and rate limit is 100 calls/minute.
- The most accurate docs are workspace-specific and appear in Settings after creating an API key.

## Reliability Notes

- Official Twenty docs from the monorepo.
- High reliability for product intent and public contract.
- Generic rather than tenant-specific, so it omits workspace-specific object names and fields.
