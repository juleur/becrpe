package cache

import (
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/pkg/errors"
)

// Cache struct
type Cache struct {
	client redis.UniversalClient
	ttl    time.Duration
}

const userPrefix = "user:"

// NewCache func
func NewCache(redisAddress string, password string, ttl time.Duration) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: password,
		DB:       0,
	})
	if err := client.Ping().Err(); err != nil {
		return nil, errors.WithStack(err)
	}
	return &Cache{client: client, ttl: ttl}, nil
}

//** USER IP **//
// AddIP func
func (c *Cache) AddIP(userID string, userIP string, duration time.Duration) {
	_ = c.client.Set(userPrefix+userID, userIP, duration)
}

// GetIP returns addresses IP
func (c *Cache) GetIP(userID string) (string, bool) {
	s, err := c.client.Get(userPrefix + userID).Result()
	if err == redis.Nil || s != "" {
		return s, true
	}
	return s, false
}

// DeleteIP func
func (c *Cache) DeleteIP(userID string) {
	_ = c.client.Del(userPrefix + userID)
}
