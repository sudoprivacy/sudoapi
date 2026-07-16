// sudoapi: Model catalog.

//go:build unit

package service_model_catalog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type stubModelCatalogMetadataRepo struct {
	items map[string]*MetadataOverride
}

func (r *stubModelCatalogMetadataRepo) List(context.Context) ([]*MetadataOverride, error) {
	out := make([]*MetadataOverride, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, cloneModelCatalogMetadataOverride(item))
	}
	return out, nil
}

func (r *stubModelCatalogMetadataRepo) GetByModelNames(_ context.Context, modelNames []string) (map[string]*MetadataOverride, error) {
	out := make(map[string]*MetadataOverride, len(modelNames))
	for _, name := range modelNames {
		key := normalizeMetadataModelKey(name)
		if item, ok := r.items[key]; ok {
			out[key] = cloneModelCatalogMetadataOverride(item)
		}
	}
	return out, nil
}

func (r *stubModelCatalogMetadataRepo) Upsert(_ context.Context, override *MetadataOverride) error {
	if r.items == nil {
		r.items = make(map[string]*MetadataOverride)
	}
	r.items[normalizeMetadataModelKey(override.ModelName)] = cloneModelCatalogMetadataOverride(override)
	return nil
}

func (r *stubModelCatalogMetadataRepo) DeleteByModelName(_ context.Context, modelName string) error {
	delete(r.items, normalizeMetadataModelKey(modelName))
	return nil
}

func TestModelCatalogMetadataService_ListForAdminMergesCurrentModelsAndOverrides(t *testing.T) {
	g := service.Group{ID: 1, Name: "auto", Platform: service.PlatformOpenAI, SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: 1.0, Status: service.StatusActive}
	channels := []service.Channel{{
		ID: 1, Name: "ch", Status: service.StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []service.ChannelModelPricing{{
			Platform: service.PlatformOpenAI, Models: []string{"custom-model"},
			BillingMode: service.BillingModeToken, InputPrice: new(1e-6),
		}},
	}}
	repo := &stubModelCatalogMetadataRepo{items: map[string]*MetadataOverride{
		"custom-model": {
			ModelName:        "custom-model",
			DisplayName:      "Custom Model",
			Description:      "Ready for display",
			Category:         "gpt",
			ModelType:        "chat",
			ContextWindow:    100000,
			MaxOutput:        4096,
			Capabilities:     []string{"reasoning"},
			InputModalities:  []string{"text", "image"},
			OutputModalities: []string{"text"},
			SupportFlags:     []string{"reasoning", "vision"},
			IconURL:          "https://example.com/icon.png",
		},
	}}
	svc := NewModelCatalogMetadataService(repo, stubChannelServiceProvider(t, channels, []service.Group{g}), nil)

	items, err := svc.ListForAdmin(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "custom-model", items[0].ModelName)
	require.Equal(t, []string{service.PlatformOpenAI}, items[0].Platforms)
	require.Equal(t, "Custom Model", items[0].Metadata.DisplayName)
	require.Equal(t, "chat", items[0].Metadata.ModelType)
	require.Equal(t, []string{"text", "image"}, items[0].Metadata.InputModalities)
	require.Equal(t, []string{"text"}, items[0].Metadata.OutputModalities)
	require.Equal(t, []string{"reasoning", "vision"}, items[0].Metadata.SupportFlags)
	require.Empty(t, items[0].MissingFields)
	require.NotNil(t, items[0].Override)
}

func TestModelCatalogMetadataService_UpsertValidatesAndNormalizes(t *testing.T) {
	repo := &stubModelCatalogMetadataRepo{}
	svc := NewModelCatalogMetadataService(repo, nil, nil)

	_, err := svc.Upsert(context.Background(), &MetadataOverride{
		ModelName:        "  custom-model  ",
		ModelType:        " Chat ",
		Category:         " GPT ",
		Capabilities:     []string{"Reasoning", "reasoning", " function_calling "},
		InputModalities:  []string{"Text", " image ", "text"},
		OutputModalities: []string{" Text "},
		SupportFlags:     []string{"Vision", "vision", " web_search "},
	})
	require.NoError(t, err)
	got := repo.items["custom-model"]
	require.NotNil(t, got)
	require.Equal(t, "custom-model", got.ModelName)
	require.Equal(t, "chat", got.ModelType)
	require.Equal(t, "GPT", got.Category)
	require.Equal(t, []string{"reasoning", "function_calling"}, got.Capabilities)
	require.Equal(t, []string{"text", "image"}, got.InputModalities)
	require.Equal(t, []string{"text"}, got.OutputModalities)
	require.Equal(t, []string{"vision", "web_search"}, got.SupportFlags)

	_, err = svc.Upsert(context.Background(), &MetadataOverride{ModelName: "x", Category: "bad value"})
	require.ErrorIs(t, err, ErrMetadataInvalidField)
}
