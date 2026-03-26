# Command Reference

This document centralizes the official operational commands for Atlas ERP Core Phase 7.

The published interface uses standard shell commands and Makefile shortcuts. Local wrappers are intentionally not documented here.

## Quick Start

### 1. Create the local environment file

```bash
cp .env.example .env
```

### 2. Start the local stack

Make shortcut:

```bash
make up
```

Underlying command:

```bash
docker compose up --build -d
```

Expected services:

- API at `http://localhost:8080`
- PostgreSQL at `localhost:5432`
- Jaeger at `http://localhost:16686`
- Prometheus at `http://localhost:9090`

### 3. Run migrations

Make shortcut:

```bash
make migrate-up
```

Underlying command:

```bash
go run ./cmd/migrate --direction up
```

### 4. Run the API from source

Make shortcut:

```bash
make run
```

Underlying command:

```bash
go run ./cmd/api
```

If you want traces in the local Jaeger instance while running outside Compose:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 make run
```

### 5. Stop the stack

Make shortcut:

```bash
make down
```

Underlying command:

```bash
docker compose down --remove-orphans
```

## Fast Validation

### Healthcheck

```bash
curl http://localhost:8080/health
```

### Metrics

```bash
curl http://localhost:8080/metrics
```

## Main Flow Examples

### Create customer

```bash
curl -X POST http://localhost:8080/customers \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-ID: demo-phase7-001' \
  -d '{"name":"Atlas Co","document":"12345678900","email":"team@atlas.io"}'
```

### Create invoice and trigger the automatic flow

```bash
curl -X POST http://localhost:8080/invoices \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-ID: demo-phase7-002' \
  -d '{"customer_id":"<customer-id>","amount_cents":1599,"due_date":"2026-03-31"}'
```

### List customer invoices

```bash
curl http://localhost:8080/customers/<customer-id>/invoices \
  -H 'X-Correlation-ID: demo-phase7-003'
```

### Manually retry payment

```bash
curl -X POST http://localhost:8080/payments \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-ID: demo-phase7-004' \
  -d '{"invoice_id":"<invoice-id>"}'
```

## Test Commands

### Full suite

Make shortcut:

```bash
make test
```

Underlying command:

```bash
go test ./...
```

### Unit-focused suite

Make shortcut:

```bash
make test-unit
```

Underlying command:

```bash
go test ./internal/...
```

### Integration suite

Make shortcut:

```bash
make test-integration
```

Underlying command:

```bash
go test ./test/integration/...
```

### Functional suite

Make shortcut:

```bash
make test-functional
```

Underlying command:

```bash
go test ./test/functional/...
```

## Reproducible Benchmark

### Run the Phase 7 HTTP benchmarks

```bash
go test -run '^$' -bench . -benchmem -benchtime=10x ./test/benchmark
```

### Export the baseline as JSON and Markdown

```bash
go test -run '^$' -bench . -benchmem -benchtime=10x ./test/benchmark \
  -args \
  -report-json docs/benchmarks/phase7-baseline.json \
  -report-md docs/benchmarks/phase7-baseline.md
```

Expected fields:

- `avg_ms`
- `p95_ms`
- `ops_per_sec`
- `error_rate_pct`

If Docker or `testcontainers-go` is unavailable, the export still writes the artifacts with `status: no_samples` and a note explaining the missing prerequisite.

## Controlled Failure Profiles

All profiles below are local-only evaluation tools. In `APP_ENV=production`, `ATLAS_FAULT_PROFILE` must remain `none`.

### No injected fault

```bash
ATLAS_FAULT_PROFILE=none OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/api
```

### Gateway timeout

```bash
ATLAS_FAULT_PROFILE=payment_timeout OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/api
```

### First gateway call fails, manual retry can approve

```bash
ATLAS_FAULT_PROFILE=payment_flaky_first OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/api
```

### First `BillingRequested` delivery is duplicated

```bash
ATLAS_FAULT_PROFILE=duplicate_billing_requested OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/api
```

### First `BillingRequested` delivery to `payments` fails

```bash
ATLAS_FAULT_PROFILE=event_consumer_failure OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/api
```

### First outbox append fails before consumers run

```bash
ATLAS_FAULT_PROFILE=outbox_append_failure OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/api
```

## Observability

### Traces

- Local UI: `http://localhost:16686`
- Expected service name: `atlas-erp-core`

### Prometheus

- Local UI: `http://localhost:9090`

### Key span names

- `http.request {METHOD} {route}`
- `application.usecase {module}.{UseCase}`
- `event.publish {EventName}`
- `event.consume {consumer_module}.{EventName}`
- `db.query {operation} {table}`
- `integration.gateway payments.Process`

### Key metrics

- `atlas_erp_http_requests_total`
- `atlas_erp_http_request_errors_total`
- `atlas_erp_http_request_duration_seconds`
- `atlas_erp_events_published_total`
- `atlas_erp_events_consumed_total`
- `atlas_erp_event_handler_failures_total`
- `atlas_erp_payment_retries_total`
- `atlas_erp_db_query_duration_seconds`
- `atlas_erp_gateway_request_duration_seconds`
- `atlas_erp_gateway_failures_total`

## Troubleshooting

### Gateway timeout

- validate `PAYMENT_GATEWAY_TIMEOUT_MS`
- inspect `failure_category=gateway_timeout`
- inspect traces for `integration.gateway payments.Process`

### Event duplication

- use `ATLAS_FAULT_PROFILE=duplicate_billing_requested`
- validate that only one payment is approved per invoice
- inspect `attempt_number` and `idempotency_key`

### Internal consumer failure

- use `ATLAS_FAULT_PROFILE=event_consumer_failure`
- validate `outbox_events.status=failed`
- confirm the absence of `PaymentApproved`

### Outbox append failure

- use `ATLAS_FAULT_PROFILE=outbox_append_failure`
- validate that the upstream aggregate was still persisted
- validate the absence of downstream side effects and missing outbox append
