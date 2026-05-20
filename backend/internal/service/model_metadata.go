// sudoapi: Model market.

package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrModelMetadataNotFound     = infraerrors.NotFound("MODEL_METADATA_NOT_FOUND", "model metadata override not found")
	ErrModelMetadataMissingModel = infraerrors.BadRequest("MODEL_METADATA_MODEL_REQUIRED", "model name is required")
	ErrModelMetadataInvalidField = infraerrors.BadRequest("MODEL_METADATA_INVALID_FIELD", "model metadata field is invalid")
)

// ModelMetadataOverride is an admin-maintained presentation override for /models.
type ModelMetadataOverride struct {
	ID               int64
	ModelName        string
	DisplayName      string
	Description      string
	Category         string
	ModelType        string
	ContextWindow    int
	MaxOutput        int
	Capabilities     []string
	InputModalities  []string
	OutputModalities []string
	SupportFlags     []string
	Featured         bool
	IconURL          string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ModelMetadataView struct {
	DisplayName      string
	Description      string
	Category         string
	ModelType        string
	ContextWindow    int
	MaxOutput        int
	Capabilities     []string
	InputModalities  []string
	OutputModalities []string
	SupportFlags     []string
	Featured         bool
	IconURL          string
}

type ModelMetadataListItem struct {
	ModelName     string
	Platforms     []string
	Metadata      ModelMetadataView
	Override      *ModelMetadataOverride
	MissingFields []string
}

type ModelMetadataRepository interface {
	List(ctx context.Context) ([]*ModelMetadataOverride, error)
	GetByModelNames(ctx context.Context, modelNames []string) (map[string]*ModelMetadataOverride, error)
	Upsert(ctx context.Context, override *ModelMetadataOverride) error
	DeleteByModelName(ctx context.Context, modelName string) error
}

type ModelMetadataOverrideReader interface {
	GetOverridesByModelNames(ctx context.Context, modelNames []string) (map[string]*ModelMetadataOverride, error)
}

type ModelMetadataService struct {
	repo       ModelMetadataRepository
	channelSvc *ChannelService
	pricingSvc *PricingService
}

func NewModelMetadataService(repo ModelMetadataRepository, channelSvc *ChannelService, pricingSvc *PricingService) *ModelMetadataService {
	return &ModelMetadataService{repo: repo, channelSvc: channelSvc, pricingSvc: pricingSvc}
}

func (s *ModelMetadataService) GetOverridesByModelNames(ctx context.Context, modelNames []string) (map[string]*ModelMetadataOverride, error) {
	if s == nil || s.repo == nil {
		return map[string]*ModelMetadataOverride{}, nil
	}
	return s.repo.GetByModelNames(ctx, modelNames)
}

func (s *ModelMetadataService) ListForAdmin(ctx context.Context) ([]ModelMetadataListItem, error) {
	if s == nil {
		return nil, nil
	}
	channels, err := s.channelSvc.ListAvailable(ctx)
	if err != nil {
		return nil, fmt.Errorf("list current model metadata targets: %w", err)
	}

	type currentModel struct {
		name      string
		platforms map[string]struct{}
	}
	current := make(map[string]*currentModel)
	for _, ch := range channels {
		if ch.Status != StatusActive {
			continue
		}
		for _, m := range ch.SupportedModels {
			name := strings.TrimSpace(m.Name)
			if name == "" {
				continue
			}
			key := normalizeMetadataModelKey(name)
			item := current[key]
			if item == nil {
				item = &currentModel{name: name, platforms: make(map[string]struct{})}
				current[key] = item
			}
			if m.Platform != "" {
				item.platforms[m.Platform] = struct{}{}
			}
		}
	}

	names := make([]string, 0, len(current))
	for _, item := range current {
		names = append(names, item.name)
	}
	overrides, err := s.GetOverridesByModelNames(ctx, names)
	if err != nil {
		return nil, err
	}

	out := make([]ModelMetadataListItem, 0, len(current))
	for key, item := range current {
		metadata := s.effectiveView(item.name, overrides[key])
		platforms := make([]string, 0, len(item.platforms))
		for p := range item.platforms {
			platforms = append(platforms, p)
		}
		sort.Strings(platforms)
		out = append(out, ModelMetadataListItem{
			ModelName:     item.name,
			Platforms:     platforms,
			Metadata:      metadata,
			Override:      cloneModelMetadataOverride(overrides[key]),
			MissingFields: missingModelMetadataFields(item.name, metadata, overrides[key]),
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].ModelName) < strings.ToLower(out[j].ModelName)
	})
	return out, nil
}

func (s *ModelMetadataService) Upsert(ctx context.Context, override *ModelMetadataOverride) (*ModelMetadataOverride, error) {
	if override == nil {
		return nil, ErrModelMetadataMissingModel
	}
	cleaned, err := cleanModelMetadataOverride(*override)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Upsert(ctx, &cleaned); err != nil {
		return nil, fmt.Errorf("upsert model metadata: %w", err)
	}
	return &cleaned, nil
}

