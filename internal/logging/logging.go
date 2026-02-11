package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

const maxLogSize = 5 * 1024 * 1024 // 5MB

func Setup(logPath string, verbose bool) (*slog.Logger, error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, err
	}

	if err := RotateIfNeeded(logPath); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	var w io.Writer = f
	if verbose {
		w = io.MultiWriter(f, os.Stderr)
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	return slog.New(handler), nil
}

func RotateIfNeeded(logPath string) error {
	info, err := os.Stat(logPath)
	if err != nil {
		return nil // file doesn't exist yet
	}

	if info.Size() <= maxLogSize {
		return nil
	}

	backup := logPath + ".old"
	os.Remove(backup)
	return os.Rename(logPath, backup)
}

type NopHandler struct{}

func (NopHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (NopHandler) Handle(context.Context, slog.Record) error { return nil }
func (h NopHandler) WithAttrs([]slog.Attr) slog.Handler      { return h }
func (h NopHandler) WithGroup(string) slog.Handler            { return h }
