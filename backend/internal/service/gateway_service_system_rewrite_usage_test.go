// sudoapi: Deduct proxy-injected Claude Code system prompt usage.

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type systemRewriteTokenSettingRepoStub struct {
	values        map[string]string
	getValueCalls int
}

func (s *systemRewriteTokenSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *systemRewriteTokenSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	s.getValueCalls++
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *systemRewriteTokenSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *systemRewriteTokenSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *systemRewriteTokenSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *systemRewriteTokenSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *systemRewriteTokenSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func TestGatewayService_SystemRewriteInputTokens_Defaults(t *testing.T) {
	systemRewriteTokenConfigCache.Store((*systemRewriteTokenConfig)(nil))
	svc := &GatewayService{}

	require.Equal(t, 500, svc.systemRewriteInputTokens(context.Background(), "claude-opus-4-7"))
	require.Equal(t, 360, svc.systemRewriteInputTokens(context.Background(), "claude-sonnet-4-5"))
}

func TestSettingService_GetSystemRewriteInputTokens_UsesDatabaseConfig(t *testing.T) {
	systemRewriteTokenConfigCache.Store((*systemRewriteTokenConfig)(nil))
	repo := &systemRewriteTokenSettingRepoStub{values: map[string]string{
		systemRewriteTokensUsage: `{"default":333,"claude-opus-4-7":777,"claude-fable-5":555}`,
	}}
	svc := &GatewayService{settingService: NewSettingService(repo, &config.Config{})}

	require.Equal(t, 777, svc.systemRewriteInputTokens(context.Background(), "claude-opus-4-7"))
	require.Equal(t, 555, svc.systemRewriteInputTokens(context.Background(), "claude-fable-5"))
	require.Equal(t, 333, svc.systemRewriteInputTokens(context.Background(), "claude-sonnet-4-5"))
	require.Equal(t, 1, repo.getValueCalls)
}

func TestSettingService_GetSystemRewriteInputTokens_MergesDatabaseConfigWithDefaults(t *testing.T) {
	systemRewriteTokenConfigCache.Store((*systemRewriteTokenConfig)(nil))
	repo := &systemRewriteTokenSettingRepoStub{values: map[string]string{
		systemRewriteTokensUsage: `{"claude-sonnet-4-5":444}`,
	}}
	svc := &GatewayService{settingService: NewSettingService(repo, &config.Config{})}

	require.Equal(t, 444, svc.systemRewriteInputTokens(context.Background(), "claude-sonnet-4-5"))
	require.Equal(t, 500, svc.systemRewriteInputTokens(context.Background(), "claude-opus-4-7"))
	require.Equal(t, 360, svc.systemRewriteInputTokens(context.Background(), "unknown-model"))
	require.Equal(t, 1, repo.getValueCalls)
}

func TestGatewayService_DeductSystemRewriteUsage_SkipsWhenCacheReadHit(t *testing.T) {
	usage := &ClaudeUsage{InputTokens: 150, CacheReadInputTokens: 10}

	require.False(t, applySystemRewriteUsage(usage, 20))

	require.Equal(t, 150, usage.InputTokens)
}

func TestGatewayService_DeductSystemRewriteUsage_AppliesTokens(t *testing.T) {
	usage := &ClaudeUsage{InputTokens: 500}

	require.True(t, applySystemRewriteUsage(usage, 360))

	require.Equal(t, 140, usage.InputTokens)
}
