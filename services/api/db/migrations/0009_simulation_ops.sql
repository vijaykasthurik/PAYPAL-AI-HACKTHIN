-- +goose Up
CREATE TABLE simulation_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  scenario_id TEXT NOT NULL,
  mode TEXT NOT NULL,
  case_id UUID REFERENCES payment_cases(id) ON DELETE CASCADE,
  environment TEXT NOT NULL CHECK (environment IN ('LOCAL_SIMULATION','PAYPAL_SIMULATOR')),
  status TEXT NOT NULL DEFAULT 'RUNNING' CHECK (status IN ('RUNNING','COMPLETED','FAILED')),
  details JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ
);

CREATE TABLE route_registry (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source_country CHAR(2) NOT NULL,
  target_country CHAR(2) NOT NULL,
  currency_pair TEXT NOT NULL,
  provider TEXT NOT NULL,
  purpose_code_required BOOLEAN NOT NULL DEFAULT FALSE,
  supported BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (source_country, target_country, currency_pair, provider)
);

CREATE TABLE provider_connections (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  provider TEXT NOT NULL,
  environment TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  config_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE provider_status_samples (
  id BIGSERIAL PRIMARY KEY,
  provider TEXT NOT NULL,
  environment TEXT NOT NULL,
  endpoint TEXT NOT NULL,
  status_code INT NOT NULL,
  latency_ms INT NOT NULL,
  is_healthy BOOLEAN NOT NULL,
  sampled_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ix_provider_status ON provider_status_samples(provider, environment, sampled_at DESC);

-- +goose Down
DROP INDEX IF EXISTS ix_provider_status;
DROP TABLE IF EXISTS provider_status_samples;
DROP TABLE IF EXISTS provider_connections;
DROP TABLE IF EXISTS route_registry;
DROP TABLE IF EXISTS simulation_runs;
