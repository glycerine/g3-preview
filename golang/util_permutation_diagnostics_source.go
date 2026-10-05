// Source: src/util/permutation.cpp:21-76; src/util/vector.h:102-120,303-310,389-405; src/util/debug.h:60-65.
// Source author: Leonardo de Moura (2011-06-10).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

func permutationAssertions(d *DebugContext) (bool, error) {
	if !satStorageSourceDebug {
		return false, nil
	}
	if d == nil {
		return true, nil
	}
	return d.AssertionsEnabled()
}
func (p *RuntimePermutation) BindAssertions(d *DebugContext) error {
	if e := p.open(); e != nil {
		return e
	}
	if d != nil && (d.self != d || d.budget != p.budget) {
		return arena.ErrConfig
	}
	p.debugContext = d
	return nil
}
func (p *RuntimePermutation) read(v *RuntimeUnsignedVector, i uint32) (uint32, error) {
	return p.readAtSource(v, i, 309)
}
func (p *RuntimePermutation) readMutable(v *RuntimeUnsignedVector, i uint32) (uint32, error) {
	return p.readAtSource(v, i, 304)
}
func (p *RuntimePermutation) readAtSource(v *RuntimeUnsignedVector, i uint32, line int32) (uint32, error) {
	view, e := p.acquireAtSource(v, i, line)
	if e != nil {
		return 0, e
	}
	return view.read()
}
func (p *RuntimePermutation) write(v *RuntimeUnsignedVector, i, x uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	on, e := permutationAssertions(p.debugContext)
	if e != nil {
		return e
	}
	if on {
		n, e := v.column.size()
		if e != nil {
			return e
		}
		if i >= n {
			if e := debugAssertionFailure(p.debugContext, "src/util/vector.h", 304, "idx < size()"); e != nil {
				return e
			}
		}
	}
	if v.column.raw.block == (memory.Block{}) {
		return arena.ErrBounds
	}
	return satAssignmentWriteWord(v.column.raw.runtime, v.column.raw.block, 8+4*uint64(i), x)
}
func (p *RuntimePermutation) push(v *RuntimeUnsignedVector, x uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	if v.column.raw.pending != (memory.Block{}) {
		return ErrSATOwner
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
		if e = v.column.raw.expandStrideDebug(4, p.debugContext); e != nil {
			return e
		}
	}
	if e = satAssignmentWriteWord(v.column.raw.runtime, v.column.raw.block, 8+4*uint64(n), x); e != nil {
		return e
	}
	return satAssignmentWriteWord(v.column.raw.runtime, v.column.raw.block, 4, n+1)
}

// DisplayInto evaluates each source ostream insertion separately. A file badbit
// erases emission, never the later argument's reached vector access.
func (p *RuntimePermutation) DisplayInto(s *NumericTextStream) error {
	n, e := p.Size()
	if e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		if i > 0 {
			if e = s.AppendCString(" "); e != nil {
				return e
			}
		}
		if e = s.AppendUint32(i); e != nil {
			return e
		}
		if e = s.AppendCString(":"); e != nil {
			return e
		}
		v, e := p.read(&p.forward, i)
		if e != nil {
			return e
		}
		if e = s.AppendUint32(v); e != nil {
			return e
		}
	}
	return nil
}
