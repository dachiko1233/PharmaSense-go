-- Add EXPIRED risk level support.
-- risk_level is VARCHAR(20) with no prior CHECK constraint, so we just add one.
ALTER TABLE risk_assessments
    DROP CONSTRAINT IF EXISTS risk_assessments_risk_level_check;

ALTER TABLE risk_assessments
    ADD CONSTRAINT risk_assessments_risk_level_check
    CHECK (risk_level IN ('EXPIRED', 'CRITICAL', 'HIGH', 'MEDIUM', 'LOW'));
