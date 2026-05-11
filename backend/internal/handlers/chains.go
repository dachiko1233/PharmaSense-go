package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"pharmasense/internal/middleware"
)

type ChainHandler struct {
	db *pgxpool.Pool
}

func NewChainHandler(db *pgxpool.Pool) *ChainHandler {
	return &ChainHandler{db: db}
}

func (h *ChainHandler) Get(c *gin.Context) {
	role := middleware.GetRole(c)
	if role != "chain_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "chain_admin role required"})
		return
	}

	chainID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain id"})
		return
	}

	var chain struct {
		ID         uuid.UUID `json:"id"`
		Name       string    `json:"name"`
		OwnerEmail string    `json:"owner_email"`
	}
	err = h.db.QueryRow(c.Request.Context(), `
		SELECT id, name, owner_email FROM chains WHERE id = $1
	`, chainID).Scan(&chain.ID, &chain.Name, &chain.OwnerEmail)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chain not found"})
		return
	}
	c.JSON(http.StatusOK, chain)
}

func (h *ChainHandler) Dashboard(c *gin.Context) {
	role := middleware.GetRole(c)
	userID := middleware.GetUserID(c)
	if role != "chain_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "chain_admin role required"})
		return
	}

	chainID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain id"})
		return
	}

	// Verify user is chain_admin of this chain
	var hasAccess bool
	_ = h.db.QueryRow(c.Request.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM pharmacies p
			JOIN pharmacy_users pu ON pu.pharmacy_id = p.id
			WHERE p.chain_id = $1 AND pu.user_id = $2 AND pu.role = 'chain_admin'
		)
	`, chainID, userID).Scan(&hasAccess)
	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	rows, err := h.db.Query(c.Request.Context(), `
		SELECT p.id, p.name, p.plan, p.city,
		       COUNT(ib.id) as total_batches,
		       COUNT(*) FILTER (WHERE ra.risk_level = 'CRITICAL') as critical_count,
		       COUNT(*) FILTER (WHERE ra.risk_level = 'HIGH') as high_count,
		       COALESCE(SUM(ra.estimated_loss) FILTER (WHERE ra.risk_level IN ('CRITICAL','HIGH')), 0) as estimated_loss,
		       COALESCE(SUM(ib.current_quantity * ib.purchase_price), 0) as inventory_value
		FROM pharmacies p
		LEFT JOIN inventory_batches ib ON ib.pharmacy_id = p.id AND ib.current_quantity > 0
		LEFT JOIN LATERAL (
			SELECT risk_level, estimated_loss
			FROM risk_assessments
			WHERE batch_id = ib.id
			ORDER BY calculated_at DESC
			LIMIT 1
		) ra ON true
		WHERE p.chain_id = $1
		GROUP BY p.id
		ORDER BY p.name
	`, chainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load chain dashboard"})
		return
	}
	defer rows.Close()

	type PharmacyStats struct {
		ID             uuid.UUID `json:"id"`
		Name           string    `json:"name"`
		Plan           string    `json:"plan"`
		City           *string   `json:"city"`
		TotalBatches   int       `json:"total_batches"`
		CriticalCount  int       `json:"critical_count"`
		HighCount      int       `json:"high_count"`
		EstimatedLoss  float64   `json:"estimated_loss"`
		InventoryValue float64   `json:"inventory_value"`
	}

	var pharmacies []PharmacyStats
	totals := struct {
		TotalBatches  int     `json:"total_batches"`
		CriticalCount int     `json:"critical_count"`
		EstimatedLoss float64 `json:"estimated_loss"`
	}{}

	for rows.Next() {
		var ps PharmacyStats
		if err := rows.Scan(&ps.ID, &ps.Name, &ps.Plan, &ps.City,
			&ps.TotalBatches, &ps.CriticalCount, &ps.HighCount, &ps.EstimatedLoss, &ps.InventoryValue); err != nil {
			continue
		}
		totals.TotalBatches += ps.TotalBatches
		totals.CriticalCount += ps.CriticalCount
		totals.EstimatedLoss += ps.EstimatedLoss
		pharmacies = append(pharmacies, ps)
	}
	if pharmacies == nil {
		pharmacies = []PharmacyStats{}
	}

	c.JSON(http.StatusOK, gin.H{
		"chain_id":   chainID,
		"pharmacies": pharmacies,
		"totals":     totals,
	})
}
