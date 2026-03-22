# ADR 0005 - Phase 5 Observability and Operations

## Status

Accepted

## Context

Ao final da Phase 4, o fluxo financeiro ja era resiliente e auditavel, mas ainda faltavam sinais operacionais mais ricos para diagnostico rapido:

- o `request_id` ajudava na correlacao, mas nao existia tracing ponta a ponta
- nao havia metricas tecnicas para HTTP, eventos, banco e gateway
- troubleshooting de timeout, retry e falha tecnica ainda dependia mais de leitura de logs do que do conjunto de sinais
- o ambiente local nao oferecia uma stack simples para inspecao de traces e metricas

## Decision

Adotar as seguintes decisoes para a Phase 5:

1. usar OpenTelemetry como base unica para traces e metricas
2. manter `slog` como sistema de logs estruturados, enriquecido com `trace_id` e `span_id`
3. instrumentar HTTP, use cases, event bus, PostgreSQL e gateway de pagamento
4. expor metricas em `GET /metrics`
5. padronizar categorias de erro em `validation_error`, `domain_error`, `integration_error` e `infrastructure_error`
6. subir Jaeger all-in-one e Prometheus no ambiente local via `docker compose`

## Consequences

### Positive

- o fluxo principal passa a ser rastreavel ponta a ponta
- retries, falhas de gateway e handlers com erro ficam mais visiveis
- o projeto ganha uma convencao unica para spans, metricas e logs
- operacao local fica simples, sem exigir collector, Grafana ou stack adicional

### Negative

- o bootstrap da aplicacao fica um pouco mais rico
- a instrumentacao aumenta a superficie de testes e documentacao
- a stack local passa a depender de mais dois containers

## Notes

- esta ADR nao introduz logs do OpenTelemetry; a fase usa `slog` como fonte de logs
- esta ADR nao adiciona mensageria externa, microservices, CQRS, Tempo ou Grafana
- Jaeger all-in-one e Prometheus sao adotados apenas como stack local minima de observabilidade
