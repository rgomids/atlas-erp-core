# Skill: Testing and TDD

## Quando usar

Use esta skill em qualquer mudança que altere comportamento, corrija bug ou adicione regra nova.

## Contexto mínimo a carregar

- `.agents/rules/30-testing.md`
- `.agents/rules/60-delivery.md`

## Estratégia

- unitário para domínio e regras puras
- integração para persistência, migrations e adapters
- funcional para fluxo HTTP e cenários ponta a ponta

## Cobertura mínima de referência da Phase 2

- `internal/shared/config`
- `internal/shared/logging`
- `internal/shared/correlation`
- `internal/shared/http`
- `internal/customers/domain`
- `internal/customers/infrastructure/http`
- `internal/invoices/domain`
- `internal/invoices/infrastructure/http`
- `internal/payments/domain`
- `internal/payments/infrastructure/http`
- `internal/*/application`
- `test/integration`
- `test/functional`

## Regras práticas

- bug corrigido ganha regressão
- teste frágil deve ser reescrito
- mock não substitui regra de negócio
- cenários com infraestrutura real devem usar `testcontainers-go` quando isso fizer parte do comportamento validado
- testes que sobem PostgreSQL real com `testcontainers-go` não devem usar `t.Parallel()`; reduzir concorrência de containers evita flakiness no CI

## Evidência mínima sugerida

Escolher o menor conjunto honesto entre:

- `rtk make test-unit`
- `rtk make test-integration`
- `rtk make test-functional`
- `rtk make test`
