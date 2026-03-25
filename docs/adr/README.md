# ADR Catalog

This directory tracks the main architectural decisions of Atlas ERP Core.

## Accepted ADRs

| ADR | Focus | Summary |
| --- | --- | --- |
| [0001](0001-phase-0-foundation.md) | Foundation | bootstrap, runtime, CI, and project base |
| [0002](0002-phase-1-core-domain.md) | Core domain | modular monolith, DDD, and first transactional flow |
| [0003](0003-phase-3-event-driven-internal.md) | Internal events | synchronous in-process event communication between modules |
| [0004](0004-phase-4-resilience-and-maturity.md) | Resilience | idempotency by attempt, retry, timeout classification, and outbox preparation |
| [0005](0005-phase-5-observability-and-operations.md) | Observability | traces, metrics, logs, Jaeger, and Prometheus |
| [0006](0006-phase-6-architectural-evolution-and-distribution-readiness.md) | Distribution readiness | public contracts, event envelope, outbox lifecycle, and extraction criteria |

## Reading Order

1. `0002` for the baseline architectural shape
2. `0003` for internal events
3. `0004` for financial resilience behavior
4. `0005` for operational visibility
5. `0006` for distribution readiness and extraction criteria

## ADR Policy

- update the ADR set when a structural decision becomes hard to reverse
- prefer explicit trade-offs over aspirational wording
- keep ADRs aligned with the implemented code, not with future ideas
