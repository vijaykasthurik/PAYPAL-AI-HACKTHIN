-- +goose Up
CREATE TABLE investigations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  case_id UUID NOT NULL REFERENCES payment_cases(id) ON DELETE CASCADE,
  trigger TEXT NOT NULL,
  snapshot_version BIGINT NOT NULL,
  status TEXT NOT NULL CHECK (status IN (
    'QUEUED','RUNNING','COMPLETED','FAILED_FALLBACK','STALE_DISCARDED','NEEDS_HUMAN_REVIEW'
  )),
  llm_available BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ
);

CREATE INDEX ix_investigations_case ON investigations(case_id, created_at DESC);

CREATE TABLE agent_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  investigation_id UUID NOT NULL REFERENCES investigations(id) ON DELETE CASCADE,
  agent_id TEXT NOT NULL CHECK (agent_id IN (
    'A01','A02','A03','A04','A05','A06','A07','A08','A09','A10'
  )),
  model_id TEXT,
  prompt_sha256 CHAR(64),
  input_evidence_ids UUID[] NOT NULL DEFAULT '{}',
  output JSONB,
  tokens_in INT,
  tokens_out INT,
  latency_ms INT,
  rejection_reasons TEXT[] NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE diagnosis_claims (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  case_id UUID NOT NULL REFERENCES payment_cases(id) ON DELETE CASCADE,
  investigation_id UUID NOT NULL REFERENCES investigations(id) ON DELETE CASCADE,
  kind TEXT NOT NULL CHECK (kind IN ('confirmed','inference','unknown')),
  code TEXT NOT NULL,
  statement TEXT NOT NULL,
  inference_rule TEXT,
  viewer_scope TEXT NOT NULL CHECK (viewer_scope IN ('SENDER','RECEIVER','OPS','ALL')),
  validated BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE claim_evidence (
  claim_id UUID NOT NULL REFERENCES diagnosis_claims(id) ON DELETE CASCADE,
  evidence_id UUID NOT NULL REFERENCES evidence_items(id) ON DELETE CASCADE,
  PRIMARY KEY (claim_id, evidence_id)
);

CREATE TABLE recovery_actions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  case_id UUID NOT NULL REFERENCES payment_cases(id) ON DELETE CASCADE,
  investigation_id UUID NOT NULL REFERENCES investigations(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'RECOMMENDED' CHECK (status IN ('RECOMMENDED','APPROVED','ACKNOWLEDGED','SUPERSEDED','BLOCKED')),
  rationale TEXT NOT NULL,
  designated_actor TEXT NOT NULL,
  preconditions_met BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE action_acks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  action_id UUID NOT NULL REFERENCES recovery_actions(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  note TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS action_acks;
DROP TABLE IF EXISTS recovery_actions;
DROP TABLE IF EXISTS claim_evidence;
DROP TABLE IF EXISTS diagnosis_claims;
DROP TABLE IF EXISTS agent_runs;
DROP INDEX IF EXISTS ix_investigations_case;
DROP TABLE IF EXISTS investigations;
