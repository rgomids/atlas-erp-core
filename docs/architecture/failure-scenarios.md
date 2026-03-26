# Failure Scenarios

Phase 7 adds controlled failure profiles to demonstrate predictable system behavior under technical stress without changing HTTP contracts or business rules.

## Scenario Matrix

| Profile | Purpose | Injection point | Expected outcome |
| --- | --- | --- | --- |
| `payment_timeout` | Simulate a gateway timeout | payment gateway decorator | payment attempt becomes `Failed`, `failure_category=gateway_timeout`, invoice stays `Pending` |
| `payment_flaky_first` | Simulate an intermittent gateway failure | payment gateway decorator | first automatic payment fails with `gateway_error`, manual retry can approve a second attempt |
| `duplicate_billing_requested` | Simulate duplicated event delivery | event bus duplicate delivery hook | only one payment is approved for the same `(billing_id, attempt_number)` |
| `event_consumer_failure` | Simulate a failing internal consumer | event bus consumer failure hook | `BillingRequested` outbox record becomes `failed`, downstream payment is not approved |
| `outbox_append_failure` | Simulate failure before consumers are reached | outbox recorder decorator | upstream aggregate stays persisted, outbox append fails, downstream side effects do not run |

## How To Run

Run the API locally with the desired profile:

```bash
ATLAS_FAULT_PROFILE=<profile> OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/api
```

Examples:

```bash
ATLAS_FAULT_PROFILE=payment_timeout OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/api
ATLAS_FAULT_PROFILE=duplicate_billing_requested OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/api
```

## Expected Behavior By Scenario

### `payment_timeout`

- automatic invoice creation still returns `201`
- billing is created
- payment attempt is persisted as `Failed`
- invoice remains `Pending`
- traces show `integration.gateway payments.Process`
- payment payload or logs include `failure_category=gateway_timeout`

### `payment_flaky_first`

- first gateway call fails technically
- billing becomes retryable
- `POST /payments` creates a second attempt
- second attempt can become `Approved`
- a repeated `POST /payments` after approval returns conflict

### `duplicate_billing_requested`

- the first `BillingRequested` delivery is duplicated once
- the duplicate does not create a second approved payment
- idempotency is preserved by `(billing_id, attempt_number)` and `idempotency_key`

### `event_consumer_failure`

- `BillingRequested` fails before `payments` completes consumption
- `outbox_events.status` for `BillingRequested` becomes `failed`
- no `PaymentApproved` is emitted
- invoice remains `Pending`
- billing remains `Requested`

### `outbox_append_failure`

- the upstream aggregate save already happened
- event publication stops before consumer execution
- no outbox record is appended for that failing publish
- no downstream billing or payment side effect occurs

This scenario is intentionally useful for explaining the main synchronous trade-off of the current design.

## Validation Signals

Use a combination of:

- HTTP responses
- `outbox_events`
- `payments` and `billings` tables
- Jaeger traces
- Prometheus metrics
- structured logs with `request_id`, `event_name`, `attempt_number`, and `failure_category`
