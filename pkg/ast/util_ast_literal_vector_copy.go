// Source: src/util/vector.h:102-120,159-167,191-202,536-541; src/util/debug.h:60-65.
package ast

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// copyCore preserves exact-cap allocation, header then pointer publication,
// followed by ascending iterator dereferences. A failed pre-publication block
// remains a named owning-abort root, separate from the source m_data pointer.
func (v *RuntimeASTLiteralVector) copyCore(src *RuntimeASTLiteralVector) error {
	return v.copyCoreDebug(src, v.debug)
}
func (v *RuntimeASTLiteralVector) copyCoreDebug(src *RuntimeASTLiteralVector, d *util.DebugContext) error {
	n, e := src.raw.Size()
	if e != nil {
		return e
	}
	cap, e := src.raw.Capacity()
	if e != nil {
		return e
	}
	node, e := v.prepareCopyAcquisition()
	if e != nil {
		return e
	}
	node.called = true
	v.copyExecuting = true
	h, e := node.runtime.Allocate(uint64(cap)*16 + 8)
	v.copyExecuting = false
	if h != (util.Block{}) {
		node.block = h
		node.acquired = true
	}
	node.sourceCause = e
	if e != nil {
		return e
	}
	if e = util.WriteRuntimeWord(v.raw.Runtime, h, 0, cap); e != nil {
		return e
	}
	if e = util.WriteRuntimeWord(v.raw.Runtime, h, 4, n); e != nil {
		return e
	}
	v.raw.Block = h
	node.block = util.Block{}
	node.published = true
	v.assignmentDestroyed = false
	captured := src.raw.Block
	// Source it2=begin(), then SASSERT(it2==m_data): both name the
	// same freshly published destination pointer. No extent validation.
	it2 := v.raw.Block
	on, e := util.PermutationAssertions(d)
	if e != nil {
		return e
	}
	if on && it2 != v.raw.Block {
		if e := util.DebugAssertionFailure(d, "src/util/vector.h", 113, "it2 == m_data"); e != nil {
			return e
		}
	}
	end, e := src.raw.Size()
	if e != nil {
		return e
	}
	span := ASTLiteralVectorSpan{src.raw.Runtime, captured, 8}
	for i := uint32(0); i < end; i++ {
		x, e := span.At(i)
		if e != nil {
			return e
		}
		if e = putASTLiteral(v.raw.Runtime, h, 8+16*uint64(i), x); e != nil {
			return e
		}
	}

	return nil
}
func (v *RuntimeASTLiteralVector) copyConstruct(src *RuntimeASTLiteralVector) error {
	return v.copyConstructDebug(src, v.debug)
}
func (v *RuntimeASTLiteralVector) copyConstructDebug(src *RuntimeASTLiteralVector, d *util.DebugContext) error {
	if src.raw.Block != (util.Block{}) {
		if e := v.copyCoreDebug(src, d); e != nil {
			v.constructorCause = e
			return e
		}
	}
	on, e := util.PermutationAssertions(d)
	if e != nil {
		v.constructorCause = e
		return e
	}
	if on {
		a, e := v.raw.Size()
		if e != nil {
			v.constructorCause = e
			return e
		}
		b, e := src.raw.Size()
		if e != nil {
			v.constructorCause = e
			return e
		}
		if a != b {
			if e := util.DebugAssertionFailure(d, "src/util/vector.h", 167, "size() == source.size()"); e != nil {
				v.constructorCause = e
				return e
			}
		}
	}
	v.constructed = true
	v.retirePublishedCopyAcquisitions()
	return nil
}
func (v *RuntimeASTLiteralVector) Clone() (*RuntimeASTLiteralVector, error) {
	if e := v.open(); e != nil {
		return nil, e
	}
	out, e := NewRuntimeASTLiteralVector(v.raw.Runtime, v.budget, v.debug)
	if e != nil {
		return nil, e
	}
	out.constructed = false
	if e = out.copyConstruct(v); e != nil {
		return out, e
	}
	return out, nil
}
func (v *RuntimeASTLiteralVector) Assign(src *RuntimeASTLiteralVector) error {
	return v.assignDebug(src, v.debug)
}
func (v *RuntimeASTLiteralVector) assignDebug(src *RuntimeASTLiteralVector, d *util.DebugContext) error {
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
	if v.raw.Block != (util.Block{}) {
		consumed := v.raw.Block
		e := v.raw.Runtime.Deallocate(&consumed)
		if consumed == (util.Block{}) {
			v.assignmentDestroyed = true
		}
		if e != nil {
			if v.assignmentCause == nil {
				v.assignmentCause = e
			}
			return e
		}
	}
	if src.raw.Block != (util.Block{}) {
		if e := v.copyCoreDebug(src, d); e != nil {
			if v.assignmentCause == nil {
				v.assignmentCause = e
			}
			return e
		}
	} else {
		v.raw.Block = util.Block{}
		v.assignmentDestroyed = false
	}
	v.retirePublishedCopyAcquisitions()
	return nil
}

// The linked contexts are real Go cold owners, not source allocation tickets.
// Each token is acquired only from the actual allocator return and transferred
// only at the literal source pointer publication. Older unjoined allocations
// remain named across native-valid retries; source DT never traverses them.
type runtimeASTLiteralCopyAcquisition struct {
	owner                                 *RuntimeASTLiteralVector
	runtime                               *util.Runtime
	next                                  *runtimeASTLiteralCopyAcquisition
	block                                 util.Block
	header, invocation                    uint64
	sourceCause, cleanupCause             error
	called, acquired, published, released bool
}

func (v *RuntimeASTLiteralVector) prepareCopyAcquisition() (*runtimeASTLiteralCopyAcquisition, error) {
	if v.copySerial == math.MaxUint64 {
		return nil, arena.ErrSlotLimit
	}
	n := uint64(reflect.TypeOf(runtimeASTLiteralCopyAcquisition{}).Size())
	if e := v.budget.Reserve(n); e != nil {
		return nil, e
	}
	v.copySerial++
	a := &runtimeASTLiteralCopyAcquisition{owner: v, runtime: v.raw.Runtime, next: v.copyAcquisitions, header: n, invocation: v.copySerial}
	v.copyAcquisitions = a
	return a, nil
}
func (v *RuntimeASTLiteralVector) cleanupCopyAcquisitions() error {
	for v.copyAcquisitions != nil {
		a := v.copyAcquisitions
		if a.owner != v || a.runtime != v.raw.Runtime || a.invocation == 0 || a.invocation > v.copySerial {
			return util.ErrRuntimeVectorOwner
		}
		if a.block != (util.Block{}) {
			e := v.raw.Release(&a.block)
			if a.block == (util.Block{}) {
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
func (v *RuntimeASTLiteralVector) retirePublishedCopyAcquisitions() {
	link := &v.copyAcquisitions
	for *link != nil {
		a := *link
		if a.block == (util.Block{}) && (a.published || (a.called && !a.acquired)) {
			*link = a.next
			v.budget.Release(a.header)
			a.header = 0
			continue
		}
		link = &a.next
	}
}
