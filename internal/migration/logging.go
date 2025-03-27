package migration

import (
	"fmt"
	"gopkg.in/natefinch/lumberjack.v2"
	"log/slog"
	"os"
)

func setLogger(file *os.File, debug bool) error {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(&lumberjack.Logger{
		Filename:   file.Name(),
		MaxSize:    5,
		MaxBackups: 5,
		Compress:   true,
	}, &slog.HandlerOptions{
		Level: level,
	}))

	slog.SetDefault(logger)
	slog.Debug("DEBUGGING ENABLED")
	return nil
}

func openLogFile(filePath string) (*os.File, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return file, nil
}
