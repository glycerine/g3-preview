// Source: src/sat/sat_solver.cpp; src/sat/sat_clause.h. Licensed under the MIT license.
// Concrete standalone storage translation of sat_clause.h/.cpp.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"sync/atomic"
)

var (
	ErrSATOwner     = errors.New("sat: foreign or copied storage owner")
	ErrSATAssertion = errors.New("sat: source assertion failure")
	ErrSATKind      = errors.New("sat: invalid pointer-union kind")
)
var satStorageNextOwner atomic.Uint64

func satNextOwner() (uint64, error) {
	for {
		n := satStorageNextOwner.Load()
		if n == ^uint64(0) {
			return 0, arena.ErrSlotLimit
		}
		if satStorageNextOwner.CompareAndSwap(n, n+1) {
			return n + 1, nil
		}
	}
}

type satStoredClauseRef struct {
	owner                         uint64
	slot, generation, storageKind uint32
}
type satClauseOffset satStoredClauseRef
type satClauseHotHeader struct{ ID, Size, Capacity, Approx, Flags uint32 }
type satClauseAllocatorConfig struct {
	Storage        arena.SmallBytesConfig
	OutOfMemory    func() bool
	PooledOverride bool
}
type satClauseAllocator struct {
	self                                   *satClauseAllocator
	origin                                 uint64
	storage                                *arena.SmallBytes
	ids                                    utilIDGen
	budget                                 *arena.Budget
	large                                  []satLargeClauseEntry
	largeFree                              uint32
	largeBytes, largeDirectory, ownerBytes uint64
	temporary, closed                      bool
}

