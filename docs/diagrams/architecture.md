# Atlas ERP Core Architecture

## C1 - Context

```mermaid
C4Context
    title System Context Diagram for Atlas ERP Core

    Person(admin, "Administrador", "Usuario interno que opera clientes, invoices e pagamentos")
    System(atlas, "Atlas ERP Core", "ERP backend em Go, modular monolith, orientado a eventos internos")
    System_Ext(payment_gateway, "Mock Payment Gateway", "Adapter local para processamento de pagamento")
    System_Ext(ops, "Jaeger + Prometheus", "Stack local de observabilidade")

    Rel(admin, atlas, "Gerencia clientes, invoices e retries", "HTTPS")
    Rel(atlas, payment_gateway, "Processa pagamentos", "Port/Adapter")
    Rel(admin, ops, "Inspeciona traces e metricas", "Browser")
    Rel(atlas, ops, "Exporta traces e expoe metricas", "OTLP/HTTP + /metrics")
```

## C2 - Containers

```mermaid
C4Container
    title Container Diagram for Atlas ERP Core

    Person(admin, "Administrador", "Usuario do sistema")
    System_Ext(payment_gateway, "Mock Payment Gateway", "Gateway local")

    System_Boundary(atlas, "Atlas ERP Core") {
        Container(web_api, "Web API", "Go + chi", "Expõe endpoints, preserva correlation id, mapeia erros, publica /metrics")
        Container(app_core, "Application Core", "Go", "Módulos de domínio, use cases, event bus síncrono, outbox, tracing e logs")
        ContainerDb(main_db, "PostgreSQL", "Relational Database", "customers, invoices, billings, payments, outbox_events")
        Container(jaeger, "Jaeger", "Jaeger all-in-one", "UI local de traces")
        Container(prometheus, "Prometheus", "Prometheus", "Coleta métricas técnicas")
    }

    Rel(admin, web_api, "Usa", "HTTPS")
    Rel(web_api, app_core, "Invoca casos de uso")
    Rel(app_core, main_db, "Lê e escreve", "SQL")
    Rel(app_core, payment_gateway, "Processa pagamento", "Port/Adapter")
    Rel(app_core, jaeger, "Exporta traces", "OTLP/HTTP")
    Rel(prometheus, web_api, "Scrape", "HTTP /metrics")
```

## C3 - Phase 7 Core Components

```mermaid
C4Component
    title Phase 7 Core Component Diagram

    Container_Boundary(core, "Application Core") {
        Component(router, "HTTP Router", "internal/shared/http", "Borda HTTP, correlation, tracing, request logging, health e metrics")
        Component(config_loader, "Config Loader", "internal/shared/config", "Carrega .env, overlays e ATLAS_FAULT_PROFILE")
        Component(event_bus, "Sync Event Bus", "internal/shared/event", "Publica eventos, aciona handlers, registra lifecycle do outbox")
        Component(runtime_faults, "Runtime Fault Decorators", "internal/shared/runtimefaults", "Perfis locais de timeout, falha, duplicidade e falha de outbox")
        Component(outbox_recorder, "Outbox Recorder", "internal/shared/outbox", "Append e status pending/processed/failed")
        Component(observability_runtime, "Observability Runtime", "internal/shared/observability", "Spans, métricas, query tracing e /metrics")
        Component(customers_module, "Customers Module", "customers", "Cadastro e atualização de clientes")
        Component(invoices_module, "Invoices Module", "invoices", "Emissão e listagem de invoices")
        Component(billing_module, "Billing Module", "billing", "Geração e reativação de cobranças com attempt_number")
        Component(payments_module, "Payments Module", "payments", "Processamento de gateway, idempotência e retry")
    }

    ContainerDb(main_db, "PostgreSQL", "Relational Database", "Persistência transacional")

    Rel(router, customers_module, "HTTP")
    Rel(router, invoices_module, "HTTP")
    Rel(router, payments_module, "HTTP")
    Rel(config_loader, runtime_faults, "Resolve perfil")
    Rel(runtime_faults, event_bus, "Configura hooks de entrega")
    Rel(runtime_faults, outbox_recorder, "Configura falha de append")
    Rel(runtime_faults, payments_module, "Decora gateway")
    Rel(customers_module, event_bus, "Publish CustomerCreated")
    Rel(invoices_module, event_bus, "Publish InvoiceCreated / InvoicePaid")
    Rel(event_bus, billing_module, "Consume InvoiceCreated / PaymentApproved / PaymentFailed")
    Rel(billing_module, event_bus, "Publish BillingRequested")
    Rel(event_bus, payments_module, "Consume BillingRequested")
    Rel(payments_module, event_bus, "Publish PaymentApproved / PaymentFailed")
    Rel(event_bus, outbox_recorder, "Append + status")
    Rel(customers_module, main_db, "SQL")
    Rel(invoices_module, main_db, "SQL")
    Rel(billing_module, main_db, "SQL")
    Rel(payments_module, main_db, "SQL")
```

