package cacheya

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/kongxinchi/cacheya/adaptor/ctxcache"
	"github.com/kongxinchi/cacheya/adaptor/goredis/v9"
	"github.com/kongxinchi/cacheya/adaptor/memory"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

var (
	CacheTTL    = 1 * time.Second
	CacheJitter = 1 * time.Millisecond
)

type TestObject struct {
	Id   string
	A    float32
	B    uint32
	Subs []*SubTestObject
}

type SubTestObject struct {
	C string
	D int
}

type TestStringValue string

func TestMemoryCache(t *testing.T) {
	ctx := context.Background()

	client, _ := ristretto.NewCache(
		&ristretto.Config{
			NumCounters: 1e7,  // number of keys to track frequency of (10M).
			MaxCost:     10e7, // maximum cost of cache (100M).
			BufferItems: 64,   // number of keys per Get buf
		},
	)
	adapter := memory.NewRistrettoCache(client)
	manager := NewCacheManager[string, *TestObject]("TEST_OBJECT_PTR", adapter, TTL(CacheTTL, CacheJitter))
	RunTestObjectPtrCase(t, ctx, manager)
	RunMultiTestObjectPtrCase(t, ctx, manager)

	manager2 := NewCacheManager[string, TestObject]("TEST_OBJECT", adapter, TTL(CacheTTL, CacheJitter))
	RunTestObjectCase(t, ctx, manager2)
	RunMultiTestObjectCase(t, ctx, manager2)

	manager3 := NewCacheManager[string, TestStringValue]("TEST_STRING", adapter, TTL(CacheTTL, CacheJitter))
	RunTestStringValueCase(t, ctx, manager3)
	RunMultiTestStringValueCase(t, ctx, manager3)
}

func TestRedisCache(t *testing.T) {
	ctx := context.Background()

	opt := &redis.Options{
		Addr: "127.0.0.1:6379",
	}
	client := redis.NewClient(opt)

	adapter := goredis.NewRedisCache(client)
	manager := NewCacheManager[string, *TestObject]("TEST_OBJECT_PTR", adapter, TTL(CacheTTL, CacheJitter))
	RunTestObjectPtrCase(t, ctx, manager)
	RunMultiTestObjectPtrCase(t, ctx, manager)

	manager2 := NewCacheManager[string, TestObject]("TEST_OBJECT", adapter, TTL(CacheTTL, CacheJitter))
	RunTestObjectCase(t, ctx, manager2)
	RunMultiTestObjectCase(t, ctx, manager2)

	manager3 := NewCacheManager[string, TestStringValue]("TEST_STRING", adapter, TTL(CacheTTL, CacheJitter))
	RunTestStringValueCase(t, ctx, manager3)
	RunMultiTestStringValueCase(t, ctx, manager3)
}

func TestContextCache(t *testing.T) {
	adapter := ctxcache.NewContextCache()
	manager := NewCacheManager[string, *TestObject]("TEST_OBJECT_PTR", adapter, TTL(CacheTTL, CacheJitter))

	ctx := context.Background()
	ctx = ctxcache.WithCtxCache(ctx) // Important

	RunTestObjectPtrCase(t, ctx, manager)
	RunMultiTestObjectPtrCase(t, ctx, manager)

	manager2 := NewCacheManager[string, TestObject]("TEST_OBJECT", adapter, TTL(CacheTTL, CacheJitter))
	RunTestObjectCase(t, ctx, manager2)
	RunMultiTestObjectCase(t, ctx, manager2)

	manager3 := NewCacheManager[string, TestStringValue]("TEST_STRING", adapter, TTL(CacheTTL, CacheJitter))
	RunTestStringValueCase(t, ctx, manager3)
	RunMultiTestStringValueCase(t, ctx, manager3)

	manager4 := NewCacheManager[string, *TestObject]("TEST_OBJECT", ctxcache.NewContextCache(&ctxcache.Config{Silent: false}))
	r, c, err := manager4.Get(context.Background(), "1")
	assert.Nil(t, r)
	assert.False(t, c)
	assert.Error(t, err, "context cache store not found")
}

