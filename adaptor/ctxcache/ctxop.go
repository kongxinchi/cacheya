package ctxcache

import (
	"context"
	"errors"
	"sync"
)

var (
	CacheStoreNotFound = errors.New("context cache store not found")
	_ctxCacheKey       = ctxCacheKey{}
)

type ctxCacheKey struct{}

func WithCtxCache(ctx context.Context) context.Context {
	m := ctx.Value(_ctxCacheKey)
	if m != nil {
		return ctx
	}
	out := new(sync.Map)
	ctx = context.WithValue(ctx, _ctxCacheKey, out)
	return ctx
}

func GetCacheStore(ctx context.Context) (*sync.Map, error) {
	m := ctx.Value(_ctxCacheKey)
	out, ok := m.(*sync.Map)
	if ok {
		return out, nil
	}
	return nil, CacheStoreNotFound
}
