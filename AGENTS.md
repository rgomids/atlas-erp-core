# AGENTS.md

## Missão do projeto

`atlas-erp-core` evolui um **modular monolith em Go** para o domínio central de ERP, preservando fronteiras explícitas entre módulos, regras de negócio encapsuladas no domínio e uma base operacional simples, reproduzível e auditável.

## Fase atual

**Phase 1 — Core Domain**

Estado de referência desta fase:

- foundation da Phase 0 continua válida
- fluxo funcional ponta a ponta já existe
- módulos ativos: `customers`, `invoices`, `payments`
- módulo em scaffold: `billing`

O `README.md` deve sempre refletir a fase atual. Quando a fase mudar, atualizar também `.agents/templates/phase-status.md` ou o artefato de status adotado pelo repositório.

## Papel deste arquivo

Este arquivo é o **roteador principal** do engine de agentes.  
Ele define como trabalhar no repositório e quais documentos carregar por demanda.  
Ele **não** concentra guideline detalhado de código, testes ou segurança.

## Ordem de leitura

Leia apenas o necessário para a tarefa atual:

1. este `AGENTS.md`
2. `.agents/rules/00-global.md`
3. `.agents/rules/70-phase-governance.md`
4. a rule especializada do domínio da tarefa:
   - arquitetura: `.agents/rules/10-architecture.md`
   - implementação: `.agents/rules/20-coding.md`
   - testes: `.agents/rules/30-testing.md`
   - documentação: `.agents/rules/40-documentation.md`
   - segurança operacional: `.agents/rules/50-security.md`
   - entrega: `.agents/rules/60-delivery.md`
5. skills e subagentes apenas se houver ganho claro de contexto
6. templates apenas no momento de abrir tarefa, revisar ou registrar handoff

## Política de contexto mínimo

Carregue o mínimo suficiente para executar com segurança:

- para mudar um módulo, leia a rule de arquitetura e a skill do domínio correspondente
- para mudar runtime, bootstrap, config, compose, CI ou observabilidade, leia `foundation-runtime.md` e `observability-and-operations.md`
- para mudar comportamento de domínio, leia `modular-monolith-ddd.md` e `testing-and-tdd.md`
- para encerrar uma entrega, valide `documentation-and-governance.md` e `review-checklist.md`
- não abra todos os arquivos de `.agents` por padrão

## Como usar `.agents`

```text
.agents/
├── rules/       # regras permanentes por responsabilidade
├── templates/   # handoff, fase, task brief e revisão
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
- contratos síncronos entre módulos são exceção explícita; preferir eventos internos quando fizer sentido
- `internal/shared` não pode virar depósito de acoplamento ou regra de negócio

> Se um módulo depender da implementação interna de outro módulo, a arquitetura está quebrada.

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

## Quando consultar código, docs ou integrações

Consulte o **código-fonte** quando houver conflito entre documentação e implementação.  
Consulte `README.md`, `docs/commands.md`, `docs/adr/` e `docs/diagrams/` antes de propor mudança estrutural ou operacional.  
Consulte integrações externas, MCP, plugins ou ferramentas adicionais apenas quando a tarefa realmente exigir; por padrão, preferir contexto local do repositório.

## Opções avançadas assumidas nesta versão

Assumidas de forma conservadora, até decisão explícita em contrário:

- **CLI/skills no lugar de MCP:** preferido apenas para fluxos locais simples e repetitivos
- **autoevolução assistida de rules/skills:** desabilitada por padrão; só com revisão humana
- **paralelização com múltiplos agentes/worktrees:** desabilitada por padrão; só com partição explícita de domínio e handoff obrigatório

## Estrutura de referência do repositório

```text
.
├── .agents/
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
├── README.md
├── CHANGELOG.md
└── Makefile
```

## Ordem sugerida de uso no dia a dia

1. `AGENTS.md`
2. `00-global.md`
3. `70-phase-governance.md`
4. rule especializada da tarefa
5. skill do fluxo
6. subagente, se houver especialização real
7. template de review ou handoff no fechamento
