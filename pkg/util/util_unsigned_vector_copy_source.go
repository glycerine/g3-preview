// Source: src/util/vector.h:102-120,159-167,191-202,536-541; src/util/debug.h:60-65.
package util

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// copyCore preserves exact-cap allocation, header then pointer publication,
// followed by ascending iterator dereferences. A failed pre-publication block
// remains a named owning-abort root, separate from the source m_data pointer.
func (v *RuntimeUnsignedVector) copyCore(src *RuntimeUnsignedVector) error {
	return v.copyCoreDebug(src, nil)
}
func (v *RuntimeUnsignedVector) copyCoreDebug(src *RuntimeUnsignedVector, d *DebugContext) error {
	n, e := src.Column.Size()
	if e != nil {
		return e
	}
	cap, e := src.Column.Capacity()
	if e != nil {
		return e
	}
	node, e := v.prepareCopyAcquisition()
	if e != nil {
		return e
	}
	node.called = true
	h, e := node.runtime.Allocate(uint64(cap)*4 + 8)
	if h != (Block{}) {
		node.block = h
		node.acquired = true
	}
	node.sourceCause = e
	if e != nil {
		return e
	}
	if e = WriteRuntimeWord(v.Column.Raw.Runtime, h, 0, cap); e != nil {
		return e
	}
	if e = WriteRuntimeWord(v.Column.Raw.Runtime, h, 4, n); e != nil {
		return e
	}
	v.Column.Raw.Block = h
	node.block = Block{}
	node.published = true
	v.assignmentDestroyed = false
	captured := src.Column.Raw.Block
	// Source it2=begin(), then SASSERT(it2==m_data): both name the
	// same freshly published destination pointer. No extent validation.
	it2 := v.Column.Raw.Block
	on, e := PermutationAssertions(d)
	if e != nil {
		return e
	}
	if on && it2 != v.Column.Raw.Block {
		if e := DebugAssertionFailure(d, "src/util/vector.h", 113, "it2 == m_data"); e != nil {
			return e
		}
	}
	end, e := src.Column.Size()
	if e != nil {
		return e
	}
	for i := uint32(0); i < end; i++ {
		x, e := ReadRuntimeWord(src.Column.Raw.Runtime, captured, 8+4*uint64(i))
		if e != nil {
			return e
		}
		if e = WriteRuntimeWord(v.Column.Raw.Runtime, h, 8+4*uint64(i), x); e != nil {
			return e
		}
	}
	return nil
}
func (v *RuntimeUnsignedVector) copyConstruct(src *RuntimeUnsignedVector) error {
	return v.copyConstructDebug(src, nil)
}
func (v *RuntimeUnsignedVector) copyConstructDebug(src *RuntimeUnsignedVector, d *DebugContext) error {
	if src.Column.Raw.Block != (Block{}) {
		if e := v.copyCoreDebug(src, d); e != nil {
			v.constructorCause = e
			return e
		}
	}
	on, e := PermutationAssertions(d)
	if e != nil {
		v.constructorCause = e
		return e
	}
	if on {
		a, e := v.Column.Size()
		if e != nil {
			v.constructorCause = e
			return e
		}
		b, e := src.Column.Size()
		if e != nil {
			v.constructorCause = e
			return e
		}
		if a != b {
			if e := DebugAssertionFailure(d, "src/util/vector.h", 167, "size() == source.size()"); e != nil {
				v.constructorCause = e
				return e
			}
		}
	}
	v.Constructed = true
	v.retirePublishedCopyAcquisitions()
	return nil
}
func (v *RuntimeUnsignedVector) Clone() (*RuntimeUnsignedVector, error) {
	if e := v.open(); e != nil {
		return nil, e
	}
	out, e := NewRuntimeUnsignedVector(v.Column.Raw.Runtime, v.budget)
	if e != nil {
		return nil, e
	}
	out.Constructed = false
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
	if v.Column.Raw.Block != (Block{}) {
		consumed := v.Column.Raw.Block
		e := v.Column.Raw.Runtime.Deallocate(&consumed)
		if consumed == (Block{}) {
			v.assignmentDestroyed = true
		}
		if e != nil {
			if v.assignmentCause == nil {
				v.assignmentCause = e
			}
			return e
		}
	}
	if src.Column.Raw.Block != (Block{}) {
		if e := v.copyCoreDebug(src, d); e != nil {
			if v.assignmentCause == nil {
				v.assignmentCause = e
			}
			return e
		}
	} else {
		v.Column.Raw.Block = Block{}
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
	runtime                               *Runtime
	next                                  *runtimeUnsignedCopyAcquisition
	block                                 Block
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
	a := &runtimeUnsignedCopyAcquisition{owner: v, runtime: v.Column.Raw.Runtime, next: v.copyAcquisitions, header: n, invocation: v.copySerial}
	v.copyAcquisitions = a
	return a, nil
}
func (v *RuntimeUnsignedVector) cleanupCopyAcquisitions() error {
	for v.copyAcquisitions != nil {
		a := v.copyAcquisitions
		if a.owner != v || a.runtime != v.Column.Raw.Runtime || a.invocation == 0 || a.invocation > v.copySerial {
			return ErrRuntimeVectorOwner
		}
		if a.block != (Block{}) {
			e := v.Column.Raw.Release(&a.block)
			if a.block == (Block{}) {
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
		if a.block == (Block{}) && (a.published || (a.called && !a.acquired)) {
			*link = a.next
			v.budget.Release(a.header)
			a.header = 0
			continue
		}
		link = &a.next
	}
}
