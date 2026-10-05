// Source: src/util/heap.h:24-290; src/util/vector.h:102-129,191-202.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2006-09-14.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

func (h *IntHeap) runtimeBacked() bool { return h.runtimeQueue.runtimeBacked() }
func newIntHeapRuntime(bounds int32, c IntHeapConfig) (*IntHeap, error) {
	if c.Budget == nil || c.Runtime.Budget() != c.Budget {
		return nil, ErrSATOwner
	}
	if e := c.Budget.Reserve(intHeapOwnerBytes); e != nil {
		return nil, e
	}
	h := &IntHeap{budget: c.Budget, less: c.Less, debugContext: c.DebugContext, epoch: 1}
	h.self = h
	q := &h.runtimeQueue
	q.self = q
	q.budget = c.Budget
	q.generalSource = h
	q.bindRuntime(c.Runtime)
	e := h.pushUintRuntime(&q.runtimeValues, ^uint32(0))
	if e == nil {
		e = h.setBoundsRuntime(uint32(bounds))
	}
	if e == nil {
		e = h.cassertRuntimeAt(126)
	}
	if e != nil {
		if cleanup := h.Close(); cleanup != nil {
			return h, errors.Join(e, cleanup)
		}
		return nil, e
	}
	return h, nil
}
func (h *IntHeap) assignRuntime(other *IntHeap) error {
	if !h.runtimeBacked() || !other.runtimeBacked() {
		return ErrSATOwner
	}
	// A caller chooses the destination Runtime; copy reads the actual source
	// blocks without materializing coefficients or accounting-ticket arrays.
	h.less = other.less
	dst, src := &h.runtimeQueue, &other.runtimeQueue
	for i := uint8(0); i < 2; i++ {
		d, s := &dst.runtimeValues.raw, &src.runtimeValues.raw
		if i == 1 {
			d, s = &dst.runtimeIndices.raw, &src.runtimeIndices.raw
		}
		if d.block != (memory.Block{}) {
			actual := d.block
			e := d.runtime.Deallocate(&actual)
			if actual == (memory.Block{}) {
				h.destroyed |= 1 << i
			}
			if e != nil {
				h.sourceStop = e
				return e
			}
		}
		h.destroyed |= 1 << i
		old := d.block
		e := intHeapCopyColumn(d, s)
		if d.block != old || d.block == (memory.Block{}) {
			h.destroyed &^= 1 << i
		}
		if e != nil {
			h.sourceStop = e
			return e
		}
		h.destroyed &^= 1 << i
	}
	return nil
}
func (h *IntHeap) closeRuntime() error {
	if h.self != h {
		return ErrSATOwner
	}
	if h.closed {
		return nil
	}
	h.closing = true
	if h.sourceStop != nil {
		return h.sourceStop
	}
	if h.findFrame != nil {
		if e := h.finishFindFrame(); e != nil {
			return e
		}
	}
	if e := h.runtimeQueue.Close(); e != nil {
		return e
	}
	h.budget.Release(intHeapOwnerBytes)
	h.closed = true
	return nil
}

// CleanupStopped acknowledges only exact destruction receipts. It does not
// retry source assignment or reconstruct fields whose allocations refused.
func (h *IntHeap) CleanupStopped() error {
	if h == nil || h.self != h {
		return ErrSATOwner
	}
	if h.closed {
		return nil
	}
	if h.runtimeBacked() {
		if h.destroyed&1 != 0 {
			h.runtimeQueue.runtimeValues.raw.block = memory.Block{}
		}
		if h.destroyed&2 != 0 {
			h.runtimeQueue.runtimeIndices.raw.block = memory.Block{}
		}
		h.destroyed = 0
		if h.findFrame != nil {
			f := h.findFrame
			if f.stop != nil {
				if f.todo.raw.block != (memory.Block{}) {
					if !f.todo.raw.pendingReplaces {
						return f.stop
					}
					f.todo.raw.block = memory.Block{}
				}
				// Explicit owning acknowledgment of actual consumed-token progress only.
				// Original source stop remains in the retained frame until it is released.
				f.sourceDone = true
			}
			if e := h.finishFindFrame(); e != nil {
				return e
			}
		}
		if h.runtimeQueue.runtimeSourceStop != nil {
			if e := h.runtimeQueue.CleanupStopped(); e != nil {
				return e
			}
		} else if e := h.runtimeQueue.Close(); e != nil {
			return e
		}
		h.budget.Release(intHeapOwnerBytes)
		h.closed = true
		return nil
	}
	if h.destroyed&1 != 0 {
		h.values = nil
	}
	if h.destroyed&2 != 0 {
		h.indices = nil
	}
	h.destroyed = 0
	cause := h.sourceStop
	h.sourceStop = nil
	e := h.Close()
	h.sourceStop = cause
	return e
}

// IntHeapVector is a caller-owned int_vector, with real fixed-width payloads.
// The Runtime must outlive this owner. The reflected cold header is Go metadata.
type IntHeapVector struct {
	self            *IntHeapVector
	column          satSolverRuntimeUintColumn
	budget          *arena.Budget
	closed, closing bool
}