## C4 - Customers Module

```mermaid
C4Component
    title Customers Module

    Container_Boundary(customers, "Customers") {
        Component(customers_http, "HTTP Handler", "infrastructure/http", "Valida payload e serializa respostas")
        Component(customers_usecases, "Use Cases", "application/usecases", "CreateCustomer, UpdateCustomer, DeactivateCustomer")
        Component(customers_domain, "Customer Aggregate", "domain/entities + valueobjects", "Regras de cadastro, documento e email")
        Component(customers_repo, "Postgres Repository", "infrastructure/persistence", "Persistência de customers")
        Component(customers_events, "Public Events", "public/events", "CustomerCreated")
    }

    ContainerDb(main_db, "PostgreSQL", "Relational Database", "customers")

    Rel(customers_http, customers_usecases, "Invoca")
    Rel(customers_usecases, customers_domain, "Aplica regras")
    Rel(customers_usecases, customers_repo, "Usa")
    Rel(customers_usecases, customers_events, "Publica")
    Rel(customers_repo, main_db, "SQL")
```

## C4 - Invoices Module

```mermaid
C4Component
    title Invoices Module

    Container_Boundary(invoices, "Invoices") {
        Component(invoices_http, "HTTP Handler", "infrastructure/http", "POST /invoices e GET /customers/{id}/invoices")
        Component(invoices_usecases, "Use Cases", "application/usecases", "CreateInvoice, ListCustomerInvoices, ApplyPaymentApproved")
        Component(invoices_domain, "Invoice Aggregate", "domain/entities", "Valor, vencimento e transição para Paid")
        Component(invoices_repo, "Postgres Repository", "infrastructure/persistence", "Persistência de invoices")
        Component(customers_port, "Customer Existence Checker", "customers/public", "Contrato público para validar customer ativo")
        Component(invoices_events, "Public Events", "public/events", "InvoiceCreated, InvoicePaid")
    }

    ContainerDb(main_db, "PostgreSQL", "Relational Database", "invoices")

    Rel(invoices_http, invoices_usecases, "Invoca")
    Rel(invoices_usecases, invoices_domain, "Aplica regras")
    Rel(invoices_usecases, customers_port, "Verifica")
    Rel(invoices_usecases, invoices_repo, "Usa")
    Rel(invoices_usecases, invoices_events, "Publica")
    Rel(invoices_repo, main_db, "SQL")
```

## C4 - Billing Module

```mermaid
C4Component
    title Billing Module

    Container_Boundary(billing, "Billing") {
        Component(billing_handlers, "Event Handlers", "application/handlers", "Consume InvoiceCreated, PaymentApproved e PaymentFailed")
        Component(billing_usecases, "Use Cases", "application/usecases", "CreateBillingFromInvoice, GetProcessableBillingByInvoiceID, MarkBillingApproved, MarkBillingFailed")
        Component(billing_domain, "Billing Aggregate", "domain/entities", "Ownership de cobrança e attempt_number")
        Component(billing_repo, "Postgres Repository", "infrastructure/persistence", "Persistência de billings")
        Component(billing_public, "Public Port", "public", "PaymentCompatibilityPort e BillingSnapshot")
        Component(billing_events, "Public Events", "public/events", "BillingRequested")
    }

    ContainerDb(main_db, "PostgreSQL", "Relational Database", "billings")

    Rel(billing_handlers, billing_usecases, "Invoca")
    Rel(billing_usecases, billing_domain, "Aplica regras")
    Rel(billing_usecases, billing_repo, "Usa")
    Rel(billing_usecases, billing_events, "Publica")
    Rel(billing_public, billing_usecases, "Expõe retry manual")
    Rel(billing_repo, main_db, "SQL")
```

## C4 - Payments Module

```mermaid
C4Component
    title Payments Module

    Container_Boundary(payments, "Payments") {
        Component(payments_http, "HTTP Handler", "infrastructure/http", "POST /payments")
        Component(payments_handlers, "Event Handler", "application/handlers", "Consume BillingRequested")
        Component(payments_usecases, "Use Cases", "application/usecases", "ProcessBillingRequest, ProcessPayment")
        Component(payments_domain, "Payment Aggregate", "domain/entities", "Idempotency, approval and failure categories")
        Component(payments_repo, "Postgres Repository", "infrastructure/persistence", "Persistência de payments")
        Component(billing_public, "Billing Public Port", "billing/public", "Busca billing processável para retry manual")
        Component(payment_gateway, "Payment Gateway Port + Adapter", "application/ports + infrastructure/integration", "Gateway mock e decoradores de falha")
        Component(payments_events, "Public Events", "public/events", "PaymentApproved, PaymentFailed")
    }

    ContainerDb(main_db, "PostgreSQL", "Relational Database", "payments")

    Rel(payments_http, payments_usecases, "Invoca")
    Rel(payments_handlers, payments_usecases, "Invoca")
    Rel(payments_usecases, payments_domain, "Aplica regras")
    Rel(payments_usecases, payments_repo, "Usa")
    Rel(payments_usecases, billing_public, "Consulta retry")
    Rel(payments_usecases, payment_gateway, "Processa")
    Rel(payments_usecases, payments_events, "Publica")
    Rel(payments_repo, main_db, "SQL")
```

