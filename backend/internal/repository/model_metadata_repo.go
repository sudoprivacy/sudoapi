package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type modelMetadataRepository struct {
	db *sql.DB
}

func NewModelMetadataRepository(db *sql.DB) service.ModelMetadataRepository {
	return &modelMetadataRepository{db: db}
}

func (r *modelMetadataRepository) List(ctx context.Context) ([]*service.ModelMetadataOverride, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, model_name, display_name, description, category, model_type, context_window,
		       max_output, capabilities::text, input_modalities::text, output_modalities::text,
		       support_flags::text, featured, icon_url, created_at, updated_at
		FROM model_metadata_overrides
		ORDER BY lower(model_name)
	`)
	if err != nil {
		return nil, fmt.Errorf("list model metadata overrides: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanModelMetadataRows(rows)
}

func (r *modelMetadataRepository) GetByModelNames(ctx context.Context, modelNames []string) (map[string]*service.ModelMetadataOverride, error) {
	keys := uniqueMetadataKeys(modelNames)
	out := make(map[string]*service.ModelMetadataOverride, len(keys))
	if len(keys) == 0 {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, model_name, display_name, description, category, model_type, context_window,
		       max_output, capabilities::text, input_modalities::text, output_modalities::text,
		       support_flags::text, featured, icon_url, created_at, updated_at
		FROM model_metadata_overrides
		WHERE lower(model_name) = ANY($1)
	`, pq.Array(keys))
	if err != nil {
		return nil, fmt.Errorf("get model metadata overrides: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items, err := scanModelMetadataRows(rows)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		out[strings.ToLower(strings.TrimSpace(item.ModelName))] = item
	}
	return out, nil
}

func (r *modelMetadataRepository) Upsert(ctx context.Context, override *service.ModelMetadataOverride) error {
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
		INSERT INTO model_metadata_overrides (
			model_name, display_name, description, category, model_type, context_window,
			max_output, capabilities, input_modalities, output_modalities,
			support_flags, featured, icon_url
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10::jsonb, $11::jsonb, $12, $13)
		ON CONFLICT ((lower(model_name))) DO UPDATE SET
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
		return fmt.Errorf("upsert model metadata override: %w", err)
	}
	return nil
}

func (r *modelMetadataRepository) DeleteByModelName(ctx context.Context, modelName string) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM model_metadata_overrides
		WHERE lower(model_name) = lower($1)
	`, strings.TrimSpace(modelName))
	if err != nil {
		return fmt.Errorf("delete model metadata override: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return service.ErrModelMetadataNotFound
	}
	return nil
}

func scanModelMetadataRows(rows *sql.Rows) ([]*service.ModelMetadataOverride, error) {
	out := make([]*service.ModelMetadataOverride, 0)
	for rows.Next() {
		item := &service.ModelMetadataOverride{}
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
			return nil, fmt.Errorf("scan model metadata override: %w", err)
		}
		if err := unmarshalNullableStringSlice(capabilities, &item.Capabilities); err != nil {
			return nil, fmt.Errorf("unmarshal model metadata capabilities: %w", err)
		}
		if err := unmarshalNullableStringSlice(inputModalities, &item.InputModalities); err != nil {
			return nil, fmt.Errorf("unmarshal model metadata input modalities: %w", err)
		}
		if err := unmarshalNullableStringSlice(outputModalities, &item.OutputModalities); err != nil {
			return nil, fmt.Errorf("unmarshal model metadata output modalities: %w", err)
		}
		if err := unmarshalNullableStringSlice(supportFlags, &item.SupportFlags); err != nil {
			return nil, fmt.Errorf("unmarshal model metadata support flags: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate model metadata overrides: %w", err)
	}
	return out, nil
}

func uniqueMetadataKeys(modelNames []string) []string {
	seen := make(map[string]struct{}, len(modelNames))
	out := make([]string, 0, len(modelNames))
	for _, name := range modelNames {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func marshalStringSliceNullable(values []string) (any, error) {
	if len(values) == 0 {
		return nil, nil
	}
	body, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("marshal model metadata string slice: %w", err)
	}
	return string(body), nil
}

func unmarshalNullableStringSlice(value sql.NullString, out *[]string) error {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}
	return json.Unmarshal([]byte(value.String), out)
}
