# AGENTS.md

## Missão do projeto

`atlas-erp-core` evolui um **modular monolith em Go** para o domínio central de ERP, preservando fronteiras explícitas entre módulos, regras de negócio encapsuladas no domínio e uma base operacional simples, reproduzível e auditável.

## Fase atual

**Phase 2 - Quality & Engineering**

Estado de referência desta fase:

- foundation da Phase 0 continua válida
- fluxo funcional ponta a ponta da Phase 1 continua ativo
- contratos HTTP da borda estão padronizados
- observabilidade por request está consolidada com `request_id`
- módulos ativos: `customers`, `invoices`, `payments`
- módulo em scaffold: `billing`

O `README.md` deve sempre refletir a fase atual. Quando a fase mudar, atualizar também `.agents/templates/phase-status.md` ou o artefato de status adotado pelo repositório.

## Papel deste arquivo

Este arquivo é o **roteador principal** do engine de agentes.
Ele define como trabalhar no repositório, quais artefatos carregar por demanda e quais contratos vivos precisam permanecer sincronizados com a implementação.

## Visão geral da arquitetura

- estilo principal: **Modular Monolith**
- modelagem: **DDD**
- organização interna: **Clean Architecture**
- integração entre camadas: **Ports and Adapters**
- persistência: **PostgreSQL** com ownership lógico por módulo
- observabilidade: logging JSON, `request_id` e erro HTTP canônico
- contrato de fluxo atual:

```text
Create Customer -> Create Invoice -> Process Payment -> Invoice Paid
```

## Stack tecnológico completo

- Go 1.26
- `chi` para roteamento HTTP
- `log/slog` para logging estruturado
- `godotenv` para bootstrap de `.env`
- PostgreSQL
- `pgx/v5` para acesso ao banco
- `golang-migrate` para migrations
- Docker + Docker Compose
- `testcontainers-go` para testes com PostgreSQL real
- GitHub Actions para CI

## Ordem de leitura

Leia apenas o necessário para a tarefa atual:

1. este `AGENTS.md`
2. `.agents/rules/00-global.md`
3. `.agents/rules/70-phase-governance.md`
4. a rule especializada da tarefa:
   - arquitetura: `.agents/rules/10-architecture.md`
   - implementação: `.agents/rules/20-coding.md`
   - testes: `.agents/rules/30-testing.md`
   - documentação: `.agents/rules/40-documentation.md`
   - segurança operacional: `.agents/rules/50-security.md`
   - entrega: `.agents/rules/60-delivery.md`
5. a skill do fluxo, quando houver ganho direto:
   - runtime/config/bootstrap: `.agents/skills/foundation-runtime.md`
   - domínio/arquitetura: `.agents/skills/modular-monolith-ddd.md`
   - testes/TDD: `.agents/skills/testing-and-tdd.md`
   - observabilidade/operação: `.agents/skills/observability-and-operations.md`
   - documentação/governança: `.agents/skills/documentation-and-governance.md`
6. subagentes apenas se houver especialização real e particionamento seguro
7. templates apenas no momento de abrir tarefa, revisar ou registrar handoff

## Política de contexto mínimo

Carregue o mínimo suficiente para executar com segurança:

- para mudar um módulo, leia a rule de arquitetura e a skill do domínio correspondente
- para mudar runtime, bootstrap, config, compose, CI ou observabilidade, leia `foundation-runtime.md` e `observability-and-operations.md`
- para mudar comportamento de domínio, leia `modular-monolith-ddd.md` e `testing-and-tdd.md`
- para encerrar uma entrega, valide `documentation-and-governance.md` e `review-checklist.md`
- não abra todos os arquivos de `.agents` por padrão

## Variáveis de ambiente

