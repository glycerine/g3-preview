// Source: src/util/uint_set.h; src/util/vector.h.
package util

import (
	"errors"
	"io"
	"math/bits"
	"reflect"
	"strconv"

	"github.com/glycerine/g3/pkg/arena"
)

// RuntimeUintSet borrows its Runtime and owns exactly one unsigned svector.
// Source unsigned arithmetic is retained, including wrapped iteration endpoints.
// The reflected cold owner is Go storage, not an additional source allocation.
type RuntimeUintSet struct {
	self           *RuntimeUintSet
	words          RuntimeUintColumn
	budget         *arena.Budget
	ownerBytes     uint64
	stopped        error
	destructorStop error
	sourceDone     bool
	destroyed      bool
	closed         bool
}

func NewRuntimeUintSet(r *Runtime, b *arena.Budget) (*RuntimeUintSet, error) {
	if r == nil || b == nil || r.Budget() != b {
		return nil, arena.ErrConfig
	}
	if e := r.CheckOpen(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(RuntimeUintSet{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	s := &RuntimeUintSet{budget: b, ownerBytes: n}
	s.self = s
	s.words.Bind(r)
	return s, nil
}
func (s *RuntimeUintSet) identity() error {
	if s == nil || s.self != s {
		return ErrRuntimeVectorOwner
	}
	if s.closed || s.sourceDone || s.destructorStop != nil {
		return arena.ErrClosed
	}
	return s.words.Raw.Open()
}
func (s *RuntimeUintSet) open() error {
	if e := s.identity(); e != nil {
		return e
	}
	if s.stopped != nil {
		return s.stopped
	}
	if s.words.Raw.Pending != (Block{}) {
		return ErrRuntimeVectorOwner
	}
	return nil
}
func (s *RuntimeUintSet) resize(n uint32) error {
	old, e := s.words.Size()
	if e != nil {
		return e
	}
	if n <= old {
		return s.words.Shrink(n)
	}
	for {
		cap, e := s.words.Capacity()
		if e != nil {
			return e
		}
		if n <= cap {
			break
		}
		if e = s.words.Raw.ExpandStride(4); e != nil {
			return e
		}
	}
	if e = WriteRuntimeWord(s.words.Raw.Runtime, s.words.Raw.Block, 4, n); e != nil {
		return e
	}
	for i := old; i < n; i++ {
		if e = WriteRuntimeWord(s.words.Raw.Runtime, s.words.Raw.Block, 8+4*uint64(i), 0); e != nil {
			return e
		}
	}
	return nil
}
func (s *RuntimeUintSet) MaxElem() (uint32, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	n, e := s.words.Size()
	return 32 * n, e
}
func (s *RuntimeUintSet) Reset() error {
	if e := s.open(); e != nil {
		return e
	}
	return s.words.Shrink(0)
}
func (s *RuntimeUintSet) Contains(v uint32) (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	n, e := s.words.Size()
	if e != nil {
		return false, e
	}
	i := v >> 5
	if i >= n {
		return false, nil
	}
	w, e := s.words.Read(i)
	return w&(uint32(1)<<(v&31)) != 0, e
}
func (s *RuntimeUintSet) Insert(v uint32) error {
	if e := s.open(); e != nil {
		return e
	}
	n, e := s.words.Size()
	if e != nil {
		return e
	}
	i := v >> 5
	if i >= n {
		if e = s.resize(i + 1); e != nil {
			return e
		}
	}
	w, e := s.words.Read(i)
	if e != nil {
		return e
	}
	return s.words.Write(i, w|uint32(1)<<(v&31))
}
func (s *RuntimeUintSet) Remove(v uint32) error {
	if e := s.open(); e != nil {
		return e
	}
	n, e := s.words.Size()
	if e != nil {
		return e
	}
	i := v >> 5
	if i >= n {
		return nil
	}
	w, e := s.words.Read(i)
	if e != nil {
		return e
	}
	return s.words.Write(i, w&^(uint32(1)<<(v&31)))
}
func (s *RuntimeUintSet) Empty() (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	for i := uint32(0); ; i++ {
		n, e := s.words.Size()
		if e != nil {
			return false, e
		}
		if i >= n {
			return true, nil
		}
		w, e := s.words.Read(i)
		if e != nil {
			return false, e
		}
		if w != 0 {
			return false, nil
		}
	}
}
func (s *RuntimeUintSet) NumElems() (uint32, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	var out uint32
	for i := uint32(0); ; i++ {
		n, e := s.words.Size()
		if e != nil {
			return 0, e
		}
		if i >= n {
			return out, nil
		}
		w, e := s.words.Read(i)
		if e != nil {
			return 0, e
		}
		out += uint32(bits.OnesCount32(w))
	}
}
func (s *RuntimeUintSet) Hash() (uint32, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	var out uint32
	for i := uint32(0); ; i++ {
		n, e := s.words.Size()
		if e != nil {
			return 0, e
		}
		if i >= n {
			return out, nil
		}
		w, e := s.words.Read(i)
		if e != nil {
			return 0, e
		}
		out += (i + 1) * w
	}
}
func (s *RuntimeUintSet) Union(o *RuntimeUintSet) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := o.open(); e != nil {
		return e
	}
	n, e := o.words.Size()
	if e != nil {
		return e
	}
	own, e := s.words.Size()
	if e != nil {
		return e
	}
	if n > own {
		if e = s.resize(n + 1); e != nil {
			return e
		}
	}
	for i := uint32(0); i < n; i++ {
		a, e := s.words.Read(i)
		if e != nil {
			return e
		}
		b, e := o.words.Read(i)
		if e != nil {
			return e
		}
		if e = s.words.Write(i, a|b); e != nil {
			return e
		}
	}
	return nil
}
func (s *RuntimeUintSet) Intersect(o *RuntimeUintSet) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := o.open(); e != nil {
		return e
	}
	n, e := o.words.Size()
	if e != nil {
		return e
	}
	own, e := s.words.Size()
	if e != nil {
		return e
	}
	if n < own {
		if e = s.resize(n); e != nil {
			return e
		}
	}
	for i := uint32(0); ; i++ {
		n, e := s.words.Size()
		if e != nil {
			return e
		}
		if i >= n {
			return nil
		}
		a, e := s.words.Read(i)
		if e != nil {
			return e
		}
		b, e := o.words.Read(i)
		if e != nil {
			return e
		}
		if e = s.words.Write(i, a&b); e != nil {
			return e
		}
	}
}
func (s *RuntimeUintSet) compare(o *RuntimeUintSet, subset bool) (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	if e := o.open(); e != nil {
		return false, e
	}
	n, e := s.words.Size()
	if e != nil {
		return false, e
	}
	other, e := o.words.Size()
	if e != nil {
		return false, e
	}
	if other < n {
		n = other
	}
	for i := uint32(0); i < n; i++ {
		a, e := s.words.Read(i)
		if e != nil {
			return false, e
		}
		b, e := o.words.Read(i)
		if e != nil {
			return false, e
		}
		if (subset && a&^b != 0) || (!subset && a != b) {
			return false, nil
		}
	}
	for i := n; ; i++ {
		own, e := s.words.Size()
		if e != nil {
			return false, e
		}
		if i >= own {
			break
		}
		a, e := s.words.Read(i)
		if e != nil {
			return false, e
		}
		if a != 0 {
			return false, nil
		}
	}
	if !subset {
		for i := n; ; i++ {
			other, e := o.words.Size()
			if e != nil {
				return false, e
			}
			if i >= other {
				break
			}
			b, e := o.words.Read(i)
			if e != nil {
				return false, e
			}
			if b != 0 {
				return false, nil
			}
		}
	}
	return true, nil
}
func (s *RuntimeUintSet) Equal(o *RuntimeUintSet) (bool, error)    { return s.compare(o, false) }
func (s *RuntimeUintSet) SubsetOf(o *RuntimeUintSet) (bool, error) { return s.compare(o, true) }

// Source has one global allocator. This split-provider adapter moves physical
// block provenance with the pointer; both caller-owned providers must outlive
// either wrapper after exchange. Cold wrapper storage and refunds remain on each original Budget; raw
// allocation/refunds follow the moved provider.
func (s *RuntimeUintSet) Swap(o *RuntimeUintSet) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := o.open(); e != nil {
		return e
	}
	s.words.Raw.Block, o.words.Raw.Block = o.words.Raw.Block, s.words.Raw.Block
	s.words.Raw.Runtime, o.words.Raw.Runtime = o.words.Raw.Runtime, s.words.Raw.Runtime
	return nil
}
func (s *RuntimeUintSet) Clone(r *Runtime, b *arena.Budget) (*RuntimeUintSet, error) {
	if e := s.open(); e != nil {
		return nil, e
	}
	out, e := NewRuntimeUintSet(r, b)
	if e != nil {
		return nil, e
	}
	for i := uint32(0); ; i++ {
		n, e := s.words.Size()
		if e != nil {
			return out, out.copyConstructorUnwind(e)
		}
		if i >= n {
			return out, nil
		}
		w, e := s.words.Read(i)
		if e != nil {
			return out, out.copyConstructorUnwind(e)
		}
		if e = out.words.Push(w); e != nil {
			return out, out.copyConstructorUnwind(e)
		}
	}
}

