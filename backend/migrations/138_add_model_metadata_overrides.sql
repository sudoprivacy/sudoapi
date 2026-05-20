-- Admin-maintained display metadata for /models.
--
-- The model list, platforms, groups, and prices still come from channels. This
-- table only stores presentation overrides for fields that otherwise come from
-- LiteLLM metadata, name inference, or have no input source at all.

CREATE TABLE IF NOT EXISTS model_metadata_overrides (
    id             BIGSERIAL    PRIMARY KEY,
    model_name     VARCHAR(200) NOT NULL,
    display_name   VARCHAR(200) NOT NULL DEFAULT '',
    description    TEXT         NOT NULL DEFAULT '',
    category       VARCHAR(40)  NOT NULL DEFAULT '',
    context_window INTEGER      NOT NULL DEFAULT 0,
    max_output     INTEGER      NOT NULL DEFAULT 0,
    capabilities   JSONB        NULL,
    featured       BOOLEAN      NOT NULL DEFAULT FALSE,
    icon_url       TEXT         NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT model_metadata_overrides_model_name_nonempty
        CHECK (btrim(model_name) <> ''),
    CONSTRAINT model_metadata_overrides_category_check
        CHECK (category = '' OR category IN ('claude', 'gpt', 'gemini', 'image', 'embedding', 'audio', 'other')),
    CONSTRAINT model_metadata_overrides_context_window_nonnegative
        CHECK (context_window >= 0),
    CONSTRAINT model_metadata_overrides_max_output_nonnegative
        CHECK (max_output >= 0),
    CONSTRAINT model_metadata_overrides_capabilities_array
        CHECK (capabilities IS NULL OR jsonb_typeof(capabilities) = 'array')
);

CREATE UNIQUE INDEX IF NOT EXISTS model_metadata_overrides_model_name_lower
    ON model_metadata_overrides (lower(model_name));

CREATE INDEX IF NOT EXISTS idx_model_metadata_overrides_featured
    ON model_metadata_overrides (featured)
    WHERE featured = TRUE;
