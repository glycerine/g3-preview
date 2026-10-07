// Source: src/util/scoped_numeral_vector.h:25-72; src/util/vector.h:59-100,407-420,509-512.
// Source author: Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package ast

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Sole physical payload is the Runtime block. Source sizeof(mpf) is 32;
// the six meaningful codec words occupy 24 bytes and padding stays unread.
type astFloatVectorAcquisition struct {
	self                *astFloatVectorAcquisition
	owner               *astScopedFloatVector
	next                *astFloatVectorAcquisition
	original, active    util.Block
	invocation, header  uint64
	bodyCause, ownCause error
	acknowledged        bool
}

type astScopedFloatVector struct {
	copyChain                              util.ArbitraryFloatValueCopyChain
	copyReceipt                            util.ArbitraryFloatValueCopyReceipt
	copyFalseVisited, copyTrueAcknowledged bool
	copyRow                                uint32
	history                                *astFloatVectorAcquisition
	invocation                             uint64
	raw                                    util.RuntimeVectorColumn
	manager                                *util.ArbitraryFloatManager
	owner                                  *astFloatingPointPluginOwner
	initialized, destructorIndex           uint32
	sourceDone                             bool
	bodyCause, sourceCause, ownCause       error
}

func (v *astScopedFloatVector) bind(p *astFloatingPointPluginOwner) {
	v.owner = p
	v.manager = p.numeric
	v.raw.Bind(p.numeric.RuntimeOwner())
}
func (v *astScopedFloatVector) guard() error {
	if v == nil || v.owner == nil || v.owner.self != v.owner || &v.owner.values != v || v.manager != v.owner.numeric || v.raw.Self != &v.raw {
		return ErrASTHandle
	}
	if v.sourceDone {
		return arena.ErrClosed
	}
	return nil
}
func (v *astScopedFloatVector) size() (uint32, error) {
	if e := v.guard(); e != nil {
		return 0, e
	}
	if v.raw.Block == (util.Block{}) {
		return 0, nil
	}
	return util.ReadRuntimeWord(v.raw.Runtime, v.raw.Block, 4)
}
func (v *astScopedFloatVector) at(i uint32) (util.ArbitraryFloat, error) {
	if e := v.guard(); e != nil {
		return util.ArbitraryFloat{}, e
	}
	n, e := v.size()
	if e != nil {
		return util.ArbitraryFloat{}, e
	}
	if i >= n || i >= v.initialized {
		return util.ArbitraryFloat{}, arena.ErrBounds
	}
	return v.manager.CaptureRuntimeValue(v.raw.Block, 8+32*uint64(i))
}
func (v *astScopedFloatVector) expand() error {
	if v.raw.Pending != (util.Block{}) {
		// Only a NULL source retry can move this exact unjoined acquisition.
		// A failed realloc leaves source m_data dangling and is never repaired.
		if v.raw.PendingReplaces {
			return ErrASTHandle
		}
		if v.invocation == ^uint64(0) {
			return arena.ErrBounds
		}
		bytes := uint64(reflect.TypeOf(astFloatVectorAcquisition{}).Size())
		if e := v.owner.owner.BudgetField.Reserve(bytes); e != nil {
			return e
		}
		v.invocation++
		a := &astFloatVectorAcquisition{owner: v, next: v.history, original: v.raw.Pending, active: v.raw.Pending, invocation: v.invocation, header: bytes, bodyCause: v.bodyCause}
		a.self = a
		v.history = a
		v.raw.Pending = util.Block{}
	}
	old := uint32(0)
	var e error
	if v.raw.Block != (util.Block{}) {
		old, e = util.ReadRuntimeWord(v.raw.Runtime, v.raw.Block, 0)
		if e != nil {
			return e
		}
	}
	n, request, e := util.RuntimeScalarColumnNext(old, 32)
	if e != nil {
		return e
	}
	before := v.raw.Block
	h, e := v.manager.GrowRuntimeValueStorage(before, uint64(request))
	if e != nil {
		if h != before {
			v.raw.Pending = h
			v.raw.PendingReplaces = before != (util.Block{})
		}
		return e
	}
	if before == (util.Block{}) {
		v.raw.Pending = h
		if e = util.WriteRuntimeWord(v.raw.Runtime, h, 0, n); e != nil {
			return e
		}
		if e = util.WriteRuntimeWord(v.raw.Runtime, h, 4, 0); e != nil {
			return e
		}
		v.raw.Block = h
		v.raw.Pending = util.Block{}
		return nil
	}
	v.raw.Block = h
	return util.WriteRuntimeWord(v.raw.Runtime, h, 0, n)
}
func (v *astScopedFloatVector) reserve(n uint32) (err error) {
	if e := v.guard(); e != nil {
		return e
	}
	m := v.owner.owner
	if m == nil || m.Self != m {
		return ErrASTHandle
	}
	if m.formatExecuting {
		return ErrASTHandle
	}
	m.formatExecuting = true
	defer func() {
		m.formatExecuting = false
		if err != nil && v.bodyCause == nil {
			v.bodyCause = err
		}
	}()
	old, e := v.size()
	if e != nil {
		return e
	}
	if n <= old {
		return nil
	}
	for {
		cap := uint32(0)
		if v.raw.Block != (util.Block{}) {
			cap, e = util.ReadRuntimeWord(v.raw.Runtime, v.raw.Block, 0)
			if e != nil {
				return e
			}
		}
		if n <= cap {
			break
		}
		if e = v.expand(); e != nil {
			return e
		}
	}
	// Source resize publishes size before placement constructors.
	if e = util.WriteRuntimeWord(v.raw.Runtime, v.raw.Block, 4, n); e != nil {
		return e
	}
	for i := old; i < n; i++ {
		base := 8 + 32*uint64(i)
		// mpf(): format, Int default, then exponent; no padding writes.
		for _, k := range [6]uint64{0, 3, 4, 5, 1, 2} {
			if e = v.raw.Runtime.WriteUint32(v.raw.Block, base+4*k, 0); e != nil {
				return e
			}
		}
		v.initialized = i + 1
	}
	return nil
}

