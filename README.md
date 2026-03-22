# Atlas ERP Core

Atlas ERP Core é um ERP backend em Go modelado como modular monolith, com DDD, Clean Architecture e observabilidade orientada a rastreabilidade de request.

## Project Links

- Notion: [Atlas ERP Core](https://www.notion.so/mrgomides/Atlas-ERP-Core-32ae01f2262680aea1a1dd408f0001d9?source=copy_link)

## Project Status

Current Phase: **Phase 2 - Quality & Engineering**

## Phase 2 Delivery

Esta fase consolida a base funcional da Phase 1 e endurece a plataforma sem adicionar features novas:

- validação explícita na borda HTTP
- contrato único de erro com `request_id`
- logs estruturados com `module` e `request_id`
- cobertura reforçada em domain, application, integration e functional
- documentação operacional e de engenharia sincronizada com a implementação real

O fluxo principal continua:

`Create Customer -> Create Invoice -> Process Payment -> Invoice Paid`

## Architecture Summary

- Estilo principal: Modular Monolith
- Modelagem: DDD
- Organização interna: Clean Architecture + Ports and Adapters
- Runtime atual: um único processo HTTP em Go
- Persistência: PostgreSQL
- Comunicação entre módulos: contratos síncronos explícitos e pequenos
- Observabilidade da Phase 2: logging JSON, `request_id` por request e erro HTTP padronizado

## Implemented Modules

### Customers

- Aggregate: `Customer`
- Value objects: `Document`, `Email`
- Use cases: `CreateCustomer`, `UpdateCustomer`, `DeactivateCustomer`
- Regras principais: documento único, email válido, inativação lógica

### Invoices

- Aggregate: `Invoice`
- Use cases: `CreateInvoice`, `ListCustomerInvoices`
- Regras principais: customer ativo, `amount_cents > 0`, `due_date` obrigatória, invoice imutável após pagamento

### Payments

- Aggregate: `Payment`
- Use case: `ProcessPayment`
- Regras principais: pagamento por invoice, idempotência por `invoice_id`, atualização da invoice para `Paid` quando aprovado

### Billing

- Permanece como scaffold estrutural para fases futuras

## Public HTTP Endpoints

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/health` | Healthcheck da aplicação |
| `POST` | `/customers` | Cria cliente |
| `PUT` | `/customers/{id}` | Atualiza nome e email do cliente |
| `PATCH` | `/customers/{id}/inactive` | Inativa cliente logicamente |
| `POST` | `/invoices` | Cria invoice |
| `GET` | `/customers/{id}/invoices` | Lista invoices do cliente |
| `POST` | `/payments` | Processa pagamento mockado |

## JSON Contracts

- IDs são UUID strings
- `amount_cents` é inteiro
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
│   ├── customers/
│   │   ├── application/
│   │   ├── domain/
│   │   └── infrastructure/
│   ├── invoices/
│   │   ├── application/
│   │   ├── domain/
│   │   └── infrastructure/
│   ├── payments/
│   │   ├── application/
│   │   ├── domain/
│   │   └── infrastructure/
│   └── shared/
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

5. Execute the main flow:

```bash
curl -X POST http://localhost:8080/customers \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-ID: demo-req-001' \
  -d '{"name":"Atlas Co","document":"12345678900","email":"team@atlas.io"}'

curl -X POST http://localhost:8080/invoices \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-ID: demo-req-002' \
  -d '{"customer_id":"<customer-id>","amount_cents":1599,"due_date":"2026-03-25"}'

curl -X POST http://localhost:8080/payments \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-ID: demo-req-003' \
  -d '{"invoice_id":"<invoice-id>"}'
```

6. Stop the stack:

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

- Domain: entities, value objects and invariants for `customers`, `invoices` and `payments`
- Application: use cases for happy path, validation, conflicts and cross-module orchestration
- Integration: PostgreSQL real via `testcontainers-go`, including persistence, uniqueness and payment idempotency
- Functional: HTTP contract, canonical error payload and end-to-end Phase 1 flow

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
- `request_id` is sourced from `X-Correlation-ID` when present, otherwise generated at the edge
- request logs include at least `time`, `level`, `msg`, `module` and `request_id`
- internal failures return generic `internal_error` without leaking technical details
- logs remain machine-friendly and do not use emoji

## Documentation

- Operational contract: [AGENTS.md](AGENTS.md)
- Commands: [docs/commands.md](docs/commands.md)
- Architecture diagrams: [docs/diagrams/architecture.md](docs/diagrams/architecture.md)
- ADR foundation: [docs/adr/0001-phase-0-foundation.md](docs/adr/0001-phase-0-foundation.md)
- ADR core domain: [docs/adr/0002-phase-1-core-domain.md](docs/adr/0002-phase-1-core-domain.md)
- Change history: [CHANGELOG.md](CHANGELOG.md)
