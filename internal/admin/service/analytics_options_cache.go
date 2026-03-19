package service

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/sync/singleflight"
)

const (
	analyticsOptionsCacheSize = 256
	analyticsOptionsCacheTTL  = time.Minute
)

type analyticsOptionsLoader func(context.Context, AnalyticsOptionsQuery) (AnalyticsOptions, error)

type analyticsOptionsCache struct {
	values *expirable.LRU[analyticsOptionsCacheKey, AnalyticsOptions]
	group  singleflight.Group
}

type analyticsOptionsCacheKey struct {
	Window   time.Duration
	RuleID   string
	Province string
	City     string
	Status   string
	SrcIP    string
}

func newAnalyticsOptionsCache() *analyticsOptionsCache {
	return &analyticsOptionsCache{
		values: expirable.NewLRU[analyticsOptionsCacheKey, AnalyticsOptions](analyticsOptionsCacheSize, nil, analyticsOptionsCacheTTL),
	}
}

func (c *analyticsOptionsCache) GetOrLoad(ctx context.Context, q AnalyticsOptionsQuery, loader analyticsOptionsLoader) (AnalyticsOptions, error) {
	key := analyticsOptionsCacheKeyFromQuery(q)
	if value, ok := c.values.Get(key); ok {
		return value, nil
	}

	resultCh := c.group.DoChan(key.singleflightKey(), func() (any, error) {
		if value, ok := c.values.Get(key); ok {
			return value, nil
		}

		value, err := loader(ctx, q)
		if err != nil {
			return AnalyticsOptions{}, err
		}

		c.values.Add(key, value)
		return value, nil
	})

	select {
	case <-ctx.Done():
		return AnalyticsOptions{}, ctx.Err()
	case result := <-resultCh:
		if result.Err != nil {
			return AnalyticsOptions{}, result.Err
		}
		value, _ := result.Val.(AnalyticsOptions)
		return value, nil
	}
}

func analyticsOptionsCacheKeyFromQuery(q AnalyticsOptionsQuery) analyticsOptionsCacheKey {
	return analyticsOptionsCacheKey{
		Window:   q.Window,
		RuleID:   q.RuleID,
		Province: q.Province,
		City:     q.City,
		Status:   q.Status,
		SrcIP:    q.SrcIP,
	}
}

func (k analyticsOptionsCacheKey) singleflightKey() string {
	return fmt.Sprintf("%d\x00%s\x00%s\x00%s\x00%s\x00%s", k.Window, k.RuleID, k.Province, k.City, k.Status, k.SrcIP)
}