// The unsigned_vector base is already fully constructed before the derived
// uint_set copy loop. A failed loop therefore destroys that actual base before
// returning. The remaining cold/pending owner is only a Go owning-abort context.
func (s *RuntimeUintSet) copyConstructorUnwind(cause error) error {
	s.stopped = cause
	if cleanup := s.SourceFinalize(); cleanup != nil {
		return errors.Join(cause, cleanup)
	}
	return cause
}

// Clone returns its real incomplete owner on refusal for explicit caller cleanup.
func (s *RuntimeUintSet) Assign(o *RuntimeUintSet) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := o.open(); e != nil {
		return e
	}
	if s == o {
		return nil
	}
	old := s.words.Raw.Block
	if old != (Block{}) {
		token := old
		e := s.words.Raw.Runtime.Deallocate(&token)
		if token == (Block{}) {
			s.destroyed = true
		}
		if e != nil {
			s.stopped = e
			return e
		}
	}
	e := intHeapCopyColumn(&s.words.Raw, &o.words.Raw)
	if s.words.Raw.Block != old {
		s.destroyed = false
	}
	if e != nil {
		s.stopped = e
		return e
	}
	s.destroyed = false

	return nil
}

// SourceFinalize is the unsigned vector member destructor. It never absorbs
// dead source fields or failed-construction pending allocations into success.
func (s *RuntimeUintSet) SourceFinalize() error {
	if s == nil || s.self != s {
		return ErrRuntimeVectorOwner
	}
	if s.closed || s.sourceDone {
		return nil
	}
	if s.destructorStop != nil {
		return s.destructorStop
	}
	if s.words.Raw.Block != (Block{}) {
		if e := s.words.Raw.Runtime.Deallocate(&s.words.Raw.Block); e != nil {
			s.destructorStop = e
			return e
		}
	}
	s.sourceDone = true
	return nil
}
func (s *RuntimeUintSet) closeOwning() error {
	if e := s.words.Finalize(); e != nil {
		return e
	}
	s.budget.Release(s.ownerBytes)
	s.ownerBytes = 0
	s.closed = true
	return nil
}
func (s *RuntimeUintSet) Close() error {
	if s == nil || s.self != s {
		return ErrRuntimeVectorOwner
	}
	if s.closed {
		return nil
	}
	if e := s.SourceFinalize(); e != nil {
		return e
	}
	return s.closeOwning()
}

