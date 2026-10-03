// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
// Translation of src/sat/sat_model_converter.h/.cpp, Leonardo de Moura.
package g3

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/golang/arena"
)

var ErrSATIncompleteGrowth = errors.New("sat: access to unconstructed nested vector entry")

type satModelConversionKind uint32

const (
	satModelElimVar satModelConversionKind = iota
	satModelBlockLit
)

type satModelConversionEntry struct {
	varKind  uint32 // source unsigned var:31, kind:1.
	literals satSolverLiteralListRef
}
type satModelConversionEntryRef struct {
	owner, epoch uint64
	slot         uint32
}
type satModelConverter struct {
	self                                                *satModelConverter
	budget                                              *arena.Budget
	store                                               *satSolverLiteralStore
	entries, pending                                    []satModelConversionEntry
	origin, epoch, ownerBytes, entryBytes, pendingBytes uint64
	copyCursor, sourceCapacity, pendingCapacity         uint32
	closed                                              bool
}

func newSATModelConverter(c arena.VectorConfig) (*satModelConverter, error) {
	b := c.Descriptors.Budget
	header := uint64(reflect.TypeOf(satModelConverter{}).Size())
	if e := b.Reserve(header); e != nil {
		return nil, e
	}
	s, e := newSATSolverLiteralStore(c)
	if e != nil {
		b.Release(header)
		return nil, e
	}
	id, e := satNextOwner()
	if e != nil {
		_ = s.Close()
		b.Release(header)
		return nil, e
	}
	m := &satModelConverter{budget: b, store: s, origin: id, epoch: 1, ownerBytes: header}
	m.self = m
	return m, nil
}
func (m *satModelConverter) open() error {
	if m == nil || m.self != m {
		return ErrSATOwner
	}
	if m.closed {
		return arena.ErrClosed
	}
	return nil
}
func (m *satModelConverter) entry(r satModelConversionEntryRef) (*satModelConversionEntry, error) {
	if e := m.open(); e != nil {
		return nil, e
	}
	if r.owner != m.origin {
		return nil, ErrSATOwner
	}
	if r.epoch != m.epoch || uint64(r.slot) >= uint64(len(m.entries)) {
		return nil, arena.ErrInvalidSlot
	}
	if m.pending != nil && r.slot >= m.copyCursor {
		return nil, ErrSATIncompleteGrowth
	}
	return &m.entries[r.slot], nil
}
func (m *satModelConverter) EntryAt(i uint32) (satModelConversionEntryRef, error) {
	if e := m.open(); e != nil {
		return satModelConversionEntryRef{}, e
	}
	if uint64(i) >= uint64(len(m.entries)) {
		return satModelConversionEntryRef{}, arena.ErrBounds
	}
	return satModelConversionEntryRef{m.origin, m.epoch, i}, nil
}
func (m *satModelConverter) KindAndVar(r satModelConversionEntryRef) (satModelConversionKind, satBoolVar, error) {
	x, e := m.entry(r)
	if e != nil {
		return 0, 0, e
	}
	return satModelConversionKind(x.varKind >> 31), satBoolVar(x.varKind & 0x7fffffff), nil
}
func (m *satModelConverter) Size() uint32 {
	if m == nil || m.closed {
		return 0
	}
	return uint32(len(m.entries))
}
func (m *satModelConverter) Capacity() uint32 {
	if m == nil || m.closed {
		return 0
	}
	return m.sourceCapacity
}
func (m *satModelConverter) Empty() bool { return m.Size() == 0 }

