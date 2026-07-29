ALTER TABLE pharmacies ADD COLUMN dodo_customer_id VARCHAR(255);
ALTER TABLE pharmacies ADD COLUMN dodo_subscription_id VARCHAR(255);

CREATE TABLE dodo_events (
    id VARCHAR(255) PRIMARY KEY,
    type VARCHAR(100) NOT NULL,
    processed_at TIMESTAMPTZ DEFAULT NOW()
);
