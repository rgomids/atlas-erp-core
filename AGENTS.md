# AGENTS.md

## Contexto

O Atlas ERP Core esta na **Phase 1 — Core Domain**. A foundation da Phase 0 continua valida, e o primeiro fluxo funcional ponta a ponta ja existe para validar os limites modulares do monolito.

Este arquivo funciona como contrato de alto nivel, guia de governanca e indice dos documentos de apoio em `.agents`, `docs/` e README.

## Principios inegociaveis

- Modular Monolith com limites explicitos entre modulos
- DDD para modelagem de dominio
- Clean Architecture com dependencias apontando para dentro
- Nenhuma regra de negocio em handlers, adapters ou detalhes de infraestrutura
- Comunicacao entre modulos preferencialmente por eventos internos; na Phase 1, contratos sincronos explicitos sao permitidos
- Banco compartilhado fisicamente nao significa acesso livre entre modulos

### Regra de ouro

> Se um modulo depender da implementacao interna de outro modulo, a arquitetura esta quebrada.

## Visao geral da arquitetura

- Deploy unico em Go
- Borda HTTP com `chi`
- Modulos ativos: `customers`, `invoices`, `payments`
- Modulo em scaffold: `billing`
- Persistencia principal: PostgreSQL
- Migrations versionadas em `migrations/`
- Logging estruturado com `log/slog`
- Correlation ID propagado pela borda HTTP
- Fluxo funcional vigente:
  `Create Customer -> Create Invoice -> Process Payment -> Invoice Paid`

## Stack tecnologico completo

- Linguagem principal: Go 1.26
- HTTP router: `github.com/go-chi/chi/v5`
- Logging: `log/slog`
- Config local: `.env` via `github.com/joho/godotenv`
- Banco transacional: PostgreSQL 16
- Driver/acesso: `github.com/jackc/pgx/v5`
- Migrations: `github.com/golang-migrate/migrate/v4`
- Containers locais: Docker + Docker Compose
- Build/test tooling: Makefile
- CI: GitHub Actions
- Testes de integracao/funcionais com banco real: `testcontainers-go`
- IDs: `github.com/google/uuid`

## Todas as variaveis de ambiente

| Variavel | Obrigatoria | Default | Descricao |
| --- | --- | --- | --- |
| `APP_PORT` | Sim | - | Porta HTTP exposta pela aplicacao |
| `DB_HOST` | Sim | - | Host do PostgreSQL |
| `DB_PORT` | Sim | - | Porta do PostgreSQL |
| `DB_USER` | Sim | - | Usuario do PostgreSQL |
| `DB_PASSWORD` | Sim | - | Senha do PostgreSQL |
| `DB_NAME` | Sim | - | Nome do banco PostgreSQL |
| `APP_NAME` | Nao | `atlas-erp-core` | Nome logico da aplicacao |
| `APP_ENV` | Nao | `local` | Ambiente atual |
| `LOG_LEVEL` | Nao | `info` | Nivel de log estruturado |
| `CORRELATION_ID_HEADER` | Nao | `X-Correlation-ID` | Header propagado entre request e logs |

### Baseline futura documentada, mas fora do runtime atual

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

