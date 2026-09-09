# ADR-0006: Stable validation identity

- Status: Accepted
- Date: 2026-09-09

## Context

Human error text is useful for diagnostics but unstable for adapters and
conformance suites. Manifest bytes also vary when set-like declarations are
ordered differently.

## Decision

Public validation APIs return `validation.Error` values with stable codes and
paths. Human messages are not compatibility identifiers.

`Manifest.Digest` validates the manifest, canonicalizes set-like values,
serializes canonical JSON, and returns a versioned SHA-256 identifier with the
`codex-manifest/v1:sha256:` prefix.

FormSet field and option order remains semantic and is not sorted.

## Consequences

- Consumers can assert failure behavior without matching prose.
- Schema identity is reproducible across adapters.
- Any future canonical serialization change requires a new digest prefix.

## Rejected alternatives

- Let consumers parse error strings.
- Hash input JSON before validation.
- Sort presentation-significant field and option order.
