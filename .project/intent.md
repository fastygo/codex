# Intent

## Problem

GoBackend contains the canonical WordPress-like Codex model under
`internal/domain`. Go's internal import boundary correctly prevents products
from consuming it. FormSet publicly defines fields, relations, and form
binding, while SvelteCMS receives client projections. A desktop product that
needs Codex-shaped entities would otherwise duplicate the domain and drift.

## Outcome

Publish `github.com/fastygo/codex` as the single protocol-neutral Go contract
for:

- entries and content lifecycle;
- manifest-backed resource kinds;
- FormSet-backed fields, relations, scopes, and capabilities;
- taxonomy definitions, terms, and assignments;
- revision snapshots;
- localization, validation, canonicalization, and conformance.

GoBackend will later consume this module through a dedicated migration. New
products may consume Codex immediately without importing GoBackend.

## In scope

- public Go packages with stable JSON representations;
- deterministic canonicalization and validation;
- resource-aware Entry validation and sensitivity projection;
- a lossless FormSet profile for current GoBackend field semantics;
- versioned manifest digests and machine-readable validation failures;
- reproducible FormSet dependency from GitHub;
- conformance fixtures and tests;
- migration documentation for current GoBackend internal types.

## Out of scope

- REST, GraphQL, HTTP envelopes, routing, or OpenAPI;
- authentication, authorization, sessions, or CSRF;
- repositories, SQL, bbolt, migrations, media blobs, or backups;
- Wails, Svelte, Templ, or another renderer;
- Telegram or another product manifest;
- changing GoBackend or FormSet in this repository.
- route collection names, REST/GraphQL exposure, or injected core resources.

## Compatibility promise

Codex remains pre-v1 while its public FormSet dependency is pre-v1 and the
GoBackend migration is unproven. During `v0.x`, breaking changes require an
explicit migration note and a new minor release. After the first stable
release, breaking type, JSON, lifecycle, or canonicalization changes require a
major version. Product kinds remain manifest data and never become hard-coded
core defaults.
