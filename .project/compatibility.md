# GoBackend compatibility

Codex starts from the protocol-neutral behavior currently implemented in
GoBackend. This document is a migration map, not evidence that GoBackend has
already adopted the module.

| Current GoBackend package | Public Codex package |
| --- | --- |
| `internal/domain/content` | `content` |
| `internal/domain/schema` | `schema` |
| `internal/domain/taxonomy` | `taxonomy` |
| `internal/domain/revision` | `revision` |

## Deliberate changes

### FormSet schema ownership

GoBackend currently defines a second field/relation vocabulary. Public Codex
uses `github.com/fastygo/formset` types instead. Migration requires an explicit
adapter or manifest update for:

- `integer`, `decimal`, `money`, `date`, `uri`, `uuid`, `enum`, and `media`
  GoBackend types that are not identically named in current FormSet;
- FormSet `select`, `textarea`, `computed`, `encrypted`, rules, indexing,
  scopes, capabilities, and relation policies absent from GoBackend schema;
- GoBackend field-level relation metadata versus FormSet record relations.
- GoBackend's separate `Fields` and `Form` lists versus Codex's single FormSet
  `RecordType.Fields` contract. Migration must define one non-lossy ordered
  field list and retain storage-only/read-only intent through field metadata.

No lossy automatic conversion belongs in Codex.

Current FormSet string vocabularies are extensible: validation accepts
non-empty custom field types, scopes, and cardinalities. Codex preserves that
owner decision. GoBackend or a UI adapter may enforce its own closed supported
profile without changing the shared declaration.

### Stable JSON

GoBackend separates domain structs from `internal/persist` DTOs. Public Codex
adds stable JSON tags to contract types. GoBackend storage migration must
compare persisted fixtures before replacing its DTOs.

### Deterministic locale fallback

Public Codex sorts otherwise unordered fallback locale keys. This removes map
iteration nondeterminism and may change fallback selection for entries that
contain multiple locales while neither requested nor configured fallback
exists.

Public Codex also returns an error from `MergeLocales` when differently written
keys normalize to the same locale. GoBackend's current merge helper has no
error result, so adoption requires callers to handle this validation failure.

### Canonical identifiers and locales

Public Codex uses one lowercase ASCII identifier grammar for kinds, resources,
collections, and taxonomy IDs. Locale map keys must already equal their
trimmed lowercase normalization. GoBackend migration must normalize and reject
collisions before constructing public contract values.

### Exported validation

Taxonomy and revision validation methods are public. Error text is diagnostic,
not a machine protocol; consumers should not branch on it.

## GoBackend adoption gate

Before GoBackend imports Codex:

1. map every manifest field type without loss;
2. compare durable JSON fixtures;
3. run storage adapter tests for SQLite, PostgreSQL, and bbolt;
4. run lifecycle, revision, taxonomy, authz, REST, and conformance suites;
5. update GoBackend ADRs and contract documentation;
6. pin a released Codex version without a local `replace`.

## Pre-v1 dependency

Codex currently exposes FormSet contract types from a pre-v1 release. Codex
therefore remains pre-v1 until FormSet compatibility is stable and GoBackend
adoption proves the combined API. Consumers must pin an exact Codex version.
