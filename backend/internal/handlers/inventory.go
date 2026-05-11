package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"pharmasense/internal/middleware"
)

type InventoryHandler struct {
	db *pgxpool.Pool
}

func NewInventoryHandler(db *pgxpool.Pool) *InventoryHandler {
	return &InventoryHandler{db: db}
}

func (h *InventoryHandler) List(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)
	riskFilter := c.Query("risk_level")
	search := c.Query("search")
	limit := 100
	offset := 0
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 500 {
		limit = l
	}
	if o, err := strconv.Atoi(c.Query("offset")); err == nil && o >= 0 {
		offset = o
	}

	query := `
		SELECT ib.id, ib.pharmacy_id, ib.product_id, ib.batch_number, ib.expiry_date,
		       ib.initial_quantity, ib.current_quantity, ib.purchase_price, ib.selling_price,
		       ib.supplier, ib.received_date, ib.created_at, ib.updated_at,
		       p.name as product_name, p.category, p.manufacturer,
		       ra.risk_level, ra.days_until_expiry, ra.estimated_loss, ra.suggested_discount_percent
		FROM inventory_batches ib
		JOIN products p ON p.id = ib.product_id
		LEFT JOIN LATERAL (
			SELECT risk_level, days_until_expiry, estimated_loss, suggested_discount_percent
			FROM risk_assessments
			WHERE batch_id = ib.id
			ORDER BY calculated_at DESC
			LIMIT 1
		) ra ON true
		WHERE ib.pharmacy_id = $1
	`
	args := []interface{}{pharmacyID}
	argIdx := 2

	if riskFilter != "" {
		query += fmt.Sprintf(" AND ra.risk_level = $%d", argIdx)
		args = append(args, riskFilter)
		argIdx++
	}
	if search != "" {
		query += fmt.Sprintf(" AND p.name ILIKE $%d", argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY ib.expiry_date ASC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := h.db.Query(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list inventory"})
		return
	}
	defer rows.Close()

	type BatchRow struct {
		ID              uuid.UUID  `json:"id"`
		PharmacyID      uuid.UUID  `json:"pharmacy_id"`
		ProductID       uuid.UUID  `json:"product_id"`
		BatchNumber     *string    `json:"batch_number"`
		ExpiryDate      time.Time  `json:"expiry_date"`
		InitialQuantity int        `json:"initial_quantity"`
		CurrentQuantity int        `json:"current_quantity"`
		PurchasePrice   float64    `json:"purchase_price"`
		SellingPrice    float64    `json:"selling_price"`
		Supplier        *string    `json:"supplier"`
		ReceivedDate    time.Time  `json:"received_date"`
		CreatedAt       time.Time  `json:"created_at"`
		UpdatedAt       time.Time  `json:"updated_at"`
		ProductName     string     `json:"product_name"`
		Category        *string    `json:"category"`
		Manufacturer    *string    `json:"manufacturer"`
		RiskLevel       *string    `json:"risk_level"`
		DaysUntilExpiry *int       `json:"days_until_expiry"`
		EstimatedLoss   *float64   `json:"estimated_loss"`
		SuggestedDisc   *int       `json:"suggested_discount_percent"`
	}

	var batches []BatchRow
	for rows.Next() {
		var b BatchRow
		if err := rows.Scan(
			&b.ID, &b.PharmacyID, &b.ProductID, &b.BatchNumber, &b.ExpiryDate,
			&b.InitialQuantity, &b.CurrentQuantity, &b.PurchasePrice, &b.SellingPrice,
			&b.Supplier, &b.ReceivedDate, &b.CreatedAt, &b.UpdatedAt,
			&b.ProductName, &b.Category, &b.Manufacturer,
			&b.RiskLevel, &b.DaysUntilExpiry, &b.EstimatedLoss, &b.SuggestedDisc,
		); err != nil {
			continue
		}
		batches = append(batches, b)
	}
	if batches == nil {
		batches = []BatchRow{}
	}
	c.JSON(http.StatusOK, gin.H{"data": batches, "total": len(batches)})
}

type createBatchRequest struct {
	ProductID       uuid.UUID `json:"product_id"       binding:"required"`
	BatchNumber     *string   `json:"batch_number"`
	ExpiryDate      string    `json:"expiry_date"      binding:"required"`
	InitialQuantity int       `json:"initial_quantity" binding:"required,min=1"`
	CurrentQuantity int       `json:"current_quantity" binding:"required,min=0"`
	PurchasePrice   float64   `json:"purchase_price"   binding:"required,min=0"`
	SellingPrice    float64   `json:"selling_price"    binding:"required,min=0"`
	Supplier        *string   `json:"supplier"`
	ReceivedDate    string    `json:"received_date"    binding:"required"`
}

func (h *InventoryHandler) Create(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)
	var req createBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	expiryDate, err := time.Parse("2006-01-02", req.ExpiryDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expiry_date format (use YYYY-MM-DD)"})
		return
	}
	receivedDate, err := time.Parse("2006-01-02", req.ReceivedDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid received_date format (use YYYY-MM-DD)"})
		return
	}

	var id uuid.UUID
	err = h.db.QueryRow(c.Request.Context(), `
		INSERT INTO inventory_batches
		  (pharmacy_id, product_id, batch_number, expiry_date, initial_quantity, current_quantity,
		   purchase_price, selling_price, supplier, received_date)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id
	`, pharmacyID, req.ProductID, req.BatchNumber, expiryDate, req.InitialQuantity,
		req.CurrentQuantity, req.PurchasePrice, req.SellingPrice, req.Supplier, receivedDate,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create batch"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "batch created"})
}

func (h *InventoryHandler) Delete(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)
	batchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid batch id"})
		return
	}

	result, err := h.db.Exec(c.Request.Context(), `
		DELETE FROM inventory_batches WHERE id = $1 AND pharmacy_id = $2
	`, batchID, pharmacyID)
	if err != nil || result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "batch not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *InventoryHandler) Import(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "csv file required"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid csv"})
		return
	}

	if len(records) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "csv must have header + at least one row"})
		return
	}

	imported := 0
	errors := []string{}

	for i, row := range records[1:] {
		lineNum := i + 2
		if len(row) < 7 {
			errors = append(errors, fmt.Sprintf("row %d: not enough columns", lineNum))
			continue
		}

		productName := strings.TrimSpace(row[0])
		batchNum := strings.TrimSpace(row[1])
		expiryStr := strings.TrimSpace(row[2])
		qtyStr := strings.TrimSpace(row[3])
		purchasePriceStr := strings.TrimSpace(row[4])
		sellingPriceStr := strings.TrimSpace(row[5])
		receivedStr := strings.TrimSpace(row[6])

		expiry, err := time.Parse("2006-01-02", expiryStr)
		if err != nil {
			errors = append(errors, fmt.Sprintf("row %d: invalid expiry_date %s", lineNum, expiryStr))
			continue
		}
		received, err := time.Parse("2006-01-02", receivedStr)
		if err != nil {
			errors = append(errors, fmt.Sprintf("row %d: invalid received_date %s", lineNum, receivedStr))
			continue
		}
		qty, err := strconv.Atoi(qtyStr)
		if err != nil {
			errors = append(errors, fmt.Sprintf("row %d: invalid quantity", lineNum))
			continue
		}
		purchasePrice, err := strconv.ParseFloat(purchasePriceStr, 64)
		if err != nil {
			errors = append(errors, fmt.Sprintf("row %d: invalid purchase_price", lineNum))
			continue
		}
		sellingPrice, err := strconv.ParseFloat(sellingPriceStr, 64)
		if err != nil {
			errors = append(errors, fmt.Sprintf("row %d: invalid selling_price", lineNum))
			continue
		}

		// Find or create product
		var productID uuid.UUID
		err = h.db.QueryRow(c.Request.Context(), `
			INSERT INTO products (name) VALUES ($1)
			ON CONFLICT (barcode) DO NOTHING
			RETURNING id
		`, productName).Scan(&productID)
		if err != nil {
			// Try to find existing product by name
			err = h.db.QueryRow(c.Request.Context(), `SELECT id FROM products WHERE name = $1`, productName).Scan(&productID)
			if err != nil {
				// Create without barcode uniqueness
				err = h.db.QueryRow(c.Request.Context(), `INSERT INTO products (name) VALUES ($1) RETURNING id`, productName).Scan(&productID)
				if err != nil {
					errors = append(errors, fmt.Sprintf("row %d: failed to create product", lineNum))
					continue
				}
			}
		}

		_, err = h.db.Exec(c.Request.Context(), `
			INSERT INTO inventory_batches
			  (pharmacy_id, product_id, batch_number, expiry_date, initial_quantity, current_quantity,
			   purchase_price, selling_price, received_date)
			VALUES ($1,$2,$3,$4,$5,$5,$6,$7,$8)
		`, pharmacyID, productID, batchNum, expiry, qty, purchasePrice, sellingPrice, received)
		if err != nil {
			errors = append(errors, fmt.Sprintf("row %d: failed to insert batch", lineNum))
			continue
		}
		imported++
	}

	c.JSON(http.StatusOK, gin.H{
		"imported": imported,
		"errors":   errors,
		"total":    len(records) - 1,
	})
}
