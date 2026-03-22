# Atlas ERP Core Architecture

## C1 - Context

```mermaid
C4Context
    title System Context Diagram for Atlas ERP Core

    Person(admin, "Administrador", "Usuario interno responsavel pela operacao financeira e administrativa")
    System(atlas, "Atlas ERP Core", "ERP em monolito modular orientado a eventos internos")
    System_Ext(payment_gateway, "Mock Payment Gateway", "Gateway local usado nas fases atuais")
    System_Ext(ops, "Operacao Local", "Jaeger UI e Prometheus para troubleshooting")

    Rel(admin, atlas, "Gerencia clientes, emite invoices e acompanha pagamentos", "HTTPS")
    Rel(atlas, payment_gateway, "Solicita processamento de pagamento", "Port/Adapter")
    Rel(admin, ops, "Inspeciona traces e metricas", "Browser")
    Rel(atlas, ops, "Exporta traces OTLP e expoe metricas Prometheus", "HTTP")
```

## C2 - Containers

```mermaid
C4Container
    title Container Diagram for Atlas ERP Core

    Person(admin, "Administrador", "Usuario do sistema")
    System_Ext(payment_gateway, "Mock Payment Gateway", "Adapter local")

    System_Boundary(atlas, "Atlas ERP Core") {
        Container(web_api, "Web API", "Go + chi", "Expoe endpoints HTTP, padroniza erros, aceita X-Correlation-ID e traceparent, e publica metricas em /metrics")
        Container(app_core, "Application Core", "Go", "Contem modulos de dominio, use cases, event bus sincronico, tracing OpenTelemetry e logs estruturados")
        ContainerDb(main_db, "PostgreSQL", "Relational Database", "Armazena customers, invoices, billings, payments e outbox_events")
        Container(jaeger, "Jaeger All-in-One", "Jaeger", "Recebe traces OTLP/HTTP e oferece UI local")
        Container(prometheus, "Prometheus", "Prometheus", "Coleta metricas da aplicacao em /metrics")
    }

    Rel(admin, web_api, "Usa", "HTTPS")
    Rel(admin, jaeger, "Consulta traces", "Browser")
    Rel(admin, prometheus, "Consulta metricas", "Browser")
    Rel(web_api, app_core, "Invoca casos de uso")
    Rel(app_core, main_db, "Le e escreve", "SQL")
    Rel(app_core, payment_gateway, "Processa pagamentos", "Port/Adapter")
    Rel(app_core, jaeger, "Exporta traces", "OTLP/HTTP")
    Rel(prometheus, web_api, "Coleta metricas", "HTTP /metrics")
```

## C3 - Phase 5 Components

```mermaid
C4Component
    title Component Diagram for Atlas ERP Core Phase 5

    Container_Boundary(core, "Application Core") {
        Component(router, "HTTP Router", "internal/shared/http", "Registra middleware de correlation, tracing HTTP, request logging, healthcheck e /metrics")
        Component(observability_runtime, "Observability Runtime", "internal/shared/observability", "Inicializa tracer provider, meter provider, propagadores, metricas e query tracer do PostgreSQL")
        Component(event_bus, "Sync Event Bus", "internal/shared/event", "Publica eventos in-process, registra payload no outbox e cria spans de publish e consume")
        Component(customers_module, "Customers Module", "application/domain/infrastructure", "Cria, atualiza e inativa clientes; instrumenta casos de uso")
        Component(invoices_module, "Invoices Module", "application/domain/infrastructure", "Cria e lista invoices; publica InvoiceCreated e consome PaymentApproved")
        Component(billing_module, "Billing Module", "application/domain/infrastructure", "Cria cobranca por invoice, controla attempt_number e reage a PaymentApproved/PaymentFailed")
        Component(payments_module, "Payments Module", "application/domain/infrastructure", "Consome BillingRequested, processa gateway com tracing, metricas e retry idempotente")
        Component(shared_pg, "Postgres Query Tracer", "internal/shared/postgres + internal/shared/observability", "Cria spans db.query e registra latencia por operacao e tabela sanitizada")
        Component(shared_outbox, "Outbox Recorder", "internal/shared/outbox", "Registra eventos emitidos em outbox_events no mesmo contexto transacional quando existir")
        Component(shared_logs, "Structured Logs", "internal/shared/logging + correlation", "Produz logs JSON com module, request_id, trace_id, span_id, event_name, ids de dominio e error_type")
    }

    ContainerDb(main_db, "PostgreSQL", "Relational Database", "Persistencia transacional")
    Container(jaeger, "Jaeger All-in-One", "Jaeger", "UI local de traces")
    Container(prometheus, "Prometheus", "Prometheus", "Coleta metricas tecnicas")

    Rel(router, customers_module, "HTTP -> use cases")
    Rel(router, invoices_module, "HTTP -> use cases")
    Rel(router, payments_module, "HTTP -> use cases")
    Rel(router, observability_runtime, "Cria spans HTTP e expoe /metrics")
    Rel(customers_module, event_bus, "Publish CustomerCreated")
    Rel(invoices_module, event_bus, "Publish InvoiceCreated / InvoicePaid")
    Rel(event_bus, billing_module, "InvoiceCreated / PaymentApproved / PaymentFailed")
    Rel(event_bus, payments_module, "BillingRequested")
    Rel(payments_module, event_bus, "Publish PaymentApproved / PaymentFailed")
    Rel(event_bus, shared_outbox, "Record payload")
    Rel(event_bus, observability_runtime, "Record publish/consume metrics")
    Rel(shared_pg, main_db, "Exec / Query / QueryRow")
    Rel(observability_runtime, jaeger, "Exporta traces")
    Rel(prometheus, router, "Scrape /metrics")
    Rel(router, shared_logs, "Loga request")
    Rel(event_bus, shared_logs, "Loga eventos")
```

