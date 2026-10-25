# Architecture

## Dependency direction

```text
github.com/fastygo/codex
          ↑
GoBackend, form projections, and product applications
```

Codex owns content identity, lifecycle, resources, fields, relations,
taxonomies, revisions, and localization. It has no dependency on GoBackend,
FormSet, or a delivery protocol. A form package may import Codex to project
fields into editor slots.

## Packages

### `content`

Defines `Entry`, `Kind`, `Status`, `Visibility`, string chrome, metadata
values, taxonomy references, locale pick, public projection, slug
normalization, comment/menu/dictionary/settings records, and protocol-neutral
validation. The record key is `(id, locale)`.

### `canonical`

Implements RFC 8785 JCS (`canonical.JCS`) and the profiles `codex-entry/v1`,
`codex-term/v1`, `codex-settings/v1`, and `codex-manifest/v2`, ported from
`@ui8kit/contract` `src/canonical.ts`. Go `encoding/json` is not used for
canonical output.

### `schema`

Defines `Manifest` and `Resource`. `Resource.Record` is a Codex `RecordType`.
Resource adds assigned taxonomy identifiers and content-aware Entry validation.

Manifest validation checks identifiers, uniqueness, field declarations,
relation targets, taxonomy identifiers, and field/relation identity.
Canonicalization sorts resources, relations, capabilities, and taxonomy names.
It preserves field and option order because renderers may treat that order as
presentation semantics.

`Field.Localized` means translatable. Declared non-chrome fields live in
`Entry.Metadata`. Chrome is always localized. `Resource.PublicProjection`
applies field sensitivity. `Manifest.DigestV2` hashes JCS of the unsorted
full manifest; `Manifest.Digest` remains v1 of `Canonical()`.

### `validate`

Ports `validateCollectionV04`, including `entry.localized_mismatch`.

### `taxonomy`

Defines flat or hierarchical taxonomy definitions, string+locale terms,
assignments, allowed resource kinds, and cycle/missing-parent validation.

### `revision`

Defines immutable snapshots of a content Entry and verifies
`(entry_id, locale, version)` identity. Persistence and restore orchestration
remain application concerns.

### `migrate`

Converts v0.3 JSON documents to v0.4 records for a requested locale.

### `conformance`

Ships public fixtures and assertions for downstream adapters. Conformance
tests verify JSON round trips, canonical manifests, lifecycle validation,
locale pick, taxonomy hierarchy, revisions, and the official
`codex-entry/v1` and `codex-manifest/v2` vectors pinned from `@ui8kit/contract`.

## JSON boundary

JSON tags are part of the public compatibility contract but do not imply REST.
Adapters may store or transport the same shapes. Unknown product fields belong
inside metadata and must survive compatible round trips.

Entry, term, and settings canonicalization is RFC 8785 JCS. Manifest v1
canonicalization remains Go `encoding/json` of `Canonical()`; v2 is JCS of
the unsorted full manifest. Neither digest is a cryptographic signature
format. Consumers sign a specified serialized form if needed.

## Product and delivery boundary

Codex injects no resources. Constants such as `content.KindPost` name known
identifiers only. Product kinds, including any `post`, `page`, `menu`,
`setting`, or `media` resource, are explicit manifest data.

Collection route names, REST/GraphQL exposure, authentication, publication
filtering, and media blob delivery belong to GoBackend or another adapter.
They are deliberately absent from `schema.Resource`.

## Contract identity

`Manifest.Digest` hashes canonical JSON with a versioned
`codex-manifest/v1:sha256:` prefix. `canonical.ManifestDigest` /
`Manifest.DigestV2` use JCS and `codex-manifest/v2:sha256:` without sorting
arrays. Validation errors expose machine-readable codes and paths through the
`validation` package. Human messages may improve without changing the contract.

## Migration boundary

This initial module does not change GoBackend. Compatibility documentation maps
each current `internal/domain` type to the new public package. A later
GoBackend change will replace internal types and prove storage, lifecycle,
REST, and conformance parity before deleting duplicates.
