-- +goose Up
CREATE TABLE passport_versions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  case_id UUID NOT NULL REFERENCES payment_cases(id) ON DELETE CASCADE,
  viewer_role TEXT NOT NULL CHECK (viewer_role IN ('sender','receiver','ops')),
  version INT NOT NULL CHECK (version > 0),
  snapshot_version BIGINT NOT NULL,
  content JSONB NOT NULL,
  content_sha256 CHAR(64) NOT NULL,
  prev_sha256 CHAR(64),
  evidence_ids UUID[] NOT NULL DEFAULT '{}',
  policy_version INT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (case_id, viewer_role, version)
);

CREATE TABLE passport_shares (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  case_id UUID NOT NULL REFERENCES payment_cases(id) ON DELETE CASCADE,
  viewer_role TEXT NOT NULL CHECK (viewer_role IN ('sender','receiver','ops')),
  token_hash BYTEA NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE invitations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  case_id UUID NOT NULL REFERENCES payment_cases(id) ON DELETE CASCADE,
  invitee_email CITEXT NOT NULL,
  token_hash BYTEA NOT NULL UNIQUE,
  proposed_fields TEXT[] NOT NULL DEFAULT '{}',
  expires_at TIMESTAMPTZ NOT NULL,
  accepted_at TIMESTAMPTZ,
  declined_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE consent_grants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  case_id UUID NOT NULL REFERENCES payment_cases(id) ON DELETE CASCADE,
  grantor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  grantee_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  allowed_fields TEXT[] NOT NULL DEFAULT '{}',
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS consent_grants;
DROP TABLE IF EXISTS invitations;
DROP TABLE IF EXISTS passport_shares;
DROP TABLE IF EXISTS passport_versions;
