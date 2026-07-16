// sudoapi: Model catalog.

//go:build unit

package service_model_catalog

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type stubPricingReader map[string]*service.LiteLLMModelPricing

func (s stubPricingReader) GetModelPricing(modelName string) *service.LiteLLMModelPricing {
	return s[normalizeMetadataModelKey(modelName)]
}

type mockChannelRepository struct {
	listAllFn func(ctx context.Context) ([]service.Channel, error)
}

func (m *mockChannelRepository) Create(context.Context, *service.Channel) error { return nil }
func (m *mockChannelRepository) GetByID(context.Context, int64) (*service.Channel, error) {
	return nil, service.ErrChannelNotFound
}
func (m *mockChannelRepository) Update(context.Context, *service.Channel) error { return nil }
func (m *mockChannelRepository) Delete(context.Context, int64) error            { return nil }
func (m *mockChannelRepository) List(context.Context, pagination.PaginationParams, string, string) ([]service.Channel, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (m *mockChannelRepository) ListAll(ctx context.Context) ([]service.Channel, error) {
	if m.listAllFn != nil {
		return m.listAllFn(ctx)
	}
	return nil, nil
}
func (m *mockChannelRepository) ExistsByName(context.Context, string) (bool, error) {
	return false, nil
}
func (m *mockChannelRepository) ExistsByNameExcluding(context.Context, string, int64) (bool, error) {
	return false, nil
}
func (m *mockChannelRepository) GetGroupIDs(context.Context, int64) ([]int64, error) {
	return nil, nil
}
func (m *mockChannelRepository) SetGroupIDs(context.Context, int64, []int64) error {
	return nil
}
func (m *mockChannelRepository) GetChannelIDByGroupID(context.Context, int64) (int64, error) {
	return 0, nil
}
func (m *mockChannelRepository) GetGroupsInOtherChannels(context.Context, int64, []int64) ([]int64, error) {
	return nil, nil
}
func (m *mockChannelRepository) GetGroupPlatforms(context.Context, []int64) (map[int64]string, error) {
	return nil, nil
}
func (m *mockChannelRepository) ListModelPricing(context.Context, int64) ([]service.ChannelModelPricing, error) {
	return nil, nil
}
func (m *mockChannelRepository) CreateModelPricing(context.Context, *service.ChannelModelPricing) error {
	return nil
}
func (m *mockChannelRepository) UpdateModelPricing(context.Context, *service.ChannelModelPricing) error {
	return nil
}
func (m *mockChannelRepository) DeleteModelPricing(context.Context, int64) error { return nil }
func (m *mockChannelRepository) ReplaceModelPricing(context.Context, int64, []service.ChannelModelPricing) error {
	return nil
}

type stubGroupRepoForAvailable struct {
	activeGroups []service.Group
}

func (s *stubGroupRepoForAvailable) Create(context.Context, *service.Group) error { return nil }
func (s *stubGroupRepoForAvailable) GetByID(context.Context, int64) (*service.Group, error) {
	return nil, nil
}
func (s *stubGroupRepoForAvailable) GetByIDLite(context.Context, int64) (*service.Group, error) {
	return nil, nil
}
func (s *stubGroupRepoForAvailable) Update(context.Context, *service.Group) error { return nil }
func (s *stubGroupRepoForAvailable) Delete(context.Context, int64) error          { return nil }
func (s *stubGroupRepoForAvailable) DeleteCascade(context.Context, int64) ([]int64, error) {
	return nil, nil
}
func (s *stubGroupRepoForAvailable) List(context.Context, pagination.PaginationParams) ([]service.Group, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *stubGroupRepoForAvailable) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, *bool) ([]service.Group, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *stubGroupRepoForAvailable) ListActive(context.Context) ([]service.Group, error) {
	return s.activeGroups, nil
}
func (s *stubGroupRepoForAvailable) ListActiveByPlatform(context.Context, string) ([]service.Group, error) {
	return nil, nil
}
func (s *stubGroupRepoForAvailable) ExistsByName(context.Context, string) (bool, error) {
	return false, nil
}
func (s *stubGroupRepoForAvailable) GetAccountCount(context.Context, int64) (int64, int64, error) {
	return 0, 0, nil
}
func (s *stubGroupRepoForAvailable) DeleteAccountGroupsByGroupID(context.Context, int64) (int64, error) {
	return 0, nil
}
func (s *stubGroupRepoForAvailable) GetAccountIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	return nil, nil
}
func (s *stubGroupRepoForAvailable) BindAccountsToGroup(context.Context, int64, []int64) error {
	return nil
}
func (s *stubGroupRepoForAvailable) UpdateSortOrders(context.Context, []service.GroupSortOrderUpdate) error {
	return nil
}
