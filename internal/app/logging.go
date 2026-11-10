package app

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// LogPath is where the debug log is written when PRETTY_ALIAS_DEBUG is set:
// <user cache dir>/pretty-alias/debug.log (Linux: $XDG_CACHE_HOME or ~/.cache;
// macOS: ~/Library/Caches).
// It is never written into the current directory.
func LogPath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "pretty-alias", "debug.log"), nil
}

// setupLogging installs the default slog logger. Without PRETTY_ALIAS_DEBUG
// all logging is discarded. The returned func closes the log file.
func setupLogging() (closeLog func()) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if os.Getenv("PRETTY_ALIAS_DEBUG") == "" {
		return func() {}
	}
	path, err := LogPath()
	if err != nil {
		return func() {}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return func() {}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return func() {}
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug})))
	slog.Debug("started")
	return func() { _ = f.Close() }
}
