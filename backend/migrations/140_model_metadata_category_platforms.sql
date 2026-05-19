-- Model metadata category now stores a platform key selected from configured
-- platforms. Keep the database constraint format-based so new platform keys can
-- be introduced by configuration without another migration.

ALTER TABLE model_metadata_overrides
    DROP CONSTRAINT IF EXISTS model_metadata_overrides_category_check;

ALTER TABLE model_metadata_overrides
    ADD CONSTRAINT model_metadata_overrides_category_check
        CHECK (category = '' OR category ~ '^[a-z0-9][a-z0-9_-]{0,49}$');
