CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Pharmacy chains (parent of pharmacies)
CREATE TABLE chains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    owner_email VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE pharmacies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chain_id UUID REFERENCES chains(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    license_number VARCHAR(100) UNIQUE NOT NULL,
    address TEXT,
    city VARCHAR(100),
    phone VARCHAR(50),
    email VARCHAR(255),
    language VARCHAR(10) DEFAULT 'en',
    plan VARCHAR(20) NOT NULL DEFAULT 'free',
    stripe_customer_id VARCHAR(255),
    stripe_subscription_id VARCHAR(255),
    subscription_status VARCHAR(50),
    subscription_current_period_end TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_pharmacies_chain ON pharmacies(chain_id);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    default_pharmacy_id UUID REFERENCES pharmacies(id),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    phone_number VARCHAR(20),
    sms_enabled BOOLEAN DEFAULT FALSE,
    email_verified BOOLEAN DEFAULT FALSE,
    email_verification_token VARCHAR(255),
    password_reset_token VARCHAR(255),
    password_reset_expires_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Per-pharmacy role (a user may access multiple pharmacies in a chain)
CREATE TABLE pharmacy_users (
    pharmacy_id UUID NOT NULL REFERENCES pharmacies(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'staff',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (pharmacy_id, user_id)
);

CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    barcode VARCHAR(100) UNIQUE,
    name VARCHAR(500) NOT NULL,
    name_el VARCHAR(500),
    category VARCHAR(100),
    manufacturer VARCHAR(255),
    requires_prescription BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE inventory_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pharmacy_id UUID NOT NULL REFERENCES pharmacies(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    batch_number VARCHAR(100),
    expiry_date DATE NOT NULL,
    initial_quantity INTEGER NOT NULL,
    current_quantity INTEGER NOT NULL,
    purchase_price DECIMAL(10,2) NOT NULL,
    selling_price DECIMAL(10,2) NOT NULL,
    supplier VARCHAR(255),
    received_date DATE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_batches_pharmacy_expiry ON inventory_batches(pharmacy_id, expiry_date);
CREATE INDEX idx_batches_product ON inventory_batches(product_id);

CREATE TABLE sales (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pharmacy_id UUID NOT NULL REFERENCES pharmacies(id) ON DELETE CASCADE,
    batch_id UUID NOT NULL REFERENCES inventory_batches(id),
    product_id UUID NOT NULL REFERENCES products(id),
    quantity INTEGER NOT NULL,
    unit_price DECIMAL(10,2) NOT NULL,
    total_amount DECIMAL(10,2) NOT NULL,
    sale_date DATE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_sales_pharmacy_date ON sales(pharmacy_id, sale_date);
CREATE INDEX idx_sales_product ON sales(product_id, sale_date);

CREATE TABLE risk_assessments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id UUID NOT NULL REFERENCES inventory_batches(id) ON DELETE CASCADE,
    pharmacy_id UUID NOT NULL REFERENCES pharmacies(id) ON DELETE CASCADE,
    risk_level VARCHAR(20) NOT NULL,
    days_until_expiry INTEGER NOT NULL,
    avg_daily_sales DECIMAL(10,2),
    expected_sales INTEGER,
    estimated_surplus INTEGER,
    estimated_loss DECIMAL(10,2),
    suggested_discount_percent INTEGER,
    calculated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_risk_pharmacy_level ON risk_assessments(pharmacy_id, risk_level);

CREATE TABLE alert_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id UUID NOT NULL REFERENCES inventory_batches(id),
    pharmacy_id UUID NOT NULL REFERENCES pharmacies(id),
    user_id UUID NOT NULL REFERENCES users(id),
    action_type VARCHAR(50) NOT NULL,
    discount_percent INTEGER,
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Log of every email/SMS sent (for debugging and audit)
CREATE TABLE notification_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    pharmacy_id UUID REFERENCES pharmacies(id),
    channel VARCHAR(20) NOT NULL,
    template VARCHAR(100) NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    sent_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_notif_log_pharmacy ON notification_log(pharmacy_id, sent_at DESC);

-- Stripe webhook event log (idempotency)
CREATE TABLE stripe_events (
    id VARCHAR(255) PRIMARY KEY,
    type VARCHAR(100) NOT NULL,
    processed_at TIMESTAMPTZ DEFAULT NOW()
);
