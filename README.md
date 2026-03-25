# Atlas ERP Core

Atlas ERP Core is a Go backend for ERP core operations, designed as a modular monolith with DDD, Clean Architecture, explicit public contracts between modules, and an internal event-driven financial flow.

It exists to demonstrate two things at once:

- a realistic transactional core for customers, invoices, billing, and payments
- a portfolio-grade engineering narrative with measurable behavior, controlled resilience, and documented architectural trade-offs

## Project Links

- Notion hub: [Atlas ERP Core](https://www.notion.so/mrgomides/Atlas-ERP-Core-32ae01f2262680aea1a1dd408f0001d9?source=copy_link)
- Architecture readiness: [docs/architecture/distribution-readiness.md](docs/architecture/distribution-readiness.md)
- Trade-offs: [docs/architecture/trade-offs.md](docs/architecture/trade-offs.md)
- Failure scenarios: [docs/architecture/failure-scenarios.md](docs/architecture/failure-scenarios.md)
- Diagrams: [docs/diagrams/architecture.md](docs/diagrams/architecture.md)
- ADR catalog: [docs/adr/README.md](docs/adr/README.md)
- Commands: [docs/commands.md](docs/commands.md)
- Benchmark baseline: [docs/benchmarks/phase7-baseline.md](docs/benchmarks/phase7-baseline.md)

## Project Status

Current Phase: **Phase 7 - Portfolio Differentiation & Advanced Engineering**

## Why This Project Exists

Atlas ERP Core models a small but non-trivial ERP financial backbone:

- customer registration and lifecycle
- invoice issuance and listing
- billing generation from invoices
- payment processing with idempotency, retry, timeout handling, and auditability

The goal is not feature volume. The goal is to show a defendable architecture, explicit trade-offs, and operational evidence that can be reviewed in code, tests, diagrams, ADRs, traces, metrics, and benchmark artifacts.

## Architecture Summary

- Style: Modular Monolith
- Modeling: DDD
- Internal organization: Clean Architecture + Ports and Adapters
- Module communication: internal synchronous event bus with public event contracts
- Persistence: PostgreSQL with logical ownership by module
- Observability: OpenTelemetry traces and metrics, `slog` JSON logs, Jaeger, Prometheus
- Delivery model: one deployable process, prepared for future extraction without pretending to be distributed already

### Public module contracts

| Module | Public contracts |
| --- | --- |
| `customers` | `ExistenceChecker`, public errors, `public/events` |
| `invoices` | `public/events` |
| `billing` | `PaymentCompatibilityPort`, `BillingSnapshot`, public errors, `public/events` |
| `payments` | `public/events` |

### Active modules

#### Customers

- Aggregate: `Customer`
- Value objects: `Document`, `Email`
- Use cases: `CreateCustomer`, `UpdateCustomer`, `DeactivateCustomer`
- Published event: `CustomerCreated`

#### Invoices

- Aggregate: `Invoice`
- Use cases: `CreateInvoice`, `ListCustomerInvoices`, `ApplyPaymentApproved`
- Published events: `InvoiceCreated`, `InvoicePaid`

#### Billing

- Aggregate: `Billing`
- Use cases: `CreateBillingFromInvoice`, `GetProcessableBillingByInvoiceID`, `MarkBillingApproved`, `MarkBillingFailed`
- Published event: `BillingRequested`
- Important behavior: monotonic `attempt_number`, safe retry reactivation, explicit `Requested`, `Failed`, and `Approved` states

#### Payments

- Aggregate: `Payment`
- Use cases: `ProcessBillingRequest`, `ProcessPayment`
- Published events: `PaymentApproved`, `PaymentFailed`
- Important behavior: idempotency by `(billing_id, attempt_number)`, persisted `idempotency_key`, `failure_category`, timeout handling, and one approved payment per invoice

## Main Flows

### Automatic flow

```text
Create Customer -> Create Invoice -> InvoiceCreated -> BillingRequested -> PaymentApproved -> Invoice Paid
```

### Compatibility and retry path

```text
POST /payments -> reprocess an existing billing after PaymentFailed or technical gateway failure
```

## Internal Event Catalog

All internal events use a stable envelope:

```json
{
  "metadata": {
    "event_id": "uuid",
    "event_name": "BillingRequested",
    "occurred_at": "2026-03-25T10:00:00Z",
    "aggregate_id": "uuid",
    "correlation_id": "req-123"
  },
  "payload": {}
}
```

| Event | Producer | Consumers |
| --- | --- | --- |
| `CustomerCreated` | `customers` | none |
| `InvoiceCreated` | `invoices` | `billing` |
| `BillingRequested` | `billing` | `payments` |
| `PaymentApproved` | `payments` | `billing`, `invoices` |
| `PaymentFailed` | `payments` | `billing` |
| `InvoicePaid` | `invoices` | none |

## Technology Stack

- Go 1.26
- `chi` for HTTP routing
- `log/slog` for structured logging
- `godotenv` for `.env` bootstrap and `.env.<APP_ENV>` overlay
- OpenTelemetry Go SDK for traces and metrics
- `otelhttp` for HTTP instrumentation
- PostgreSQL
- `pgx/v5` for access and query tracing
- `golang-migrate` for migrations
- Docker + Docker Compose
- Jaeger all-in-one for local tracing
- Prometheus for local metrics scraping
- `testcontainers-go` for integration and functional tests
- GitHub Actions for CI baseline

## Runtime And Environment

### Environment variables

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `APP_PORT` | Yes | - | HTTP port exposed by the API |
| `DB_HOST` | Yes | - | PostgreSQL host |
| `DB_PORT` | Yes | - | PostgreSQL port |
| `DB_USER` | Yes | - | PostgreSQL user |
| `DB_PASSWORD` | Yes | - | PostgreSQL password |
| `DB_NAME` | Yes | - | PostgreSQL database |
| `APP_NAME` | No | `atlas-erp-core` | Logical application name |
| `APP_ENV` | No | `local` | Current environment |
| `LOG_LEVEL` | No | `info` | Structured log level |
| `CORRELATION_ID_HEADER` | No | `X-Correlation-ID` | Request correlation header |
| `ATLAS_FAULT_PROFILE` | No | `none` | Controlled failure profile for local evaluation; must stay `none` in `production` |
| `PAYMENT_GATEWAY_TIMEOUT_MS` | No | `2000` | Gateway timeout per payment attempt |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | empty | OTLP HTTP trace export endpoint; empty disables remote export |

### Local stack

The official local stack runs:

- API
- PostgreSQL
- Jaeger
- Prometheus

## How To Run Locally

### 1. Prepare `.env`

```bash
rtk cp .env.example .env
```

### 2. Start the local stack

```bash
rtk docker compose up --build -d
```

### 3. Run migrations

```bash
rtk go run ./cmd/migrate --direction up
```

### 4. Validate health and metrics

```bash
rtk curl http://localhost:8080/health
rtk curl http://localhost:8080/metrics
```

### 5. Run the API outside Compose if needed

```bash
rtk env OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/api
```

### 6. Stop the stack

```bash
rtk docker compose down --remove-orphans
```

More operational commands live in [docs/commands.md](docs/commands.md).

## HTTP Endpoints

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/health` | healthcheck |
| `GET` | `/metrics` | Prometheus metrics |
| `POST` | `/customers` | create customer |
| `PUT` | `/customers/{id}` | update customer |
| `PATCH` | `/customers/{id}/inactive` | deactivate customer |
| `POST` | `/invoices` | create invoice and trigger automatic billing/payment flow |
| `GET` | `/customers/{id}/invoices` | list invoices by customer |
| `POST` | `/payments` | manually retry payment for an existing billing |

## Testing Strategy

This repository follows behavior-first validation:

- unit tests for domain invariants, application orchestration, decorators, and collectors
- integration tests with real PostgreSQL for persistence, outbox, duplicate delivery, retry, timeout, and consumer/outbox failures
- functional tests for HTTP contracts, traceability, and controlled failure profiles

### Run the full suite

```bash
rtk go test ./...
```

### Run by layer

```bash
rtk go test ./internal/...
rtk go test ./test/integration/...
rtk go test ./test/functional/...
```

## Benchmark

Phase 7 adds a reproducible HTTP benchmark suite in [`test/benchmark`](test/benchmark) covering:

- customer creation
- invoice creation
- manual payment retry
- end-to-end flow

### Run the benchmark suite

```bash
rtk proxy go test -run '^$' -bench . -benchmem -benchtime=10x ./test/benchmark
```

### Export JSON and Markdown evidence

```bash
rtk proxy go test -run '^$' -bench . -benchmem -benchtime=10x ./test/benchmark \
  -args \
  -report-json docs/benchmarks/phase7-baseline.json \
  -report-md docs/benchmarks/phase7-baseline.md
```

Reported metrics:

- `avg_ms`
- `p95_ms`
- `ops/s`
- `error_rate_pct`

If Docker or testcontainers are unavailable, the export still writes `docs/benchmarks/phase7-baseline.{json,md}` with `status: no_samples` and a note explaining the missing runtime prerequisite.
The benchmark commands use `rtk proxy go test` so the benchmark runner and custom export flags are forwarded without wrapper interference.

## Controlled Failure Simulation

Phase 7 adds `ATLAS_FAULT_PROFILE` to exercise predictable failure scenarios without changing business contracts.

Supported profiles:

- `none`
- `payment_timeout`
- `payment_flaky_first`
- `duplicate_billing_requested`
- `event_consumer_failure`
- `outbox_append_failure`

### Example: timeout profile

```bash
rtk env ATLAS_FAULT_PROFILE=payment_timeout OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/api
```

### Example: duplicate delivery profile

```bash
rtk env ATLAS_FAULT_PROFILE=duplicate_billing_requested OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/api
```

See [docs/architecture/failure-scenarios.md](docs/architecture/failure-scenarios.md) for the scenario matrix, expected outcomes, and validation steps.

## Observability

### Tracing

- Jaeger UI: `http://localhost:16686`
- Expected service name: `atlas-erp-core`

### Metrics

- Prometheus UI: `http://localhost:9090`
- `GET /metrics` exposes technical metrics for HTTP, events, retries, DB queries, and gateway calls

### Key span names

- `http.request {METHOD} {route}`
- `application.usecase {module}.{UseCase}`
- `event.publish {EventName}`
- `event.consume {consumer_module}.{EventName}`
- `db.query {operation} {table}`
- `integration.gateway payments.Process`

### Key log fields

Always present when applicable:

- `module`
- `request_id`
- `event_id`
- `aggregate_id`
- `correlation_id`
- `trace_id`
- `span_id`
- `event_name`
- `attempt_number`
- `retry_count`
- `failure_category`
- `error_type`

## Engineering Evidence

Phase 7 is considered presentable because the repository now includes:

- benchmark summary artifacts in `docs/benchmarks/`
- controlled fault profiles in runtime composition
- critical scenario coverage across unit, integration, and functional layers
- explicit ADR catalog and phase-governance artifacts
- refined architecture and sequence diagrams
- trade-off and known limitation documents that match the implementation

## Trade-Offs

The project deliberately stays in a modular monolith because:

- one-process operation is still cheaper than distributed coordination
- the current traffic and complexity do not justify broker or microservice overhead
- internal events and public contracts already create a safe path to future extraction
- local debugging, tests, and operational reasoning remain simpler

The longer version is documented in [docs/architecture/trade-offs.md](docs/architecture/trade-offs.md).

## Known Limitations

- the outbox reflects synchronous dispatch lifecycle only; there is no asynchronous dispatcher
- the benchmark suite is local evidence, not a CI gate or a production capacity claim
- failure profiles are intentionally local-only and must remain disabled in production
- PostgreSQL ownership is logical by module, not isolated by schema or database
- there is no external broker, collector, Grafana, or multi-process deployment yet

## Roadmap Snapshot

| Phase | Focus | Outcome |
| --- | --- | --- |
| Phase 0 | Foundation | repo, runtime, CI, healthcheck |
| Phase 1 | Core Domain | first end-to-end business flow |
| Phase 2 | Quality | validation, tests, traceability |
| Phase 3 | Internal Events | event-driven modular flow |
| Phase 4 | Resilience | idempotency, retry, outbox seed |
| Phase 5 | Observability | traces, metrics, Jaeger, Prometheus |
| Phase 6 | Distribution Readiness | public contracts, envelope, extraction criteria |
| Phase 7 | Portfolio Differentiation | benchmark, fault simulation, ADR/diagram showcase |

## Directory Map

```text
.
├── cmd/
│   ├── api/
│   └── migrate/
├── docs/
│   ├── adr/
│   ├── architecture/
│   ├── benchmarks/
│   ├── commands.md
│   └── diagrams/
├── internal/
│   ├── billing/
│   ├── customers/
│   ├── invoices/
│   ├── payments/
│   └── shared/
├── migrations/
├── test/
│   ├── benchmark/
│   ├── functional/
│   ├── integration/
│   └── support/
├── AGENTS.md
├── CHANGELOG.md
└── docker-compose.yml
```
