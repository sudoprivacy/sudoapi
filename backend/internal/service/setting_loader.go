// sudoapi: Deduct proxy-injected system prompt usage.

package service

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

func NewSettingLoader[T any](key string, parser func(string) (T, error)) *SettingLoader[T] {
	return &SettingLoader[T]{key: key, parser: parser}
}

type SettingLoader[T any] struct {
	key   string
	cache atomic.Pointer[valueSnapshot[T]]
	sf    singleflight.Group

	parser func(string) (T, error)
}

type valueSnapshot[T any] struct {
	value     T
	expiresAt time.Time
}

func (loader *SettingLoader[T]) Get(repo SettingRepository, fallback T) T {
	if loader == nil || repo == nil || loader.parser == nil {
		return fallback
	}
	if cached := loader.cache.Load(); cached != nil && time.Now().Before(cached.expiresAt) {
		return cached.value
	}
	loaded, _, _ := loader.sf.Do(loader.key, func() (any, error) {
		if cached := loader.cache.Load(); cached != nil && time.Now().Before(cached.expiresAt) {
			return cached, nil
		}
		snapshot := loader.load(repo, fallback)
		loader.cache.Store(snapshot)
		return snapshot, nil
	})
	cached, ok := loaded.(*valueSnapshot[T])
	if !ok || cached == nil {
		return fallback
	}
	return cached.value
}

func (loader *SettingLoader[T]) load(repo SettingRepository, fallback T) *valueSnapshot[T] {
	const timeout = time.Second * 5
	const ttl = time.Second * 60
	const ttlErr = time.Second * 5

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	raw, err := repo.GetValue(ctx, loader.key)
	if err != nil {
		slog.Warn("load settings failed", "key", loader.key, "error", err)
		return &valueSnapshot[T]{value: fallback, expiresAt: time.Now().Add(ttlErr)}
	}

	value, err := loader.parser(raw)
	if err != nil {
		slog.Warn("parse settings failed", "key", loader.key, "error", err)
		return &valueSnapshot[T]{value: fallback, expiresAt: time.Now().Add(ttlErr)}
	}

	return &valueSnapshot[T]{value: value, expiresAt: time.Now().Add(ttl)}
}
