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
uses `github.com/fastygo/formset` types plus namespaced Codex rules.

The migration profile is:

- `string`, `text`, `boolean`, `number`, `datetime`, `json`, `collection`,
  `object`, `richtext`, and `markdown` map to their same-named FormSet types;
- `integer`, `decimal`, and `money` map to `FieldNumber` plus the matching
  `fastygo.codex/` rule;
- `date` maps to `FieldDateTime` plus `fastygo.codex/date`;
- `uri` and `uuid` map to `FieldString` plus their semantic rule;
- `enum` maps to `FieldSelect` with ordered FormSet options;
- `nullable` and `read-only` map to namespaced rules; required plus nullable is
  invalid;
- `sensitive` and `localized` map directly to FormSet flags;
- `media` maps to a relation field with `schema.UIHintMedia` and an explicitly
  declared media target resource;
- GoBackend one/many relation values map to FormSet one-to-one/array
  cardinalities, with the relation ID equal to the field ID.

GoBackend's separate `Fields` and `Form` lists become one ordered
`RecordType.Fields` list. The adapter unions by field ID, retains storage
semantics from `Fields`, overlays compatible presentation metadata from
`Form`, and rejects incompatible duplicate declarations. Built-in localized
`title`, `content`, and `excerpt` declarations map to the corresponding Entry
chrome and locale documents.

No lossy automatic conversion belongs in Codex. The adapter is implemented and
proven in GoBackend when that repository adopts a released version.

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
and taxonomy IDs. Locale map keys must already equal their
trimmed lowercase normalization. GoBackend migration must normalize and reject
collisions before constructing public contract values.

### Resource and delivery split

GoBackend `Collection`, `RESTVisible`, and `GraphQLVisible` are delivery
configuration. They do not map into public `schema.Resource`. GoBackend keeps
those values in its delivery manifest projection. `Public` maps to
`Record.Visibility` only when it describes resource visibility rather than a
route.

GoBackend currently injects post/page/menu/setting resources. Codex does not.
The GoBackend bootstrap may retain that defaulting behavior before it
constructs and validates a public manifest.

### Exported validation

Public validation failures expose `validation.Error` codes and paths. Error
text remains diagnostic; adapters branch only on codes.

## GoBackend adoption gate

Before GoBackend imports Codex:

1. implement the field/form union adapter above without loss;
2. compare durable JSON fixtures;
3. run storage adapter tests for SQLite, PostgreSQL, and bbolt;
4. run lifecycle, revision, taxonomy, authz, REST, and conformance suites;
5. update GoBackend ADRs and contract documentation;
6. pin a released Codex version without a local `replace`.

## Pre-v1 dependency

Codex currently exposes FormSet contract types from a pre-v1 release. Codex
therefore remains pre-v1 until FormSet compatibility is stable and GoBackend
adoption proves the combined API. Consumers must pin an exact Codex version.
