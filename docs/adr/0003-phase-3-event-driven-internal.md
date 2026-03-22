# ADR 0003 - Phase 3 Event-Driven Internal

## Status

Accepted

## Context

Na Phase 2, o fluxo principal ainda dependia de acoplamentos sincronos diretos entre modulos, com `payments` consumindo um port de `invoices` para validar e liquidar a invoice. Esse desenho funcionava para o primeiro fluxo ponta a ponta, mas deixava o caminho financeiro mais acoplado do que o desejado para a evolucao do monolito modular.

Ao mesmo tempo, `billing` existia apenas como scaffold, o que mantinha uma lacuna entre a arquitetura documentada e a implementacao real.

## Decision

Adotar um event bus interno sincronico e in-process como mecanismo padrao de comunicacao entre modulos no fluxo financeiro da Phase 3.

As decisoes concretas desta fase sao:

- introduzir `internal/shared/event` com `SyncBus`
- publicar `InvoiceCreated` apos persistir a invoice
- fazer `billing` consumir `InvoiceCreated` e publicar `BillingRequested`
- fazer `payments` consumir `BillingRequested` e publicar `PaymentApproved` ou `PaymentFailed`
- fazer `invoices` consumir `PaymentApproved` para marcar a invoice como `Paid`
- manter `POST /payments` como caminho manual de compatibilidade e retry funcional
- permitir multiplas tentativas `Failed`, mantendo apenas um `Approved` por invoice

## Consequences

### Positive

- reduz o acoplamento direto entre modulos
- ativa `billing` como parte real do dominio
- melhora a rastreabilidade do fluxo com logs estruturados por evento
- prepara o sistema para evolucoes futuras como outbox ou mensageria externa

### Negative

- `InvoiceCreated` continua publicado fora da transacao de criacao da invoice
- falhas tecnicas downstream ainda podem gerar resposta de erro com invoice ja persistida
- a simplicidade do bus sincronico nao cobre resiliencia avancada, replay ou persistencia de eventos

## Alternatives Considered

### Manter orquestracao sincrona direta

Rejeitado porque perpetua o acoplamento entre modulos e posterga a ativacao real de `billing`.

### Introduzir mensageria externa agora

Rejeitado porque adicionaria complexidade operacional fora do objetivo desta fase.

### Implementar outbox pattern junto com o bus interno

Rejeitado porque a fase atual prioriza clareza de fluxo e baixo acoplamento, nao resiliencia de producao avancada.
