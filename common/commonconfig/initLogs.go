package commonconfig

import (
	"log/slog"
	"os"
)

func InitLogs() {
	conf := NewLoggingConfig()

	handler := slog.NewTextHandler(os.Stdout, nil)
	logger := slog.New(handler)
	switch conf.Level {
	case "debug":
		slog.SetLogLoggerLevel(slog.LevelDebug)
	case "info":
		slog.SetLogLoggerLevel(slog.LevelInfo)
	case "error":
		slog.SetLogLoggerLevel(slog.LevelError)
	default:
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	slog.SetDefault(logger)
}
