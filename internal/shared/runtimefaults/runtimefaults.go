package runtimefaults

import (
	"context"
	"errors"
	"sync"
	"time"

	paymentports "github.com/rgomids/atlas-erp-core/internal/payments/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/shared/config"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
)

var ErrSimulatedGatewayFailure = errors.New("simulated gateway flaky failure")
var ErrSimulatedOutboxAppendFailure = errors.New("simulated outbox append failure")

func EventBusOptions(
	profile config.FaultProfile,
	telemetry *observability.Runtime,
	recorder sharedevent.Recorder,
) sharedevent.SyncBusOptions {
	options := sharedevent.SyncBusOptions{
		Recorder:      recorder,
		Observability: telemetry,
	}

	switch profile {
	case config.FaultProfileDuplicateBillingRequest:
		options.DuplicateFirstEventName = "BillingRequested"
	case config.FaultProfileEventConsumerFailure:
		options.FailFirstConsumerEventName = "BillingRequested"
		options.FailFirstConsumerModule = "payments"
	}

	return options
}

func DecorateRecorder(profile config.FaultProfile, next sharedevent.Recorder) sharedevent.Recorder {
	if profile != config.FaultProfileOutboxAppendFailure {
		return next
	}

	return &failFirstAppendRecorder{next: next}
}

func DecorateGateway(
	profile config.FaultProfile,
	timeout time.Duration,
	next paymentports.PaymentGateway,
) paymentports.PaymentGateway {
	if next == nil {
		return nil
	}

	switch profile {
	case config.FaultProfilePaymentTimeout:
		return delayedGateway{
			next:  next,
			delay: timeout + 50*time.Millisecond,
		}
	case config.FaultProfilePaymentFlakyFirst:
		return &failFirstGateway{
			next: next,
			err:  ErrSimulatedGatewayFailure,
		}
	default:
		return next
	}
}

type delayedGateway struct {
	next  paymentports.PaymentGateway
	delay time.Duration
}

func (gateway delayedGateway) Process(ctx context.Context, request paymentports.GatewayRequest) (paymentports.GatewayResult, error) {
	timer := time.NewTimer(gateway.delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return paymentports.GatewayResult{}, ctx.Err()
	case <-timer.C:
	}

	return gateway.next.Process(ctx, request)
}

type failFirstGateway struct {
	next     paymentports.PaymentGateway
	err      error
	mu       sync.Mutex
	consumed bool
}

func (gateway *failFirstGateway) Process(ctx context.Context, request paymentports.GatewayRequest) (paymentports.GatewayResult, error) {
	gateway.mu.Lock()
	if !gateway.consumed {
		gateway.consumed = true
		gateway.mu.Unlock()
		return paymentports.GatewayResult{}, gateway.err
	}
	gateway.mu.Unlock()

	return gateway.next.Process(ctx, request)
}

type failFirstAppendRecorder struct {
	next     sharedevent.Recorder
	mu       sync.Mutex
	consumed bool
}

func (recorder *failFirstAppendRecorder) Append(ctx context.Context, record sharedevent.EventRecord) error {
	recorder.mu.Lock()
	if !recorder.consumed {
		recorder.consumed = true
		recorder.mu.Unlock()
		return ErrSimulatedOutboxAppendFailure
	}
	recorder.mu.Unlock()

	if recorder.next == nil {
		return nil
	}

	return recorder.next.Append(ctx, record)
}

func (recorder *failFirstAppendRecorder) MarkProcessed(ctx context.Context, eventID string, processedAt time.Time) error {
	if recorder.next == nil {
		return nil
	}

	return recorder.next.MarkProcessed(ctx, eventID, processedAt)
}

func (recorder *failFirstAppendRecorder) MarkFailed(ctx context.Context, eventID string, failedAt time.Time, errorMessage string) error {
	if recorder.next == nil {
		return nil
	}

	return recorder.next.MarkFailed(ctx, eventID, failedAt, errorMessage)
}
