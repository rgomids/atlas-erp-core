package events

type CustomerCreated struct {
	CustomerID string
}

func (CustomerCreated) Name() string {
	return "CustomerCreated"
}
