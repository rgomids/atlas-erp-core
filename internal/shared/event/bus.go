package event

import (
	"context"
	"encoding/json"
	"log/slog"
	"reflect"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/rgomids/atlas-erp-core/internal/shared/correlation"
	httpapi "github.com/rgomids/atlas-erp-core/internal/shared/http"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
)

type Event interface {
	Name() string
}

type EventHandler interface {
	Handle(ctx context.Context, event Event) error
}

type EventBus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(eventName string, handler EventHandler)
}

type EventRecord struct {
	EventName     string
	EmitterModule string
	RequestID     string
	Payload       []byte
	OccurredAt    time.Time
}

type Recorder interface {
	Record(ctx context.Context, record EventRecord) error
}

type HandlerFunc func(ctx context.Context, event Event) error

func (handler HandlerFunc) Handle(ctx context.Context, event Event) error {
	return handler(ctx, event)
}

type SyncBus struct {
	mu            sync.RWMutex
	handlers      map[string][]EventHandler
	recorder      Recorder
	now           func() time.Time
	observability *observability.Runtime
}

type emitterModuleContextKey string

const emitterModuleKey emitterModuleContextKey = "event_emitter_module"

func NewSyncBus(recorders ...Recorder) *SyncBus {
	return NewSyncBusWithObservability(nil, recorders...)
}

func NewSyncBusWithObservability(telemetry *observability.Runtime, recorders ...Recorder) *SyncBus {
	var recorder Recorder
	if len(recorders) > 0 {
		recorder = recorders[0]
	}

	return &SyncBus{
		handlers:      map[string][]EventHandler{},
		recorder:      recorder,
		now:           time.Now,
		observability: telemetry,
	}
}

func (bus *SyncBus) Publish(ctx context.Context, domainEvent Event) error {
	bus.mu.RLock()
	handlers := append([]EventHandler(nil), bus.handlers[domainEvent.Name()]...)
	bus.mu.RUnlock()

	emitterModule := emitterModuleFromContext(ctx)
	if emitterModule == "" {
		emitterModule = "shared"
	}

	logger := httpapi.LoggerFromContext(ctx).With(
		slog.String("module", emitterModule),
		slog.String("event", domainEvent.Name()),
		slog.String("event_name", domainEvent.Name()),
		slog.String("emitter_module", emitterModule),
	)

	eventArgs := append([]any{slog.Int("handler_count", len(handlers))}, eventLogArgs(domainEvent)...)
	telemetry := observability.FromOptional(bus.observability)
	publishContext, publishSpan := telemetry.StartEventPublish(
		ctx,
		domainEvent.Name(),
		emitterModule,
		eventAttributes(domainEvent)...,
	)
	defer telemetry.CompleteSpan(publishSpan, nil, "")

	if err := bus.record(ctx, emitterModule, domainEvent); err != nil {
		logger.Error("🧾 event record failed", append([]any{slog.Any("err", err)}, eventLogArgs(domainEvent)...)...)
		telemetry.RecordSpanError(publishSpan, err, observability.ErrorTypeInfrastructure)
		return err
	}

	logger.Info("📣 event published", eventArgs...)
	telemetry.RecordEventPublished(publishContext, domainEvent.Name(), emitterModule)

	for _, handler := range handlers {
		consumerModule := handlerModule(handler)
		handlerLogger := logger.With(
			slog.String("module", consumerModule),
			slog.String("consumer_module", consumerModule),
		)
		handlerContext, handlerSpan := telemetry.StartEventConsume(
			publishContext,
			domainEvent.Name(),
			consumerModule,
			eventAttributes(domainEvent)...,
		)

		handlerLogger.Info("📥 event handling", eventLogArgs(domainEvent)...)
		telemetry.RecordEventConsumed(handlerContext, domainEvent.Name(), consumerModule)

		if err := handler.Handle(handlerContext, domainEvent); err != nil {
			handlerLogger.Error(
				"💥 event handler failed",
				append(
					[]any{
						slog.Any("err", err),
						slog.String("error_type", observability.ErrorTypeInfrastructure),
					},
					eventLogArgs(domainEvent)...,
				)...,
			)
			telemetry.RecordEventHandlerFailure(handlerContext, domainEvent.Name(), consumerModule, observability.ErrorTypeInfrastructure)
			telemetry.CompleteSpan(handlerSpan, err, observability.ErrorTypeInfrastructure)
			return err
		}

		handlerLogger.Info("✅ event handled", eventLogArgs(domainEvent)...)
		telemetry.CompleteSpan(handlerSpan, nil, "")
	}

	return nil
}

func (bus *SyncBus) Subscribe(eventName string, handler EventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.handlers[eventName] = append(bus.handlers[eventName], handler)
}

func Publish(ctx context.Context, bus EventBus, emitterModule string, domainEvent Event) error {
	if bus == nil {
		return nil
	}

	return bus.Publish(context.WithValue(ctx, emitterModuleKey, emitterModule), domainEvent)
}

func Subscribe(bus EventBus, eventName string, consumerModule string, handler EventHandler) {
	if bus == nil || handler == nil {
		return
	}

	bus.Subscribe(eventName, moduleAwareHandler{
		module:  consumerModule,
		handler: handler,
	})
}

