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
		Email         string  `json:"email"`
		FullName      string  `json:"full_name"`
		PhoneNumber   *string `json:"phone_number"`
		SMSEnabled    bool    `json:"sms_enabled"`
		EmailVerified bool    `json:"email_verified"`
	}

	err := h.db.QueryRow(c.Request.Context(), `
		SELECT email, full_name, phone_number, sms_enabled, email_verified
		FROM users WHERE id = $1
	`, userID).Scan(&result.Email, &result.FullName, &result.PhoneNumber, &result.SMSEnabled, &result.EmailVerified)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load settings"})
		return
	}
	c.JSON(http.StatusOK, result)
}

type patchNotificationsRequest struct {
	SMSEnabled  *bool   `json:"sms_enabled"`
	PhoneNumber *string `json:"phone_number"`
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
		  sms_enabled  = COALESCE($1, sms_enabled),
		  phone_number = COALESCE($2, phone_number)
		WHERE id = $3
	`, req.SMSEnabled, req.PhoneNumber, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update settings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "settings updated"})
}
