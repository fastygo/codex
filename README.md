# Codex

`github.com/fastygo/codex` is the public, protocol-neutral content contract for
FastyGo applications.

Codex provides one WordPress-like entry aggregate with manifest-backed kinds,
fields and relations, taxonomies, revisions, localization, validation, and
deterministic RFC 8785 canonicalization. A `Kind` is the equivalent of a
`post_type`: products can declare `post`, `page`, `message`, `conversation`,
or another kind without creating a parallel content model.

The current contract is Codex data **v0.4.1**, aligned with `@ui8kit/contract`
0.2.0 at commit `a70c243`. The Go module is not tagged until the contract
owner confirms the official vectors.

## Ownership

Codex owns:

- content identity, lifecycle, visibility, locale, and term refs;
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
content/      Entry, lifecycle, visibility, locale, records
canonical/    RFC 8785 JCS; entry/term/settings/manifest digests
schema/       Manifest, fields, relations, entry validation
validate/     Collection validator (ISSUE_CODES_V04 + Q7)
taxonomy/     Definitions, terms, assignments, hierarchy validation
revision/     Immutable Entry snapshots
migrate/      v0.3 → v0.4 JSON conversion
conformance/  Stable fixtures, contract pin, compatibility checks
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
	Locale:     "und",
	Kind:       "message",
	Status:     content.StatusPublished,
	Visibility: content.VisibilityPrivate,
	Slug:       "message-1001",
	Title:      "Message 1001",
	Content:    "Hello",
	Version:    1,
	CreatedAt:  sentAt,
	UpdatedAt:  sentAt,
}
resource := schema.Resource{Record: schema.RecordType{
	ID: "message", Label: "Messages", Scope: schema.ScopeUser,
	Fields: []schema.Field{
		{ID: "blurb", Label: "Blurb", Type: schema.FieldText, Localized: true},
	},
}}
if err := resource.ValidateEntry(entry); err != nil {
	return err
}
digest, err := canonical.Digest(string(jsonBytes))
```

See [`docs/model.md`](docs/model.md) for the complete model and
[`.project/README.md`](.project/README.md) for the contract and decisions.
Existing consumers follow the
[`v0.2.0`](docs/migrations/v0.2.0.md),
[`v0.3.0`](docs/migrations/v0.3.0.md),
[`v0.4.0`](docs/migrations/v0.4.0.md), and
[`v0.4.1`](docs/migrations/v0.4.1.md) migration notes.

## Verify

```bash
go test -race -count=1 ./...
go vet ./...
gofmt -w .
```
