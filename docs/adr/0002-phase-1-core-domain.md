# ADR 0002: Core Domain da Fase 1

- Status: Accepted
- Date: 2026-03-21

## Context

O projeto ja possuia foundation operacional, mas ainda sem fluxo funcional real de negocio. A primeira fase de dominio precisava validar os limites do modular monolith sem introduzir eventos internos, mensageria externa ou integracao real com gateway.

## Decision

- Confirmar **Modular Monolith** como estilo arquitetural base da aplicacao.
- Confirmar **DDD** como abordagem de modelagem do core domain.
- Confirmar **Clean Architecture** e **Ports and Adapters** como organizacao interna padrao dos modulos.
- Implementar o fluxo minimo `Create Customer -> Create Invoice -> Process Payment -> Invoice Paid`.
- Evoluir apenas `customers`, `invoices` e `payments`; `billing` permanece scaffold.
- Usar contratos sincronos explicitos entre modulos na Phase 1:
  - `CustomerExistenceChecker`
  - `InvoicePaymentPort`
- Persistir dados em PostgreSQL com ownership logico por modulo e migrations versionadas.
- Usar transaction boundary local para persistir `payments` e atualizar `invoices` no mesmo fluxo.
- Manter gateway de pagamento como adapter mockado/local com `auto-approve` em runtime.

## Consequences

- O sistema passa a ter um fluxo funcional real sem romper a simplicidade operacional do monolito.
- A colaboracao entre modulos continua explicita e revisavel.
- O projeto fica pronto para introduzir eventos internos em fase posterior sem reescrever o dominio entregue.
- A decisao por monolito modular continua sendo o baseline contra o qual futuras extracoes devem ser justificadas.
