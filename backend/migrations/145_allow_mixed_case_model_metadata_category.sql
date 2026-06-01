-- sudoapi: Model market.

-- Preserve the admin-entered vendor/category casing for /models display while
-- keeping the same character set and length validation.

ALTER TABLE model_metadata_overrides
    ALTER COLUMN category TYPE VARCHAR(50);

ALTER TABLE model_metadata_overrides
    DROP CONSTRAINT IF EXISTS model_metadata_overrides_category_check;

ALTER TABLE model_metadata_overrides
    ADD CONSTRAINT model_metadata_overrides_category_check
        CHECK (category = '' OR category ~ '^[A-Za-z0-9][A-Za-z0-9_-]{0,49}$');
