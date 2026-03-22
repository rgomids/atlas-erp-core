package integration

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/rgomids/atlas-erp-core/internal/payments/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/entities"
)

type MockGateway struct {
	status string
	delay  time.Duration
	err    error
}

func NewMockGateway() MockGateway {
	return MockGateway{status: string(entities.StatusApproved)}
}

func NewMockGatewayWithStatus(status string) MockGateway {
	return MockGateway{status: status}
}

func NewMockGatewayWithDelay(status string, delay time.Duration) MockGateway {
	return MockGateway{status: status, delay: delay}
}

func NewMockGatewayWithError(err error) MockGateway {
	return MockGateway{status: string(entities.StatusFailed), err: err}
}

func (gateway MockGateway) Process(ctx context.Context, _ ports.GatewayRequest) (ports.GatewayResult, error) {
	if gateway.status == "" {
		return ports.GatewayResult{}, fmt.Errorf("gateway status must be configured")
	}

	if gateway.delay > 0 {
		timer := time.NewTimer(gateway.delay)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return ports.GatewayResult{}, ctx.Err()
		case <-timer.C:
		}
	}

	if gateway.err != nil {
		return ports.GatewayResult{}, gateway.err
	}

	return ports.GatewayResult{
		Status:           gateway.status,
		GatewayReference: "mock-" + uuid.NewString(),
	}, nil
}
