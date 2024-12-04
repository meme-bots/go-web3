package utils

import (
	"github.com/dgraph-io/ristretto"
	"github.com/eko/gocache/lib/v4/cache"
	rstore "github.com/eko/gocache/store/ristretto/v4"
)

func NewCache() (*cache.Cache[[]byte], error) {
	rcache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000000,
		MaxCost:     100,
		BufferItems: 64,
	})
	if err != nil {
		return nil, err
	}

	store_ := rstore.NewRistretto(rcache)
	manager := cache.New[[]byte](store_)
	return manager, nil
}
