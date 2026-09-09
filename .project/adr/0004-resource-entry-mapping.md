# ADR-0004: Resource-aware Entry mapping

- Status: Accepted
- Date: 2026-09-09

## Context

An Entry aggregate and a FormSet record declaration are insufficient if each
adapter decides independently where field values live. That would let
GoBackend and offline applications drift while claiming the same contract.

## Decision

`Field.Localized` is the sole placement discriminator. Localized fields live
in `Entry.Locales[locale].Data`; all other declared fields live in
`Entry.Metadata`. Relation fields are non-localized.

The built-in localized IDs `slug`, `title`, `content`, and `excerpt` mirror
their Entry chrome maps. A duplicate value in locale data must match.

`Resource.ValidateEntry` applies FormSet binding, Codex semantic rules, and
relation cardinality. `Resource.PublicProjection` removes private metadata and
all schema-sensitive fields from either storage location.

## Consequences

- Adapters have one testable mapping instead of a convention.
- Resource validation rejects declared fields stored in the wrong location.
- Unknown product data remains round-trippable in metadata or locale
  documents, but it is not silently treated as a declared field.

## Rejected alternatives

- Let every application choose metadata versus locale documents.
- Store all fields in one untyped map.
- Put localization policy in REST or UI code.
