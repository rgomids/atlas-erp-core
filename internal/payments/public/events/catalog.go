package events

import sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"

func Catalog() []sharedevent.Descriptor {
	return []sharedevent.Descriptor{
		{
			Name:           EventNamePaymentApproved,
			ProducerModule: "payments",
			Aggregate:      "payment",
			Description:    "Published when a payment attempt succeeds.",
		},
		{
			Name:           EventNamePaymentFailed,
			ProducerModule: "payments",
			Aggregate:      "payment",
			Description:    "Published when a payment attempt fails.",
		},
	}
}
