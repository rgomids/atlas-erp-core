# Skill: Modular Monolith + DDD

## Objetivo

Definir como o projeto deve evoluir do scaffold da Phase 0 para módulos de domínio reais, sem erosionar fronteiras internas.

## Princípios obrigatórios

1. O sistema é um **modular monolith**: um deploy, múltiplos módulos isolados.
2. O domínio deve ser modelado com linguagem de negócio explícita.
3. Dependências sempre apontam para dentro:

```text
interface -> application -> domain
```

4. Um módulo não pode acessar implementação interna de outro módulo.
5. Banco compartilhado fisicamente não significa acesso livre entre módulos.

## Bounded contexts iniciais

### `customers`

- Cadastro e ciclo de vida de clientes
- Services esperados: `CreateCustomer`, `UpdateCustomerProfile`, `DeactivateCustomer`
- Jobs esperados: `RebuildCustomerProjections`, `SyncCustomerReadModel`
- Models esperados: `Customer`, `CustomerDocument`, `CustomerStatus`, `CustomerCreated`

### `billing`

- Cobrança, cálculo de valores e políticas de vencimento
- Services esperados: `GenerateCharge`, `ApplyBillingPolicy`, `CloseBillingCycle`
- Jobs esperados: `CloseOverdueCharges`, `RecalculateBillingCycle`
- Models esperados: `Charge`, `BillingPolicy`, `BillingCycle`, `ChargeGenerated`

### `invoices`

- Emissão e consolidação de invoices
- Services esperados: `GenerateInvoice`, `IssueInvoice`, `CancelInvoice`
- Jobs esperados: `ReconcileInvoices`, `RetryInvoiceDispatch`
- Models esperados: `Invoice`, `InvoiceLine`, `InvoiceStatus`, `InvoiceGenerated`

### `payments`

- Processamento e estorno de pagamentos
- Services esperados: `ProcessPayment`, `ConfirmPayment`, `RefundPayment`
- Jobs esperados: `RetryPaymentSettlement`, `ExpirePendingPayments`
- Models esperados: `Payment`, `PaymentAttempt`, `PaymentStatus`, `PaymentProcessed`

## Estrutura padrão de módulo

```text
internal/<module>/
├── domain/
├── application/
│   ├── usecase/
│   └── dto/
├── infrastructure/
│   ├── repository/
│   ├── http/
│   └── persistence/
└── module.go
```

## Regras de dependência

### Permitido

- `infrastructure` depende de `application` e `domain`
- `application` depende de `domain`
- `domain` não depende de infraestrutura
- módulos interagem por contratos explícitos ou eventos

### Proibido

- handler com regra de negócio
- import direto de `infrastructure` de outro módulo
- leitura/escrita em tabela de outro módulo sem contrato explícito
- modelos compartilhados mutáveis entre domínios

## Comunicação entre módulos

### Preferência

- eventos internos in-process

### Exceção

- chamada síncrona apenas via interface pública de borda

### Eventos de referência

- `InvoiceCreated`
- `BillingRequested`
- `PaymentApproved`
- `PaymentFailed`
- `InvoicePaid`

## Regra de ouro

> Se um módulo depender da implementação interna de outro módulo, a arquitetura está quebrada.
