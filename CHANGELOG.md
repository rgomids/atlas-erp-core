# Changelog

All notable changes to this project will be documented in this file.

## [0.7.0] - 2026-03-25

### Added

- Module-owned public contracts in `internal/<module>/public` for `customers`, `billing`, and all public event catalogs.
- Standardized event envelope with `metadata` and `payload`, including `event_id`, `event_name`, `occurred_at`, `aggregate_id`, and `correlation_id`.
- Outbox lifecycle updates in `internal/shared/outbox` to mark events as `pending`, `processed`, or `failed` during the current synchronous dispatch.
- Architecture guard test blocking cross-module imports outside `public`, plus unit coverage for event catalogs and metadata constructors.
- Distribution-readiness documentation in `docs/architecture/distribution-readiness.md` and ADR 0006 for future extraction criteria and trade-offs.

### Changed

- Project status moved from Phase 5 to Phase 6 - Architectural Evolution & Distribution Readiness.
- `customers`, `invoices`, `billing`, and `payments` now publish and consume public events instead of importing other modules through `domain/events`.
- `outbox_events` now stores `aggregate_id`, `correlation_id`, `processed_at`, `failed_at`, and `error_message`, while persisting the full event envelope.
- README, AGENTS, commands guide, diagrams, phase status, and governance artifacts now document Phase 6 and use `rtk`-prefixed operational commands.

### Fixed

- Cross-module dependencies from application code no longer bypass public contracts.
- Failed synchronous event dispatches now leave an auditable `failed` status in the outbox when the append has already succeeded.

## [0.6.0] - 2026-03-22

### Added

- OpenTelemetry runtime in `internal/shared/observability` covering traces, metrics, propagators and `/metrics`.
- Tracing for HTTP, application use cases, PostgreSQL queries, internal event publish/consume and payment gateway integration.
- Technical metrics for HTTP, events, retries, PostgreSQL and gateway latency/failures.
- Jaeger all-in-one and Prometheus in the local Docker Compose stack.
- Unit, integration and functional observability coverage for trace propagation, route labeling, metrics exposure, log context and event bus instrumentation.
- ADR documenting the Phase 5 observability decisions.

### Changed

- Project status moved from Phase 4 to Phase 5 - Observability & Operations.
- Structured logs now include `trace_id`, `span_id`, `event_name`, `retry_count` and canonical `error_type` categories when applicable.
- Router now accepts `traceparent` and `tracestate` while preserving `X-Correlation-ID` as the primary operational correlation contract.
- PostgreSQL access now uses `pgx` query tracing with SQL sanitization to emit only operation and table metadata.
- README, AGENTS, commands guide, diagrams and phase status now document the local observability stack and troubleshooting workflow.

### Fixed

- Internal HTTP failures now preserve `infrastructure_error` in request metrics and logs.
- Gateway integration spans no longer risk inconsistent double-finalization on approved or declined paths.

## [0.5.0] - 2026-03-21

### Added

- `attempt_number` control in `billings` and `payments`, plus persisted `idempotency_key` and `failure_category` for payment attempts.
- `outbox_events` table and recorder integration in the synchronous event bus.
- Gateway timeout configuration through `PAYMENT_GATEWAY_TIMEOUT_MS` and optional `.env.<APP_ENV>` overlay loading.
- Unit, integration and functional coverage for duplicate event handling, gateway timeout classification and outbox recording.
- ADR documenting the Phase 4 resilience decisions.

### Changed

- Project status moved from Phase 3 to Phase 4 - Resilience & Maturity.
- `POST /payments` now returns `201` with a failed payment payload when the gateway fails technically, instead of surfacing a transport error.
- Automatic payment processing now reserves a pending attempt before calling the gateway, preventing duplicate execution for the same `(billing_id, attempt_number)`.
- Billing retry now advances `attempt_number` only when retry is legitimate and keeps approved billings blocked.
- Logs now include domain identifiers, `attempt_number`, `idempotency_key` and `failure_category`.

### Fixed