// Source push constructs a default row and publishes size before SET. A failed
// SET leaves that same physical row and source size intact.
func (v *astScopedFloatVector) push(source *util.ArbitraryFloat) error {
	n, e := v.size()
	if e != nil {
		return e
	}
	if n == ^uint32(0) {
		return arena.ErrBounds
	}
	if e = v.reserve(n + 1); e != nil {
		return e
	}
	return v.set(n, source)
}

func (v *astScopedFloatVector) set(i uint32, source *util.ArbitraryFloat) error {
	if e := v.guard(); e != nil {
		return e
	}
	m := v.owner.owner
	if m.formatExecuting {
		return ErrASTPending
	}
	m.formatExecuting = true
	defer func() { m.formatExecuting = false }()
	n, e := v.size()
	if e != nil {
		return e
	}
	if i >= n || i >= v.initialized {
		return arena.ErrBounds
	}
	var returned util.ArbitraryFloatValueCopyReceipt
	if e = v.copyChain.Bind(v.manager); e != nil {
		return e
	}
	e = v.copyChain.CopyToRuntimeWithReceipt(v.raw.Block, 8+32*uint64(i), v.manager, source, &returned)
	// Inline/reused-cell branches return no child and cannot overwrite the
	// retained exact older allocation call. A new allocation record is issued
	// at the real Integer allocation leaf; the chain preserves every older issued
	// child, while this field remains a diagnostic latest-call tuple.
	if returned != (util.ArbitraryFloatValueCopyReceipt{}) {
		v.copyReceipt = returned
		v.copyRow = i
		v.copyFalseVisited = false
		v.copyTrueAcknowledged = false
	}
	if e != nil && v.bodyCause == nil {
		v.bodyCause = e
	}
	return e
}
func (v *astScopedFloatVector) copyFalsePhase() error {
	if v.copyFalseVisited {
		return nil
	}
	if e := v.copyChain.Cleanup(false); e != nil {
		return e
	}
	v.copyFalseVisited = true
	return nil
}
func (v *astScopedFloatVector) copyTruePhase() error {
	if v.copyTrueAcknowledged {
		return nil
	}
	if e := v.copyChain.Cleanup(true); e != nil {
		return e
	}
	if e := v.copyChain.End(); e != nil {
		return e
	}
	v.copyTrueAcknowledged = true
	return nil
}

func (v *astScopedFloatVector) destroy(own bool) error {
	if v.sourceDone {
		if own {
			if e := v.copyFalsePhase(); e != nil {
				return e
			}
			if e := v.copyTruePhase(); e != nil {
				return e
			}
			if e := v.raw.Finalize(); e != nil {
				return e
			}
			return v.cleanupAcquisitions()
		}
		return nil
	}
	if e := v.guard(); e != nil {
		return e
	}
	if !own && v.sourceCause != nil {
		return v.sourceCause
	}
	stop := func(e error) error {
		if e != nil {
			if own {
				if v.ownCause == nil {
					v.ownCause = e
				}
			} else if v.sourceCause == nil {
				v.sourceCause = e
			}
		}
		return e
	}
	if own {
		if e := v.copyFalsePhase(); e != nil {
			return stop(e)
		}
	}
	for v.destructorIndex < v.initialized {
		offset := 8 + 32*uint64(v.destructorIndex)
		var e error
		if own {
			e = v.manager.CleanupRuntimeValue(v.raw.Block, offset)
		} else {
			e = v.manager.DelRuntimeValue(v.raw.Block, offset)
		}
		if e != nil {
			return stop(e)
		}
		v.destructorIndex++
	}
	if own {
		if e := v.copyTruePhase(); e != nil {
			return stop(e)
		}
	}
	if v.raw.Block != (util.Block{}) {
		if e := util.WriteRuntimeWord(v.raw.Runtime, v.raw.Block, 4, 0); e != nil {
			return stop(e)
		}
		if own {
			if e := v.raw.Finalize(); e != nil {
				return stop(e)
			}
		} else {
			if e := v.raw.Runtime.Deallocate(&v.raw.Block); e != nil {
				return stop(e)
			}
		}
	} else if own {
		if e := v.raw.Finalize(); e != nil {
			return stop(e)
		}
	}
	if own {
		if e := v.cleanupAcquisitions(); e != nil {
			return stop(e)
		}
	}
	v.initialized = 0
	v.sourceDone = true
	return nil
}

func (v *astScopedFloatVector) cleanupAcquisitions() error {
	for a := v.history; a != nil; a = a.next {
		if a.self != a || a.owner != v || a.invocation == 0 {
			return ErrASTHandle
		}
		if a.acknowledged {
			continue
		}
		if e := v.raw.Release(&a.active); e != nil {
			if a.ownCause == nil {
				a.ownCause = e
			}
			return e
		}
		a.acknowledged = true
	}
	return nil
}
func (v *astScopedFloatVector) acquisitionBytes() uint64 {
	var n uint64
	for a := v.history; a != nil; a = a.next {
		n += a.header
	}
	return n
}

// Metadata retirement belongs to the enclosing actual AST context end, not
// a child ACK: a later AST Close stop must retain immutable tuple/history.
func (v *astScopedFloatVector) endAcquisitionHistory() {
	for a := v.history; a != nil; a = a.next {
		v.owner.owner.BudgetField.Release(a.header)
		a.header = 0
	}
	v.history = nil
}
