// sudoapi: Model catalog.

package service_model_catalog

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func NewModelCatalogMetadataService(repo MetadataRepository, channelSvc *service.ChannelService, pricingSvc PricingService) *MetadataService {
	return &MetadataService{repo: repo, channelSvc: channelSvc, pricingSvc: pricingSvc}
}

type MetadataService struct {
	repo       MetadataRepository
	channelSvc *service.ChannelService
	pricingSvc PricingService
}

func (s *MetadataService) GetOverridesByModelNames(ctx context.Context, modelNames []string) (map[string]*MetadataOverride, error) {
	if s == nil || s.repo == nil || len(modelNames) == 0 {
		return map[string]*MetadataOverride{}, nil
	}
	return s.repo.GetByModelNames(ctx, modelNames)
}

func (s *MetadataService) ListForAdmin(ctx context.Context) ([]MetadataListItem, error) {
	if s == nil {
		return nil, nil
	}
	channels, err := s.channelSvc.ListAvailable(ctx)
	if err != nil {
		return nil, fmt.Errorf("list current model catalog metadata targets: %w", err)
	}

	type currentModel struct {
		name      string
		platforms map[string]struct{}
	}
	current := make(map[string]*currentModel)
	for _, ch := range channels {
		if ch.Status != service.StatusActive {
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

	out := make([]MetadataListItem, 0, len(current))
	for key, item := range current {
		metadata := s.effectiveView(item.name, overrides[key])
		platforms := make([]string, 0, len(item.platforms))
		for p := range item.platforms {
			platforms = append(platforms, p)
		}
		sort.Strings(platforms)
		out = append(out, MetadataListItem{
			ModelName:     item.name,
			Platforms:     platforms,
			Metadata:      metadata,
			Override:      cloneModelCatalogMetadataOverride(overrides[key]),
			MissingFields: missingModelCatalogMetadataFields(item.name, metadata, overrides[key]),
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].ModelName) < strings.ToLower(out[j].ModelName)
	})
	return out, nil
}

func (s *MetadataService) Upsert(ctx context.Context, override *MetadataOverride) (*MetadataOverride, error) {
	if override == nil {
		return nil, ErrMetadataMissingModel
	}
	cleaned, err := cleanModelCatalogMetadataOverride(*override)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Upsert(ctx, &cleaned); err != nil {
		return nil, fmt.Errorf("upsert model catalog metadata: %w", err)
	}
	return &cleaned, nil
}

func (s *MetadataService) DeleteByModelName(ctx context.Context, modelName string) error {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return ErrMetadataMissingModel
	}
	if err := s.repo.DeleteByModelName(ctx, modelName); err != nil {
		return fmt.Errorf("delete model catalog metadata: %w", err)
	}
	return nil
}

func (s *MetadataService) effectiveView(modelName string, override *MetadataOverride) MetadataView {
	cb := newCardBuilder(modelName)
	fillModelCatalogMetadata(s.pricingSvc, cb)
	applyModelCatalogMetadataOverride(cb, override)
	return MetadataView{
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

var (
	ErrMetadataNotFound     = infraerrors.NotFound("MODEL_CATALOG_METADATA_NOT_FOUND", "model catalog metadata override not found")
	ErrMetadataMissingModel = infraerrors.BadRequest("MODEL_CATALOG_METADATA_MODEL_REQUIRED", "model name is required")
	ErrMetadataInvalidField = infraerrors.BadRequest("MODEL_CATALOG_METADATA_INVALID_FIELD", "model catalog metadata field is invalid")

	categoryPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,49}$`)
)

type (
	MetadataRepository interface {
		List(ctx context.Context) ([]*MetadataOverride, error)
		GetByModelNames(ctx context.Context, modelNames []string) (map[string]*MetadataOverride, error)
		Upsert(ctx context.Context, override *MetadataOverride) error
		DeleteByModelName(ctx context.Context, modelName string) error
	}
	MetadataOverrideReader interface {
		GetOverridesByModelNames(ctx context.Context, modelNames []string) (map[string]*MetadataOverride, error)
	}
	// MetadataOverride is an admin-maintained presentation override for /models.
	MetadataOverride struct {
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
	MetadataView struct {
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

	MetadataListItem struct {
		ModelName     string
		Platforms     []string
		Metadata      MetadataView
		Override      *MetadataOverride
		MissingFields []string
	}
)

func cleanModelCatalogMetadataOverride(in MetadataOverride) (MetadataOverride, error) {
	in.ModelName = strings.TrimSpace(in.ModelName)
	if in.ModelName == "" {
		return in, ErrMetadataMissingModel
	}
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.Description = strings.TrimSpace(in.Description)
	in.Category = strings.TrimSpace(in.Category)
	in.ModelType = strings.TrimSpace(strings.ToLower(in.ModelType))
	in.IconURL = strings.TrimSpace(in.IconURL)
	if in.ContextWindow < 0 || in.MaxOutput < 0 {
		return in, ErrMetadataInvalidField.WithMetadata(map[string]string{"field": "context_window|max_output"})
	}
	if in.Category != "" && !categoryPattern.MatchString(in.Category) {
		return in, ErrMetadataInvalidField.WithMetadata(map[string]string{"field": "category"})
	}
	in.Capabilities = cleanStringList(in.Capabilities)
	in.InputModalities = cleanStringList(in.InputModalities)
	in.OutputModalities = cleanStringList(in.OutputModalities)
	in.SupportFlags = cleanStringList(in.SupportFlags)
	return in, nil
}

func cleanStringList(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.ToLower(strings.TrimSpace(item))
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

func missingModelCatalogMetadataFields(modelName string, metadata MetadataView, override *MetadataOverride) []string {
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

func cloneModelCatalogMetadataOverride(in *MetadataOverride) *MetadataOverride {
	if in == nil {
		return nil
	}
	return &MetadataOverride{
		ID:               in.ID,
		ModelName:        in.ModelName,
		DisplayName:      in.DisplayName,
		Description:      in.Description,
		Category:         in.Category,
		ModelType:        in.ModelType,
		ContextWindow:    in.ContextWindow,
		MaxOutput:        in.MaxOutput,
		Capabilities:     slices.Clone(in.Capabilities),
		InputModalities:  slices.Clone(in.InputModalities),
		OutputModalities: slices.Clone(in.OutputModalities),
		SupportFlags:     slices.Clone(in.SupportFlags),
		Featured:         in.Featured,
		IconURL:          in.IconURL,
		CreatedAt:        in.CreatedAt,
		UpdatedAt:        in.UpdatedAt,
	}
}