// CleanupStopped acknowledges only the reached physical guard/consumed member
// and drains actual named pending storage. Source stop history remains retained.
func (s *RuntimeUintSet) CleanupStopped() error {
	if s == nil || s.self != s {
		return ErrRuntimeVectorOwner
	}
	if s.closed {
		return nil
	}
	if s.destructorStop == nil {
		return ErrRuntimeVectorOwner
	}
	if s.destroyed {
		s.words.Raw.Block = Block{}
		s.destroyed = false
	}
	return s.closeOwning()
}
func (s *RuntimeUintSet) Display(w io.Writer) error {
	n, e := s.MaxElem()
	if e != nil {
		return e
	}
	n++
	if _, e = io.WriteString(w, "{"); e != nil {
		return e
	}
	first := true
	for i := uint32(0); i < n; i++ {
		yes, e := s.Contains(i)
		if e != nil {
			return e
		}
		if yes {
			if !first {
				if _, e = io.WriteString(w, ", "); e != nil {
					return e
				}
			}
			first = false
			if _, e = io.WriteString(w, strconv.FormatUint(uint64(i), 10)); e != nil {
				return e
			}
		}
	}
	_, e = io.WriteString(w, "}")
	return e
}

// Iterator captures endpoint but reads the current owner vector on every scan.
// In-place changes and backing relocation therefore remain source-defined.
type RuntimeUintSetIterator struct {
	owner       *RuntimeUintSet
	index, last uint32
}