// Source vector<entry> reallocates nontrivial entries. entry has an explicit
// copy ctor and no move ctor: std::move selects its COPY constructor. New
// outer pointer/size publish before ascending deep copies and old destructors.
// The charged pending directory retains source-unsafe interrupted ownership;
// retry resumes unfinished copies, an explicit Go continuation, not a claim
// that native C++ recovers from its partially constructed vector exception.
func (m *satModelConverter) finishGrowth() error {
	for m.pending != nil && m.copyCursor < uint32(len(m.pending)) {
		i := m.copyCursor
		old := m.pending[i]
		m.entries[i].varKind = old.varKind
		payload, e := m.store.CloneList(old.literals)
		if e != nil {
			return e
		}
		m.entries[i].literals = payload
		if e = m.store.Free(old.literals); e != nil {
			return e
		}
		m.pending[i].literals = satSolverLiteralListRef{}
		m.copyCursor++
	}
	if m.pending != nil {
		m.budget.Release(m.pendingBytes)
		m.pending = nil
		m.pendingBytes = 0
		m.sourceCapacity = m.pendingCapacity
		m.pendingCapacity = 0
		m.copyCursor = 0
	}
	return nil
}
func (m *satModelConverter) grow() error {
	if e := m.open(); e != nil {
		return e
	}
	if e := m.finishGrowth(); e != nil {
		return e
	}
	if len(m.entries) < cap(m.entries) {
		return nil
	}
	if m.epoch == ^uint64(0) {
		return arena.ErrSlotLimit
	}
	stride := uint64(reflect.TypeOf(satModelConversionEntry{}).Size())
	// Selected source entry is 16 bytes (4 bitfield bytes, LP64 vector ptr,
	// padding). Go's scalar locator is independently physically charged.
	n, e := satVectorNextCapacityFor(uint64(m.sourceCapacity), stride, 16)
	if e != nil {
		return e
	}
	bytes := n * stride
	if e = m.budget.Reserve(bytes); e != nil {
		return e
	}
	data := make([]satModelConversionEntry, len(m.entries), int(n))
	old := m.entries
	m.pending = old
	m.pendingBytes = m.entryBytes
	m.entries = data
	m.entryBytes = bytes
	m.pendingCapacity = uint32(n)
	m.copyCursor = 0
	m.epoch++
	if old == nil {
		m.sourceCapacity = uint32(n)
		m.pendingCapacity = 0
		return nil
	}
	return m.finishGrowth()
}
func (m *satModelConverter) Mk(k satModelConversionKind, v satBoolVar) (satModelConversionEntryRef, error) {
	if e := m.open(); e != nil {
		return satModelConversionEntryRef{}, e
	}
	// The source enum has values0..1; out-of-range C++ enum conversion is
	// outside its defined domain. The Go typed boundary rejects it before action.
	if k > satModelBlockLit {
		return satModelConversionEntryRef{}, ErrSATKind
	}
	if e := m.grow(); e != nil {
		return satModelConversionEntryRef{}, e
	}
	payload, e := m.store.NewList()
	if e != nil {
		return satModelConversionEntryRef{}, e
	}
	i := uint32(len(m.entries))
	bits := uint32(v)&0x7fffffff | (uint32(k)&1)<<31
	m.entries = append(m.entries, satModelConversionEntry{bits, payload})
	r := satModelConversionEntryRef{m.origin, m.epoch, i}
	if satStorageSourceDebug && (satBoolVar(bits&0x7fffffff) != v || satModelConversionKind(bits>>31) != k) {
		return r, ErrSATAssertion
	}
	return r, nil
}
func (m *satModelConverter) InsertLiterals(r satModelConversionEntryRef, lits []satLiteral) error {
	x, e := m.entry(r)
	if e != nil {
		return e
	}
	v := satBoolVar(x.varKind & 0x7fffffff)
	if satStorageSourceDebug {
		contains := false
		for _, l := range lits {
			contains = contains || l.variable() == v
		}
		if !contains {
			return ErrSATAssertion
		}
	}
	for _, l := range lits {
		if e = m.store.Push(x.literals, l); e != nil {
			return e
		}
	}
	return m.store.Push(x.literals, satDefaultLiteral())
}
func (m *satModelConverter) InsertBinary(r satModelConversionEntryRef, a, b satLiteral) error {
	lits := [2]satLiteral{a, b}
	return m.InsertLiterals(r, lits[:])
}
func (m *satModelConverter) InsertClause(r satModelConversionEntryRef, a *satClauseAllocator, c satStoredClauseRef) error {
	x, e := m.entry(r)
	if e != nil {
		return e
	}
	if satStorageSourceDebug {
		ok, e := a.ContainsVar(c, satBoolVar(x.varKind&0x7fffffff))
		if e != nil {
			return e
		}
		if !ok {
			return ErrSATAssertion
		}
	}
	n, e := a.Size(c)
	if e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		l, e := a.Literal(c, i)
		if e != nil {
			return e
		}
		if e = m.store.Push(x.literals, l); e != nil {
			return e
		}
	}
	return m.store.Push(x.literals, satDefaultLiteral())
}
func (m *satModelConverter) InsertWrapper(r satModelConversionEntryRef, a *satClauseAllocator, c satClauseWrapper) error {
	x, e := m.entry(r)
	if e != nil {
		return e
	}
	if satStorageSourceDebug {
		ok, e := c.ContainsVar(a, satBoolVar(x.varKind&0x7fffffff))
		if e != nil {
			return e
		}
		if !ok {
			return ErrSATAssertion
		}
	}
	n, e := c.Size(a)
	if e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		l, e := c.Literal(a, i)
		if e != nil {
			return e
		}
		if e = m.store.Push(x.literals, l); e != nil {
			return e
		}
	}
	return m.store.Push(x.literals, satDefaultLiteral())
}
func (m *satModelConverter) Copy(src *satModelConverter) error {
	if e := m.open(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	// Source captures begin/end once; defined self-copy does not traverse
	// newly appended entries. Relocation destroys the referenced source entry
	// before push_back can copy it, so guard at that actual dangling access.
	end, sourceEpoch := uint32(len(src.entries)), src.epoch
	for i := uint32(0); i < end; i++ {
		r, e := src.EntryAt(i)
		if e != nil {
			return e
		}
		x, e := src.entry(r)
		if e != nil {
			return e
		}
		old := *x
		if e = m.grow(); e != nil {
			return e
		}
		if src == m && src.epoch != sourceEpoch {
			return ErrSATSelfAppend
		}
		payload, e := m.store.CopyList(src.store, old.literals)
		if e != nil {
			return e
		}
		m.entries = append(m.entries, satModelConversionEntry{old.varKind, payload})
	}
	return nil
}
func satConverterModelValue(model satModel, l satLiteral) (lbool, error) {
	if uint64(l.variable()) >= uint64(len(model)) {
		return lUndef, arena.ErrBounds
	}
	v := model[l.variable()]
	if l.sign() {
		v = v.negated()
	}
	return v, nil
}
func (m *satModelConverter) Apply(model satModel) error {
	if e := m.open(); e != nil {
		return e
	}
	for i := len(m.entries); i > 0; {
		i--
		r, e := m.EntryAt(uint32(i))
		if e != nil {
			return e
		}
		x, e := m.entry(r)
		if e != nil {
			return e
		}
		v := satBoolVar(x.varKind & 0x7fffffff)
		if satStorageSourceDebug && satModelConversionKind(x.varKind>>31) == satModelElimVar {
			if uint64(v) >= uint64(len(model)) {
				return arena.ErrBounds
			}
			if model[v] != lUndef {
				return ErrSATAssertion
			}
		}
		sat, sign := false, false
		n, e := m.store.Size(x.literals)
		if e != nil {
			return e
		}
		for j := uint32(0); j < n; j++ {
			l, e := m.store.Read(x.literals, j)
			if e != nil {
				return e
			}
			if l == satDefaultLiteral() {
				if !sat {
					if uint64(v) >= uint64(len(model)) {
						return arena.ErrBounds
					}
					model[v] = lTrue
					if sign {
						model[v] = lFalse
					}
					break
				}
				sat = false
				continue
			}
			if sat {
				continue
			}
			lv := l.variable()
			if lv == v {
				sign = l.sign()
			}
			value, e := satConverterModelValue(model, l)
			if e != nil {
				return e
			}
			if value == lTrue {
				sat = true
			} else if lv != v && model[lv] == lUndef {
				model[lv] = lTrue
				if l.sign() {
					model[lv] = lFalse
				}
				sat = true
			}
		}
		if satStorageSourceDebug {
			if e := m.assertEntryModel(x, model); e != nil {
				return e
			}
		}
	}
	return nil
}

// Apply's DEBUG_CODE asserts at each separator, before the next clause.
// CheckModel deliberately keeps its separate accumulating source traversal.
func (m *satModelConverter) assertEntryModel(x *satModelConversionEntry, model satModel) error {
	n, e := m.store.Size(x.literals)
	if e != nil {
		return e
	}
	sat := false
	for j := uint32(0); j < n; j++ {
		l, e := m.store.Read(x.literals, j)
		if e != nil {
			return e
		}
		if l == satDefaultLiteral() {
			if !sat {
				return ErrSATAssertion
			}
			sat = false
			continue
		}
		if sat {
			continue
		}
		v, e := satConverterModelValue(model, l)
		if e != nil {
			return e
		}
		sat = v == lTrue
	}
	return nil
}
func (m *satModelConverter) checkEntryModel(x *satModelConversionEntry, model satModel) (bool, error) {
	n, e := m.store.Size(x.literals)
	if e != nil {
		return false, e
	}
	ok, sat := true, false
	for j := uint32(0); j < n; j++ {
		l, e := m.store.Read(x.literals, j)
		if e != nil {
			return false, e
		}
		if l == satDefaultLiteral() {
			if !sat {
				ok = false
			}
			sat = false
			continue
		}
		if sat {
			continue
		}
		v, e := satConverterModelValue(model, l)
		if e != nil {
			return false, e
		}
		sat = v == lTrue
	}
	return ok, nil
}
func (m *satModelConverter) CheckModel(model satModel) (bool, error) {
	if e := m.open(); e != nil {
		return false, e
	}
	ok := true
	for i := len(m.entries); i > 0; {
		i--
		r, e := m.EntryAt(uint32(i))
		if e != nil {
			return false, e
		}
		x, e := m.entry(r)
		if e != nil {
			return false, e
		}
		good, e := m.checkEntryModel(x, model)
		if e != nil {
			return false, e
		}
		ok = ok && good
	}
	return ok, nil
}
func (m *satModelConverter) CheckInvariant(numVars uint32) (bool, error) {
	if e := m.open(); e != nil {
		return false, e
	}
	for i := uint32(0); i < uint32(len(m.entries)); i++ {
		r, e := m.EntryAt(i)
		if e != nil {
			return false, e
		}
		x, e := m.entry(r)
		if e != nil {
			return false, e
		}
		if !satStorageSourceDebug {
			continue
		}
		v := x.varKind & 0x7fffffff
		if v >= numVars {
			return false, ErrSATAssertion
		}
		if satModelConversionKind(x.varKind>>31) != satModelElimVar {
			continue
		}
		for j := i + 1; j < uint32(len(m.entries)); j++ {
			r, e := m.EntryAt(j)
			if e != nil {
				return false, e
			}
			other, e := m.entry(r)
			if e != nil {
				return false, e
			}
			if other.varKind&0x7fffffff == v {
				return false, ErrSATAssertion
			}
			n, e := m.store.Size(other.literals)
			if e != nil {
				return false, e
			}
			for k := uint32(0); k < n; k++ {
				l, e := m.store.Read(other.literals, k)
				if e != nil {
					return false, e
				}
				if uint32(l.variable()) == v || l != satDefaultLiteral() && uint32(l.variable()) >= numVars {
					return false, ErrSATAssertion
				}
			}
		}
	}
	return true, nil
}

// Caller-owned append buffer is the explicit set-output boundary. Preserve
// source collect_vars insertion order and duplicate suppression, not literals.
func (m *satModelConverter) AppendVars(dst []satBoolVar) ([]satBoolVar, error) {
	if e := m.open(); e != nil {
		return dst, e
	}
	for i := uint32(0); i < uint32(len(m.entries)); i++ {
		r, e := m.EntryAt(i)
		if e != nil {
			return dst, e
		}
		x, e := m.entry(r)
		if e != nil {
			return dst, e
		}
		v := satBoolVar(x.varKind & 0x7fffffff)
		found := false
		for _, old := range dst {
			found = found || old == v
		}
		if !found {
			dst = append(dst, v)
		}
	}
	return dst, nil
}
func (m *satModelConverter) MaxVar(min uint32) (uint32, error) {
	if e := m.open(); e != nil {
		return min, e
	}
	result := min
	for i := uint32(0); i < uint32(len(m.entries)); i++ {
		r, e := m.EntryAt(i)
		if e != nil {
			return result, e
		}
		x, e := m.entry(r)
		if e != nil {
			return result, e
		}
		n, e := m.store.Size(x.literals)
		if e != nil {
			return result, e
		}
		for j := uint32(0); j < n; j++ {
			l, e := m.store.Read(x.literals, j)
			if e != nil {
				return result, e
			}
			if l != satDefaultLiteral() && uint32(l.variable()) > result {
				result = uint32(l.variable())
			}
		}
	}
	return result, nil
}
func (m *satModelConverter) DisplayInto(out *NumericTextStream) error {
	if e := m.open(); e != nil {
		return e
	}
	if e := out.AppendCString("(sat::model-converter"); e != nil {
		return e
	}
	for i := uint32(0); i < uint32(len(m.entries)); i++ {
		r, e := m.EntryAt(i)
		if e != nil {
			return e
		}
		x, e := m.entry(r)
		if e != nil {
			return e
		}
		if e = out.AppendCString("\n  ("); e != nil {
			return e
		}
		label := "elim"
		if x.varKind>>31 != 0 {
			label = "blocked"
		}
		if e = out.AppendCString(label); e != nil {
			return e
		}
		if e = out.AppendByte(' '); e != nil {
			return e
		}
		if e = out.AppendUint32(x.varKind & 0x7fffffff); e != nil {
			return e
		}
		start := true
		n, e := m.store.Size(x.literals)
		if e != nil {
			return e
		}
		for j := uint32(0); j < n; j++ {
			l, e := m.store.Read(x.literals, j)
			if e != nil {
				return e
			}
			if start {
				if e = out.AppendCString("\n    ("); e != nil {
					return e
				}
				start = false
			} else if l != satDefaultLiteral() {
				if e = out.AppendByte(' '); e != nil {
					return e
				}
			}
			if l == satDefaultLiteral() {
				if e = out.AppendByte(')'); e != nil {
					return e
				}
				start = true
				continue
			}
			if e = satLiteralInto(out, l); e != nil {
				return e
			}
		}
		if e = out.AppendByte(')'); e != nil {
			return e
		}
	}
	return out.AppendCString(")\n")
}
func (m *satModelConverter) clearStorage() error {
	for i := range m.entries {
		if m.pending != nil && uint32(i) >= m.copyCursor {
			break
		}
		if e := m.store.Free(m.entries[i].literals); e != nil {
			return e
		}
	}
	if m.pending != nil {
		for i := m.copyCursor; i < uint32(len(m.pending)); i++ {
			if e := m.store.Free(m.pending[i].literals); e != nil {
				return e
			}
		}
	}
	m.entries = nil
	m.pending = nil
	m.budget.Release(m.entryBytes + m.pendingBytes)
	m.entryBytes = 0
	m.pendingBytes = 0
	m.sourceCapacity = 0
	m.pendingCapacity = 0
	m.copyCursor = 0
	return nil
}
func (m *satModelConverter) Reset() error {
	if e := m.open(); e != nil {
		return e
	}
	if m.epoch == ^uint64(0) {
		return arena.ErrSlotLimit
	}
	if e := m.clearStorage(); e != nil {
		return e
	}
	m.epoch++
	return nil
}

func (m *satModelConverter) Close() error {
	if m == nil || m.self != m {
		return ErrSATOwner
	}
	if m.closed {
		return nil
	}
	// Close cleans interrupted roots without advancing or reusing any epoch.
	if e := m.clearStorage(); e != nil {
		return e
	}
	if e := m.store.Close(); e != nil {
		return e
	}
	m.budget.Release(m.ownerBytes)
	m.ownerBytes = 0
	m.closed = true
	return nil
}

// Real source collect_vars: ordered traversal calls sat::uint_set::insert,
// preserving membership reserve/flag/list-allocation failure prefixes.
func (m *satModelConverter) CollectVars(dst *satSolverUintSet) error {
	if e := m.open(); e != nil {
		return e
	}
	for i := uint32(0); i < uint32(len(m.entries)); i++ {
		r, e := m.EntryAt(i)
		if e != nil {
			return e
		}
		x, e := m.entry(r)
		if e != nil {
			return e
		}
		if e = dst.Insert(x.varKind & 0x7fffffff); e != nil {
			return e
		}
	}
	return nil
}
