package cron

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"pharmasense/internal/domain"
	"pharmasense/internal/services/notifications"
)

type DigestJob struct {
	db     *pgxpool.Pool
	email  *notifications.EmailService
	appURL string
}

func NewDigestJob(db *pgxpool.Pool, email *notifications.EmailService, appURL string) *DigestJob {
	return &DigestJob{db: db, email: email, appURL: appURL}
}

func Start(job *DigestJob, expiryJob *ExpiryCleanupJob) (gocron.Scheduler, error) {
	scheduler, err := gocron.NewScheduler(gocron.WithLocation(cyprusLocation()))
	if err != nil {
		return nil, fmt.Errorf("create scheduler: %w", err)
	}

	_, err = scheduler.NewJob(
		gocron.CronJob("0 * * * *", false), // every hour — sends to users whose digest_time matches
		gocron.NewTask(job.Run),
	)
	if err != nil {
		return nil, fmt.Errorf("schedule digest job: %w", err)
	}

	_, err = scheduler.NewJob(
		gocron.CronJob("0 2 * * *", false), // once a day at 2am — purge batches expired 7+ days
		gocron.NewTask(expiryJob.Run),
	)
	if err != nil {
		return nil, fmt.Errorf("schedule expiry cleanup job: %w", err)
	}

	scheduler.Start()
	slog.Info("hourly digest cron scheduled", "schedule", "0 * * * * Cyprus time")
	slog.Info("daily expiry cleanup cron scheduled", "schedule", "0 2 * * * Cyprus time")
	return scheduler, nil
}

// Run is called by the scheduler each hour. It sends the digest only to users
// whose preferred digest_time hour matches the current Cyprus time hour.
func (j *DigestJob) Run() {
	cyprusNow := time.Now().In(cyprusLocation())
	currentHour := cyprusNow.Hour()
	sent, failed := j.RunWithStats(context.Background(), "", currentHour)
	slog.Info("hourly digest complete", "hour", currentHour, "emails_sent", sent, "failed", failed)
}