func (s *RuntimeUintSet) iterator(end bool) (RuntimeUintSetIterator, error) {
	i := RuntimeUintSetIterator{owner: s}
	if end {
		n, e := s.MaxElem()
		if e != nil {
			return RuntimeUintSetIterator{}, e
		}
		i.index = n
	}
	n, e := s.MaxElem()
	if e != nil {
		return RuntimeUintSetIterator{}, e
	}
	i.last = n
	if e = i.scan(); e != nil {
		return i, e
	}
	return i, i.invariant()
}
func (s *RuntimeUintSet) Begin() (RuntimeUintSetIterator, error) { return s.iterator(false) }
func (s *RuntimeUintSet) End() (RuntimeUintSetIterator, error)   { return s.iterator(true) }
func (i *RuntimeUintSetIterator) invariant() error {
	if sourceDebug && i.index > i.last {
		return ErrRuntimeVectorAssertion
	}
	return nil
}
func (i *RuntimeUintSetIterator) scanIndex() error {
	if e := i.invariant(); e != nil {
		return e
	}
	for i.index != i.last {
		has, e := i.owner.Contains(i.index)
		if e != nil {
			return e
		}
		if has || i.index&31 == 0 {
			break
		}
		i.index++
	}
	return i.invariant()
}
func (i *RuntimeUintSetIterator) scan() error {
	if i == nil || i.owner == nil {
		return ErrRuntimeVectorOwner
	}
	if e := i.scanIndex(); e != nil {
		return e
	}
	has, e := i.owner.Contains(i.index)
	if e != nil {
		return e
	}
	if has || i.index == i.last {
		return nil
	}
	if sourceDebug && i.index&31 != 0 {
		return ErrRuntimeVectorAssertion
	}
	if e = i.invariant(); e != nil {
		return e
	}
	idx := i.index >> 5
	for i.index != i.last {
		w, e := i.owner.words.Read(idx)
		if e != nil {
			return e
		}
		if w != 0 {
			break
		}
		idx++
		i.index += 32
	}
	if e = i.invariant(); e != nil {
		return e
	}
	if i.index != i.last {
		has, e = i.owner.Contains(i.index)
		if e != nil {
			return e
		}
		if !has {
			i.index++
		}
	}
	if e = i.scanIndex(); e != nil {
		return e
	}
	return i.invariant()
}

// Source operator* reads only the cached scalar, even after its borrowed
// set has been destroyed. Actual owner reads remain at Contains/word sites.
func (i RuntimeUintSetIterator) Value() (uint32, error) {
	if i.owner == nil {
		return 0, ErrRuntimeVectorOwner
	}
	return i.index, nil
}
func (i RuntimeUintSetIterator) Equal(o RuntimeUintSetIterator) bool { return i.index == o.index }
func (i *RuntimeUintSetIterator) Advance() error {
	if i == nil {
		return ErrRuntimeVectorOwner
	}
	i.index++
	return i.scan()
}
func (i *RuntimeUintSetIterator) PostAdvance() (RuntimeUintSetIterator, error) {
	if i == nil {
		return RuntimeUintSetIterator{}, ErrRuntimeVectorOwner
	}
	old := *i
	return old, i.Advance()
}
func (i *RuntimeUintSetIterator) Assign(o RuntimeUintSetIterator) error {
	if i == nil {
		return ErrRuntimeVectorOwner
	}
	*i = o
	return nil
}
