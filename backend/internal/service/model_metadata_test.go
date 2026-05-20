// sudoapi: Model market.

//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubModelMetadataRepo struct {
	items map[string]*ModelMetadataOverride
}

func (r *stubModelMetadataRepo) List(context.Context) ([]*ModelMetadataOverride, error) {
	out := make([]*ModelMetadataOverride, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, cloneModelMetadataOverride(item))
	}
	return out, nil
}

func (r *stubModelMetadataRepo) GetByModelNames(_ context.Context, modelNames []string) (map[string]*ModelMetadataOverride, error) {
	out := make(map[string]*ModelMetadataOverride, len(modelNames))
	for _, name := range modelNames {
		key := normalizeMetadataModelKey(name)
		if item, ok := r.items[key]; ok {
			out[key] = cloneModelMetadataOverride(item)
		}
	}
	return out, nil
}

func (r *stubModelMetadataRepo) Upsert(_ context.Context, override *ModelMetadataOverride) error {
	if r.items == nil {
		r.items = make(map[string]*ModelMetadataOverride)
	}
	r.items[normalizeMetadataModelKey(override.ModelName)] = cloneModelMetadataOverride(override)
	return nil
}

func (r *stubModelMetadataRepo) DeleteByModelName(_ context.Context, modelName string) error {
	delete(r.items, normalizeMetadataModelKey(modelName))
	return nil
}

func TestModelMetadataService_ListForAdminMergesCurrentModelsAndOverrides(t *testing.T) {
	g := Group{ID: 1, Name: "auto", Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, RateMultiplier: 1.0, Status: StatusActive}
	channels := []Channel{{
		ID: 1, Name: "ch", Status: StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []ChannelModelPricing{{
			Platform: PlatformOpenAI, Models: []string{"custom-model"},
			BillingMode: BillingModeToken, InputPrice: msPtrFloat(1e-6),
		}},
	}}
	repo := &stubModelMetadataRepo{items: map[string]*ModelMetadataOverride{
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
	svc := NewModelMetadataService(repo, stubChannelServiceProvider(t, channels, []Group{g}), nil)

	items, err := svc.ListForAdmin(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "custom-model", items[0].ModelName)
	require.Equal(t, []string{PlatformOpenAI}, items[0].Platforms)
	require.Equal(t, "Custom Model", items[0].Metadata.DisplayName)
	require.Equal(t, "chat", items[0].Metadata.ModelType)
	require.Equal(t, []string{"text", "image"}, items[0].Metadata.InputModalities)
	require.Equal(t, []string{"text"}, items[0].Metadata.OutputModalities)
	require.Equal(t, []string{"reasoning", "vision"}, items[0].Metadata.SupportFlags)
	require.Empty(t, items[0].MissingFields)
	require.NotNil(t, items[0].Override)
}

func TestModelMetadataService_UpsertValidatesAndNormalizes(t *testing.T) {
	repo := &stubModelMetadataRepo{}
	svc := NewModelMetadataService(repo, nil, nil)

	_, err := svc.Upsert(context.Background(), &ModelMetadataOverride{
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
	require.Equal(t, "gpt", got.Category)
	require.Equal(t, []string{"reasoning", "function_calling"}, got.Capabilities)
	require.Equal(t, []string{"text", "image"}, got.InputModalities)
	require.Equal(t, []string{"text"}, got.OutputModalities)
	require.Equal(t, []string{"vision", "web_search"}, got.SupportFlags)

	_, err = svc.Upsert(context.Background(), &ModelMetadataOverride{ModelName: "x", Category: "bad value"})
	require.ErrorIs(t, err, ErrModelMetadataInvalidField)
}
