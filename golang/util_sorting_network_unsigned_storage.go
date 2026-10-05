// Source: src/util/sorting_network.h:36-37,42-45,98-108; src/util/vector.h:303-329,350-365,437-440.
// Source author: Nikolaj Bjorner (2013-11-07); Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

func (s *RuntimeUnsignedSortingNetwork) acquire(v *RuntimeUnsignedVector, i uint32, line int32) (permutationWordView, error) {
	if e := v.open(); e != nil {
		return permutationWordView{}, e
	}
	on, e := permutationAssertions(s.debug)
	if e != nil {
		return permutationWordView{}, e
	}
	if on {
		n, e := v.column.size()
		if e != nil {
			return permutationWordView{}, e
		}
		if i >= n {
			if e = debugAssertionFailure(s.debug, "src/util/vector.h", line, "idx < size()"); e != nil {
				return permutationWordView{}, e
			}
		}
	}
	if v.column.raw.block == (memory.Block{}) {
		return permutationWordView{}, arena.ErrBounds
	}
	return permutationWordView{runtime: v.column.raw.runtime, block: v.column.raw.block, offset: 8 + 4*uint64(i)}, nil
}
func (s *RuntimeUnsignedSortingNetwork) read(v *RuntimeUnsignedVector, i uint32, line int32) (uint32, error) {
	p, e := s.acquire(v, i, line)
	if e != nil {
		return 0, e
	}
	return p.read()
}
func (s *RuntimeUnsignedSortingNetwork) write(v *RuntimeUnsignedVector, i, x uint32, line int32) error {
	p, e := s.acquire(v, i, line)
	if e != nil {
		return e
	}
	return p.write(x)
}
func (s *RuntimeUnsignedSortingNetwork) push(v *RuntimeUnsignedVector, x uint32) error {
	if e := v.open(); e != nil {
		return e
	}

	n, e := v.column.size()
	if e != nil {
		return e
	}
	cap, e := v.column.capacity()
	if e != nil {
		return e
	}
	if n == cap {
		s.executing = true
		e = v.expandUnsignedSource(s.debug)
		s.executing = false
		if e != nil {
			return e
		}
	}
	if e = satAssignmentWriteWord(v.column.raw.runtime, v.column.raw.block, 8+4*uint64(n), x); e != nil {
		return e
	}
	return satAssignmentWriteWord(v.column.raw.runtime, v.column.raw.block, 4, n+1)
}
