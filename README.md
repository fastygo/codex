# Codex

`github.com/fastygo/codex` is the public, protocol-neutral content contract for
FastyGo applications.

Codex provides one WordPress-like entry aggregate with manifest-backed kinds,
fields and relations, taxonomies, revisions, localization, validation, and
deterministic canonicalization. A `Kind` is the equivalent of a `post_type`:
products can declare `post`, `page`, `message`, `conversation`, or another
kind without creating a parallel content model.

## Ownership

Codex owns:

- content identity, lifecycle, visibility, locale documents, and term refs;
- manifest resources, fields, and relations;
- taxonomy definitions, terms, and assignments;
- immutable revision snapshots;
- validation, canonicalization, and conformance fixtures.

Codex does not own HTTP, REST, GraphQL, authentication, sessions, CSRF,
databases, blob storage, UI rendering, product manifests, or application
services. It does not register default resources or define route collection
names. Those stay in GoBackend and product repositories.

## Packages

```text
content/      Entry, lifecycle, visibility, locales
schema/       Manifest, fields, relations, entry validation
taxonomy/     Definitions, terms, assignments, hierarchy validation
revision/     Immutable Entry snapshots
conformance/  Stable fixtures and compatibility checks
validation/   Machine-readable validation codes and paths
```

## Install

Codex is pre-v1 while the GoBackend migration is being proven. Pin an exact
published version:

```bash
go get github.com/fastygo/codex@<version>
```

## Example

```go
sentAt := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
entry := content.Entry{
	ID:         "telegram:group:42:1001",
	Kind:       "message",
	Status:     content.StatusPublished,
	Visibility: content.VisibilityPrivate,
	Slug:       content.LocalizedText{"und": "message-1001"},
	Title:      content.LocalizedText{"und": "Message 1001"},
	Content:    content.LocalizedText{"und": "Hello"},
	Version:    1,
	CreatedAt:  sentAt,
	UpdatedAt:  sentAt,
}
resource := schema.Resource{Record: schema.RecordType{
	ID: "message", Label: "Messages", Scope: schema.ScopeUser,
	Fields: []schema.Field{
		{ID: "content", Label: "Content", Type: schema.FieldText, Localized: true},
	},
}}
if err := resource.ValidateEntry(entry); err != nil {
	return err
}
```

See [`docs/model.md`](docs/model.md) for the complete model and
[`.project/README.md`](.project/README.md) for the contract and decisions.
Existing consumers follow the
[`v0.2.0`](docs/migrations/v0.2.0.md) and
[`v0.3.0`](docs/migrations/v0.3.0.md) migration notes.

## Verify

```bash
go test -count=1 ./...
go vet ./...
```