type moduleAwareHandler struct {
	module  string
	handler EventHandler
}

func (handler moduleAwareHandler) Handle(ctx context.Context, event Event) error {
	return handler.handler.Handle(ctx, event)
}

func handlerModule(handler EventHandler) string {
	moduleHandler, ok := handler.(moduleAwareHandler)
	if ok && moduleHandler.module != "" {
		return moduleHandler.module
	}

	return "shared"
}

func emitterModuleFromContext(ctx context.Context) string {
	module, ok := ctx.Value(emitterModuleKey).(string)
	if !ok {
		return ""
	}

	return module
}

func (bus *SyncBus) record(ctx context.Context, emitterModule string, domainEvent Event) error {
	if bus.recorder == nil {
		return nil
	}

	payload, err := json.Marshal(domainEvent)
	if err != nil {
		return err
	}

	return bus.recorder.Record(ctx, EventRecord{
		EventName:     domainEvent.Name(),
		EmitterModule: emitterModule,
		RequestID:     correlation.ID(ctx),
		Payload:       payload,
		OccurredAt:    bus.now().UTC(),
	})
}

func eventLogArgs(domainEvent Event) []any {
	value := reflect.ValueOf(domainEvent)
	if !value.IsValid() {
		return nil
	}

	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}

		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return nil
	}

	args := make([]any, 0, 6)
	appendStringField(&args, value, "CustomerID", "customer_id")
	appendStringField(&args, value, "InvoiceID", "invoice_id")
	appendStringField(&args, value, "BillingID", "billing_id")
	appendStringField(&args, value, "PaymentID", "payment_id")
	appendStringField(&args, value, "IdempotencyKey", "idempotency_key")
	appendIntField(&args, value, "AttemptNumber", "attempt_number")
	appendRetryCountField(&args, value, "AttemptNumber", "retry_count")
	appendStringField(&args, value, "FailureCategory", "failure_category")

	return args
}

func eventAttributes(domainEvent Event) []attribute.KeyValue {
	value := reflect.ValueOf(domainEvent)
	if !value.IsValid() {
		return nil
	}

	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}

		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return nil
	}

	attributes := make([]attribute.KeyValue, 0, 7)
	appendStringAttribute(&attributes, value, "CustomerID", "atlas.customer_id")
	appendStringAttribute(&attributes, value, "InvoiceID", "atlas.invoice_id")
	appendStringAttribute(&attributes, value, "BillingID", "atlas.billing_id")
	appendStringAttribute(&attributes, value, "PaymentID", "atlas.payment_id")
	appendStringAttribute(&attributes, value, "IdempotencyKey", "atlas.idempotency_key")
	appendIntAttribute(&attributes, value, "AttemptNumber", "atlas.attempt_number")
	appendRetryCountAttribute(&attributes, value, "AttemptNumber", "atlas.retry_count")
	appendStringAttribute(&attributes, value, "FailureCategory", "atlas.failure_category")

	return attributes
}

func appendStringField(args *[]any, value reflect.Value, fieldName string, key string) {
	field := value.FieldByName(fieldName)
	if !field.IsValid() || field.Kind() != reflect.String || field.String() == "" {
		return
	}

	*args = append(*args, slog.String(key, field.String()))
}

func appendIntField(args *[]any, value reflect.Value, fieldName string, key string) {
	field := value.FieldByName(fieldName)
	if !field.IsValid() {
		return
	}

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Int() > 0 {
			*args = append(*args, slog.Int64(key, field.Int()))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if field.Uint() > 0 {
			*args = append(*args, slog.Uint64(key, field.Uint()))
		}
	}
}

func appendRetryCountField(args *[]any, value reflect.Value, fieldName string, key string) {
	field := value.FieldByName(fieldName)
	if !field.IsValid() {
		return
	}

	var retryCount int64
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		retryCount = field.Int() - 1
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		retryCount = int64(field.Uint()) - 1
	default:
		return
	}

	if retryCount > 0 {
		*args = append(*args, slog.Int64(key, retryCount))
	}
}

func appendStringAttribute(attrs *[]attribute.KeyValue, value reflect.Value, fieldName string, key string) {
	field := value.FieldByName(fieldName)
	if !field.IsValid() || field.Kind() != reflect.String || field.String() == "" {
		return
	}

	*attrs = append(*attrs, attribute.String(key, field.String()))
}

func appendIntAttribute(attrs *[]attribute.KeyValue, value reflect.Value, fieldName string, key string) {
	field := value.FieldByName(fieldName)
	if !field.IsValid() {
		return
	}

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Int() > 0 {
			*attrs = append(*attrs, attribute.Int64(key, field.Int()))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if field.Uint() > 0 {
			*attrs = append(*attrs, attribute.Int64(key, int64(field.Uint())))
		}
	}
}

func appendRetryCountAttribute(attrs *[]attribute.KeyValue, value reflect.Value, fieldName string, key string) {
	field := value.FieldByName(fieldName)
	if !field.IsValid() {
		return
	}

	var retryCount int64
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		retryCount = field.Int() - 1
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		retryCount = int64(field.Uint()) - 1
	default:
		return
	}

	if retryCount > 0 {
		*attrs = append(*attrs, attribute.Int64(key, retryCount))
	}
}