func RunTestObjectPtrCase(t *testing.T, ctx context.Context, manager *CacheManager[string, *TestObject]) {
	// Prepare
	_ = manager.MDel(ctx, []string{"1", "2", "3", "4"})

	// Get
	r, c, err := manager.Get(ctx, "1")
	assert.Nil(t, r)
	assert.False(t, c)
	assert.Nil(t, err)

	// Load
	r, err = manager.Load(
		ctx, "1", func(k string) (*TestObject, error) {
			subs := make([]*SubTestObject, 0)
			subs = append(subs, &SubTestObject{C: "c1", D: 1})
			subs = append(subs, &SubTestObject{C: "c2", D: 2})
			return &TestObject{Id: k, A: 1, B: 2, Subs: subs}, nil
		},
	)
	assert.Equal(t, "1", r.Id)
	assert.Equal(t, float32(1), r.A)
	assert.Equal(t, uint32(2), r.B)
	assert.Equal(t, "c1", r.Subs[0].C)
	assert.Equal(t, 1, r.Subs[0].D)
	assert.Equal(t, "c2", r.Subs[1].C)
	assert.Equal(t, 2, r.Subs[1].D)

	// Set
	err = manager.Set(ctx, "2", &TestObject{Id: "2"})
	assert.Nil(t, err)

	// Get
	r, c, err = manager.Get(ctx, "2")
	assert.Equal(t, "2", r.Id)
	assert.True(t, c)
	assert.Nil(t, err)

	// TTL
	time.Sleep(CacheTTL + CacheJitter)
	r, c, err = manager.Get(ctx, "2")
	assert.Nil(t, r)
	assert.False(t, c)
	assert.Nil(t, err)

	// Del
	err = manager.Del(ctx, "1")
	assert.Nil(t, err)

	// Load_err
	r, err = manager.Load(
		ctx, "1", func(k string) (*TestObject, error) {
			return nil, errors.New("DataLoadFailed")
		},
	)
	assert.Nil(t, r)
	assert.Error(t, err, "DataLoadFailed")

	// Load Concurrent
	result := make([]*TestObject, 0)
	mu := sync.Mutex{}
	begin := time.Now()
	g, ctx := errgroup.WithContext(ctx)
	for i := 0; i < 100; i++ {
		g.Go(
			func() error {
				r, _ := manager.Load(
					ctx, "10", func(k string) (*TestObject, error) {
						time.Sleep(100 * time.Millisecond)
						fmt.Printf("Load Concurrent In: %d\n", i)
						return &TestObject{Id: "10"}, nil
					},
				)
				mu.Lock()
				result = append(result, r)
				mu.Unlock()
				return nil
			},
		)
	}
	_ = g.Wait()
	fmt.Printf("Load Concurrent Cost: %s\n", time.Since(begin))
	for _, r = range result {
		assert.Equal(t, "10", r.Id)
	}
}

