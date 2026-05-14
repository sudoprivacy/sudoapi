package service

import (
	"context"
	"strings"
)

type APIKeyModelRouteResolver struct {
	accountRepo    AccountRepository
	channelService *ChannelService
}

func NewAPIKeyModelRouteResolver(accountRepo AccountRepository, channelService *ChannelService) *APIKeyModelRouteResolver {
	return &APIKeyModelRouteResolver{
		accountRepo:    accountRepo,
		channelService: channelService,
	}
}

func (r *APIKeyModelRouteResolver) ResolveAPIKeyGroupForModel(ctx context.Context, apiKey *APIKey, model string) *Group {
	if r == nil || apiKey == nil || strings.TrimSpace(model) == "" {
		return nil
	}
	groups := apiKeyRouteGroups(apiKey)
	if len(groups) == 0 {
		return nil
	}

	var best *Group
	bestScore := 0
	tied := false
	for _, group := range groups {
		score := r.scoreGroupForModel(ctx, group, model)
		if score == 0 {
			continue
		}
		if score > bestScore {
			best = group
			bestScore = score
			tied = false
			continue
		}
		if score == bestScore && best != nil && group.Platform != best.Platform {
			// 跨平台同分说明仅凭模型无法确定真实目标协议，交回调用方走原有路由默认值。
			tied = true
		}
	}
	if tied {
		return nil
	}
	return best
}

func apiKeyRouteGroups(apiKey *APIKey) []*Group {
	if apiKey == nil {
		return nil
	}
	seen := make(map[int64]struct{})
	groups := make([]*Group, 0, len(apiKey.Groups)+1)
	add := func(group *Group) {
		if group == nil || group.ID <= 0 || strings.TrimSpace(group.Platform) == "" {
			return
		}
		if _, ok := seen[group.ID]; ok {
			return
		}
		seen[group.ID] = struct{}{}
		groups = append(groups, group)
	}
	for _, group := range apiKey.Groups {
		add(group)
	}
	add(apiKey.Group)
	return groups
}

func (r *APIKeyModelRouteResolver) scoreGroupForModel(ctx context.Context, group *Group, model string) int {
	if group == nil || group.ID <= 0 || strings.TrimSpace(group.Platform) == "" {
		return 0
	}
	if r.channelHasModelSignal(ctx, group.ID, model) {
		// channel pricing/mapping 是最明确的分组信号，但仍必须有可调度账号承接。
		if r.groupHasSchedulableAccountModelSupport(ctx, group, model) {
			return 100
		}
		return 0
	}
	if r.channelRestrictsModel(ctx, group.ID, model) {
		return 0
	}
	if r.groupHasExplicitAccountModelSupport(ctx, group, model) {
		return 80
	}
	if r.groupHasDefaultAccountModelSupport(ctx, group, model) {
		// 未配置账号级 model_mapping 表示账号不限制模型；这里仅作为平台模型族 fallback。
		return 10
	}
	return 0
}

func (r *APIKeyModelRouteResolver) channelHasModelSignal(ctx context.Context, groupID int64, model string) bool {
	if r.channelService == nil {
		return false
	}
	if pricing := r.channelService.GetChannelModelPricing(ctx, groupID, model); pricing != nil {
		return true
	}
	mapping := r.channelService.ResolveChannelMapping(ctx, groupID, model)
	return mapping.Mapped
}

func (r *APIKeyModelRouteResolver) channelRestrictsModel(ctx context.Context, groupID int64, model string) bool {
	if r.channelService == nil {
		return false
	}
	return r.channelService.IsModelRestricted(ctx, groupID, model)
}

func (r *APIKeyModelRouteResolver) groupHasExplicitAccountModelSupport(ctx context.Context, group *Group, model string) bool {
	return r.anySchedulableAccountSupports(ctx, group, model, true)
}

func (r *APIKeyModelRouteResolver) groupHasDefaultAccountModelSupport(ctx context.Context, group *Group, model string) bool {
	return r.anySchedulableAccountSupports(ctx, group, model, false)
}

func (r *APIKeyModelRouteResolver) groupHasSchedulableAccountModelSupport(ctx context.Context, group *Group, model string) bool {
	// channel mapping 可能把自定义别名映射到上游模型，所以显式命中或账号不限制模型都可承接。
	return r.anySchedulableAccountSupports(ctx, group, model, true) ||
		r.groupHasUnrestrictedSchedulableAccount(ctx, group)
}

