# Atlas ERP Core Architecture

## C1 - Context

```mermaid
C4Context
    title System Context Diagram for Atlas ERP Core

    Person(admin, "Administrador", "Usuario interno responsavel pela operacao financeira e administrativa")
    System(atlas, "Atlas ERP Core", "ERP em monolito modular orientado a eventos internos")
    System_Ext(payment_gateway, "Mock Payment Gateway", "Gateway local usado nas fases atuais")

    Rel(admin, atlas, "Gerencia clientes, emite invoices e acompanha pagamentos", "HTTPS")
    Rel(atlas, payment_gateway, "Solicita processamento de pagamento", "In-process adapter")
```

## C2 - Containers

```mermaid
C4Container
    title Container Diagram for Atlas ERP Core

    Person(admin, "Administrador", "Usuario do sistema")
    System_Ext(payment_gateway, "Mock Payment Gateway", "Adapter local")

    System_Boundary(atlas, "Atlas ERP Core") {
        Container(web_api, "Web API", "Go + chi", "Expoe endpoints HTTP, valida payload, propaga request_id e padroniza respostas de erro")
        Container(app_core, "Application Core", "Go", "Contem modulos de dominio, use cases, handlers de evento e um event bus sincronico in-process")
        ContainerDb(main_db, "PostgreSQL", "Relational Database", "Armazena customers, invoices, billings e payments")
    }

    Rel(admin, web_api, "Usa", "HTTPS")
    Rel(web_api, app_core, "Invoca casos de uso")
    Rel(app_core, main_db, "Le e escreve", "SQL")
    Rel(app_core, payment_gateway, "Processa pagamentos", "Port/Adapter")
```

## C3 - Phase 4 Components

```mermaid
C4Component
    title Component Diagram for Atlas ERP Core Phase 4

    Container_Boundary(core, "Application Core") {
        Component(router, "HTTP Router", "internal/shared/http", "Registra middleware, validacao de borda, correlation/request_id, error contract e rotas")
        Component(event_bus, "Sync Event Bus", "internal/shared/event", "Publica eventos in-process, registra payload no outbox e loga emitter/consumer com contexto de dominio")
        Component(customers_module, "Customers Module", "application/domain/infrastructure", "Cria, atualiza e inativa clientes; publica CustomerCreated")
        Component(invoices_module, "Invoices Module", "application/domain/infrastructure", "Cria e lista invoices; publica InvoiceCreated e consome PaymentApproved")
        Component(billing_module, "Billing Module", "application/domain/infrastructure", "Cria cobranca por invoice, controla attempt_number e prepara retry seguro")
        Component(payments_module, "Payments Module", "application/domain/infrastructure", "Consome BillingRequested, reserva tentativa idempotente, aplica timeout de gateway e publica PaymentApproved ou PaymentFailed")
        Component(shared_pg, "Postgres Tx Context", "internal/shared/postgres", "Coordena transacoes locais para handlers executados dentro do publish")
        Component(shared_outbox, "Outbox Recorder", "internal/shared/outbox", "Registra eventos emitidos em outbox_events no mesmo contexto transacional quando existir")
        Component(shared_obs, "Structured Logging", "internal/shared/logging + correlation", "Produz logs JSON com module, event, emitter_module, consumer_module, ids de dominio e request_id")
    }

    ContainerDb(main_db, "PostgreSQL", "Relational Database", "Persistencia transacional")

    Rel(router, customers_module, "HTTP -> use cases")
    Rel(router, invoices_module, "HTTP -> use cases")
    Rel(router, payments_module, "HTTP -> use cases")
    Rel(customers_module, event_bus, "Publish CustomerCreated")
    Rel(invoices_module, event_bus, "Publish InvoiceCreated / InvoicePaid")
    Rel(event_bus, billing_module, "InvoiceCreated / PaymentApproved / PaymentFailed")
    Rel(event_bus, payments_module, "BillingRequested")
    Rel(payments_module, event_bus, "Publish PaymentApproved / PaymentFailed")
    Rel(event_bus, shared_outbox, "Record payload")
    Rel(event_bus, invoices_module, "PaymentApproved")
    Rel(payments_module, shared_pg, "WithinTransaction")
    Rel(router, shared_obs, "Anexa module/request_id")
    Rel(event_bus, shared_obs, "Loga eventos")
    Rel(customers_module, main_db, "customers")
    Rel(invoices_module, main_db, "invoices")
    Rel(billing_module, main_db, "billings")
    Rel(payments_module, main_db, "payments")
```