func RunTestObjectCase(t *testing.T, ctx context.Context, manager *CacheManager[string, TestObject]) {
	// Prepare
	_ = manager.MDel(ctx, []string{"1", "2", "3", "4"})

	// Get
	r, c, err := manager.Get(ctx, "1")
	assert.Empty(t, r)
	assert.False(t, c)
	assert.Nil(t, err)

	// Load
	r, err = manager.Load(
		ctx, "1", func(k string) (TestObject, error) {
			subs := make([]*SubTestObject, 0)
			subs = append(subs, &SubTestObject{C: "c1", D: 1})
			subs = append(subs, &SubTestObject{C: "c2", D: 2})
			return TestObject{Id: k, A: 1, B: 2, Subs: subs}, nil
		},
	)
	assert.Equal(t, "1", r.Id)
	assert.Equal(t, float32(1), r.A)
	assert.Equal(t, uint32(2), r.B)
	assert.Equal(t, "c1", r.Subs[0].C)
	assert.Equal(t, 1, r.Subs[0].D)
	assert.Equal(t, "c2", r.Subs[1].C)
	assert.Equal(t, 2, r.Subs[1].D)

	// Set
	err = manager.Set(ctx, "2", TestObject{Id: "2"})
	assert.Nil(t, err)

	// Get
	r, c, err = manager.Get(ctx, "2")
	assert.Equal(t, "2", r.Id)
	assert.True(t, c)
	assert.Nil(t, err)

	// TTL
	time.Sleep(CacheTTL + CacheJitter)
	r, c, err = manager.Get(ctx, "2")
	assert.Empty(t, r)
	assert.False(t, c)
	assert.Nil(t, err)

	// Del
	err = manager.Del(ctx, "1")
	assert.Nil(t, err)

	// Load_err
	r, err = manager.Load(
		ctx, "1", func(k string) (TestObject, error) {
			return TestObject{}, errors.New("DataLoadFailed")
		},
	)
	assert.Empty(t, r)
	assert.Error(t, err, "DataLoadFailed")

	// Load Concurrent
	result := make([]TestObject, 0)
	mu := sync.Mutex{}
	begin := time.Now()
	g, ctx := errgroup.WithContext(ctx)
	for i := 0; i < 100; i++ {
		g.Go(
			func() error {
				r, _ := manager.Load(
					ctx, "10", func(k string) (TestObject, error) {
						time.Sleep(100 * time.Millisecond)
						fmt.Printf("Load Concurrent In: %d\n", i)
						return TestObject{Id: "10"}, nil
					},
				)
				mu.Lock()
				result = append(result, r)
				mu.Unlock()
				return nil
			},
		)
	}
	_ = g.Wait()
	fmt.Printf("Load Concurrent Cost: %s\n", time.Since(begin))
	for _, r = range result {
		assert.Equal(t, "10", r.Id)
	}
}

func RunTestStringValueCase(t *testing.T, ctx context.Context, manager *CacheManager[string, TestStringValue]) {
	// Prepare
	_ = manager.MDel(ctx, []string{"1", "2", "3", "4"})

	// Get
	r, c, err := manager.Get(ctx, "1")
	assert.Empty(t, r)
	assert.False(t, c)
	assert.Nil(t, err)

	// Load
	r, err = manager.Load(
		ctx, "1", func(k string) (TestStringValue, error) {
			return "S_1", nil
		},
	)
	assert.Equal(t, TestStringValue("S_1"), r)

	// Set
	err = manager.Set(ctx, "2", "S_2")
	assert.Nil(t, err)

	// Get
	r, c, err = manager.Get(ctx, "2")
	assert.Equal(t, TestStringValue("S_2"), r)
	assert.True(t, c)
	assert.Nil(t, err)

	// TTL
	time.Sleep(CacheTTL + CacheJitter)
	r, c, err = manager.Get(ctx, "2")
	assert.Empty(t, r)
	assert.False(t, c)
	assert.Nil(t, err)

	// Del
	err = manager.Del(ctx, "1")
	assert.Nil(t, err)

	// Load_err
	r, err = manager.Load(
		ctx, "1", func(k string) (TestStringValue, error) {
			return "", errors.New("DataLoadFailed")
		},
	)
	assert.Empty(t, r)
	assert.Error(t, err, "DataLoadFailed")

	// Load Concurrent
	result := make([]TestStringValue, 0)
	mu := sync.Mutex{}
	begin := time.Now()
	g, ctx := errgroup.WithContext(ctx)
	for i := 0; i < 100; i++ {
		g.Go(
			func() error {
				r, _ := manager.Load(
					ctx, "10", func(k string) (TestStringValue, error) {
						time.Sleep(100 * time.Millisecond)
						fmt.Printf("Load Concurrent In: %d\n", i)
						return "S_10", nil
					},
				)
				mu.Lock()
				result = append(result, r)
				mu.Unlock()
				return nil
			},
		)
	}
	_ = g.Wait()
	fmt.Printf("Load Concurrent Cost: %s\n", time.Since(begin))
	for _, r = range result {
		assert.Equal(t, TestStringValue("S_10"), r)
	}
}

