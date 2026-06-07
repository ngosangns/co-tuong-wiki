package cacheutil

import (
	"context"
	"sync"
)

type Group[T any] struct {
	mu    sync.Mutex
	calls map[string]*call[T]
}

type call[T any] struct {
	done  chan struct{}
	value T
	err   error
}

func NewGroup[T any]() *Group[T] {
	return &Group[T]{calls: map[string]*call[T]{}}
}

func (group *Group[T]) Do(ctx context.Context, key string, fn func() (T, error)) (T, bool, error) {
	group.mu.Lock()
	if group.calls == nil {
		group.calls = map[string]*call[T]{}
	}
	if active := group.calls[key]; active != nil {
		group.mu.Unlock()
		select {
		case <-active.done:
			return active.value, true, active.err
		case <-ctx.Done():
			var zero T
			return zero, true, ctx.Err()
		}
	}

	active := &call[T]{done: make(chan struct{})}
	group.calls[key] = active
	group.mu.Unlock()

	value, err := fn()

	group.mu.Lock()
	active.value = value
	active.err = err
	close(active.done)
	delete(group.calls, key)
	group.mu.Unlock()

	return value, false, err
}
