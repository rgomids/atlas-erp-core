package integration

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/rgomids/atlas-erp-core/internal/payments/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/entities"
)

type MockGateway struct {
	status string
}

func NewMockGateway() MockGateway {
	return MockGateway{status: string(entities.StatusApproved)}
}

func NewMockGatewayWithStatus(status string) MockGateway {
	return MockGateway{status: status}
}

func (gateway MockGateway) Process(_ context.Context, _ ports.GatewayRequest) (ports.GatewayResult, error) {
	if gateway.status == "" {
		return ports.GatewayResult{}, fmt.Errorf("gateway status must be configured")
	}

	return ports.GatewayResult{
		Status:           gateway.status,
		GatewayReference: "mock-" + uuid.NewString(),
	}, nil
}
