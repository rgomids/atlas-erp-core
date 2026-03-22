# Atlas ERP Core

Atlas ERP Core e um ERP backend em Go modelado como modular monolith, com DDD, Clean Architecture e comunicacao interna orientada a eventos para reduzir acoplamento entre modulos.

## Project Links

- Notion: [Atlas ERP Core](https://www.notion.so/mrgomides/Atlas-ERP-Core-32ae01f2262680aea1a1dd408f0001d9?source=copy_link)

## Project Status

Current Phase: **Phase 5 - Observability & Operations**

## Phase 5 Delivery

Esta fase torna o fluxo principal rastreavel e mais operavel sem alterar regras de negocio:

- OpenTelemetry instrumenta HTTP, use cases, PostgreSQL, event bus e gateway
- `GET /metrics` expoe metricas tecnicas com Prometheus format
- logs JSON continuam em `slog`, agora com `trace_id`, `span_id`, `event_name`, `error_type` e ids de dominio
- `X-Correlation-ID` continua como correlacao operacional principal e `traceparent` passa a ser aceito na borda HTTP
- `make up` sobe `app`, `postgres`, `jaeger` e `prometheus`
- retries, falhas de gateway e handlers com erro agora aparecem de forma consistente em traces, metricas e logs

O fluxo principal continua:

`Create Customer -> Create Invoice -> InvoiceCreated -> BillingRequested -> PaymentApproved -> Invoice Paid`

O caminho de compatibilidade continua:

`POST /payments -> reprocessa billing existente apos PaymentFailed ou falha tecnica de gateway`

## Architecture Summary

- Estilo principal: Modular Monolith
- Modelagem: DDD
- Organizacao interna: Clean Architecture + Ports and Adapters
- Comunicacao entre modulos: event bus interno sincronico
- Runtime atual: um unico processo HTTP em Go
- Persistencia: PostgreSQL
- Resiliencia herdada da Phase 4: idempotencia por tentativa, retry controlado, timeout configuravel de gateway e outbox inicial
- Observabilidade da Phase 5: OpenTelemetry para traces e metricas, `slog` para logs estruturados, Jaeger e Prometheus para inspeccao local

## Implemented Modules

### Customers

- Aggregate: `Customer`
- Value objects: `Document`, `Email`
- Use cases: `CreateCustomer`, `UpdateCustomer`, `DeactivateCustomer`
- Eventos publicados: `CustomerCreated`

### Invoices

- Aggregate: `Invoice`
- Use cases: `CreateInvoice`, `ListCustomerInvoices`, `ApplyPaymentApproved`
- Eventos publicados: `InvoiceCreated`, `InvoicePaid`
- Regras principais: customer ativo, `amount_cents > 0`, `due_date` obrigatoria, invoice imutavel apos pagamento

### Billing

- Aggregate: `Billing`
- Use cases: `CreateBillingFromInvoice`, `GetProcessableBillingByInvoiceID`, `MarkBillingApproved`, `MarkBillingFailed`
- Evento publicado: `BillingRequested`
- Regras principais: uma cobranca por invoice, `attempt_number` monotonicamente crescente, reativacao segura apos `Failed`, status `Requested`, `Failed` e `Approved`

### Payments

- Aggregate: `Payment`
- Use cases: `ProcessBillingRequest`, `ProcessPayment`
- Eventos publicados: `PaymentApproved`, `PaymentFailed`
- Regras principais: idempotencia por `(billing_id, attempt_number)`, `idempotency_key` persistida, retry permitido apos `Failed`, no maximo um `Approved` por invoice

## Internal Event Catalog

| Event | Producer | Consumers |
| --- | --- | --- |
| `CustomerCreated` | `customers` | none |
| `InvoiceCreated` | `invoices` | `billing` |
| `BillingRequested` | `billing` | `payments` |
| `PaymentApproved` | `payments` | `billing`, `invoices` |
| `PaymentFailed` | `payments` | `billing` |
| `InvoicePaid` | `invoices` | none |

## Public HTTP Endpoints

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/health` | Healthcheck da aplicacao |
| `GET` | `/metrics` | Endpoint Prometheus com metricas tecnicas |
| `POST` | `/customers` | Cria cliente |
| `PUT` | `/customers/{id}` | Atualiza nome e email do cliente |
| `PATCH` | `/customers/{id}/inactive` | Inativa cliente logicamente |
| `POST` | `/invoices` | Cria invoice e dispara o fluxo automatico de billing e payment |
| `GET` | `/customers/{id}/invoices` | Lista invoices do cliente |
| `POST` | `/payments` | Reprocessa manualmente o pagamento de uma invoice com billing existente |

## JSON Contracts

- IDs sao UUID strings
- `amount_cents` e inteiro
- `due_date` usa `YYYY-MM-DD`
- erros usam:

```json
{
  "error": "invalid_input",
  "message": "document is required",
  "request_id": "req-123"
}
```

- o header `X-Correlation-ID` continua sendo aceito e devolvido
- `traceparent` e `tracestate` passam a ser aceitos para propagacao de trace
- `request_id` continua aparecendo no body de erro e nos logs

## Directory Structure

```text
.
├── .agents/
│   ├── rules/
│   ├── skills/
│   ├── subagents/
│   └── templates/
├── cmd/
│   ├── api/
│   └── migrate/
├── docs/
│   ├── adr/
│   ├── commands.md
│   └── diagrams/
├── internal/
│   ├── billing/
│   ├── customers/
│   ├── invoices/
│   ├── payments/
│   └── shared/
│       ├── config/
│       ├── correlation/
│       ├── event/
│       ├── http/
│       ├── logging/
│       ├── observability/
│       ├── outbox/
│       └── postgres/
├── migrations/
├── test/
│   ├── functional/
│   ├── integration/
│   └── support/
├── CHANGELOG.md
├── Makefile
├── docker-compose.yml
├── prometheus.yml
└── README.md
```

## Technology Stack

- Go 1.26
- `chi` for HTTP routing
- `log/slog` for structured logging
- OpenTelemetry Go SDK for traces and metrics
- `otelhttp` for HTTP instrumentation
- Internal synchronous event bus
- Outbox event recorder stored in PostgreSQL
- PostgreSQL
- `pgx/v5` for database access and query tracing
- `golang-migrate` for migrations
- Docker + Docker Compose for local runtime
- Jaeger all-in-one for local trace inspection
- Prometheus for local metrics scraping
- `testcontainers-go` for integration and functional tests with real PostgreSQL
- GitHub Actions for CI baseline

## Environment Variables

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `APP_PORT` | Yes | - | HTTP port exposed by the application |
| `DB_HOST` | Yes | - | PostgreSQL host |
| `DB_PORT` | Yes | - | PostgreSQL port |
| `DB_USER` | Yes | - | PostgreSQL username |
| `DB_PASSWORD` | Yes | - | PostgreSQL password |
| `DB_NAME` | Yes | - | PostgreSQL database name |
| `APP_NAME` | No | `atlas-erp-core` | Logical application name |
| `APP_ENV` | No | `local` | Current environment |
| `LOG_LEVEL` | No | `info` | Structured log level |
| `CORRELATION_ID_HEADER` | No | `X-Correlation-ID` | Header propagated across requests and logs |
| `PAYMENT_GATEWAY_TIMEOUT_MS` | No | `2000` | Maximum gateway processing time per payment attempt in milliseconds |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | empty | OTLP HTTP endpoint used to export traces; empty disables remote export |

## Local Setup

Prerequisites:

- Go 1.26+
- Docker Desktop or Docker daemon running

1. Copy the environment file:

```bash
make setup
```

2. Start the local stack:

```bash
make up
```

Isso sobe:

- API em `http://localhost:8080`
- Jaeger em `http://localhost:16686`
- Prometheus em `http://localhost:9090`
- PostgreSQL em `localhost:5432`

3. Run migrations:

```bash
make migrate-up
```

4. Validate health and metrics:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/metrics
```

5. Execute the automatic event-driven flow:

```bash
curl -X POST http://localhost:8080/customers \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-ID: demo-req-001' \
  -d '{"name":"Atlas Co","document":"12345678900","email":"team@atlas.io"}'

curl -X POST http://localhost:8080/invoices \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-ID: demo-req-002' \
  -d '{"customer_id":"<customer-id>","amount_cents":1599,"due_date":"2026-03-25"}'

curl http://localhost:8080/customers/<customer-id>/invoices \
  -H 'X-Correlation-ID: demo-req-003'
```

6. Retry a failed payment manually when needed:

```bash
curl -X POST http://localhost:8080/payments \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-ID: demo-req-004' \
  -d '{"invoice_id":"<invoice-id>"}'
```

7. If you run the API outside Compose and still want traces exported, set:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
make run
```

8. Stop the stack:

```bash
make down
```

## Tracing And Metrics

### Span naming

- `http.request {METHOD} {route}`
- `application.usecase {module}.{UseCase}`
- `event.publish {EventName}`
- `event.consume {consumer_module}.{EventName}`
- `db.query {operation} {table}`
- `integration.gateway payments.Process`

### Available metrics

#### HTTP

- `atlas_erp_http_requests_total`
- `atlas_erp_http_request_errors_total`
- `atlas_erp_http_request_duration_seconds`

#### Application

- `atlas_erp_events_published_total`
- `atlas_erp_events_consumed_total`
- `atlas_erp_event_handler_failures_total`
- `atlas_erp_payment_retries_total`

#### Persistence and integration

- `atlas_erp_db_query_duration_seconds`
- `atlas_erp_gateway_request_duration_seconds`
- `atlas_erp_gateway_failures_total`

### Log fields

Sempre:

- `module`
- `request_id`

Quando aplicavel:

- `trace_id`
- `span_id`
- `event_name`
- `event`
- `customer_id`
- `invoice_id`
- `billing_id`
- `payment_id`
- `attempt_number`
- `retry_count`
- `failure_category`
- `error_type`

### Error categories

- `validation_error`
- `domain_error`
- `integration_error`
- `infrastructure_error`

## Troubleshooting

### Follow the main trace

1. Abra o Jaeger em `http://localhost:16686`
2. Selecione o servico `atlas-erp-core`
3. Gere um `POST /invoices`
4. Procure o trace `http.request POST /invoices`
5. Expanda os spans filhos de `application.usecase`, `event.publish`, `event.consume`, `db.query` e `integration.gateway`

### Inspect metrics

1. Abra `http://localhost:9090`
2. Consulte `atlas_erp_http_request_duration_seconds`
3. Consulte `atlas_erp_event_handler_failures_total`
4. Consulte `atlas_erp_gateway_failures_total`
5. Consulte `atlas_erp_payment_retries_total`

### Diagnose payment failures

- `gateway_timeout`: o gateway excedeu `PAYMENT_GATEWAY_TIMEOUT_MS`
- `gateway_error`: erro tecnico no adapter ou na chamada externa
- `gateway_declined`: o gateway respondeu, mas recusou a cobranca
- `attempt_number`: tentativa persistida na cobranca e no pagamento
- `retry_count`: contador operacional derivado de `attempt_number - 1`

## Main Commands

Main commands are documented in [docs/commands.md](docs/commands.md).

```bash
make setup
make up
make down
make run
make build
make fmt
make lint
make test
make test-unit
make test-integration
make test-functional
make migrate-up
make migrate-down
```

## Testing Strategy

### Coverage by layer

- Domain: entities, value objects, idempotency and status transitions
- Application: use cases and event handlers for invoice creation, billing generation, payment processing and manual retry
- Unit observability: telemetry bootstrap, route labeling, SQL sanitization, log enrichment, error taxonomy and event bus spans/metrics
- Integration: PostgreSQL real via `testcontainers-go`, cobrindo fluxo critico, spans, metricas de retry e falha tecnica
- Functional: HTTP contract, `/metrics`, `traceparent`, log context minimo e rastreabilidade ponta a ponta

### How to run tests

```bash
make test-unit
make test-integration
make test-functional
make test
```
