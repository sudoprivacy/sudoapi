-- sudoapi: Contributor account self-service authorization.

-- Contributor proxy reservations prevent multiple contributors from selecting
-- the same proxy during the OAuth authorization window.

CREATE TABLE IF NOT EXISTS contributor_proxy_reservations (
    id BIGSERIAL PRIMARY KEY,
    proxy_id BIGINT NOT NULL REFERENCES proxies(id) ON DELETE CASCADE,
    owner_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    country VARCHAR(16) NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'contributor_proxy_reservations_status_check'
  ) THEN
    ALTER TABLE contributor_proxy_reservations
      ADD CONSTRAINT contributor_proxy_reservations_status_check
      CHECK (status IN ('active', 'consumed', 'released', 'expired'));
  END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_contributor_proxy_reservations_active_proxy
  ON contributor_proxy_reservations(proxy_id)
  WHERE status = 'active';

CREATE UNIQUE INDEX IF NOT EXISTS idx_contributor_proxy_reservations_active_owner
  ON contributor_proxy_reservations(owner_user_id)
  WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_contributor_proxy_reservations_owner_active
  ON contributor_proxy_reservations(owner_user_id, status, expires_at);

CREATE INDEX IF NOT EXISTS idx_contributor_proxy_reservations_expires
  ON contributor_proxy_reservations(status, expires_at);
