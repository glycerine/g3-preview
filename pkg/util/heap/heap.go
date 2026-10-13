// Concrete port of src/util/heap.h; src/util/vector.h:102-129,191-202.
package heap

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

var ErrStopped = errors.New("g3: heap source assignment stopped after destruction")

var ErrValue = errors.New("g3: invalid, absent, or duplicate heap value")
var ErrOwner = errors.New("g3: invalid heap owner")

// Config borrows Less, its environment and an optional DebugContext for the
// heap's lifetime.
// Less must be a strict ordering, must not panic or reenter the heap, and may
// change a present value's priority only together with Increased/Decreased.
// A nil Less compares the integer values themselves.
type Config struct {
	Budget *arena.Budget
	Less   func(int32, int32) bool
	Debug  *diagnostic.DebugContext
}

// Int stores source integer values and inverse positions in pointer-free
// arrays. It is owner-local; callers serialize operations and borrowed priorities.
// Do not copy an owning Int value; use Clone or Assign for source deep copies.
type Int struct {
	self            *Int
	destroyed       uint8
	sourceStop      error
	values, indices []int32
	budget          *arena.Budget
	less            func(int32, int32) bool
	debug           *diagnostic.DebugContext
	epoch           uint64
	closed          bool
}

var ownerBytes = uint64(reflect.TypeOf(Int{}).Size())

