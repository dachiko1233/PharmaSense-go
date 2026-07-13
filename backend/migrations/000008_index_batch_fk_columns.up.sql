-- sales.batch_id and alert_actions.batch_id reference inventory_batches(id)
-- without ON DELETE CASCADE, so every DELETE on inventory_batches makes
-- Postgres scan these tables to check for referencing rows. Without an
-- index on the FK column that scan is a full table scan, making batch
-- deletes get slower as sales/alert_actions grow.
CREATE INDEX IF NOT EXISTS idx_sales_batch ON sales(batch_id);
CREATE INDEX IF NOT EXISTS idx_alert_actions_batch ON alert_actions(batch_id);
