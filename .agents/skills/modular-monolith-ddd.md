# Skill: Modular Monolith + DDD

## Quando usar

Use esta skill ao:

- criar ou evoluir bounded contexts
- introduzir entidades, value objects e use cases
- revisar fronteiras entre módulos
- decidir entre contrato síncrono e evento interno
- preparar expansão do `billing`

## Contexto mínimo a carregar

- `AGENTS.md`
- `.agents/rules/10-architecture.md`
- `.agents/rules/30-testing.md`
- `.agents/rules/70-phase-governance.md`

## Módulos de referência

### Ativos

- `customers`
- `invoices`
- `billing`
- `payments`

## Guardrails

- não cair em CRUD anêmico como padrão
- não compartilhar internals entre módulos
- não usar `internal/shared` para mascarar acoplamento
- não introduzir integração externa real antes de contrato claro
- preferir linguagem de domínio explícita
- preferir eventos internos ao expandir o fluxo financeiro

## Critérios de saída

- comportamento encapsulado no domínio
- use cases orquestrando sem vazar regra
- contrato entre módulos explícito
- testes cobrindo invariantes e fluxos relevantes
