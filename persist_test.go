package mxcache_test

import (
	"testing"

	"github.com/diegohce/mxcache"
)

func Test00PersistenceSet(t *testing.T) {

	cache, err := mxcache.NewMXCache("memory://mem/?persist=cache.dat")
	if err != nil {
		t.Fatal(err)
	}

	cache.Set("name", "Diego", 3600)
}

func Test01PersistenceGet(t *testing.T) {

	cache, err := mxcache.NewMXCache("memory://mem/?persist=cache.dat")
	if err != nil {
		t.Fatal(err)
	}

	v, _ := cache.Get("name")

	t.Log(v)
}
