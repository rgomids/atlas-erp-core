# Command Reference

Este arquivo centraliza os principais comandos operacionais do projeto e deve ser atualizado sempre que novos fluxos recorrentes forem introduzidos.

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

## Healthcheck manual

```bash
curl http://localhost:8080/health
```

## Fluxo principal da aplicação

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

### 4. Criar invoice

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

### 6. Processar pagamento

```bash
curl -X POST http://localhost:8080/payments \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-ID: demo-req-006' \
  -d '{"invoice_id":"<invoice-id>"}'
```

## Contrato de erro canônico

### Exemplo de erro de validação

```bash
curl -X POST http://localhost:8080/customers \
  -H 'Content-Type: application/json' \
  -H 'X-Correlation-ID: demo-req-007' \
  -d '{"name":"Atlas Co","email":"team@atlas.io"}'
```

Resposta esperada:

```json
{
  "error": "invalid_input",
  "message": "document is required",
  "request_id": "demo-req-007"
}
```
