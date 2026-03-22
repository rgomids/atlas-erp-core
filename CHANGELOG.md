# Changelog

All notable changes to this project will be documented in this file.

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
