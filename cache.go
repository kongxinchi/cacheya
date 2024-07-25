package cacheya

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/kongxinchi/cacheya/compressor"
	"github.com/kongxinchi/cacheya/marshaler"
	"golang.org/x/sync/singleflight"
)

type CacheAdaptor interface {
	Name() string

	MGet(ctx context.Context, keys []string) (map[string][]byte, error)
	MSet(ctx context.Context, kvMap map[string][]byte, ttl time.Duration) error
	MDel(ctx context.Context, keys []string) error
}

type CacheValue[T any] struct {
	Object  *T     `json:"o"`
	Version string `json:"v"`
}

func (c *CacheValue[T]) isValid(validVersion string) bool {
	return c != nil && c.Version == validVersion
}

type MultiDataLoader[T any, O comparable] func(keys []O) (map[O]*T, error)
type MultiDataValLoader[T any, O comparable] func(keys []O) (map[O]T, error)

type SingleDataLoader[T any, O comparable] func(k O) (*T, error)
type SingleDataValLoader[T any, O comparable] func(k O) (T, error)

type CacheManager[T any, O comparable] struct {
	name    string // 缓存名称，不同的实例请使用不同的名称，否则可能会串
	version string // 缓存版本，调整缓存内容时需要升级版本，避免存量缓存的影响

	adaptor    CacheAdaptor // 缓存实现
	marshaler  Marshaler    // 序列化实现
	compressor Compressor   // 压缩实现
	keyBuilder Builder      // 缓存key生成函数，默认拼接规则为：{name}:{account}:{key}

	ttl       time.Duration // 缓存超时时间，默认1h
	ttlJitter time.Duration // 缓存超时时间随机间隔上限，默认10s
	cacheNil  bool          // 是否缓存nil

	logger  Logger            // 日志
	metrics Metrics           // 监控埋点
	mLabels map[string]string // 监控埋点用的固定labels

	sg singleflight.Group // Get时控制并发
}

func NewCacheManager[T any, O comparable](name string, adaptor CacheAdaptor, opts ...Option) *CacheManager[T, O] {
	c := &Config{
		version: "1",

		keyBuilder: DefaultKeyBuilder,
		marshaler:  marshaler.NewJsonMarshaler(),
		compressor: compressor.NewNoopCompressor(),
		logger:     DefaultNopLogger(),
		metrics:    DefaultNopMetrics(),

		ttl:       1 * time.Hour,
		ttlJitter: 10 * time.Second,
		cacheNil:  true,
	}
	for _, opt := range opts {
		opt(c)
	}

	return &CacheManager[T, O]{
		name:    name,
		version: c.version,

		adaptor:    adaptor,
		keyBuilder: c.keyBuilder,
		marshaler:  c.marshaler,
		compressor: c.compressor,

		logger:  c.logger,
		metrics: c.metrics,

		ttl:       c.ttl,
		ttlJitter: c.ttlJitter,
		cacheNil:  c.cacheNil,

		mLabels: map[string]string{
			"name":    name,
			"adaptor": adaptor.Name(),
		},
	}
}

func (l *CacheManager[T, O]) buildCacheKey(ctx context.Context, key O) string {
	return l.keyBuilder(ctx, l.name, key)
}

func (l *CacheManager[T, O]) Get(ctx context.Context, key O) (result *T, isCached bool, err error) {
	kv, err := l.MGet(ctx, []O{key})
	if err != nil {
		return nil, false, err
	}

	result, isCached = kv[key] // isCached = true，意味着有缓存（即使 result = Nil）
	return
}

