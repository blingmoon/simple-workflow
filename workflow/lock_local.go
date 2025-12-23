package workflow

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/pkg/errors"
)

func NewLocalWorkflowLock() WorkflowLock {
	return &localWorkflowLock{
		locks: &sync.Map{},
	}
}

type localWorkflowLock struct {
	locks *sync.Map // key -> *localLockInfo
}

type localLockInfo struct {
	mu       sync.Mutex
	value    string      // 锁的值，用于验证是否是同一个持有者
	expireAt time.Time   // 过期时间
	timer    *time.Timer // 超时定时器
}

// NonBlockingSynchronized 非阻塞同步执行
func (l *localWorkflowLock) NonBlockingSynchronized(ctx context.Context, key string, maxLockTimeDuration time.Duration, f func(context.Context) error) error {
	// 检查是否已经持有锁（可重入）
	valueInterface := ctx.Value(lockKey(key))
	_, ok := valueInterface.(string)

	if ok {
		// 已经持有锁，可重入，直接执行
		return f(ctx)
	}

	// 生成随机值作为锁标识
	value := l.getRandomValue()

	// 尝试获取锁
	lockInfo, _ := l.locks.LoadOrStore(key, &localLockInfo{})
	info := lockInfo.(*localLockInfo)

	// 尝试加锁
	locked := info.mu.TryLock()
	if !locked {
		// 锁被占用，立即返回失败
		return errors.WithMessage(LockFailedError, "[localWorkflowLock.NonBlockingSynchronized] has been locked")
	}

	// 成功获取锁，设置锁信息
	info.value = value
	info.expireAt = time.Now().Add(maxLockTimeDuration)

	// 设置超时自动释放
	info.timer = time.AfterFunc(maxLockTimeDuration, func() {
		l.releaseKey(key, value)
	})

	// 创建带锁标识的 context
	withKeyCtx := context.WithValue(ctx, lockKey(key), value)

	// 确保释放锁
	defer l.releaseKey(key, value)

	// 执行函数
	return f(withKeyCtx)
}

// getRandomValue 生成随机值
func (l *localWorkflowLock) getRandomValue() string {
	return fmt.Sprintf("%d_%d", rand.Int(), time.Now().UnixNano())
}

// releaseKey 释放锁
func (l *localWorkflowLock) releaseKey(key string, value string) {
	lockInfo, ok := l.locks.Load(key)
	if !ok {
		// 锁不存在，可能已经被释放
		return
	}

	info := lockInfo.(*localLockInfo)

	// 验证是否是同一个持有者
	if info.value != value {
		log.Printf("[localWorkflowLock.releaseKey] value mismatch, expected: %s, got: %s", info.value, value)
		return
	}

	// 取消定时器
	if info.timer != nil {
		info.timer.Stop()
	}

	// 释放互斥锁
	info.mu.Unlock()

	// 从 map 中删除
	l.locks.Delete(key)
}
