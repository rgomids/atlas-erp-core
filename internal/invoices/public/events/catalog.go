package events

import sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"

func Catalog() []sharedevent.Descriptor {
	return []sharedevent.Descriptor{
		{
			Name:           EventNameInvoiceCreated,
			ProducerModule: "invoices",
			Aggregate:      "invoice",
			Description:    "Published when an invoice is created.",
		},
		{
			Name:           EventNameInvoicePaid,
			ProducerModule: "invoices",
			Aggregate:      "invoice",
			Description:    "Published when an invoice is marked as paid.",
		},
	}
}
