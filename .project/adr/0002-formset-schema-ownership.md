# ADR-0002: FormSet owns fields and relations

- Status: Accepted
- Date: 2026-09-09

## Context

GoBackend and FormSet currently contain overlapping field and relation types.
Publishing a third copy in Codex would preserve ambiguity rather than create a
single contract.

## Decision

Codex `schema.Resource.Record` is a `formset.RecordType` and uses
`[]formset.Field` for editor form slots. Field types, nested fields, options,
rules, sensitivity, encryption hints, relations, cardinality, delete behavior,
scope, and capabilities remain owned by FormSet.

Codex validates resource identifiers, taxonomies, field placement, and
cross-resource references around FormSet's own validation and review. Route
collection names are not part of the protocol-neutral resource.

FormSet string vocabularies remain extensible. Codex does not reinterpret an
unknown non-empty field type, scope, or cardinality as invalid when the pinned
FormSet release accepts it. A product or delivery profile may impose a closed
supported vocabulary at its own boundary.

Current GoBackend semantics that narrow a generic FormSet renderer type use
namespaced validation rules (`fastygo.codex/integer`, `decimal`, `money`,
`date`, `uri`, `uuid`, `nullable`, `read-only`, and `json-any`). Relation IDs
equal their field IDs. Media is a normal relation with a media UI hint.

The initial release pins the published FormSet commit required by the contract
and uses no local module replacement.

## Consequences

- Downstream UI and backend adapters see one field vocabulary.
- Codex has a small public dependency on FormSet.
- GoBackend migration has an explicit non-lossy mapping profile for field
  types that differ today; its adapter remains GoBackend-owned.
- Codex does not silently translate unsupported field semantics.

## Rejected alternatives

- Copy GoBackend `schema.Field` into Codex.
- Copy FormSet types under new names.
- Use `map[string]any` as the schema contract.
- Change FormSet within this repository.
