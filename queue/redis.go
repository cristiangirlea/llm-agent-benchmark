package queue

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type RedisQueue struct {
	client *redis.Client
	queue  string
}

var DefaultQueue *RedisQueue

func InitDefaultQueue(addr, queueName string) {
	DefaultQueue = NewRedisQueue(addr, queueName)
}

func Ping() error {
	if DefaultQueue == nil {
		return fmt.Errorf("queue not initialized")
	}
	return DefaultQueue.client.Ping(ctx).Err()
}

func NewRedisQueue(addr, queueName string) *RedisQueue {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	return &RedisQueue{
		client: rdb,
		queue:  queueName,
	}
}