## Estrutura do diretorio de conteudo

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
│   │   ├── application/{dto,ports,usecases}
│   │   ├── domain/{entities,repositories,valueobjects}
│   │   └── infrastructure/{http,mappers,persistence}
│   ├── invoices/
│   │   ├── application/{dto,ports,usecases}
│   │   ├── domain/{entities,repositories}
│   │   └── infrastructure/{http,mappers,persistence}
│   ├── payments/
│   │   ├── application/{dto,ports,usecases}
│   │   ├── domain/{entities,repositories}
│   │   └── infrastructure/{http,integration,mappers,persistence}
│   └── shared/
├── migrations/
├── test/
│   ├── functional/
│   ├── integration/
│   └── support/
├── README.md
├── CHANGELOG.md
└── Makefile
```

## Servicos, jobs e models de cada app

### Customers

- Services implementados:
  `CreateCustomer`, `UpdateCustomer`, `DeactivateCustomer`
- Jobs planejados:
  `RebuildCustomerProjections`, `SyncCustomerReadModel`
- Models atuais:
  `Customer`, `Document`, `Email`, `Customer Status`

### Invoices

- Services implementados:
  `CreateInvoice`, `ListCustomerInvoices`, `InvoicePaymentPort`
- Jobs planejados:
  `ReconcileInvoices`, `RetryInvoiceDispatch`
- Models atuais:
  `Invoice`, `Invoice Status`, `InvoiceSnapshot`

### Payments

- Services implementados:
  `ProcessPayment`, `MockGateway`
- Jobs planejados:
  `RetryPaymentSettlement`, `ExpirePendingPayments`
- Models atuais:
  `Payment`, `Payment Status`, `GatewayRequest`, `GatewayResult`

### Billing

- Services atuais:
  nenhum; permanece scaffold
- Jobs planejados:
  `CloseOverdueCharges`, `RecalculateBillingCycle`
- Models planejados:
  `Charge`, `BillingPolicy`, `BillingCycle`

## Design patterns do projeto

- Modular Monolith
- DDD com aggregates e value objects
- Clean Architecture
- Ports and Adapters
- Repository pattern
- Transaction script apenas na camada de use case quando necessario
- Transaction boundary local com contexto transacional em PostgreSQL
- Mock Adapter para gateway externo na Phase 1

## Regras de dependencia

### Permitido

- `infrastructure` depende de `application` e `domain`
- `application` depende de `domain`
- `domain` nao depende de infraestrutura
- um modulo consumir apenas portas publicas de outro modulo

### Proibido

- regra de negocio em handler HTTP
- import de `infrastructure` de outro modulo
- acesso direto a tabela de outro modulo sem contrato publico
- vazamento de regra de dominio para `internal/shared`
- emojis em logs estruturados

## Estado atual por dominio

- `customers`: implementado com persistencia PostgreSQL e endpoints HTTP
- `invoices`: implementado com criacao, listagem e atualizacao para `Paid` por contrato de pagamento
- `payments`: implementado com gateway mock local auto-approve e idempotencia minima por invoice
- `billing`: mantido como scaffold para fases futuras

## Common hurdles

### Docker daemon indisponivel

- Sintoma: `make up` ou `go test ./...` com `testcontainers-go` falha
- Solucao: iniciar Docker Desktop ou garantir acesso ao daemon/socket

### Migrations nao aplicadas

- Sintoma: endpoints de dominio falham com erro de tabela inexistente
- Solucao: executar `make migrate-up` apos subir o banco

### Correlation ID ausente

- Sintoma: logs sem rastreabilidade por request
- Solucao: manter `correlation.Middleware` no bootstrap HTTP e nao contornar `httpapi.NewRouter`

### Pagamento duplicado

- Sintoma: segunda tentativa de `POST /payments` para a mesma invoice retorna conflito
- Solucao: comportamento esperado da Phase 1; retries e reconciliacao ficam para fases futuras

### Contrato quebrado entre modulos

- Sintoma: um modulo precisa conhecer tabela ou repositorio interno de outro
- Solucao: extrair uma porta publica explicita no modulo provedor e injetar a implementacao no bootstrap

## TDD obrigatorio

Aplicar sempre o ciclo:

1. escrever um teste que falha
2. implementar o minimo para faze-lo passar
3. refatorar preservando comportamento

### Cobertura minima atual

- `domain`: invariantes de `customers`, `invoices` e `payments`
- `application`: criacao/atualizacao/inativacao de customer, criacao/listagem de invoice, processamento de payment
- `integration`: PostgreSQL real com migrations e fluxo completo entre modulos
- `functional`: `GET /health` e fluxo HTTP da Phase 1

## Diagramas e modelagem

- Todo diagrama novo deve ser escrito em Mermaid
- Diagramas arquiteturais devem usar C4Model quando fizer sentido
- Atualizacoes arquiteturais devem refletir `docs/diagrams/architecture.md`

## Politica de documentacao

- Toda alteracao funcional ou arquitetural relevante deve atualizar:
  `README.md`, `CHANGELOG.md`, `docs/commands.md`, `docs/diagrams/architecture.md` e este `AGENTS.md`
- Toda decisao estrutural relevante deve virar ADR em `docs/adr/`
- O `README.md` deve sempre refletir a fase atual do projeto

## Instrucoes para criar e manter o CHANGELOG.md

- Registrar toda evolucao relevante no mesmo change set do codigo
- Agrupar por versao ou marco
- Separar `Added`, `Changed`, `Fixed` e `Removed`
- Descrever comportamento entregue, nao so lista de arquivos alterados
- Nao deixar evolucao arquitetural sem registro

## Checklist pos-implementacao

- limites modulares preservados
- dominio sem dependencias de infraestrutura
- contratos publicos entre modulos explicitados
- migrations atualizadas quando necessario
- logs estruturados e correlation ID preservados
- testes unitarios, de integracao e funcionais atualizados
- `README.md` atualizado
- `CHANGELOG.md` atualizado
- `docs/commands.md` atualizado
- `docs/diagrams/architecture.md` atualizado
- `AGENTS.md` atualizado
- role/skill afetado revisado em `.agents`

## Como usar `.agents`

### roles

- [architecture-steward.md](/Users/rgomids/Projects/atlas-erp-core/.agents/roles/architecture-steward.md)
- [foundation-engineer.md](/Users/rgomids/Projects/atlas-erp-core/.agents/roles/foundation-engineer.md)
- [domain-evolution-engineer.md](/Users/rgomids/Projects/atlas-erp-core/.agents/roles/domain-evolution-engineer.md)
- [quality-and-release-guardian.md](/Users/rgomids/Projects/atlas-erp-core/.agents/roles/quality-and-release-guardian.md)

### skills

- [foundation-runtime.md](/Users/rgomids/Projects/atlas-erp-core/.agents/skills/foundation-runtime.md)
- [modular-monolith-ddd.md](/Users/rgomids/Projects/atlas-erp-core/.agents/skills/modular-monolith-ddd.md)
- [testing-and-tdd.md](/Users/rgomids/Projects/atlas-erp-core/.agents/skills/testing-and-tdd.md)
- [observability-and-operations.md](/Users/rgomids/Projects/atlas-erp-core/.agents/skills/observability-and-operations.md)
- [documentation-and-governance.md](/Users/rgomids/Projects/atlas-erp-core/.agents/skills/documentation-and-governance.md)
