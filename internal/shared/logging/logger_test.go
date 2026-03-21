package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewWithWriterCreatesStructuredLogger(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}

	logger, err := NewWithWriter("info", buffer)
	if err != nil {
		t.Fatalf("expected logger to be created, got error: %v", err)
	}

	logger.Info("foundation ready", "component", "api")

	output := buffer.String()
	for _, fragment := range []string{`"time"`, `"level":"INFO"`, `"msg":"foundation ready"`, `"component":"api"`} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected log output to contain %s, got %s", fragment, output)
		}
	}
}

func TestNewWithWriterRejectsUnknownLevel(t *testing.T) {
	t.Parallel()

	_, err := NewWithWriter("trace", &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected logger creation to fail for an unknown level")
	}
}
