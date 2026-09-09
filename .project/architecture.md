# Architecture

## Dependency direction

```text
github.com/fastygo/formset
          ↑
github.com/fastygo/codex
          ↑
GoBackend and product applications
```

FormSet owns generic record fields, relations, validation hints, scopes, and
capabilities. Codex adds content identity, lifecycle, resources, taxonomies,
revisions, and localization. It has no dependency on GoBackend or a delivery
protocol.

## Packages

### `content`

Defines `Entry`, `Kind`, `Status`, `Visibility`, `LocalizedText`,
`LocaleDocument`, metadata values, taxonomy references, locale resolution,
public projection, slug normalization, and protocol-neutral validation.

### `schema`

Defines `Manifest` and `Resource`. A resource embeds a FormSet `RecordType` so
there is one owner for field, form, and relation semantics. Resource adds a
collection name, assigned taxonomies, and public content semantics.

Manifest validation checks identifiers, uniqueness, FormSet validation,
relation targets, taxonomy identifiers, and reserved core resources.
Canonicalization sorts resources, relations, capabilities, and taxonomy names.
It preserves FormSet field and option order because renderers may treat that
order as presentation semantics.

### `taxonomy`

Defines flat or hierarchical taxonomy definitions, localized terms,
assignments, allowed resource kinds, and cycle/missing-parent validation.

### `revision`

Defines immutable snapshots of a content Entry and verifies entry/version
identity. Persistence and restore orchestration remain application concerns.

### `conformance`

Ships public fixtures and assertions for downstream adapters. Conformance
tests verify JSON round trips, canonical manifests, lifecycle validation,
locale fallback, taxonomy hierarchy, revisions, and unknown product kinds.

## JSON boundary

JSON tags are part of the public compatibility contract but do not imply REST.
Adapters may store or transport the same shapes. Unknown product fields belong
inside locale documents or metadata and must survive compatible round trips.

Canonicalization provides deterministic semantic ordering; it is not a
cryptographic signature format. Consumers sign a specified serialized form if
needed.

## Core and product resources

Codex reserves four core kinds:

- `post`
- `page`
- `menu`
- `setting`

Product kinds such as `message` and `conversation` are supplied through a
manifest. `WithCoreResources` adds missing core resources without replacing a
product declaration, preserving current GoBackend behavior.

## Migration boundary

This initial module does not change GoBackend. Compatibility documentation maps
each current `internal/domain` type to the new public package. A later
GoBackend change will replace internal types and prove storage, lifecycle,
REST, and conformance parity before deleting duplicates.
