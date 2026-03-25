package events

import sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"

func Catalog() []sharedevent.Descriptor {
	return []sharedevent.Descriptor{
		{
			Name:           EventNameCustomerCreated,
			ProducerModule: "customers",
			Aggregate:      "customer",
			Description:    "Published after a customer is created successfully.",
		},
	}
}
