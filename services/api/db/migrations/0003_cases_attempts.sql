-- +goose Up
CREATE TABLE payment_cases (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  external_reference TEXT,
  sender_user_id UUID NOT NULL REFERENCES users(id),
  receiver_user_id UUID REFERENCES users(id),
  sender_country CHAR(2) NOT NULL,
  receiver_country CHAR(2) NOT NULL,
  payment_purpose TEXT NOT NULL,
  amount NUMERIC(20,6) NOT NULL CHECK (amount > 0),
  currency CHAR(3) NOT NULL,
  environment TEXT NOT NULL CHECK (environment IN ('REAL_SANDBOX','PAYPAL_SIMULATOR','LOCAL_SIMULATION','PRODUCTION')),
  provider_state TEXT NOT NULL DEFAULT 'NOT_CREATED' CHECK (provider_state IN (
    'NOT_CREATED','CREATED','APPROVED','AUTHORISED','CAPTURE_PENDING','CAPTURE_COMPLETED',
    'CAPTURE_DENIED','REFUNDED','PARTIALLY_REFUNDED','UNKNOWN'
  )),
  last_confirmed_provider_state TEXT NOT NULL DEFAULT 'NOT_CREATED',
  provider_state_evidence_id UUID,
  provider_state_observed_at TIMESTAMPTZ,
  application_state TEXT NOT NULL DEFAULT 'SYNCHRONISED' CHECK (application_state IN (
    'SYNCHRONISED','WAITING_FOR_WEBHOOK','API_TIMEOUT','RECONCILING','EVENT_GAP','INCONSISTENT'
  )),
  receiver_state TEXT NOT NULL DEFAULT 'NOT_OBSERVABLE' CHECK (receiver_state IN (
    'NOT_APPLICABLE','NOT_OBSERVABLE','REPORTED_PENDING','VERIFIED_AVAILABLE',
    'WITHDRAWAL_IN_PROGRESS','VERIFIED_BANK_CREDITED'
  )),
  case_status TEXT NOT NULL DEFAULT 'OPEN' CHECK (case_status IN (
    'OPEN','OBSERVING','INVESTIGATING','GUIDANCE_READY','AWAITING_USER_ACTION',
    'VERIFYING_RESOLUTION','RESOLVED','NEEDS_HUMAN_REVIEW','PAUSED_INSUFFICIENT_EVIDENCE',
    'CLOSED_UNRESOLVED','ARCHIVED','REOPENED'
  )),
  state_version BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ix_cases_tenant_status ON payment_cases(tenant_id, case_status, updated_at DESC);

CREATE TABLE payment_attempts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  case_id UUID NOT NULL REFERENCES payment_cases(id) ON DELETE CASCADE,
  attempt_no INT NOT NULL CHECK (attempt_no > 0),
  provider TEXT NOT NULL DEFAULT 'PAYPAL',
  environment TEXT NOT NULL,
  order_operation_request_id TEXT NOT NULL,
  capture_operation_request_id TEXT,
  status TEXT NOT NULL DEFAULT 'ORDER_CREATING' CHECK (status IN (
    'ORDER_CREATING','ORDER_CREATED','ORDER_UNKNOWN','APPROVAL_PENDING','APPROVED','CANCELLED',
    'CAPTURE_REQUESTED','CAPTURE_COMPLETED','CAPTURE_PENDING','CAPTURE_DENIED','CAPTURE_UNKNOWN',
    'REVIEW_REQUIRED','APPLICATION_RECONCILED','APPLICATION_MISMATCH'
  )),
  terminal BOOLEAN GENERATED ALWAYS AS (status IN ('CANCELLED','CAPTURE_DENIED','APPLICATION_RECONCILED')) STORED,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, case_id, attempt_no),
  UNIQUE (tenant_id, provider, environment, order_operation_request_id)
);

CREATE UNIQUE INDEX uq_one_active_attempt ON payment_attempts(case_id) WHERE terminal = FALSE;

CREATE TABLE provider_resources (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  attempt_id UUID NOT NULL REFERENCES payment_attempts(id) ON DELETE CASCADE,
  resource_type TEXT NOT NULL CHECK (resource_type IN ('ORDER','AUTHORIZATION','CAPTURE','REFUND')),
  provider_resource_id TEXT NOT NULL,
  environment TEXT NOT NULL,
  provider_status TEXT,
  last_verified_at TIMESTAMPTZ,
  UNIQUE (environment, resource_type, provider_resource_id)
);

CREATE TABLE idempotency_keys (
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  actor_id UUID NOT NULL REFERENCES users(id),
  route TEXT NOT NULL,
  key TEXT NOT NULL,
  request_hash BYTEA NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('IN_FLIGHT','DONE')),
  response_code INT,
  response_body JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (tenant_id, actor_id, route, key)
);

-- +goose Down
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS provider_resources;
DROP INDEX IF EXISTS uq_one_active_attempt;
DROP TABLE IF EXISTS payment_attempts;
DROP INDEX IF EXISTS ix_cases_tenant_status;
DROP TABLE IF EXISTS payment_cases;
