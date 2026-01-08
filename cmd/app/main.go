package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"simple-invest/internal/app"
	"simple-invest/internal/config"
	"syscall"
	"time"
)

func main() {
	cfg := config.MustLoad()
	log := setupLogger()
	app := app.New(log, cfg.StoragePath, cfg.Port)

	go func() {
		app.MustRun()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(cfg.Timeout))
	defer cancel()

	app.Stop(ctx)

	log.Info("server stoped")
}

func setupLogger() *slog.Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	return logger
}
