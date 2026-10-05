// Source: src/util/params.cpp:59-86; src/util/params.h.
// Author: Leonardo de Moura (2011-05-09).
// Licensed under the MIT license.
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
// Concrete translation of util/params.cpp param_descrs and dictionary<info>.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"strings"
	"sync"
)

type ParamKind uint32

const (
	ParamUInt ParamKind = iota
	ParamBool
	ParamDouble
	ParamNumeral
	ParamDecimal
	ParamString
	ParamOptionValue
	ParamKeyword
	ParamSymbol
	ParamSymbolList
	ParamSort
	ParamSortList
	ParamExpr
	ParamExprList
	ParamFuncDecl
	ParamFuncDeclList
	ParamSortedVar
	ParamSortedVarList
	ParamSExpr
	ParamInvalid
)

func (k ParamKind) String() string {
	switch k {
	case ParamUInt:
		return "unsigned int"
	case ParamBool:
		return "bool"
	case ParamDouble:
		return "double"
	case ParamNumeral, ParamDecimal:
		return "rational"
	case ParamString:
		return "string"
	case ParamOptionValue:
		return "optional-value"
	case ParamKeyword:
		return "keyword"
	case ParamSymbol:
		return "symbol"
	case ParamSymbolList:
		return "symbol-list"
	case ParamSort:
		return "sort"
	case ParamSortList:
		return "sort-list"
	case ParamExpr:
		return "expression"
	case ParamExprList:
		return "expression-list"
	case ParamFuncDecl:
		return "declaration"
	case ParamFuncDeclList:
		return "declaration-list"
	case ParamSortedVar:
		return "sorted-variable"
	case ParamSortedVarList:
		return "sorted-variable-list"
	case ParamSExpr:
		return "s-expression"
	default:
		return "unknown"
	}
}

var ErrParamDescrsPrecondition = errors.New("g3: parameter descriptors source precondition")
var paramDescrsCopyMutex sync.Mutex

// OutOfMemory models the source global memory diagnostic for tombstone cleanup.
// Its environment is borrowed, must not reenter, and is not charged to Budget.
type ParamDescrsConfig struct {
	Budget          *arena.Budget
	InitialCapacity uint32
	Text            arena.VectorConfig
	OutOfMemory     func() bool
}
type paramDescrText struct {
	ref    arena.VectorRef
	length uint32
}
type paramDescrInfo struct {
	kind               ParamKind
	descr, def, module paramDescrText
}
type paramDescrEntry struct {
	key         Symbol
	hash, state uint32
	info        paramDescrInfo
}
type ParamDescrsStats struct {
	Live, Deleted, Capacity, Names, NamesCapacity uint32
	OwnerBytes, TableBytes, NamesBytes            uint64
	Text                                          arena.VectorStats
}

// Symbols are borrowed from one manager, which must outlive this owner and may
// not be torn down concurrently. Text is an immutable owned copy of each C
// string; nil and present-empty are distinct. No entry stores a Go pointer.
// The cold mutex protects this owner; a shared external Budget requires caller
// serialization. Owning Go values must not be shallow-copied.
type ParamDescrs struct {
	mu                   sync.Mutex
	self                 *ParamDescrs
	manager              *SymbolManager
	budget               *arena.Budget
	texts                *arena.Words
	entries              []paramDescrEntry
	names                []Symbol
	size, deleted        uint32
	outOfMemory          func() bool
	closed               bool
	constructorReady     bool
	constructorCompleted uint8
	constructorCause     error
	ownerBytes           uint64
}

