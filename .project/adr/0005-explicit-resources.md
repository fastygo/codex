# ADR-0005: Explicit resources and delivery separation

- Status: Accepted
- Date: 2026-09-09

## Context

Automatic post/page/menu/setting registration and route collection names are
GoBackend behavior. Keeping them in a protocol-neutral module would make
Codex a partial delivery framework and create hidden manifest state.

## Decision

Codex injects no resources. Every application supplies every resource it uses.
Known `content.Kind` constants are identifiers only.

`schema.Resource` does not contain route collection names or REST/GraphQL
visibility. Delivery adapters own those projections and may apply their own
defaults before constructing a Codex manifest.

## Consequences

- A manifest is complete and deterministic without bootstrap side effects.
- Offline applications do not inherit HTTP concepts.
- GoBackend retains its route and default-resource behavior in its own
  adapter.

## Rejected alternatives

- Keep `CoreResources` and `WithCoreResources` in the public kernel.
- Put REST base paths and protocol flags on every resource.
- Make offline applications understand GoBackend route registration.
