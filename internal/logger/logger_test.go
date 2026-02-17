package logger

import (
	"sync"
	"testing"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "default config",
			cfg:  DefaultConfig(),
		},
		{
			name: "debug level",
			cfg: Config{
				Level:  "debug",
				Format: "text",
			},
		},
		{
			name: "json format",
			cfg: Config{
				Level:  "info",
				Format: "json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset instance for test
			instance = nil
			once = sync.Once{}

			Init(tt.cfg)
			if Get() == nil {
				t.Error("Expected logger to be initialized")
			}
		})
	}
}

func TestLogging(t *testing.T) {
	// Reset for test
	instance = nil
	once = sync.Once{}

	Init(Config{
		Level:  "debug",
		Format: "text",
	})

	// Test that logging doesn't panic
	Debug("debug message", "key", "value")
	Info("info message", "key", "value")
	Warn("warn message", "key", "value")
	Error("error message", nil, "key", "value")
}

func TestWithComponent(t *testing.T) {
	// Reset for test
	instance = nil
	once = sync.Once{}

	Init(DefaultConfig())

	logger := WithComponent("test-component")
	if logger == nil {
		t.Error("Expected logger with component")
	}
}

func TestWithRequestID(t *testing.T) {
	// Reset for test
	instance = nil
	once = sync.Once{}

	Init(DefaultConfig())

	logger := WithRequestID("req-123")
	if logger == nil {
		t.Error("Expected logger with request ID")
	}
}
