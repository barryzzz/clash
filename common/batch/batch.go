package batch

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"
)

type Option = func(b *Batch)

type Result struct {
	Value interface{}
	Err   error
}

type Error struct {
	Key string
	Err error
}

type Queue struct {
	sem *semaphore.Weighted
}

func WithConcurrencyNum(n int) Option {
	return WithQueue(MakeQueue(n))
}

func WithQueue(q *Queue) Option {
	return func(b *Batch) {
		b.queue = q
	}
}

// Batch similar to errgroup, but can control the maximum number of concurrent
type Batch struct {
	result map[string]Result
	queue  *Queue
	wg     sync.WaitGroup
	mux    sync.Mutex
	err    *Error
	once   sync.Once
	ctx    context.Context
	cancel func()
}

func (b *Batch) Go(key string, fn func() (interface{}, error)) {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		if b.queue != nil {
			b.queue.sem.Acquire(b.ctx, 1)
			defer b.queue.sem.Release(1)
		}

		value, err := fn()
		if err != nil {
			b.once.Do(func() {
				b.err = &Error{key, err}
				if b.cancel != nil {
					b.cancel()
				}
			})
		}

		ret := Result{value, err}
		b.mux.Lock()
		defer b.mux.Unlock()
		b.result[key] = ret
	}()
}

func (b *Batch) Wait() *Error {
	b.wg.Wait()
	if b.cancel != nil {
		b.cancel()
	}
	return b.err
}

func (b *Batch) WaitAndGetResult() (map[string]Result, *Error) {
	err := b.Wait()
	return b.Result(), err
}

func (b *Batch) Result() map[string]Result {
	b.mux.Lock()
	defer b.mux.Unlock()
	copy := map[string]Result{}
	for k, v := range b.result {
		copy[k] = v
	}
	return copy
}

func New(ctx context.Context, opts ...Option) (*Batch, context.Context) {
	ctx, cancel := context.WithCancel(ctx)

	b := &Batch{
		result: map[string]Result{},
	}

	for _, o := range opts {
		o(b)
	}

	b.ctx = ctx
	b.cancel = cancel

	return b, ctx
}

func MakeQueue(concurrencyNum int) *Queue {
	return &Queue{sem: semaphore.NewWeighted(int64(concurrencyNum))}
}