## Sequence - Create Customer

```mermaid
sequenceDiagram
    participant Admin
    participant API as Web API
    participant Customers
    participant DB as PostgreSQL
    participant Bus as Sync Event Bus
    participant Outbox

    Admin->>API: POST /customers
    API->>Customers: CreateCustomer
    Customers->>DB: insert customer
    Customers->>Bus: publish CustomerCreated
    Bus->>Outbox: append pending
    Bus->>Outbox: mark processed
    API-->>Admin: 201 Created
```

## Sequence - Create Invoice

```mermaid
sequenceDiagram
    participant Admin
    participant API as Web API
    participant Invoices
    participant Customers as customers/public
    participant DB as PostgreSQL
    participant Bus as Sync Event Bus
    participant Billing

    Admin->>API: POST /invoices
    API->>Invoices: CreateInvoice
    Invoices->>Customers: ExistsActiveCustomer
    Invoices->>DB: insert invoice Pending
    Invoices->>Bus: publish InvoiceCreated
    Bus->>Billing: consume InvoiceCreated
    Billing->>DB: insert billing Requested
    Billing->>Bus: publish BillingRequested
    API-->>Admin: 201 Created
```

## Sequence - Process Payment Approved

```mermaid
sequenceDiagram
    participant Bus as Sync Event Bus
    participant Payments
    participant Gateway as Mock Gateway
    participant DB as PostgreSQL
    participant Billing
    participant Invoices

    Bus->>Payments: consume BillingRequested
    Payments->>DB: reserve payment attempt Pending
    Payments->>Gateway: Process
    Gateway-->>Payments: Approved
    Payments->>DB: update payment Approved
    Payments->>Bus: publish PaymentApproved
    Bus->>Billing: consume PaymentApproved
    Billing->>DB: update billing Approved
    Bus->>Invoices: consume PaymentApproved
    Invoices->>DB: update invoice Paid
```

## Sequence - Process Payment Failed And Manual Retry

```mermaid
sequenceDiagram
    participant Bus as Sync Event Bus
    participant Payments
    participant Gateway as Mock Gateway
    participant DB as PostgreSQL
    participant Billing
    participant API as Web API

    Bus->>Payments: consume BillingRequested
    Payments->>DB: reserve payment attempt Pending
    Payments->>Gateway: Process
    Gateway-->>Payments: timeout or technical failure
    Payments->>DB: update payment Failed
    Payments->>Bus: publish PaymentFailed
    Bus->>Billing: consume PaymentFailed
    Billing->>DB: update billing Failed
    API->>Payments: POST /payments
    Payments->>Billing: GetProcessableBillingByInvoiceID
    Billing->>DB: reactivate billing and increment attempt_number
    Payments->>Gateway: Process retry
    Gateway-->>Payments: Approved
    Payments->>DB: update payment Approved
```

## Sequence - Internal Event Flow

```mermaid
sequenceDiagram
    participant Invoices
    participant Bus as Sync Event Bus
    participant Outbox
    participant Billing
    participant Payments

    Invoices->>Bus: InvoiceCreated
    Bus->>Outbox: append pending
    Bus->>Billing: consume InvoiceCreated
    Billing->>Bus: BillingRequested
    Bus->>Outbox: append pending
    Bus->>Payments: consume BillingRequested
    Payments->>Bus: PaymentApproved or PaymentFailed
    Bus->>Outbox: mark processed or failed
```

## Sequence - Persistence And Outbox Lifecycle

```mermaid
sequenceDiagram
    participant UseCase as Application Use Case
    participant DB as PostgreSQL
    participant Bus as Sync Event Bus
    participant Outbox
    participant Handler as Event Handler

    UseCase->>DB: persist aggregate
    UseCase->>Bus: publish domain fact
    Bus->>Outbox: append pending
    alt handler succeeds
        Bus->>Handler: dispatch
        Handler-->>Bus: ok
        Bus->>Outbox: mark processed
    else handler fails
        Bus->>Handler: dispatch
        Handler-->>Bus: error
        Bus->>Outbox: mark failed
    end
```

## Flowchart - Event Dependencies

```mermaid
flowchart LR
    customers["CustomerCreated"]
    invoice_created["InvoiceCreated"]
    billing_requested["BillingRequested"]
    payment_approved["PaymentApproved"]
    payment_failed["PaymentFailed"]
    invoice_paid["InvoicePaid"]

    customers -->|"traceability only"| invoice_created
    invoice_created --> billing_requested
    billing_requested --> payment_approved
    billing_requested --> payment_failed
    payment_approved --> invoice_paid
```
