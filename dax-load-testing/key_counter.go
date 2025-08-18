package main

import "sync"

type KeyCounter[T number] struct {
	pool map[T]*NumberManager[T]
	mtx  sync.Mutex
}

func (k *KeyCounter[T]) NextSK(pk T) T {
	k.mtx.Lock()
	defer k.mtx.Unlock()

	if k.pool == nil {
		k.pool = map[T]*NumberManager[T]{}
	}

	if k.pool[pk] == nil {
		k.pool[pk] = NewNumberManager[T](1, 100)
	}

	return k.pool[pk].Next()
}
