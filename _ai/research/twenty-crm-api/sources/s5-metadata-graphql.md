# Source: Twenty Metadata GraphQL Tests And Utilities

**URL**: Multiple official repo files under `packages/twenty-server/test/integration/metadata/`
**Retrieved**: 2026-03-13T10:31:41Z
**Relevance**: Concrete evidence for the metadata endpoint and the names/shapes of core metadata operations.

## Summary

- Metadata GraphQL requests are posted to `/metadata`.
- Confirmed mutation names include `createOneObject` and `createOneField`.
- Confirmed field-list query shape uses a connection model: `fields(filter:, paging:) { edges { node { ... }}}`.
- Object metadata inputs include naming, labels, descriptions, icons, and label-sync settings.
- Field metadata inputs include name, label, type, object association, and select options.

## Reliability Notes

- Official test helpers and integration specs from the source repo.
- Strong evidence for the GraphQL metadata surface.
- Less helpful for undocumented REST metadata paths.
