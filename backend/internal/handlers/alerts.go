package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"pharmasense/internal/middleware"
)

type AlertHandler struct {
	db *pgxpool.Pool
}

func NewAlertHandler(db *pgxpool.Pool) *AlertHandler {
	return &AlertHandler{db: db}
}

func (h *AlertHandler) List(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)
	riskLevel := c.Query("risk_level")

	query := `
		SELECT ra.id, ra.batch_id, ra.risk_level, ra.days_until_expiry,
		       ra.estimated_loss, ra.suggested_discount_percent,
		       p.name as product_name, p.category,
		       ib.batch_number, ib.expiry_date, ib.current_quantity, ib.purchase_price
		FROM risk_assessments ra
		JOIN inventory_batches ib ON ib.id = ra.batch_id
		JOIN products p ON p.id = ib.product_id
		WHERE ra.pharmacy_id = $1 AND ib.current_quantity > 0
	`
	args := []interface{}{pharmacyID}
	if riskLevel != "" && riskLevel != "ALL" {
		query += " AND ra.risk_level = $2"
		args = append(args, riskLevel)
	}
	query += " ORDER BY CASE ra.risk_level WHEN 'CRITICAL' THEN 1 WHEN 'HIGH' THEN 2 WHEN 'MEDIUM' THEN 3 ELSE 4 END, ra.days_until_expiry ASC LIMIT 200"

	rows, err := h.db.Query(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load alerts"})
		return
	}
	defer rows.Close()

	type AlertRow struct {
		AssessmentID    uuid.UUID `json:"assessment_id"`
		BatchID         uuid.UUID `json:"batch_id"`
		RiskLevel       string    `json:"risk_level"`
		DaysUntilExpiry int       `json:"days_until_expiry"`
		EstimatedLoss   *float64  `json:"estimated_loss"`
		SuggestedDisc   *int      `json:"suggested_discount_percent"`
		ProductName     string    `json:"product_name"`
		Category        *string   `json:"category"`
		BatchNumber     *string   `json:"batch_number"`
		ExpiryDate      string    `json:"expiry_date"`
		CurrentQty      int       `json:"current_quantity"`
		PurchasePrice   float64   `json:"purchase_price"`
	}

	var alerts []AlertRow
	for rows.Next() {
		var a AlertRow
		if err := rows.Scan(
			&a.AssessmentID, &a.BatchID, &a.RiskLevel, &a.DaysUntilExpiry,
			&a.EstimatedLoss, &a.SuggestedDisc,
			&a.ProductName, &a.Category, &a.BatchNumber, &a.ExpiryDate,
			&a.CurrentQty, &a.PurchasePrice,
		); err != nil {
			continue
		}
		alerts = append(alerts, a)
	}
	if alerts == nil {
		alerts = []AlertRow{}
	}
	c.JSON(http.StatusOK, alerts)
}

type alertActionRequest struct {
	ActionType     string  `json:"action_type"     binding:"required"`
	DiscountPct    *int    `json:"discount_percent"`
	Notes          *string `json:"notes"`
}

func (h *AlertHandler) Action(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)
	userID := middleware.GetUserID(c)

	batchID, err := uuid.Parse(c.Param("batch_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid batch_id"})
		return
	}

	var req alertActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify batch belongs to pharmacy
	var exists bool
	_ = h.db.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM inventory_batches WHERE id = $1 AND pharmacy_id = $2)`,
		batchID, pharmacyID,
	).Scan(&exists)
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "batch not found"})
		return
	}

	var id uuid.UUID
	err = h.db.QueryRow(c.Request.Context(), `
		INSERT INTO alert_actions (batch_id, pharmacy_id, user_id, action_type, discount_percent, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, batchID, pharmacyID, userID, req.ActionType, req.DiscountPct, req.Notes).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record action"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "action recorded"})
}
