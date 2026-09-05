package logger

import (
	"log/slog"
	"os"
)

func Setup() {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	if env == "development" {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))
	} else {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	}
}