## Sequence - Automatic Event-Driven Flow With Resilience

```mermaid
sequenceDiagram
    participant Admin
    participant API as Web API
    participant Invoices
    participant Bus as Sync Event Bus
    participant Billing
    participant Payments
    participant DB as PostgreSQL
    participant Gateway as Mock Gateway

    Admin->>API: POST /invoices + X-Correlation-ID
    API->>API: validate payload + bind request_id
    API->>Invoices: CreateInvoice
    Invoices->>DB: insert invoice (Pending)
    Invoices->>Bus: publish InvoiceCreated
    Bus->>Billing: handle InvoiceCreated
    Billing->>DB: insert billing (Requested)
    Billing->>Bus: publish BillingRequested
    Bus->>Payments: handle BillingRequested
    Payments->>DB: insert pending payment attempt (billing_id + attempt_number)
    Payments->>Gateway: Process with timeout
    Gateway-->>Payments: Approved or Failed
    Payments->>DB: update payment attempt + persist outbox record
    alt Approved
        Payments->>Bus: publish PaymentApproved
        Bus->>Billing: handle PaymentApproved
        Billing->>DB: update billing (Approved)
        Bus->>Invoices: handle PaymentApproved
        Invoices->>DB: update invoice (Paid)
        Invoices->>Bus: publish InvoicePaid
    else Failed
        Payments->>Bus: publish PaymentFailed
        Bus->>Billing: handle PaymentFailed
        Billing->>DB: update billing (Failed)
    end
    API-->>Admin: 201 + invoice payload
```

## Sequence - Manual Retry

```mermaid
sequenceDiagram
    participant Admin
    participant API as Web API
    participant Payments
    participant Billing
    participant Bus as Sync Event Bus
    participant DB as PostgreSQL
    participant Gateway as Mock Gateway
    participant Invoices

    Admin->>API: POST /payments + invoice_id
    API->>Payments: ProcessPayment (compat)
    Payments->>Billing: GetProcessableBillingByInvoiceID
    Billing->>DB: reactivate billing when status is Failed and advance attempt_number
    Billing-->>Payments: billing snapshot
    Payments->>DB: reserve idempotent payment attempt
    Payments->>Gateway: Process with timeout
    alt Approved
        Gateway-->>Payments: Approved
        Payments->>DB: finalize approved attempt
        Payments->>Bus: publish PaymentApproved
        Bus->>Billing: handle PaymentApproved
        Billing->>DB: update billing (Approved)
        Bus->>Invoices: handle PaymentApproved
        Invoices->>DB: update invoice (Paid)
        API-->>Admin: 201 + payment payload
    else Technical failure
        Gateway-->>Payments: timeout / error
        Payments->>DB: finalize failed attempt
        Payments->>Bus: publish PaymentFailed
        Bus->>Billing: handle PaymentFailed
        Billing->>DB: update billing (Failed)
        API-->>Admin: 201 + failed payment payload
    end
```

## Sequence - Validation Failure

```mermaid
sequenceDiagram
    participant Admin
    participant API as Web API
    participant Logs as Structured Logs

    Admin->>API: POST /customers (missing document) + X-Correlation-ID
    API->>API: validate payload at HTTP boundary
    API->>Logs: log request with module=customers and request_id
    API-->>Admin: 400 {error, message, request_id}
```
