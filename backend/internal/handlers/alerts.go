package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"pharmasense/internal/middleware"
	"pharmasense/internal/services/notifications"
)

type AlertHandler struct {
	db    *pgxpool.Pool
	email *notifications.EmailService
}

func NewAlertHandler(db *pgxpool.Pool, email *notifications.EmailService) *AlertHandler {
	return &AlertHandler{db: db, email: email}
}

func (h *AlertHandler) List(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)

	if _, err := recalculateForPharmacy(c.Request.Context(), h.db, pharmacyID); err != nil {
		slog.Warn("risk recalc failed, returning cached data", "error", err)
	}

	riskLevel := c.Query("risk_level")

	baseQuery := `
		SELECT ra.id, ib.id as batch_id, ra.risk_level, ra.days_until_expiry,
		       ra.estimated_loss, ra.suggested_discount_percent,
		       p.name as product_name, p.category,
		       ib.batch_number, ib.expiry_date, ib.current_quantity, ib.purchase_price,
		       ib.supplier_email
		FROM inventory_batches ib
		JOIN products p ON p.id = ib.product_id
		LEFT JOIN LATERAL (
			SELECT id, risk_level, days_until_expiry, estimated_loss, suggested_discount_percent
			FROM risk_assessments
			WHERE batch_id = ib.id
			ORDER BY calculated_at DESC
			LIMIT 1
		) ra ON true
		WHERE ib.pharmacy_id = $1 AND ib.current_quantity > 0 AND ra.risk_level IS NOT NULL
	`
	args := []interface{}{pharmacyID}
	query := baseQuery
	if riskLevel != "" && riskLevel != "ALL" {
		query += " AND ra.risk_level = $2"
		args = append(args, riskLevel)
	}
	query += " ORDER BY CASE ra.risk_level WHEN 'EXPIRED' THEN 0 WHEN 'CRITICAL' THEN 1 WHEN 'HIGH' THEN 2 WHEN 'MEDIUM' THEN 3 ELSE 4 END, ra.days_until_expiry ASC LIMIT 200"

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
		ExpiryDate      time.Time `json:"expiry_date"`
		CurrentQty      int       `json:"current_quantity"`
		PurchasePrice   float64   `json:"purchase_price"`
		SupplierEmail   *string   `json:"supplier_email"`
	}

	var alerts []AlertRow
	for rows.Next() {
		var a AlertRow
		if err := rows.Scan(
			&a.AssessmentID, &a.BatchID, &a.RiskLevel, &a.DaysUntilExpiry,
			&a.EstimatedLoss, &a.SuggestedDisc,
			&a.ProductName, &a.Category, &a.BatchNumber, &a.ExpiryDate,
			&a.CurrentQty, &a.PurchasePrice, &a.SupplierEmail,
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
	ActionType    string  `json:"action_type"     binding:"required"`
	Quantity      *int    `json:"quantity"`
	DiscountPct   *int    `json:"discount_percent"`
	Notes         *string `json:"notes"`
	SupplierEmail *string `json:"supplier_email"`
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

	// Fetch batch (verify ownership + get current_quantity and purchase_price)
	var currentQty int
	var purchasePrice float64
	err = h.db.QueryRow(c.Request.Context(),
		`SELECT current_quantity, purchase_price FROM inventory_batches WHERE id = $1 AND pharmacy_id = $2`,
		batchID, pharmacyID,
	).Scan(&currentQty, &purchasePrice)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "batch not found"})
		return
	}

	if strings.EqualFold(req.ActionType, "dispose") {
		// Zero out the remaining stock rather than hard-deleting the row:
		// a batch that already has sales recorded against it can't be
		// deleted (sales.batch_id has no ON DELETE CASCADE), and that's the
		// common case in practice — most expired batches sold some units
		// before the rest expired. Zeroing current_quantity always
		// succeeds and already removes the batch from Inventory/Alerts,
		// since both list it only where current_quantity > 0.
		tx, err := h.db.Begin(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
			return
		}
		defer tx.Rollback(c.Request.Context())

		_, err = tx.Exec(c.Request.Context(),
			`UPDATE inventory_batches SET current_quantity = 0, updated_at = NOW() WHERE id = $1 AND pharmacy_id = $2`,
			batchID, pharmacyID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to dispose batch"})
			return
		}

		if _, err := tx.Exec(c.Request.Context(), `
			INSERT INTO alert_actions (batch_id, pharmacy_id, user_id, action_type, quantity, notes)
			VALUES ($1, $2, $3, 'dispose', $4, $5)
		`, batchID, pharmacyID, userID, currentQty, req.Notes); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record action"})
			return
		}

		if err := tx.Commit(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "batch disposed and removed from inventory", "quantity": currentQty})
		return
	}

	if strings.EqualFold(req.ActionType, "RETURN") {
		// Default to returning the full quantity if not specified
		qty := currentQty
		if req.Quantity != nil {
			qty = *req.Quantity
		}
		if qty <= 0 || qty > currentQty {
			c.JSON(http.StatusBadRequest, gin.H{"error": "quantity must be between 1 and current stock"})
			return
		}

		tx, err := h.db.Begin(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
			return
		}
		defer tx.Rollback(c.Request.Context())

		_, err = tx.Exec(c.Request.Context(),
			`UPDATE inventory_batches SET current_quantity = current_quantity - $1, updated_at = NOW() WHERE id = $2`,
			qty, batchID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update inventory"})
			return
		}

		var id uuid.UUID
		err = tx.QueryRow(c.Request.Context(), `
			INSERT INTO alert_actions (batch_id, pharmacy_id, user_id, action_type, quantity, discount_percent, notes)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id
		`, batchID, pharmacyID, userID, req.ActionType, qty, req.DiscountPct, req.Notes).Scan(&id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record action"})
			return
		}

		if err := tx.Commit(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit"})
			return
		}

		refundValue := float64(qty) * purchasePrice

		// The return is now committed. Notify the supplier by email as a
		// best-effort step: a send failure must NOT roll back the return —
		// we surface it to the client as a warning instead.
		resp := gin.H{
			"id":           id,
			"message":      "action recorded",
			"refund_value": refundValue,
			"quantity":     qty,
			"email_sent":   false,
		}
		supplierEmail := ""
		if req.SupplierEmail != nil {
			supplierEmail = strings.TrimSpace(*req.SupplierEmail)
		}
		if supplierEmail != "" {
			if err := h.sendSupplierReturnEmail(c.Request.Context(), batchID, pharmacyID, supplierEmail, qty, refundValue, req.Notes); err != nil {
				slog.Error("return: supplier email failed", "batch_id", batchID, "err", err)
				resp["email_error"] = "Return recorded, but the supplier email could not be sent."
			} else {
				resp["email_sent"] = true
			}
		}

		c.JSON(http.StatusCreated, resp)
		return
	}

	// Non-RETURN actions — simple insert
	var id uuid.UUID
	err = h.db.QueryRow(c.Request.Context(), `
		INSERT INTO alert_actions (batch_id, pharmacy_id, user_id, action_type, quantity, discount_percent, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, batchID, pharmacyID, userID, req.ActionType, req.Quantity, req.DiscountPct, req.Notes).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record action"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "action recorded"})
}

// sendSupplierReturnEmail loads the product/batch/pharmacy names for a returned
// batch and emails the supplier a return notification via Resend.
func (h *AlertHandler) sendSupplierReturnEmail(ctx context.Context, batchID, pharmacyID uuid.UUID, to string, qty int, refund float64, notes *string) error {
	var productName, pharmacyName string
	var batchNumber *string
	if err := h.db.QueryRow(ctx, `
		SELECT pr.name, ib.batch_number, ph.name
		FROM inventory_batches ib
		JOIN products pr ON pr.id = ib.product_id
		JOIN pharmacies ph ON ph.id = ib.pharmacy_id
		WHERE ib.id = $1 AND ib.pharmacy_id = $2
	`, batchID, pharmacyID).Scan(&productName, &batchNumber, &pharmacyName); err != nil {
		return fmt.Errorf("load return email data: %w", err)
	}

	data := notifications.SupplierReturnData{
		PharmacyName: pharmacyName,
		ProductName:  productName,
		Quantity:     qty,
		RefundValue:  refund,
	}
	if batchNumber != nil {
		data.BatchNumber = *batchNumber
	}
	if notes != nil {
		data.Notes = *notes
	}
	return h.email.SendSupplierReturn(ctx, to, data)
}