func NewSATClauseAllocator(c satClauseAllocatorConfig) (*satClauseAllocator, error) {
	return newSATClauseAllocator(c, false)
}
func newSATClauseAllocator(c satClauseAllocatorConfig, tmp bool) (*satClauseAllocator, error) {
	if tmp || satStorageSourceDebug && !c.PooledOverride {
		c.Storage.DirectOnly = true
	}
	b := c.Storage.Descriptors.Budget
	n := uint64(reflect.TypeOf(satClauseAllocator{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	id, e := satNextOwner()
	if e != nil {
		b.Release(n)
		return nil, e
	}
	store, e := arena.NewSmallBytes(c.Storage)
	if e != nil {
		b.Release(n)
		return nil, e
	}
	a := &satClauseAllocator{origin: id, storage: store, budget: b, ids: newBudgetedUtilIDGen(0, b, c.OutOfMemory), ownerBytes: n, temporary: tmp}
	a.self = a
	return a, nil
}
func (a *satClauseAllocator) open() error {
	if a == nil || a.self != a {
		return ErrSATOwner
	}
	if a.closed {
		return arena.ErrClosed
	}
	return nil
}
func satClauseObjectBytes(n uint32) (uint64, error) {
	return uint64(reflect.TypeOf(satClauseHotHeader{}).Size()) + 4*uint64(n), nil
}
func satClauseWideExtent(bytes uint64) bool { return bytes > uint64(^uint32(0)) }
func satClauseHostExtent(bytes, max uint64) error {
	if bytes > max {
		return arena.ErrBounds
	}
	return nil
}
func (a *satClauseAllocator) validate(r satStoredClauseRef) error {
	if e := a.open(); e != nil {
		return e
	}
	if r.owner != a.origin {
		return ErrSATOwner
	}
	if r.storageKind == 0 {
		if !a.storage.IsLive(arena.SmallBytesRef{Slot: arena.Slot(r.slot), Generation: arena.Generation(r.generation)}) {
			return arena.ErrInvalidSlot
		}
	} else if r.storageKind == 1 {
		if uint64(r.slot) >= uint64(len(a.large)) || !a.large[r.slot].live || a.large[r.slot].generation != r.generation {
			return arena.ErrInvalidSlot
		}
	} else {
		return arena.ErrInvalidSlot
	}
	return nil
}
func (a *satClauseAllocator) IsLive(r satStoredClauseRef) bool { return a.validate(r) == nil }
func (a *satClauseAllocator) readWord(r satStoredClauseRef, i uint64) (uint32, error) {
	if e := a.validate(r); e != nil {
		return 0, e
	}
	if r.storageKind == 1 {
		if i >= uint64(len(a.large[r.slot].data)) {
			return 0, arena.ErrBounds
		}
		return a.large[r.slot].data[i], nil
	}
	if i > uint64(^uint32(0))/4 {
		return 0, arena.ErrBounds
	}
	var v uint32
	ref := arena.SmallBytesRef{Slot: arena.Slot(r.slot), Generation: arena.Generation(r.generation)}
	for j := uint32(0); j < 4; j++ {
		x, e := a.storage.Read(ref, uint32(i*4)+j)
		if e != nil {
			return 0, e
		}
		v |= uint32(x) << (8 * j)
	}
	return v, nil
}
func (a *satClauseAllocator) writeWord(r satStoredClauseRef, i uint64, v uint32) error {
	if e := a.validate(r); e != nil {
		return e
	}
	if r.storageKind == 1 {
		if i >= uint64(len(a.large[r.slot].data)) {
			return arena.ErrBounds
		}
		a.large[r.slot].data[i] = v
		return nil
	}
	if i > uint64(^uint32(0))/4 {
		return arena.ErrBounds
	}
	ref := arena.SmallBytesRef{Slot: arena.Slot(r.slot), Generation: arena.Generation(r.generation)}
	for j := uint32(0); j < 4; j++ {
		if e := a.storage.Write(ref, uint32(i*4)+j, byte(v>>(8*j))); e != nil {
			return e
		}
	}
	return nil
}
func (a *satClauseAllocator) allocate(n uint32) (satStoredClauseRef, error) {
	bytes, _ := satClauseObjectBytes(n)
	if e := satClauseHostExtent(bytes, uint64(^uint(0)>>1)); e != nil {
		return satStoredClauseRef{}, e
	}
	if satClauseWideExtent(bytes) {
		return a.allocateLarge(bytes)
	}
	r, e := a.storage.Allocate(uint32(bytes))
	if e != nil {
		return satStoredClauseRef{}, e
	}
	return satStoredClauseRef{a.origin, uint32(r.Slot), uint32(r.Generation), 0}, nil
}
func (a *satClauseAllocator) construct(r satStoredClauseRef, id, n uint32, lits []satLiteral, learned bool) error {
	return a.constructSpan(r, id, n, satSolverLiteralSpan{literals: lits}, learned)
}
func (a *satClauseAllocator) constructSpan(r satStoredClauseRef, id, n uint32, span satSolverLiteralSpan, learned bool) error {
	flags := uint32(0)
	if learned {
		flags = 4
	}
	for i, v := range [5]uint32{id, n, n, 0, flags} {
		if e := a.writeWord(r, uint64(i), v); e != nil {
			return e
		}
	}
	if span.runtime == nil && span.words == nil && uint64(len(span.literals)) < uint64(n) {
		return arena.ErrBounds
	}
	for i := uint32(0); i < n; i++ {
		lit, e := span.literal(i)
		if e != nil {
			return e
		}
		if e := a.writeWord(r, uint64(i)+5, uint32(lit)); e != nil {
			return e
		}
	}
	if e := a.MarkStrengthened(r); e != nil {
		return e
	}
	if satStorageSourceDebug {
		if _, e := a.CheckApprox(r); e != nil {
			return e
		}
		if n <= 1 {
			return ErrSATAssertion
		}
	}
	return nil
}
func (a *satClauseAllocator) MkClause(lits []satLiteral, learned bool) (satStoredClauseRef, error) {
	if uint64(len(lits)) > uint64(^uint32(0)) {
		return satStoredClauseRef{}, arena.ErrBounds
	}
	return a.MkClauseN(uint32(len(lits)), lits, learned)
}

// A constructor error after storage/ID publication returns its owned ref so the
// caller can inspect/reclaim the completed prefix; no hidden rollback of IDs.
func (a *satClauseAllocator) MkClauseN(n uint32, lits []satLiteral, learned bool) (satStoredClauseRef, error) {
	return a.mkClauseSpan(n, satSolverLiteralSpan{literals: lits}, learned)
}
func (a *satClauseAllocator) mkClauseSpan(n uint32, span satSolverLiteralSpan, learned bool) (satStoredClauseRef, error) {
	if e := a.open(); e != nil {
		return satStoredClauseRef{}, e
	}
	r, e := a.allocate(n)
	if e != nil {
		return r, e
	}
	id := uint32(^uint32(0))
	if !a.temporary {
		id = a.ids.mk()
	}
	return r, a.constructSpan(r, id, n, span, learned)
}
func (a *satClauseAllocator) release(r satStoredClauseRef) error {
	if e := a.validate(r); e != nil {
		return e
	}
	if r.storageKind == 1 {
		return a.releaseLarge(r)
	}
	cap, e := a.Capacity(r)
	if e != nil {
		return e
	}
	n, _ := satClauseObjectBytes(cap)
	return a.storage.Deallocate(uint32(n), arena.SmallBytesRef{Slot: arena.Slot(r.slot), Generation: arena.Generation(r.generation)})
}
func (a *satClauseAllocator) Delete(r satStoredClauseRef) error {
	if e := a.validate(r); e != nil {
		return e
	}
	if !a.temporary {
		id, e := a.ID(r)
		if e != nil {
			return e
		}
		if e = a.ids.recycle(id); e != nil {
			return e
		}
	}
	return a.release(r)
}
func (a *satClauseAllocator) Offset(r satStoredClauseRef) (satClauseOffset, error) {
	if e := a.validate(r); e != nil {
		return satClauseOffset{}, e
	}
	return satClauseOffset(r), nil
}
func (a *satClauseAllocator) GetClause(o satClauseOffset) (satStoredClauseRef, error) {
	r := satStoredClauseRef(o)
	return r, a.validate(r)
}
func (a *satClauseAllocator) ID(r satStoredClauseRef) (uint32, error)       { return a.readWord(r, 0) }
func (a *satClauseAllocator) Size(r satStoredClauseRef) (uint32, error)     { return a.readWord(r, 1) }
func (a *satClauseAllocator) Capacity(r satStoredClauseRef) (uint32, error) { return a.readWord(r, 2) }
func (a *satClauseAllocator) Approx(r satStoredClauseRef) (satApproxSet, error) {
	n, e := a.readWord(r, 3)
	return satApproxSet(n), e
}
func (a *satClauseAllocator) Literal(r satStoredClauseRef, i uint32) (satLiteral, error) {
	n, e := a.Size(r)
	if e != nil {
		return 0, e
	}
	if satStorageSourceDebug && i >= n {
		return 0, ErrSATAssertion
	}
	capacity, e := a.Capacity(r)
	if e != nil {
		return 0, e
	}
	if i >= capacity {
		return 0, arena.ErrBounds
	}
	v, e := a.readWord(r, 5+uint64(i))
	return satLiteral(v), e
}
func (a *satClauseAllocator) SetLiteral(r satStoredClauseRef, i uint32, l satLiteral) error {
	n, e := a.Size(r)
	if e != nil {
		return e
	}
	if satStorageSourceDebug && i >= n {
		return ErrSATAssertion
	}
	capacity, e := a.Capacity(r)
	if e != nil {
		return e
	}
	if i >= capacity {
		return arena.ErrBounds
	}
	return a.writeWord(r, 5+uint64(i), uint32(l))
}
func (a *satClauseAllocator) flag(r satStoredClauseRef, mask uint32) (bool, error) {
	v, e := a.readWord(r, 4)
	return v&mask != 0, e
}
func (a *satClauseAllocator) setFlag(r satStoredClauseRef, mask uint32, v bool) error {
	old, e := a.readWord(r, 4)
	if e != nil {
		return e
	}
	old &^= mask
	if v {
		old |= mask
	}
	return a.writeWord(r, 4, old)
}
func (a *satClauseAllocator) Learned(r satStoredClauseRef) (bool, error)       { return a.flag(r, 4) }
func (a *satClauseAllocator) Strengthened(r satStoredClauseRef) (bool, error)  { return a.flag(r, 1) }
func (a *satClauseAllocator) Removed(r satStoredClauseRef) (bool, error)       { return a.flag(r, 2) }
func (a *satClauseAllocator) Used(r satStoredClauseRef) (bool, error)          { return a.flag(r, 8) }
func (a *satClauseAllocator) Frozen(r satStoredClauseRef) (bool, error)        { return a.flag(r, 16) }
func (a *satClauseAllocator) OnReinitStack(r satStoredClauseRef) (bool, error) { return a.flag(r, 32) }
func (a *satClauseAllocator) SetRemoved(r satStoredClauseRef, b bool) error {
	return a.setFlag(r, 2, b)
}
func (a *satClauseAllocator) MarkUsed(r satStoredClauseRef) error   { return a.setFlag(r, 8, true) }
func (a *satClauseAllocator) UnmarkUsed(r satStoredClauseRef) error { return a.setFlag(r, 8, false) }
func (a *satClauseAllocator) SetReinitStack(r satStoredClauseRef, b bool) error {
	return a.setFlag(r, 32, b)
}
func (a *satClauseAllocator) UnmarkStrengthened(r satStoredClauseRef) error {
	return a.setFlag(r, 1, false)
}
func (a *satClauseAllocator) UnsetLearned(r satStoredClauseRef) error {
	v, e := a.Learned(r)
	if e != nil {
		return e
	}
	if satStorageSourceDebug && !v {
		return ErrSATAssertion
	}
	return a.setFlag(r, 4, false)
}
func (a *satClauseAllocator) Freeze(r satStoredClauseRef) error {
	l, e := a.Learned(r)
	if e != nil {
		return e
	}
	f, e := a.Frozen(r)
	if e != nil {
		return e
	}
	if satStorageSourceDebug && (!l || f) {
		return ErrSATAssertion
	}
	return a.setFlag(r, 16, true)
}
func (a *satClauseAllocator) Unfreeze(r satStoredClauseRef) error {
	l, e := a.Learned(r)
	if e != nil {
		return e
	}
	f, e := a.Frozen(r)
	if e != nil {
		return e
	}
	if satStorageSourceDebug && (!l || !f) {
		return ErrSATAssertion
	}
	return a.setFlag(r, 16, false)
}
func SATApproxLiterals(ls []satLiteral) satApproxSet {
	var m satApproxSet
	for _, l := range ls {
		m.insert(uint32(l.variable()))
	}
	return m
}
func (a *satClauseAllocator) UpdateApprox(r satStoredClauseRef) error {
	n, e := a.Size(r)
	if e != nil {
		return e
	}
	var m satApproxSet
	for i := uint32(0); i < n; i++ {
		l, e := a.Literal(r, i)
		if e != nil {
			return e
		}
		m.insert(uint32(l.variable()))
	}
	return a.writeWord(r, 3, uint32(m))
}
func (a *satClauseAllocator) MarkStrengthened(r satStoredClauseRef) error {
	if e := a.setFlag(r, 1, true); e != nil {
		return e
	}
	return a.UpdateApprox(r)
}
func (a *satClauseAllocator) CheckApprox(r satStoredClauseRef) (bool, error) {
	old, e := a.Approx(r)
	if e != nil {
		return false, e
	}
	if e = a.UpdateApprox(r); e != nil {
		return false, e
	}
	m, e := a.Approx(r)
	if e != nil {
		return false, e
	}
	if satStorageSourceDebug && old != m {
		return false, ErrSATAssertion
	}
	return true, nil
}
func (a *satClauseAllocator) Shrink(r satStoredClauseRef, n uint32) error {
	old, e := a.Size(r)
	if e != nil {
		return e
	}
	if satStorageSourceDebug && n > old {
		return ErrSATAssertion
	}
	if n < old {
		if e = a.writeWord(r, 1, n); e != nil {
			return e
		}
		return a.MarkStrengthened(r)
	}
	return nil
}
func (a *satClauseAllocator) Elim(r satStoredClauseRef, l satLiteral) error {
	n, e := a.Size(r)
	if e != nil {
		return e
	}
	i := uint32(0)
	for ; i < n; i++ {
		v, e := a.Literal(r, i)
		if e != nil {
			return e
		}
		if v == l {
			break
		}
	}
	if satStorageSourceDebug && i == n {
		return ErrSATAssertion
	}
	if n == 0 {
		return arena.ErrBounds
	}
	if e := satClauseElimCopyStart(n, i); e != nil {
		return e
	}
	for j := uint64(i) + 1; j < uint64(n); j++ {
		v, e := a.Literal(r, uint32(j))
		if e != nil {
			return e
		}
		if e = a.SetLiteral(r, uint32(j-1), v); e != nil {
			return e
		}
	}
	if e = a.writeWord(r, 1, n-1); e != nil {
		return e
	}
	return a.MarkStrengthened(r)
}
func (a *satClauseAllocator) ContainsLiteral(r satStoredClauseRef, l satLiteral) (bool, error) {
	n, e := a.Size(r)
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < n; i++ {
		v, e := a.Literal(r, i)
		if e != nil {
			return false, e
		}
		if v == l {
			return true, nil
		}
	}
	return false, nil
}
func (a *satClauseAllocator) ContainsVar(r satStoredClauseRef, v satBoolVar) (bool, error) {
	n, e := a.Size(r)
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < n; i++ {
		l, e := a.Literal(r, i)
		if e != nil {
			return false, e
		}
		if l.variable() == v {
			return true, nil
		}
	}
	return false, nil
}
func (a *satClauseAllocator) SatisfiedBy(r satStoredClauseRef, m satModel) (bool, error) {
	n, e := a.Size(r)
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < n; i++ {
		l, e := a.Literal(r, i)
		if e != nil {
			return false, e
		}
		if uint64(l.variable()) >= uint64(len(m)) {
			return false, arena.ErrBounds
		}
		v := m[l.variable()]
		if l.sign() && v == lFalse || !l.sign() && v == lTrue {
			return true, nil
		}
	}
	return false, nil
}
func (a *satClauseAllocator) lane(r satStoredClauseRef, shift uint32) (uint32, error) {
	v, e := a.readWord(r, 4)
	return v >> shift & 255, e
}
func (a *satClauseAllocator) setLane(r satStoredClauseRef, shift, v uint32) error {
	old, e := a.readWord(r, 4)
	if e != nil {
		return e
	}
	return a.writeWord(r, 4, old&^(255<<shift)|((v&255)<<shift))
}
func (a *satClauseAllocator) IncInactivity(r satStoredClauseRef) error {
	v, e := a.Inactivity(r)
	if e != nil {
		return e
	}
	return a.setLane(r, 6, v+1)
}
func (a *satClauseAllocator) ResetInactivity(r satStoredClauseRef) error      { return a.setLane(r, 6, 0) }
func (a *satClauseAllocator) Inactivity(r satStoredClauseRef) (uint32, error) { return a.lane(r, 6) }
func (a *satClauseAllocator) SetGlue(r satStoredClauseRef, v uint32) error {
	if v > 255 {
		v = 255
	}
	return a.setLane(r, 14, v)
}
func (a *satClauseAllocator) Glue(r satStoredClauseRef) (uint32, error) { return a.lane(r, 14) }
func (a *satClauseAllocator) SetPSM(r satStoredClauseRef, v uint32) error {
	if v > 255 {
		v = 255
	}
	return a.setLane(r, 22, v)
}
func (a *satClauseAllocator) PSM(r satStoredClauseRef) (uint32, error) { return a.lane(r, 22) }
func (a *satClauseAllocator) AppendLiterals(r satStoredClauseRef, dst []satLiteral) ([]satLiteral, error) {
	n, e := a.Size(r)
	if e != nil {
		return dst, e
	}
	for i := uint32(0); i < n; i++ {
		l, e := a.Literal(r, i)
		if e != nil {
			return dst, e
		}
		dst = append(dst, l)
	}
	return dst, nil
}
func satLiteralInto(out *NumericTextStream, l satLiteral) error {
	sign := ""
	if l.sign() {
		sign = "-"
	}
	if e := out.AppendCString(sign); e != nil {
		return e
	}
	return out.AppendUint32(uint32(l.variable()))
}
func (a *satClauseAllocator) DisplayInto(r satStoredClauseRef, out *NumericTextStream) error {
	n, e := a.Size(r)
	if e != nil {
		return e
	}
	if e = out.AppendCString("("); e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		if i != 0 {
			if e = out.AppendCString(" "); e != nil {
				return e
			}
		}
		l, e := a.Literal(r, i)
		if e != nil {
			return e
		}
		if e = satLiteralInto(out, l); e != nil {
			return e
		}
	}
	if e = out.AppendCString(")"); e != nil {
		return e
	}
	for _, flag := range []struct {
		mask uint32
		text string
	}{{2, "x"}, {1, "+"}, {4, "*"}} {
		v, e := a.flag(r, flag.mask)
		if e != nil {
			return e
		}
		if v {
			if e = out.AppendCString(flag.text); e != nil {
				return e
			}
		}
	}
	return nil
}
func (a *satClauseAllocator) Close() error {
	if a == nil || a.self != a {
		return ErrSATOwner
	}
	if a.closed {
		return nil
	}
	for i := range a.large {
		if a.large[i].live {
			a.budget.Release(uint64(len(a.large[i].data)) * 4)
			a.large[i].data = nil
		}
	}
	a.budget.Release(a.largeDirectory + a.ownerBytes)
	a.large = nil
	a.largeDirectory = 0
	a.largeBytes = 0
	a.ownerBytes = 0
	a.ids.cleanup(0)
	a.storage.Close()
	a.closed = true
	return nil
}

type satTmpClause struct {
	self         *satTmpClause
	owner        *satClauseAllocator
	ref, pending satStoredClauseRef
	budget       *arena.Budget
	ownerBytes   uint64
	closed       bool
}

func NewSATTmpClause(b *arena.Budget) (*satTmpClause, error) {
	n := uint64(reflect.TypeOf(satTmpClause{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	a, e := newSATClauseAllocator(satClauseAllocatorConfig{Storage: arena.SmallBytesConfig{Descriptors: arena.Config{Budget: b}}}, true)
	if e != nil {
		b.Release(n)
		return nil, e
	}
	t := &satTmpClause{owner: a, budget: b, ownerBytes: n}
	t.self = t
	return t, nil
}
func (t *satTmpClause) open() error {
	if t == nil || t.self != t {
		return ErrSATOwner
	}
	if t.closed {
		return arena.ErrClosed
	}
	if t.pending != (satStoredClauseRef{}) {
		return ErrSATAssertion
	}
	return nil
}
func (t *satTmpClause) Get() satStoredClauseRef { return t.ref }
func (t *satTmpClause) Set(ls []satLiteral, learned bool) error {
	if uint64(len(ls)) > uint64(^uint32(0)) {
		return arena.ErrBounds
	}
	return t.SetN(uint32(len(ls)), ls, learned)
}
func (t *satTmpClause) SetN(n uint32, ls []satLiteral, learned bool) error {
	if e := t.open(); e != nil {
		return e
	}
	if t.ref != (satStoredClauseRef{}) {
		cap, e := t.owner.Capacity(t.ref)
		if e != nil {
			return e
		}
		if cap < n {
			if e = t.owner.release(t.ref); e != nil {
				return e
			}
			t.ref = satStoredClauseRef{}
		}
	}
	if t.ref == (satStoredClauseRef{}) {
		r, e := t.owner.MkClauseN(n, ls, learned)
		if e != nil {
			t.pending = r
			return e
		}
		t.ref = r
		return nil
	}
	id, e := t.owner.ID(t.ref)
	if e != nil {
		return e
	}
	if satStorageSourceDebug && id != ^uint32(0) {
		return ErrSATAssertion
	}
	if e = t.owner.writeWord(t.ref, 1, n); e != nil {
		return e
	}
	if e = t.owner.setFlag(t.ref, 4, learned); e != nil {
		return e
	}
	if uint64(len(ls)) < uint64(n) {
		return arena.ErrBounds
	}
	for i := uint32(0); i < n; i++ {
		if e = t.owner.SetLiteral(t.ref, i, ls[i]); e != nil {
			return e
		}
	}
	return nil
}
func (t *satTmpClause) SetBinary(a, b satLiteral, l bool) error {
	v := [2]satLiteral{a, b}
	return t.Set(v[:], l)
}
func (t *satTmpClause) SetBinClause(b satBinClause) error {
	return t.SetBinary(b.Literal1(), b.Literal2(), b.Learned())
}
func (t *satTmpClause) Close() error {
	if t == nil || t.self != t {
		return ErrSATOwner
	}
	if t.closed {
		return nil
	}
	t.owner.Close()
	t.budget.Release(t.ownerBytes)
	t.ownerBytes = 0
	t.ref = satStoredClauseRef{}
	t.pending = satStoredClauseRef{}
	t.closed = true
	return nil
}

type satBinClause struct{ val1, val2 uint32 }

func newSATBinClause(a, b satLiteral, l bool) satBinClause {
	n := uint32(0)
	if l {
		n = 1
	}
	return satBinClause{uint32(a), (uint32(b) << 1) + n}
}
func (b satBinClause) Literal1() satLiteral { return satLiteral(b.val1) }
func (b satBinClause) Literal2() satLiteral { return satLiteral(b.val2 >> 1) }
func (b satBinClause) Learned() bool        { return b.val2&1 == 1 }
func (b satBinClause) Equal(c satBinClause) bool {
	return b.val1 == c.val1 && b.val2 == c.val2 || b.val1 == c.val2 && b.val2 == c.val1
}

type satClauseWrapper struct {
	ref           satStoredClauseRef
	first, second satLiteral
}

func newSATClauseWrapper(r satStoredClauseRef) satClauseWrapper {
	return satClauseWrapper{ref: r, second: satDefaultLiteral()}
}
func newSATBinaryWrapper(a, b satLiteral) satClauseWrapper {
	return satClauseWrapper{first: a, second: b}
}
func (w satClauseWrapper) Binary() bool { return w.second != satDefaultLiteral() }
func (w satClauseWrapper) Size(a *satClauseAllocator) (uint32, error) {
	if w.Binary() {
		return 2, nil
	}
	if w.ref.owner == 0 {
		return 0, ErrSATKind
	}
	return a.Size(w.ref)
}
func (w satClauseWrapper) Literal(a *satClauseAllocator, i uint32) (satLiteral, error) {
	if w.Binary() {
		if satStorageSourceDebug && i > 1 {
			return 0, ErrSATAssertion
		}
		if i == 0 {
			return w.first, nil
		}
		return w.second, nil
	}
	if w.ref.owner == 0 {
		return 0, ErrSATKind
	}
	return a.Literal(w.ref, i)
}
func (w satClauseWrapper) ClauseRef() (satStoredClauseRef, error) {
	if w.Binary() {
		return satStoredClauseRef{}, ErrSATKind
	}
	if w.ref.owner == 0 {
		return satStoredClauseRef{}, ErrSATKind
	}
	return w.ref, nil
}
func (w satClauseWrapper) ContainsLiteral(a *satClauseAllocator, l satLiteral) (bool, error) {
	n, e := w.Size(a)
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < n; i++ {
		v, e := w.Literal(a, i)
		if e != nil {
			return false, e
		}
		if v == l {
			return true, nil
		}
	}
	return false, nil
}
func (w satClauseWrapper) ContainsVar(a *satClauseAllocator, v satBoolVar) (bool, error) {
	n, e := w.Size(a)
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < n; i++ {
		l, e := w.Literal(a, i)
		if e != nil {
			return false, e
		}
		if l.variable() == v {
			return true, nil
		}
	}
	return false, nil
}
func (w satClauseWrapper) DisplayInto(a *satClauseAllocator, out *NumericTextStream) error {
	n, e := w.Size(a)
	if e != nil {
		return e
	}
	if e = out.AppendCString("("); e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		if i > 0 {
			if e = out.AppendCString(" "); e != nil {
				return e
			}
		}
		l, e := w.Literal(a, i)
		if e != nil {
			return e
		}
		if e = satLiteralInto(out, l); e != nil {
			return e
		}
	}
	return out.AppendCString(")")
}

// Elim increments the uint32 search index before entering its copy loop.
func satClauseElimCopyStart(size, index uint32) error {
	if size == ^uint32(0) && index == size {
		return arena.ErrBounds
	}
	return nil
}

// Iterators are caller-owned values; only scalar locators are retained
// by storage. No mutable slice aliases escape.
type satClauseLiteralIterator struct {
	owner      *satClauseAllocator
	ref        satStoredClauseRef
	index, end uint32
}

func (a *satClauseAllocator) LiteralIter(r satStoredClauseRef) (satClauseLiteralIterator, error) {
	n, e := a.Size(r)
	if e != nil {
		return satClauseLiteralIterator{}, e
	}
	return satClauseLiteralIterator{owner: a, ref: r, end: n}, nil
}
func (i *satClauseLiteralIterator) Next() (satLiteral, bool, error) {
	if i == nil {
		return 0, false, ErrSATOwner
	}
	if e := i.owner.validate(i.ref); e != nil {
		return 0, false, e
	}
	if i.index >= i.end {
		return 0, false, nil
	}
	v, e := i.owner.readWord(i.ref, 5+uint64(i.index))
	if e != nil {
		return 0, false, e
	}
	i.index++
	return satLiteral(v), true, nil
}
func SATApproxLiteralsN(n uint32, ls []satLiteral) (satApproxSet, error) {
	var m satApproxSet
	for i := uint32(0); i < n; i++ {
		if uint64(i) >= uint64(len(ls)) {
			return m, arena.ErrBounds
		}
		m.insert(uint32(ls[i].variable()))
	}
	return m, nil
}

type satClauseAllocatorStats struct {
	OwnerBytes, IDFreeBytes, WideDirectoryBytes, WidePayloadBytes uint64
	SmallBytes                                                    arena.SmallBytesStats
}

func (a *satClauseAllocator) Stats() satClauseAllocatorStats {
	if a == nil {
		return satClauseAllocatorStats{}
	}
	return satClauseAllocatorStats{a.ownerBytes, uint64(cap(a.ids.free)) * 4, a.largeDirectory, a.largeBytes, a.storage.Stats()}
}
func (w satWatched) NotEqual(v satWatched) bool { return !w.Equal(v) }