func pdEntryBytes(n int) uint64 { return uint64(n) * uint64(reflect.TypeOf(paramDescrEntry{}).Size()) }
func pdNameBytes(n int) uint64  { return uint64(n) * uint64(reflect.TypeOf(Symbol{}).Size()) }
func pdGrowth(n int) (int, error) {
	if n == 0 {
		return 2, nil
	}
	if uint64(n) > (uint64(^uint(0)>>1)-1)/3 {
		return 0, arena.ErrBounds
	}
	v := (3*n + 1) / 2
	if uint64(v) > uint64(^uint32(0)) || v <= n {
		return 0, arena.ErrBounds
	}
	return v, nil
}
func newSourceParamDescrs(m *SymbolManager, c ParamDescrsConfig) (*ParamDescrs, error) {
	if m == nil {
		return nil, ErrParamDescrsPrecondition
	}
	if _, e := m.Hash(Symbol{}); e != nil {
		return nil, e
	}
	n := c.InitialCapacity
	if n == 0 {
		n = 8
	}
	if n&(n-1) != 0 {
		return nil, ErrParamDescrsPrecondition
	}
	if uint64(n) > uint64(^uint(0)>>1)/uint64(reflect.TypeOf(paramDescrEntry{}).Size()) {
		return nil, arena.ErrBounds
	}
	if c.Text.Descriptors.Budget != nil && c.Text.Descriptors.Budget != c.Budget {
		return nil, ErrParamDescrsPrecondition
	}
	c.Text.Descriptors.Budget = c.Budget
	if c.Text.PageSize == 0 {
		c.Text.PageSize = 256
	}
	if c.Text.Descriptors.PageSize == 0 {
		c.Text.Descriptors.PageSize = 2
	}
	owner := uint64(reflect.TypeOf(ParamDescrs{}).Size())
	if e := c.Budget.Reserve(owner); e != nil {
		return nil, e
	}
	d := &ParamDescrs{manager: m, budget: c.Budget, outOfMemory: c.OutOfMemory, ownerBytes: owner}
	d.self = d
	// Copied-text storage is actual additional Go cold context, prepared before
	// source m_info construction; it is not a native table allocation ticket.
	text, e := arena.NewWords(c.Text)
	if e != nil {
		d.constructorCause = e
		return d, e
	}
	d.texts = text
	if e = c.Budget.Reserve(pdEntryBytes(int(n))); e != nil {
		d.constructorCause = e
		return d, e
	}
	d.entries = make([]paramDescrEntry, n)
	for i := range d.entries {
		d.entries[i].info.kind = ParamInvalid
	}
	d.constructorCompleted = 1 // actual info dictionary completed
	d.names = nil              // source null names vector default construction cannot fail
	d.constructorCompleted = 2
	d.constructorReady = true
	return d, nil
}
func (d *ParamDescrs) open() error {
	if d == nil || d.self != d {
		return ErrParamDescrsPrecondition
	}
	if d.closed {
		return arena.ErrClosed
	}
	if !d.constructorReady {
		return ErrParamDescrsPrecondition
	}
	_, e := d.manager.Hash(Symbol{})
	return e
}
func (d *ParamDescrs) findHash(k Symbol, h uint32) int {
	mask := uint32(len(d.entries)) - 1
	for n := uint32(0); n <= mask; n++ {
		i := int((h + n) & mask)
		x := d.entries[i]
		if x.state == 0 {
			return -1
		}
		if x.state == 2 && x.hash == h && x.key == k {
			return i
		}
	}
	return -1
}
func (d *ParamDescrs) find(k Symbol) (int, error) {
	h, e := d.manager.Hash(k)
	if e != nil {
		return -1, e
	}
	return d.findHash(k, h), nil
}
func (d *ParamDescrs) rehash(n int) error {
	if uint64(n) > uint64(^uint(0)>>1)/uint64(reflect.TypeOf(paramDescrEntry{}).Size()) {
		return arena.ErrBounds
	}
	if e := d.budget.Reserve(pdEntryBytes(n)); e != nil {
		return e
	}
	next := make([]paramDescrEntry, n)
	mask := uint32(n - 1)
	for _, x := range d.entries {
		if x.state != 2 {
			continue
		}
		i := x.hash & mask
		for next[i].state != 0 {
			i = (i + 1) & mask
		}
		next[i] = x
	}
	d.budget.Release(pdEntryBytes(len(d.entries)))
	d.entries = next
	d.deleted = 0
	return nil
}
func (d *ParamDescrs) growNames() error {
	if uint64(len(d.names)) >= uint64(^uint32(0)) {
		return arena.ErrBounds
	}
	if len(d.names) < cap(d.names) {
		return nil
	}
	n, e := pdGrowth(cap(d.names))
	if e != nil {
		return e
	}
	if uint64(n) > uint64(^uint(0)>>1)/uint64(reflect.TypeOf(Symbol{}).Size()) {
		return arena.ErrBounds
	}
	if e = d.budget.Reserve(pdNameBytes(n)); e != nil {
		return e
	}
	a := make([]Symbol, len(d.names), n)
	copy(a, d.names)
	d.budget.Release(pdNameBytes(cap(d.names)))
	d.names = a
	return nil
}
func (d *ParamDescrs) store(s *string) (paramDescrText, error) {
	if s == nil {
		return paramDescrText{}, nil
	}
	length := cSymbolLength(*s)
	if uint64(length) > uint64(^uint32(0))-4 {
		return paramDescrText{}, arena.ErrBounds
	}
	words := (uint32(length) + 4) / 4
	r, e := d.texts.Allocate(words, words)
	if e != nil {
		return paramDescrText{}, e
	}
	for i := uint32(0); i < words; i++ {
		var v uint32
		for j := uint32(0); j < 4; j++ {
			p := i*4 + j
			if p < uint32(length) {
				v |= uint32((*s)[p]) << (8 * j)
			}
		}
		if e = d.texts.Write(r, i, v); e != nil {
			d.texts.Free(r)
			return paramDescrText{}, e
		}
	}
	return paramDescrText{r, uint32(length)}, nil
}
func (d *ParamDescrs) load(r paramDescrText) (string, bool, error) {
	if r.ref == (arena.VectorRef{}) {
		return "", false, nil
	}
	var s strings.Builder
	s.Grow(int(r.length))
	for i := uint32(0); i < r.length; i++ {
		v, e := d.texts.Read(r.ref, i/4)
		if e != nil {
			return "", false, e
		}
		s.WriteByte(byte(v >> (8 * (i % 4))))
	}
	return s.String(), true, nil
}
func (d *ParamDescrs) drop(i paramDescrInfo) {
	for _, r := range []paramDescrText{i.descr, i.def, i.module} {
		if r.ref != (arena.VectorRef{}) {
			if e := d.texts.Free(r.ref); e != nil {
				panic("g3: descriptor text lifetime violated")
			}
		}
	}
}
func (d *ParamDescrs) prepareInsert(k Symbol, kind ParamKind) (uint32, bool, error) {
	h, e := d.manager.Hash(k)
	if e != nil {
		return 0, false, e
	}
	if k.IsNumerical() {
		return 0, false, ErrParamDescrsPrecondition
	}
	if i := d.findHash(k, h); i >= 0 {
		if d.entries[i].info.kind != kind {
			return 0, false, ErrParamDescrsPrecondition
		}
		return h, true, nil
	}
	if (uint64(d.size)+uint64(d.deleted))*4 > uint64(len(d.entries))*3 {
		if len(d.entries) > int(^uint(0)>>1)/2 {
			return 0, false, arena.ErrBounds
		}
		if e = d.rehash(len(d.entries) * 2); e != nil {
			return 0, false, e
		}
	}
	return h, false, nil
}
func (d *ParamDescrs) publishInsert(k Symbol, h uint32, info paramDescrInfo) error {
	// Preserve source insertion publication before names.push_back. A names
	// allocation failure keeps the descriptor live with the old history length.
	mask := uint32(len(d.entries)) - 1
	deleted := -1
	for n := uint32(0); n <= mask; n++ {
		i := int((h + n) & mask)
		switch d.entries[i].state {
		case 1:
			deleted = i
		case 0:
			if deleted >= 0 {
				i = deleted
				d.deleted--
			}
			d.entries[i] = paramDescrEntry{k, h, 2, info}
			d.size++
			if e := d.growNames(); e != nil {
				return e
			}
			d.names = append(d.names, k)
			return nil
		}
	}
	d.drop(info)
	return ErrParamDescrsPrecondition
}
func (d *ParamDescrs) insert(k Symbol, kind ParamKind, descr, def, module *string) error {
	h, existing, e := d.prepareInsert(k, kind)
	if e != nil || existing {
		return e
	}
	info := paramDescrInfo{kind: kind}
	if info.descr, e = d.store(descr); e != nil {
		return e
	}
	if info.def, e = d.store(def); e != nil {
		d.drop(info)
		return e
	}
	if info.module, e = d.store(module); e != nil {
		d.drop(info)
		return e
	}
	return d.publishInsert(k, h, info)
}
func (d *ParamDescrs) copyText(source *ParamDescrs, old paramDescrText) (paramDescrText, error) {
	if old.ref == (arena.VectorRef{}) {
		return paramDescrText{}, nil
	}
	n, e := source.texts.Len(old.ref)
	if e != nil {
		return paramDescrText{}, e
	}
	r, e := d.texts.Allocate(n, n)
	if e != nil {
		return paramDescrText{}, e
	}
	for i := uint32(0); i < n; i++ {
		v, e := source.texts.Read(old.ref, i)
		if e != nil {
			d.texts.Free(r)
			return paramDescrText{}, e
		}
		if e = d.texts.Write(r, i, v); e != nil {
			d.texts.Free(r)
			return paramDescrText{}, e
		}
	}
	return paramDescrText{r, old.length}, nil
}
func (d *ParamDescrs) insertCopy(source *ParamDescrs, x paramDescrEntry) error {
	h, existing, e := d.prepareInsert(x.key, x.info.kind)
	if e != nil || existing {
		return e
	}
	info := paramDescrInfo{kind: x.info.kind}
	if info.descr, e = d.copyText(source, x.info.descr); e != nil {
		return e
	}
	if info.def, e = d.copyText(source, x.info.def); e != nil {
		d.drop(info)
		return e
	}
	if info.module, e = d.copyText(source, x.info.module); e != nil {
		d.drop(info)
		return e
	}
	return d.publishInsert(x.key, h, info)
}
func (d *ParamDescrs) Insert(k Symbol, kind ParamKind, descr, def, module *string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return e
	}
	return d.insert(k, kind, descr, def, module)
}
func (d *ParamDescrs) InsertName(name string, kind ParamKind, descr, def, module *string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return e
	}
	k, e := d.manager.Intern(name)
	if e != nil {
		return e
	}
	return d.insert(k, kind, descr, def, module)
}
func (d *ParamDescrs) contains(k Symbol) (bool, error) { i, e := d.find(k); return i >= 0, e }
func (d *ParamDescrs) Contains(k Symbol) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return false, e
	}
	return d.contains(k)
}
func (d *ParamDescrs) ContainsName(name string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return false, e
	}
	k, e := d.manager.Intern(name)
	if e != nil {
		return false, e
	}
	return d.contains(k)
}
func (d *ParamDescrs) erase(k Symbol) error {
	i, e := d.find(k)
	if e != nil || i < 0 {
		return e
	}
	d.drop(d.entries[i].info)
	next := (i + 1) & (len(d.entries) - 1)
	if d.entries[next].state == 0 {
		d.entries[i].state = 0
	} else {
		d.entries[i].state = 1
		d.deleted++
	}
	d.size--
	if d.entries[i].state == 1 && d.deleted > d.size && d.deleted > 64 && (d.outOfMemory == nil || !d.outOfMemory()) {
		return d.rehash(len(d.entries))
	}
	return nil
}
func (d *ParamDescrs) Erase(k Symbol) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return e
	}
	return d.erase(k)
}
func (d *ParamDescrs) EraseName(name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return e
	}
	k, e := d.manager.Intern(name)
	if e != nil {
		return e
	}
	return d.erase(k)
}
func (d *ParamDescrs) kind(k Symbol) (ParamKind, error) {
	i, e := d.find(k)
	if e != nil || i < 0 {
		return ParamInvalid, e
	}
	return d.entries[i].info.kind, nil
}
func (d *ParamDescrs) Kind(k Symbol) (ParamKind, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return ParamInvalid, e
	}
	return d.kind(k)
}
func (d *ParamDescrs) KindName(name string) (ParamKind, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return ParamInvalid, e
	}
	k, e := d.manager.Intern(name)
	if e != nil {
		return ParamInvalid, e
	}
	return d.kind(k)
}
func (d *ParamDescrs) text(k Symbol, which uint32) (string, bool, error) {
	i, e := d.find(k)
	if e != nil || i < 0 {
		return "", false, e
	}
	r := d.entries[i].info.descr
	if which == 1 {
		r = d.entries[i].info.def
	}
	if which == 2 {
		r = d.entries[i].info.module
	}
	return d.load(r)
}
func (d *ParamDescrs) getText(k Symbol, which uint32) (string, bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return "", false, e
	}
	return d.text(k, which)
}
func (d *ParamDescrs) getTextName(name string, which uint32) (string, bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return "", false, e
	}
	k, e := d.manager.Intern(name)
	if e != nil {
		return "", false, e
	}
	return d.text(k, which)
}
func (d *ParamDescrs) Description(k Symbol) (string, bool, error)     { return d.getText(k, 0) }
func (d *ParamDescrs) Default(k Symbol) (string, bool, error)         { return d.getText(k, 1) }
func (d *ParamDescrs) Module(k Symbol) (string, bool, error)          { return d.getText(k, 2) }
func (d *ParamDescrs) DescriptionName(n string) (string, bool, error) { return d.getTextName(n, 0) }
func (d *ParamDescrs) DefaultName(n string) (string, bool, error)     { return d.getTextName(n, 1) }
func (d *ParamDescrs) ModuleName(n string) (string, bool, error)      { return d.getTextName(n, 2) }
func (d *ParamDescrs) Size() (uint32, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return 0, e
	}
	return uint32(len(d.names)), nil
}
func (d *ParamDescrs) ParamName(i uint32) (Symbol, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return Symbol{}, e
	}
	if uint64(i) >= uint64(len(d.names)) {
		return Symbol{}, arena.ErrBounds
	}
	return d.names[i], nil
}

