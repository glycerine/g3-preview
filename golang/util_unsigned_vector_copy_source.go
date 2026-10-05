// Source: src/util/vector.h:102-120,159-167,191-202,536-541; src/util/debug.h:60-65.
// Source author: Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
)

// copyCore preserves exact-cap allocation, header then pointer publication,
// followed by ascending iterator dereferences. A failed pre-publication block
// remains a named owning-abort root, separate from the source m_data pointer.
func (v *RuntimeUnsignedVector) copyCore(src *RuntimeUnsignedVector) error {
	return v.copyCoreDebug(src, nil)
}
func (v *RuntimeUnsignedVector) copyCoreDebug(src *RuntimeUnsignedVector, d *DebugContext) error {
	n, e := src.column.size()
	if e != nil {
		return e
	}
	cap, e := src.column.capacity()
	if e != nil {
		return e
	}
	node, e := v.prepareCopyAcquisition()
	if e != nil {
		return e
	}
	node.called = true
	h, e := node.runtime.Allocate(uint64(cap)*4 + 8)
	if h != (memory.Block{}) {
		node.block = h
		node.acquired = true
	}
	node.sourceCause = e
	if e != nil {
		return e
	}
	if e = satAssignmentWriteWord(v.column.raw.runtime, h, 0, cap); e != nil {
		return e
	}
	if e = satAssignmentWriteWord(v.column.raw.runtime, h, 4, n); e != nil {
		return e
	}
	v.column.raw.block = h
	node.block = memory.Block{}
	node.published = true
	v.assignmentDestroyed = false
	captured := src.column.raw.block
	// Source it2=begin(), then SASSERT(it2==m_data): both name the
	// same freshly published destination pointer. No extent validation.
	it2 := v.column.raw.block
	on, e := permutationAssertions(d)
	if e != nil {
		return e
	}
	if on && it2 != v.column.raw.block {
		return ErrSATAssertion
	}
	end, e := src.column.size()
	if e != nil {
		return e
	}
	for i := uint32(0); i < end; i++ {
		x, e := satAssignmentReadWord(src.column.raw.runtime, captured, 8+4*uint64(i))
		if e != nil {
			return e
		}
		if e = satAssignmentWriteWord(v.column.raw.runtime, h, 8+4*uint64(i), x); e != nil {
			return e
		}
	}
	return nil
}
func (v *RuntimeUnsignedVector) copyConstruct(src *RuntimeUnsignedVector) error {
	return v.copyConstructDebug(src, nil)
}
func (v *RuntimeUnsignedVector) copyConstructDebug(src *RuntimeUnsignedVector, d *DebugContext) error {
	if src.column.raw.block != (memory.Block{}) {
		if e := v.copyCoreDebug(src, d); e != nil {
			v.constructorCause = e
			return e
		}
	}
	on, e := permutationAssertions(d)
	if e != nil {
		v.constructorCause = e
		return e
	}
	if on {
		a, e := v.column.size()
		if e != nil {
			v.constructorCause = e
			return e
		}
		b, e := src.column.size()
		if e != nil {
			v.constructorCause = e
			return e
		}
		if a != b {
			v.constructorCause = ErrSATAssertion
			return ErrSATAssertion
		}
	}
	v.constructed = true
	v.retirePublishedCopyAcquisitions()
	return nil
}
func (v *RuntimeUnsignedVector) Clone() (*RuntimeUnsignedVector, error) {
	if e := v.open(); e != nil {
		return nil, e
	}
	out, e := NewRuntimeUnsignedVector(v.column.raw.runtime, v.budget)
	if e != nil {
		return nil, e
	}
	out.constructed = false
	if e = out.copyConstruct(v); e != nil {
		return out, e
	}
	return out, nil
}
func (v *RuntimeUnsignedVector) Assign(src *RuntimeUnsignedVector) error {
	return v.assignDebug(src, nil)
}
func (v *RuntimeUnsignedVector) assignDebug(src *RuntimeUnsignedVector, d *DebugContext) error {
	if e := v.open(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	if v == src {
		return nil
	}
	// destroy does not clear m_data. Only a successful new pointer publication
	// or null-source branch replaces that literal dangling source field.
	if v.column.raw.block != (memory.Block{}) {
		consumed := v.column.raw.block
		e := v.column.raw.runtime.Deallocate(&consumed)
		if consumed == (memory.Block{}) {
			v.assignmentDestroyed = true
		}
		if e != nil {
			if v.assignmentCause == nil {
				v.assignmentCause = e
			}
			return e
		}
	}
	if src.column.raw.block != (memory.Block{}) {
		if e := v.copyCoreDebug(src, d); e != nil {
			if v.assignmentCause == nil {
				v.assignmentCause = e
			}
			return e
		}
	} else {
		v.column.raw.block = memory.Block{}
		v.assignmentDestroyed = false
	}
	v.retirePublishedCopyAcquisitions()
	return nil
}

// The linked contexts are real Go cold owners, not source allocation tickets.
// Each token is acquired only from the actual allocator return and transferred
// only at the literal source pointer publication. Older unjoined allocations
// remain named across native-valid retries; source DT never traverses them.
type runtimeUnsignedCopyAcquisition struct {
	owner                                 *RuntimeUnsignedVector
	runtime                               *memory.Runtime
	next                                  *runtimeUnsignedCopyAcquisition
	block                                 memory.Block
	header, invocation                    uint64
	sourceCause, cleanupCause             error
	called, acquired, published, released bool
}

func (v *RuntimeUnsignedVector) prepareCopyAcquisition() (*runtimeUnsignedCopyAcquisition, error) {
	if v.copySerial == math.MaxUint64 {
		return nil, arena.ErrSlotLimit
	}
	n := uint64(reflect.TypeOf(runtimeUnsignedCopyAcquisition{}).Size())
	if e := v.budget.Reserve(n); e != nil {
		return nil, e
	}
	v.copySerial++
	a := &runtimeUnsignedCopyAcquisition{owner: v, runtime: v.column.raw.runtime, next: v.copyAcquisitions, header: n, invocation: v.copySerial}
	v.copyAcquisitions = a
	return a, nil
}
func (v *RuntimeUnsignedVector) cleanupCopyAcquisitions() error {
	for v.copyAcquisitions != nil {
		a := v.copyAcquisitions
		if a.owner != v || a.runtime != v.column.raw.runtime || a.invocation == 0 || a.invocation > v.copySerial {
			return ErrSATOwner
		}
		if a.block != (memory.Block{}) {
			e := v.column.raw.release(&a.block)
			if a.block == (memory.Block{}) {
				a.released = true
			}
			if e != nil {
				if a.cleanupCause == nil {
					a.cleanupCause = e
				}
				if v.copyCleanupCause == nil {
					v.copyCleanupCause = e
				}
				return e
			}
		}
		v.copyAcquisitions = a.next
		v.budget.Release(a.header)
		a.header = 0
	}
	return nil
}

// A completed copy no longer needs its acquisition-only context after the
// actual block token has transferred to m_data. Failed constructors and body
// prefixes do not call this; genuinely unjoined tokens remain named. A
// completed later operation also retires earlier calls that returned no token;
// their first source cause is already retained on the actual vector owner.
func (v *RuntimeUnsignedVector) retirePublishedCopyAcquisitions() {
	link := &v.copyAcquisitions
	for *link != nil {
		a := *link
		if a.block == (memory.Block{}) && (a.published || (a.called && !a.acquired)) {
			*link = a.next
			v.budget.Release(a.header)
			a.header = 0
			continue
		}
		link = &a.next
	}
}
