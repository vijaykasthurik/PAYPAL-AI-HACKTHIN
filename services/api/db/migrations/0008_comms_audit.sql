-- +goose Up
CREATE TABLE notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  case_id UUID REFERENCES payment_cases(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  channel TEXT NOT NULL CHECK (channel IN ('IN_APP','EMAIL','WEBHOOK')),
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  read_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE audit_events (
  id BIGSERIAL PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
  case_id UUID REFERENCES payment_cases(id) ON DELETE SET NULL,
  action TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  reason_code TEXT,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  prev_hash CHAR(64),
  curr_hash CHAR(64) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ix_audit_tenant_case ON audit_events(tenant_id, case_id, created_at);

CREATE TABLE retention_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  policy_name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','RUNNING','COMPLETED','FAILED')),
  rows_purged INT NOT NULL DEFAULT 0,
  executed_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE IF EXISTS retention_jobs;
DROP INDEX IF EXISTS ix_audit_tenant_case;
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS notifications;
