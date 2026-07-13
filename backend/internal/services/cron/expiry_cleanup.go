package cron

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ExpiryCleanupJob permanently removes inventory batches once they have been
// expired for more than 7 days, giving pharmacies a grace period to act on
// expiry alerts before the stock is purged.
type ExpiryCleanupJob struct {
	db *pgxpool.Pool
}

func NewExpiryCleanupJob(db *pgxpool.Pool) *ExpiryCleanupJob {
	return &ExpiryCleanupJob{db: db}
}

// Run deletes each batch individually so that a single row blocked by
// referencing sales/alert_actions history doesn't abort the whole cleanup.
func (j *ExpiryCleanupJob) Run() {
	ctx := context.Background()

	rows, err := j.db.Query(ctx, `
		SELECT id FROM inventory_batches
		WHERE expiry_date < CURRENT_DATE - INTERVAL '7 days'
	`)
	if err != nil {
		slog.Error("expiry cleanup: query failed", "error", err)
		return
	}

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		slog.Error("expiry cleanup: row iteration error", "error", err)
	}

	deleted, skipped := 0, 0
	for _, id := range ids {
		if _, err := j.db.Exec(ctx, `DELETE FROM inventory_batches WHERE id = $1`, id); err != nil {
			slog.Warn("expiry cleanup: skipped batch (likely has sales/alert history)", "batch_id", id, "error", err)
			skipped++
			continue
		}
		deleted++
	}

	if deleted > 0 || skipped > 0 {
		slog.Info("expiry cleanup complete", "deleted", deleted, "skipped", skipped)
	}
}
