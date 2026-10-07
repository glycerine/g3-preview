// Source: src/util/buffer.h:25-48.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
	"math"
	"reflect"
)

// Lexical source ptr_buffer<expr>: inline16, doubling overflow, full old+new
// charge. Handles are scalar; the helper retains no borrowed caller backing.
type PointerBuffer[T any] struct {
	inline  [16]T
	count   int
	spill   []T
	charged uint64
}

func (a *PointerBuffer[T]) Slice() []T {
	if a.spill == nil {
		return a.inline[:a.count]
	}
	return a.spill
}
func (a *PointerBuffer[T]) Push(budget *arena.Budget, h T) error {
	if a.spill == nil && a.count < len(a.inline) {
		a.inline[a.count] = h
		a.count++
		return nil
	}
	if a.spill == nil || len(a.spill) == cap(a.spill) {
		n := uint64(32)
		if a.spill != nil {
			n = uint64(cap(a.spill)) * 2
		}
		width := uint64(reflect.TypeFor[T]().Size())
		if n > math.MaxUint32 || n > uint64(^uint(0)>>1)/width {
			return arena.ErrBounds
		}
		bytes := n * width
		if e := budget.Reserve(bytes); e != nil {
			return e
		}
		next := make([]T, a.count, int(n))
		copy(next, a.Slice())
		budget.Release(a.charged)
		a.charged = bytes
		a.spill = next
	}
	a.spill = append(a.spill, h)
	a.count++
	return nil
}
func (a *PointerBuffer[T]) Append(budget *arena.Budget, hs []T) error {
	for _, h := range hs {
		if e := a.Push(budget, h); e != nil {
			return e
		}
	}
	return nil
}
func (a *PointerBuffer[T]) Close(budget *arena.Budget) {
	budget.Release(a.charged)
	a.spill = nil
	a.count = 0
	a.charged = 0
}
