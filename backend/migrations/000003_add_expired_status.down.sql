ALTER TABLE risk_assessments
    DROP CONSTRAINT IF EXISTS risk_assessments_risk_level_check;

ALTER TABLE risk_assessments
    ADD CONSTRAINT risk_assessments_risk_level_check
    CHECK (risk_level IN ('CRITICAL', 'HIGH', 'MEDIUM', 'LOW'));
