# ADR-0007: Codex owns content fields

- Status: Accepted
- Date: 2026-09-27

## Context

ADR-0002 embedded `formset.RecordType` in `schema.Resource`. Every consumer of a manifest then imported FormSet, including products that only store and validate entries. FormSet is the editor form projection, not the content contract.

## Decision

Codex owns resource fields, relations, scopes, capabilities, and entry-field validation. `schema.Resource.Record` is a Codex `RecordType`. The module does not import FormSet.

Unknown non-empty field types, scopes, and cardinalities stay valid. A delivery profile may apply a closed vocabulary at its own boundary. Namespaced rules (`fastygo.codex/integer` and the rest listed in `schema`) keep scalar meanings that are narrower than the base field type.

FormSet may later import a Codex tag and project these fields into editor slots. That projection is not part of this module.

## Consequences

- A manifest consumer does not compile FormSet.
- JSON field names on `record` stay the portable document shape.
- Validation codes `schema.resource.record_invalid` and `schema.entry.field_invalid` replace the previous `formset_invalid` codes.
- GoBackend adopts this module only from a tag that contains this decision.

## Rejected alternatives

- Keep FormSet as the field owner and project the other way.
- Leave a type alias that still imports FormSet.
