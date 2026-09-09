# Codex model

## WordPress-like mapping

Codex uses one `content.Entry` aggregate for every content record.

| WordPress concept | Codex contract |
| --- | --- |
| `wp_posts` row | `content.Entry` |
| `post_type` | `content.Entry.Kind` / `schema.Resource.ID` |
| post status | `content.Status` |
| post visibility | `content.Visibility` |
| post meta | `Entry.Metadata` and FormSet fields |
| taxonomy | `taxonomy.Definition` |
| term | `taxonomy.Term` |
| term relationship | `taxonomy.Assignment` and `Entry.Terms` |
| revision | `revision.Revision` |
| custom post type registration | `schema.Manifest.Resources` |

Kinds are open lowercase identifiers. Codex reserves core definitions for
`post`, `page`, `menu`, and `setting`; products own all additional kinds.

## Entry

An Entry contains common content chrome:

- stable ID and kind;
- lifecycle status and visibility;
- localized slug, title, content, and excerpt;
- optional author, parent, featured media, and template references;
- typed-by-manifest metadata and locale documents;
- taxonomy term references;
- optimistic version and timestamps.

Storage adapters may project these values into SQL columns and indexes, but
their table shape is not part of this module.

## Resource manifest

A `schema.Resource` embeds `formset.RecordType`. This means the same public
FormSet declaration owns fields, relations, scope, capabilities, validation
hints, sensitivity, indexing hints, and schema version.

```go
message := schema.Resource{
	RecordType: formset.RecordType{
		ID:            "message",
		Label:         "Messages",
		SchemaVersion: "1",
		OwnerModule:   "telegram-reader",
		Scope:         formset.ScopeUser,
		Fields: []formset.Field{
			{
				ID:       "telegram_message_id",
				Label:    "Telegram message ID",
				Type:     formset.FieldNumber,
				Required: true,
				Indexed:  true,
			},
			{
				ID:         "content",
				Label:      "Content",
				Type:       formset.FieldText,
				Searchable: true,
			},
		},
	},
	Collection: "messages",
	Taxonomies: []string{"message_type"},
}
```

The application decides how FormSet values map into Entry locale documents or
metadata. Codex validates the shared declarations but does not render forms or
persist their data.

## Relations

Relations are FormSet declarations between resource IDs. Cardinality and
delete behavior describe policy; applications enforce that policy. Codex
manifest validation proves that declared source and target resources exist.

## Locales

`LocalizedText` covers built-in content chrome. `LocaleDocument` holds one
whole FormSet data document per locale. Locale fallback returns one complete
document and never mixes fields from different locales.

Use the BCP 47 language tag appropriate to the product. Products with content
whose language is intentionally unspecified may use `und`.

## Taxonomies

A taxonomy is either flat or hierarchical and explicitly lists allowed Entry
kinds. Hierarchical validation rejects missing parents and cycles. An
assignment joins one Entry identity to one term; persistence and referential
integrity remain adapter responsibilities.

## Revisions

A revision stores an immutable Entry snapshot at exactly one Entry version.
Creation, retention, restore, and authorization remain application concerns.

## Conformance

The `conformance` package embeds stable Entry and manifest fixtures. Storage or
delivery adapters should decode, round-trip, and validate those fixtures in
their own test suites before claiming Codex compatibility.
