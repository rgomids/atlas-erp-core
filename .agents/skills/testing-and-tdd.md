# Skill: Testing and TDD

## Objetivo

Definir a estratégia de testes para a foundation atual e para a evolução futura do domínio.

## Tipos obrigatórios de teste

- testes unitários
- testes de integração
- testes funcionais

## Mapeamento por camada

- `domain`: testes unitários puros para entidades, value objects e invariantes
- `application`: testes unitários e de orquestração de use cases
- `infrastructure`: testes de integração para persistência, handlers, migrations e integrações
- fluxos críticos: testes funcionais ou E2E

## Cobertura mínima vigente da Phase 0

- `internal/shared/config`: carregamento e validação de config
- `internal/shared/logging`: logger estruturado
- `internal/shared/correlation`: middleware de correlation ID
- `internal/shared/http`: contrato do `GET /health`
- `test/integration`: PostgreSQL real e migrations vazias com `testcontainers-go`
- `test/functional`: contrato funcional do healthcheck

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