func (r *APIKeyModelRouteResolver) groupHasUnrestrictedSchedulableAccount(ctx context.Context, group *Group) bool {
	if r.accountRepo == nil || group == nil {
		return false
	}
	platforms := routeResolverPlatformsForGroup(group.Platform)
	accounts, err := r.accountRepo.ListSchedulableByGroupIDAndPlatforms(ctx, group.ID, platforms)
	if err != nil || len(accounts) == 0 {
		return false
	}
	for i := range accounts {
		account := &accounts[i]
		if routeResolverAccountAllowedForGroupPlatform(account, group.Platform) && len(account.GetModelMapping()) == 0 {
			return true
		}
	}
	return false
}

func (r *APIKeyModelRouteResolver) anySchedulableAccountSupports(ctx context.Context, group *Group, model string, explicitOnly bool) bool {
	if r.accountRepo == nil || group == nil {
		return false
	}
	platforms := routeResolverPlatformsForGroup(group.Platform)
	accounts, err := r.accountRepo.ListSchedulableByGroupIDAndPlatforms(ctx, group.ID, platforms)
	if err != nil || len(accounts) == 0 {
		return false
	}
	for i := range accounts {
		account := &accounts[i]
		if !routeResolverAccountAllowedForGroupPlatform(account, group.Platform) {
			continue
		}
		if explicitOnly {
			if len(account.GetModelMapping()) > 0 && account.IsModelSupported(model) {
				return true
			}
			continue
		}
		if len(account.GetModelMapping()) == 0 && routeResolverModelMatchesPlatform(model, group.Platform) {
			return true
		}
	}
	return false
}

func routeResolverPlatformsForGroup(platform string) []string {
	switch platform {
	case PlatformAnthropic, PlatformGemini:
		// 与调度逻辑保持一致：Anthropic/Gemini 分组可混合调度启用 mixed_scheduling 的 Antigravity 账号。
		return []string{platform, PlatformAntigravity}
	default:
		return []string{platform}
	}
}

func routeResolverAccountAllowedForGroupPlatform(account *Account, groupPlatform string) bool {
	if account == nil {
		return false
	}
	if account.Platform == groupPlatform {
		return true
	}
	return account.Platform == PlatformAntigravity &&
		(groupPlatform == PlatformAnthropic || groupPlatform == PlatformGemini) &&
		account.IsMixedSchedulingEnabled()
}

func routeResolverModelMatchesPlatform(model, platform string) bool {
	normalized := strings.ToLower(strings.TrimSpace(model))
	switch platform {
	case PlatformGemini:
		return strings.HasPrefix(normalized, "gemini-") ||
			strings.HasPrefix(normalized, "models/gemini-") ||
			strings.Contains(normalized, "/gemini-") ||
			strings.HasPrefix(normalized, "imagen-") ||
			strings.Contains(normalized, "/imagen-")
	case PlatformOpenAI:
		return strings.HasPrefix(normalized, "gpt-") ||
			strings.HasPrefix(normalized, "o1") ||
			strings.HasPrefix(normalized, "o3") ||
			strings.HasPrefix(normalized, "o4") ||
			strings.HasPrefix(normalized, "o5") ||
			strings.HasPrefix(normalized, "chatgpt-") ||
			strings.HasPrefix(normalized, "codex-") ||
			strings.HasPrefix(normalized, "text-embedding-") ||
			strings.HasPrefix(normalized, "dall-e-") ||
			strings.HasPrefix(normalized, "tts-") ||
			strings.HasPrefix(normalized, "whisper-")
	case PlatformAnthropic:
		return strings.HasPrefix(normalized, "claude-") ||
			strings.HasPrefix(normalized, "anthropic.claude-") ||
			strings.Contains(normalized, "/claude-")
	case PlatformAntigravity:
		return strings.HasPrefix(normalized, "claude-") ||
			strings.HasPrefix(normalized, "gemini-") ||
			strings.HasPrefix(normalized, "models/gemini-") ||
			strings.Contains(normalized, "/gemini-")
	default:
		return false
	}
}
