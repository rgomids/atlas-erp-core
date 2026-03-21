package valueobjects

import (
	"errors"
	"net/mail"
	"strings"
)

var ErrInvalidEmail = errors.New("invalid email")

type Email struct {
	value string
}

func NewEmail(raw string) (Email, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	address, err := mail.ParseAddress(normalized)
	if err != nil || address.Address != normalized {
		return Email{}, ErrInvalidEmail
	}

	return Email{value: normalized}, nil
}

func (email Email) Value() string {
	return email.value
}
