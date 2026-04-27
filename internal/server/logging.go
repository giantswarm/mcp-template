package server

import (
	"log/slog"
	"os"
)

// NewLogger builds the root slog logger. JSON when format=="json" or when
// running in Kubernetes (caller passes resolveLogFormat's output); text
// otherwise. debug switches the level from Info to Debug.
func NewLogger(debug bool, format string) *slog.Logger {
	lvl := slog.LevelInfo
	if debug {
		lvl = slog.LevelDebug
	}
	opts := &slog.HandlerOptions{Level: lvl}
	var h slog.Handler
	if format == LogFormatJSON {
		h = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		h = slog.NewTextHandler(os.Stderr, opts)
	}
	return slog.New(h)
}
