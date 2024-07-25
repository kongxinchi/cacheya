package goredis

import (
	"context"
	"time"

	"github.com/go-redis/redis"
)

type RedisCache struct {
	client redis.UniversalClient
}

func NewRedisCache(client redis.UniversalClient) *RedisCache {
	return &RedisCache{client: client}
}

func (r *RedisCache) Name() string {
	return "RedisCache.v6"
}

func (r *RedisCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	values, err := r.client.MGet(keys...).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte)
	for i, value := range values {
		if value == nil {
			continue
		}
		result[keys[i]] = []byte(value.(string))
	}
	return result, nil
}

func (r *RedisCache) MSet(ctx context.Context, kvMap map[string][]byte, ttl time.Duration) error {
	if len(kvMap) == 0 {
		return nil
	}

	if ttl == 0 {
		flatValues := make([]interface{}, 0, 2*len(kvMap))
		for k, v := range kvMap {
			flatValues = append(flatValues, k)
			flatValues = append(flatValues, v)
		}
		if err := r.client.MSet(flatValues...).Err(); err != nil {
			return err
		}
		return nil
	}

	// 仅有一条数据时，不用Pipeline
	if len(kvMap) <= 1 {
		for k, v := range kvMap {
			if err := r.client.Set(k, v, ttl).Err(); err != nil {
				return err
			}
		}
		return nil
	}

	_, err := r.client.Pipelined(
		func(pipe redis.Pipeliner) error {
			for k, v := range kvMap {
				if err := pipe.Set(k, v, ttl).Err(); err != nil {
					return err
				}
			}
			return nil
		},
	)
	return err
}

func (r *RedisCache) MDel(ctx context.Context, keys []string) error {
	return r.client.Del(keys...).Err()
}
