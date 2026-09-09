# ADR-0003: Publish stable JSON and deterministic canonicalization

- Status: Accepted
- Date: 2026-09-09

## Context

The same Codex values may be persisted locally, delivered by GoBackend, used
in manifests, or included in conformance fixtures. Go map iteration and
adapter-specific DTOs cannot define a stable shared contract.

## Decision

Public Codex types carry explicit snake_case JSON tags. Optional values use
`omitempty` only where absence and zero have the same contract meaning.

Manifest canonicalization returns a copy with deterministic ordering for every
set-like collection. It does not mutate its input. Locale fallback sorts
otherwise unordered available locale identifiers.

Conformance fixtures pin representative JSON. A breaking JSON name, required
field, lifecycle value, or canonical ordering change requires an explicit
pre-v1 migration note and minor release; after v1 it requires a major version.

Canonicalization is a semantic ordering helper, not canonical JSON suitable
for signatures. Signing consumers must specify their serialization profile.

## Consequences

- Storage and delivery adapters can prove compatibility against the same
  fixtures.
- Product declarations produce reproducible digests after a consumer chooses
  a deterministic JSON encoder.
- Map-valued content remains outside byte-level canonicalization.

## Rejected alternatives

- Treat Go struct layout as the wire contract.
- Reuse GoBackend REST projections.
- Depend on insertion order or Go map iteration.
- Advertise ordinary `encoding/json` output as a signature format.
