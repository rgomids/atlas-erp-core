package events

import sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"

func Catalog() []sharedevent.Descriptor {
	return []sharedevent.Descriptor{
		{
			Name:           EventNameBillingRequested,
			ProducerModule: "billing",
			Aggregate:      "billing",
			Description:    "Published when a billing attempt is ready to be processed.",
		},
	}
}
