package storage

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-redis/redis/v8"
	"github.com/loreum-org/cortex/pkg/types"
)

// RedisStorage implements the StorageService interface for Redis
type RedisStorage struct {
	Client *redis.Client
	Config *types.StorageConfig
	ctx    context.Context
}

// NewRedisStorage creates a new Redis storage service
func NewRedisStorage(config *types.StorageConfig) (*RedisStorage, error) {
	// Parse the database number
	dbNum, err := strconv.Atoi(config.Database)
	if err != nil {
		return nil, fmt.Errorf("invalid database number: %s", config.Database)
	}

	// Create the client
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       dbNum,
	})

	// Check the connection
	ctx := context.Background()
	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	return &RedisStorage{
		Client: client,
		Config: config,
		ctx:    ctx,
	}, nil
}

// Store stores a value with the given key
func (s *RedisStorage) Store(key string, value []byte) error {
	return s.Client.Set(s.ctx, key, value, 0).Err()
}

// Retrieve retrieves a value with the given key
func (s *RedisStorage) Retrieve(key string) ([]byte, error) {
	val, err := s.Client.Get(s.ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return val, err
}

// Delete deletes a value with the given key
func (s *RedisStorage) Delete(key string) error {
	return s.Client.Del(s.ctx, key).Err()
}

// Close closes the Redis client
func (s *RedisStorage) Close() error {
	return s.Client.Close()
}
