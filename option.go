package cacheya

import (
	"context"
	"fmt"
	"time"
)

type Config struct {
	version    string        // 缓存版本，调整缓存内容时需要升级版本，避免存量缓存的影响
	ttl        time.Duration // 缓存超时时间，默认1h
	ttlJitter  time.Duration // 缓存超时时间随机间隔上限，默认10s
	cacheNil   bool          // 是否缓存nil
	keyBuilder Builder       // 缓存key生成函数，默认拼接规则为：{prefix}:{key}
	marshaller Marshaller    // 序列化实现
	compressor Compressor    // 压缩实现
	metrics    Metrics       // 监控埋点
	logger     Logger        // 日志
}

type Builder func(ctx context.Context, prefix string, k any) string

type Option func(*Config)

func KeyBuilder(f Builder) Option {
	return func(c *Config) {
		c.keyBuilder = f
	}
}

func Version(version string) Option {
	return func(c *Config) {
		c.version = version
	}
}

func TTL(ttl time.Duration, jitter time.Duration) Option {
	return func(c *Config) {
		c.ttl = ttl
		c.ttlJitter = jitter
	}
}

func CacheNil(cacheNil bool) Option {
	return func(c *Config) {
		c.cacheNil = cacheNil
	}
}

func SetMarshaller(marshaller Marshaller) Option {
	return func(c *Config) {
		c.marshaller = marshaller
	}
}

func SetCompressor(compressor Compressor) Option {
	return func(c *Config) {
		c.compressor = compressor
	}
}

func SetMetrics(metrics Metrics) Option {
	return func(c *Config) {
		c.metrics = metrics
	}
}

func SetLogger(logger Logger) Option {
	return func(c *Config) {
		c.logger = logger
	}
}

func DefaultKeyBuilder(ctx context.Context, name string, key any) string {
	return fmt.Sprintf("%s:%v", name, key)
}
