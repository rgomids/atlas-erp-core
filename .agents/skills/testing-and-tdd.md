# Skill: Testing and TDD

## Objetivo

Definir a estrategia de testes da foundation e da Phase 1, preservando cobertura relevante do fluxo de negocio ja entregue.

## Tipos obrigatórios de teste

- testes unitários
- testes de integração
- testes funcionais

## Mapeamento por camada

- `domain`: testes unitários puros para entidades, value objects e invariantes
- `application`: testes unitários e de orquestração de use cases
- `infrastructure`: testes de integração para persistência, handlers, migrations e integrações
- fluxos críticos: testes funcionais ou E2E

## Cobertura minima vigente da Phase 1

- `internal/shared/config`: carregamento e validação de config
- `internal/shared/logging`: logger estruturado
- `internal/shared/correlation`: middleware de correlation ID
- `internal/shared/http`: contrato do `GET /health`
- `internal/customers/domain`: regras centrais de cliente, documento e email
- `internal/invoices/domain`: valor, vencimento e imutabilidade apos pagamento
- `internal/payments/domain`: estados do pagamento
- `internal/*/application`: use cases com fakes e cenarios de duplicidade/erro
- `test/integration`: PostgreSQL real, migrations e fluxo Phase 1
- `test/functional`: contrato funcional do healthcheck e fluxo HTTP ponta a ponta

## Regras de qualidade

- Todo bug corrigido deve ganhar teste que falha antes e passa depois.
- Toda nova regra de negócio deve nascer orientada por teste.
- Teste frágil deve ser reescrito para validar comportamento observável.
- Usar `testcontainers-go` para infraestrutura real quando o cenário exigir PostgreSQL ou Redis.

## TDD obrigatório

Adotar o ciclo:

1. Escrever um teste que falha.
2. Implementar o mínimo para fazê-lo passar.
3. Refatorar preservando comportamento.

## Regras práticas

- Não usar testes para justificar acoplamento entre módulos.
- Não esconder regra de negócio em mocks ou fixtures mágicas.
- Se Docker não estiver disponível, testes dependentes de `testcontainers` podem ser pulados localmente, mas não devem ser removidos.