var intHeapVectorOwnerBytes = uint64(reflect.TypeOf(IntHeapVector{}).Size())

func NewIntHeapVector(r *memory.Runtime, b *arena.Budget) (*IntHeapVector, error) {
	if r == nil || b == nil || r.Budget() != b {
		return nil, ErrSATOwner
	}
	if e := b.Reserve(intHeapVectorOwnerBytes); e != nil {
		return nil, e
	}
	v := &IntHeapVector{budget: b}
	v.self = v
	v.column.bind(r)
	return v, nil
}
func (v *IntHeapVector) open() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed || v.closing {
		return arena.ErrClosed
	}
	return nil
}
func (v *IntHeapVector) Push(x int32) error {
	if e := v.open(); e != nil {
		return e
	}
	return v.column.push(uint32(x))
}
func (v *IntHeapVector) Size() (uint32, error) {
	if e := v.open(); e != nil {
		return 0, e
	}
	return v.column.size()
}
func (v *IntHeapVector) Get(i uint32) (int32, error) {
	if e := v.open(); e != nil {
		return 0, e
	}
	x, e := v.column.read(i)
	return int32(x), e
}
func (v *IntHeapVector) Close() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed {
		return nil
	}
	v.closing = true
	if e := v.column.finalize(); e != nil {
		return e
	}
	v.budget.Release(intHeapVectorOwnerBytes)
	v.closed = true
	return nil
}

// The source lexical todo is always fresh. Its cold Go context is charged and
// retained only if source destruction/owning abort is interrupted.
type intHeapFindFrame struct {
	todo       satSolverRuntimeUintColumn
	stop       error
	sourceDone bool
}

var intHeapFindFrameBytes = uint64(reflect.TypeOf(intHeapFindFrame{}).Size())

func (h *IntHeap) startFindFrame() error {
	if h.findFrame != nil {
		return ErrHeapStopped
	}
	if e := h.budget.Reserve(intHeapFindFrameBytes); e != nil {
		return e
	}
	h.findFrame = &intHeapFindFrame{}
	h.findFrame.todo.bind(h.runtimeQueue.runtimeValues.raw.runtime)
	return nil
}
func (h *IntHeap) finishFindFrame() error {
	f := h.findFrame
	if f == nil {
		return nil
	}
	if !f.sourceDone {
		if f.stop != nil {
			return f.stop
		}
		if f.todo.raw.block != (memory.Block{}) {
			if e := f.todo.raw.runtime.Deallocate(&f.todo.raw.block); e != nil {
				f.stop = e
				if h.sourceStop == nil {
					h.sourceStop = e
				}
				return e
			}
		}
		f.sourceDone = true
	}
	if e := f.todo.finalize(); e != nil {
		return e
	}
	h.budget.Release(intHeapFindFrameBytes)
	h.findFrame = nil
	return nil
}
func (h *IntHeap) FindLEInto(v int32, out *IntHeapVector) (err error) {
	if e := h.alive(); e != nil {
		return e
	}
	if !h.runtimeBacked() {
		return ErrSATOwner
	}
	if e := out.open(); e != nil {
		return e
	}
	if e := h.startFindFrame(); e != nil {
		return e
	}
	defer func() { err = errors.Join(err, h.finishFindFrame()) }()
	return h.findRuntime(v, out, nil)
}
func (h *IntHeap) findLESlice(v int32, dst []int32) (result []int32, err error) {
	if e := h.startFindFrame(); e != nil {
		return dst, e
	}
	defer func() { err = errors.Join(err, h.finishFindFrame()) }()
	result = dst
	err = h.findRuntime(v, nil, &result)
	return
}
func (h *IntHeap) findRuntime(v int32, out *IntHeapVector, dst *[]int32) error {
	todo := &h.findFrame.todo
	if e := h.pushUintRuntime(todo, 1); e != nil {
		return e
	}
	for {
		n, e := todo.size()
		if e != nil {
			return e
		}
		if n == 0 {
			return nil
		}
		x, e := h.backUintRuntime(todo)
		if e != nil {
			return e
		}
		if e = h.popUintRuntime(todo); e != nil {
			return e
		}
		i := int32(x)
		sz, e := h.runtimeQueue.runtimeValues.size()
		if e != nil {
			return e
		}
		if i >= int32(sz) {
			continue
		}
		value, e := h.runtimeQueue.valueRuntime(i)
		if e != nil {
			return e
		}
		if h.lt(v, value) {
			continue
		}
		value, e = h.runtimeQueue.valueRuntime(i)
		if e != nil {
			return e
		}
		if out != nil {
			if e = out.open(); e != nil {
				return e
			}
			if e = h.pushUintRuntime(&out.column, uint32(value)); e != nil {
				return e
			}
		} else {
			*dst = append(*dst, value)
		}
		if i < 0 || i > int32(^uint32(0)>>2) {
			return ErrSATSignedHeapIndexDomain
		}
		if e = h.pushUintRuntime(todo, uint32(i<<1)); e != nil {
			return e
		}
		if e = h.pushUintRuntime(todo, uint32((i<<1)+1)); e != nil {
			return e
		}
	}
}

