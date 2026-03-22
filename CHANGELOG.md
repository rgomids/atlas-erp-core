# Changelog

All notable changes to this project will be documented in this file.

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