// Checked spelling byte access does not allocate a borrowed spelling copy.
func (d *ParamDescrs) symbolByte(k Symbol, index uint32) (byte, error) {
	m := d.manager
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.checkPlain(k); e != nil {
		return 0, e
	}
	if k.value == 0 || k.IsNumerical() {
		return 0, ErrParamDescrsPrecondition
	}
	return m.region.Read(arena.BytesRegionRef{Slot: arena.Slot(k.value), Generation: k.generation}, m.pointerWidth+index)
}
func (d *ParamDescrs) internBytes(b []byte) (Symbol, error) {
	// Go's immutable string bridge is temporary owner storage, distinct from the
	// borrowed manager's subsequently interned spelling graph.
	if len(b) == 0 {
		return d.manager.Intern("")
	}
	amount := uint64(len(b))
	if e := d.budget.Reserve(amount); e != nil {
		return Symbol{}, e
	}
	defer d.budget.Release(amount)
	return d.manager.Intern(string(b))
}
func (d *ParamDescrs) internSymbolRange(k Symbol, start, n uint32) (Symbol, error) {
	if uint64(n) > uint64(^uint(0)>>1) {
		return Symbol{}, arena.ErrBounds
	}
	if n == 0 {
		return d.manager.Intern("")
	}
	if e := d.budget.Reserve(uint64(n)); e != nil {
		return Symbol{}, e
	}
	defer d.budget.Release(uint64(n))
	b := make([]byte, n)
	for i := uint32(0); i < n; i++ {
		v, e := d.symbolByte(k, start+i)
		if e != nil {
			return Symbol{}, e
		}
		b[i] = v
	}
	return d.internBytes(b)
}
func (d *ParamDescrs) internText(r paramDescrText) (Symbol, error) {
	if r.ref == (arena.VectorRef{}) {
		return Symbol{}, nil
	}
	if r.length == 0 {
		return d.manager.Intern("")
	}
	if uint64(r.length) > uint64(^uint(0)>>1) {
		return Symbol{}, arena.ErrBounds
	}
	if e := d.budget.Reserve(uint64(r.length)); e != nil {
		return Symbol{}, e
	}
	defer d.budget.Release(uint64(r.length))
	b := make([]byte, r.length)
	for i := uint32(0); i < r.length; i++ {
		v, e := d.texts.Read(r.ref, i/4)
		if e != nil {
			return Symbol{}, e
		}
		b[i] = byte(v >> (8 * (i % 4)))
	}
	return d.internBytes(b)
}
func (d *ParamDescrs) splitName(name Symbol) (Symbol, Symbol, bool, error) {
	n, e := d.manager.Size(name)
	if e != nil {
		return Symbol{}, Symbol{}, false, e
	}
	dot := uint32(0)
	for dot < n {
		ch, e := d.symbolByte(name, dot)
		if e != nil {
			return Symbol{}, Symbol{}, false, e
		}
		if ch == '.' {
			break
		}
		dot++
	}
	if dot == n {
		return Symbol{}, Symbol{}, false, nil
	}
	// vector(s,data) pushes each prefix character, then push_back(0).
	// Preserve every capacity growth and its old-plus-new allocation peak.
	var prefixBytes []byte
	defer func() { d.budget.Release(uint64(cap(prefixBytes))) }()
	for i := uint32(0); i <= dot; i++ {
		if len(prefixBytes) == cap(prefixBytes) {
			n, e := pdGrowth(cap(prefixBytes))
			if e != nil {
				return Symbol{}, Symbol{}, false, e
			}
			if e = d.budget.Reserve(uint64(n)); e != nil {
				return Symbol{}, Symbol{}, false, e
			}
			next := make([]byte, len(prefixBytes), n)
			copy(next, prefixBytes)
			d.budget.Release(uint64(cap(prefixBytes)))
			prefixBytes = next
		}
		var v byte
		if i < dot {
			v, e = d.symbolByte(name, i)
			if e != nil {
				return Symbol{}, Symbol{}, false, e
			}
		}
		prefixBytes = append(prefixBytes, v)
	}
	prefix, e := d.internBytes(prefixBytes[:dot])
	if e != nil {
		return Symbol{}, Symbol{}, false, e
	}
	suffix, e := d.internSymbolRange(name, dot+1, n-dot-1)
	if e != nil {
		return Symbol{}, Symbol{}, false, e
	}
	return prefix, suffix, true, nil
}
func (d *ParamDescrs) KindInModule(name *Symbol) (ParamKind, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return ParamInvalid, e
	}
	if name == nil {
		return ParamInvalid, ErrParamDescrsPrecondition
	}
	k, e := d.kind(*name)
	if e != nil || k != ParamInvalid || name.IsNumerical() {
		return k, e
	}
	if *name == (Symbol{}) {
		return ParamInvalid, ErrParamDescrsPrecondition
	}
	prefix, suffix, split, e := d.splitName(*name)
	if e != nil || !split {
		return ParamInvalid, e
	}
	k, e = d.kind(suffix)
	if e != nil || k == ParamInvalid {
		return k, e
	}
	i, e := d.find(suffix)
	if e != nil {
		return ParamInvalid, e
	}
	module, e := d.internText(d.entries[i].info.module)
	if e != nil {
		return ParamInvalid, e
	}
	if module == prefix {
		*name = suffix
		return k, nil
	}
	return ParamInvalid, nil
}

