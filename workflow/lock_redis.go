package workflow

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

type lockKey string

const (
	delCommand = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end
`
	waitMaxTime = 20
)

func NewRedisWorkflowLock(redisClient redis.Cmdable) WorkflowLock {
	return &redisWorkflowLock{redisClient: redisClient}
}

type redisWorkflowLock struct {
	redisClient redis.Cmdable
}

func (d *redisWorkflowLock) NonBlockingSynchronized(ctx context.Context, key string, maxLockTimeDuration time.Duration, f func(ctx2 context.Context) error) error {
	valueInterface := ctx.Value(lockKey(key))
	_, ok := valueInterface.(string)
	if !ok {
		// 之前没有上锁成功
		value := d.getRandomValue()

		isLock, err := d.redisClient.SetNX(ctx, key, value, maxLockTimeDuration).Result()
		if err != nil {
			return errors.WithMessagef(LockFailedError, "[distributedLockWithRedisV8Impl.NonBlockingSynchronized], err:%v", err)
		}
		if !isLock {
			return errors.WithMessage(LockFailedError, "[distributedLockWithRedisV8Impl.NonBlockingSynchronized] has been locked")
		}

		withKeyCtx := context.WithValue(ctx, lockKey(key), value)
		defer d.releaseKey(key, value)
		return f(withKeyCtx)
	}
	// 之前成功上锁了,继续执行即可
	return f(ctx)
}

func (d *redisWorkflowLock) getRandomValue() string {
	return fmt.Sprintf("%d_%d", rand.Int(), time.Now().UnixNano())
}

func (d *redisWorkflowLock) releaseKey(key string, value string) {
	// 释放锁, 因为context 可能会被cancel，确保释放锁需要新开一个context,不能用原来的
	replyInterface, err := d.redisClient.Eval(context.Background(), delCommand, []string{key}, value).Result()
	if err != nil {
		log.Printf("[redisWorkflowLock.releaseKey] release key failed, err:%v", err)
		return
	}
	reply, ok := replyInterface.(int64)
	if !ok {
		log.Printf("[redisWorkflowLock.releaseKey] reply is not int64, reply:%v", replyInterface)
		return
	}
	if reply != 1 {
		// 没有成功释放
		log.Printf("[redisWorkflowLock.releaseKey] reply is not 1, reply:%v", reply)
		return
	}
}