| Variável | Obrigatória | Default | Uso |
| --- | --- | --- | --- |
| `APP_PORT` | Sim | - | Porta HTTP da API |
| `DB_HOST` | Sim | - | Host do PostgreSQL |
| `DB_PORT` | Sim | - | Porta do PostgreSQL |
| `DB_USER` | Sim | - | Usuário do PostgreSQL |
| `DB_PASSWORD` | Sim | - | Senha do PostgreSQL |
| `DB_NAME` | Sim | - | Banco do PostgreSQL |
| `APP_NAME` | Não | `atlas-erp-core` | Nome lógico da aplicação |
| `APP_ENV` | Não | `local` | Ambiente atual |
| `LOG_LEVEL` | Não | `info` | Nível de log estruturado |
| `CORRELATION_ID_HEADER` | Não | `X-Correlation-ID` | Header usado para propagar `request_id` |

## Estrutura do diretório de conteúdo

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
│   ├── shared/
│   ├── customers/
│   ├── invoices/
│   ├── payments/
│   └── billing/
├── migrations/
├── test/
│   ├── functional/
│   ├── integration/
│   └── support/
├── CHANGELOG.md
├── Makefile
└── README.md
```

## Serviços, jobs e models por app

### Shared

- serviços/capacidades: `config`, `http`, `correlation`, `logging`, `postgres`
- jobs: nenhum
- models: apenas primitives técnicas e utilitários transversais estáveis

### Customers

- services/use cases: `CreateCustomer`, `UpdateCustomer`, `DeactivateCustomer`
- jobs: nenhum
- models: `Customer`, `Document`, `Email`

### Invoices

- services/use cases: `CreateInvoice`, `ListCustomerInvoices`
- jobs: nenhum
- models: `Invoice`

### Payments

- services/use cases: `ProcessPayment`
- jobs: nenhum
- models: `Payment`

### Billing

- services/use cases: nenhum na fase atual
- jobs: nenhum
- models: scaffold vazio

## Como usar `.agents`

```text
.agents/
├── rules/       # regras permanentes por responsabilidade
├── templates/   # handoff, fase, task brief e review
├── skills/      # workflows repetitivos e encapsuláveis
└── subagents/   # especialização por responsabilidade
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

## Padrões inegociáveis

- modular monolith com limites explícitos entre módulos
- DDD para modelagem de domínio
- clean architecture com dependências apontando para dentro
- nenhuma regra de negócio em handlers, adapters ou detalhes de infraestrutura
- banco compartilhado fisicamente não significa acesso livre entre módulos
- contratos síncronos entre módulos são exceção explícita; preferir eventos internos quando fizer sentido em fases futuras
- `internal/shared` não pode virar depósito de acoplamento ou regra de negócio

> Se um módulo depender da implementação interna de outro módulo, a arquitetura está quebrada.

## Design patterns do projeto

- Modular Monolith
- Domain-Driven Design
- Clean Architecture
- Ports and Adapters
- Repository Pattern
- Transaction boundary local para fluxos financeiros
- Request-scoped logging com `request_id`
- Error mapping explícito entre domínio, aplicação e HTTP

## Common hurdles

### Docker indisponível

- sintoma: `docker compose` ou `testcontainers-go` falha
- ação: validar daemon/socket antes de alterar código

### `request_id` ausente

- sintoma: resposta HTTP sem rastreabilidade
- ação: revisar `internal/shared/correlation` e o uso do header `X-Correlation-ID`

### Divergência entre docs e implementação

- sintoma: README, AGENTS, diagrams ou commands não batem com o código
- ação: corrigir a divergência no mesmo change set; nunca deixar "para depois"

### Flakiness em testes com PostgreSQL real

- sintoma: testes de integração/funcionais instáveis
- ação: evitar `t.Parallel()` em cenários com `testcontainers-go`, validar migrations e isolar fixtures por teste

### Erro interno vazando detalhe técnico

- sintoma: stack trace, query ou erro bruto aparecendo no response
- ação: mapear para `internal_error`, logar só na borda e preservar mensagem genérica ao cliente

## Contrato operacional mínimo

Antes de iniciar uma mudança:

- identificar fase atual e escopo permitido
- identificar módulo(s) afetado(s)
- carregar apenas as rules/skills necessárias
- registrar hipóteses quando o contexto estiver incompleto

