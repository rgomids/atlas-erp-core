# AGENTS.md

## Missao do projeto

`atlas-erp-core` evolui um **modular monolith em Go** para o dominio central de ERP, preservando fronteiras explicitas entre modulos, regras de negocio encapsuladas no dominio e uma base operacional simples, reproduzivel e auditavel.

## Fase atual

**Phase 7 - Portfolio Differentiation & Advanced Engineering**

Estado de referencia desta fase:

- foundation da Phase 0 continua valida
- fluxo funcional ponta a ponta da Phase 1 continua ativo
- endurecimento tecnico da Phase 2 continua valido
- comunicacao entre modulos continua priorizando **eventos internos in-process**
- contratos entre modulos agora vivem em `internal/<module>/public`
- eventos internos agora usam envelope padronizado com `event_id`, `event_name`, `occurred_at`, `aggregate_id`, `correlation_id` e `payload`
- observabilidade continua incluindo traces OpenTelemetry, metricas Prometheus e logs enriquecidos com `trace_id` e `span_id`
- pagamentos e handlers financeiros operam com **idempotencia por tentativa**
- retry manual e controlado usa `attempt_number`
- timeout de gateway e falhas tecnicas passam a gerar tentativa auditavel em `Failed`
- `outbox_events` agora reflete `pending`, `processed` e `failed` no dispatch sincronico atual
- o runtime local agora pode ativar falhas controladas via `ATLAS_FAULT_PROFILE`
- o repositorio agora possui benchmark reproduzivel em `test/benchmark`
- README, ADRs, trade-offs e diagramas passam a servir tambem como evidencia de portfolio
- stack local de desenvolvimento sobe `app`, `postgres`, `jaeger` e `prometheus`
- modulos ativos: `customers`, `invoices`, `billing`, `payments`

O `README.md` deve sempre refletir a fase atual. Quando a fase mudar, atualizar tambem `.agents/templates/phase-status.md` ou o artefato de status adotado pelo repositorio.

## Papel deste arquivo

Este arquivo e o **roteador principal** do engine de agentes.
Ele define como trabalhar no repositorio, quais artefatos carregar por demanda e quais contratos vivos precisam permanecer sincronizados com a implementacao.

## Visao geral da arquitetura

- estilo principal: **Modular Monolith**
- modelagem: **DDD**
- organizacao interna: **Clean Architecture**
- integracao entre camadas: **Ports and Adapters**
- comunicacao entre modulos: **Internal Event-Driven Communication**
- persistencia: **PostgreSQL** com ownership logico por modulo
- observabilidade: OpenTelemetry para tracing e metricas, `slog` para logs JSON, `request_id`, `trace_id`, `event_name`, ids de dominio e erro HTTP canonico
- contrato de fluxo atual:

```text
Create Customer -> Create Invoice -> InvoiceCreated -> BillingRequested -> PaymentApproved -> Invoice Paid
```

- caminho de compatibilidade:

```text
POST /payments -> reprocessa billing existente apos PaymentFailed ou falha tecnica de gateway
```

## Stack tecnologico completo

- Go 1.26
- `chi` para roteamento HTTP
- `log/slog` para logging estruturado
- `godotenv` para bootstrap de `.env`
- OpenTelemetry Go SDK para traces e metricas
- `otelhttp` para instrumentacao HTTP
- Event bus interno sincronico em `internal/shared/event`
- Recorder tecnico de outbox em `internal/shared/outbox`
- PostgreSQL
- `pgx/v5` para acesso ao banco
- `golang-migrate` para migrations
- Docker + Docker Compose
- Jaeger all-in-one para traces locais
- Prometheus para metricas locais
- `testcontainers-go` para testes com PostgreSQL real
- GitHub Actions para CI

## Ordem de leitura

Leia apenas o necessario para a tarefa atual:

1. este `AGENTS.md`
2. `.agents/rules/00-global.md`
3. `.agents/rules/70-phase-governance.md`
4. a rule especializada da tarefa:
   - arquitetura: `.agents/rules/10-architecture.md`
   - implementacao: `.agents/rules/20-coding.md`
   - testes: `.agents/rules/30-testing.md`
   - documentacao: `.agents/rules/40-documentation.md`
   - seguranca operacional: `.agents/rules/50-security.md`
   - entrega: `.agents/rules/60-delivery.md`
