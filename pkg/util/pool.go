// Source: src/util/pool.h:25-46.
package util

// Pool retains recycled pointer values for deterministic LIFO reuse. The
// source is templated over T; this port accepts dynamically typed pointers to
// avoid introducing Go generics. newValue constructs a value when the cache is
// empty. destroy performs any ownership action required when Close disposes of
// cached values and may be nil when garbage collection is sufficient.
type Pool struct {
	self     *Pool
	newValue func() any
	destroy  func(any)
	objects  []any
	closed   bool
}

// NewPool constructs an empty object pool.
func NewPool(newValue func() any, destroy func(any)) *Pool {
	if newValue == nil {
		panic("g3: pool requires a factory")
	}
	pool := &Pool{newValue: newValue, destroy: destroy}
	pool.self = pool
	return pool
}

func (pool *Pool) requireOpen() {
	if pool == nil || pool.self != pool || pool.closed {
		panic("g3: invalid pool owner")
	}
}

func requirePoolPointer(value any) {
	if !validScopedPointer(value) || nilScopedPointer(value) {
		panic("g3: pool requires non-nil pointer values")
	}
}

// Mk returns the most recently recycled value, or constructs a new value when
// no recycled value is available.
func (pool *Pool) Mk() any {
	pool.requireOpen()
	if len(pool.objects) == 0 {
		value := pool.newValue()
		requirePoolPointer(value)
		return value
	}
	last := len(pool.objects) - 1
	value := pool.objects[last]
	pool.objects[last] = nil
	pool.objects = pool.objects[:last]
	return value
}

// Recycle transfers value to the pool for later reuse.
func (pool *Pool) Recycle(value any) {
	pool.requireOpen()
	requirePoolPointer(value)
	pool.objects = append(pool.objects, value)
}

// Cached returns the number of values retained for reuse.
func (pool *Pool) Cached() uint32 {
	pool.requireOpen()
	return uint32(len(pool.objects))
}

// Close destroys cached values from first to last and invalidates the owner.
// Values checked out by Mk remain owned by their callers.
func (pool *Pool) Close() {
	pool.requireOpen()
	for i, value := range pool.objects {
		if pool.destroy != nil {
			pool.destroy(value)
		}
		pool.objects[i] = nil
	}
	pool.objects = nil
	pool.newValue = nil
	pool.destroy = nil
	pool.closed = true
}
