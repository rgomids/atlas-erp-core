package httpapi

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type InputError struct {
	message string
}

func (err InputError) Error() string {
	return err.message
}

func IsInputError(err error) bool {
	var inputErr InputError
	return errors.As(err, &inputErr)
}

func RequireNonBlank(fieldName, value string) error {
	if strings.TrimSpace(value) != "" {
		return nil
	}

	return InputError{message: fmt.Sprintf("%s is required", fieldName)}
}

func RequireUUID(fieldName, value string) error {
	if err := RequireNonBlank(fieldName, value); err != nil {
		return err
	}

	if _, err := uuid.Parse(strings.TrimSpace(value)); err != nil {
		return InputError{message: fmt.Sprintf("%s must be a valid UUID", fieldName)}
	}

	return nil
}

func RequirePositiveInt64(fieldName string, value int64) error {
	if value > 0 {
		return nil
	}

	return InputError{message: fmt.Sprintf("%s must be greater than zero", fieldName)}
}

func RequireDate(fieldName, value string) error {
	if err := RequireNonBlank(fieldName, value); err != nil {
		return err
	}

	if _, err := time.Parse("2006-01-02", strings.TrimSpace(value)); err != nil {
		return InputError{message: fmt.Sprintf("%s must use YYYY-MM-DD", fieldName)}
	}

	return nil
}