// RunWithStats sends the daily digest and returns counts.
// filterPharmacyID limits execution to one pharmacy (manual-trigger endpoint).
// filterHour, when >= 0, limits to users whose digest_time hour matches (cron path).
// Pass filterHour = -1 to send regardless of the user's chosen time.
func (j *DigestJob) RunWithStats(ctx context.Context, filterPharmacyID string, filterHour int) (sent, failed int) {
	slog.Info("running digest job", "filter_pharmacy_id", filterPharmacyID, "filter_hour", filterHour)

	// ── query all eligible users+pharmacies ──────────────────────────
	//
	// We use DISTINCT ON to get the latest risk assessment per batch,
	// then aggregate counts and totals per user-pharmacy pair.
	args := []any{}
	pharmacyFilter := ""
	if filterPharmacyID != "" {
		args = append(args, filterPharmacyID)
		pharmacyFilter = fmt.Sprintf("AND p.id = $%d", len(args))
	}
	hourFilter := ""
	if filterHour >= 0 {
		args = append(args, filterHour)
		hourFilter = fmt.Sprintf("AND EXTRACT(HOUR FROM u.digest_time) = $%d", len(args))
	}

	query := fmt.Sprintf(`
		WITH latest_risks AS (
		    SELECT DISTINCT ON (batch_id)
		        id, batch_id, pharmacy_id, risk_level, days_until_expiry,
		        estimated_loss, suggested_discount_percent
		    FROM risk_assessments
		    ORDER BY batch_id, calculated_at DESC
		)
		SELECT
		    u.id,
		    u.email,
		    u.full_name,
		    p.id,
		    p.name,
		    p.language,
		    p.plan,
		    COALESCE(COUNT(lr.id) FILTER (WHERE lr.risk_level = 'EXPIRED'), 0)                               AS expired_count,
		    COALESCE(COUNT(lr.id) FILTER (WHERE lr.risk_level = 'CRITICAL' AND lr.days_until_expiry <= 7), 0) AS week_count,
		    COALESCE(COUNT(lr.id) FILTER (WHERE lr.risk_level = 'CRITICAL'), 0)                              AS critical_count,
		    COALESCE(COUNT(lr.id) FILTER (WHERE lr.risk_level = 'HIGH'), 0)                                  AS high_count,
		    COALESCE(SUM(lr.estimated_loss) FILTER (
		        WHERE lr.risk_level IN ('EXPIRED','CRITICAL','HIGH')
		    ), 0) AS total_loss,
		    COALESCE(SUM(lr.estimated_loss * lr.suggested_discount_percent / 100.0) FILTER (
		        WHERE lr.risk_level IN ('CRITICAL','HIGH') AND lr.suggested_discount_percent IS NOT NULL
		    ), 0) AS potential_savings
		FROM users u
		JOIN pharmacy_users pu ON pu.user_id = u.id
		JOIN pharmacies p ON p.id = pu.pharmacy_id
		LEFT JOIN latest_risks lr ON lr.pharmacy_id = p.id
		WHERE u.email_verified = true
		  AND u.daily_digest_enabled = true
		  AND p.plan IN ('free','pro','chain')
		  %s
		  %s
		GROUP BY u.id, u.email, u.full_name, p.id, p.name, p.language, p.plan
	`, pharmacyFilter, hourFilter)

	rows, err := j.db.Query(ctx, query, args...)
	if err != nil {
		slog.Error("daily digest: summary query failed", "error", err)
		return
	}
	defer rows.Close()

	type recipientRow struct {
		userID       string
		email        string
		fullName     string
		pharmacyID   string
		pharmacyName string
		language     string
		plan         string
		expiredCount int
		weekCount    int
		criticalCount int
		highCount    int
		totalLoss    float64
		savings      float64
	}

	var recipients []recipientRow
	for rows.Next() {
		var r recipientRow
		if err := rows.Scan(
			&r.userID, &r.email, &r.fullName,
			&r.pharmacyID, &r.pharmacyName, &r.language, &r.plan,
			&r.expiredCount, &r.weekCount, &r.criticalCount, &r.highCount,
			&r.totalLoss, &r.savings,
		); err != nil {
			slog.Error("daily digest: scan row failed", "error", err)
			continue
		}
		recipients = append(recipients, r)
	}
	if err := rows.Err(); err != nil {
		slog.Error("daily digest: row iteration error", "error", err)
	}
	rows.Close()

	skipped := 0
	for _, r := range recipients {
		// Enforce the Free plan's monthly email-alert cap before doing any
		// work. Paid plans return Unlimited and are never counted or blocked.
		if limit := domain.MaxEmailAlertsPerMonth(r.plan); limit != domain.Unlimited {
			if used := j.emailAlertsSentThisMonth(ctx, r.userID); used >= limit {
				slog.Info("daily digest: monthly email-alert limit reached, skipping",
					"user_id", r.userID, "pharmacy_id", r.pharmacyID, "plan", r.plan, "used", used, "limit", limit)
				j.logNotification(ctx, r.userID, r.pharmacyID, r.email, "skipped_limit", nil)
				skipped++
				continue
			}
		}

		topItems, err := j.fetchTopItems(ctx, r.pharmacyID, r.language)
		if err != nil {
			slog.Warn("daily digest: top items query failed", "pharmacy_id", r.pharmacyID, "error", err)
		}

		data := notifications.DigestData{
			FullName:         r.fullName,
			PharmacyName:     r.pharmacyName,
			Language:         r.language,
			AppURL:           j.appURL,
			ExpiredCount:     r.expiredCount,
			WeekCount:        r.weekCount,
			CriticalCount:    r.criticalCount,
			HighCount:        r.highCount,
			TotalAtRisk:      r.totalLoss,
			PotentialSavings: r.savings,
			TopItems:         topItems,
		}

		sendErr := j.email.SendDailyDigest(ctx, r.email, data)
		status := "sent"
		var errMsg *string
		if sendErr != nil {
			slog.Error("daily digest: send failed", "email", r.email, "error", sendErr)
			status = "failed"
			msg := sendErr.Error()
			errMsg = &msg
			failed++
		} else {
			sent++
		}

		j.logNotification(ctx, r.userID, r.pharmacyID, r.email, status, errMsg)
	}

	if skipped > 0 {
		slog.Info("daily digest: recipients skipped for monthly email-alert limit", "skipped", skipped)
	}
	return sent, failed
}

