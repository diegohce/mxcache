package mxcache

import (
	"log"
	"net/url"
	"strings"
)

type memRedisCache struct {
	mem   MXCacher
	redis MXCacher
}

func newMemRedisCache(u *url.URL) (MXCacher, error) {

	// mem+redis://:master@127.0.0.1/1

	c := &memRedisCache{}

	c.mem, _ = newMemoryCache(u)

	redisUri, _ := url.Parse(strings.SplitN(u.String(), "+", 2)[1])

	if r, err := newRedisCache(redisUri); err != nil {
		return nil, err
	} else {
		c.redis = r
	}

	return c, nil
}

func (c *memRedisCache) Get(key string) (interface{}, error) {

	if val, _ := c.mem.Get(key); val != nil {
		log.Println("mem+redis: mem")
		return val, nil
	}

	val, err := c.redis.Get(key)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, nil
	}

	rc := c.redis.(*redisCache)
	keyTTL := rc.RedisClient().TTL(rc.ctx, key).Val()

	c.mem.Set(key, val, int(keyTTL))

	log.Println("mem+redis: redis. TTL:", keyTTL)
	return val, nil
}

func (c *memRedisCache) Set(key string, data interface{}, ex int) error {
	c.mem.Set(key, data, ex)
	return c.redis.Set(key, data, ex)
}

func (c *memRedisCache) Expire(pattern string) (expiredKeys, error) {
	c.mem.Expire(pattern)
	return c.redis.Expire(pattern)
}
