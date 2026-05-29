package handlers

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
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

// ── List ──────────────────────────────────────────────────────────────────────

func (h *InventoryHandler) List(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)

	if _, err := recalculateForPharmacy(c.Request.Context(), h.db, pharmacyID); err != nil {
		slog.Warn("risk recalc failed, returning cached data", "error", err)
	}

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
		ID              uuid.UUID `json:"id"`
		PharmacyID      uuid.UUID `json:"pharmacy_id"`
		ProductID       uuid.UUID `json:"product_id"`
		BatchNumber     *string   `json:"batch_number"`
		ExpiryDate      time.Time `json:"expiry_date"`
		InitialQuantity int       `json:"initial_quantity"`
		CurrentQuantity int       `json:"current_quantity"`
		PurchasePrice   float64   `json:"purchase_price"`
		SellingPrice    float64   `json:"selling_price"`
		Supplier        *string   `json:"supplier"`
		ReceivedDate    time.Time `json:"received_date"`
		CreatedAt       time.Time `json:"created_at"`
		UpdatedAt       time.Time `json:"updated_at"`
		ProductName     string    `json:"product_name"`
		Category        *string   `json:"category"`
		Manufacturer    *string   `json:"manufacturer"`
		RiskLevel       *string   `json:"risk_level"`
		DaysUntilExpiry *int      `json:"days_until_expiry"`
		EstimatedLoss   *float64  `json:"estimated_loss"`
		SuggestedDisc   *int      `json:"suggested_discount_percent"`
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

// ── Create (single batch) ─────────────────────────────────────────────────────

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

// ── Delete ────────────────────────────────────────────────────────────────────

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

// ── CSV Import ────────────────────────────────────────────────────────────────

func (h *InventoryHandler) Import(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "csv file required"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read file"})
		return
	}

	batches, total, parseErrors := parseBatchCSV(data)

	// If we have only fatal errors (missing columns, empty file), return 400
	if total == 0 && len(parseErrors) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": parseErrors[0]})
		return
	}

	imported := 0
	var insertErrors []string

	for _, b := range batches {
		productID, err := h.findOrCreateProduct(c.Request.Context(), b)
		if err != nil {
			insertErrors = append(insertErrors, fmt.Sprintf("row %d: failed to save product '%s': %v", b.RowNum, b.ProductName, err))
			continue
		}

		var batchNum *string
		if b.BatchNumber != "" {
			batchNum = &b.BatchNumber
		}
		var supplier *string
		if b.Supplier != "" {
			supplier = &b.Supplier
		}

		_, err = h.db.Exec(c.Request.Context(), `
			INSERT INTO inventory_batches
			  (pharmacy_id, product_id, batch_number, expiry_date, initial_quantity, current_quantity,
			   purchase_price, selling_price, supplier, received_date)
			VALUES ($1,$2,$3,$4,$5,$5,$6,$7,$8,$9)
		`, pharmacyID, productID, batchNum, b.ExpiryDate, b.Quantity,
			b.PurchasePrice, b.SellingPrice, supplier, b.ReceivedDate)
		if err != nil {
			insertErrors = append(insertErrors, fmt.Sprintf("row %d: failed to import '%s': %v", b.RowNum, b.ProductName, err))
			continue
		}
		imported++
	}

	allErrors := append(parseErrors, insertErrors...)
	if allErrors == nil {
		allErrors = []string{}
	}

	c.JSON(http.StatusOK, gin.H{
		"imported": imported,
		"errors":   allErrors,
		"total":    total,
	})
}