- Reprocessing the same `BillingRequested` no longer issues duplicate payment execution.
- Technical gateway failures no longer break the invoice creation flow; the invoice stays persisted and retryable.

## [0.4.0] - 2026-03-21

### Added

- Internal synchronous event bus in `internal/shared/event` with structured event logging and deterministic handler ordering.
- Domain event catalog for `customers`, `invoices`, `billing` and `payments`.
- Active `billing` module with aggregate, persistence, handlers and compatibility port for manual retry.
- PostgreSQL migrations for `billings` and for `payments.billing_id` plus approved-only uniqueness by invoice.
- Unit, integration and functional coverage for automatic event flow, payment failure and manual retry.
- ADR documenting the Phase 3 event-driven transition.

### Changed

- `POST /invoices` now triggers the main billing and payment flow through internal events.
- `POST /payments` now works as a manual retry path over an existing billing instead of the primary orchestration path.
- Payments now store one attempt per execution and allow retry after `Failed`, while keeping a single `Approved` per invoice.
- README, AGENTS, command guide, architecture diagrams and phase status now document Phase 3 instead of Phase 2.

### Fixed

- Direct coupling from `payments` to `invoices` was removed from the primary flow.
- Billing no longer remains as scaffold only; invoice-to-payment orchestration is now explicit and traceable.

## [0.3.0] - 2026-03-21

### Added

- Canonical HTTP error contract with `error`, `message` and `request_id`.
- Explicit HTTP boundary validation for customers, invoices and payments handlers.
- Request-scoped logging enrichment with `module` and `request_id`.
- Handler-level tests for happy path, invalid input and domain error mapping.
- Extra application, domain, integration and functional tests for validation, uniqueness and idempotency.
- Phase 2 status artifact in `.agents/templates/phase-status.md`.

### Changed

- `X-Correlation-ID` is now exposed consistently as `request_id` in error bodies and structured logs.
- Bootstrap logging now emits `module=bootstrap` and follows the same JSON contract used by request logs.
- README, AGENTS, commands guide and architecture diagrams now document Phase 2 instead of Phase 1 only.
- Functional negative-path coverage now validates canonical error payloads and request traceability.

### Fixed

- HTTP handlers no longer rely exclusively on deeper layers for required field, UUID and date validation.
- Unexpected request failures are now logged at the edge without leaking internal details to clients.

## [0.2.0] - 2026-03-21

### Added

- Phase 1 core domain flow: `Create Customer -> Create Invoice -> Process Payment -> Invoice Paid`.
- `customers`, `invoices` and `payments` modules with domain, application and infrastructure layers.
- PostgreSQL migrations for `customers`, `invoices` and `payments`.
- HTTP endpoints for customer lifecycle, invoice creation/listing and payment processing.
- Domain, application, integration and functional tests for the first end-to-end flow.
- ADR for Phase 1 core domain architecture and updated Mermaid/C4 diagrams.

### Changed

- Router composition now wires business modules in addition to `GET /health`.
- Cross-module collaboration now uses explicit synchronous ports between `customers`, `invoices` and `payments`.
- Functional `health` test no longer depends on PostgreSQL, reducing CI flakiness.
- README, AGENTS, commands guide and project skills now reflect Phase 1 instead of Phase 0 only.

## [0.1.0] - 2026-03-21

### Added

- Phase 0 foundation with Go module bootstrap, HTTP API entrypoint and migration CLI.
- Structured configuration, JSON logging, correlation ID middleware and PostgreSQL bootstrap validation.
- `GET /health` endpoint with unit, integration and functional coverage.
- Dockerfile, Docker Compose, Makefile and GitHub Actions CI workflow.
- Initial repository documentation set: README, AGENTS alignment, ADR, commands guide and Mermaid C4 diagrams.

### Changed

- Repository structure now reflects the target modular monolith foundation with `internal/shared` and scaffolded bounded contexts.
- Root `AGENTS.md` was reduced to a high-level contract and the detailed evolution rules were split into `.agents/roles` and `.agents/skills`.
