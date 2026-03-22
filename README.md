# Atlas ERP Core

Atlas ERP Core e um ERP backend em Go desenhado como modular monolith, com DDD, Clean Architecture e uma trilha de evolucao segura para eventos internos e futura extracao seletiva de modulos.

## Project Links

- Notion: [Atlas ERP Core](https://www.notion.so/mrgomides/Atlas-ERP-Core-32ae01f2262680aea1a1dd408f0001d9?source=copy_link)

## Project Status

Current Phase: **Phase 1 — Core Domain**

## Phase 1 Delivery

Esta fase entrega o primeiro fluxo funcional ponta a ponta do sistema:

`Create Customer -> Create Invoice -> Process Payment -> Invoice Paid`

O escopo implementado nesta fase cobre:

- `customers` com criacao, atualizacao e inativacao logica
- `invoices` com criacao, listagem por cliente e transicao para `Paid`
- `payments` com processamento local/mockado e idempotencia minima por invoice
- persistencia PostgreSQL versionada com migrations por dominio
- testes unitarios, de integracao e funcionais cobrindo regras centrais e fluxo HTTP

## Architecture Summary

- Estilo principal: Modular Monolith
- Modelagem: DDD
- Organizacao interna: Clean Architecture
- Runtime atual: um unico processo HTTP em Go
- Persistencia: PostgreSQL
- Comunicacao entre modulos na Phase 1: contratos sincronos explicitos
- Baseline futura: eventos internos in-process, maior observabilidade e reducao adicional de acoplamento

## Implemented Modules

### Customers

- Aggregate: `Customer`
- Value objects: `Document`, `Email`
- Use cases: `CreateCustomer`, `UpdateCustomer`, `DeactivateCustomer`
- Regras principais: documento unico, email valido, soft delete via status

### Invoices

- Aggregate: `Invoice`
- Use cases: `CreateInvoice`, `ListCustomerInvoices`
- Regras principais: customer ativo, `amount_cents > 0`, `due_date` obrigatoria, status `Pending|Paid|Overdue|Cancelled`

### Payments

- Aggregate: `Payment`
- Use case: `ProcessPayment`
- Regras principais: pagamento por invoice, mock gateway auto-approve em runtime, atualizacao da invoice para `Paid` quando aprovado

### Billing

- Continua como scaffold estrutural para fases futuras

## Public HTTP Endpoints

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/health` | Healthcheck da aplicacao |
| `POST` | `/customers` | Cria cliente |
| `PUT` | `/customers/{id}` | Atualiza nome e email de cliente |
| `PATCH` | `/customers/{id}/inactive` | Inativa cliente logicamente |
| `POST` | `/invoices` | Cria fatura |
| `GET` | `/customers/{id}/invoices` | Lista faturas do cliente |
| `POST` | `/payments` | Processa pagamento mockado |

### JSON Contracts

- IDs sao UUID strings
- `amount_cents` e inteiro
- `due_date` usa `YYYY-MM-DD`
- erros usam `{ "code", "message", "correlation_id" }`

## Directory Structure

```text
.
├── .agents/
│   ├── roles/
│   └── skills/
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
├── .github/workflows/
├── Dockerfile
├── Makefile
└── docker-compose.yml
```

## Environment Variables

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `APP_PORT` | Yes | - | HTTP port exposed by the application. |
| `DB_HOST` | Yes | - | PostgreSQL host for runtime and migrations. |
| `DB_PORT` | Yes | - | PostgreSQL port. |
| `DB_USER` | Yes | - | PostgreSQL username. |
| `DB_PASSWORD` | Yes | - | PostgreSQL password. |
| `DB_NAME` | Yes | - | PostgreSQL database name. |
| `APP_NAME` | No | `atlas-erp-core` | Logical application name. |
| `APP_ENV` | No | `local` | Current environment. |
| `LOG_LEVEL` | No | `info` | Structured log level. |
| `CORRELATION_ID_HEADER` | No | `X-Correlation-ID` | Header propagated across requests and logs. |

Nenhuma nova variavel de ambiente foi introduzida na Phase 1.

## Local Setup

Pre-requisitos:

- Go 1.26+
- Docker Desktop ou daemon Docker em execucao

1. Copie o arquivo de ambiente:

```bash
make setup
```

2. Suba a stack local:

```bash
make up
```

3. Rode as migrations da Phase 1:

```bash
make migrate-up
```

4. Valide o healthcheck:

```bash
curl http://localhost:8080/health
```

5. Execute o fluxo principal:

```bash
curl -X POST http://localhost:8080/customers \
  -H 'Content-Type: application/json' \
  -d '{"name":"Atlas Co","document":"12345678900","email":"team@atlas.io"}'

curl -X POST http://localhost:8080/invoices \
  -H 'Content-Type: application/json' \
  -d '{"customer_id":"<customer-id>","amount_cents":1599,"due_date":"2026-03-25"}'

curl -X POST http://localhost:8080/payments \
  -H 'Content-Type: application/json' \
  -d '{"invoice_id":"<invoice-id>"}'
```

6. Derrube a stack quando terminar:

```bash
make down
```

## Main Commands

Os comandos recorrentes estao detalhados em [docs/commands.md](/Users/rgomids/Projects/atlas-erp-core/docs/commands.md).

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

- Domain: entidades, value objects e invariantes de `customers`, `invoices` e `payments`
- Application: casos de uso com fakes para duplicidade, validacao e orquestracao cross-module
- Integration: PostgreSQL real com `testcontainers-go`, migrations e fluxo de persistencia ponta a ponta
- Functional: `GET /health` e fluxo HTTP completo da Phase 1

## Observability

- logs estruturados em JSON com `log/slog`
- correlation ID na borda HTTP
- erro HTTP padronizado com `code`, `message` e `correlation_id`
- sem emojis em logs

## Documentation

- Arquitetura viva: [AGENTS.md](/Users/rgomids/Projects/atlas-erp-core/AGENTS.md)
- Roles e skills do projeto: [.agents](/Users/rgomids/Projects/atlas-erp-core/.agents)
- Comandos operacionais: [docs/commands.md](/Users/rgomids/Projects/atlas-erp-core/docs/commands.md)
- Diagramas: [docs/diagrams/architecture.md](/Users/rgomids/Projects/atlas-erp-core/docs/diagrams/architecture.md)
- ADR foundation: [docs/adr/0001-phase-0-foundation.md](/Users/rgomids/Projects/atlas-erp-core/docs/adr/0001-phase-0-foundation.md)
- ADR core domain: [docs/adr/0002-phase-1-core-domain.md](/Users/rgomids/Projects/atlas-erp-core/docs/adr/0002-phase-1-core-domain.md)
