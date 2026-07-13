package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"pharmasense/internal/middleware"
	"pharmasense/internal/services"
)

type RiskHandler struct {
	db *pgxpool.Pool
}

func NewRiskHandler(db *pgxpool.Pool) *RiskHandler {
	return &RiskHandler{db: db}
}

func (h *RiskHandler) Dashboard(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)

	if _, err := recalculateForPharmacy(c.Request.Context(), h.db, pharmacyID); err != nil {
		slog.Warn("risk recalc failed, returning cached data", "error", err)
	}

	var stats struct {
		ExpiredCount        int     `json:"expired_count"`
		CriticalCount       int     `json:"critical_count"`
		HighCount           int     `json:"high_count"`
		MediumCount         int     `json:"medium_count"`
		LowCount            int     `json:"low_count"`
		EstimatedLoss       float64 `json:"estimated_loss"`
		PotentialSavings    float64 `json:"potential_savings"`
		TotalInventoryValue float64 `json:"total_inventory_value"`
		TotalBatches        int     `json:"total_batches"`
	}

	err := h.db.QueryRow(c.Request.Context(), `
		SELECT
		  COUNT(*) FILTER (WHERE ra.risk_level = 'EXPIRED') as expired_count,
		  COUNT(*) FILTER (WHERE ra.risk_level = 'CRITICAL') as critical_count,
		  COUNT(*) FILTER (WHERE ra.risk_level = 'HIGH') as high_count,
		  COUNT(*) FILTER (WHERE ra.risk_level = 'MEDIUM') as medium_count,
		  COUNT(*) FILTER (WHERE ra.risk_level = 'LOW') as low_count,
		  COALESCE(SUM(ra.estimated_loss) FILTER (WHERE ra.risk_level IN ('EXPIRED','CRITICAL','HIGH')), 0) as estimated_loss,
		  COALESCE(SUM(ra.estimated_loss * 0.6) FILTER (WHERE ra.risk_level IN ('CRITICAL','HIGH')), 0) as potential_savings,
		  COALESCE(SUM(ib.current_quantity * ib.purchase_price), 0) as total_inventory_value,
		  COUNT(DISTINCT ib.id) as total_batches
		FROM inventory_batches ib
		LEFT JOIN LATERAL (
			SELECT risk_level, estimated_loss
			FROM risk_assessments
			WHERE batch_id = ib.id
			ORDER BY calculated_at DESC
			LIMIT 1
		) ra ON true
		WHERE ib.pharmacy_id = $1 AND ib.current_quantity > 0
	`, pharmacyID).Scan(
		&stats.ExpiredCount, &stats.CriticalCount, &stats.HighCount, &stats.MediumCount, &stats.LowCount,
		&stats.EstimatedLoss, &stats.PotentialSavings, &stats.TotalInventoryValue, &stats.TotalBatches,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load dashboard"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *RiskHandler) Assessments(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)

	if _, err := recalculateForPharmacy(c.Request.Context(), h.db, pharmacyID); err != nil {
		slog.Warn("risk recalc failed, returning cached data", "error", err)
	}

	riskLevel := c.Query("risk_level")

	query := `
		SELECT ra.id, ra.batch_id, ra.pharmacy_id, ra.risk_level, ra.days_until_expiry,
		       ra.avg_daily_sales, ra.expected_sales, ra.estimated_surplus, ra.estimated_loss,
		       ra.suggested_discount_percent, ra.calculated_at,
		       p.name as product_name, ib.batch_number, ib.expiry_date,
		       ib.current_quantity, ib.purchase_price
		FROM risk_assessments ra
		JOIN inventory_batches ib ON ib.id = ra.batch_id
		JOIN products p ON p.id = ib.product_id
		WHERE ra.pharmacy_id = $1
	`
	args := []interface{}{pharmacyID}
	if riskLevel != "" {
		query += " AND ra.risk_level = $2"
		args = append(args, riskLevel)
	}
	query += " ORDER BY CASE ra.risk_level WHEN 'EXPIRED' THEN 0 WHEN 'CRITICAL' THEN 1 WHEN 'HIGH' THEN 2 WHEN 'MEDIUM' THEN 3 ELSE 4 END, ra.days_until_expiry ASC LIMIT 200"

	rows, err := h.db.Query(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load assessments"})
		return
	}
	defer rows.Close()

	type AssessmentRow struct {
		ID                   uuid.UUID  `json:"id"`
		BatchID              uuid.UUID  `json:"batch_id"`
		PharmacyID           uuid.UUID  `json:"pharmacy_id"`
		RiskLevel            string     `json:"risk_level"`
		DaysUntilExpiry      int        `json:"days_until_expiry"`
		AvgDailySales        *float64   `json:"avg_daily_sales"`
		ExpectedSales        *int       `json:"expected_sales"`
		EstimatedSurplus     *int       `json:"estimated_surplus"`
		EstimatedLoss        *float64   `json:"estimated_loss"`
		SuggestedDiscountPct *int       `json:"suggested_discount_percent"`
		CalculatedAt         time.Time  `json:"calculated_at"`
		ProductName          string     `json:"product_name"`
		BatchNumber          *string    `json:"batch_number"`
		ExpiryDate           time.Time  `json:"expiry_date"`
		CurrentQuantity      int        `json:"current_quantity"`
		PurchasePrice        float64    `json:"purchase_price"`
	}

	var results []AssessmentRow
	for rows.Next() {
		var r AssessmentRow
		if err := rows.Scan(
			&r.ID, &r.BatchID, &r.PharmacyID, &r.RiskLevel, &r.DaysUntilExpiry,
			&r.AvgDailySales, &r.ExpectedSales, &r.EstimatedSurplus, &r.EstimatedLoss,
			&r.SuggestedDiscountPct, &r.CalculatedAt,
			&r.ProductName, &r.BatchNumber, &r.ExpiryDate, &r.CurrentQuantity, &r.PurchasePrice,
		); err != nil {
			continue
		}
		results = append(results, r)
	}
	if results == nil {
		results = []AssessmentRow{}
	}
	c.JSON(http.StatusOK, results)
}

