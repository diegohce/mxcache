package mxcache

import (
	"fmt"
	"log"
	"net/url"
)

type MXCacheCreator func(u *url.URL) (MXCacher, error)

type expiredKeys []string

type MXCacher interface {
	Get(key string) (interface{}, error)
	Set(key string, data interface{}, ex int) error
	Expire(pattern string) (expiredKeys, error)
}

var cacheBackends = map[string]MXCacheCreator{
	"memory":    newMemoryCache,
	"redis":     newRedisCache,
	"mem+redis": newMemRedisCache,
}

func NewMXCache(uri string) (MXCacher, error) {
	if uri == "" {
		return nilCache{}, nil
	}
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	backendCreator, ok := cacheBackends[u.Scheme]
	if !ok {
		return nil, fmt.Errorf("invalid cache backend %s", u.Scheme)
	}
	log.Println("Setting up", u.Scheme, "cache with", u.String())
	return backendCreator(u)
}

type nilCache struct{}

func (c nilCache) Set(key string, data interface{}, ex int) error {
	return nil
}

func (c nilCache) Get(key string) (interface{}, error) {
	return nil, nil
}

func (c nilCache) Expire(pattern string) (expiredKeys, error) {
	return []string{}, nil
}
