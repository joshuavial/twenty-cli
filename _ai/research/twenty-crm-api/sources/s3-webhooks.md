# Source: Twenty Webhooks

**URL**: https://raw.githubusercontent.com/twentyhq/twenty/main/packages/twenty-docs/developers/extend/capabilities/webhooks.mdx
**Retrieved**: 2026-03-13T10:31:41Z
**Relevance**: Primary public contract for webhook events, payloads, and request signing.

## Summary

- Webhooks are outbound notifications for record creation, updates, and deletion.
- Event names use object-qualified names like `person.created`.
- Payloads include `event`, `data`, and `timestamp`.
- Request signing uses HMAC SHA256 with timestamp and JSON payload.
- Signature headers are `X-Twenty-Webhook-Signature` and `X-Twenty-Webhook-Timestamp`.
- Receivers must return `2xx` to acknowledge successful delivery.

## Reliability Notes

- Official public docs and likely the intended external contract.
- High reliability for integration behavior at the feature level.
