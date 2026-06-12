package main

import "sync"

type number interface {
	uint | uint8 | uint16 | uint32 | uint64 | int | int8 | int16 | int32 | int64 | float32 | float64
}

// Random is an exhaustive random number generator that relies on the fact that Go does not guarantee the order in memory for maps
type Random[T number] struct {
	numberPool map[T]T
	limit      uint
	offset     T
	mtx        sync.Mutex
}

func (r *Random[T]) Next() T {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if len(r.numberPool) == 0 {
		r.numberPool = make(map[T]T)
		for c := 0; c < int(r.limit); c++ {
			r.numberPool[T(c)] = T(c)
		}
	}

	var out T

	for k, v := range r.numberPool {
		out = v
		delete(r.numberPool, k)
		break
	}

	return out + r.offset
}

func NewRandom[T number](limit uint, offset T) *Random[T] {
	return &Random[T]{
		limit:  limit,
		offset: offset,
	}
}