func nextCapacity(old uint64) (uint64, error) {
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

type Stats struct {
	OwnerBytes, ValuesBytes, IndicesBytes uint64
}

func New(bounds int32, c Config) (*Int, error) {
	if bounds < 0 {
		return nil, arena.ErrBounds
	}
	if e := c.Budget.Reserve(ownerBytes); e != nil {
		return nil, e
	}
	h := &Int{budget: c.Budget, less: c.Less, debug: c.Debug, epoch: 1}
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
	if e := h.cassertInvariant(); e != nil {
		h.Close()
		return nil, e
	}
	return h, nil
}

// CASSERT first tests the process assertion switch and the "heap" tag. The
// borrowed context is explicit until the process-wide debug binding exists.
func (h *Int) cassert(line int32, condition string, test func() bool) error {
	if h.debug == nil {
		return nil
	}
	on, err := diagnostic.SourceAssertionsEnabled(h.debug)
	if err != nil || !on {
		return err
	}
	on, err = h.debug.IsDebugEnabled("heap")
	if err != nil || !on {
		return err
	}
	if test() {
		return nil
	}
	return h.assertFailure(line, condition)
}

func (h *Int) assertFailure(line int32, condition string) error {
	if err := h.debug.NotifyAssertionViolation("src/util/heap.h", line, condition); err != nil {
		return err
	}
	return h.debug.InvokeAssertionDebugger()
}

func (h *Int) cassertInvariant() error {
	if h.debug == nil {
		return nil
	}
	on, err := diagnostic.SourceAssertionsEnabled(h.debug)
	if err != nil || !on {
		return err
	}
	on, err = h.debug.IsDebugEnabled("heap")
	if err != nil || !on {
		return err
	}
	// check_invariant_core reports failures through nested SASSERTs. Its
	// Boolean return is always true once those actions continue.
	var walk func(int) error
	walk = func(i int) error {
		if i >= len(h.values) {
			return nil
		}
		v := h.values[i]
		if v < 0 || int64(v) >= int64(len(h.indices)) || h.indices[v] != int32(i) {
			if err := h.assertFailure(63, "m_value2indices[m_values[idx]] == idx"); err != nil {
				return err
			}
		}
		if i > 1 && h.lt(v, h.values[i/2]) {
			if err := h.assertFailure(64, "parent(idx) == 0 || !less_than(m_values[idx], m_values[parent(idx)])"); err != nil {
				return err
			}
		}
		if err := walk(i * 2); err != nil {
			return err
		}
		return walk(i*2 + 1)
	}
	return walk(1)
}

// Source resize repeatedly expands by (3*old+1)/2, starting at two.
// Each successful expansion remains retained if a later reservation fails.
// Old and replacement storage are both charged while copying.
func (h *Int) grow(v *[]int32, need int) error {
	for need > cap(*v) {
		n, e := nextCapacity(uint64(cap(*v)))
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
func (h *Int) alive() error {
	if h == nil || h.self != h {
		return ErrOwner
	}
	if h.closed {
		return arena.ErrClosed
	}
	if h.sourceStop != nil {
		return h.sourceStop
	}
	return nil
}
func (h *Int) canChange() error {
	if e := h.alive(); e != nil {
		return e
	}
	if h.epoch == ^uint64(0) {
		return arena.ErrSlotLimit
	}
	return nil
}
func (h *Int) lt(a, b int32) bool {
	if h.less == nil {
		return a < b
	}
	return h.less(a, b)
}
func (h *Int) Empty() bool {
	return h.Size() == 0
}
func (h *Int) Size() uint32 {
	if h == nil || h.self != h || h.closed {
		return 0
	}
	return uint32(len(h.values) - 1)
}
func (h *Int) Bounds() int32 {
	if h == nil || h.self != h || h.closed {
		return 0
	}
	return int32(len(h.indices))
}
func (h *Int) Contains(v int32) bool {
	return h != nil && h.self == h && !h.closed && v >= 0 && int64(v) < int64(len(h.indices)) && h.indices[v] != 0
}

// CheckInvariant checks live values, as the source traversal does. Failed
// insertion can leave additional inverse entries that this check does not visit.
func (h *Int) CheckInvariant() bool {
	if h == nil || h.self != h || h.closed || len(h.values) == 0 || h.values[0] != -1 {
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
func (h *Int) up(i int32) error {
	v := h.values[i]
	for p := i / 2; p != 0 && h.lt(v, h.values[p]); p = i / 2 {
		h.values[i] = h.values[p]
		h.indices[h.values[i]] = i
		i = p
	}
	h.values[i] = v
	h.indices[v] = i
	return h.cassertInvariant()
}
func (h *Int) down(i int32) error {
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
	return h.cassertInvariant()
}
func (h *Int) Insert(v int32) error {
	if e := h.canChange(); e != nil {
		return e
	}
	if e := h.cassertInvariant(); e != nil {
		return e
	}
	if e := h.cassert(232, "!contains(val)", func() bool { return !h.Contains(v) }); e != nil {
		return e
	}
	if v < 0 || int64(v) >= int64(len(h.indices)) || h.Contains(v) {
		return ErrValue
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
	if e := h.up(i); e != nil {
		return e
	}
	return h.cassertInvariant()
}
func (h *Int) Min() (int32, error) {
	if e := h.alive(); e != nil {
		return 0, e
	}
	if h.Empty() {
		return 0, ErrValue
	}
	return h.values[1], nil
}
func (h *Int) EraseMin() (int32, error) {
	if e := h.alive(); e != nil {
		return 0, e
	}
	if e := h.cassertInvariant(); e != nil {
		return 0, e
	}
	if h.Empty() {
		return 0, ErrValue
	}
	v := h.values[1]
	h.indices[v] = 0
	last := h.values[len(h.values)-1]
	h.values = h.values[:len(h.values)-1]
	h.epoch++
	if !h.Empty() {
		h.values[1] = last
		h.indices[last] = 1
		if e := h.down(1); e != nil {
			return 0, e
		}
	}
	return v, h.cassertInvariant()
}
func (h *Int) Erase(v int32) error {
	if e := h.canChange(); e != nil {
		return e
	}
	if e := h.cassertInvariant(); e != nil {
		return e
	}
	if !h.Contains(v) || h.indices[v] < 1 || int64(h.indices[v]) >= int64(len(h.values)) {
		// A failed source insert can leave a nonzero inverse index without
		// a value slot. Guard the source's next out-of-bounds access, retaining
		// the completed prefix. In-range indices still follow source actions.
		return ErrValue
	}
	i := h.indices[v]
	last := h.values[len(h.values)-1]
	h.indices[v] = 0
	h.values = h.values[:len(h.values)-1]
	h.epoch++
	if int(i) < len(h.values) {
		h.values[i] = last
		h.indices[last] = i
		if i/2 != 0 && h.lt(last, h.values[i/2]) {
			if e := h.up(i); e != nil {
				return e
			}
		} else {
			if e := h.down(i); e != nil {
				return e
			}
		}
	}
	return h.cassertInvariant()
}
func (h *Int) Increased(v int32) error {
	if e := h.canChange(); e != nil {
		return e
	}
	if !h.Contains(v) || h.indices[v] < 1 || int64(h.indices[v]) >= int64(len(h.values)) {
		// A failed source insert can leave a nonzero inverse index without
		// a value slot. Guard the source's next out-of-bounds access, retaining
		// the completed prefix. In-range indices still follow source actions.
		return ErrValue
	}
	h.epoch++
	if e := h.down(h.indices[v]); e != nil {
		return e
	}
	return nil
}
func (h *Int) Decreased(v int32) error {
	if e := h.canChange(); e != nil {
		return e
	}
	if !h.Contains(v) || h.indices[v] < 1 || int64(h.indices[v]) >= int64(len(h.values)) {
		// A failed source insert can leave a nonzero inverse index without
		// a value slot. Guard the source's next out-of-bounds access, retaining
		// the completed prefix. In-range indices still follow source actions.
		return ErrValue
	}
	h.epoch++
	if e := h.up(h.indices[v]); e != nil {
		return e
	}
	return nil
}
func (h *Int) SetBounds(n int32) error {
	if e := h.alive(); e != nil {
		return e
	}
	if n < 0 {
		return arena.ErrBounds
	}
	if int(n) == len(h.indices) {
		return h.cassertInvariant()
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
	return h.cassertInvariant()
}
func (h *Int) Reserve(n int32) error {
	if e := h.alive(); e != nil {
		return e
	}
	if e := h.cassertInvariant(); e != nil {
		return e
	}
	if n > h.Bounds() {
		if e := h.SetBounds(n); e != nil {
			return e
		}
	}
	return h.cassertInvariant()
}

func (h *Int) Reset() error {
	if e := h.alive(); e != nil {
		return e
	}
	if e := h.cassertInvariant(); e != nil {
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
	return h.cassertInvariant()
}
func (h *Int) Clear() error { return h.Reset() }
func (h *Int) AppendValues(dst []int32) ([]int32, error) {
	if e := h.alive(); e != nil {
		return dst, e
	}
	return append(dst, h.values[1:]...), nil
}

// Swap exchanges only the source value and inverse-index vectors, leaving
// comparators and temporary workspaces in place. Comparators must be compatible
// with each incoming heap, as required by the source invariant assertions.
func (h *Int) Swap(other *Int) error {
	if e := h.alive(); e != nil {
		return e
	}
	if e := other.alive(); e != nil {
		return e
	}
	if h == other {
		return nil
	}
	if e := other.cassertInvariant(); e != nil {
		return e
	}
	if e := h.cassertInvariant(); e != nil {
		return e
	}
	if e := h.canChange(); e != nil {
		return e
	}
	if e := other.canChange(); e != nil {
		return e
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
	if e := other.cassertInvariant(); e != nil {
		return e
	}
	return h.cassertInvariant()
}

// Clone implements the source implicit heap copy constructor. Vector capacities
// are copied exactly.
func (h *Int) Clone(budget *arena.Budget) (*Int, error) {
	if e := h.alive(); e != nil {
		return nil, e
	}
	if e := budget.Reserve(ownerBytes); e != nil {
		return nil, e
	}
	c := &Int{budget: budget, less: h.less, debug: h.debug, epoch: 1}
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
func (h *Int) Assign(other *Int) error {
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

// CleanupStopped releases an assignment prefix after a source-ordered copy
// refusal. It does not retry the assignment or reconstruct destroyed fields.
func (h *Int) CleanupStopped() error {
	if h == nil || h.self != h {
		return ErrOwner
	}
	if h.closed {
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

func (h *Int) Stats() Stats {
	if h == nil || h.self != h || h.closed {
		return Stats{}
	}
	v, i := uint64(cap(h.values))*4, uint64(cap(h.indices))*4
	if h.destroyed&1 != 0 {
		v = 0
	}
	if h.destroyed&2 != 0 {
		i = 0
	}
	return Stats{ownerBytes, v, i}
}
func (h *Int) Close() error {
	if h != nil && h.self != h {
		return ErrOwner
	}
	if h != nil && h.sourceStop != nil {
		return h.sourceStop
	}
	if h == nil || h.closed {
		return nil
	}
	s := h.Stats()
	h.budget.Release(s.OwnerBytes + s.ValuesBytes + s.IndicesBytes)
	h.values = nil
	h.indices = nil
	h.less = nil
	h.debug = nil
	h.closed = true
	return nil
}

// Iterator is cold checked metadata, never a raw internal slice.
type Iterator struct {
	owner *Int
	epoch uint64
	index int
}

func (h *Int) Begin() Iterator {
	if h == nil {
		return Iterator{}
	}
	return Iterator{owner: h, epoch: h.epoch, index: 1}
}
func (h *Int) End() Iterator {
	if h == nil {
		return Iterator{}
	}
	return Iterator{owner: h, epoch: h.epoch, index: len(h.values)}
}
func (it Iterator) valid() error {
	if e := it.owner.alive(); e != nil {
		return e
	}
	if it.epoch != it.owner.epoch || it.index < 1 || it.index > len(it.owner.values) {
		return arena.ErrInvalidSlot
	}
	return nil
}
func (it Iterator) Value() (int32, error) {
	if e := it.valid(); e != nil {
		return 0, e
	}
	if it.index == len(it.owner.values) {
		return 0, arena.ErrBounds
	}
	return it.owner.values[it.index], nil
}
func (it *Iterator) Next() error {
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
func (it Iterator) Equal(other Iterator) bool { return it == other }
