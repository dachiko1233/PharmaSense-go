package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"pharmasense/internal/domain"
	"pharmasense/internal/middleware"
)

type AuthService struct {
	db         *pgxpool.Pool
	jwtSecret  []byte
	jwtExpiry  time.Duration
	appURL     string
}

func NewAuthService(db *pgxpool.Pool, jwtSecret string, jwtExpiryHours int, appURL string) *AuthService {
	return &AuthService{
		db:        db,
		jwtSecret: []byte(jwtSecret),
		jwtExpiry: time.Duration(jwtExpiryHours) * time.Hour,
		appURL:    appURL,
	}
}

func (s *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func (s *AuthService) CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (s *AuthService) GenerateToken(userID, pharmacyID uuid.UUID, email, role, plan string) (string, error) {
	claims := &middleware.Claims{
		UserID:     userID,
		PharmacyID: pharmacyID,
		Email:      email,
		Role:       role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

func (s *AuthService) GenerateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Signup creates a new pharmacy + admin user in a single transaction.
func (s *AuthService) Signup(ctx context.Context, pharmacyName, licenseNumber, city, fullName, email, password string) (*domain.User, *domain.Pharmacy, string, error) {
	hash, err := s.HashPassword(password)
	if err != nil {
		return nil, nil, "", fmt.Errorf("signup: %w", err)
	}

	verificationToken, err := s.GenerateSecureToken()
	if err != nil {
		return nil, nil, "", fmt.Errorf("signup: %w", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, "", fmt.Errorf("signup: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create pharmacy
	var pharmacy domain.Pharmacy
	err = tx.QueryRow(ctx, `
		INSERT INTO pharmacies (name, license_number, city, plan)
		VALUES ($1, $2, $3, 'free')
		RETURNING id, name, license_number, city, plan, language, created_at, updated_at
	`, pharmacyName, licenseNumber, city).Scan(
		&pharmacy.ID, &pharmacy.Name, &pharmacy.LicenseNumber,
		&pharmacy.City, &pharmacy.Plan, &pharmacy.Language,
		&pharmacy.CreatedAt, &pharmacy.UpdatedAt,
	)
	if err != nil {
		return nil, nil, "", fmt.Errorf("signup: create pharmacy: %w", err)
	}

	// Create user
	var user domain.User
	err = tx.QueryRow(ctx, `
		INSERT INTO users (default_pharmacy_id, email, password_hash, full_name, email_verification_token)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, full_name, default_pharmacy_id, email_verified, sms_enabled, is_active, created_at
	`, pharmacy.ID, email, hash, fullName, verificationToken).Scan(
		&user.ID, &user.Email, &user.FullName, &user.DefaultPharmacyID,
		&user.EmailVerified, &user.SMSEnabled, &user.IsActive, &user.CreatedAt,
	)
	if err != nil {
		return nil, nil, "", fmt.Errorf("signup: create user: %w", err)
	}

	// Link user to pharmacy as admin
	_, err = tx.Exec(ctx, `
		INSERT INTO pharmacy_users (pharmacy_id, user_id, role)
		VALUES ($1, $2, 'admin')
	`, pharmacy.ID, user.ID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("signup: link pharmacy user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, "", fmt.Errorf("signup: commit: %w", err)
	}

	token, err := s.GenerateToken(user.ID, pharmacy.ID, user.Email, domain.RoleAdmin, pharmacy.Plan)
	if err != nil {
		return nil, nil, "", fmt.Errorf("signup: %w", err)
	}

	return &user, &pharmacy, token, nil
}

// Login authenticates a user and returns a JWT.
func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.User, *domain.Pharmacy, string, error) {
	var user domain.User
	err := s.db.QueryRow(ctx, `
		SELECT id, email, password_hash, full_name, default_pharmacy_id, email_verified, sms_enabled, is_active, created_at
		FROM users WHERE email = $1 AND is_active = true
	`, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName,
		&user.DefaultPharmacyID, &user.EmailVerified, &user.SMSEnabled,
		&user.IsActive, &user.CreatedAt,
	)
	if err != nil {
		return nil, nil, "", fmt.Errorf("login: user not found")
	}

	if !s.CheckPassword(user.PasswordHash, password) {
		return nil, nil, "", fmt.Errorf("login: invalid credentials")
	}

	if user.DefaultPharmacyID == nil {
		return nil, nil, "", fmt.Errorf("login: no pharmacy assigned")
	}

	// Get the pharmacy and user's role
	var pharmacy domain.Pharmacy
	var role string
	err = s.db.QueryRow(ctx, `
		SELECT p.id, p.name, p.license_number, p.plan, p.language,
		       p.subscription_status, p.created_at, p.updated_at,
		       pu.role
		FROM pharmacies p
		JOIN pharmacy_users pu ON pu.pharmacy_id = p.id
		WHERE p.id = $1 AND pu.user_id = $2
	`, user.DefaultPharmacyID, user.ID).Scan(
		&pharmacy.ID, &pharmacy.Name, &pharmacy.LicenseNumber,
		&pharmacy.Plan, &pharmacy.Language, &pharmacy.SubscriptionStatus,
		&pharmacy.CreatedAt, &pharmacy.UpdatedAt,
		&role,
	)
	if err != nil {
		return nil, nil, "", fmt.Errorf("login: get pharmacy: %w", err)
	}

	// Update last_login_at
	_, _ = s.db.Exec(ctx, `UPDATE users SET last_login_at = NOW() WHERE id = $1`, user.ID)

	token, err := s.GenerateToken(user.ID, pharmacy.ID, user.Email, role, pharmacy.Plan)
	if err != nil {
		return nil, nil, "", fmt.Errorf("login: %w", err)
	}

	return &user, &pharmacy, token, nil
}

// SwitchPharmacy issues a new JWT for a different pharmacy (user must have access).
func (s *AuthService) SwitchPharmacy(ctx context.Context, userID, targetPharmacyID uuid.UUID) (string, *domain.Pharmacy, error) {
	var pharmacy domain.Pharmacy
	var role string
	err := s.db.QueryRow(ctx, `
		SELECT p.id, p.name, p.license_number, p.plan, p.language,
		       p.subscription_status, p.created_at, p.updated_at,
		       pu.role
		FROM pharmacies p
		JOIN pharmacy_users pu ON pu.pharmacy_id = p.id
		WHERE p.id = $1 AND pu.user_id = $2
	`, targetPharmacyID, userID).Scan(
		&pharmacy.ID, &pharmacy.Name, &pharmacy.LicenseNumber,
		&pharmacy.Plan, &pharmacy.Language, &pharmacy.SubscriptionStatus,
		&pharmacy.CreatedAt, &pharmacy.UpdatedAt,
		&role,
	)
	if err != nil {
		return "", nil, fmt.Errorf("switch pharmacy: access denied or not found")
	}

	var email string
	_ = s.db.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email)

	token, err := s.GenerateToken(userID, pharmacy.ID, email, role, pharmacy.Plan)
	if err != nil {
		return "", nil, fmt.Errorf("switch pharmacy: %w", err)
	}

	return token, &pharmacy, nil
}

// VerifyEmail verifies a user's email using a token.
func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	result, err := s.db.Exec(ctx, `
		UPDATE users SET email_verified = true, email_verification_token = NULL
		WHERE email_verification_token = $1
	`, token)
	if err != nil {
		return fmt.Errorf("verify email: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("verify email: invalid token")
	}
	return nil
}

// InitiatePasswordReset generates a reset token and stores it.
func (s *AuthService) InitiatePasswordReset(ctx context.Context, email string) (string, error) {
	token, err := s.GenerateSecureToken()
	if err != nil {
		return "", fmt.Errorf("password reset: %w", err)
	}
	expires := time.Now().Add(1 * time.Hour)
	result, err := s.db.Exec(ctx, `
		UPDATE users SET password_reset_token = $1, password_reset_expires_at = $2
		WHERE email = $3 AND is_active = true
	`, token, expires, email)
	if err != nil {
		return "", fmt.Errorf("password reset: %w", err)
	}
	if result.RowsAffected() == 0 {
		// Don't reveal if email exists
		return "", nil
	}
	return token, nil
}

// ResetPassword applies a new password using the reset token.
func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	hash, err := s.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("reset password: %w", err)
	}
	result, err := s.db.Exec(ctx, `
		UPDATE users
		SET password_hash = $1, password_reset_token = NULL, password_reset_expires_at = NULL
		WHERE password_reset_token = $2 AND password_reset_expires_at > NOW()
	`, hash, token)
	if err != nil {
		return fmt.Errorf("reset password: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("reset password: invalid or expired token")
	}
	return nil
}
