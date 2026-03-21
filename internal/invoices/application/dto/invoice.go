package dto

import "time"

type Invoice struct {
	ID          string     `json:"id"`
	CustomerID  string     `json:"customer_id"`
	AmountCents int64      `json:"amount_cents"`
	DueDate     string     `json:"due_date"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
}
