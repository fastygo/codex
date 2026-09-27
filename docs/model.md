# Codex model

## Entry

`content.Entry` is the one aggregate for every content record. `Entry.Kind`
identifies its manifest resource. The aggregate owns identity, lifecycle,
visibility, built-in localized chrome, metadata, locale documents, taxonomy
references, optimistic version, and timestamps.

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
				ID:         "content",
				Label:      "Content",
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

The resource declaration defines one storage rule:

- `Field.Localized == true` places the value in
  `Entry.Locales[locale].Data[field.ID]`;
- every other declared field is placed in
  `Entry.Metadata[field.ID].Value`;
- relation fields are never localized.

The built-in `slug`, `title`, `content`, and `excerpt` fields are mirrored by
their `Entry` maps. If a resource declares one of those IDs as localized and a
locale document also carries it, both values must match. This preserves the
existing entry chrome while keeping one resource-aware validation rule.

`Resource.ValidateEntry` checks declared field types and Codex semantic rules.
`Resource.PublicProjection` removes private metadata and every sensitive field,
including localized data.

## Codex field profile

Names that are narrower than the base field type use namespaced rules:

- `integer`, `decimal`, and `money` use `schema.FieldNumber`;
- `date` uses `schema.FieldDateTime`;
- `uri` and `uuid` use `schema.FieldString`;
- JSON that may contain any JSON scalar or container uses `schema.FieldJSON`;
- `nullable` and `read-only` remain explicit policy rules;
- `enum` uses `schema.FieldSelect` and ordered `Options`;
- object and collection fields retain nested `Fields` and `Items`;
- media is a `schema.FieldRelation` with `UIHintMedia` and an explicitly
  declared media target resource.

Rule constants use the `fastygo.codex/` namespace. Unknown non-empty field,
scope, rule, and visibility values remain round-trippable.

`read-only` is mutation policy, not value validity. `ValidateEntry` accepts a
stored read-only value; application mutation adapters must reject client
changes to it.

## Relations

A relation is declared once in `Record.Relations`. Its ID equals the
top-level relation field ID, and its source equals the owning record ID.
One-to-one relations store one entry ID. One-to-many and many-to-many
relations store an array of entry IDs. Manifest validation proves that source
and target resources exist.

## Locales

`LocalizedText` covers built-in entry chrome. `LocaleDocument` holds one whole
localized field document. Locale fallback returns one complete document and
never mixes fields from different locales. Products whose language is
intentionally unspecified may use `und`.

## Taxonomies and revisions

A taxonomy is flat or hierarchical and explicitly lists allowed entry kinds.
Assignments join an entry identity to a term. A revision stores an immutable
entry snapshot at exactly one entry version. Persistence, referential
integrity, revision retention, restore, and authorization remain application
concerns.

## Canonical identity and validation

`Manifest.Canonical` sorts set-like declarations while preserving field and
option order. `Manifest.Digest` returns a versioned SHA-256 identity
over canonical JSON.

Validation failures expose stable `validation.Error` codes and paths.
Human-readable messages are diagnostic and are not compatibility keys.

## Conformance

The `conformance` package embeds stable Entry, resource-aware Entry, manifest,
taxonomy, and revision fixtures. It also pins canonical output and expected
failure codes. Adapters should run these fixtures without local module
replacements before claiming Codex compatibility.
