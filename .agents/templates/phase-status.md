# Phase Status

## Fase corrente

- Nome: Phase 4 - Resilience & Maturity
- Status: active

## Objetivo da fase

Tornar o fluxo financeiro resiliente e previsivel com idempotencia por tentativa, retry controlado, timeout de gateway, outbox inicial e logs ricos em contexto.

## Escopo permitido

- aplicar idempotencia real em `payments`
- controlar retry em `billing` e `payments` com `attempt_number`
- persistir falhas tecnicas como tentativa auditavel
- preparar `outbox_events` sem worker assincrono
- reforcar logs, config e testes de resiliencia

## Entregaveis esperados

- duplicacao do mesmo evento financeiro nao gera nova execucao
- retry manual reutiliza `billing` e avanca `attempt_number`
- falha de gateway resulta em `PaymentFailed` persistido e invoice ainda `Pending`
- `outbox_events` registra eventos emitidos
- README, AGENTS, commands, diagrams, ADR e changelog atualizados para Phase 4

## Criterios de conclusao

- pagamentos nao duplicam por reprocessamento do mesmo evento
- fluxo automatico e retry manual toleram timeout/falha tecnica do gateway
- outbox inicial existe e esta coberto por validacao
- documentacao critica reflete a arquitetura e a operacao da Phase 4

## Restricoes

- nao introduzir Kafka, SQS ou qualquer mensageria externa
- nao ativar worker ou dispatch assincrono do outbox nesta fase
- nao migrar para microservices
- nao implementar backoff exponencial complexo ou scheduler dedicado

## Riscos aceitos

- `InvoiceCreated` continua sendo publicado apos persistencia local da invoice
- o outbox ainda nao possui dispatcher assincrono
- tracing e metricas de runtime aprofundadas ficam para a fase seguinte

## Proximos marcos

- ativar processamento assincrono do outbox quando houver pressao real
- aprofundar observabilidade com metricas e tracing
- preparar contratos para consistencia eventual entre modulos
