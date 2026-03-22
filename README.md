# Atlas ERP Core

Atlas ERP Core e um ERP backend em Go modelado como modular monolith, com DDD, Clean Architecture e comunicacao interna orientada a eventos para reduzir acoplamento entre modulos.

## Project Links

- Notion: [Atlas ERP Core](https://www.notion.so/mrgomides/Atlas-ERP-Core-32ae01f2262680aea1a1dd408f0001d9?source=copy_link)

## Project Status

Current Phase: **Phase 3 - Event-Driven Internal**

## Phase 3 Delivery

Esta fase substitui o acoplamento sincrono entre modulos por um fluxo interno baseado em eventos in-process:

- `SyncBus` sincronico em `internal/shared/event`
- `POST /invoices` agora dispara o fluxo principal `InvoiceCreated -> BillingRequested -> PaymentApproved/PaymentFailed`
- `POST /payments` permanece como caminho manual de compatibilidade e retry funcional apos falha
- `billing` deixa de ser scaffold e passa a participar do fluxo principal
- logs estruturados passam a registrar `event`, `emitter_module`, `consumer_module` e `request_id`

O fluxo principal agora e:

`Create Customer -> Create Invoice -> InvoiceCreated -> BillingRequested -> PaymentApproved -> Invoice Paid`

## Architecture Summary

- Estilo principal: Modular Monolith
- Modelagem: DDD
- Organizacao interna: Clean Architecture + Ports and Adapters
- Comunicacao entre modulos: Event bus interno sincronico
- Runtime atual: um unico processo HTTP em Go
- Persistencia: PostgreSQL
- Observabilidade da Phase 3: logging JSON com `module`, `event`, `emitter_module`, `consumer_module` e `request_id`

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
- Regras principais: uma cobranca por invoice, reativacao de cobranca falha para retry manual, status `Requested`, `Failed` e `Approved`

### Payments

- Aggregate: `Payment`
- Use cases: `ProcessBillingRequest`, `ProcessPayment`
- Eventos publicados: `PaymentApproved`, `PaymentFailed`
- Regras principais: uma tentativa por execucao, retry permitido apos `Failed`, no maximo um `Approved` por invoice

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

- o header `X-Correlation-ID` continua sendo aceito e devolvido; o mesmo valor aparece como `request_id` no body de erro e nos logs

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
│   │   ├── application/
│   │   │   ├── handlers/
│   │   │   ├── ports/
│   │   │   └── usecases/
│   │   ├── domain/
│   │   │   ├── entities/
│   │   │   ├── events/
│   │   │   └── repositories/
│   │   └── infrastructure/
│   ├── customers/
│   ├── invoices/
│   ├── payments/
│   └── shared/
│       ├── event/
│       ├── http/
│       ├── logging/
│       └── postgres/
├── migrations/
├── test/
│   ├── functional/
│   ├── integration/
│   └── support/
├── CHANGELOG.md
├── Makefile
└── README.md
```

## Technology Stack

- Go 1.26
- `chi` for HTTP routing
- `log/slog` for structured logging
- `godotenv` for `.env` loading
- Internal synchronous event bus
- PostgreSQL
- `pgx/v5` for database access
- `golang-migrate` for migrations
- Docker + Docker Compose for local runtime
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

3. Run migrations:

```bash
make migrate-up
```

4. Validate the API:

```bash
curl http://localhost:8080/health
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

7. Stop the stack:

```bash
make down
```

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

### Coverage by Layer

- Domain: entities, value objects, idempotency and status transitions
- Application: use cases and event handlers for invoice creation, billing generation, payment processing and manual retry
- Integration: PostgreSQL real via `testcontainers-go`, cobrindo invoice -> billing -> payment -> invoice paid
- Functional: HTTP contract, canonical error payload, automatic event-driven flow and manual retry path

### How to Run Tests

```bash
make test-unit
make test-integration
make test-functional
make test
```

## Observability

- logs are emitted as JSON through `slog`
- every request carries a `request_id`
- event logs include at least `event`, `module`, `emitter_module`, `consumer_module` and `request_id`
- `request_id` is sourced from `X-Correlation-ID` when present, otherwise generated at the edge
- internal failures return generic `internal_error` without leaking technical details
