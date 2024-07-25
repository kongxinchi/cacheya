package memory

import (
	"context"
	"time"

	"github.com/dgraph-io/ristretto"
)

type RistrettoCache struct {
	client *ristretto.Cache
}

func NewRistrettoCache(client *ristretto.Cache) *RistrettoCache {
	return &RistrettoCache{client: client}
}

func (m *RistrettoCache) Name() string {
	return "RistrettoCache"
}

func (m *RistrettoCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte, 0)
	for _, k := range keys {
		r, isCached := m.client.Get(k)
		if isCached {
			result[k] = r.([]byte)
		}
	}
	return result, nil
}

func (m *RistrettoCache) MSet(ctx context.Context, kvMap map[string][]byte, ttl time.Duration) error {
	for k, v := range kvMap {
		if ttl == 0 {
			m.client.Set(k, v, 1)
		} else {
			m.client.SetWithTTL(k, v, 1, ttl)
		}
	}
	m.client.Wait()
	return nil
}

func (m *RistrettoCache) MDel(ctx context.Context, keys []string) error {
	for _, k := range keys {
		m.client.Del(k)
	}
	m.client.Wait()
	return nil
}