func (h *RiskHandler) Recalculate(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)
	start := time.Now()
	n, err := recalculateForPharmacy(c.Request.Context(), h.db, pharmacyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "recalculation failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"recalculated": n,
		"duration_ms":  time.Since(start).Milliseconds(),
	})
}

func (h *RiskHandler) Timeline(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)

	rows, err := h.db.Query(c.Request.Context(), `
		SELECT
		  date_trunc('month', ib.expiry_date) as month,
		  COUNT(*) as batch_count,
		  SUM(ib.current_quantity * ib.purchase_price) as value
		FROM inventory_batches ib
		WHERE ib.pharmacy_id = $1
		  AND ib.expiry_date BETWEEN NOW() AND NOW() + INTERVAL '12 months'
		  AND ib.current_quantity > 0
		GROUP BY month
		ORDER BY month
	`, pharmacyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load timeline"})
		return
	}
	defer rows.Close()

	type TimelinePoint struct {
		Month      time.Time `json:"month"`
		BatchCount int       `json:"batch_count"`
		Value      float64   `json:"value"`
	}
	var points []TimelinePoint
	for rows.Next() {
		var p TimelinePoint
		if err := rows.Scan(&p.Month, &p.BatchCount, &p.Value); err != nil {
			continue
		}
		points = append(points, p)
	}
	if points == nil {
		points = []TimelinePoint{}
	}
	c.JSON(http.StatusOK, points)
}

