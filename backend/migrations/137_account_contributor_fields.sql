-- Add contributor ownership and review workflow fields to accounts.

ALTER TABLE accounts ADD COLUMN IF NOT EXISTS owner_user_id BIGINT DEFAULT NULL;
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS review_status VARCHAR(20) NOT NULL DEFAULT 'approved';

COMMENT ON COLUMN accounts.owner_user_id IS 'User ID of the external contributor who owns this account. NULL for internal/admin accounts.';
COMMENT ON COLUMN accounts.review_status IS 'Review status for contributor-submitted accounts: pending, approved, rejected.';

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'accounts_owner_user_id_fkey'
  ) THEN
    ALTER TABLE accounts
      ADD CONSTRAINT accounts_owner_user_id_fkey
      FOREIGN KEY (owner_user_id) REFERENCES users(id)
      ON DELETE SET NULL;
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'accounts_review_status_check'
  ) THEN
    ALTER TABLE accounts
      ADD CONSTRAINT accounts_review_status_check
      CHECK (review_status IN ('pending', 'approved', 'rejected'));
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_accounts_owner_user_id ON accounts(owner_user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_accounts_review_status ON accounts(review_status) WHERE deleted_at IS NULL;
