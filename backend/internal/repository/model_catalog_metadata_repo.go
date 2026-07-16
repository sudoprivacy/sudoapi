// sudoapi: Model catalog.

package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service_model_catalog"

	"github.com/lib/pq"
)

func NewModelCatalogMetadataRepository(db *sql.DB) service_model_catalog.MetadataRepository {
	return &ModelCatalogMetadataRepository{db: db}
}

type ModelCatalogMetadataRepository struct {
	db *sql.DB
}

func (r *ModelCatalogMetadataRepository) List(ctx context.Context) ([]*service_model_catalog.MetadataOverride, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, model_name, display_name, description, category, model_type, context_window,
		       max_output, capabilities::text, input_modalities::text, output_modalities::text,
		       support_flags::text, featured, icon_url, created_at, updated_at
		FROM model_catalog_metadata_overrides
		ORDER BY LOWER(model_name)
	`)
	if err != nil {
		return nil, fmt.Errorf("list model catalog metadata overrides: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanModelCatalogMetadataRows(rows)
}

func (r *ModelCatalogMetadataRepository) GetByModelNames(ctx context.Context, modelNames []string) (map[string]*service_model_catalog.MetadataOverride, error) {
	keys := normalizeModelCatalogMetadataKeys(modelNames)
	out := make(map[string]*service_model_catalog.MetadataOverride, len(keys))
	if len(keys) == 0 {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, model_name, display_name, description, category, model_type, context_window,
		       max_output, capabilities::text, input_modalities::text, output_modalities::text,
		       support_flags::text, featured, icon_url, created_at, updated_at
		FROM model_catalog_metadata_overrides
		WHERE LOWER(model_name) = ANY($1)
	`, pq.Array(keys))
	if err != nil {
		return nil, fmt.Errorf("get model catalog metadata overrides: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items, err := scanModelCatalogMetadataRows(rows)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		out[strings.ToLower(strings.TrimSpace(item.ModelName))] = item
	}
	return out, nil
}

func (r *ModelCatalogMetadataRepository) Upsert(ctx context.Context, override *service_model_catalog.MetadataOverride) error {
	capabilitiesJSON, err := marshalStringSliceNullable(override.Capabilities)
	if err != nil {
		return err
	}
	inputModalitiesJSON, err := marshalStringSliceNullable(override.InputModalities)
	if err != nil {
		return err
	}
	outputModalitiesJSON, err := marshalStringSliceNullable(override.OutputModalities)
	if err != nil {
		return err
	}
	supportFlagsJSON, err := marshalStringSliceNullable(override.SupportFlags)
	if err != nil {
		return err
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO model_catalog_metadata_overrides (
			model_name, display_name, description, category, model_type, context_window,
			max_output, capabilities, input_modalities, output_modalities,
			support_flags, featured, icon_url
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10::jsonb, $11::jsonb, $12, $13)
		ON CONFLICT ((LOWER(model_name))) DO UPDATE SET
			model_name = EXCLUDED.model_name,
			display_name = EXCLUDED.display_name,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			model_type = EXCLUDED.model_type,
			context_window = EXCLUDED.context_window,
			max_output = EXCLUDED.max_output,
			capabilities = EXCLUDED.capabilities,
			input_modalities = EXCLUDED.input_modalities,
			output_modalities = EXCLUDED.output_modalities,
			support_flags = EXCLUDED.support_flags,
			featured = EXCLUDED.featured,
			icon_url = EXCLUDED.icon_url,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`, override.ModelName, override.DisplayName, override.Description, override.Category,
		override.ModelType, override.ContextWindow, override.MaxOutput, capabilitiesJSON,
		inputModalitiesJSON, outputModalitiesJSON, supportFlagsJSON, override.Featured, override.IconURL)
	if err := row.Scan(&override.ID, &override.CreatedAt, &override.UpdatedAt); err != nil {
		return fmt.Errorf("upsert model catalog metadata override: %w", err)
	}
	return nil
}

func (r *ModelCatalogMetadataRepository) DeleteByModelName(ctx context.Context, modelName string) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM model_catalog_metadata_overrides
		WHERE LOWER(model_name) = LOWER($1)
	`, strings.TrimSpace(modelName))
	if err != nil {
		return fmt.Errorf("delete model catalog metadata override: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return service_model_catalog.ErrMetadataNotFound
	}
	return nil
}

func scanModelCatalogMetadataRows(rows *sql.Rows) ([]*service_model_catalog.MetadataOverride, error) {
	out := make([]*service_model_catalog.MetadataOverride, 0)
	for rows.Next() {
		item := &service_model_catalog.MetadataOverride{}
		var capabilities, inputModalities, outputModalities, supportFlags sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.ModelName,
			&item.DisplayName,
			&item.Description,
			&item.Category,
			&item.ModelType,
			&item.ContextWindow,
			&item.MaxOutput,
			&capabilities,
			&inputModalities,
			&outputModalities,
			&supportFlags,
			&item.Featured,
			&item.IconURL,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan model catalog metadata override: %w", err)
		}
		if err := unmarshalNullableStringSlice(capabilities, &item.Capabilities); err != nil {
			return nil, fmt.Errorf("unmarshal model catalog metadata capabilities: %w", err)
		}
		if err := unmarshalNullableStringSlice(inputModalities, &item.InputModalities); err != nil {
			return nil, fmt.Errorf("unmarshal model catalog metadata input modalities: %w", err)
		}
		if err := unmarshalNullableStringSlice(outputModalities, &item.OutputModalities); err != nil {
			return nil, fmt.Errorf("unmarshal model catalog metadata output modalities: %w", err)
		}
		if err := unmarshalNullableStringSlice(supportFlags, &item.SupportFlags); err != nil {
			return nil, fmt.Errorf("unmarshal model catalog metadata support flags: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate model catalog metadata overrides: %w", err)
	}
	return out, nil
}

func marshalStringSliceNullable(values []string) (any, error) {
	if len(values) == 0 {
		return nil, nil
	}
	body, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("marshal model catalog metadata string slice: %w", err)
	}
	return string(body), nil
}

func unmarshalNullableStringSlice(value sql.NullString, out *[]string) error {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}
	return json.Unmarshal([]byte(value.String), out)
}

func normalizeModelCatalogMetadataKeys(modelNames []string) []string {
	seen := make(map[string]struct{}, len(modelNames))
	keys := make([]string, 0, len(modelNames))
	for _, modelName := range modelNames {
		key := strings.ToLower(strings.TrimSpace(modelName))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	return keys
}
