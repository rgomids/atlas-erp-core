# Command Reference

Este arquivo centraliza os comandos operacionais oficiais da Phase 7.

Superficie preferencial:

- comandos diretos com `rtk`
- `docker compose` para subir a stack local
- `go test` para validacao e benchmark

O Makefile permanece apenas como conveniencia secundaria e nao e a interface principal documentada.

## Preparacao local

### 1. Criar `.env` local

```bash
rtk cp .env.example .env
```

### 2. Subir a stack oficial

```bash
rtk docker compose up --build -d
```

Servicos esperados:

- API em `http://localhost:8080`
- PostgreSQL em `localhost:5432`
- Jaeger em `http://localhost:16686`
- Prometheus em `http://localhost:9090`

### 3. Rodar migracoes manualmente

```bash
rtk go run ./cmd/migrate --direction up
```

### 4. Rodar a API fora do Compose

Use este caminho quando quiser iterar localmente no binario da API mantendo PostgreSQL, Jaeger e Prometheus no Compose:

```bash
rtk env OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/api
```

### 5. Encerrar a stack

```bash
rtk docker compose down --remove-orphans
```

## Validacao rapida

### Healthcheck

```bash
rtk curl http://localhost:8080/health
```

### Metricas

```bash
rtk curl http://localhost:8080/metrics
```

## Fluxo principal

### Criar cliente

```bash
rtk curl -X POST http://localhost:8080/customers \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-ID: demo-phase7-001' \
  -d '{"name":"Atlas Co","document":"12345678900","email":"team@atlas.io"}'
```

### Criar invoice e disparar o fluxo automatico

```bash
rtk curl -X POST http://localhost:8080/invoices \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-ID: demo-phase7-002' \
  -d '{"customer_id":"<customer-id>","amount_cents":1599,"due_date":"2026-03-31"}'
```

### Listar invoices do cliente

```bash
rtk curl http://localhost:8080/customers/<customer-id>/invoices \
  -H 'X-Correlation-ID: demo-phase7-003'
```

### Retry manual de pagamento

```bash
rtk curl -X POST http://localhost:8080/payments \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-ID: demo-phase7-004' \
  -d '{"invoice_id":"<invoice-id>"}'
```

## Testes

### Suite completa

```bash
rtk go test ./...
```

### Testes unitarios

```bash
rtk go test ./internal/...
```

### Testes de integracao

```bash
rtk go test ./test/integration/...
```

### Testes funcionais

```bash
rtk go test ./test/functional/...
```

## Benchmark reproduzivel

### Rodar benchmarks HTTP da Phase 7

```bash
rtk proxy go test -run '^$' -bench . -benchmem -benchtime=10x ./test/benchmark
```

### Gerar baseline em JSON e Markdown

```bash
rtk proxy go test -run '^$' -bench . -benchmem -benchtime=10x ./test/benchmark \
  -args \
  -report-json docs/benchmarks/phase7-baseline.json \
  -report-md docs/benchmarks/phase7-baseline.md
```

Saidas esperadas:

- `avg_ms`
- `p95_ms`
- `ops/s`
- `error_rate_pct`

Se Docker ou `testcontainers-go` nao estiverem disponiveis, os artefatos ainda sao gerados com `status: no_samples` e uma nota explicando o pre-requisito ausente.
Use `rtk proxy go test` nesses comandos para preservar os flags de benchmark e exportacao.

## Perfis de falha controlada

Todos os perfis abaixo sao exclusivos para avaliacao local e dev. Em `APP_ENV=production`, `ATLAS_FAULT_PROFILE` deve ser `none`.

### Sem falha injetada

```bash
rtk env ATLAS_FAULT_PROFILE=none OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/api
```

### Timeout no gateway

```bash
rtk env ATLAS_FAULT_PROFILE=payment_timeout OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/api
```

### Primeira chamada ao gateway falha, retry manual pode aprovar

```bash
rtk env ATLAS_FAULT_PROFILE=payment_flaky_first OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/api
```

### Primeira entrega de `BillingRequested` e duplicada

```bash
rtk env ATLAS_FAULT_PROFILE=duplicate_billing_requested OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/api
```

### Primeira entrega de `BillingRequested` para `payments` falha

```bash
rtk env ATLAS_FAULT_PROFILE=event_consumer_failure OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/api
```

### Primeiro append no outbox falha antes dos consumidores

```bash
rtk env ATLAS_FAULT_PROFILE=outbox_append_failure OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/api
```

## Observabilidade

### Traces

- UI local: `http://localhost:16686`
- service name esperado: `atlas-erp-core`

### Prometheus

- UI local: `http://localhost:9090`

### Span names principais

- `http.request {METHOD} {route}`
- `application.usecase {module}.{UseCase}`
- `event.publish {EventName}`
- `event.consume {consumer_module}.{EventName}`
- `db.query {operation} {table}`
- `integration.gateway payments.Process`

### Metricas principais

- `atlas_erp_http_requests_total`
- `atlas_erp_http_request_errors_total`
- `atlas_erp_http_request_duration_seconds`
- `atlas_erp_events_published_total`
- `atlas_erp_events_consumed_total`
- `atlas_erp_event_handler_failures_total`
- `atlas_erp_payment_retries_total`
- `atlas_erp_db_query_duration_seconds`
- `atlas_erp_gateway_request_duration_seconds`
- `atlas_erp_gateway_failures_total`

## Diagnostico rapido

### Timeout de gateway

- validar `PAYMENT_GATEWAY_TIMEOUT_MS`
- conferir `failure_category=gateway_timeout`
- conferir traces `integration.gateway payments.Process`

### Duplicidade de evento

- usar `ATLAS_FAULT_PROFILE=duplicate_billing_requested`
- validar que ha apenas um pagamento `Approved` por invoice
- conferir `attempt_number` e `idempotency_key`

### Falha no consumo interno

- usar `ATLAS_FAULT_PROFILE=event_consumer_failure`
- validar `outbox_events.status=failed`
- conferir ausencia de `PaymentApproved`

### Falha no append do outbox

- usar `ATLAS_FAULT_PROFILE=outbox_append_failure`
- validar persistencia do aggregate upstream
- validar ausencia de consumidores downstream e de registro no outbox
