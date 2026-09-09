# ADR-0001: Extract a public protocol-neutral Codex kernel

- Status: Accepted
- Date: 2026-09-09

## Context

GoBackend owns the current Codex domain under Go `internal/` packages.
Products cannot import those packages and should not depend on backend
delivery, auth, or storage merely to share content semantics.

## Decision

Create `github.com/fastygo/codex` with public `content`, `schema`, `taxonomy`,
`revision`, and `conformance` packages.

The module contains only protocol-neutral values, validation,
canonicalization, and fixtures. GoBackend and products depend on Codex; Codex
never depends on them.

## Consequences

- Products can model `message`, `conversation`, or another kind as the same
  Entry aggregate used by GoBackend.
- GoBackend needs a separately reviewed migration before duplicate internal
  types can be removed.
- Public type and JSON compatibility now require semantic versioning.

## Rejected alternatives

- `fastygo/entity`: too broad for content lifecycle and taxonomy semantics.
- Copy GoBackend internal packages into each product: guaranteed drift.
- Make products import GoBackend: couples domain to server behavior.
- Put REST, auth, or storage in this module: violates protocol neutrality.
