package ctxcache

import (
	"context"
	"time"
)

type cacheValue struct {
	data   []byte
	expire time.Time
}

type Config struct {
	Silent bool // 安静模式：当没有调用WithCtxCache初始化就访问缓存时，在安静模式下不会报错
}

type ContextCache struct {
	silent bool
}

func NewContextCache(configs ...*Config) *ContextCache {
	config := &Config{Silent: true}
	if len(configs) == 1 {
		config = configs[0]
	}
	return &ContextCache{silent: config.Silent}
}

func (c *ContextCache) Name() string {
	return "ContextCache"
}

func (c *ContextCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte, 0)

	cacheStore, err := GetCacheStore(ctx)
	if err != nil {
		if err == CacheStoreNotFound && c.silent {
			return result, nil
		}
		return nil, err
	}

	for _, k := range keys {
		r, ok := cacheStore.Load(k)
		if ok {
			cv := r.(*cacheValue)
			if cv.expire.IsZero() || cv.expire.After(time.Now()) {
				result[k] = cv.data
			}
		}
	}
	return result, nil
}

func (c *ContextCache) MSet(ctx context.Context, kvMap map[string][]byte, ttl time.Duration) error {
	cacheStore, err := GetCacheStore(ctx)
	if err != nil {
		if err == CacheStoreNotFound && c.silent {
			return nil
		}
		return err
	}

	for k, v := range kvMap {
		expire := time.Time{}
		if ttl > 0 {
			expire = time.Now().Add(ttl)
		}
		cv := &cacheValue{data: v, expire: expire}
		cacheStore.Store(k, cv)
	}
	return nil
}

func (c *ContextCache) MDel(ctx context.Context, keys []string) error {
	cacheStore, err := GetCacheStore(ctx)
	if err != nil {
		if err == CacheStoreNotFound && c.silent {
			return nil
		}
		return err
	}

	for _, k := range keys {
		cacheStore.Delete(k)
	}
	return nil
}
