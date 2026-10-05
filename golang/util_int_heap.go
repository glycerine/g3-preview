// Licensed under the MIT license.
// Concrete port of src/util/heap.h; src/util/vector.h:102-129,191-202.
// Source author: Leonardo de Moura, 2006-09-14.
package g3

import (
	"errors"
	"reflect"
	"strconv"
	"strings"

	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

var ErrHeapStopped = errors.New("g3: heap source assignment stopped after destruction")

var ErrHeapValue = errors.New("g3: invalid, absent, or duplicate heap value")

// IntHeapConfig borrows Less and its environment for the heap's lifetime.
// Less must be a strict ordering, must not panic or reenter the heap, and may
// change a present value's priority only together with Increased/Decreased.
// A nil Less compares the integer values themselves.
type IntHeapConfig struct {
	Budget  *arena.Budget
	Less    func(int32, int32) bool
	Runtime *memory.Runtime
	// DebugContext borrows the actual source assertion switch and lazy tag table.
	// This selected diagnostic binding requires the actual Runtime profile.
	// The external owner/table may use its own Budget and are never heap-owned.
	DebugContext *DebugContext
}

// IntHeap stores source integer values and inverse positions in pointer-free
// arrays. It is owner-local; callers serialize operations and borrowed priorities.
// Do not copy an owning IntHeap value; use Clone or Assign for source deep copies.
type IntHeap struct {
	self            *IntHeap
	debugContext    *DebugContext
	runtimeQueue    satVarQueue
	findFrame       *intHeapFindFrame
	destroyed       uint8
	sourceStop      error
	closing         bool
	values, indices []int32
	todo, found     []int32
	budget          *arena.Budget
	less            func(int32, int32) bool
	epoch           uint64
	closed          bool
}

var intHeapOwnerBytes = uint64(reflect.TypeOf(IntHeap{}).Size())

func intHeapNextCapacity(old uint64) (uint64, error) {
	n := uint64(2)
	if old != 0 {
		if old > uint64(1<<32-1) {
			return 0, arena.ErrSlotLimit
		}
		n = (3*old + 1) / 2
	}
	// Source SZ is uint32, including allocation byte-size arithmetic and
	// the two source vector header words. Go additionally bounds host ints.
	if n <= old || n > (uint64(1<<32-1)-8)/4 || n > uint64(^uint(0)>>1)/4 {
		return 0, arena.ErrSlotLimit
	}
	return n, nil
}

type IntHeapStats struct {
	OwnerBytes, ValuesBytes, IndicesBytes, TodoBytes, FoundBytes uint64
}

func NewIntHeap(bounds int32, c IntHeapConfig) (*IntHeap, error) {
	if c.Runtime != nil {
		return newIntHeapRuntime(bounds, c)
	}
	if c.DebugContext != nil {
		return nil, ErrSATOwner
	}
	if bounds < 0 {
		return nil, arena.ErrBounds
	}
	if e := c.Budget.Reserve(intHeapOwnerBytes); e != nil {
		return nil, e
	}
	h := &IntHeap{budget: c.Budget, less: c.Less, epoch: 1}
	h.self = h
	if e := h.grow(&h.values, 1); e != nil {
		h.Close()
		return nil, e
	}
	h.values = append(h.values, -1)
	if e := h.SetBounds(bounds); e != nil {
		h.Close()
		return nil, e
	}
	return h, nil
}

// Source resize repeatedly expands by (3*old+1)/2, starting at two.
// Each successful expansion remains retained if a later reservation fails.
// Old and replacement storage are both charged while copying.
func (h *IntHeap) grow(v *[]int32, need int) error {
	for need > cap(*v) {
		n, e := intHeapNextCapacity(uint64(cap(*v)))
		if e != nil {
			return e
		}
		if e := h.budget.Reserve(n * 4); e != nil {
			return e
		}
		next := make([]int32, len(*v), int(n))
		copy(next, *v)
		h.budget.Release(uint64(cap(*v)) * 4)
		*v = next
	}
	return nil
}
func (h *IntHeap) alive() error {
	if h == nil || h.self != h {
		return ErrSATOwner
	}
	if h.closed || h.closing {
		return arena.ErrClosed
	}
	if h.sourceStop != nil {
		return h.sourceStop
	}
	return nil
}
func (h *IntHeap) canChange() error {
	if e := h.alive(); e != nil {
		return e
	}
	if h.epoch == ^uint64(0) {
		return arena.ErrSlotLimit
	}
	return nil
}
func (h *IntHeap) lt(a, b int32) bool {
	if h.less == nil {
		return a < b
	}
	return h.less(a, b)
}
func (h *IntHeap) Empty() bool {
	if h != nil && h.runtimeBacked() {
		if e := h.alive(); e != nil {
			panic(e)
		}
		empty, e := h.runtimeQueue.emptyChecked()
		if e != nil {
			panic(e)
		}
		return empty
	}
	return h.Size() == 0
}
func (h *IntHeap) Size() uint32 {
	if h != nil && h.runtimeBacked() {
		n, e := h.runtimeQueue.sizeChecked()
		if e != nil {
			panic(e)
		}
		return n
	}
	if h == nil || h.closed {
		return 0
	}
	return uint32(len(h.values) - 1)
}
func (h *IntHeap) Bounds() int32 {
	if h != nil && h.runtimeBacked() {
		n, e := h.runtimeQueue.runtimeIndices.size()
		if e != nil {
			panic(e)
		}
		return int32(n)
	}
	if h == nil || h.closed {
		return 0
	}
	return int32(len(h.indices))
}
func (h *IntHeap) Contains(v int32) bool {
	if h != nil && h.runtimeBacked() {
		yes, e := h.runtimeQueue.containsChecked(satBoolVar(v))
		if e != nil {
			panic(e)
		}
		return yes
	}
	return h != nil && !h.closed && v >= 0 && int64(v) < int64(len(h.indices)) && h.indices[v] != 0
}

// CheckInvariant checks live values, as the source traversal does. Failed
// insertion can leave additional inverse entries that this check does not visit.
func (h *IntHeap) CheckInvariant() bool {
	if h != nil && h.runtimeBacked() {
		yes, e := h.invariantRuntime()
		if e != nil {
			panic(e)
		}
		return yes
	}
	if h == nil || h.closed || len(h.values) == 0 || h.values[0] != -1 {
		return false
	}
	for i := 1; i < len(h.values); i++ {
		v := h.values[i]
		if v < 0 || int64(v) >= int64(len(h.indices)) || h.indices[v] != int32(i) {
			return false
		}
		if i > 1 && h.lt(v, h.values[i/2]) {
			return false
		}
	}
	return true
}
func (h *IntHeap) up(i int32) {
	v := h.values[i]
	for p := i / 2; p != 0 && h.lt(v, h.values[p]); p = i / 2 {
		h.values[i] = h.values[p]
		h.indices[h.values[i]] = i
		i = p
	}
	h.values[i] = v
	h.indices[v] = i
}
func (h *IntHeap) down(i int32) {
	v := h.values[i]
	for {
		left := uint64(i) * 2
		if left >= uint64(len(h.values)) {
			break
		}
		child := int32(left)
		if left+1 < uint64(len(h.values)) && h.lt(h.values[child+1], h.values[child]) {
			child++
		}
		if !h.lt(h.values[child], v) {
			break
		}
		h.values[i] = h.values[child]
		h.indices[h.values[i]] = i
		i = child
	}
	h.values[i] = v
	h.indices[v] = i
}
func (h *IntHeap) Insert(v int32) error {
	if h.runtimeBacked() {
		if e := h.alive(); e != nil {
			return e
		}
		return h.runtimeQueue.insertRuntime(satBoolVar(v))
	}
	if e := h.canChange(); e != nil {
		return e
	}
	if v < 0 || int64(v) >= int64(len(h.indices)) || h.Contains(v) {
		return ErrHeapValue
	}
	if len(h.values) == 1<<31-1 {
		return arena.ErrSlotLimit
	}
	i := int32(len(h.values))
	// Source publishes membership before the fallible vector push. Preserve
	// that inverse index even when the replacement backing cannot be reserved.
	h.indices[v] = i
	h.epoch++
	if e := h.grow(&h.values, len(h.values)+1); e != nil {
		return e
	}
	h.values = append(h.values, v)
	h.up(i)
	return nil
}
func (h *IntHeap) Min() (int32, error) {
	if h.runtimeBacked() {
		if e := h.alive(); e != nil {
			return 0, e
		}
		if enabled, e := h.assertionsRuntime(); e != nil {
			return 0, e
		} else if enabled {
			yes, e := h.runtimeQueue.emptyChecked()
			if e != nil {
				return 0, e
			}
			if yes {
				if e := debugAssertionFailure(h.debugContext, "src/util/heap.h", 169, "!empty()"); e != nil {
					return 0, e
				}
			}
		}
		return h.runtimeQueue.valueRuntime(1)
	}
	if e := h.alive(); e != nil {
		return 0, e
	}
	if h.Empty() {
		return 0, ErrHeapValue
	}
	return h.values[1], nil
}
func (h *IntHeap) EraseMin() (int32, error) {
	if h.runtimeBacked() {
		if e := h.alive(); e != nil {
			return 0, e
		}
		v, e := h.runtimeQueue.nextRuntime()
		return int32(v), e
	}
	v, e := h.Min()
	if e != nil {
		return 0, e
	}
	if e = h.Erase(v); e != nil {
		return 0, e
	}
	return v, nil
}
func (h *IntHeap) Erase(v int32) error {
	if h.runtimeBacked() {
		if e := h.alive(); e != nil {
			return e
		}
		return h.runtimeQueue.eraseRuntime(satBoolVar(v))
	}
	if e := h.canChange(); e != nil {
		return e
	}
	if !h.Contains(v) || h.indices[v] < 1 || int64(h.indices[v]) >= int64(len(h.values)) {
		// A failed source insert can leave a nonzero inverse index without
		// a value slot. Guard the source's next out-of-bounds access, retaining
		// the completed prefix. In-range indices still follow source actions.
		return ErrHeapValue
	}
	i := h.indices[v]
	last := h.values[len(h.values)-1]
	h.indices[v] = 0
	h.values = h.values[:len(h.values)-1]
	if int(i) < len(h.values) {
		h.values[i] = last
		h.indices[last] = i
		if i/2 != 0 && h.lt(last, h.values[i/2]) {
			h.up(i)
		} else {
			h.down(i)
		}
	}
	h.epoch++
	return nil
}
func (h *IntHeap) Increased(v int32) error {
	if h.runtimeBacked() {
		if e := h.alive(); e != nil {
			return e
		}
		if enabled, e := h.assertionsRuntime(); e != nil {
			return e
		} else if enabled {
			yes, e := h.runtimeQueue.containsChecked(satBoolVar(v))
			if e != nil {
				return e
			}
			if !yes {
				if e := debugAssertionFailure(h.debugContext, "src/util/heap.h", 226, "contains(val)"); e != nil {
					return e
				}
			}
		}
		i, e := h.runtimeQueue.indexMutableRuntime(v)
		if e != nil {
			return e
		}
		return h.runtimeQueue.downRuntime(i)
	}
	if e := h.canChange(); e != nil {
		return e
	}
	if !h.Contains(v) || h.indices[v] < 1 || int64(h.indices[v]) >= int64(len(h.values)) {
		// A failed source insert can leave a nonzero inverse index without
		// a value slot. Guard the source's next out-of-bounds access, retaining
		// the completed prefix. In-range indices still follow source actions.
		return ErrHeapValue
	}
	h.down(h.indices[v])
	h.epoch++
	return nil
}
func (h *IntHeap) Decreased(v int32) error {
	if h.runtimeBacked() {
		if e := h.alive(); e != nil {
			return e
		}
		if enabled, e := h.assertionsRuntime(); e != nil {
			return e
		} else if enabled {
			yes, e := h.runtimeQueue.containsChecked(satBoolVar(v))
			if e != nil {
				return e
			}
			if !yes {
				if e := debugAssertionFailure(h.debugContext, "src/util/heap.h", 221, "contains(val)"); e != nil {
					return e
				}
			}
		}
		i, e := h.runtimeQueue.indexMutableRuntime(v)
		if e != nil {
			return e
		}
		return h.runtimeQueue.upRuntime(i)
	}
	if e := h.canChange(); e != nil {
		return e
	}
	if !h.Contains(v) || h.indices[v] < 1 || int64(h.indices[v]) >= int64(len(h.values)) {
		// A failed source insert can leave a nonzero inverse index without
		// a value slot. Guard the source's next out-of-bounds access, retaining
		// the completed prefix. In-range indices still follow source actions.
		return ErrHeapValue
	}
	h.up(h.indices[v])
	h.epoch++
	return nil
}
func (h *IntHeap) SetBounds(n int32) error {
	if h.runtimeBacked() {
		if e := h.alive(); e != nil {
			return e
		}
		return h.setBoundsRuntime(uint32(n))
	}
	if e := h.alive(); e != nil {
		return e
	}
	if n < 0 {
		return arena.ErrBounds
	}
	if int(n) == len(h.indices) {
		return nil
	}
	if e := h.canChange(); e != nil {
		return e
	}
	if int(n) < len(h.indices) {
		for _, v := range h.values[1:] {
			if v >= n {
				return arena.ErrBounds
			}
		}
	}
	if e := h.grow(&h.indices, int(n)); e != nil {
		return e
	}
	old := len(h.indices)
	h.indices = h.indices[:n]
	if int(n) > old {
		clear(h.indices[old:])
	}
	h.epoch++
	return nil
}
func (h *IntHeap) Reserve(n int32) error {
	if e := h.alive(); e != nil {
		return e
	}
	if h.runtimeBacked() {
		if e := h.cassertRuntimeAt(162); e != nil {
			return e
		}
	}
	if n > h.Bounds() {
		if e := h.SetBounds(n); e != nil {
			return e
		}
	}
	if h.runtimeBacked() {
		return h.cassertRuntimeAt(165)
	}
	return nil
}

func (h *IntHeap) Reset() error {
	if h.runtimeBacked() {
		if e := h.alive(); e != nil {
			return e
		}
		return h.runtimeQueue.resetRuntime()
	}
	if e := h.alive(); e != nil {
		return e
	}
	if h.Empty() {
		return nil
	}
	if e := h.canChange(); e != nil {
		return e
	}
	clear(h.indices)
	h.values = h.values[:1]
	h.values[0] = -1
	h.epoch++
	return nil
}
func (h *IntHeap) Clear() error { return h.Reset() }
func (h *IntHeap) AppendValues(dst []int32) ([]int32, error) {
	if h.runtimeBacked() {
		if e := h.alive(); e != nil {
			return dst, e
		}
		return h.appendRuntime(dst)
	}
	if e := h.alive(); e != nil {
		return dst, e
	}
	return append(dst, h.values[1:]...), nil
}

// FindLE appends each visited value before its separate left/right todo pushes.
// The slice result is a borrowed Go adapter; it is never counted as Runtime
// storage. The actual source todo is lexical and destroyed at return.
func (h *IntHeap) FindLE(v int32, dst []int32) ([]int32, error) {
	if e := h.alive(); e != nil {
		return dst, e
	}
	if h.runtimeBacked() {
		return h.findLESlice(v, dst)
	}
	var todo []int32
	defer func() { h.budget.Release(uint64(cap(todo)) * 4) }()
	if e := h.grow(&todo, 1); e != nil {
		return dst, e
	}
	todo = append(todo, 1)
	for len(todo) > 0 {
		i := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		if i >= int32(len(h.values)) || h.lt(v, h.values[i]) {
			continue
		}
		dst = append(dst, h.values[i])
		if i < 0 || i > int32(^uint32(0)>>2) {
			return dst, ErrSATSignedHeapIndexDomain
		}
		if e := h.grow(&todo, len(todo)+1); e != nil {
			return dst, e
		}
		todo = append(todo, i<<1)
		if e := h.grow(&todo, len(todo)+1); e != nil {
			return dst, e
		}
		todo = append(todo, (i<<1)+1)
	}
	return dst, nil
}
func (h *IntHeap) Display() (string, error) {
	if e := h.alive(); e != nil {
		return "", e
	}
	var out strings.Builder
	var walk func(int32, uint32) error
	walk = func(i int32, indent uint32) error {
		n := uint32(len(h.values))
		if h.runtimeBacked() {
			var e error
			n, e = h.runtimeQueue.runtimeValues.size()
			if e != nil {
				return e
			}
		}
		if i >= int32(n) {
			return nil
		}
		var value int32
		if h.runtimeBacked() {
			var e error
			value, e = h.runtimeQueue.valueRuntime(i)
			if e != nil {
				return e
			}
		} else {
			if i < 0 {
				return arena.ErrBounds
			}
			value = h.values[i]
		}
		for j := uint32(0); j < indent; j++ {
			out.WriteByte(' ')
		}
		out.WriteString(strconv.FormatInt(int64(value), 10))
		out.WriteByte('\n')
		if i < 0 || i > int32(^uint32(0)>>2) {
			return ErrSATSignedHeapIndexDomain
		}
		if e := walk(i<<1, indent+1); e != nil {
			return e
		}
		return walk((i<<1)+1, indent+1)
	}
	e := walk(1, 0)
	return out.String(), e
}

// Swap exchanges only the source value and inverse-index vectors, leaving
// comparators and temporary workspaces in place. Comparators must be compatible
// with each incoming heap, as required by the source invariant assertions.
func (h *IntHeap) Swap(other *IntHeap) error {
	if e := h.alive(); e != nil {
		return e
	}
	if e := other.alive(); e != nil {
		return e
	}
	if h == other {
		return nil
	}
	if e := h.canChange(); e != nil {
		return e
	}
	if e := other.canChange(); e != nil {
		return e
	}
	if h.runtimeBacked() || other.runtimeBacked() {
		return h.swapRuntime(other)
	}
	a := uint64(cap(h.values)+cap(h.indices)) * 4
	b := uint64(cap(other.values)+cap(other.indices)) * 4
	if h.budget != other.budget {
		if e := h.budget.Reserve(b); e != nil {
			return e
		}
		if e := other.budget.Reserve(a); e != nil {
			h.budget.Release(b)
			return e
		}
		h.budget.Release(a)
		other.budget.Release(b)
	}
	h.values, other.values = other.values, h.values
	h.indices, other.indices = other.indices, h.indices
	h.epoch++
	other.epoch++
	return nil
}

// Clone implements the source implicit heap copy constructor. Vector capacities
// are copied exactly; the source's local FindLE workspaces are not copied.
func (h *IntHeap) Clone(budget *arena.Budget) (*IntHeap, error) {
	if h != nil && h.runtimeBacked() {
		if e := h.alive(); e != nil {
			return nil, e
		}
		return h.cloneRuntime(budget)
	}
	if e := h.alive(); e != nil {
		return nil, e
	}
	if e := budget.Reserve(intHeapOwnerBytes); e != nil {
		return nil, e
	}
	c := &IntHeap{budget: budget, less: h.less, epoch: 1}
	c.self = c
	for _, p := range []struct {
		src []int32
		dst *[]int32
	}{{h.values, &c.values}, {h.indices, &c.indices}} {
		if cap(p.src) == 0 {
			continue
		}
		if e := budget.Reserve(uint64(cap(p.src)) * 4); e != nil {
			c.Close()
			return nil, e
		}
		*p.dst = make([]int32, len(p.src), cap(p.src))
		copy(*p.dst, p.src)
	}
	return c, nil
}

// Assign executes implicit base assignment followed by vector members in
// declaration order. A refused copy retains the destroyed source field and
// an actual consumed-ownership receipt; only CleanupStopped acknowledges it.
func (h *IntHeap) Assign(other *IntHeap) error {
	if e := h.alive(); e != nil {
		return e
	}
	if e := other.alive(); e != nil {
		return e
	}
	if h == other {
		return nil
	}
	if e := h.canChange(); e != nil {
		return e
	}
	if h.runtimeBacked() || other.runtimeBacked() {
		return h.assignRuntime(other)
	}
	h.less = other.less
	for i := uint8(0); i < 2; i++ {
		dst, src := &h.values, other.values
		if i == 1 {
			dst, src = &h.indices, other.indices
		}
		h.budget.Release(uint64(cap(*dst)) * 4)
		h.destroyed |= 1 << i
		if cap(src) == 0 {
			*dst = nil
			h.destroyed &^= 1 << i
			continue
		}
		if e := h.budget.Reserve(uint64(cap(src)) * 4); e != nil {
			h.sourceStop = e
			return e
		}
		next := make([]int32, len(src), cap(src))
		copy(next, src)
		*dst = next
		h.destroyed &^= 1 << i
	}
	h.epoch++
	return nil
}
func (h *IntHeap) Stats() IntHeapStats {
	if h == nil || h.closed {
		return IntHeapStats{}
	}
	v, i := uint64(cap(h.values))*4, uint64(cap(h.indices))*4
	if h.destroyed&1 != 0 {
		v = 0
	}
	if h.destroyed&2 != 0 {
		i = 0
	}
	return IntHeapStats{intHeapOwnerBytes, v, i, uint64(cap(h.todo)) * 4, uint64(cap(h.found)) * 4}
}
func (h *IntHeap) Close() error {
	if h != nil && h.self != h {
		return ErrSATOwner
	}
	if h != nil && h.runtimeBacked() {
		return h.closeRuntime()
	}
	if h != nil && h.sourceStop != nil {
		return h.sourceStop
	}
	if h == nil || h.closed {
		return nil
	}
	s := h.Stats()
	h.budget.Release(s.OwnerBytes + s.ValuesBytes + s.IndicesBytes + s.TodoBytes + s.FoundBytes)
	h.values = nil
	h.indices = nil
	h.todo = nil
	h.found = nil
	h.less = nil
	h.closed = true
	return nil
}

// IntHeapIterator is cold checked metadata, never a raw internal slice.
type IntHeapIterator struct {
	owner   *IntHeap
	epoch   uint64
	index   int
	runtime *memory.Runtime
	block   memory.Block
}

func (h *IntHeap) Begin() IntHeapIterator {
	if h != nil && h.runtimeBacked() {
		if e := h.alive(); e != nil {
			panic(e)
		}
		return IntHeapIterator{runtime: h.runtimeQueue.runtimeValues.raw.runtime, block: h.runtimeQueue.runtimeValues.raw.block, index: 1}
	}
	if h == nil {
		return IntHeapIterator{}
	}
	return IntHeapIterator{owner: h, epoch: h.epoch, index: 1}
}
func (h *IntHeap) End() IntHeapIterator {
	if h != nil && h.runtimeBacked() {
		if e := h.alive(); e != nil {
			panic(e)
		}
		n, e := h.runtimeQueue.runtimeValues.size()
		if e != nil {
			panic(e)
		}
		return IntHeapIterator{runtime: h.runtimeQueue.runtimeValues.raw.runtime, block: h.runtimeQueue.runtimeValues.raw.block, index: int(n)}
	}
	if h == nil {
		return IntHeapIterator{}
	}
	return IntHeapIterator{owner: h, epoch: h.epoch, index: len(h.values)}
}
func (it IntHeapIterator) valid() error {
	if e := it.owner.alive(); e != nil {
		return e
	}
	if it.epoch != it.owner.epoch || it.index < 1 || it.index > len(it.owner.values) {
		return arena.ErrInvalidSlot
	}
	return nil
}
func (it IntHeapIterator) Value() (int32, error) {
	if it.runtime != nil {
		x, e := it.runtime.ReadUint32(it.block, 8+4*uint64(it.index))
		return int32(x), e
	}
	if e := it.valid(); e != nil {
		return 0, e
	}
	if it.index == len(it.owner.values) {
		return 0, arena.ErrBounds
	}
	return it.owner.values[it.index], nil
}
func (it *IntHeapIterator) Next() error {
	if it != nil && it.runtime != nil {
		n, e := it.runtime.Size(it.block)
		if e != nil {
			return e
		}
		if uint64(it.index+1) > (n-8)/4 {
			return arena.ErrBounds
		}
		it.index++
		return nil
	}
	if it == nil {
		return arena.ErrInvalidSlot
	}
	if e := it.valid(); e != nil {
		return e
	}
	if it.index == len(it.owner.values) {
		return arena.ErrBounds
	}
	it.index++
	return nil
}
func (it IntHeapIterator) Equal(other IntHeapIterator) bool { return it == other }
