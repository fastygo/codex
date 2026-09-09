# Codex project

Codex is the public protocol-neutral contract extracted from GoBackend's
internal domain model. It lets GoBackend, desktop applications, and future
delivery adapters share one content model without importing one another.

## Documents

- [Intent](intent.md)
- [Architecture](architecture.md)
- [Compatibility](compatibility.md)
- [ADR-0001: Public protocol-neutral kernel](adr/0001-public-kernel.md)
- [ADR-0002: FormSet owns fields and relations](adr/0002-formset-schema-ownership.md)
- [ADR-0003: Stable JSON and canonicalization](adr/0003-json-canonicalization.md)

## Release gate

The module is complete only when its public packages, conformance fixtures,
documentation, tests, formatting, vet, and clean-checkout consumer path pass
without local module replacements.
