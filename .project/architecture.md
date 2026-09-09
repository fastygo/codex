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

Defines `Manifest` and `Resource`. `Resource.Record` is a FormSet `RecordType`,
so there is one owner for field, form, and relation semantics. Resource adds
assigned taxonomy identifiers and content-aware Entry validation.

Manifest validation checks identifiers, uniqueness, FormSet validation,
relation targets, taxonomy identifiers, and field/relation identity.
Canonicalization sorts resources, relations, capabilities, and taxonomy names.
It preserves FormSet field and option order because renderers may treat that
order as presentation semantics.

`Field.Localized` is the storage discriminator: localized values live in
`Entry.Locales`; all other declared values live in `Entry.Metadata`.
`Resource.PublicProjection` applies FormSet sensitivity to both locations.

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

## Product and delivery boundary

Codex injects no resources. Constants such as `content.KindPost` name known
identifiers only. Product kinds, including any `post`, `page`, `menu`,
`setting`, or `media` resource, are explicit manifest data.

Collection route names, REST/GraphQL exposure, authentication, publication
filtering, and media blob delivery belong to GoBackend or another adapter.
They are deliberately absent from `schema.Resource`.

## Contract identity

`Manifest.Digest` hashes canonical JSON with a versioned prefix. Validation
errors expose machine-readable codes and paths through the `validation`
package. Human messages may improve without changing the contract.

## Migration boundary

This initial module does not change GoBackend. Compatibility documentation maps
each current `internal/domain` type to the new public package. A later
GoBackend change will replace internal types and prove storage, lifecycle,
REST, and conformance parity before deleting duplicates.
