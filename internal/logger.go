package internal

import (
	"log/slog"
	"os"
)

func NewLogger(config *Config) *slog.Logger {

	var level slog.Level
	switch config.Env {
	case Development:
		level = slog.LevelDebug
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
