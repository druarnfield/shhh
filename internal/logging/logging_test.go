package logging

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestSetup_CreatesLogFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	logger, err := Setup(logPath, false)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	logger.Info("test message", "key", "value")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}

	if len(data) == 0 {
		t.Error("log file is empty")
	}
}

func TestSetup_VerboseAddsStderr(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	logger, err := Setup(logPath, true)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	// Just verify it doesn't panic
	logger.Info("verbose test")
}

func TestRotateIfNeeded(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	// Create a file just over the max size
	data := make([]byte, maxLogSize+1)
	if err := os.WriteFile(logPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	if err := RotateIfNeeded(logPath); err != nil {
		t.Fatalf("RotateIfNeeded: %v", err)
	}

	// Original should be gone or truncated
	info, err := os.Stat(logPath)
	if err != nil {
		// File was rotated away, that's fine
		return
	}
	if info.Size() > maxLogSize {
		t.Errorf("log file still %d bytes, want <= %d", info.Size(), maxLogSize)
	}
}

func TestNopLogger(t *testing.T) {
	logger := slog.New(NopHandler{})
	// Should not panic
	logger.Info("nop")
}
