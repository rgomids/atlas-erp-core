package event

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"reflect"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"

	httpapi "github.com/rgomids/atlas-erp-core/internal/shared/http"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
)

type Event interface {
	Name() string
	EventMetadata() Metadata
	EventPayload() any
}

type EventHandler interface {
	Handle(ctx context.Context, event Event) error
}

type EventBus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(eventName string, handler EventHandler)
}

type EventRecord struct {
	EventID       string
	EventName     string
	AggregateID   string
	EmitterModule string
	CorrelationID string
	Payload       []byte
	OccurredAt    time.Time
}

type Recorder interface {
	Append(ctx context.Context, record EventRecord) error
	MarkProcessed(ctx context.Context, eventID string, processedAt time.Time) error
	MarkFailed(ctx context.Context, eventID string, failedAt time.Time, errorMessage string) error
}

type HandlerFunc func(ctx context.Context, event Event) error

func (handler HandlerFunc) Handle(ctx context.Context, event Event) error {
	return handler(ctx, event)
}

type SyncBusOptions struct {
	Recorder                   Recorder
	Observability              *observability.Runtime
	Now                        func() time.Time
	DuplicateFirstEventName    string
	FailFirstConsumerEventName string
	FailFirstConsumerModule    string
}

type SyncBus struct {
	mu            sync.RWMutex
	handlers      map[string][]EventHandler
	recorder      Recorder
	now           func() time.Time
	observability *observability.Runtime
	faultMu       sync.Mutex

	duplicateFirstEventName    string
	duplicateTriggered         bool
	failFirstConsumerEventName string
	failFirstConsumerModule    string
	failFirstConsumerTriggered bool
}

type emitterModuleContextKey string

const emitterModuleKey emitterModuleContextKey = "event_emitter_module"

var ErrInjectedConsumerFailure = errors.New("simulated event consumer failure")

func NewSyncBus(recorders ...Recorder) *SyncBus {
	var recorder Recorder
	if len(recorders) > 0 {
		recorder = recorders[0]
	}

	return NewSyncBusWithOptions(SyncBusOptions{
		Recorder: recorder,
	})
}

func NewSyncBusWithObservability(telemetry *observability.Runtime, recorders ...Recorder) *SyncBus {
	var recorder Recorder
	if len(recorders) > 0 {
		recorder = recorders[0]
	}

	return NewSyncBusWithOptions(SyncBusOptions{
		Recorder:      recorder,
		Observability: telemetry,
	})
}

func NewSyncBusWithOptions(options SyncBusOptions) *SyncBus {
	now := options.Now
	if now == nil {
		now = time.Now
	}

	return &SyncBus{
		handlers:                   map[string][]EventHandler{},
		recorder:                   options.Recorder,
		now:                        now,
		observability:              options.Observability,
		duplicateFirstEventName:    options.DuplicateFirstEventName,
		failFirstConsumerEventName: options.FailFirstConsumerEventName,
		failFirstConsumerModule:    options.FailFirstConsumerModule,
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

	if err := bus.appendRecord(ctx, emitterModule, domainEvent); err != nil {
		logger.Error("🧾 event record failed", append([]any{slog.Any("err", err)}, eventLogArgs(domainEvent)...)...)
		telemetry.RecordSpanError(publishSpan, err, observability.ErrorTypeInfrastructure)
		return err
	}

	logger.Info("📣 event published", eventArgs...)
	telemetry.RecordEventPublished(publishContext, domainEvent.Name(), emitterModule)

	if err := bus.dispatch(publishContext, domainEvent, handlers, logger, telemetry); err != nil {
		return bus.handleDispatchFailure(ctx, domainEvent, logger, telemetry, err)
	}

	if bus.shouldDuplicate(domainEvent.Name()) {
		logger.Info("event delivery duplicated", eventArgs...)
		if err := bus.dispatch(publishContext, domainEvent, handlers, logger, telemetry); err != nil {
			return bus.handleDispatchFailure(ctx, domainEvent, logger, telemetry, err)
		}
	}

	if err := bus.markProcessed(ctx, domainEvent); err != nil {
		logger.Error("🧾 event mark processed failed", append([]any{slog.Any("err", err)}, eventLogArgs(domainEvent)...)...)
		telemetry.RecordSpanError(publishSpan, err, observability.ErrorTypeInfrastructure)
		return err
	}

	return nil
}

func (bus *SyncBus) dispatch(
	publishContext context.Context,
	domainEvent Event,
	handlers []EventHandler,
	logger *slog.Logger,
	telemetry *observability.Runtime,
) error {
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

		handleErr := error(nil)
		if bus.shouldFailConsumer(domainEvent.Name(), consumerModule) {
			handleErr = ErrInjectedConsumerFailure
		} else {
			handleErr = handler.Handle(handlerContext, domainEvent)
		}

		if handleErr != nil {
			handlerLogger.Error(
				"💥 event handler failed",
				append(
					[]any{
						slog.Any("err", handleErr),
						slog.String("error_type", observability.ErrorTypeInfrastructure),
					},
					eventLogArgs(domainEvent)...,
				)...,
			)
			telemetry.RecordEventHandlerFailure(handlerContext, domainEvent.Name(), consumerModule, observability.ErrorTypeInfrastructure)
			telemetry.CompleteSpan(handlerSpan, handleErr, observability.ErrorTypeInfrastructure)
			return handleErr
		}

		handlerLogger.Info("✅ event handled", eventLogArgs(domainEvent)...)
		telemetry.CompleteSpan(handlerSpan, nil, "")
	}

	return nil
}

