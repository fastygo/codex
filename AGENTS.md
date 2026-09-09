# Agent notes

Codex is the public protocol-neutral content contract for FastyGo products.
It owns entries, lifecycle, manifest resources, taxonomies, revisions,
localization, validation, canonicalization, and conformance fixtures.

Read `.project/README.md`, `.project/intent.md`, and
`.project/architecture.md` before changing public types.

## Rules

1. Do not add HTTP, REST, GraphQL, auth, sessions, CSRF, repositories,
   databases, migrations, blob storage, UI, or product manifests.
2. FormSet owns fields, relations, scopes, capabilities, and form binding.
   Codex must not create a parallel field vocabulary.
3. Product kinds remain manifest data. Core defaults are only `post`, `page`,
   `menu`, and `setting`.
4. Public JSON names, lifecycle values, and canonicalization are compatibility
   contracts. Pre-v1 breaking changes require migration notes and a minor
   release; after v1 they require a major version.
5. Do not import GoBackend or use local module replacements.
6. Comments and documentation are written in English.

Run before completion:

```text
go test ./...
go vet ./...
gofmt -w .
```
