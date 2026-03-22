# ADR 0004 - Phase 4 Resilience and Maturity

## Status

Accepted

## Context

O fluxo event-driven da Phase 3 removeu acoplamentos diretos relevantes, mas ainda deixava lacunas para operacao mais segura:

- duplicacao do mesmo `BillingRequested` podia reexecutar pagamento
- retry manual dependia apenas de status, sem controle explicito de tentativa
- timeout ou erro tecnico de gateway ainda precisava virar tentativa auditavel
- nao existia persistencia de eventos emitidos para preparar consistencia eventual

## Decision

Adotar as seguintes decisoes para a Phase 4:

1. `billing` passa a controlar `attempt_number` por invoice.
2. `payments` passa a persistir `attempt_number`, `idempotency_key` e `failure_category`.
3. a tentativa e reservada em `Pending` antes da chamada ao gateway, garantindo idempotencia por `(billing_id, attempt_number)`.
4. timeout e erro tecnico de gateway passam a resultar em `PaymentFailed` persistido, sem quebrar a criacao da invoice nem o retry manual.
5. o event bus continua sincronico, mas agora tambem registra cada evento em `outbox_events`.

## Consequences

### Positive

- reprocessar o mesmo evento financeiro nao gera nova chamada ao gateway
- retry manual passa a ser previsivel e auditavel
- falha tecnica externa nao remove rastreabilidade da tentativa
- o repositorio ganha base concreta para ativar outbox assincrono no futuro

### Negative

- o fluxo continua em um unico processo e o outbox ainda nao possui dispatcher assincrono
- a quantidade de metadados persistidos e logs aumenta
- a modelagem de `billing` e `payments` fica mais rica e exige mais cobertura de testes

## Notes

- esta ADR nao introduz Kafka, SQS, microservices, CQRS nem backoff exponencial
- a publicacao assincrona do outbox fica como passo futuro, nao como parte da Phase 4
