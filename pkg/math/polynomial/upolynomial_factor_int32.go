package polynomial

import (
	"github.com/glycerine/g3/pkg/arena"
)

// vector<int>(size,elem) invokes resize: all geometric capacity expansions
// precede logical size/default-element publication. The exact-size vector(size)
// constructor is a separate source overload and is not this allocation path.
func upolyFactorInt32Resize(values *[]int32, n uint32, value int32, budget *arena.Budget) error {
	old := uint32(len(*values))
	if n <= old {
		*values = (*values)[:n]
		return nil
	}
	for n > uint32(cap(*values)) {
		capacity, e := upolyCombinationCapacity(uint32(cap(*values)), 4)
		if e != nil {
			return e
		}
		bytes := uint64(capacity) * 4
		if e = budget.Reserve(bytes); e != nil {
			return e
		}
		next := make([]int32, capacity)
		copy(next, *values)
		budget.Release(uint64(cap(*values)) * 4)
		*values = next[:old]
	}
	*values = (*values)[:n] // source logical-size publication before construction
	for i := old; i < n; i++ {
		(*values)[i] = value
	}
	return nil
}
