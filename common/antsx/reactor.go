package antsx

import (
	"context"
	"fmt"
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

// ---------------------- Reactor ----------------------

type Reactor struct {
	pool     *ants.Pool
	registry *sync.Map
}

// NewReactor 创建 Reactor
func NewReactor(size int) (*Reactor, error) {
	pool, err := ants.NewPool(size)
	if err != nil {
		return nil, err
	}
	return &Reactor{
		pool:     pool,
		registry: &sync.Map{},
	}, nil
}

// Submit 提交带 id 的任务，返回 Promise[T]
func Submit[T any](ctx context.Context, r *Reactor, id string, task func(ctx context.Context) (T, error)) (*Promise[T], error) {
	if _, loaded := r.registry.LoadOrStore(id, struct{}{}); loaded {
		return nil, fmt.Errorf("%w: %s", ErrDuplicateID, id)
	}

	promise := NewPromise[T](id)

	err := r.pool.Submit(func() {
		defer func() {
			r.registry.Delete(id)
			if p := recover(); p != nil {
				promise.Reject(fmt.Errorf("antsx: task %q panicked: %v", id, p))
			}
		}()

		val, err := task(ctx)
		if err != nil {
			promise.Reject(err)
		} else {
			promise.Resolve(val)
		}
	})

	if err != nil {
		r.registry.Delete(id)
		return nil, err
	}

	return promise, nil
}

// Post 提交 fire-and-forget 任务
func Post[T any](ctx context.Context, r *Reactor, task func(ctx context.Context) (T, error)) error {
	return r.pool.Submit(func() {
		defer func() {
			if p := recover(); p != nil {
				logx.WithContext(ctx).Errorf("antsx: task panicked: %v", p)
			}
		}()

		_, err := task(ctx)
		if err != nil {
			logx.WithContext(ctx).Errorf("task error: %v", err)
		}
	})
}

// Go 提交裸任务到池中，不需要 Promise/ID 开销
func (r *Reactor) Go(fn func()) error {
	return r.pool.Submit(fn)
}

// Release 释放池
func (r *Reactor) Release() {
	r.pool.Release()
}

// ActiveCount 当前运行任务数
func (r *Reactor) ActiveCount() int {
	return r.pool.Running()
}
