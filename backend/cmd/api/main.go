package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pharmasense/internal/config"
	"pharmasense/internal/db"
	"pharmasense/internal/server"
	"pharmasense/internal/services/cron"
	"pharmasense/internal/services/notifications"
)

func main() {
	// JSON structured logging to stdout (Railway captures it)
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := config.Load()

	ctx := context.Background()

	// Auto-migrate on startup (Railway doesn't need manual migration steps)
	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	slog.Info("database connected")

	// Start cron jobs (unless DISABLE_CRON=true for multi-instance scale-out)
	if !cfg.DisableCron {
		emailSvc := notifications.NewEmailService(cfg.ResendAPIKey, cfg.ResendFromEmail)
		digestJob := cron.NewDigestJob(pool, emailSvc, cfg.AppURL)
		scheduler, err := cron.Start(digestJob)
		if err != nil {
			slog.Warn("cron startup failed", "error", err)
		} else {
			defer scheduler.Shutdown()
		}
	}

	r := server.New(cfg, pool)

	port := cfg.Port
	if port == "" {
		port = "3001"
	}

	srv := &http.Server{
		Addr:              "0.0.0.0:" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Graceful shutdown on SIGTERM (Railway sends this)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine; signal quit on error so defers still run
	go func() {
		slog.Info("server starting", "port", port, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			quit <- syscall.SIGTERM
		}
	}()
	<-quit

	slog.Info("shutting down gracefully...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("server stopped")
}
