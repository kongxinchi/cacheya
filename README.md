# cacheya

## 介绍
Golang 缓存封装，支持Context、内存、Redis多种缓存适配，实现便捷的 Cache-Aside 操作

## 基本使用
```go

type TestObject struct {
    Id   string
    A    float32
    B    uint32
}


ctx := context.Background()

// 初始化：Redis缓存
opt := &redis.Options{
    Addr:     "127.0.0.1:6379",
}
client := redis.NewClient(opt)

adapter := goredis.NewRedisCache(client)
manager := NewCacheManager[string, *TestObject]("TEST_OBJECT", adapter)   // 范型中第一个指定id的类型，第二个指定缓存的对象

// 初始化：内存缓存
client, _ := ristretto.NewCache(
    &ristretto.Config{
        NumCounters: 1e7,  // number of keys to track frequency of (10M).
        MaxCost:     10e7, // maximum cost of cache (100M).
        BufferItems: 64,   // number of keys per Get buf
    },
)
adapter := memory.NewRistrettoCache(client)
manager := NewCacheManager[string, *TestObject]("TEST_OBJECT", adapter)

// 初始化：Context缓存
// 注意，使用时需要在服务内ctx生命周期开始时（比如grpc的中间件中），使用WithCtxCache对ctx开启缓存，否则无效
adapter := ctxcache.NewContextCache()
manager := NewCacheManager[string, *TestObject]("TEST_OBJECT", adapter)

// 单数据操作
r, c, err := manager.Get(ctx, "1")                 // Get
err = manager.Set(ctx, "2", &TestObject{Id: "2"})  // Set
err = manager.Del(ctx, "1")                        // Del
r, err = manager.Load(                             // Cache-Aside
    ctx, "1", func(k string) (*TestObject, error) {
        // 这里实现回源的代码
    },
)

// 批量操作
r, err := manager.MGet(ctx, []string{"1", "2"})    // MGet
err = manager.MSet(ctx, kv)                        // MSet
err = manager.MDel(ctx, []string{"1", "2"})        // MDel
r, err = manager.MLoad(                            // Multi Cache-Aside
    ctx, []string{"1", "2"}, func(ks []string) (map[string]*TestObject, error) {
        // 这里实现回源的代码
    },
)
```

## 缓存基本类型
```go
type TestStringValue string
manager := NewCacheManager[string, TestStringValue]("TEST_STRING", adapter)

r, err = manager.Load(                             // Cache-Aside
    ctx, "1", func(k string) (TestStringValue, error) {
        // 这里实现回源的代码
    },
)
r, err = manager.MLoad(                            // Multi Cache-Aside
    ctx, []string{"1", "2"}, func(ks []string) (map[string]TestStringValue, error) {
        // 这里实现回源的代码
    },
)
```

## 支持的参数
```go
type Config struct {
    version    string        // 缓存版本，调整缓存内容时需要升级版本，避免存量缓存的影响，默认：1
    ttl        time.Duration // 缓存超时时间，默认：1h
    ttlJitter  time.Duration // 缓存超时时间随机间隔上限，默认：10s
    cacheNil   bool          // 是否缓存nil，默认：true
    keyBuilder Builder       // 缓存key生成函数，默认：{prefix}:{key}
    marshaller Marshaller    // 序列化实现，默认：JSON
    compressor Compressor    // 压缩实现，默认：Noop
    metrics    Metrics       // 监控埋点，默认：Noop
    logger     Logger        // 日志，默认：Noop
}

manager := NewCacheManager[string, *TestObject]("TEST_OBJECT_PTR", adapter, TTL(CacheTTL, CacheJitter))
```