func RunMultiTestObjectPtrCase(t *testing.T, ctx context.Context, manager *CacheManager[string, *TestObject]) {
	// Prepare
	_ = manager.MDel(ctx, []string{"1", "2", "3", "4"})

	// MGet
	r, err := manager.MGet(ctx, []string{"1", "2"})
	assert.Len(t, r, 0)
	assert.Nil(t, err)

	// MLoad
	r, err = manager.MLoad(
		ctx, []string{"1", "2"}, func(ks []string) (map[string]*TestObject, error) {
			rs := make(map[string]*TestObject)
			subs := make([]*SubTestObject, 0)
			subs = append(subs, &SubTestObject{C: "c1", D: 1})
			subs = append(subs, &SubTestObject{C: "c2", D: 2})
			rs["1"] = &TestObject{Id: "1", A: 1, B: 2, Subs: subs}
			rs["2"] = nil
			return rs, nil
		},
	)
	assert.Equal(t, "1", r["1"].Id)
	assert.Equal(t, float32(1), r["1"].A)
	assert.Equal(t, uint32(2), r["1"].B)
	assert.Equal(t, "c1", r["1"].Subs[0].C)
	assert.Equal(t, 1, r["1"].Subs[0].D)
	assert.Equal(t, "c2", r["1"].Subs[1].C)
	assert.Equal(t, 2, r["1"].Subs[1].D)
	assert.Nil(t, r["2"])

	// MLoad & Err
	rv1, err := manager.MLoad(
		ctx, []string{"1", "2", "3"}, func(ks []string) (map[string]*TestObject, error) {
			return nil, errors.New("MLoad err")
		},
	)
	assert.Error(t, err, "MLoad err")
	assert.Len(t, rv1, 0)

	// MSet
	kv := make(map[string]*TestObject, 2)
	kv["3"] = &TestObject{Id: "3"}
	kv["4"] = &TestObject{Id: "4"}
	err = manager.MSet(ctx, kv)
	assert.Nil(t, err)

	// MGet
	r, err = manager.MGet(ctx, []string{"2", "3", "4"})
	assert.Nil(t, r["2"])
	assert.Equal(t, "3", r["3"].Id)
	assert.Equal(t, "4", r["4"].Id)
	assert.Nil(t, err)

	// TTL
	time.Sleep(CacheTTL + CacheJitter)
	r, err = manager.MGet(ctx, []string{"3", "4"})
	assert.Len(t, r, 0)
	assert.Nil(t, err)

	// MDel
	err = manager.MDel(ctx, []string{"1", "2"})
	assert.Nil(t, err)

	// Load_err
	r, err = manager.MLoad(
		ctx, []string{"1", "2"}, func(k []string) (map[string]*TestObject, error) {
			return nil, errors.New("DataLoadFailed")
		},
	)
	assert.Nil(t, r)
	assert.Error(t, err, "DataLoadFailed")
}