func (l *CacheManager[T, O]) MGet(ctx context.Context, keys []O) (result map[O]*T, err error) {
	defer l.deferMetrics(ctx, "cache_get")(&err)

	if len(keys) == 0 {
		return
	}

	// 拼接缓存 Key 前缀
	cacheKeys := make([]string, 0)

	// 维护 Key -> CacheKey 的关系，用于后续组装结果
	keyIndex := make(map[O]string, 0)
	for _, k := range keys {
		ck := l.buildCacheKey(ctx, k)
		cacheKeys = append(cacheKeys, ck)
		keyIndex[k] = ck
	}

	// 先从缓存查询
	cached, err := l.adaptor.MGet(ctx, cacheKeys)
	if err != nil {
		l.logger.Error(ctx, "cache_get_error", map[string]any{"error": err, "keys": keys})
		return nil, err
	}

	// 解压缩 & 反序列化
	result = make(map[O]*T, 0)
	mismatch := 0
	hit := 0
	miss := 0

	for _, k := range keys {
		c, ok := cached[keyIndex[k]]
		if ok {
			var unmarshalled CacheValue[T]

			// 解压缩
			decompressed, err := l.compressor.Decode(c)
			if err != nil {
				// 解压缩失败视为未命中，仅埋点不报错
				mismatch++
				continue
			}

			// 反序列化
			err = l.marshaler.Unmarshal(decompressed, &unmarshalled)
			if err != nil || !unmarshalled.isValid(l.version) {
				// 反序列化失败或版本不符，视为未命中，仅埋点不报错
				mismatch++
				continue
			}

			result[k] = unmarshalled.Object
			hit++

		} else {
			miss++
		}
	}

	if mismatch > 0 {
		l.metrics.Counter(ctx, "cache_mismatch_total", float64(mismatch), l.mLabels)
	}
	if hit > 0 {
		l.metrics.Counter(ctx, "cache_hit_total", float64(hit), l.mLabels)
	}
	if miss > 0 {
		l.metrics.Counter(ctx, "cache_miss_total", float64(miss), l.mLabels)
	}

	return result, nil
}

func (l *CacheManager[T, O]) Set(ctx context.Context, k O, v *T) error {
	kv := map[O]*T{k: v}
	return l.MSet(ctx, kv)
}

func (l *CacheManager[T, O]) MSet(ctx context.Context, kv map[O]*T) (err error) {
	defer l.deferMetrics(ctx, "cache_set")(&err)

	if kv == nil || len(kv) == 0 {
		return
	}

	ckv := make(map[string][]byte, len(kv))
	for k, v := range kv {
		// 序列化
		marshaled, e := l.marshaler.Marshal(CacheValue[T]{Object: v, Version: l.version})
		if e != nil {
			l.logger.Error(ctx, "cache_set_error", map[string]any{"error": e, "kv": kv})
			err = e
			return
		}

		// 压缩
		compressed, e := l.compressor.Encode(marshaled)
		if e != nil {
			l.logger.Error(ctx, "cache_set_error", map[string]any{"error": e, "kv": kv})
			err = e
			return
		}

		ckv[l.buildCacheKey(ctx, k)] = compressed
	}

	ttl := l.ttl
	if l.ttlJitter > 0 {
		ttl += time.Duration(rand.Int63n(int64(l.ttlJitter)))
	}
	return l.adaptor.MSet(ctx, ckv, ttl)
}

func (l *CacheManager[T, O]) Del(ctx context.Context, key O) error {
	return l.MDel(ctx, []O{key})
}

func (l *CacheManager[T, O]) MDel(ctx context.Context, keys []O) (err error) {
	defer l.deferMetrics(ctx, "cache_del")(&err)

	if len(keys) == 0 {
		return
	}

	cKeys := make([]string, 0)
	for _, k := range keys {
		cKeys = append(cKeys, l.buildCacheKey(ctx, k))
	}
	err = l.adaptor.MDel(ctx, cKeys)
	if err != nil {
		l.logger.Error(ctx, "cache_del_error", map[string]any{"error": err, "keys": keys})
	}
	return
}

// Load Cache-Aside 实现，单个查询，返回引用
func (l *CacheManager[T, O]) Load(ctx context.Context, key O, loader SingleDataLoader[T, O]) (*T, error) {
	r, err, _ := l.sg.Do(
		l.buildCacheKey(ctx, key), func() (interface{}, error) {
			return l.load(ctx, key, loader)
		},
	)
	if err != nil {
		return nil, err
	}
	return r.(*T), nil
}

