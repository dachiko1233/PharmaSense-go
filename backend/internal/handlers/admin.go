package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"pharmasense/internal/middleware"
	"pharmasense/internal/services/cron"
)

type AdminHandler struct {
	db        *pgxpool.Pool
	digestJob *cron.DigestJob
}

func NewAdminHandler(db *pgxpool.Pool, digestJob *cron.DigestJob) *AdminHandler {
	return &AdminHandler{db: db, digestJob: digestJob}
}

// TriggerDigest manually fires the daily digest for testing.
//
// POST /api/v1/admin/trigger-digest?pharmacy_id=<uuid>
// Requires admin or chain_admin role.
// Returns { sent, failed, duration_ms }.
func (h *AdminHandler) TriggerDigest(c *gin.Context) {
	role := middleware.GetRole(c)
	if role != "admin" && role != "chain_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin or chain_admin role required"})
		return
	}

	pharmacyID := c.Query("pharmacy_id")
	start := time.Now()

	sent, failed := h.digestJob.RunWithStats(c.Request.Context(), pharmacyID)

	c.JSON(http.StatusOK, gin.H{
		"sent":        sent,
		"failed":      failed,
		"duration_ms": time.Since(start).Milliseconds(),
	})
}
