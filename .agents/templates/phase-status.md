# Phase Status

## Fase corrente

- Nome: Phase 3 - Event-Driven Internal
- Status: active

## Objetivo da fase

Reduzir acoplamento entre modulos com eventos internos in-process, ativar `billing` no fluxo principal e preparar o sistema para resiliencia e evolucao distribuida futura.

## Escopo permitido

- introduzir event bus interno sincronico
- substituir orquestracao direta entre modulos por eventos internos
- ativar `billing` com persistencia e handlers
- permitir retry manual apos `PaymentFailed`
- reforcar observabilidade por evento e documentacao da nova fase

## Entregaveis esperados

- fluxo principal disparado por `POST /invoices` e fechado por eventos internos
- `billing` persistido e integrado ao ciclo financeiro
- pagamentos com multiplas tentativas e unicidade apenas para `Approved`
- logs com `event`, `emitter_module`, `consumer_module` e `request_id`
- README, AGENTS, commands, diagrams, ADR e changelog atualizados

## Criterios de conclusao

- event bus interno funcional e coberto por testes
- fluxo automatico e retry manual estao verdes ponta a ponta
- billing deixa de ser scaffold e participa do runtime oficial
- documentacao critica reflete a arquitetura da Phase 3

## Restricoes

- nao introduzir Kafka, SQS ou qualquer mensageria externa
- nao implementar outbox pattern nesta fase
- nao migrar para microservices
- nao adicionar goroutines complexas ou persistencia de eventos

## Riscos aceitos

- `InvoiceCreated` e publicado fora da transacao de criacao da invoice
- falha tecnica downstream ainda pode devolver erro HTTP com invoice ja persistida
- retry automatico e resiliencia avancada continuam fora da fase

## Proximos marcos

- avaliar outbox e retry tecnico para eventos internos
- aprofundar observabilidade com metricas e tracing
- preparar criterios e contratos para possivel extracao de modulos
