// Source: src/util/vector.h, ptr_vector. Licensed under the MIT license.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
	"reflect"
)

type PointerVector[T comparable] struct{ values []T }

func (v *PointerVector[T]) Push(budget *arena.Budget, d T) error {
	if len(v.values) == cap(v.values) {
		stride := uint64(reflect.TypeOf(d).Size())
		n, e := SourceVectorCapacity(cap(v.values), stride)
		if e != nil {
			return e
		}
		if e = budget.Reserve(uint64(n) * stride); e != nil {
			return e
		}
		next := make([]T, len(v.values), n)
		copy(next, v.values)
		budget.Release(uint64(cap(v.values)) * stride)
		v.values = next
	}
	v.values = append(v.values, d)
	return nil
}
func (v *PointerVector[T]) Erase(d T) {
	for i, h := range v.values {
		if h == d {
			copy(v.values[i:], v.values[i+1:])
			v.values[len(v.values)-1] = *new(T)
			v.values = v.values[:len(v.values)-1]
			return
		}
	}
}
func (v *PointerVector[T]) Close(budget *arena.Budget) {
	budget.Release(uint64(cap(v.values)) * uint64(reflect.TypeFor[T]().Size()))
	v.values = nil
}

func (v *PointerVector[T]) Slice() []T { return v.values }
