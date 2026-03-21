# AGENTS.md

## Contexto

O Atlas ERP Core está na **Phase 0 — Foundation**. A base técnica já existe, mas o domínio de negócio ainda não foi implementado.

Este arquivo agora funciona como **contrato de alto nível e índice de governança**. As regras detalhadas de evolução foram extraídas para `.agents/roles` e `.agents/skills`.

## Princípios inegociáveis

- Modular Monolith com limites explícitos entre módulos.
- DDD para modelagem de domínio quando os bounded contexts evoluírem.
- Clean Architecture com dependências apontando para dentro.
- Comunicação entre módulos preferencialmente por eventos internos, não por acoplamento direto.
- Nenhuma lógica de negócio em handlers, adapters ou detalhes de infraestrutura.

### Regra de ouro

> Se um módulo depender da implementação interna de outro módulo, a arquitetura está quebrada.

## Estado atual

- Foundation operacional em Go com `chi`, `slog`, `pgx`, `golang-migrate`, Docker Compose e CI.
- `GET /health` implementado.
- PostgreSQL conectado no bootstrap.
- `customers`, `billing`, `invoices` e `payments` existem apenas como scaffold estrutural.
- Ainda não existem aggregates, entidades, casos de uso de domínio ou integrações externas reais.

## Bounded contexts de referência

- `customers`
- `billing`
- `invoices`
- `payments`

Os detalhes de responsabilidades, modelos esperados e regras de fronteira vivem em [modular-monolith-ddd.md](/Users/rgomids/Projects/atlas-erp-core/.agents/skills/modular-monolith-ddd.md).

## Como usar `.agents`

### roles

- [architecture-steward.md](/Users/rgomids/Projects/atlas-erp-core/.agents/roles/architecture-steward.md): protege limites modulares, coerência arquitetural e decisões estruturais.
- [foundation-engineer.md](/Users/rgomids/Projects/atlas-erp-core/.agents/roles/foundation-engineer.md): mantém runtime, bootstrap, config, Docker, CI e infraestrutura da foundation.
- [domain-evolution-engineer.md](/Users/rgomids/Projects/atlas-erp-core/.agents/roles/domain-evolution-engineer.md): conduz a evolução dos módulos de negócio a partir da base da Phase 0.
- [quality-and-release-guardian.md](/Users/rgomids/Projects/atlas-erp-core/.agents/roles/quality-and-release-guardian.md): garante testes, documentação, changelog e definição de pronto.

### skills

- [foundation-runtime.md](/Users/rgomids/Projects/atlas-erp-core/.agents/skills/foundation-runtime.md): stack, estrutura de diretórios, ambiente, runtime e comandos da fase atual.
- [modular-monolith-ddd.md](/Users/rgomids/Projects/atlas-erp-core/.agents/skills/modular-monolith-ddd.md): limites modulares, camadas, bounded contexts e evolução para domínio.
- [testing-and-tdd.md](/Users/rgomids/Projects/atlas-erp-core/.agents/skills/testing-and-tdd.md): estratégia de testes, TDD e cobertura mínima por camada.
- [observability-and-operations.md](/Users/rgomids/Projects/atlas-erp-core/.agents/skills/observability-and-operations.md): logs, correlation ID, runtime local, compose e troubleshooting operacional.
- [documentation-and-governance.md](/Users/rgomids/Projects/atlas-erp-core/.agents/skills/documentation-and-governance.md): README, CHANGELOG, ADR, diagramas, checklist e política de atualização.

## Regras de atualização

- Mudança estrutural relevante exige revisão do `AGENTS.md` raiz e do skill/role afetado.
- Toda alteração operacional precisa refletir README, CHANGELOG e o skill correspondente.
- Sempre preferir simplicidade, baixo acoplamento e documentação rastreável.