// symbolCompare is strcmp over the borrowed manager's checked spelling,
// without materializing uncharged intermediate Go strings.
func (d *ParamDescrs) symbolCompare(a, b Symbol) (int, error) {
	m := d.manager
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.checkPlain(a); e != nil {
		return 0, e
	}
	if e := m.checkPlain(b); e != nil {
		return 0, e
	}
	if a.value == 0 || b.value == 0 || a.IsNumerical() || b.IsNumerical() {
		return 0, ErrParamDescrsPrecondition
	}
	ar := arena.BytesRegionRef{Slot: arena.Slot(a.value), Generation: a.generation}
	br := arena.BytesRegionRef{Slot: arena.Slot(b.value), Generation: b.generation}
	for i := m.pointerWidth; ; i++ {
		x, e := m.region.Read(ar, i)
		if e != nil {
			return 0, e
		}
		y, e := m.region.Read(br, i)
		if e != nil {
			return 0, e
		}
		if x < y {
			return -1, nil
		}
		if x > y {
			return 1, nil
		}
		if x == 0 {
			return 0, nil
		}
	}
}

// Concrete in-place sort realizes std::sort's comparator/output contract;
// it has no reflection swapper or additional heap-backed work arrays.
func (d *ParamDescrs) siftSymbols(a []Symbol, root, end int) error {
	for root < end/2 {
		child := root*2 + 1
		if child+1 < end {
			v, e := d.symbolCompare(a[child], a[child+1])
			if e != nil {
				return e
			}
			if v < 0 {
				child++
			}
		}
		v, e := d.symbolCompare(a[root], a[child])
		if e != nil {
			return e
		}
		if v >= 0 {
			return nil
		}
		a[root], a[child] = a[child], a[root]
		root = child
	}
	return nil
}
func (d *ParamDescrs) sortSymbols(a []Symbol) error {
	for i := len(a) / 2; i > 0; i-- {
		if e := d.siftSymbols(a, i-1, len(a)); e != nil {
			return e
		}
	}
	for end := len(a) - 1; end > 0; end-- {
		a[0], a[end] = a[end], a[0]
		if e := d.siftSymbols(a, 0, end); e != nil {
			return e
		}
	}
	return nil
}
func (d *ParamDescrs) writeName(out *strings.Builder, k Symbol, smt2 bool) error {
	m := d.manager
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.checkPlain(k); e != nil {
		return e
	}
	if k.value == 0 || k.IsNumerical() {
		return ErrParamDescrsPrecondition
	}
	r := arena.BytesRegionRef{Slot: arena.Slot(k.value), Generation: k.generation}
	for i := m.pointerWidth; ; i++ {
		ch, e := m.region.Read(r, i)
		if e != nil {
			return e
		}
		if ch == 0 {
			return nil
		}
		if smt2 && ch == '_' {
			ch = '-'
		} else if !smt2 && ch == '-' {
			ch = '_'
		} else if ch >= 'A' && ch <= 'Z' {
			ch = ch - 'A' + 'a'
		}
		out.WriteByte(ch)
	}
}
func (d *ParamDescrs) writeText(out *strings.Builder, r paramDescrText) error {
	for i := uint32(0); i < r.length; i++ {
		v, e := d.texts.Read(r.ref, i/4)
		if e != nil {
			return e
		}
		out.WriteByte(byte(v >> (8 * (i % 4))))
	}
	return nil
}
func (d *ParamDescrs) Display(indent uint32, smt2, includeDescr bool) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return "", e
	}
	var names []Symbol
	defer func() { d.budget.Release(pdNameBytes(cap(names))) }()
	for _, x := range d.entries {
		if x.state != 2 {
			continue
		}
		if len(names) == cap(names) {
			n, e := pdGrowth(cap(names))
			if e != nil {
				return "", e
			}
			if uint64(n) > uint64(^uint(0)>>1)/uint64(reflect.TypeOf(Symbol{}).Size()) {
				return "", arena.ErrBounds
			}
			if e = d.budget.Reserve(pdNameBytes(n)); e != nil {
				return "", e
			}
			next := make([]Symbol, len(names), n)
			copy(next, names)
			d.budget.Release(pdNameBytes(cap(names)))
			names = next
		}
		names = append(names, x.key)
	}
	for _, k := range names {
		if _, e := d.symbolCompare(k, k); e != nil {
			return "", e
		}
		i, e := d.find(k)
		if e != nil {
			return "", e
		}
		if d.entries[i].info.descr.ref == (arena.VectorRef{}) {
			return "", ErrParamDescrsPrecondition
		}
	}
	if e := d.sortSymbols(names); e != nil {
		return "", e
	}
	var out strings.Builder
	for _, key := range names {
		for j := uint32(0); j < indent; j++ {
			out.WriteByte(' ')
		}
		if smt2 {
			out.WriteByte(':')
		}
		if e := d.writeName(&out, key, smt2); e != nil {
			return "", e
		}
		idx, e := d.find(key)
		if e != nil {
			return "", e
		}
		info := d.entries[idx].info
		out.WriteString(" (")
		out.WriteString(info.kind.String())
		out.WriteByte(')')
		if includeDescr {
			out.WriteByte(' ')
			if e = d.writeText(&out, info.descr); e != nil {
				return "", e
			}
		}
		if info.def.ref != (arena.VectorRef{}) {
			out.WriteString(" (default: ")
			if e = d.writeText(&out, info.def); e != nil {
				return "", e
			}
			out.WriteByte(')')
		}
		out.WriteByte('\n')
	}
	return out.String(), nil
}
func (d *ParamDescrs) Copy(other *ParamDescrs) error {
	if other == nil {
		return ErrParamDescrsPrecondition
	}
	paramDescrsCopyMutex.Lock()
	defer paramDescrsCopyMutex.Unlock()
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return e
	}
	if d == other {
		return nil
	}
	other.mu.Lock()
	defer other.mu.Unlock()
	if e := other.open(); e != nil {
		return e
	}
	if d.manager != other.manager {
		return ErrParamDescrsPrecondition
	}
	for _, x := range other.entries {
		if x.state != 2 {
			continue
		}
		if e := d.insertCopy(other, x); e != nil {
			return e
		}
	}
	return nil
}
func (d *ParamDescrs) Stats() ParamDescrsStats {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return ParamDescrsStats{}
	}
	var text arena.VectorStats
	if d.texts != nil {
		text = d.texts.Stats()
	}
	return ParamDescrsStats{d.size, d.deleted, uint32(len(d.entries)), uint32(len(d.names)), uint32(cap(d.names)), d.ownerBytes, pdEntryBytes(len(d.entries)), pdNameBytes(cap(d.names)), text}
}
func (d *ParamDescrs) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.self != d {
		return ErrParamDescrsPrecondition
	}
	if d.closed {
		return nil
	}
	if !d.constructorReady {
		return ErrParamDescrsPrecondition
	}
	d.texts.Close()
	d.budget.Release(pdEntryBytes(len(d.entries)) + pdNameBytes(cap(d.names)) + d.ownerBytes)
	d.ownerBytes = 0
	d.entries = nil
	d.names = nil
	d.size = 0
	d.deleted = 0
	d.closed = true
	return nil
}
func InsertMaxMemory(d *ParamDescrs) error {
	return d.InsertName("max_memory", ParamUInt, pdLiteral("(default: infty) maximum amount of memory in megabytes."), nil, nil)
}
func InsertMaxSteps(d *ParamDescrs) error {
	return d.InsertName("max_steps", ParamUInt, pdLiteral("(default: infty) maximum number of steps."), nil, nil)
}
func InsertProduceModels(d *ParamDescrs) error {
	return d.InsertName("produce_models", ParamBool, pdLiteral("(default: false) model generation."), nil, nil)
}
func InsertProduceProofs(d *ParamDescrs) error {
	return d.InsertName("produce_proofs", ParamBool, pdLiteral("(default: false) proof generation."), nil, nil)
}
func InsertTimeout(d *ParamDescrs) error {
	return d.InsertName("timeout", ParamUInt, pdLiteral("(default: infty) timeout in milliseconds."), nil, nil)
}
func pdLiteral(s string) *string { return &s }

// sourceFinalizeMembers follows the actual reverse m_names/m_info declaration
// order. Copied text is additional Go storage and drains after source members.
func (d *ParamDescrs) sourceFinalizeMembers() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.self != d {
		return ErrParamDescrsPrecondition
	}
	if d.closed {
		return nil
	}
	if !d.constructorReady {
		return ErrParamDescrsPrecondition
	}
	return d.closeSourceMembers()
}

// Caller holds mu; each consumed concrete backing is cleared immediately.
func (d *ParamDescrs) closeSourceMembers() error {
	d.budget.Release(pdNameBytes(cap(d.names)))
	d.names = nil
	d.budget.Release(pdEntryBytes(len(d.entries)))
	d.entries = nil
	d.size = 0
	d.deleted = 0
	d.texts.Close()
	d.budget.Release(d.ownerBytes)
	d.ownerBytes = 0
	d.closed = true
	return nil
}