5. a skill do fluxo, quando houver ganho direto:
   - runtime/config/bootstrap: `.agents/skills/foundation-runtime.md`
   - dominio/arquitetura: `.agents/skills/modular-monolith-ddd.md`
   - testes/TDD: `.agents/skills/testing-and-tdd.md`
   - observabilidade/operacao: `.agents/skills/observability-and-operations.md`
   - documentacao/governanca: `.agents/skills/documentation-and-governance.md`
6. subagentes apenas se houver especializacao real e particionamento seguro
7. templates apenas no momento de abrir tarefa, revisar ou registrar handoff

## Politica de contexto minimo

Carregue o minimo suficiente para executar com seguranca:

- para mudar um modulo, leia a rule de arquitetura e a skill do dominio correspondente
- para mudar runtime, bootstrap, config, compose, CI ou observabilidade, leia `foundation-runtime.md` e `observability-and-operations.md`
- para mudar comportamento de dominio, leia `modular-monolith-ddd.md` e `testing-and-tdd.md`
- para encerrar uma entrega, valide `documentation-and-governance.md` e `review-checklist.md`
- nao abra todos os arquivos de `.agents` por padrao

## Variaveis de ambiente

| Variavel | Obrigatoria | Default | Uso |
| --- | --- | --- | --- |
| `APP_PORT` | Sim | - | Porta HTTP da API |
| `DB_HOST` | Sim | - | Host do PostgreSQL |
| `DB_PORT` | Sim | - | Porta do PostgreSQL |
| `DB_USER` | Sim | - | Usuario do PostgreSQL |
| `DB_PASSWORD` | Sim | - | Senha do PostgreSQL |
| `DB_NAME` | Sim | - | Banco do PostgreSQL |
| `APP_NAME` | Nao | `atlas-erp-core` | Nome logico da aplicacao |
| `APP_ENV` | Nao | `local` | Ambiente atual |
| `LOG_LEVEL` | Nao | `info` | Nivel de log estruturado |
| `CORRELATION_ID_HEADER` | Nao | `X-Correlation-ID` | Header usado para propagar `request_id` |
| `ATLAS_FAULT_PROFILE` | Nao | `none` | Perfil de falha controlada para avaliacao local; em `production` deve permanecer `none` |
| `PAYMENT_GATEWAY_TIMEOUT_MS` | Nao | `2000` | Timeout maximo do gateway de pagamento por tentativa |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Nao | vazio | Endpoint OTLP HTTP para exportacao de traces; vazio desabilita export remoto |

