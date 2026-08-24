package handlers

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"pharmasense/internal/middleware"
	"pharmasense/internal/services"
	"pharmasense/internal/services/notifications"
	"pharmasense/internal/services/triplecheck"
)

type AuthHandler struct {
	auth   *services.AuthService
	email  *notifications.EmailService
	screen *triplecheck.Service
	appURL string
}

func NewAuthHandler(auth *services.AuthService, email *notifications.EmailService, screen *triplecheck.Service, appURL string) *AuthHandler {
	return &AuthHandler{auth: auth, email: email, screen: screen, appURL: appURL}
}

// clientIP extracts the end user's IP for anti-abuse screening, preferring the
// proxy headers set in front of the app (Cloudflare, then X-Forwarded-For's
// first hop) before falling back to the direct connection's address.
func clientIP(c *gin.Context) string {
	if ip := strings.TrimSpace(c.GetHeader("CF-Connecting-IP")); ip != "" {
		return ip
	}
	if fwd := c.GetHeader("X-Forwarded-For"); fwd != "" {
		if first := strings.TrimSpace(strings.Split(fwd, ",")[0]); first != "" {
			return first
		}
	}
	if host, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
		return host
	}
	return c.Request.RemoteAddr
}

type signupRequest struct {
	PharmacyName  string `json:"pharmacy_name"  binding:"required,min=2"`
	LicenseNumber string `json:"license_number" binding:"required"`
	City          string `json:"city"           binding:"required"`
	FullName      string `json:"full_name"      binding:"required"`
	Email         string `json:"email"          binding:"required,email"`
	Password      string `json:"password"       binding:"required,min=8"`
	AcceptTerms   bool   `json:"accept_terms"`
}

func (h *AuthHandler) Signup(c *gin.Context) {
	var req signupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !req.AcceptTerms {
		c.JSON(http.StatusBadRequest, gin.H{"error": "you must accept the Privacy Policy and Terms of Service"})
		return
	}

	// Anti-abuse screening. Phone is passed empty — the form does not collect
	// one yet. The screen fails open (allows) when disabled or unreachable, so
	// this only ever blocks on an explicit Allow=false from the provider.
	if check := h.screen.CheckSignup(c.Request.Context(), req.Email, "", clientIP(c)); !check.Allow {
		slog.Info("signup blocked by screening", "email", req.Email, "reason", check.Reason, "flags", check.Flags)
		c.JSON(http.StatusBadRequest, gin.H{"error": check.Reason})
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

	// Send welcome email (non-blocking); use Background so the cancelled request ctx doesn't abort it
	verToken := ""
	if user.EmailVerificationToken != nil {
		verToken = *user.EmailVerificationToken
		slog.Info("signup token generated", "email", user.Email, "token_prefix", verToken[:8]+"...")
	}
	go func(email, fullName, tok string) {
		_ = h.email.SendWelcome(context.Background(), email, fullName, h.appURL, tok)
	}(user.Email, user.FullName, verToken)

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
		go func(email, tok string) {
			_ = h.email.SendPasswordReset(context.Background(), email, "User", h.appURL, tok)
		}(req.Email, token)
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
