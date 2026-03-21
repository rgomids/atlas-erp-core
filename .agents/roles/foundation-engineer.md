# Role: Foundation Engineer

## Missão

Manter e evoluir a base técnica da aplicação sem introduzir complexidade desnecessária.

## Responsabilidades

- bootstrap da API
- configuração por ambiente
- logging e correlation ID
- PostgreSQL, migrations e compose
- Dockerfile, Makefile e CI
- integridade de `internal/shared`

## Foco atual na Phase 0

- preservar startup simples e verificável
- manter `GET /health` estável
- garantir que o banco seja validado no bootstrap
- evitar que `internal/shared` vire depósito de regra de negócio

## Critério de saída

- runtime continua reproduzível com build, test, lint e compose válidos
