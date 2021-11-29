package mxcache

import (
	"bytes"
	"context"
	"encoding/gob"
	"net/url"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

type redisCache struct {
	ctx   context.Context
	cache *redis.Client
	//options *redis.Options
}

func newRedisCache(u *url.URL) (MXCacher, error) {

	// redis://:master@127.0.0.1/1

	c := &redisCache{}

	options, err := redis.ParseURL(u.String())
	if err != nil {
		return nil, err
	}

	c.cache = redis.NewClient(options)
	c.ctx = context.Background()

	err = c.cache.Ping(c.ctx).Err()

	return c, err
}

func (c *redisCache) Get(key string) (interface{}, error) {

	result := c.cache.Get(c.ctx, key)
	err := result.Err()
	if err != nil {
		if err != redis.Nil {
			return nil, err
		}
		return nil, nil
	}

	val, err := result.Bytes()
	if err != nil {
		return nil, err
	}

	var i interface{}

	if err := gob.NewDecoder(bytes.NewReader(val)).Decode(&i); err != nil {
		return nil, err
	}

	return i, nil
}

func (c *redisCache) Set(key string, data interface{}, ex int) error {
	ttl := time.Duration(0)
	if ex > 0 {
		ttl = time.Duration(ex) * time.Second
	}

	var buf bytes.Buffer

	if err := gob.NewEncoder(&buf).Encode(data); err != nil {
		return err
	}

	return c.cache.Set(c.ctx, key, buf.Bytes(), ttl).Err()
}

func (c *redisCache) Expire(pattern string) (expiredKeys, error) {

	var exkeys expiredKeys

	if !strings.Contains(pattern, "*") {
		exkeys = append(exkeys, pattern)
		return exkeys, c.cache.Del(c.ctx, pattern).Err()
	}

	keys, err := c.cache.Keys(c.ctx, pattern).Result()
	if err != nil {
		return exkeys, err
	}
	if len(keys) == 0 {
		return exkeys, nil
	}

	return keys, c.cache.Del(c.ctx, keys...).Err()
}

func (c *redisCache) RedisClient() *redis.Client {
	return c.cache
}
