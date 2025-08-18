package main

type PoolManager[T any] struct {
	Pool chan T
}

func (p *PoolManager[T]) Acquire() T {
	return <-p.Pool
}

func (p *PoolManager[T]) Release(t T) {
	p.Pool <- t
}

func NewPoolManager[T any](size int) *PoolManager[T] {
	return &PoolManager[T]{
		Pool: make(chan T, size),
	}
}
