-- +goose Up
CREATE TABLE evidence_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  case_id UUID NOT NULL REFERENCES payment_cases(id) ON DELETE CASCADE,
  kind TEXT NOT NULL CHECK (kind IN (
    'PROVIDER_GET_ORDER','PROVIDER_GET_CAPTURE','PROVIDER_WEBHOOK','APP_TRACE','MERCHANT_DB_SNAPSHOT',
    'PUBLIC_STATUS','USER_REPORT','SIMULATION','ROUTE_RULE','FX_QUOTE'
  )),
  tier SMALLINT NOT NULL CHECK (tier BETWEEN 1 AND 6),
  source_trust TEXT NOT NULL CHECK (source_trust IN (
    'PROVIDER_VERIFIED','LOCAL_OBSERVED','PUBLIC_SOURCE','USER_REPORTED','SIMULATED'
  )),
  environment TEXT NOT NULL,
  verified BOOLEAN NOT NULL DEFAULT FALSE,
  untrusted_text BOOLEAN NOT NULL DEFAULT FALSE,
  occurred_at TIMESTAMPTZ,
  observed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  redacted_extract JSONB NOT NULL DEFAULT '{}'::jsonb,
  payload_sha256 CHAR(64),
  encrypted_payload_ref TEXT,
  access_class TEXT NOT NULL DEFAULT 'TENANT' CHECK (access_class IN ('TENANT','SENDER','RECEIVER','OPS_ONLY')),
  CHECK (source_trust <> 'SIMULATED' OR tier = 6)
);

CREATE INDEX ix_evidence_case ON evidence_items(case_id, observed_at);

CREATE TABLE evidence_edges (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  case_id UUID NOT NULL REFERENCES payment_cases(id) ON DELETE CASCADE,
  from_evidence_id UUID NOT NULL REFERENCES evidence_items(id) ON DELETE CASCADE,
  to_evidence_id UUID NOT NULL REFERENCES evidence_items(id) ON DELETE CASCADE,
  relation TEXT NOT NULL CHECK (relation IN (
    'CORROBORATES','CONTRADICTS','PRECEDES','DERIVED_FROM','REQUIRES','RESOLVES','INDEPENDENT_OF'
  )),
  rule_id TEXT NOT NULL,
  rule_version INT NOT NULL,
  origin TEXT NOT NULL CHECK (origin IN ('DETERMINISTIC','LLM_HYPOTHESIS')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE payment_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  case_id UUID NOT NULL REFERENCES payment_cases(id) ON DELETE CASCADE,
  evidence_id UUID NOT NULL REFERENCES evidence_items(id) ON DELETE CASCADE,
  provider TEXT,
  provider_event_id TEXT,
  environment TEXT NOT NULL,
  event_type TEXT NOT NULL,
  status_observed TEXT,
  source_trust TEXT NOT NULL,
  verified BOOLEAN NOT NULL DEFAULT FALSE,
  provider_occurred_at TIMESTAMPTZ,
  received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  schema_version INT NOT NULL DEFAULT 1
);

CREATE UNIQUE INDEX uq_verified_provider_events
  ON payment_events(provider, environment, provider_event_id)
  WHERE provider_event_id IS NOT NULL AND verified = TRUE;

CREATE INDEX ix_case_events ON payment_events(case_id, received_at, id);

CREATE TABLE webhook_inbox (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  environment TEXT NOT NULL,
  provider_event_id TEXT,
  transmission_id TEXT,
  event_type TEXT,
  signature_verified BOOLEAN NOT NULL DEFAULT FALSE,
  payload_sha256 CHAR(64) NOT NULL,
  encrypted_payload_ref TEXT,
  processing_status TEXT NOT NULL DEFAULT 'RECEIVED' CHECK (processing_status IN (
    'RECEIVED','REJECTED','CLAIMED','UNCORRELATED','PROCESSED','DUPLICATE','DEAD'
  )),
  duplicate_count INT NOT NULL DEFAULT 0,
  attempts INT NOT NULL DEFAULT 0,
  lease_until TIMESTAMPTZ,
  leased_by TEXT,
  case_id UUID REFERENCES payment_cases(id) ON DELETE SET NULL,
  received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  processed_at TIMESTAMPTZ,
  UNIQUE (environment, provider_event_id)
);

CREATE INDEX ix_inbox_ready ON webhook_inbox(processing_status, lease_until, received_at);

-- +goose Down
DROP TABLE IF EXISTS webhook_inbox;
DROP INDEX IF EXISTS ix_case_events;
DROP INDEX IF EXISTS uq_verified_provider_events;
DROP TABLE IF EXISTS payment_events;
DROP TABLE IF EXISTS evidence_edges;
DROP INDEX IF EXISTS ix_evidence_case;
DROP TABLE IF EXISTS evidence_items;