func (s *ModelMetadataService) DeleteByModelName(ctx context.Context, modelName string) error {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return ErrModelMetadataMissingModel
	}
	if err := s.repo.DeleteByModelName(ctx, modelName); err != nil {
		return fmt.Errorf("delete model metadata: %w", err)
	}
	return nil
}

func (s *ModelMetadataService) effectiveView(modelName string, override *ModelMetadataOverride) ModelMetadataView {
	cb := newCardBuilder(modelName)
	fillModelSquareMetadata(s.pricingSvc, cb)
	applyModelMetadataOverride(cb, override)
	return ModelMetadataView{
		DisplayName:      cb.DisplayName,
		Description:      cb.Description,
		Category:         cb.Category,
		ModelType:        cb.ModelType,
		ContextWindow:    cb.ContextWindow,
		MaxOutput:        cb.MaxOutput,
		Capabilities:     append([]string(nil), cb.Capabilities...),
		InputModalities:  append([]string(nil), cb.InputModalities...),
		OutputModalities: append([]string(nil), cb.OutputModalities...),
		SupportFlags:     append([]string(nil), cb.SupportFlags...),
		Featured:         cb.Featured,
		IconURL:          cb.IconURL,
	}
}

func cleanModelMetadataOverride(in ModelMetadataOverride) (ModelMetadataOverride, error) {
	in.ModelName = strings.TrimSpace(in.ModelName)
	if in.ModelName == "" {
		return in, ErrModelMetadataMissingModel
	}
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.Description = strings.TrimSpace(in.Description)
	in.Category = strings.TrimSpace(strings.ToLower(in.Category))
	in.ModelType = strings.TrimSpace(strings.ToLower(in.ModelType))
	in.IconURL = strings.TrimSpace(in.IconURL)
	if in.ContextWindow < 0 || in.MaxOutput < 0 {
		return in, ErrModelMetadataInvalidField.WithMetadata(map[string]string{"field": "context_window|max_output"})
	}
	if in.Category != "" && !isValidModelMetadataCategory(in.Category) {
		return in, ErrModelMetadataInvalidField.WithMetadata(map[string]string{"field": "category"})
	}
	in.Capabilities = cleanCapabilities(in.Capabilities)
	in.InputModalities = cleanMetadataStringList(in.InputModalities, true)
	in.OutputModalities = cleanMetadataStringList(in.OutputModalities, true)
	in.SupportFlags = cleanMetadataStringList(in.SupportFlags, true)
	return in, nil
}

func cleanCapabilities(in []string) []string {
	return cleanMetadataStringList(in, true)
}

func cleanMetadataStringList(in []string, lower bool) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if lower {
			item = strings.ToLower(item)
		}
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func isValidModelMetadataCategory(category string) bool {
	if len(category) > 50 {
		return false
	}
	for i, r := range category {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-'
		if !valid {
			return false
		}
		if i == 0 && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func missingModelMetadataFields(modelName string, metadata ModelMetadataView, override *ModelMetadataOverride) []string {
	missing := make([]string, 0, 7)
	if override == nil || strings.TrimSpace(override.DisplayName) == "" || strings.EqualFold(metadata.DisplayName, modelName) {
		missing = append(missing, "display_name")
	}
	if strings.TrimSpace(metadata.Description) == "" {
		missing = append(missing, "description")
	}
	if strings.TrimSpace(metadata.Category) == "" || metadata.Category == "other" {
		missing = append(missing, "category")
	}
	if metadata.ContextWindow <= 0 {
		missing = append(missing, "context_window")
	}
	if metadata.MaxOutput <= 0 {
		missing = append(missing, "max_output")
	}
	if len(metadata.Capabilities) == 0 {
		missing = append(missing, "capabilities")
	}
	if strings.TrimSpace(metadata.ModelType) == "" {
		missing = append(missing, "model_type")
	}
	if len(metadata.InputModalities) == 0 {
		missing = append(missing, "input_modalities")
	}
	if len(metadata.OutputModalities) == 0 {
		missing = append(missing, "output_modalities")
	}
	if len(metadata.SupportFlags) == 0 {
		missing = append(missing, "support_flags")
	}
	if strings.TrimSpace(metadata.IconURL) == "" {
		missing = append(missing, "icon_url")
	}
	return missing
}

func normalizeMetadataModelKey(modelName string) string {
	return strings.ToLower(strings.TrimSpace(modelName))
}

func cloneModelMetadataOverride(in *ModelMetadataOverride) *ModelMetadataOverride {
	if in == nil {
		return nil
	}
	out := *in
	if in.Capabilities != nil {
		out.Capabilities = append([]string(nil), in.Capabilities...)
	}
	if in.InputModalities != nil {
		out.InputModalities = append([]string(nil), in.InputModalities...)
	}
	if in.OutputModalities != nil {
		out.OutputModalities = append([]string(nil), in.OutputModalities...)
	}
	if in.SupportFlags != nil {
		out.SupportFlags = append([]string(nil), in.SupportFlags...)
	}
	return &out
}