func RunMultiTestObjectCase(t *testing.T, ctx context.Context, manager *CacheManager[string, TestObject]) {
	// Prepare
	_ = manager.MDel(ctx, []string{"1", "2", "3", "4"})

	// MGet
	r, err := manager.MGet(ctx, []string{"1", "2"})
	assert.Len(t, r, 0)
	assert.Nil(t, err)

	// MLoad
	r, err = manager.MLoad(
		ctx, []string{"1", "2"}, func(ks []string) (map[string]TestObject, error) {
			rs := make(map[string]TestObject)
			subs := make([]*SubTestObject, 0)
			subs = append(subs, &SubTestObject{C: "c1", D: 1})
			subs = append(subs, &SubTestObject{C: "c2", D: 2})
			rs["1"] = TestObject{Id: "1", A: 1, B: 2, Subs: subs}
			rs["2"] = TestObject{}
			return rs, nil
		},
	)
	assert.Equal(t, "1", r["1"].Id)
	assert.Equal(t, float32(1), r["1"].A)
	assert.Equal(t, uint32(2), r["1"].B)
	assert.Equal(t, "c1", r["1"].Subs[0].C)
	assert.Equal(t, 1, r["1"].Subs[0].D)
	assert.Equal(t, "c2", r["1"].Subs[1].C)
	assert.Equal(t, 2, r["1"].Subs[1].D)
	assert.Empty(t, r["2"])

	// MLoad & Err
	rv1, err := manager.MLoad(
		ctx, []string{"1", "2", "3"}, func(ks []string) (map[string]TestObject, error) {
			return nil, errors.New("MLoad err")
		},
	)
	assert.Error(t, err, "MLoad err")
	assert.Len(t, rv1, 0)

	// MSet
	kv := make(map[string]TestObject, 2)
	kv["3"] = TestObject{Id: "3"}
	kv["4"] = TestObject{Id: "4"}
	err = manager.MSet(ctx, kv)
	assert.Nil(t, err)

	// MGet
	r, err = manager.MGet(ctx, []string{"2", "3", "4"})
	assert.Empty(t, r["2"])
	assert.Equal(t, "3", r["3"].Id)
	assert.Equal(t, "4", r["4"].Id)
	assert.Nil(t, err)

	// TTL
	time.Sleep(CacheTTL + CacheJitter)
	r, err = manager.MGet(ctx, []string{"3", "4"})
	assert.Len(t, r, 0)
	assert.Nil(t, err)

	// MDel
	err = manager.MDel(ctx, []string{"1", "2"})
	assert.Nil(t, err)

	// Load_err
	r, err = manager.MLoad(
		ctx, []string{"1", "2"}, func(k []string) (map[string]TestObject, error) {
			return nil, errors.New("DataLoadFailed")
		},
	)
	assert.Empty(t, r)
	assert.Error(t, err, "DataLoadFailed")
}

func RunMultiTestStringValueCase(t *testing.T, ctx context.Context, manager *CacheManager[string, TestStringValue]) {
	// Prepare
	_ = manager.MDel(ctx, []string{"1", "2", "3", "4"})

	// MGet
	r, err := manager.MGet(ctx, []string{"1", "2"})
	assert.Len(t, r, 0)
	assert.Nil(t, err)

	// MLoad
	r, err = manager.MLoad(
		ctx, []string{"1", "2"}, func(ks []string) (map[string]TestStringValue, error) {
			rs := make(map[string]TestStringValue)
			rs["1"] = "S_1"
			rs["2"] = ""
			return rs, nil
		},
	)
	assert.Equal(t, TestStringValue("S_1"), r["1"])
	assert.Empty(t, r["2"])

	// MLoad & Err
	rv1, err := manager.MLoad(
		ctx, []string{"1", "2", "3"}, func(ks []string) (map[string]TestStringValue, error) {
			return nil, errors.New("MLoad err")
		},
	)
	assert.Error(t, err, "MLoad err")
	assert.Len(t, rv1, 0)

	// MSet
	kv := make(map[string]TestStringValue, 2)
	kv["3"] = "S_3"
	kv["4"] = "S_4"
	err = manager.MSet(ctx, kv)
	assert.Nil(t, err)

	// MGet
	r, err = manager.MGet(ctx, []string{"2", "3", "4"})
	assert.Empty(t, r["2"])
	assert.Equal(t, TestStringValue("S_3"), r["3"])
	assert.Equal(t, TestStringValue("S_4"), r["4"])
	assert.Nil(t, err)

	// TTL
	time.Sleep(CacheTTL + CacheJitter)
	r, err = manager.MGet(ctx, []string{"3", "4"})
	assert.Len(t, r, 0)
	assert.Nil(t, err)

	// MDel
	err = manager.MDel(ctx, []string{"1", "2"})
	assert.Nil(t, err)

	// Load_err
	r, err = manager.MLoad(
		ctx, []string{"1", "2"}, func(k []string) (map[string]TestStringValue, error) {
			return nil, errors.New("DataLoadFailed")
		},
	)
	assert.Empty(t, r)
	assert.Error(t, err, "DataLoadFailed")
}
