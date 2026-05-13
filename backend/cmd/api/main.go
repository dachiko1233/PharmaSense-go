package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"pharmasense/internal/config"
	"pharmasense/internal/db"
	"pharmasense/internal/server"
	"pharmasense/internal/services"
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

	// One-time risk recalculation on startup (set RECALCULATE_RISKS_ON_START=true, redeploy once, then unset)
	if os.Getenv("RECALCULATE_RISKS_ON_START") == "true" {
		if err := recalculateAllRisks(ctx, pool); err != nil {
			slog.Warn("risk recalc on start failed", "error", err)
		}
	}

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

func recalculateAllRisks(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := pool.Query(ctx, `SELECT DISTINCT pharmacy_id FROM inventory_batches WHERE current_quantity > 0`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var pharmacyIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		pharmacyIDs = append(pharmacyIDs, id)
	}
	rows.Close()

	today := time.Now().UTC().Truncate(24 * time.Hour)
	for _, pharmacyID := range pharmacyIDs {
		batchRows, err := pool.Query(ctx, `
			SELECT ib.id, ib.expiry_date, ib.current_quantity, ib.purchase_price,
			       COALESCE(
			         (SELECT SUM(s.quantity)::float / 90.0
			          FROM sales s
			          WHERE s.batch_id = ib.id
			            AND s.sale_date >= $2::date - 90),
			         0.5
			       ) as avg_daily_sales
			FROM inventory_batches ib
			WHERE ib.pharmacy_id = $1
		`, pharmacyID, today)
		if err != nil {
			slog.Warn("recalc query failed", "pharmacy_id", pharmacyID, "error", err)
			continue
		}

		type batchRow struct {
			id            string
			expiryDate    time.Time
			currentQty    int
			purchasePrice float64
			avgDailySales float64
		}
		var batches []batchRow
		for batchRows.Next() {
			var b batchRow
			if err := batchRows.Scan(&b.id, &b.expiryDate, &b.currentQty, &b.purchasePrice, &b.avgDailySales); err != nil {
				continue
			}
			batches = append(batches, b)
		}
		batchRows.Close()

		for _, b := range batches {
			result := services.CalculateRisk(services.RiskInput{
				CurrentQuantity: b.currentQty,
				ExpiryDate:      b.expiryDate,
				AvgDailySales:   b.avgDailySales,
				PurchasePrice:   b.purchasePrice,
				Today:           today,
			})
			_, _ = pool.Exec(ctx, `
				INSERT INTO risk_assessments
				  (batch_id, pharmacy_id, risk_level, days_until_expiry, avg_daily_sales,
				   expected_sales, estimated_surplus, estimated_loss, suggested_discount_percent)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			`, b.id, pharmacyID, result.RiskLevel, result.DaysUntilExpiry, b.avgDailySales,
				result.ExpectedSales, result.EstimatedSurplus, result.EstimatedLoss, result.SuggestedDiscountPct)
		}
		slog.Info("risk recalculated for pharmacy", "pharmacy_id", pharmacyID, "batches", len(batches))
	}
	return nil
}
