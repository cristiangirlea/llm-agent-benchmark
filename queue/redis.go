package queue

import (
	"context"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type RedisQueue struct {
	client *redis.Client
	queue  string
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

func (rq *RedisQueue) Enqueue(task string) error {
	return rq.client.RPush(ctx, rq.queue, task).Err()
}

func (rq *RedisQueue) Dequeue() (string, error) {
	result, err := rq.client.LPop(ctx, rq.queue).Result()
	if err == redis.Nil {
		return "", nil // empty queue
	}
	return result, err
}
