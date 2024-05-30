package main

import (
	"context"
	"log/slog"
)

type logger struct {
	slog.Handler
	level slog.Level
}

func (l *logger) Enabled(_ context.Context, level slog.Level) bool {
	return level >= l.level
}

func newLogger(level slog.Level) *slog.Logger {
	return slog.New(&logger{
		Handler: slog.Default().Handler(),
		level:   level,
	})
}
