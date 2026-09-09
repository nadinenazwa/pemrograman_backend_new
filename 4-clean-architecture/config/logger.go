package config

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

func NewLogger() *slog.Logger {
	if err := os.MkdirAll("logs", 0o755); err != nil {
		panic("gagal membuat folder logs: " + err.Error())
	}
	rotator := &lumberjack.Logger{
		Filename:   filepath.Join("logs", "app.log"),
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     14,
		Compress:   true,
	}
	writer := io.MultiWriter(os.Stdout, rotator)
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}