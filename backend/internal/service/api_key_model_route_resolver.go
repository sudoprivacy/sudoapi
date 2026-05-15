package service

import (
	"context"
	"strings"
)

type APIKeyModelRouteResolver struct {
	channelService *ChannelService
}

func NewAPIKeyModelRouteResolver(channelService *ChannelService) *APIKeyModelRouteResolver {
	return &APIKeyModelRouteResolver{
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

	if group, resolved := r.resolveByChannelModelSignal(ctx, groups, model); resolved {
		return group
	}
	if group := resolveByModelFamily(groups, model); group != nil {
		return group
	}
	return firstGroupByPlatform(groups, PlatformOpenAI)
}

func (r *APIKeyModelRouteResolver) resolveByChannelModelSignal(ctx context.Context, groups []*Group, model string) (*Group, bool) {
	var matched *Group
	for _, group := range groups {
		if group == nil || group.ID <= 0 || strings.TrimSpace(group.Platform) == "" {
			continue
		}
		if !r.channelHasModelSignal(ctx, group.ID, model) {
			continue
		}
		if matched != nil && matched.Platform != group.Platform {
			// 跨平台显式配置同名模型/别名时无法安全判断真实目标协议，交回调用方走原有默认值。
			return nil, true
		}
		matched = group
	}
	return matched, matched != nil
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

func resolveByModelFamily(groups []*Group, model string) *Group {
	for _, group := range groups {
		if group == nil {
			continue
		}
		if routeResolverModelMatchesPlatform(model, group.Platform) {
			return group
		}
	}
	return nil
}

func firstGroupByPlatform(groups []*Group, platform string) *Group {
	for _, group := range groups {
		if group != nil && group.Platform == platform {
			return group
		}
	}
	return nil
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
