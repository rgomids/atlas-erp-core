package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

func New(level string) (*slog.Logger, error) {
	return NewWithWriter(level, os.Stdout)
}

func NewWithWriter(level string, writer io.Writer) (*slog.Logger, error) {
	logLevel, err := parseLevel(level)
	if err != nil {
		return nil, err
	}

	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: logLevel,
	})

	return slog.New(handler), nil
}

func parseLevel(level string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unsupported LOG_LEVEL value %q", level)
	}
}