## Sequence - Automatic Event-Driven Flow With Observability

```mermaid
sequenceDiagram
    participant Admin
    participant API as Web API
    participant OTel as Observability Runtime
    participant Invoices
    participant Bus as Sync Event Bus
    participant Billing
    participant Payments
    participant DB as PostgreSQL
    participant Gateway as Mock Gateway
    participant Jaeger

    Admin->>API: POST /invoices + X-Correlation-ID or traceparent
    API->>OTel: start span "http.request POST /invoices"
    API->>Invoices: start span "application.usecase invoices.CreateInvoice"
    Invoices->>DB: span "db.query insert invoices"
    Invoices->>Bus: span "event.publish InvoiceCreated"
    Bus->>Billing: span "event.consume billing.InvoiceCreated"
    Billing->>DB: span "db.query insert billings"
    Billing->>Bus: span "event.publish BillingRequested"
    Bus->>Payments: span "event.consume payments.BillingRequested"
    Payments->>DB: span "db.query insert payments"
    Payments->>Gateway: span "integration.gateway payments.Process"
    Gateway-->>Payments: Approved or Failed
    Payments->>DB: span "db.query update payments"
    alt Approved
        Payments->>Bus: publish PaymentApproved
        Bus->>Billing: consume PaymentApproved
        Billing->>DB: update billing Approved
        Bus->>Invoices: consume PaymentApproved
        Invoices->>DB: update invoice Paid
    else Failed
        Payments->>Bus: publish PaymentFailed
        Bus->>Billing: consume PaymentFailed
        Billing->>DB: update billing Failed
    end
    OTel->>Jaeger: exporta spans
    API-->>Admin: 201 + invoice payload
```

## Sequence - Metrics And Troubleshooting

```mermaid
sequenceDiagram
    participant Prom as Prometheus
    participant API as Web API
    participant Jaeger
    participant Admin

    Prom->>API: GET /metrics
    API-->>Prom: atlas_erp_http_*, atlas_erp_events_*, atlas_erp_db_*, atlas_erp_gateway_*
    Admin->>Jaeger: busca trace do request ou invoice
    Jaeger-->>Admin: arvore request -> invoice -> billing -> payment
    Admin->>API: consulta logs por request_id, trace_id, payment_id
```

## Sequence - Validation Failure

```mermaid
sequenceDiagram
    participant Admin
    participant API as Web API
    participant Logs as Structured Logs

    Admin->>API: POST /customers (missing document) + X-Correlation-ID
    API->>API: validate payload at HTTP boundary
    API->>Logs: log request with module=customers, request_id and error_type=validation_error
    API-->>Admin: 400 {error, message, request_id}
```
