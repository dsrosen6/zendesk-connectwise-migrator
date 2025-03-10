package cmd

import (
	"fmt"
	"log/slog"
	"os"
)

func SetLogger(file *os.File) error {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: level,
	}))

	slog.SetDefault(logger)
	slog.Debug("DEBUGGING ENABLED")
	return nil
}

func OpenLogFile(filePath string) (*os.File, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return file, nil
}
