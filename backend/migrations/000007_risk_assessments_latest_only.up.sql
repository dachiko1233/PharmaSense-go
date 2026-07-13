-- risk_assessments was append-only (a new row inserted on every dashboard/list
-- request), so it grew unbounded and every "latest per batch" lookup got
-- progressively slower. Keep only the most recent row per batch, then enforce
-- exactly one row per batch going forward so recalculation can UPSERT instead
-- of INSERT.

-- Drop all but the most recent row per batch.
DELETE FROM risk_assessments ra
USING risk_assessments newer
WHERE ra.batch_id = newer.batch_id
  AND ra.calculated_at < newer.calculated_at;

-- Tie-break exact-timestamp duplicates deterministically by id.
DELETE FROM risk_assessments ra
USING risk_assessments dup
WHERE ra.batch_id = dup.batch_id
  AND ra.calculated_at = dup.calculated_at
  AND ra.id < dup.id;

ALTER TABLE risk_assessments ADD CONSTRAINT risk_assessments_batch_id_key UNIQUE (batch_id);
