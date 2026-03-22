# Command Reference

Este arquivo centraliza os principais comandos operacionais do projeto e deve ser atualizado sempre que setup, observabilidade ou fluxo recorrente mudarem.

## Configuracao por ambiente

- `.env` continua sendo a base local
- `.env.<APP_ENV>` e carregado como overlay opcional
- `PAYMENT_GATEWAY_TIMEOUT_MS` controla o timeout por tentativa de pagamento
- `OTEL_EXPORTER_OTLP_ENDPOINT` habilita exportacao de traces fora do Compose; vazio desabilita

## Setup e runtime

```bash
make setup
make up
make down
make run
make build
```

## Qualidade

```bash
make fmt
make lint
make test-unit
make test-integration
make test-functional
make test
```

## Banco e migrations

```bash
make migrate-up
make migrate-down
```

## Endpoints operacionais

### Healthcheck

```bash
curl http://localhost:8080/health
```

### Metricas Prometheus

```bash
curl http://localhost:8080/metrics
```

### Traces no Jaeger

- UI local: `http://localhost:16686`
- servico esperado: `atlas-erp-core`

### Prometheus local

- UI local: `http://localhost:9090`

## Fluxo principal da aplicacao

### 1. Criar cliente

```bash
curl -X POST http://localhost:8080/customers \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-ID: demo-req-001' \
  -d '{"name":"Atlas Co","document":"12345678900","email":"team@atlas.io"}'
```

### 2. Atualizar cliente

```bash
curl -X PUT http://localhost:8080/customers/<customer-id> \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-ID: demo-req-002' \
  -d '{"name":"Atlas Updated","email":"billing@atlas.io"}'
```

### 3. Inativar cliente

```bash
curl -X PATCH http://localhost:8080/customers/<customer-id>/inactive \
  -H 'X-Correlation-ID: demo-req-003'
```

### 4. Criar invoice e disparar o fluxo automatico

```bash
curl -X POST http://localhost:8080/invoices \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-ID: demo-req-004' \
  -d '{"customer_id":"<customer-id>","amount_cents":1599,"due_date":"2026-03-25"}'
```

### 5. Listar invoices do cliente

```bash
curl http://localhost:8080/customers/<customer-id>/invoices \
  -H 'X-Correlation-ID: demo-req-005'
```

### 6. Retry manual de pagamento apos falha

```bash
curl -X POST http://localhost:8080/payments \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-ID: demo-req-006' \
  -d '{"invoice_id":"<invoice-id>"}'
```

Resposta possivel em falha tecnica do gateway:

```json
{
  "status": "Failed",
  "attempt_number": 2,
  "failure_category": "gateway_timeout"
}
```

## Propagacao de traceparent

O contrato de `X-Correlation-ID` continua igual. Para tracing distribuido local, tambem e possivel enviar `traceparent`:

```bash
curl http://localhost:8080/customers/<customer-id>/invoices \
  -H 'X-Correlation-ID: demo-req-007' \
  -H 'traceparent: 00-11111111111111111111111111111111-2222222222222222-01'
```

## Sinais principais para troubleshooting

### Metricas HTTP

- `atlas_erp_http_requests_total`
- `atlas_erp_http_request_errors_total`
- `atlas_erp_http_request_duration_seconds`

### Metricas de aplicacao

- `atlas_erp_events_published_total`
- `atlas_erp_events_consumed_total`
- `atlas_erp_event_handler_failures_total`
- `atlas_erp_payment_retries_total`

### Metricas de persistencia e integracao

- `atlas_erp_db_query_duration_seconds`
- `atlas_erp_gateway_request_duration_seconds`
- `atlas_erp_gateway_failures_total`

## Diagnostico de falhas de pagamento

- `gateway_timeout`: o gateway excedeu `PAYMENT_GATEWAY_TIMEOUT_MS`
- `gateway_error`: erro tecnico do adapter ou resposta invalida
- `gateway_declined`: o gateway respondeu, mas o pagamento foi recusado
- `attempt_number`: numero da tentativa persistida na cobranca e no pagamento
- `retry_count`: campo operacional de observabilidade, derivado de `attempt_number - 1`

## Contrato de erro canonico

### Exemplo de erro de validacao

```bash
curl -X POST http://localhost:8080/customers \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-ID: demo-req-008' \
  -d '{"name":"Atlas Co","email":"team@atlas.io"}'
```

Resposta esperada:

```json
{
  "error": "invalid_input",
  "message": "document is required",
  "request_id": "demo-req-008"
}
```
