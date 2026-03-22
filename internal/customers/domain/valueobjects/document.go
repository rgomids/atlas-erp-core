package valueobjects

import (
	"errors"
	"strings"
	"unicode"
)

var ErrInvalidDocument = errors.New("invalid document")

type Document struct {
	value string
}

func NewDocument(raw string) (Document, error) {
	normalized := normalizeDigits(raw)
	if len(normalized) != 11 && len(normalized) != 14 {
		return Document{}, ErrInvalidDocument
	}

	return Document{value: normalized}, nil
}

func (document Document) Value() string {
	return document.value
}

func normalizeDigits(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))

	for _, char := range value {
		if unicode.IsDigit(char) {
			builder.WriteRune(char)
		}
	}

	return builder.String()
}
