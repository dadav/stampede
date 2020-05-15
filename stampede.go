package stampede

import (
	"context"
	"sync"
	"time"

	"github.com/goware/stampede/singleflight"
)

// Prevents cache stampede https://en.wikipedia.org/wiki/Cache_stampede by only running a
// single data fetch operation per expired / missing key regardless of number of requests to that key.

func NewCache(freshFor, ttl time.Duration) *Cache {
	return &Cache{
		freshFor: freshFor,
		ttl:      ttl,
		values:   make(map[string]*value),
	}
}

type Cache struct {
	values map[string]*value

	freshFor time.Duration
	ttl      time.Duration

	mu        sync.RWMutex
	callGroup singleflight.Group
}

func (c *Cache) Get(ctx context.Context, key string, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	return c.get(ctx, key, false, fn)
}

func (c *Cache) GetFresh(ctx context.Context, key string, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	return c.get(ctx, key, true, fn)
}

func (c *Cache) Set(ctx context.Context, key string, fn func(ctx context.Context) (interface{}, error)) (interface{}, bool, error) {
	return c.callGroup.Do(ctx, key, c.set(key, fn))
}

func (c *Cache) get(ctx context.Context, key string, freshOnly bool, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	c.mu.RLock()
	val, _ := c.values[key]
	c.mu.RUnlock()

	// value exists and is fresh - just return
	if val.IsFresh() {
		return val.Value(), nil
	}

	// value exists and is stale, and we're OK with serving it stale while updating in the background
	if !freshOnly && !val.IsExpired() {
		go c.Set(ctx, key, fn)
		return val.Value(), nil
	}

	// value doesn't exist or is expired, or is stale and we need it fresh - sync update
	v, _, err := c.Set(ctx, key, fn)
	return v, err
}

func (c *Cache) set(key string, fn singleflight.DoFunc) singleflight.DoFunc {
	return singleflight.DoFunc(func(ctx context.Context) (interface{}, error) {
		val, err := fn(ctx)
		if err != nil {
			return nil, err
		}

		c.mu.Lock()
		c.values[key] = &value{
			v:          val,
			expiry:     time.Now().Add(c.ttl),
			bestBefore: time.Now().Add(c.freshFor),
		}
		c.mu.Unlock()

		return val, nil
	})
}

type value struct {
	v interface{}

	bestBefore time.Time // cache entry freshness cutoff
	expiry     time.Time // cache entry time to live cutoff
}

func (v *value) IsFresh() bool {
	if v == nil {
		return false
	}
	return v.bestBefore.After(time.Now())
}

func (v *value) IsExpired() bool {
	if v == nil {
		return true
	}
	return v.expiry.Before(time.Now())
}

func (v *value) Value() interface{} {
	return v.v
}