// copy_core allocates in the destination owner domain and reads source bytes
// only after capacity/size and destination m_data publication.
func intHeapCopyColumn(dst, src *satSolverAssignmentColumn) error {
	if src.block == (memory.Block{}) {
		dst.block = memory.Block{}
		return nil
	}
	n, e := src.size()
	if e != nil {
		return e
	}
	c, e := src.capacity()
	if e != nil {
		return e
	}
	block, e := dst.runtime.Allocate(satMUSCopyRequestFor(c, 4, dst.runtime.SizeTBits()))
	dst.pending = block
	if e != nil {
		return e
	}
	if e = satAssignmentWriteWord(dst.runtime, block, 0, c); e != nil {
		return e
	}
	if e = satAssignmentWriteWord(dst.runtime, block, 4, n); e != nil {
		return e
	}
	dst.block = block
	dst.pending = memory.Block{}
	for i := uint32(0); i < n; i++ {
		word, e := src.runtime.ReadUint32(src.block, 8+4*uint64(i))
		if e != nil {
			return e
		}
		if e = dst.runtime.WriteUint32(block, 8+4*uint64(i), word); e != nil {
			return e
		}
	}

	return nil
}

func (h *IntHeap) setBoundsRuntime(n uint32) error {
	c := &h.runtimeQueue.runtimeIndices
	old, e := c.size()
	if e != nil {
		return e
	}
	if n <= old {
		if e = h.shrinkUintRuntime(c, n); e != nil {
			return e
		}
		return h.cassertRuntimeAt(154)
	}
	// set_bounds invokes unsigned vector resize directly; heap::reserve's
	// separate signed comparison must not suppress this actual growth loop.
	for {
		capacity, e := c.capacity()
		if e != nil {
			return e
		}
		if capacity >= n {
			break
		}
		if e = c.raw.expandStrideDebug(4, h.debugContext); e != nil {
			return e
		}
	}
	if e = satAssignmentWriteWord(c.raw.runtime, c.raw.block, 4, n); e != nil {
		return e
	}
	for i := old; i < n; i++ {
		if e = satAssignmentWriteWord(c.raw.runtime, c.raw.block, 8+4*uint64(i), 0); e != nil {
			return e
		}
	}
	return h.cassertRuntimeAt(154)
}

func (h *IntHeap) cloneRuntime(b *arena.Budget) (*IntHeap, error) {
	if b != h.budget || b != h.runtimeQueue.runtimeValues.raw.runtime.Budget() {
		return nil, ErrSATOwner
	}
	if e := b.Reserve(intHeapOwnerBytes); e != nil {
		return nil, e
	}
	clone := &IntHeap{budget: b, less: h.less, debugContext: h.debugContext, epoch: 1}
	clone.self = clone
	q := &clone.runtimeQueue
	q.self = q
	q.budget = b
	q.generalSource = clone
	q.bindRuntime(h.runtimeQueue.runtimeValues.raw.runtime)
	q.runtimeCompleted = 0
	e := intHeapCopyColumn(&q.runtimeValues.raw, &h.runtimeQueue.runtimeValues.raw)
	if e == nil {
		q.runtimeCompleted = 1
		e = intHeapCopyColumn(&q.runtimeIndices.raw, &h.runtimeQueue.runtimeIndices.raw)
		if e == nil {
			q.runtimeCompleted = 3
		}
	}
	if e != nil {
		if cleanup := clone.Close(); cleanup != nil {
			return clone, errors.Join(e, cleanup)
		}
		return nil, e
	}
	return clone, nil
}
func (h *IntHeap) swapRuntime(other *IntHeap) error {
	if !h.runtimeBacked() || !other.runtimeBacked() {
		return ErrSATOwner
	}
	if e := h.cassertHeapRuntimeAt(other, 260, "other.check_invariant()"); e != nil {
		return e
	}
	if e := h.cassertRuntimeAt(261); e != nil {
		return e
	}
	a, b := &h.runtimeQueue, &other.runtimeQueue
	a.runtimeValues, b.runtimeValues = b.runtimeValues, a.runtimeValues
	a.runtimeIndices, b.runtimeIndices = b.runtimeIndices, a.runtimeIndices
	a.runtimeValues.raw.self = &a.runtimeValues.raw
	b.runtimeValues.raw.self = &b.runtimeValues.raw
	a.runtimeIndices.raw.self = &a.runtimeIndices.raw
	b.runtimeIndices.raw.self = &b.runtimeIndices.raw
	if e := h.cassertHeapRuntimeAt(other, 264, "other.check_invariant()"); e != nil {
		return e
	}
	return h.cassertRuntimeAt(265)
}

func (h *IntHeap) appendRuntime(dst []int32) ([]int32, error) {
	n, e := h.runtimeQueue.runtimeValues.size()
	if e != nil {
		return dst, e
	}
	for i := uint32(1); i < n; i++ {
		v, e := h.runtimeQueue.valueRuntime(int32(i))
		if e != nil {
			return dst, e
		}
		dst = append(dst, v)
	}
	return dst, nil
}
