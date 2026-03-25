package runtimefaults

import (
	"context"
	"errors"
	"testing"
	"time"

	paymentports "github.com/rgomids/atlas-erp-core/internal/payments/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/shared/config"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

type recorderStub struct {
	appendCalls int
}

func (stub *recorderStub) Append(context.Context, sharedevent.EventRecord) error {
	stub.appendCalls++
	return nil
}

func (stub *recorderStub) MarkProcessed(context.Context, string, time.Time) error {
	return nil
}

func (stub *recorderStub) MarkFailed(context.Context, string, time.Time, string) error {
	return nil
}

type gatewayStub struct {
	calls int
}

func (stub *gatewayStub) Process(context.Context, paymentports.GatewayRequest) (paymentports.GatewayResult, error) {
	stub.calls++
	return paymentports.GatewayResult{
		Status:           "Approved",
		GatewayReference: "gw-approved",
	}, nil
}

func TestDecorateGatewayPaymentFlakyFirstFailsOnlyOnce(t *testing.T) {
	t.Parallel()

	next := &gatewayStub{}
	gateway := DecorateGateway(config.FaultProfilePaymentFlakyFirst, time.Second, next)

	_, err := gateway.Process(context.Background(), paymentports.GatewayRequest{})
	if !errors.Is(err, ErrSimulatedGatewayFailure) {
		t.Fatalf("expected simulated gateway failure, got %v", err)
	}

	result, err := gateway.Process(context.Background(), paymentports.GatewayRequest{})
	if err != nil {
		t.Fatalf("expected second gateway call to recover, got %v", err)
	}

	if result.Status != "Approved" {
		t.Fatalf("expected approved result after first failure, got %q", result.Status)
	}

	if next.calls != 1 {
		t.Fatalf("expected wrapped gateway to be called once after recovery, got %d", next.calls)
	}
}

func TestDecorateGatewayPaymentTimeoutReturnsContextDeadlineExceeded(t *testing.T) {
	t.Parallel()

	gateway := DecorateGateway(config.FaultProfilePaymentTimeout, 5*time.Millisecond, &gatewayStub{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	_, err := gateway.Process(ctx, paymentports.GatewayRequest{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}

func TestDecorateRecorderOutboxAppendFailureFailsOnlyFirstAppend(t *testing.T) {
	t.Parallel()

	next := &recorderStub{}
	recorder := DecorateRecorder(config.FaultProfileOutboxAppendFailure, next)

	if err := recorder.Append(context.Background(), sharedevent.EventRecord{}); !errors.Is(err, ErrSimulatedOutboxAppendFailure) {
		t.Fatalf("expected simulated outbox append failure, got %v", err)
	}

	if err := recorder.Append(context.Background(), sharedevent.EventRecord{}); err != nil {
		t.Fatalf("expected second append to recover, got %v", err)
	}

	if next.appendCalls != 1 {
		t.Fatalf("expected wrapped recorder to receive one recovered append, got %d", next.appendCalls)
	}
}
