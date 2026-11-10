-- +goose Up
CREATE TABLE outbox (
  id BIGSERIAL PRIMARY KEY,
  tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
  case_id UUID REFERENCES payment_cases(id) ON DELETE CASCADE,
  kind TEXT NOT NULL CHECK (kind IN (
    'RECONCILE','INVESTIGATE','NOTIFY','PASSPORT_REFRESH','RETENTION','SSE_PUBLISH'
  )),
  payload JSONB NOT NULL,
  dedupe_key TEXT,
  status TEXT NOT NULL DEFAULT 'READY' CHECK (status IN ('READY','LEASED','DONE','DEAD')),
  attempts INT NOT NULL DEFAULT 0,
  max_attempts INT NOT NULL DEFAULT 8,
  run_after TIMESTAMPTZ NOT NULL DEFAULT now(),
  lease_until TIMESTAMPTZ,
  leased_by TEXT,
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_outbox_dedupe ON outbox(kind, dedupe_key)
  WHERE dedupe_key IS NOT NULL AND status IN ('READY','LEASED');

CREATE INDEX ix_outbox_ready ON outbox(status, run_after);

CREATE TABLE dead_letters (
  id BIGSERIAL PRIMARY KEY,
  outbox_id BIGINT REFERENCES outbox(id) ON DELETE CASCADE,
  tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
  case_id UUID REFERENCES payment_cases(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  payload JSONB NOT NULL,
  last_error TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS dead_letters;
DROP INDEX IF EXISTS ix_outbox_ready;
DROP INDEX IF EXISTS uq_outbox_dedupe;
DROP TABLE IF EXISTS outbox;
