# Architecture Trade-Offs

## Why stay on a modular monolith

Atlas ERP Core remains a modular monolith because the current design still maximizes learning value per unit of complexity:

- one deployable keeps local operation, tracing, and debugging cheap
- transaction boundaries for invoice, billing, payment, and outbox behavior remain easier to reason about
- the repository already enforces architectural discipline through public contracts and import guards
- future extraction pressure can be evaluated from evidence instead of from fashion

This is a deliberate choice, not a temporary omission.

## Why not extract modules yet

`payments` and `billing` are the strongest extraction candidates, but extraction is not yet justified by the current codebase and runtime:

- no external broker exists yet
- the outbox is still synchronous
- deployment, secrets, and operational ownership are still single-runtime concerns
- the current workload does not require independent scale or release cadence

Extracting now would add network, operational, and observability complexity without proving a concrete system benefit.

## Why not introduce an external broker yet

The internal event bus is still the right choice because:

- it already reduces coupling between modules
- it keeps business flow explicit and testable
- it preserves simple local debugging and deterministic tests
- it allows the project to document event contracts before taking on distributed delivery semantics

The main cost is that synchronous dispatch still couples upstream completion to downstream success.

## Why keep the current observability level

The current observability stack is intentionally narrow:

- OpenTelemetry traces and metrics provide enough technical evidence for the portfolio goal
- Jaeger and Prometheus keep local operation lightweight
- `slog` JSON logs remain the canonical log output without adding a collector or centralized log stack

This is the current cost-benefit balance:

- benefit: clear request, event, DB, and gateway signals with low operational overhead
- cost: no dashboards, no long-term storage, no distributed collector pipeline

## Failure simulation trade-off

Phase 7 adds fault profiles at technical seams instead of feature-level chaos engineering:

- benefit: controlled, reproducible failure scenarios with minimal domain impact
- cost: simulations are intentionally narrow and local-only

The profiles are meant to explain architecture, not to emulate production-scale fault injection.

## Residual risks

- synchronous event dispatch means downstream technical failures can still surface back to the original use case
- outbox append failure can leave the upstream aggregate persisted while preventing downstream propagation
- duplicated delivery is simulated at runtime, but there is still no durable replay or dead-letter workflow
- benchmarks are environment-sensitive and should not be treated as production guarantees

## When to extract a module

Extraction should be considered only when at least one concrete signal is true:

- `payments` needs independent operational scale or stricter SLA isolation
- external gateway integration requires isolated secrets, release cadence, or compliance boundary
- backlog or throughput makes an asynchronous dispatcher unavoidable
- team ownership or deployment cadence starts to diverge by module

If none of these are true, staying modular inside one process remains the higher-value decision.

## Known limitations

- no asynchronous outbox dispatcher
- no external broker
- no multi-runtime deployment
- no schema-level database isolation by module
- no long-term telemetry backend beyond local Jaeger and Prometheus
