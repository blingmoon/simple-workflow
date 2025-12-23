package workflow

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

var (
	LockFailedError        = errors.New("lock failed")
	LockFailedTimeOutError = errors.New("wait time out")
)

type WorkflowLock interface {
	// NonBlockingSynchronized
	//  @Description:  1.非阻塞同步块,如果没有拿到锁，立刻返回错误
	//                 2.可以重入锁
	//  @param ctx 原来的ctx
	//  @param key 分布式锁的的key
	//  @param maxLockTimeDuration 锁最大的时间
	//  @param f 具体执行函数的闭包
	//  @return error
	NonBlockingSynchronized(ctx context.Context, key string, maxLockTimeDuration time.Duration, f func(context.Context) error) error
}
