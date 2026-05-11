package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pharmasense/internal/middleware"
	"pharmasense/internal/services"
	"pharmasense/internal/services/notifications"
)

type AuthHandler struct {
	auth  *services.AuthService
	email *notifications.EmailService
	appURL string
}

func NewAuthHandler(auth *services.AuthService, email *notifications.EmailService, appURL string) *AuthHandler {
	return &AuthHandler{auth: auth, email: email, appURL: appURL}
}

type signupRequest struct {
	PharmacyName  string `json:"pharmacy_name"  binding:"required,min=2"`
	LicenseNumber string `json:"license_number" binding:"required"`
	City          string `json:"city"           binding:"required"`
	FullName      string `json:"full_name"      binding:"required"`
	Email         string `json:"email"          binding:"required,email"`
	Password      string `json:"password"       binding:"required,min=8"`
}

func (h *AuthHandler) Signup(c *gin.Context) {
	var req signupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, pharmacy, token, err := h.auth.Signup(
		c.Request.Context(),
		req.PharmacyName, req.LicenseNumber, req.City,
		req.FullName, req.Email, req.Password,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Send welcome email (non-blocking)
	go func() {
		var verToken string
		if user.EmailVerificationToken != nil {
			verToken = *user.EmailVerificationToken
		}
		_ = h.email.SendWelcome(c.Request.Context(), user.Email, user.FullName, h.appURL, verToken)
	}()

	c.JSON(http.StatusCreated, gin.H{
		"token":    token,
		"user":     user,
		"pharmacy": pharmacy,
	})
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, pharmacy, token, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":    token,
		"user":     user,
		"pharmacy": pharmacy,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	// JWT is stateless; client should discard the token
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)
	pharmacyID := middleware.GetPharmacyID(c)

	var result struct {
		UserID     string `json:"user_id"`
		PharmacyID string `json:"pharmacy_id"`
		Email      string `json:"email"`
		Role       string `json:"role"`
	}
	result.UserID = userID.String()
	result.PharmacyID = pharmacyID.String()
	result.Email = c.GetString("email")
	result.Role = c.GetString("role")

	c.JSON(http.StatusOK, result)
}

type verifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req verifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.auth.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired verification token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "email verified"})
}

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.auth.InitiatePasswordReset(c.Request.Context(), req.Email)
	if err != nil {
		// Don't reveal errors
		c.JSON(http.StatusOK, gin.H{"message": "if that email exists, a reset link was sent"})
		return
	}

	if token != "" {
		go func() {
			_ = h.email.SendPasswordReset(c.Request.Context(), req.Email, "User", h.appURL, token)
		}()
	}
	c.JSON(http.StatusOK, gin.H{"message": "if that email exists, a reset link was sent"})
}

type resetPasswordRequest struct {
	Token    string `json:"token"    binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.auth.ResetPassword(c.Request.Context(), req.Token, req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired reset token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password reset successful"})
}