Antes de concluir uma mudança:

- validar arquitetura, testes, documentação e impacto operacional
- registrar limitações, riscos e pendências
- atualizar `README.md` e `CHANGELOG.md` quando houver impacto relevante
- atualizar ADR ou diagramas quando a decisão for estrutural
- gerar handoff compacto se a sessão não encerrar o assunto

## Regra de handoff

Quando houver interrupção, troca de sessão ou continuação posterior, registrar handoff usando `.agents/templates/handoff.md` com:

- objetivo
- escopo executado
- arquivos alterados
- decisões
- pendências
- riscos
- evidências de validação

## Regra de validação antes de concluir tarefa

Nenhuma tarefa deve ser considerada concluída sem evidência proporcional ao risco:

- código: build/lint/teste compatível com a mudança
- arquitetura: fronteiras preservadas e imports coerentes
- runtime: comandos principais ainda funcionam
- documentação: artefatos afetados atualizados
- segurança: ausência de segredo exposto e ausência de mudança destrutiva implícita

## Instruções para CHANGELOG.md

- se `CHANGELOG.md` não existir, criar antes de concluir a entrega
- atualizar no mesmo change set da implementação
- registrar comportamento entregue, não apenas lista de arquivos
- agrupar por versão, marco ou fase
- usar `Added`, `Changed`, `Fixed` e `Removed`
- não deixar mudança arquitetural, de contrato HTTP, observabilidade ou testes sem registro

## Instruções para README.md

- manter sempre alinhado com a fase atual
- refletir stack, setup, env vars, comandos, endpoints, estratégia de testes e observabilidade
- atualizar exemplos de request/response quando o contrato mudar
- usar o README como entrada principal para quem for operar ou revisar o projeto

## Diagramas e modelagem visual

- construir diagramas usando **Mermaid**
- preferir **C4Model** para contexto, containers e componentes
- atualizar `docs/diagrams/architecture.md` quando mudar fluxo, fronteira modular, observabilidade ou contrato operacional

## Instruções para TDD

Ciclo obrigatório:

1. escrever um teste que falha
2. implementar o mínimo para fazê-lo passar
3. refatorar preservando comportamento

Regras complementares:

- bug corrigido ganha teste de regressão
- regra nova precisa de teste proporcional ao risco
- preferir teste de comportamento observável
- mock não pode esconder regra de negócio
- usar `testcontainers-go` quando infraestrutura real fizer parte do comportamento validado

## Checklist pós-implementação

- `make fmt`
- `make lint`
- `make test-unit`
- `make test-integration`
- `make test-functional`
- `make test`
- revisar `README.md`
- revisar `CHANGELOG.md`
- revisar `docs/commands.md`
- revisar `docs/diagrams/architecture.md`
- revisar `.agents/templates/phase-status.md`

## Quando consultar código, docs ou integrações

Consulte o **código-fonte** quando houver conflito entre documentação e implementação.
Consulte `README.md`, `docs/commands.md`, `docs/adr/` e `docs/diagrams/` antes de propor mudança estrutural ou operacional.
Consulte integrações externas, MCP, plugins ou ferramentas adicionais apenas quando a tarefa realmente exigir; por padrão, preferir contexto local do repositório.

## Opções avançadas assumidas nesta versão

Assumidas de forma conservadora, até decisão explícita em contrário:

- **CLI/skills no lugar de MCP:** preferido apenas para fluxos locais simples e repetitivos
- **autoevolução assistida de rules/skills:** desabilitada por padrão; só com revisão humana
- **paralelização com múltiplos agentes/worktrees:** desabilitada por padrão; só com partição explícita de domínio e handoff obrigatório

## Ordem sugerida de uso no dia a dia

1. `AGENTS.md`
2. `00-global.md`
3. `70-phase-governance.md`
4. rule especializada da tarefa
5. skill do fluxo
6. subagente, se houver especialização real
7. template de review ou handoff no fechamento
