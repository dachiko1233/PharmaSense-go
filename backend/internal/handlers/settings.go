package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"pharmasense/internal/middleware"
)

type SettingsHandler struct {
	db *pgxpool.Pool
}

func NewSettingsHandler(db *pgxpool.Pool) *SettingsHandler {
	return &SettingsHandler{db: db}
}

func (h *SettingsHandler) GetNotifications(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var result struct {
		Email               string  `json:"email"`
		FullName            string  `json:"full_name"`
		PhoneNumber         *string `json:"phone_number"`
		SMSEnabled          bool    `json:"sms_enabled"`
		EmailVerified       bool    `json:"email_verified"`
		DailyDigestEnabled  bool    `json:"daily_digest_enabled"`
		DigestTime          string  `json:"digest_time"`
	}

	err := h.db.QueryRow(c.Request.Context(), `
		SELECT email, full_name, phone_number, sms_enabled, email_verified,
		       daily_digest_enabled,
		       TO_CHAR(digest_time, 'HH24:MI') AS digest_time
		FROM users WHERE id = $1
	`, userID).Scan(
		&result.Email, &result.FullName, &result.PhoneNumber,
		&result.SMSEnabled, &result.EmailVerified,
		&result.DailyDigestEnabled, &result.DigestTime,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load settings"})
		return
	}
	c.JSON(http.StatusOK, result)
}

type patchNotificationsRequest struct {
	SMSEnabled         *bool   `json:"sms_enabled"`
	PhoneNumber        *string `json:"phone_number"`
	DailyDigestEnabled *bool   `json:"daily_digest_enabled"`
	DigestTime         *string `json:"digest_time"` // "HH:MM"
}

func (h *SettingsHandler) PatchNotifications(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req patchNotificationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.db.Exec(c.Request.Context(), `
		UPDATE users SET
		    sms_enabled          = COALESCE($1, sms_enabled),
		    phone_number         = COALESCE($2, phone_number),
		    daily_digest_enabled = COALESCE($3, daily_digest_enabled),
		    digest_time          = COALESCE($4::time, digest_time)
		WHERE id = $5
	`, req.SMSEnabled, req.PhoneNumber, req.DailyDigestEnabled, req.DigestTime, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update settings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "settings updated"})
}
