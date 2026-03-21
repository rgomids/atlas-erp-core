# Changelog

All notable changes to this project will be documented in this file.

## [0.1.0] - 2026-03-21

### Added

- Phase 0 foundation with Go module bootstrap, HTTP API entrypoint and migration CLI.
- Structured configuration, JSON logging, correlation ID middleware and PostgreSQL bootstrap validation.
- `GET /health` endpoint with unit, integration and functional coverage.
- Dockerfile, Docker Compose, Makefile and GitHub Actions CI workflow.
- Initial repository documentation set: README, AGENTS alignment, ADR, commands guide and Mermaid C4 diagrams.

### Changed

- Repository structure now reflects the target modular monolith foundation with `internal/shared` and scaffolded bounded contexts.
