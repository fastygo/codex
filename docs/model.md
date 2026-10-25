# Codex model

## Entry

`content.Entry` is the one aggregate for every content record. `Entry.Kind`
identifies its manifest resource. The record key is `(id, locale)`.
Translations are separate records that share `id`.

The aggregate owns identity, locale, lifecycle, visibility, string chrome
(`slug`, `title`, `content`, `excerpt`), metadata, taxonomy references,
`order` / `pinned` / `format` / `discussion` / `origin`, optimistic version,
and timestamps.

Storage adapters may project these values into SQL columns and indexes, but
their table shape is not part of Codex.

## Resource manifest

`schema.Resource.Record` is a Codex `RecordType`. Codex owns fields, relations,
scopes, capabilities, options, and entry-field validation. A form package may
project those fields into editor slots. Codex does not import that package.

```go
message := schema.Resource{
	Record: schema.RecordType{
		ID:            "message",
		Label:         "Messages",
		SchemaVersion: "1",
		OwnerModule:   "telegram-reader",
		Scope:         schema.ScopeUser,
		Fields: []schema.Field{
			{
				ID:       "telegram_message_id",
				Label:    "Telegram message ID",
				Type:     schema.FieldNumber,
				Required: true,
				Rules: []schema.ValidationRule{
					{Name: schema.RuleInteger},
				},
			},
			{
				ID:         "blurb",
				Label:      "Blurb",
				Type:       schema.FieldText,
				Localized:  true,
				Searchable: true,
			},
		},
	},
	Taxonomies: []string{"message_type"},
}
```

Kinds are open lowercase identifiers. Constants such as `content.KindPost`
name established Codex identifiers but do not inject resources. Every
application declares exactly the resources it uses. Route collection names
and REST/GraphQL exposure are delivery configuration, not manifest fields.

## Field placement

Every declared non-chrome field is stored in `Entry.Metadata[field.ID].Value`.
`Field.Localized` means the value is translatable: records that share `id`
and differ in `locale` may disagree. `localized: false` (the default) must be
equal across those records after canonicalization, otherwise
`entry.localized_mismatch`. Relation fields are never localized.

Chrome `slug`, `title`, `content`, and `excerpt` are always localized. They
do not use `Field.Localized`.

`Resource.ValidateEntry` checks declared field types and Codex semantic rules.
`Resource.PublicProjection` removes private metadata and every sensitive
field. The `codex-entry/v1` profile does not strip sensitive fields: Entry
v0.4 has no `sensitive` member.

## Codex field profile

Names that are narrower than the base field type use namespaced rules:

- `integer`, `decimal`, and `money` use `schema.FieldNumber`;
- money is a safe integer in minor units; negatives are allowed; currency is
  an ISO 4217 code on the record or a default from settings;
- `date` uses `schema.FieldDateTime`;
- `uri` and `uuid` use `schema.FieldString`;
- JSON that may contain any JSON scalar or container uses `schema.FieldJSON`;
- `nullable` and `read-only` remain explicit policy rules;
- `enum` uses `schema.FieldSelect` and ordered `Options`;
- object and collection fields retain nested `Fields` and `Items`;
- media is a `schema.FieldRelation` with `UIHintMedia` and an explicitly
  declared media target resource.

Rule constants use the `fastygo.codex/` namespace. Unknown non-empty rule
names are `schema.field.rule_unknown`. Unknown field, scope, and visibility
values remain round-trippable on the manifest itself.

`read-only` is mutation policy, not value validity. `ValidateEntry` accepts a
stored read-only value; application mutation adapters must reject client
changes to it.

Integers must be in `±(2^53-1)`. Larger values are
`canonical.unsafe_integer`.

## Relations

A relation is declared once in `Record.Relations`. Its ID equals the
top-level relation field ID, and its source equals the owning record ID.
One-to-one relations store one entry ID. One-to-many and many-to-many
relations store an array of entry IDs. Manifest validation proves that source
and target resources exist.

## Locales

Each record carries one `locale`. `content.Pick` selects the record for an
`id` by requested locale, then fallback, then the lexicographically first
locale. Products whose language is intentionally unspecified may use `und`.

`id` is assigned once and is opaque ASCII
`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`. That grammar applies only to `id`,
not to `*_id`. An integer platform primary key is stored as `<kind>-<n>`.

## Taxonomies and revisions

A taxonomy is flat or hierarchical and explicitly lists allowed entry kinds.
Term and definition labels are strings plus `locale`. Assignments join an
entry identity to a term. A revision stores an immutable entry snapshot at
exactly one `(entry_id, locale, version)`. Persistence, referential
integrity, revision retention, restore, and authorization remain application
concerns.

## Canonical identity and validation

`canonical.Canonicalize` / `canonical.Digest` implement profile
`codex-entry/v1` (RFC 8785 JCS). Official vectors are vendored from
`@ui8kit/contract` commit `8299fee`.

`Manifest.Canonical` sorts set-like declarations while preserving field and
option order. `Manifest.Digest` is `codex-manifest/v1` over Go
`encoding/json`. `Manifest.DigestV2` is `codex-manifest/v2` over JCS.

Validation failures expose stable `validation.Error` codes and paths.
Human-readable messages are diagnostic and are not compatibility keys.

## Conformance

The `conformance` package embeds stable Entry, resource-aware Entry, manifest,
taxonomy, and revision fixtures. `conformance/contract` pins the TypeScript
schemas and the 24 official canonical vectors. Adapters should run these
fixtures without local module replacements before claiming Codex
compatibility.
