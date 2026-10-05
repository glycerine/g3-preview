// Source: src/util/permutation.cpp:35-40; src/util/vector.h:303-310.
// Source author: Leonardo de Moura (2011-06-10).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// permutationWordView is a call-local borrowed source lvalue. Its concrete
// provider/block/offset identify the address acquired by operator[], independent
// of any later vector m_data field change. It has no cold allocation or owner.
type permutationWordView struct {
	runtime *memory.Runtime
	block   memory.Block
	offset  uint64
}

func (v permutationWordView) read() (uint32, error) {
	return satAssignmentReadWord(v.runtime, v.block, v.offset)
}
func (v permutationWordView) write(x uint32) error {
	return satAssignmentWriteWord(v.runtime, v.block, v.offset, x)
}
func (p *RuntimePermutation) acquireMutable(v *RuntimeUnsignedVector, i uint32) (permutationWordView, error) {
	return p.acquireAtSource(v, i, 304)
}
func (p *RuntimePermutation) acquireAtSource(v *RuntimeUnsignedVector, i uint32, line int32) (permutationWordView, error) {
	if e := v.open(); e != nil {
		return permutationWordView{}, e
	}
	on, e := permutationAssertions(p.debugContext)
	if e != nil {
		return permutationWordView{}, e
	}
	if on {
		n, e := v.column.size()
		if e != nil {
			return permutationWordView{}, e
		}
		if i >= n {
			if e = debugAssertionFailure(p.debugContext, "src/util/vector.h", line, "idx < size()"); e != nil {
				return permutationWordView{}, e
			}
		}
	}
	if v.column.raw.block == (memory.Block{}) {
		return permutationWordView{}, arena.ErrBounds
	}
	return permutationWordView{runtime: v.column.raw.runtime, block: v.column.raw.block, offset: 8 + 4*uint64(i)}, nil
}

// std::swap acquires both arguments before it loads or publishes either word.
// Provider lifetime, initialization, and physical extent are checked only at
// the actual reached reads/writes through those acquired pointers.
func permutationSwapWords(left, right permutationWordView) error {
	x, e := left.read()
	if e != nil {
		return e
	}
	y, e := right.read()
	if e != nil {
		return e
	}
	if e = left.write(y); e != nil {
		return e
	}
	return right.write(x)
}
