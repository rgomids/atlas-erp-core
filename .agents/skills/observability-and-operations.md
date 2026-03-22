# Skill: Observability and Operations

## Quando usar

Use esta skill ao alterar:

- logging
- correlation ID
- healthcheck
- runtime local
- compose
- CI
- comandos operacionais

## Contexto mínimo a carregar

- `.agents/rules/20-coding.md`
- `.agents/rules/40-documentation.md`
- `.agents/rules/50-security.md`

## Convenções obrigatórias

- logs estruturados em JSON
- correlation ID preservado na borda HTTP
- mensagens curtas, textuais e sem emoji
- nenhum segredo em logs
- preferir `make <target>` a comandos longos

## Runtime local esperado

- `make up` sobe `app` e `postgres`, quando esse for o setup oficial
- `GET /health` responde `{"status":"ok"}`
- banco está acessível ao bootstrap, conforme contrato atual do sistema

## Hurdles comuns

### Docker indisponível

- sintoma: compose ou `testcontainers-go` falha
- ação: validar daemon/socket antes de mexer no código

### Correlation ID ausente

- sintoma: request sem rastreabilidade
- ação: revisar middleware e enriquecimento de log

### Segredo em log

- sintoma: token, senha ou payload sensível apareceu
- ação: remover imediatamente e revisar política de logging
