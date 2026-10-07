// Translation of src/sat/sat_model_converter.h/.cpp, Leonardo de Moura.
package sat

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
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
	copying, constructed, closed                        bool
}

// Cold context is a charged Go owning adapter. The source store constructor
// still executes only at its declared model-converter member stage.
func newSATColdModelConverter(b *arena.Budget) (*satModelConverter, error) {
	if b == nil {
		return nil, ErrSATOwner
	}
	header := uint64(reflect.TypeOf(satModelConverter{}).Size())
	if e := b.Reserve(header); e != nil {
		return nil, e
	}
	m := &satModelConverter{budget: b, epoch: 1, ownerBytes: header}
	m.self = m
	return m, nil
}
func (m *satModelConverter) constructionBorrow() error {
	if m == nil || m.self != m {
		return ErrSATOwner
	}
	if m.closed {
		return arena.ErrClosed
	}
	return nil
}
func (m *satModelConverter) initializeSourceStore(c arena.VectorConfig) error {
	if e := m.constructionBorrow(); e != nil {
		return e
	}
	if m.constructed || m.store != nil || c.Descriptors.Budget != m.budget {
		return ErrSATOwner
	}
	store, e := newSATSolverLiteralStore(c)
	if e != nil {
		return e
	}
	m.store = store
	id, e := satNextOwner()
	if e != nil {
		return e
	}
	m.origin = id
	m.constructed = true
	return nil
}
func newSATModelConverter(c arena.VectorConfig) (*satModelConverter, error) {
	m, e := newSATColdModelConverter(c.Descriptors.Budget)
	if e != nil {
		return nil, e
	}
	if e = m.initializeSourceStore(c); e != nil {
		if cleanup := m.Close(); cleanup != nil {
			return m, cleanup
		}
		return nil, e
	}
	return m, nil
}
func (m *satModelConverter) open() error {
	if e := m.constructionBorrow(); e != nil {
		return e
	}
	if !m.constructed {
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
	if (m.pending != nil || m.copying) && r.slot >= m.copyCursor {
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
func (m *satModelConverter) KindAndVar(r satModelConversionEntryRef) (satModelConversionKind, BoolVar, error) {
	x, e := m.entry(r)
	if e != nil {
		return 0, 0, e
	}
	return satModelConversionKind(x.varKind >> 31), BoolVar(x.varKind & 0x7fffffff), nil
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
	if m.copying {
		return ErrSATIncompleteGrowth
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
func (m *satModelConverter) Mk(k satModelConversionKind, v BoolVar) (satModelConversionEntryRef, error) {
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
	if satStorageSourceDebug && (BoolVar(bits&0x7fffffff) != v || satModelConversionKind(bits>>31) != k) {
		return r, ErrSATAssertion
	}
	return r, nil
}
func (m *satModelConverter) InsertLiterals(r satModelConversionEntryRef, lits []Literal) error {
	x, e := m.entry(r)
	if e != nil {
		return e
	}
	v := BoolVar(x.varKind & 0x7fffffff)
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
	return m.store.Push(x.literals, DefaultLiteral())
}
func (m *satModelConverter) InsertBinary(r satModelConversionEntryRef, a, b Literal) error {
	lits := [2]Literal{a, b}
	return m.InsertLiterals(r, lits[:])
}
func (m *satModelConverter) InsertClause(r satModelConversionEntryRef, a *ClauseAllocator, c StoredClauseRef) error {
	x, e := m.entry(r)
	if e != nil {
		return e
	}
	if satStorageSourceDebug {
		ok, e := a.ContainsVar(c, BoolVar(x.varKind&0x7fffffff))
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
	return m.store.Push(x.literals, DefaultLiteral())
}
func (m *satModelConverter) InsertWrapper(r satModelConversionEntryRef, a *ClauseAllocator, c satClauseWrapper) error {
	x, e := m.entry(r)
	if e != nil {
		return e
	}
	if satStorageSourceDebug {
		ok, e := c.ContainsVar(a, BoolVar(x.varKind&0x7fffffff))
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
	return m.store.Push(x.literals, DefaultLiteral())
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
func satConverterModelValue(model Model, l Literal) (util.LBool, error) {
	if uint64(l.variable()) >= uint64(len(model)) {
		return util.LUndef, arena.ErrBounds
	}
	v := model[l.variable()]
	if l.sign() {
		v = v.Negated()
	}
	return v, nil
}
func (m *satModelConverter) Apply(model Model) error {
	return m.applyModelSpan(satSolverModelSpan{slice: model})
}
func (m *satModelConverter) applyModelSpan(model satSolverModelSpan) error {
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
		v := BoolVar(x.varKind & 0x7fffffff)
		if satStorageSourceDebug && satModelConversionKind(x.varKind>>31) == satModelElimVar {
			value, e := model.read(v)
			if e != nil {
				return e
			}
			if value != util.LUndef {
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
			if l == DefaultLiteral() {
				if !sat {
					selected := util.LTrue
					if sign {
						selected = util.LFalse
					}
					if e := model.write(v, selected); e != nil {
						return e
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
			value, e := model.value(l)
			if e != nil {
				return e
			}
			if value == util.LTrue {
				sat = true
			} else if lv != v {
				current, e := model.read(lv)
				if e != nil {
					return e
				}
				if current == util.LUndef {
					selected := util.LTrue
					if l.sign() {
						selected = util.LFalse
					}
					if e := model.write(lv, selected); e != nil {
						return e
					}
					sat = true
				}
			}
		}
		if satStorageSourceDebug {
			if e := m.assertEntryModelSpan(x, model); e != nil {
				return e
			}
		}
	}
	return nil
}

// Apply's DEBUG_CODE asserts at each separator, before the next clause.
// CheckModel deliberately keeps its separate accumulating source traversal.
func (m *satModelConverter) assertEntryModel(x *satModelConversionEntry, model Model) error {
	return m.assertEntryModelSpan(x, satSolverModelSpan{slice: model})
}
func (m *satModelConverter) assertEntryModelSpan(x *satModelConversionEntry, model satSolverModelSpan) error {
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
		if l == DefaultLiteral() {
			if !sat {
				return ErrSATAssertion
			}
			sat = false
			continue
		}
		if sat {
			continue
		}
		v, e := model.value(l)
		if e != nil {
			return e
		}
		sat = v == util.LTrue
	}
	return nil
}
func (m *satModelConverter) checkEntryModel(x *satModelConversionEntry, model Model) (bool, error) {
	return m.checkEntryModelSpan(x, satSolverModelSpan{slice: model})
}
func (m *satModelConverter) checkEntryModelSpan(x *satModelConversionEntry, model satSolverModelSpan) (bool, error) {
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
		if l == DefaultLiteral() {
			if !sat {
				ok = false
			}
			sat = false
			continue
		}
		if sat {
			continue
		}
		v, e := model.value(l)
		if e != nil {
			return false, e
		}
		sat = v == util.LTrue
	}
	return ok, nil
}
func (m *satModelConverter) CheckModel(model Model) (bool, error) {
	return m.checkModelSpan(satSolverModelSpan{slice: model})
}
func (m *satModelConverter) checkModelSpan(model satSolverModelSpan) (bool, error) {
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
		good, e := m.checkEntryModelSpan(x, model)
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
				if uint32(l.variable()) == v || l != DefaultLiteral() && uint32(l.variable()) >= numVars {
					return false, ErrSATAssertion
				}
			}
		}
	}
	return true, nil
}

// Caller-owned append buffer is the explicit set-output boundary. Preserve
// source collect_vars insertion order and duplicate suppression, not literals.
func (m *satModelConverter) AppendVars(dst []BoolVar) ([]BoolVar, error) {
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
		v := BoolVar(x.varKind & 0x7fffffff)
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
			if l != DefaultLiteral() && uint32(l.variable()) > result {
				result = uint32(l.variable())
			}
		}
	}
	return result, nil
}
func (m *satModelConverter) DisplayInto(out *util.NumericTextStream) error {
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
			} else if l != DefaultLiteral() {
				if e = out.AppendByte(' '); e != nil {
					return e
				}
			}
			if l == DefaultLiteral() {
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
		if (m.pending != nil || m.copying) && uint32(i) >= m.copyCursor {
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
	m.copying = false
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
	if m.store != nil {
		if e := m.store.Close(); e != nil {
			return e
		}
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

// implicitCopyCore is source vector<entry>::copy_core, distinct from Copy's
// captured-end append. SIZE/CAPACITY publish before ascending constructors.
// copying/copyCursor tracks the actually constructed prefix for safe Go
// cleanup after source-unsafe interrupted construction, never atomic rollback.
func (m *satModelConverter) implicitCopyCore(src *satModelConverter) error {
	// Source nontrivial expansion publishes SIZE but does not write CAPACITY
	// until construction ends. Reading that unfinished header is undefined.
	if src.pending != nil {
		return ErrSATIncompleteGrowth
	}
	capacity := src.sourceCapacity
	if capacity == 0 {
		return nil
	}
	stride := uint64(reflect.TypeOf(satModelConversionEntry{}).Size())
	if uint64(capacity) > uint64(^uint(0)>>1)/stride {
		return arena.ErrBounds
	}
	bytes := uint64(capacity) * stride
	if e := m.budget.Reserve(bytes); e != nil {
		return e
	}
	m.entries = make([]satModelConversionEntry, len(src.entries), int(capacity))
	m.entryBytes = bytes
	m.sourceCapacity = capacity
	m.copying = true
	m.copyCursor = 0
	for i := uint32(0); i < uint32(len(src.entries)); i++ {
		r, e := src.EntryAt(i)
		if e != nil {
			return e
		}
		x, e := src.entry(r)
		if e != nil {
			return e
		}
		m.entries[i].varKind = x.varKind
		list, e := m.store.CopyList(src.store, x.literals)
		if e != nil {
			return e
		}
		m.entries[i].literals = list
		m.copyCursor++
	}
	m.copying = false
	m.copyCursor = 0
	return nil
}

// Clone represents the source implicit class copy constructor. A user-declared
// destructor suppresses implicit move, so source rvalue construction also copies.
func (m *satModelConverter) Clone(c arena.VectorConfig) (*satModelConverter, error) {
	if e := m.open(); e != nil {
		return nil, e
	}
	out, e := newSATModelConverter(c)
	if e != nil {
		return nil, e
	}
	if e = out.implicitCopyCore(m); e != nil {
		_ = out.Close()
		return nil, e
	}
	return out, nil
}

// Assign is the implicit class copy assignment: destroy destination, then
// source exact-capacity copy, retaining actual completed prefixes on failure.
// Assigning an rvalue follows this same source copy route, not an invented move.
func (m *satModelConverter) Assign(src *satModelConverter) error {
	if e := m.open(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	if src == m {
		return nil
	}
	if m.epoch == ^uint64(0) {
		return arena.ErrSlotLimit
	}
	if e := m.clearStorage(); e != nil {
		return e
	}
	m.epoch++
	return m.implicitCopyCore(src)
}