// findOrCreateProduct looks up a product by barcode or name, creating it if absent.
func (h *InventoryHandler) findOrCreateProduct(ctx context.Context, b parsedBatch) (uuid.UUID, error) {
	var productID uuid.UUID

	// 1. Match by barcode (most reliable)
	if b.Barcode != "" {
		if err := h.db.QueryRow(ctx, `SELECT id FROM products WHERE barcode = $1`, b.Barcode).Scan(&productID); err == nil {
			return productID, nil
		}
	}

	// 2. Match by exact product name
	if err := h.db.QueryRow(ctx, `SELECT id FROM products WHERE name = $1`, b.ProductName).Scan(&productID); err == nil {
		return productID, nil
	}

	// 3. Insert new product
	var barcode, nameEl, category, manufacturer *string
	if b.Barcode != "" {
		barcode = &b.Barcode
	}
	if b.ProductNameEl != "" {
		nameEl = &b.ProductNameEl
	}
	if b.Category != "" {
		category = &b.Category
	}
	if b.Manufacturer != "" {
		manufacturer = &b.Manufacturer
	}

	err := h.db.QueryRow(ctx, `
		INSERT INTO products (barcode, name, name_el, category, manufacturer)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, barcode, b.ProductName, nameEl, category, manufacturer).Scan(&productID)
	if err != nil {
		// Barcode conflict: race condition or duplicate — fall back to a lookup
		if b.Barcode != "" {
			if err2 := h.db.QueryRow(ctx, `SELECT id FROM products WHERE barcode = $1`, b.Barcode).Scan(&productID); err2 == nil {
				return productID, nil
			}
		}
		return uuid.UUID{}, fmt.Errorf("insert product: %w", err)
	}
	return productID, nil
}

// ── CSV parsing helpers ───────────────────────────────────────────────────────

// parsedBatch holds one validated row from the CSV before DB insertion.
type parsedBatch struct {
	ProductName   string
	ProductNameEl string
	Barcode       string
	BatchNumber   string
	ExpiryDate    time.Time
	Quantity      int
	PurchasePrice float64
	SellingPrice  float64
	ReceivedDate  time.Time
	Supplier      string
	Category      string
	Manufacturer  string
	RowNum        int
}

// dateFormats is the ordered list of formats tried for every date field.
var dateFormats = []string{
	"2006-01-02",     // ISO 8601:         2026-05-30
	"02/01/2006",     // European:         30/05/2026
	"02-01-2006",     // European dashes:  30-05-2026
	"01/02/2006",     // US:               05/30/2026
	"2006/01/02",     // Reverse ISO:      2026/05/30
	"2 January 2006", // Long:             30 May 2026
	"02 Jan 2006",    // Short month:      30 May 2026
	"2 Jan 2006",     // Short no-zero:    3 Jun 2026
	"Jan 2 2006",     // US long:          May 30 2026
	time.RFC3339,     // ISO w/ time:      2026-05-30T00:00:00Z
}

func parseDate(value string) (time.Time, error) {
	v := strings.TrimSpace(value)
	for _, format := range dateFormats {
		if t, err := time.Parse(format, v); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse %q (accepted: YYYY-MM-DD, DD/MM/YYYY, DD-MM-YYYY)", v)
}

// normalizeHeader lowercases and replaces spaces/dashes with underscores.
func normalizeHeader(h string) string {
	h = strings.TrimSpace(h)
	h = strings.ToLower(h)
	h = strings.ReplaceAll(h, " ", "_")
	h = strings.ReplaceAll(h, "-", "_")
	return h
}

// colIndices holds the resolved column index (-1 = not present) for each field.
type colIndices struct {
	productName   int
	nameEl        int
	barcode       int
	batchNumber   int
	expiryDate    int
	quantity      int
	purchasePrice int
	sellingPrice  int
	receivedDate  int
	supplier      int
	category      int
	manufacturer  int
}

// aliasToCanonical maps every accepted alias → canonical field name.
var aliasToCanonical = map[string]string{
	// product name
	"product_name": "product_name", "name": "product_name", "product": "product_name",
	"drug_name": "product_name", "item": "product_name",
	// Greek name
	"name_el": "name_el", "greek_name": "name_el", "name_greek": "name_el",
	// barcode
	"barcode": "barcode", "ean": "barcode", "gtin": "barcode", "upc": "barcode", "code": "barcode",
	// batch
	"batch_number": "batch_number", "batch": "batch_number", "lot": "batch_number",
	"lot_number": "batch_number", "batch_no": "batch_number",
	// expiry
	"expiry_date": "expiry_date", "expiry": "expiry_date", "exp_date": "expiry_date",
	"exp": "expiry_date", "expires": "expiry_date", "best_before": "expiry_date",
	// quantity
	"quantity": "quantity", "qty": "quantity", "stock": "quantity",
	"amount": "quantity", "count": "quantity",
	// purchase price
	"purchase_price": "purchase_price", "cost": "purchase_price",
	"buy_price": "purchase_price", "wholesale_price": "purchase_price", "unit_cost": "purchase_price",
	// selling price
	"selling_price": "selling_price", "sell_price": "selling_price",
	"retail_price": "selling_price", "price": "selling_price",
	// received date
	"received_date": "received_date", "received": "received_date",
	"purchase_date": "received_date", "receipt_date": "received_date", "date_received": "received_date",
	// supplier
	"supplier": "supplier", "vendor": "supplier", "distributor": "supplier",
	// category
	"category": "category", "cat": "category", "type": "category",
	// manufacturer
	"manufacturer": "manufacturer", "mfr": "manufacturer", "brand": "manufacturer", "maker": "manufacturer",
}

func resolveColumns(headerMap map[string]int) colIndices {
	idx := colIndices{
		productName: -1, nameEl: -1, barcode: -1, batchNumber: -1,
		expiryDate: -1, quantity: -1, purchasePrice: -1, sellingPrice: -1,
		receivedDate: -1, supplier: -1, category: -1, manufacturer: -1,
	}
	for header, colIdx := range headerMap {
		canonical, ok := aliasToCanonical[header]
		if !ok {
			continue
		}
		switch canonical {
		case "product_name":   idx.productName = colIdx
		case "name_el":        idx.nameEl = colIdx
		case "barcode":        idx.barcode = colIdx
		case "batch_number":   idx.batchNumber = colIdx
		case "expiry_date":    idx.expiryDate = colIdx
		case "quantity":       idx.quantity = colIdx
		case "purchase_price": idx.purchasePrice = colIdx
		case "selling_price":  idx.sellingPrice = colIdx
		case "received_date":  idx.receivedDate = colIdx
		case "supplier":       idx.supplier = colIdx
		case "category":       idx.category = colIdx
		case "manufacturer":   idx.manufacturer = colIdx
		}
	}
	return idx
}

// parseBatchCSV reads CSV bytes and returns (valid rows, total data rows, row errors).
// It never returns fatal errors — only per-row problems. Caller checks total==0&&len(errors)>0
// to distinguish "bad file" from "bad rows".
func parseBatchCSV(data []byte) (batches []parsedBatch, total int, errors []string) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // allow ragged rows

	records, err := reader.ReadAll()
	if err != nil {
		return nil, 0, []string{"invalid CSV: " + err.Error()}
	}
	if len(records) < 1 {
		return nil, 0, []string{"CSV file is empty"}
	}

	// Strip UTF-8 BOM from first header cell (Excel artefact: EF BB BF)
	records[0][0] = strings.TrimPrefix(records[0][0], "\xef\xbb\xbf")

	// Build header → column-index map
	headerMap := make(map[string]int, len(records[0]))
	for i, h := range records[0] {
		headerMap[normalizeHeader(h)] = i
	}

	cols := resolveColumns(headerMap)

	// Validate required columns
	var missing []string
	if cols.productName < 0 {
		missing = append(missing, "product_name (or: name, item)")
	}
	if cols.expiryDate < 0 {
		missing = append(missing, "expiry_date (or: expiry, exp_date, expires)")
	}
	if cols.quantity < 0 {
		missing = append(missing, "quantity (or: qty, stock)")
	}
	if cols.purchasePrice < 0 {
		missing = append(missing, "purchase_price (or: cost, buy_price)")
	}
	if cols.sellingPrice < 0 {
		missing = append(missing, "selling_price (or: price, retail_price)")
	}
	if len(missing) > 0 {
		return nil, 0, []string{"missing required columns: " + strings.Join(missing, "; ")}
	}

	dataRows := records[1:]
	total = len(dataRows)
	today := time.Now().Truncate(24 * time.Hour)

	for i, row := range dataRows {
		lineNum := i + 2

		get := func(colIdx int) string {
			if colIdx < 0 || colIdx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[colIdx])
		}

		productName := get(cols.productName)
		if productName == "" {
			errors = append(errors, fmt.Sprintf("row %d: product_name is empty", lineNum))
			continue
		}

		// ── expiry date ────────────────────────────────────────────
		expiryStr := get(cols.expiryDate)
		expiry, err := parseDate(expiryStr)
		if err != nil {
			errors = append(errors, fmt.Sprintf(
				"row %d: invalid expiry_date '%s' for '%s' — use YYYY-MM-DD or DD/MM/YYYY",
				lineNum, expiryStr, productName,
			))
			continue
		}

		// ── received date (optional, defaults to today) ────────────
		receivedDate := today
		if receivedStr := get(cols.receivedDate); receivedStr != "" {
			t, err := parseDate(receivedStr)
			if err != nil {
				errors = append(errors, fmt.Sprintf(
					"row %d: invalid received_date '%s' for '%s'",
					lineNum, receivedStr, productName,
				))
				continue
			}
			receivedDate = t
		}

		// ── quantity ───────────────────────────────────────────────
		qtyStr := get(cols.quantity)
		qty, err := strconv.Atoi(strings.TrimSpace(qtyStr))
		if err != nil || qty < 0 {
			errors = append(errors, fmt.Sprintf(
				"row %d: invalid quantity '%s' for '%s' — must be a whole number",
				lineNum, qtyStr, productName,
			))
			continue
		}

		// ── purchase price (accept comma decimal separator) ────────
		ppStr := strings.ReplaceAll(get(cols.purchasePrice), ",", ".")
		pp, err := strconv.ParseFloat(ppStr, 64)
		if err != nil {
			errors = append(errors, fmt.Sprintf(
				"row %d: invalid purchase_price '%s' for '%s'",
				lineNum, get(cols.purchasePrice), productName,
			))
			continue
		}

		// ── selling price ──────────────────────────────────────────
		spStr := strings.ReplaceAll(get(cols.sellingPrice), ",", ".")
		sp, err := strconv.ParseFloat(spStr, 64)
		if err != nil {
			errors = append(errors, fmt.Sprintf(
				"row %d: invalid selling_price '%s' for '%s'",
				lineNum, get(cols.sellingPrice), productName,
			))
			continue
		}

		batches = append(batches, parsedBatch{
			ProductName:   productName,
			ProductNameEl: get(cols.nameEl),
			Barcode:       get(cols.barcode),
			BatchNumber:   get(cols.batchNumber),
			ExpiryDate:    expiry,
			Quantity:      qty,
			PurchasePrice: pp,
			SellingPrice:  sp,
			ReceivedDate:  receivedDate,
			Supplier:      get(cols.supplier),
			Category:      get(cols.category),
			Manufacturer:  get(cols.manufacturer),
			RowNum:        lineNum,
		})
	}

	return batches, total, errors
}
