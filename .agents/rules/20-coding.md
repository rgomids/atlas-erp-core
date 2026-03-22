# Rule: Coding

## Objetivo

Guiar implementação e refatoração sem degradar o runtime, a legibilidade e as fronteiras de domínio.

## Stack vigente

- Go
- `chi` na borda HTTP
- `slog` para logging estruturado
- PostgreSQL com `pgx/v5`
- `golang-migrate`
- Docker Compose
- GitHub Actions

## Regras de implementação

- Priorizar código simples, explícito e orientado a comportamento.
- Toda regra de negócio deve viver em `domain` ou `application`, nunca em adapters.
- `internal/shared` deve conter apenas capacidades transversais estáveis.
- Evitar abstração antecipada.
- Não criar integração externa real antes de isolar porta e adapter.
- Preferir nomes de negócio a nomes genéricos como `service`, `manager` ou `helper`, quando o contexto permitir precisão maior.

## Handlers e borda HTTP

- handler faz parsing, delegação, serialização e mapeamento de erro
- handler não decide regra de negócio
- correlation ID deve seguir preservado na borda
- healthcheck deve continuar simples e estável

## Persistência e migrations

- mudanças de schema exigem migration versionada
- migration deve acompanhar a mudança de código no mesmo change set
- repositório implementa acesso a dados, não política de negócio
- não usar acesso direto cruzado entre tabelas de módulos sem contrato público

## Tratamento de erro

- retornar erros com contexto suficiente para diagnóstico
- não vazar segredo, token, senha ou payload sensível em erro/log
- manter consistência no mapeamento entre erro de domínio, aplicação e HTTP

## Logging

- usar logs estruturados
- mensagens curtas, objetivas e sem emoji
- evitar log redundante de sucesso em excesso
- nunca logar segredos

## Qualidade de mudança

Antes de considerar uma implementação pronta, verificar:

- fronteiras preservadas
- nomes aderentes ao domínio
- acoplamento não aumentou
- migration/documentação/testes foram tratados
