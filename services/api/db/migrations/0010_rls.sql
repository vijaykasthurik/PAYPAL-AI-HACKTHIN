-- +goose Up
-- Enable Row Level Security (RLS) on tenant-scoped tables

ALTER TABLE payment_cases ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_cases FORCE ROW LEVEL SECURITY;

CREATE POLICY pc_tenant_isolation ON payment_cases
  USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
  WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE payment_attempts ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_attempts FORCE ROW LEVEL SECURITY;

CREATE POLICY pa_tenant_isolation ON payment_attempts
  USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
  WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE evidence_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE evidence_items FORCE ROW LEVEL SECURITY;

CREATE POLICY ei_tenant_isolation ON evidence_items
  USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
  WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE payment_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_events FORCE ROW LEVEL SECURITY;

CREATE POLICY pe_tenant_isolation ON payment_events
  USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
  WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE passport_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE passport_versions FORCE ROW LEVEL SECURITY;

CREATE POLICY pv_tenant_isolation ON passport_versions
  USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
  WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events FORCE ROW LEVEL SECURITY;

CREATE POLICY ae_tenant_isolation ON audit_events
  USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
  WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- Append-only protection triggers for immutable tables:
CREATE OR REPLACE FUNCTION protect_append_only() RETURNS TRIGGER AS $$
BEGIN
  RAISE EXCEPTION 'Table % is strictly append-only; updates and deletes are prohibited.', TG_TABLE_NAME;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_protect_payment_events
  BEFORE UPDATE OR DELETE ON payment_events
  FOR EACH ROW EXECUTE FUNCTION protect_append_only();

CREATE TRIGGER trg_protect_evidence_items
  BEFORE UPDATE OR DELETE ON evidence_items
  FOR EACH ROW EXECUTE FUNCTION protect_append_only();

CREATE TRIGGER trg_protect_audit_events
  BEFORE UPDATE OR DELETE ON audit_events
  FOR EACH ROW EXECUTE FUNCTION protect_append_only();

-- +goose Down
DROP TRIGGER IF EXISTS trg_protect_audit_events ON audit_events;
DROP TRIGGER IF EXISTS trg_protect_evidence_items ON evidence_items;
DROP TRIGGER IF EXISTS trg_protect_payment_events ON payment_events;
DROP FUNCTION IF EXISTS protect_append_only();

DROP POLICY IF EXISTS ae_tenant_isolation ON audit_events;
ALTER TABLE audit_events DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS pv_tenant_isolation ON passport_versions;
ALTER TABLE passport_versions DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS pe_tenant_isolation ON payment_events;
ALTER TABLE payment_events DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS ei_tenant_isolation ON evidence_items;
ALTER TABLE evidence_items DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS pa_tenant_isolation ON payment_attempts;
ALTER TABLE payment_attempts DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS pc_tenant_isolation ON payment_cases;
ALTER TABLE payment_cases DISABLE ROW LEVEL SECURITY;
