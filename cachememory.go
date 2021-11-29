package mxcache

import (
	"encoding/gob"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ExpireErrors []error

func (e ExpireErrors) Error() string {
	var s strings.Builder

	last := len(e) - 1

	for i, err := range e {
		s.WriteString(err.Error())
		if i < last {
			s.WriteString(", ")
		}
	}
	return s.String()
}

func (e ExpireErrors) Err() error {
	if len(e) > 0 {
		return e
	}
	return nil
}

type cacheStorage map[string]cit

type cit struct {
	Tstamp time.Time
	Ex     time.Duration
	Datum  interface{}
}

type memoryCache struct {
	Cache   cacheStorage
	mutex   *sync.RWMutex
	persist string
	ticker  *time.Ticker
}

func newMemoryCache(u *url.URL) (MXCacher, error) {
	ttl := 0
	configTtl := u.Query().Get("ttl")
	persist := u.Query().Get("persist")

	if configTtl == "" {
		configTtl = "0"
	}

	ttl, err := strconv.Atoi(configTtl)
	if err != nil {
		return nil, err
	}

	c := &memoryCache{}

	c.mutex = &sync.RWMutex{}
	c.Cache = make(cacheStorage)
	c.persist = persist

	c.loadPersistentData()

	go func(ttl int) {
		if ttl == 0 {
			return
		}
		c.ticker = time.NewTicker(time.Duration(ttl) * time.Second)
		//tick := time.Tick(time.Duration(ttl) * time.Second)
		tick := c.ticker.C
		for range tick {
			c.gc()
		}
	}(ttl)

	return c, nil
}

func (c *memoryCache) Set(key string, data interface{}, ex int) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	ttl := time.Duration(0)
	if ex > 0 {
		ttl = time.Duration(ex) * time.Second
	}

	c.Cache[key] = cit{
		Tstamp: time.Now(),
		Ex:     ttl,
		Datum:  data,
	}

	c.persistData()

	return nil
}

func (c *memoryCache) Get(key string) (interface{}, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	cacheUnit, ok := c.Cache[key]
	if !ok {
		// no error, just no value
		return nil, nil
	}

	if cacheUnit.Ex > 0 && time.Since(cacheUnit.Tstamp) >= cacheUnit.Ex {
		//delete(c.cache, key)
		return nil, nil
	}

	//value found
	return cacheUnit.Datum, nil
}

func (c *memoryCache) Expire(pattern string) (expiredKeys, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	defer log.Printf("Memory cache after expiring %+v\n", c.Cache)

	var exkeys expiredKeys

	if !strings.Contains(pattern, "*") {
		exkeys = append(exkeys, pattern)
		delete(c.Cache, pattern)
		c.persistData()
		return exkeys, nil
	}

	var exerr ExpireErrors

	for k, cu := range c.Cache {

		if match, err := filepath.Match(pattern, k); err != nil {
			exerr = append(exerr, err)

		} else if match {
			if cu.Ex > 0 {
				exkeys = append(exkeys, k)
				delete(c.Cache, k)
			}
		}
	}

	c.persistData()
	return exkeys, exerr.Err()
}

func (c *memoryCache) gc() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	log.Println("Starting GC")
	for key, cacheUnit := range c.Cache {
		if cacheUnit.Ex > 0 && time.Since(cacheUnit.Tstamp) >= cacheUnit.Ex {
			delete(c.Cache, key)
		}
	}
	log.Println("Done GC")
	c.persistData()
}

func (c *memoryCache) persistData() {
	if c.persist == "" {
		return
	}

	f, err := os.OpenFile(c.persist, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("cache persist data opening file", c.persist, err)
		return
	}
	defer f.Close()

	if err := gob.NewEncoder(f).Encode(c.Cache); err != nil {
		log.Println("cache persisting data on", c.persist, err)
		return
	}

}

func (c *memoryCache) loadPersistentData() {
	if c.persist == "" {
		return
	}

	f, err := os.OpenFile(c.persist, os.O_RDONLY, 0644)
	if err != nil {
		//log.Println("cache load persistent data opening file", c.persist, err)
		return
	}
	defer f.Close()

	if err := gob.NewDecoder(f).Decode(&c.Cache); err != nil {
		log.Println("cache load persistent data on", c.persist, err)
		return
	}

}
