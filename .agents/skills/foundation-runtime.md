# Skill: Foundation Runtime

## Quando usar

Use esta skill ao tocar em:

- bootstrap da API
- configuração por ambiente
- compose, Dockerfile e CI
- healthcheck
- banco, migrations e comandos principais
- utilitários transversais em `internal/shared`

## Contexto mínimo a carregar

- `AGENTS.md`
- `.agents/rules/20-coding.md`
- `.agents/rules/50-security.md`
- `.agents/rules/70-phase-governance.md`

## Baseline vigente

- Go
- `chi`
- `slog`
- `.env` com `godotenv`
- PostgreSQL
- `pgx/v5`
- `golang-migrate`
- Docker + Docker Compose
- GitHub Actions
- `testcontainers-go`
- `Makefile`

## Regras específicas

- preservar startup simples e reproduzível
- `GET /health` deve continuar estável
- banco precisa ser validado no bootstrap quando esse for o comportamento atual do sistema
- não mover regra de negócio para `internal/shared`
- mudanças de runtime exigem revisão de `README.md`, `CHANGELOG.md` e `docs/commands.md`

## Checklist rápido

- build continua íntegro
- comandos principais continuam válidos
- env vars continuam coerentes
- migrations acompanham mudanças de schema
- documentação operacional foi atualizada
