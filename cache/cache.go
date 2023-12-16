package cache

import (
	"encoding/json"
	"log"

	"github.com/patrickmn/go-cache"
	"github.com/renegatemaster/wb-l0/models"
)

type AllCache struct {
	orders *cache.Cache
}

const (
	defaultExpiration = 0
	purgeTime         = 0
)

func NewCache() *AllCache {
	Cache := cache.New(defaultExpiration, purgeTime)
	return &AllCache{
		orders: Cache,
	}
}

func (c *AllCache) Read(uid string) (item []byte, ok bool) {
	order, ok := c.orders.Get(uid)
	if ok {
		data := order.(models.SimpleOrder).Data
		log.Printf("from cache, uid=[%s]", uid)
		_, err := json.Marshal(data)
		if err != nil {
			log.Fatal("Error")
		}
		return data, true
	}
	return nil, false
}

func (c *AllCache) Update(uid string, order models.SimpleOrder) {
	c.orders.Set(uid, order, cache.DefaultExpiration)
}
