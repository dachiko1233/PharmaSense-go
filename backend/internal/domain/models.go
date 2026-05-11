package domain

import (
	"time"

	"github.com/google/uuid"
)

type Chain struct {
	ID         uuid.UUID `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	OwnerEmail string    `json:"owner_email" db:"owner_email"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

type Pharmacy struct {
	ID                          uuid.UUID  `json:"id" db:"id"`
	ChainID                     *uuid.UUID `json:"chain_id,omitempty" db:"chain_id"`
	Name                        string     `json:"name" db:"name"`
	LicenseNumber               string     `json:"license_number" db:"license_number"`
	Address                     *string    `json:"address,omitempty" db:"address"`
	City                        *string    `json:"city,omitempty" db:"city"`
	Phone                       *string    `json:"phone,omitempty" db:"phone"`
	Email                       *string    `json:"email,omitempty" db:"email"`
	Language                    string     `json:"language" db:"language"`
	Plan                        string     `json:"plan" db:"plan"`
	StripeCustomerID            *string    `json:"stripe_customer_id,omitempty" db:"stripe_customer_id"`
	StripeSubscriptionID        *string    `json:"stripe_subscription_id,omitempty" db:"stripe_subscription_id"`
	SubscriptionStatus          *string    `json:"subscription_status,omitempty" db:"subscription_status"`
	SubscriptionCurrentPeriodEnd *time.Time `json:"subscription_current_period_end,omitempty" db:"subscription_current_period_end"`
	CreatedAt                   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt                   time.Time  `json:"updated_at" db:"updated_at"`
}

type User struct {
	ID                      uuid.UUID  `json:"id" db:"id"`
	DefaultPharmacyID       *uuid.UUID `json:"default_pharmacy_id,omitempty" db:"default_pharmacy_id"`
	Email                   string     `json:"email" db:"email"`
	PasswordHash            string     `json:"-" db:"password_hash"`
	FullName                string     `json:"full_name" db:"full_name"`
	PhoneNumber             *string    `json:"phone_number,omitempty" db:"phone_number"`
	SMSEnabled              bool       `json:"sms_enabled" db:"sms_enabled"`
	EmailVerified           bool       `json:"email_verified" db:"email_verified"`
	EmailVerificationToken  *string    `json:"-" db:"email_verification_token"`
	PasswordResetToken      *string    `json:"-" db:"password_reset_token"`
	PasswordResetExpiresAt  *time.Time `json:"-" db:"password_reset_expires_at"`
	IsActive                bool       `json:"is_active" db:"is_active"`
	LastLoginAt             *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt               time.Time  `json:"created_at" db:"created_at"`
}

type PharmacyUser struct {
	PharmacyID uuid.UUID `json:"pharmacy_id" db:"pharmacy_id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	Role       string    `json:"role" db:"role"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type Product struct {
	ID                   uuid.UUID `json:"id" db:"id"`
	Barcode              *string   `json:"barcode,omitempty" db:"barcode"`
	Name                 string    `json:"name" db:"name"`
	NameEl               *string   `json:"name_el,omitempty" db:"name_el"`
	Category             *string   `json:"category,omitempty" db:"category"`
	Manufacturer         *string   `json:"manufacturer,omitempty" db:"manufacturer"`
	RequiresPrescription bool      `json:"requires_prescription" db:"requires_prescription"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
}

type InventoryBatch struct {
	ID              uuid.UUID `json:"id" db:"id"`
	PharmacyID      uuid.UUID `json:"pharmacy_id" db:"pharmacy_id"`
	ProductID       uuid.UUID `json:"product_id" db:"product_id"`
	BatchNumber     *string   `json:"batch_number,omitempty" db:"batch_number"`
	ExpiryDate      time.Time `json:"expiry_date" db:"expiry_date"`
	InitialQuantity int       `json:"initial_quantity" db:"initial_quantity"`
	CurrentQuantity int       `json:"current_quantity" db:"current_quantity"`
	PurchasePrice   float64   `json:"purchase_price" db:"purchase_price"`
	SellingPrice    float64   `json:"selling_price" db:"selling_price"`
	Supplier        *string   `json:"supplier,omitempty" db:"supplier"`
	ReceivedDate    time.Time `json:"received_date" db:"received_date"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`

	// Joined fields
	ProductName  *string `json:"product_name,omitempty" db:"product_name"`
	Category     *string `json:"category,omitempty" db:"category"`
	Manufacturer *string `json:"manufacturer,omitempty" db:"manufacturer"`
}

type Sale struct {
	ID          uuid.UUID `json:"id" db:"id"`
	PharmacyID  uuid.UUID `json:"pharmacy_id" db:"pharmacy_id"`
	BatchID     uuid.UUID `json:"batch_id" db:"batch_id"`
	ProductID   uuid.UUID `json:"product_id" db:"product_id"`
	Quantity    int       `json:"quantity" db:"quantity"`
	UnitPrice   float64   `json:"unit_price" db:"unit_price"`
	TotalAmount float64   `json:"total_amount" db:"total_amount"`
	SaleDate    time.Time `json:"sale_date" db:"sale_date"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type RiskAssessment struct {
	ID                     uuid.UUID  `json:"id" db:"id"`
	BatchID                uuid.UUID  `json:"batch_id" db:"batch_id"`
	PharmacyID             uuid.UUID  `json:"pharmacy_id" db:"pharmacy_id"`
	RiskLevel              string     `json:"risk_level" db:"risk_level"`
	DaysUntilExpiry        int        `json:"days_until_expiry" db:"days_until_expiry"`
	AvgDailySales          *float64   `json:"avg_daily_sales,omitempty" db:"avg_daily_sales"`
	ExpectedSales          *int       `json:"expected_sales,omitempty" db:"expected_sales"`
	EstimatedSurplus       *int       `json:"estimated_surplus,omitempty" db:"estimated_surplus"`
	EstimatedLoss          *float64   `json:"estimated_loss,omitempty" db:"estimated_loss"`
	SuggestedDiscountPct   *int       `json:"suggested_discount_percent,omitempty" db:"suggested_discount_percent"`
	CalculatedAt           time.Time  `json:"calculated_at" db:"calculated_at"`

	// Joined fields
	ProductName    *string    `json:"product_name,omitempty" db:"product_name"`
	BatchNumber    *string    `json:"batch_number,omitempty" db:"batch_number"`
	ExpiryDate     *time.Time `json:"expiry_date,omitempty" db:"expiry_date"`
	CurrentQty     *int       `json:"current_quantity,omitempty" db:"current_quantity"`
	PurchasePrice  *float64   `json:"purchase_price,omitempty" db:"purchase_price"`
}

type AlertAction struct {
	ID             uuid.UUID `json:"id" db:"id"`
	BatchID        uuid.UUID `json:"batch_id" db:"batch_id"`
	PharmacyID     uuid.UUID `json:"pharmacy_id" db:"pharmacy_id"`
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	ActionType     string    `json:"action_type" db:"action_type"`
	DiscountPct    *int      `json:"discount_percent,omitempty" db:"discount_percent"`
	Notes          *string   `json:"notes,omitempty" db:"notes"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

type NotificationLog struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	UserID       *uuid.UUID `json:"user_id,omitempty" db:"user_id"`
	PharmacyID   *uuid.UUID `json:"pharmacy_id,omitempty" db:"pharmacy_id"`
	Channel      string     `json:"channel" db:"channel"`
	Template     string     `json:"template" db:"template"`
	Recipient    string     `json:"recipient" db:"recipient"`
	Status       string     `json:"status" db:"status"`
	ErrorMessage *string    `json:"error_message,omitempty" db:"error_message"`
	SentAt       time.Time  `json:"sent_at" db:"sent_at"`
}

// RiskLevel constants
const (
	RiskLevelCritical = "CRITICAL"
	RiskLevelHigh     = "HIGH"
	RiskLevelMedium   = "MEDIUM"
	RiskLevelLow      = "LOW"
)

// Plan constants
const (
	PlanFree  = "free"
	PlanPro   = "pro"
	PlanChain = "chain"
)

// Role constants
const (
	RoleChainAdmin = "chain_admin"
	RoleAdmin      = "admin"
	RoleStaff      = "staff"
)

// DashboardStats aggregates KPIs for the main dashboard
type DashboardStats struct {
	CriticalCount      int     `json:"critical_count"`
	HighCount          int     `json:"high_count"`
	MediumCount        int     `json:"medium_count"`
	LowCount           int     `json:"low_count"`
	EstimatedLoss      float64 `json:"estimated_loss"`
	PotentialSavings   float64 `json:"potential_savings"`
	TotalInventoryValue float64 `json:"total_inventory_value"`
	TotalBatches       int     `json:"total_batches"`
}

// ChainDashboard aggregates KPIs across all pharmacies in a chain
type ChainDashboard struct {
	ChainID    uuid.UUID        `json:"chain_id"`
	ChainName  string           `json:"chain_name"`
	Pharmacies []PharmacyStats  `json:"pharmacies"`
	Totals     DashboardStats   `json:"totals"`
}

type PharmacyStats struct {
	Pharmacy Pharmacy       `json:"pharmacy"`
	Stats    DashboardStats `json:"stats"`
}