func recalculateForPharmacy(ctx context.Context, db *pgxpool.Pool, pharmacyID uuid.UUID) (int, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// LEFT JOIN + GROUP BY instead of a per-row correlated subquery: one pass
	// over sales instead of one subquery execution per batch (was the main
	// cost of recalculation on pharmacies with a large sales history).
	rows, err := db.Query(ctx, `
		SELECT ib.id, ib.expiry_date, ib.current_quantity, ib.purchase_price,
		       COALESCE(SUM(s.quantity)::float / 90.0, 0.5) as avg_daily_sales
		FROM inventory_batches ib
		LEFT JOIN sales s ON s.batch_id = ib.id AND s.sale_date >= $2::date - 90
		WHERE ib.pharmacy_id = $1
		GROUP BY ib.id, ib.expiry_date, ib.current_quantity, ib.purchase_price
	`, pharmacyID, today)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type batchData struct {
		id            uuid.UUID
		expiryDate    time.Time
		currentQty    int
		purchasePrice float64
		avgDailySales float64
	}

	var batches []batchData
	for rows.Next() {
		var b batchData
		if err := rows.Scan(&b.id, &b.expiryDate, &b.currentQty, &b.purchasePrice, &b.avgDailySales); err != nil {
			continue
		}
		batches = append(batches, b)
	}
	rows.Close()

	if len(batches) == 0 {
		return 0, nil
	}

	batchIDs := make([]string, len(batches))
	pharmacyIDs := make([]string, len(batches))
	riskLevels := make([]string, len(batches))
	daysUntilExpiry := make([]int, len(batches))
	avgDailySales := make([]float64, len(batches))
	expectedSales := make([]int, len(batches))
	estimatedSurplus := make([]int, len(batches))
	estimatedLoss := make([]float64, len(batches))
	suggestedDiscount := make([]int, len(batches))

	pharmacyIDStr := pharmacyID.String()
	for i, b := range batches {
		result := services.CalculateRisk(services.RiskInput{
			CurrentQuantity: b.currentQty,
			ExpiryDate:      b.expiryDate,
			AvgDailySales:   b.avgDailySales,
			PurchasePrice:   b.purchasePrice,
			Today:           today,
		})

		batchIDs[i] = b.id.String()
		pharmacyIDs[i] = pharmacyIDStr
		riskLevels[i] = result.RiskLevel
		daysUntilExpiry[i] = result.DaysUntilExpiry
		avgDailySales[i] = b.avgDailySales
		expectedSales[i] = result.ExpectedSales
		estimatedSurplus[i] = result.EstimatedSurplus
		estimatedLoss[i] = result.EstimatedLoss
		suggestedDiscount[i] = result.SuggestedDiscountPct
	}

	// Single batched UPSERT (one row per batch) instead of one INSERT per batch —
	// avoids N sequential round-trips and unbounded table growth.
	tag, err := db.Exec(ctx, `
		INSERT INTO risk_assessments
		  (batch_id, pharmacy_id, risk_level, days_until_expiry, avg_daily_sales,
		   expected_sales, estimated_surplus, estimated_loss, suggested_discount_percent)
		SELECT batch_id::uuid, pharmacy_id::uuid, risk_level, days_until_expiry, avg_daily_sales,
		       expected_sales, estimated_surplus, estimated_loss, suggested_discount_percent
		FROM unnest(
			$1::text[], $2::text[], $3::text[], $4::int[], $5::float8[],
			$6::int[], $7::int[], $8::float8[], $9::int[]
		) AS t(batch_id, pharmacy_id, risk_level, days_until_expiry, avg_daily_sales,
		       expected_sales, estimated_surplus, estimated_loss, suggested_discount_percent)
		ON CONFLICT (batch_id) DO UPDATE SET
		  risk_level = EXCLUDED.risk_level,
		  days_until_expiry = EXCLUDED.days_until_expiry,
		  avg_daily_sales = EXCLUDED.avg_daily_sales,
		  expected_sales = EXCLUDED.expected_sales,
		  estimated_surplus = EXCLUDED.estimated_surplus,
		  estimated_loss = EXCLUDED.estimated_loss,
		  suggested_discount_percent = EXCLUDED.suggested_discount_percent,
		  calculated_at = NOW()
	`, batchIDs, pharmacyIDs, riskLevels, daysUntilExpiry, avgDailySales,
		expectedSales, estimatedSurplus, estimatedLoss, suggestedDiscount)
	if err != nil {
		return 0, err
	}

	return int(tag.RowsAffected()), nil
}
