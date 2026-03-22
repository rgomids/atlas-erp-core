package event

import (
	"context"
	"log/slog"
	"sync"

	httpapi "github.com/rgomids/atlas-erp-core/internal/shared/http"
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

type HandlerFunc func(ctx context.Context, event Event) error

func (handler HandlerFunc) Handle(ctx context.Context, event Event) error {
	return handler(ctx, event)
}

type SyncBus struct {
	mu       sync.RWMutex
	handlers map[string][]EventHandler
}

type emitterModuleContextKey string

const emitterModuleKey emitterModuleContextKey = "event_emitter_module"

func NewSyncBus() *SyncBus {
	return &SyncBus{
		handlers: map[string][]EventHandler{},
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
		slog.String("emitter_module", emitterModule),
	)

	logger.Info("📣 event published", slog.Int("handler_count", len(handlers)))

	for _, handler := range handlers {
		consumerModule := handlerModule(handler)
		handlerLogger := logger.With(
			slog.String("module", consumerModule),
			slog.String("consumer_module", consumerModule),
		)

		handlerLogger.Info("📥 event handling")

		if err := handler.Handle(ctx, domainEvent); err != nil {
			handlerLogger.Error("💥 event handler failed", slog.Any("err", err))
			return err
		}

		handlerLogger.Info("✅ event handled")
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
