package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"pharmasense/internal/middleware"
	"pharmasense/internal/services"
)

type PharmacyHandler struct {
	db   *pgxpool.Pool
	auth *services.AuthService
}

func NewPharmacyHandler(db *pgxpool.Pool, auth *services.AuthService) *PharmacyHandler {
	return &PharmacyHandler{db: db, auth: auth}
}

func (h *PharmacyHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	rows, err := h.db.Query(c.Request.Context(), `
		SELECT p.id, p.name, p.license_number, p.city, p.plan, p.language,
		       p.subscription_status, p.created_at, p.updated_at, pu.role
		FROM pharmacies p
		JOIN pharmacy_users pu ON pu.pharmacy_id = p.id
		WHERE pu.user_id = $1
		ORDER BY p.name
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list pharmacies"})
		return
	}
	defer rows.Close()

	type PharmacyWithRole struct {
		ID                 uuid.UUID `json:"id"`
		Name               string    `json:"name"`
		LicenseNumber      string    `json:"license_number"`
		City               *string   `json:"city"`
		Plan               string    `json:"plan"`
		Language           string    `json:"language"`
		SubscriptionStatus *string   `json:"subscription_status"`
		Role               string    `json:"role"`
	}

	var pharmacies []PharmacyWithRole
	for rows.Next() {
		var p PharmacyWithRole
		if err := rows.Scan(&p.ID, &p.Name, &p.LicenseNumber, &p.City, &p.Plan, &p.Language,
			&p.SubscriptionStatus, nil, nil, &p.Role); err != nil {
			continue
		}
		pharmacies = append(pharmacies, p)
	}
	if pharmacies == nil {
		pharmacies = []PharmacyWithRole{}
	}
	c.JSON(http.StatusOK, pharmacies)
}

func (h *PharmacyHandler) Get(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)
	paramID, err := uuid.Parse(c.Param("id"))
	if err != nil || paramID != pharmacyID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	row := h.db.QueryRow(c.Request.Context(), `
		SELECT id, name, license_number, city, phone, email, plan, language,
		       subscription_status, subscription_current_period_end
		FROM pharmacies WHERE id = $1
	`, pharmacyID)

	type PharmacyDetail struct {
		ID                           uuid.UUID `json:"id"`
		Name                         string    `json:"name"`
		LicenseNumber                string    `json:"license_number"`
		City                         *string   `json:"city"`
		Phone                        *string   `json:"phone"`
		Email                        *string   `json:"email"`
		Plan                         string    `json:"plan"`
		Language                     string    `json:"language"`
		SubscriptionStatus           *string   `json:"subscription_status"`
		SubscriptionCurrentPeriodEnd *string   `json:"subscription_current_period_end"`
	}
	var p PharmacyDetail
	if err := row.Scan(&p.ID, &p.Name, &p.LicenseNumber, &p.City, &p.Phone, &p.Email,
		&p.Plan, &p.Language, &p.SubscriptionStatus, &p.SubscriptionCurrentPeriodEnd); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pharmacy not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

type patchPharmacyRequest struct {
	Name     *string `json:"name"`
	City     *string `json:"city"`
	Phone    *string `json:"phone"`
	Email    *string `json:"email"`
	Language *string `json:"language"`
}

func (h *PharmacyHandler) Patch(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)
	role := middleware.GetRole(c)
	if role == "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin role required"})
		return
	}

	var req patchPharmacyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.db.Exec(c.Request.Context(), `
		UPDATE pharmacies SET
		  name     = COALESCE($1, name),
		  city     = COALESCE($2, city),
		  phone    = COALESCE($3, phone),
		  email    = COALESCE($4, email),
		  language = COALESCE($5, language),
		  updated_at = NOW()
		WHERE id = $6
	`, req.Name, req.City, req.Phone, req.Email, req.Language, pharmacyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update pharmacy"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

type switchPharmacyRequest struct {
	PharmacyID uuid.UUID `json:"pharmacy_id" binding:"required"`
}

func (h *PharmacyHandler) Switch(c *gin.Context) {
	var req switchPharmacyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	token, pharmacy, err := h.auth.SwitchPharmacy(c.Request.Context(), userID, req.PharmacyID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied to that pharmacy"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":    token,
		"pharmacy": pharmacy,
	})
}
