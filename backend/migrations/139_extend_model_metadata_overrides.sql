-- Extend /models display metadata overrides with LiteLLM model type,
-- modality, and full supports_* flag metadata.

ALTER TABLE model_metadata_overrides
    ADD COLUMN IF NOT EXISTS model_type TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS input_modalities JSONB NULL,
    ADD COLUMN IF NOT EXISTS output_modalities JSONB NULL,
    ADD COLUMN IF NOT EXISTS support_flags JSONB NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'model_metadata_overrides_input_modalities_array'
    ) THEN
        ALTER TABLE model_metadata_overrides
            ADD CONSTRAINT model_metadata_overrides_input_modalities_array
                CHECK (input_modalities IS NULL OR jsonb_typeof(input_modalities) = 'array');
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'model_metadata_overrides_output_modalities_array'
    ) THEN
        ALTER TABLE model_metadata_overrides
            ADD CONSTRAINT model_metadata_overrides_output_modalities_array
                CHECK (output_modalities IS NULL OR jsonb_typeof(output_modalities) = 'array');
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'model_metadata_overrides_support_flags_array'
    ) THEN
        ALTER TABLE model_metadata_overrides
            ADD CONSTRAINT model_metadata_overrides_support_flags_array
                CHECK (support_flags IS NULL OR jsonb_typeof(support_flags) = 'array');
    END IF;
END $$;
