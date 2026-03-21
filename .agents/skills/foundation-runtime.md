# Skill: Foundation Runtime

## Objetivo

Descrever a base técnica vigente da Phase 0 e as convenções que qualquer agente deve preservar ao evoluir runtime, bootstrap, ambiente local e estrutura do repositório.

## Estado vigente da foundation

- Linguagem principal: Go
- HTTP: `chi`
- Logging: `log/slog` com saída estruturada em JSON
- Config local: `.env` via `godotenv`
- Banco transacional: PostgreSQL
- Driver/acesso: `pgx/v5`
- Migrations: `golang-migrate`
- CI: GitHub Actions
- Containers: Docker + Docker Compose
- Testes de integração: `testcontainers-go`

## Estrutura oficial do repositório

```text
.
├── .agents/
│   ├── roles/
│   └── skills/
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
│   ├── shared/
│   ├── customers/
│   ├── billing/
│   ├── invoices/
│   └── payments/
├── migrations/
└── test/
    ├── integration/
    ├── functional/
    └── support/
```

## Contrato de configuração da Phase 0

### Obrigatórias

- `APP_PORT`
- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`

### Opcionais com default

- `APP_NAME=atlas-erp-core`
- `APP_ENV=local`
- `LOG_LEVEL=info`
- `CORRELATION_ID_HEADER=X-Correlation-ID`

### Baseline futura documentada

- `DATABASE_URL`
- `DATABASE_MAX_OPEN_CONNS`
- `DATABASE_MAX_IDLE_CONNS`
- `DATABASE_CONN_MAX_LIFETIME`
- `REDIS_URL`
- `HTTP_READ_TIMEOUT`
- `HTTP_WRITE_TIMEOUT`
- `HTTP_IDLE_TIMEOUT`
- `OTEL_SERVICE_NAME`
- `OTEL_EXPORTER_OTLP_ENDPOINT`
- `OTEL_EXPORTER_OTLP_HEADERS`
- `OTEL_TRACES_SAMPLER`

## Comandos oficiais

- `make setup`
- `make up`
- `make down`
- `make run`
- `make build`
- `make fmt`
- `make lint`
- `make test`
- `make test-unit`
- `make test-integration`
- `make test-functional`
- `make migrate-up`
- `make migrate-down`

## Regras para evolução da foundation

- Não introduzir lógica de negócio em `internal/shared`.
- `internal/shared` deve conter apenas utilidades transversais estáveis.
- Alterações em bootstrap, ambiente, comandos, stack ou estrutura exigem atualização de README, CHANGELOG e `docs/commands.md`.
- Redis e OpenTelemetry continuam fora do runtime da Phase 0 até decisão explícita.
