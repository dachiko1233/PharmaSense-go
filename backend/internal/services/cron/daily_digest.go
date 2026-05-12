package cron

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"pharmasense/internal/services/notifications"
)

type DigestJob struct {
	db      *pgxpool.Pool
	email   *notifications.EmailService
	appURL  string
}

func NewDigestJob(db *pgxpool.Pool, email *notifications.EmailService, appURL string) *DigestJob {
	return &DigestJob{db: db, email: email, appURL: appURL}
}

func Start(job *DigestJob) (gocron.Scheduler, error) {
	scheduler, err := gocron.NewScheduler(gocron.WithLocation(cyprusLocation()))
	if err != nil {
		return nil, fmt.Errorf("create scheduler: %w", err)
	}

	_, err = scheduler.NewJob(
		gocron.CronJob("0 8 * * *", false), // 8 AM every day
		gocron.NewTask(job.Run),
	)
	if err != nil {
		return nil, fmt.Errorf("schedule digest job: %w", err)
	}

	scheduler.Start()
	slog.Info("daily digest cron scheduled", "time", "08:00 Cyprus time")
	return scheduler, nil
}

func (j *DigestJob) Run() {
	ctx := context.Background()
	slog.Info("running daily digest job")

	// Find all pharmacies with pro/chain plan that have users with email_verified
	rows, err := j.db.Query(ctx, `
		SELECT DISTINCT u.id, u.email, u.full_name, p.id, p.name,
		       COUNT(ra.id) FILTER (WHERE ra.risk_level = 'CRITICAL') as critical_count,
		       COALESCE(SUM(ra.estimated_loss) FILTER (WHERE ra.risk_level IN ('CRITICAL','HIGH')), 0) as total_loss
		FROM users u
		JOIN pharmacy_users pu ON pu.user_id = u.id
		JOIN pharmacies p ON p.id = pu.pharmacy_id
		JOIN risk_assessments ra ON ra.pharmacy_id = p.id
		WHERE u.email_verified = true
		  AND p.plan IN ('pro', 'chain')
		  AND ra.risk_level = 'CRITICAL'
		GROUP BY u.id, u.email, u.full_name, p.id, p.name
		HAVING COUNT(ra.id) FILTER (WHERE ra.risk_level = 'CRITICAL') > 0
	`)
	if err != nil {
		slog.Error("daily digest: query failed", "error", err)
		return
	}
	defer rows.Close()

	sent := 0
	for rows.Next() {
		var userID, pharmacyID string
		var email, fullName, pharmacyName string
		var criticalCount int
		var totalLoss float64
		if err := rows.Scan(&userID, &email, &fullName, &pharmacyID, &pharmacyName, &criticalCount, &totalLoss); err != nil {
			continue
		}
		if err := j.email.SendDailyDigest(ctx, email, fullName, pharmacyName, j.appURL, criticalCount, totalLoss); err != nil {
			slog.Error("daily digest: send failed", "email", email, "error", err)
			continue
		}
		sent++
	}
	if err := rows.Err(); err != nil {
		slog.Error("daily digest: row iteration error", "error", err)
	}
	slog.Info("daily digest complete", "emails_sent", sent)
}

func cyprusLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Nicosia")
	if err != nil {
		// Fallback to UTC+2 (EET)
		loc = time.FixedZone("EET", 2*60*60)
	}
	return loc
}
