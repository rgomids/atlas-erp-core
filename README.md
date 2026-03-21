# Atlas ERP Core

Atlas ERP Core e um ERP backend em Go desenhado como modular monolith, com DDD, Clean Architecture e uma trilha de evolucao segura para event-driven e futura extracao de modulos.

## Project Links

- Notion: [Atlas ERP Core](https://www.notion.so/mrgomides/Atlas-ERP-Core-32ae01f2262680aea1a1dd408f0001d9?source=copy_link)

## Project Status

Current Phase: **Phase 0 — Foundation**

## Foundation Scope

Esta fase entrega apenas a fundacao tecnica:

- bootstrap do modulo Go
- servidor HTTP com `chi`
- endpoint `GET /health`
- configuracao por `.env`
- conexao com PostgreSQL no startup
- logger estruturado com correlation ID
- migrations vazias e operacionais
- Docker, Docker Compose, Makefile e CI
- scaffolds dos modulos `customers`, `billing`, `invoices` e `payments`

## Architecture Summary

- Estilo principal: Modular Monolith
- Modelagem: DDD
- Organizacao interna: Clean Architecture
- Runtime atual: um unico processo HTTP em Go
- Persistencia da foundation: PostgreSQL
- Comunicacao entre modulos: ainda nao implementada; eventos internos seguem como baseline futura

## Directory Structure

```text
.
├── cmd/
│   ├── api/
│   └── migrate/
├── configs/
│   ├── app/
│   └── observability/
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

### Runtime contract for Phase 0

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

### Baseline documented for future phases

As variaveis ligadas a Redis, OpenTelemetry e configuracoes HTTP avancadas continuam parte da baseline arquitetural do projeto, mas ainda nao sao exigidas em runtime na Phase 0.

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

3. Valide a aplicacao:

```bash
curl http://localhost:8080/health
```

Resposta esperada:

```json
{"status":"ok"}
```

4. Derrube a stack quando terminar:

```bash
make down
```

Se `make up` falhar com `Cannot connect to the Docker daemon`, o problema esta no ambiente local e nao na configuracao do projeto. Inicie o Docker Desktop ou o daemon equivalente e execute o comando novamente.

## Main Commands

Os comandos recorrentes estao detalhados em [docs/commands.md](/Users/rgomids/Projects/atlas-erp-core/docs/commands.md).

Comandos principais:

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

- Unitarios: config, logger, correlation middleware e handler de health
- Integracao: bootstrap com PostgreSQL real e migrations via `testcontainers-go`
- Funcional: contrato HTTP de `GET /health`

## Docker

- `Dockerfile` multi-stage para a API
- `docker-compose.yml` com `app` e `postgres`
- PostgreSQL com healthcheck antes do boot da aplicacao

## Documentation

- Arquitetura viva: [AGENTS.md](/Users/rgomids/Projects/atlas-erp-core/AGENTS.md)
- Comandos operacionais: [docs/commands.md](/Users/rgomids/Projects/atlas-erp-core/docs/commands.md)
- Diagramas: [docs/diagrams/architecture.md](/Users/rgomids/Projects/atlas-erp-core/docs/diagrams/architecture.md)
- Decisoes estruturais: [docs/adr/0001-phase-0-foundation.md](/Users/rgomids/Projects/atlas-erp-core/docs/adr/0001-phase-0-foundation.md)