func (bus *SyncBus) handleDispatchFailure(
	ctx context.Context,
	domainEvent Event,
	logger *slog.Logger,
	telemetry *observability.Runtime,
	dispatchErr error,
) error {
	if recordErr := bus.markFailed(ctx, domainEvent, dispatchErr); recordErr != nil {
		logger.Error(
			"🧾 event outbox failed status update failed",
			append([]any{slog.Any("err", recordErr)}, eventLogArgs(domainEvent)...)...,
		)
	}

	return dispatchErr
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

func (bus *SyncBus) appendRecord(ctx context.Context, emitterModule string, domainEvent Event) error {
	if bus.recorder == nil {
		return nil
	}

	payload, err := json.Marshal(domainEvent)
	if err != nil {
		return err
	}

	metadata := domainEvent.EventMetadata()

	return bus.recorder.Append(ctx, EventRecord{
		EventID:       metadata.EventID,
		EventName:     domainEvent.Name(),
		AggregateID:   metadata.AggregateID,
		EmitterModule: emitterModule,
		CorrelationID: metadata.CorrelationID,
		Payload:       payload,
		OccurredAt:    metadata.OccurredAt,
	})
}

func (bus *SyncBus) markProcessed(ctx context.Context, domainEvent Event) error {
	if bus.recorder == nil {
		return nil
	}

	return bus.recorder.MarkProcessed(ctx, domainEvent.EventMetadata().EventID, bus.now().UTC())
}

func (bus *SyncBus) markFailed(ctx context.Context, domainEvent Event, failure error) error {
	if bus.recorder == nil {
		return nil
	}

	return bus.recorder.MarkFailed(ctx, domainEvent.EventMetadata().EventID, bus.now().UTC(), failure.Error())
}

func (bus *SyncBus) shouldDuplicate(eventName string) bool {
	if bus.duplicateFirstEventName == "" || bus.duplicateFirstEventName != eventName {
		return false
	}

	bus.faultMu.Lock()
	defer bus.faultMu.Unlock()

	if bus.duplicateTriggered {
		return false
	}

	bus.duplicateTriggered = true
	return true
}

func (bus *SyncBus) shouldFailConsumer(eventName string, consumerModule string) bool {
	if bus.failFirstConsumerEventName == "" || bus.failFirstConsumerModule == "" {
		return false
	}

	if bus.failFirstConsumerEventName != eventName || bus.failFirstConsumerModule != consumerModule {
		return false
	}

	bus.faultMu.Lock()
	defer bus.faultMu.Unlock()

	if bus.failFirstConsumerTriggered {
		return false
	}

	bus.failFirstConsumerTriggered = true
	return true
}

func eventLogArgs(domainEvent Event) []any {
	metadata := domainEvent.EventMetadata()
	args := []any{
		slog.String("event_id", metadata.EventID),
		slog.String("aggregate_id", metadata.AggregateID),
		slog.String("correlation_id", metadata.CorrelationID),
	}

	value := reflect.ValueOf(domainEvent.EventPayload())
	if !value.IsValid() {
		return args
	}

	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return args
		}

		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return args
	}

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
	metadata := domainEvent.EventMetadata()
	attributes := []attribute.KeyValue{
		attribute.String("atlas.event_id", metadata.EventID),
		attribute.String("atlas.aggregate_id", metadata.AggregateID),
		attribute.String("atlas.correlation_id", metadata.CorrelationID),
	}

	value := reflect.ValueOf(domainEvent.EventPayload())
	if !value.IsValid() {
		return attributes
	}

	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return attributes
		}

		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return attributes
	}

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
