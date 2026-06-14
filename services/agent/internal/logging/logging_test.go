package logging

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func levelEnabled(t *testing.T, logger *slog.Logger, level slog.Level) bool {
	t.Helper()
	return logger.Handler().Enabled(context.Background(), level)
}

func TestSetupLevels(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		debugEnabled bool
	}{
		{name: "debug", input: "debug", debugEnabled: true},
		{name: "info", input: "info", debugEnabled: false},
		{name: "warn", input: "warn", debugEnabled: false},
		{name: "error", input: "error", debugEnabled: false},
		{name: "default", input: "invalid", debugEnabled: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := Setup(tt.input)
			assert.NotNil(t, logger)
			assert.Equal(t, tt.debugEnabled, levelEnabled(t, logger, slog.LevelDebug))
		})
	}
}
