# ADR 0001: Keep Events Slice-Local

## Status

Accepted.

## Context

The upstream [Event Modeling
Specification](https://github.com/dilgerma/event-modeling-spec) JSON Schema
stores `events` in each `Slice.events` array. Earlier exploration considered
HCL event groups, external-event blocks,
qualified event IDs, and `emits` relationships. Those constructs would alter
the source model topology and require new graph semantics not present in the
schema.

## Decision

The HCL v1 representation keeps events as direct `event "<id>"` blocks inside
their owning `slice "<id>"` block. A dependency remains a
`dependency "<id>"` block whose label and attributes preserve the raw source
values. `linked_id` is likewise an opaque string.

No event-group, external-event, qualified-event-ID, or `emits` construct is
part of HCL v1. The validator does not resolve dependencies, require globally
unique IDs, or verify a dependency target's kind.

## Consequences

- The HCL tree directly mirrors the upstream slice-local collections.
- The supplied pet-management fixture can retain `evt-pet-001`, `evt-pet-002`,
  and unresolved `evt-001` links without reinterpretation.
- Cross-slice or multi-file relationship semantics remain a future design
  decision rather than an implicit v1 behavior.
