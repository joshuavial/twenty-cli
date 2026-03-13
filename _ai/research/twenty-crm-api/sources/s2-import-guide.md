# Source: Import Data via API

**URL**: https://raw.githubusercontent.com/twentyhq/twenty/main/packages/twenty-docs/user-guide/data-migration/how-tos/import-data-via-api.mdx
**Retrieved**: 2026-03-13T10:31:41Z
**Relevance**: Practical guidance for import scale, throughput, ordering, retries, and API choice.

## Summary

- API import is recommended for recurring imports, real-time syncs, integrations, and very large data volumes.
- The docs suggest CSV for smaller one-off imports and API for larger or automated flows.
- Import order matters when relations exist; companies before people is the example ordering.
- The guide recommends batching, rate-limit handling, logging, and test runs before full imports.
- Public docs still state there is not currently a Python or Node SDK.

## Reliability Notes

- Official Twenty docs focused on migration workflows rather than exact protocol details.
- Useful for operational planning and product guidance.
- Lower precision than integration tests for exact route shapes.