## Estrutura do diretorio de conteudo

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
│   ├── architecture/
│   ├── adr/
│   ├── benchmarks/
│   ├── commands.md
│   └── diagrams/
├── internal/
│   ├── shared/
│   │   ├── config/
│   │   ├── correlation/
│   │   ├── event/
│   │   ├── http/
│   │   ├── logging/
│   │   ├── observability/
│   │   ├── outbox/
│   │   ├── postgres/
│   │   └── runtimefaults/
│   ├── customers/
│   ├── invoices/
│   ├── billing/
│   └── payments/
├── migrations/
├── test/
│   ├── functional/
│   ├── integration/
│   └── support/
├── CHANGELOG.md
├── Makefile
└── README.md
```

## Servicos, jobs e models por app

### Shared

- servicos/capacidades: `config`, `http`, `correlation`, `logging`, `observability`, `postgres`, `event`, `outbox`, `runtimefaults`
- jobs: nenhum
- models: apenas primitives tecnicas e utilitarios transversais estaveis

### Customers

- services/use cases: `CreateCustomer`, `UpdateCustomer`, `DeactivateCustomer`
- jobs: nenhum
- models: `Customer`, `Document`, `Email`

### Invoices

- services/use cases: `CreateInvoice`, `ListCustomerInvoices`, `ApplyPaymentApproved`
- jobs: nenhum
- models: `Invoice`

### Billing

- services/use cases: `CreateBillingFromInvoice`, `GetProcessableBillingByInvoiceID`, `MarkBillingApproved`, `MarkBillingFailed`
- jobs: nenhum
- models: `Billing` com `attempt_number` e `customer_id`

### Payments

- services/use cases: `ProcessBillingRequest`, `ProcessPayment`
- jobs: nenhum
- models: `Payment` com `attempt_number`, `idempotency_key` e `failure_category`

## Como usar `.agents`

```text
.agents/
├── rules/       # regras permanentes por responsabilidade
├── templates/   # handoff, fase, task brief e review
├── skills/      # workflows repetitivos e encapsulaveis
└── subagents/   # especializacao por responsabilidade
```

### Regras

- [`00-global.md`](.agents/rules/00-global.md)
- [`10-architecture.md`](.agents/rules/10-architecture.md)
- [`20-coding.md`](.agents/rules/20-coding.md)
- [`30-testing.md`](.agents/rules/30-testing.md)
- [`40-documentation.md`](.agents/rules/40-documentation.md)
- [`50-security.md`](.agents/rules/50-security.md)
- [`60-delivery.md`](.agents/rules/60-delivery.md)
- [`70-phase-governance.md`](.agents/rules/70-phase-governance.md)

### Skills

- [`foundation-runtime.md`](.agents/skills/foundation-runtime.md)
- [`modular-monolith-ddd.md`](.agents/skills/modular-monolith-ddd.md)
- [`testing-and-tdd.md`](.agents/skills/testing-and-tdd.md)
- [`observability-and-operations.md`](.agents/skills/observability-and-operations.md)
- [`documentation-and-governance.md`](.agents/skills/documentation-and-governance.md)

### Subagentes

- [`architecture-steward.md`](.agents/subagents/architecture-steward.md)
- [`foundation-engineer.md`](.agents/subagents/foundation-engineer.md)
- [`domain-evolution-engineer.md`](.agents/subagents/domain-evolution-engineer.md)
- [`quality-and-release-guardian.md`](.agents/subagents/quality-and-release-guardian.md)

## Padroes inegociaveis

- modular monolith com limites explicitos entre modulos
- DDD para modelagem de dominio
- clean architecture com dependencias apontando para dentro
- nenhuma regra de negocio em handlers HTTP, adapters ou detalhes de infraestrutura
- banco compartilhado fisicamente nao significa acesso livre entre modulos
- comunicacao entre modulos deve preferir eventos internos; chamadas diretas sao excecao explicita
- `internal/shared` nao pode virar deposito de acoplamento ou regra de negocio

> Se um modulo depender da implementacao interna de outro modulo, a arquitetura esta quebrada.

## Design patterns do projeto

- Modular Monolith
- Domain-Driven Design
- Clean Architecture
- Ports and Adapters
- Repository Pattern
- Internal Event Bus Pattern
- Idempotent Consumer Pattern
- Transaction boundary local para fluxos financeiros
- Request-scoped e event-scoped logging com `request_id`
- OpenTelemetry Instrumentation Pattern
- Prometheus Pull Metrics Pattern
- Outbox Pattern com lifecycle sincronico e envelope padronizado
- Error mapping explicito entre dominio, aplicacao e HTTP
- Observability conventions para spans, metricas e categorias de erro

## Common hurdles

### Docker indisponivel

- sintoma: `docker compose` ou `testcontainers-go` falha
- acao: validar daemon/socket antes de alterar codigo

### `request_id` ausente

- sintoma: resposta HTTP ou log de evento sem rastreabilidade
- acao: revisar `internal/shared/correlation`, `internal/shared/http` e o uso do header `X-Correlation-ID`

### Evento processado fora de ordem

- sintoma: billing ou invoices em estado inesperado apos pagamento
- acao: revisar ordem de `Subscribe` no bootstrap e a idempotencia dos handlers

### Retry manual sem billing existente

- sintoma: `POST /payments` retorna `billing_not_found`
- acao: validar se a invoice foi criada no fluxo oficial e se a cobranca foi persistida antes do retry

### Timeout de gateway

- sintoma: tentativa em `Failed` com `failure_category=gateway_timeout`
- acao: revisar `PAYMENT_GATEWAY_TIMEOUT_MS`, adapter do gateway e a categoria persistida em `payments`

### Trace ausente no Jaeger

- sintoma: request executa, mas o trace nao aparece
- acao: validar `OTEL_EXPORTER_OTLP_ENDPOINT`, servico `jaeger` no Compose e propagacao de `traceparent`

### Metricas ausentes no Prometheus

- sintoma: `atlas_erp_*` nao aparece em `/metrics` ou no Prometheus
- acao: revisar `GET /metrics`, configuracao do `prometheus.yml` e startup do `prometheus`

### Duplicacao de evento financeiro

- sintoma: o mesmo `BillingRequested` reaparece
- acao: validar `attempt_number`, `idempotency_key` e a unicidade em `payments (billing_id, attempt_number)`

### Divergencia entre docs e implementacao

- sintoma: README, AGENTS, diagrams ou commands nao batem com o codigo
- acao: corrigir a divergencia no mesmo change set; nunca deixar "para depois"

### Erro interno vazando detalhe tecnico

- sintoma: stack trace, query ou erro bruto aparecendo no response
- acao: mapear para `internal_error`, logar so na borda e preservar mensagem generica ao cliente

## Contrato operacional minimo

Antes de iniciar uma mudanca:

- identificar fase atual e escopo permitido
- identificar modulo(s) e eventos afetados
- carregar apenas as rules/skills necessarias
- registrar hipoteses quando o contexto estiver incompleto

Antes de concluir uma mudanca:

- validar arquitetura, testes, documentacao e impacto operacional
- registrar limitacoes, riscos e pendencias
- atualizar `README.md` e `CHANGELOG.md` quando houver impacto relevante
- atualizar ADR ou diagramas quando a decisao for estrutural
- gerar handoff compacto se a sessao nao encerrar o assunto

## Regra de handoff

Quando houver interrupcao, troca de sessao ou continuacao posterior, registrar handoff usando `.agents/templates/handoff.md` com:

- objetivo
- escopo executado
- arquivos alterados
- decisoes
- pendencias
- riscos
- evidencias de validacao

## Regra de validacao antes de concluir tarefa

Nenhuma tarefa deve ser considerada concluida sem evidencia proporcional ao risco:

- codigo: build/lint/teste compativel com a mudanca
- arquitetura: fronteiras preservadas e imports coerentes
- runtime: comandos principais ainda funcionam
- documentacao: artefatos afetados atualizados
- seguranca: ausencia de segredo exposto e ausencia de mudanca destrutiva implicita

## Instrucoes para CHANGELOG.md

- se `CHANGELOG.md` nao existir, criar antes de concluir a entrega
- atualizar no mesmo change set da implementacao
- registrar comportamento entregue, nao apenas lista de arquivos
- agrupar por versao, marco ou fase
- usar `Added`, `Changed`, `Fixed` e `Removed`
- nao deixar mudanca arquitetural, de contrato HTTP, observabilidade ou testes sem registro

## Instrucoes para README.md

- manter sempre alinhado com a fase atual
- refletir stack, setup, env vars, comandos, endpoints, estrategia de testes e observabilidade
- atualizar exemplos de request/response quando o contrato mudar
- usar o README como entrada principal para quem for operar ou revisar o projeto

## Diagramas e modelagem visual

- construir diagramas usando **Mermaid**
- preferir **C4Model** para contexto, containers e componentes
- atualizar `docs/diagrams/architecture.md` quando mudar fluxo, fronteira modular, observabilidade ou contrato operacional

## Instrucoes para TDD

Ciclo obrigatorio:

1. escrever um teste que falha
2. implementar o minimo para faze-lo passar
3. refatorar preservando comportamento

Regras complementares:

- bug corrigido ganha teste de regressao
- regra nova precisa de teste proporcional ao risco
- preferir teste de comportamento observavel
- mock nao pode esconder regra de negocio
- usar `testcontainers-go` quando infraestrutura real fizer parte do comportamento validado

## Checklist pos-implementacao

- `rtk gofmt -w <arquivos>`
- `rtk go vet ./...`
- `rtk go test ./internal/...`
- `rtk go test ./test/integration/...`
- `rtk go test ./test/functional/...`
- `rtk go test ./...`
- `rtk docker compose up --build -d`
- revisar `README.md`
- revisar `CHANGELOG.md`
- revisar `docs/commands.md`
- revisar `docs/diagrams/architecture.md`
- revisar `docs/adr/`
- revisar `.agents/templates/phase-status.md`

## Quando consultar codigo, docs ou integracoes

Consulte o **codigo-fonte** quando houver conflito entre documentacao e implementacao.
Consulte `README.md`, `docs/commands.md`, `docs/adr/` e `docs/diagrams/` antes de propor mudanca estrutural ou operacional.
Consulte integracoes externas, MCP, plugins ou ferramentas adicionais apenas quando a tarefa realmente exigir; por padrao, preferir contexto local do repositorio.

## Opcoes avancadas assumidas nesta versao

Assumidas de forma conservadora, ate decisao explicita em contrario:

- **CLI/skills no lugar de MCP:** preferido apenas para fluxos locais simples e repetitivos
- **autoevolucao assistida de rules/skills:** desabilitada por padrao; so com revisao humana
- **paralelizacao com multiplos agentes/worktrees:** desabilitada por padrao; so com particao explicita de dominio e handoff obrigatorio

## Ordem sugerida de uso no dia a dia

1. `AGENTS.md`
2. `00-global.md`
3. `70-phase-governance.md`
4. rule especializada da tarefa
5. skill do fluxo
6. subagente, se houver especializacao real
7. template de review ou handoff no fechamento
