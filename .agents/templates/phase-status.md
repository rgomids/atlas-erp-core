# Phase Status

## Fase corrente

- Nome: Phase 5 - Observability & Operations
- Status: active

## Objetivo da fase

Tornar o fluxo principal rastreavel ponta a ponta com traces, metricas e logs operacionais uteis, mantendo baixo acoplamento e sem alterar regras de negocio.

## Escopo permitido

- instrumentar HTTP, use cases, PostgreSQL, event bus e gateway com OpenTelemetry
- expor metricas tecnicas relevantes em `/metrics`
- enriquecer logs com `request_id`, `trace_id`, `span_id`, `event_name`, ids de dominio e `error_type`
- documentar convencoes de observabilidade, troubleshooting e operacao local
- subir Jaeger e Prometheus junto de `make up`

## Entregaveis esperados

- traces do fluxo `POST /invoices -> InvoiceCreated -> BillingRequested -> PaymentApproved|PaymentFailed`
- metricas HTTP, de eventos, de banco e de gateway disponiveis em Prometheus
- logs operacionais com contexto minimo consistente
- stack local simples de observabilidade funcionando com Jaeger e Prometheus
- README, AGENTS, commands, diagrams, ADR e changelog atualizados para Phase 5

## Criterios de conclusao

- o fluxo principal pode ser rastreado ponta a ponta
- metricas principais estao expostas e nomeadas de forma consistente
- falhas de gateway e retries ficam visiveis em traces, metricas e logs
- documentacao critica reflete a arquitetura e a operacao da Phase 5

## Restricoes

- nao introduzir microservices
- nao adicionar mensageria externa
- nao alterar regras de negocio nem payloads de eventos
- nao subir collector, Tempo ou Grafana sem necessidade real

## Riscos aceitos

- Jaeger all-in-one e Prometheus existem apenas para desenvolvimento local
- o outbox continua sincronico e sem dispatcher assincrono
- sinais de logs do OpenTelemetry permanecem fora do escopo desta fase

## Proximos marcos

- avaliar dashboards e alertas somente quando houver sinais reais de operacao continua
- decidir sobre exportacao remota de traces e metricas por ambiente
- revisar quando o outbox assincrono ou SPM fizer sentido operacional