// Loadv Cache-Aside 实现，单个查询，返回值
func (l *CacheManager[T, O]) Loadv(ctx context.Context, key O, loader SingleDataValLoader[T, O]) (T, error) {
	pLoader := func(k O) (*T, error) {
		vr, e := loader(k)
		return &vr, e
	}
	var defaultVr T
	pr, err := l.Load(ctx, key, pLoader)
	if pr == nil {
		return defaultVr, err
	}
	return *pr, err
}

func (l *CacheManager[T, O]) load(ctx context.Context, key O, loader SingleDataLoader[T, O]) (*T, error) {
	mLoader := func(keys []O) (map[O]*T, error) {
		v, err := loader(keys[0])
		if err != nil {
			return nil, err
		}
		return map[O]*T{keys[0]: v}, nil
	}

	kv, err := l.MLoad(ctx, []O{key}, mLoader)
	if err != nil {
		return nil, err
	}
	return kv[key], nil
}

// MLoad Cache-Aside 实现，批量查询，返回引用（返回的map中不会有为nil的value）
func (l *CacheManager[T, O]) MLoad(ctx context.Context, keys []O, loader MultiDataLoader[T, O]) (map[O]*T, error) {
	result := make(map[O]*T, len(keys))

	// 先从缓存查询
	cached, err := l.MGet(ctx, keys)
	if err != nil {
		return nil, err
	}
	// 把从缓存中能取到的放到结果里（排除掉nil）
	for ck, cv := range cached {
		if cv == nil {
			continue
		}
		result[ck] = cv
	}

	// 过滤出未查询到的 Key
	missedKeys := make([]O, 0)
	for _, k := range keys {
		if _, ok := cached[k]; !ok {
			missedKeys = append(missedKeys, k)
		}
	}

	// 回源从数据源查询
	supply := make(map[O]*T, 0)
	if len(missedKeys) > 0 {
		values, err := loader(missedKeys)
		if err != nil {
			return nil, err
		}

		for _, mk := range missedKeys {
			v, ok := values[mk]
			if ok && v != nil { // loader返回数据中：k not in values && values[k] == nil，均视为数据不存在
				supply[mk] = v
				result[mk] = v
			} else if l.cacheNil {
				supply[mk] = nil
			}
		}
		err = l.MSet(ctx, supply)
		if err != nil {
			return nil, err
		}
	}

	l.logger.Debug(ctx, "cache_load", map[string]any{"cached": cached, "supply": supply})

	return result, nil
}

// MLoadv Cache-Aside 实现，批量查询，返回值
func (l *CacheManager[T, O]) MLoadv(ctx context.Context, keys []O, loader MultiDataValLoader[T, O]) (map[O]T, error) {
	pLoader := func(keys []O) (map[O]*T, error) {
		vrs, e := loader(keys)
		prs := make(map[O]*T, 0)
		for k, vr := range vrs {
			tmp := vr
			prs[k] = &tmp
		}
		return prs, e
	}
	result := make(map[O]T, 0)
	prs, err := l.MLoad(ctx, keys, pLoader)
	if err != nil {
		return nil, err
	}

	for k, pr := range prs {
		if pr == nil {
			continue
		}
		result[k] = *pr
	}
	return result, nil
}

func (l *CacheManager[T, O]) deferMetrics(ctx context.Context, name string) func(*error) {
	begin := time.Now()
	return func(err *error) {
		elapsed := float64(time.Since(begin)) / float64(time.Second)
		if err != nil {
			l.metrics.Counter(ctx, fmt.Sprintf("%s_errors_total", name), 1, l.mLabels)
		}
		l.metrics.Counter(ctx, fmt.Sprintf("%s_total", name), 1, l.mLabels)
		l.metrics.Timer(ctx, fmt.Sprintf("%s_seconds", name), elapsed, l.mLabels)
	}
}
