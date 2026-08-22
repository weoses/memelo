package config

import (
	"log/slog"
	"os"
	"strings"

	"github.com/weoses/memelo/common/tracing"
)

func InitLogs(conf *LoggingConfig) {
	var base slog.Handler
	if strings.EqualFold(conf.Format, "json") {
		base = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: replaceAttrForCloudLogging})
	} else {
		base = slog.NewTextHandler(os.Stdout, nil)
	}
	logger := slog.New(tracing.NewHandler(base))

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

// replaceAttrForCloudLogging maps slog's default keys to the field names
// Cloud Logging's structured-log parser recognizes ("message", "severity"),
// and slog's level strings to Cloud Logging's severity enum (slog's "WARN"
// -> "WARNING"). Only used for the JSON handler -- Cloud Logging doesn't
// parse fields out of plain text payloads, so the text handler's output is
// left exactly as before.
func replaceAttrForCloudLogging(_ []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case slog.MessageKey:
		a.Key = "message"
	case slog.LevelKey:
		a.Key = "severity"
		if lvl, ok := a.Value.Any().(slog.Level); ok && lvl == slog.LevelWarn {
			a.Value = slog.StringValue("WARNING")
		}
	}
	return a
}
