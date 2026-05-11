package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"pharmasense/internal/middleware"
)

type ReportHandler struct {
	db *pgxpool.Pool
}

func NewReportHandler(db *pgxpool.Pool) *ReportHandler {
	return &ReportHandler{db: db}
}

func (h *ReportHandler) Savings(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)

	rows, err := h.db.Query(c.Request.Context(), `
		SELECT
		  date_trunc('month', aa.created_at) as month,
		  COUNT(*) as actions_taken,
		  COALESCE(SUM(ra.estimated_loss * (aa.discount_percent::float/100)), 0) as savings
		FROM alert_actions aa
		JOIN risk_assessments ra ON ra.batch_id = aa.batch_id
		WHERE aa.pharmacy_id = $1 AND aa.action_type = 'discount'
		GROUP BY month
		ORDER BY month DESC
		LIMIT 12
	`, pharmacyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load savings"})
		return
	}
	defer rows.Close()

	type SavingsPoint struct {
		Month       time.Time `json:"month"`
		ActionsTaken int      `json:"actions_taken"`
		Savings     float64   `json:"savings"`
	}
	var points []SavingsPoint
	for rows.Next() {
		var p SavingsPoint
		if err := rows.Scan(&p.Month, &p.ActionsTaken, &p.Savings); err != nil {
			continue
		}
		points = append(points, p)
	}
	if points == nil {
		points = []SavingsPoint{}
	}
	c.JSON(http.StatusOK, points)
}

func (h *ReportHandler) Waste(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)

	rows, err := h.db.Query(c.Request.Context(), `
		SELECT
		  date_trunc('month', ib.expiry_date) as month,
		  COUNT(*) as expired_batches,
		  COALESCE(SUM(ib.current_quantity * ib.purchase_price), 0) as waste_value
		FROM inventory_batches ib
		WHERE ib.pharmacy_id = $1
		  AND ib.expiry_date < NOW()
		  AND ib.current_quantity > 0
		GROUP BY month
		ORDER BY month DESC
		LIMIT 12
	`, pharmacyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load waste report"})
		return
	}
	defer rows.Close()

	type WastePoint struct {
		Month          time.Time `json:"month"`
		ExpiredBatches int       `json:"expired_batches"`
		WasteValue     float64   `json:"waste_value"`
	}
	var points []WastePoint
	for rows.Next() {
		var p WastePoint
		if err := rows.Scan(&p.Month, &p.ExpiredBatches, &p.WasteValue); err != nil {
			continue
		}
		points = append(points, p)
	}
	if points == nil {
		points = []WastePoint{}
	}
	c.JSON(http.StatusOK, points)
}

func (h *ReportHandler) Categories(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)

	rows, err := h.db.Query(c.Request.Context(), `
		SELECT
		  COALESCE(p.category, 'Uncategorized') as category,
		  COUNT(ib.id) as batch_count,
		  COUNT(*) FILTER (WHERE ra.risk_level IN ('CRITICAL','HIGH')) as at_risk_count,
		  COALESCE(SUM(ra.estimated_loss), 0) as total_loss
		FROM inventory_batches ib
		JOIN products p ON p.id = ib.product_id
		LEFT JOIN LATERAL (
			SELECT risk_level, estimated_loss
			FROM risk_assessments
			WHERE batch_id = ib.id
			ORDER BY calculated_at DESC
			LIMIT 1
		) ra ON true
		WHERE ib.pharmacy_id = $1
		GROUP BY category
		ORDER BY total_loss DESC
	`, pharmacyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load category report"})
		return
	}
	defer rows.Close()

	type CategoryStat struct {
		Category   string  `json:"category"`
		BatchCount int     `json:"batch_count"`
		AtRiskCount int    `json:"at_risk_count"`
		TotalLoss  float64 `json:"total_loss"`
	}
	var stats []CategoryStat
	for rows.Next() {
		var s CategoryStat
		if err := rows.Scan(&s.Category, &s.BatchCount, &s.AtRiskCount, &s.TotalLoss); err != nil {
			continue
		}
		stats = append(stats, s)
	}
	if stats == nil {
		stats = []CategoryStat{}
	}
	c.JSON(http.StatusOK, stats)
}