// emailAlertsSentThisMonth counts alert emails successfully sent to a user in
// the current calendar month. The month boundary makes the quota reset
// automatically on the 1st. On a query error it fails open (returns 0) so a
// transient DB blip doesn't silently suppress a user's alerts.
func (j *DigestJob) emailAlertsSentThisMonth(ctx context.Context, userID string) int {
	var n int
	err := j.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM notification_log
		WHERE user_id = $1
		  AND channel = 'email'
		  AND status = 'sent'
		  AND sent_at >= date_trunc('month', now())
	`, userID).Scan(&n)
	if err != nil {
		slog.Warn("daily digest: failed to count monthly email alerts", "user_id", userID, "error", err)
		return 0
	}
	return n
}

// fetchTopItems returns the top 5 most urgent at-risk products for a pharmacy.
func (j *DigestJob) fetchTopItems(ctx context.Context, pharmacyID, language string) ([]notifications.DigestItem, error) {
	rows, err := j.db.Query(ctx, `
		WITH latest_risks AS (
		    SELECT DISTINCT ON (batch_id)
		        id, batch_id, pharmacy_id, risk_level, days_until_expiry,
		        estimated_loss
		    FROM risk_assessments
		    ORDER BY batch_id, calculated_at DESC
		)
		SELECT
		    CASE WHEN $2 = 'el' AND pr.name_el IS NOT NULL THEN pr.name_el ELSE pr.name END AS product_name,
		    lr.days_until_expiry,
		    ib.current_quantity,
		    COALESCE(lr.estimated_loss, 0) AS estimated_loss,
		    lr.risk_level
		FROM latest_risks lr
		JOIN inventory_batches ib ON ib.id = lr.batch_id
		JOIN products pr ON pr.id = ib.product_id
		WHERE lr.pharmacy_id = $1
		  AND lr.risk_level IN ('EXPIRED','CRITICAL','HIGH')
		  AND ib.current_quantity > 0
		ORDER BY
		    CASE lr.risk_level WHEN 'EXPIRED' THEN 1 WHEN 'CRITICAL' THEN 2 WHEN 'HIGH' THEN 3 END ASC,
		    lr.days_until_expiry ASC,
		    lr.estimated_loss DESC NULLS LAST
		LIMIT 5
	`, pharmacyID, language)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []notifications.DigestItem
	for rows.Next() {
		var item notifications.DigestItem
		if err := rows.Scan(&item.ProductName, &item.DaysLeft, &item.Quantity, &item.Loss, &item.RiskLevel); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// logNotification writes a row to notification_log for audit purposes.
func (j *DigestJob) logNotification(ctx context.Context, userID, pharmacyID, recipient, status string, errMsg *string) {
	_, err := j.db.Exec(ctx, `
		INSERT INTO notification_log (user_id, pharmacy_id, channel, template, recipient, status, error_message)
		VALUES ($1, $2, 'email', 'daily_digest', $3, $4, $5)
	`, userID, pharmacyID, recipient, status, errMsg)
	if err != nil {
		slog.Warn("daily digest: failed to write notification log", "error", err)
	}
}

func cyprusLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Nicosia")
	if err != nil {
		loc = time.FixedZone("EET", 2*60*60)
	}
	return loc
}
