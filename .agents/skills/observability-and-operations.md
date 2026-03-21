# Skill: Observability and Operations

## Objetivo

Consolidar as convenções operacionais e de observabilidade que já valem na foundation.

## Observabilidade obrigatória

- logs estruturados em JSON
- correlation ID obrigatório na borda HTTP
- mensagens objetivas, textuais e sem emojis
- nenhum segredo em log

## Convenções de logging

Exemplos aceitos:

```text
INFO  api starting app_name=atlas-erp-core app_env=local app_port=8080
INFO  http request completed method=GET path=/health status_code=200 correlation_id=abc123
ERROR ping postgres failed correlation_id=abc123 err="timeout"
```

## Runtime local

- `make up` deve subir `app` e `postgres`
- `GET /health` deve responder `{"status":"ok"}`
- PostgreSQL deve estar acessível no bootstrap da aplicação

## Regras operacionais

- Preferir `make <target>` a comandos longos
- Tratar `docker-compose.yml`, `Dockerfile`, workflow de CI e `.env.example` como parte do runtime oficial
- Alterações em logging, correlation ID, compose, CI ou runtime exigem atualização de README e `docs/commands.md`

## Common hurdles

### Docker daemon indisponível

- Sintoma: `make up` ou `testcontainers-go` falham com `Cannot connect to the Docker daemon`
- Solução: iniciar Docker Desktop ou garantir acesso ao socket do daemon

### Falta de correlation ID

- Sintoma: request não é rastreável entre logs
- Solução: garantir middleware na borda HTTP e enrichment em logs

### Vazamento de segredo em log

- Sintoma: senha, token ou payload sensível aparece em saída estruturada
- Solução: remover o campo e revisar política de logging antes do merge
