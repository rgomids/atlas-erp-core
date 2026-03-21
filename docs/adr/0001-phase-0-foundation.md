# ADR 0001: Foundation da Fase 0

- Status: Accepted
- Date: 2026-03-21

## Context

O repositório estava vazio e precisava de uma base operacional minima para sustentar a evolucao do modular monolith sem antecipar regras de negocio.

## Decision

- Usar Go como runtime principal com `chi` na borda HTTP.
- Adotar `internal/shared` como local das utilidades transversais da foundation.
- Usar `DB_*` como contrato canonico de banco nesta fase e derivar a connection string internamente.
- Exigir PostgreSQL disponivel no bootstrap da API.
- Manter Redis e OpenTelemetry apenas como baseline documentada para fases futuras.
- Padronizar logs estruturados sem emojis, com correlation ID desde a borda HTTP.

## Consequences

- A aplicacao nasce com bootstrap simples e verificavel.
- A base documental fica coerente com a implementacao real da foundation.
- O projeto evita introduzir contratos de dominio ou integracoes externas antes da hora.
