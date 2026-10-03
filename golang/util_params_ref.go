// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
// Translation of src/util/params.h/.cpp; concrete owner-local storage.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

var ErrParamsOwner = errors.New("g3: parameters owner or lifetime mismatch")
var ErrParamsStreamFailure = errors.New("g3: null C-string failed parameter stream")
var ErrParamsPrecondition = errors.New("g3: parameters source precondition")
var paramsNextOwner uint64

// ParamsRef owns a reference, never shallow-copy it. NewValue creates the
// source null backing; Assign is the reference-counted copy operation.
type ParamsRef struct {
	owner            uint64
	slot, generation uint32
}
type ParamsConfig struct {
	Budget      *arena.Budget
	PageEntries uint32
	Text        arena.VectorConfig
}
type paramsEntriesRef struct{ page, block, generation, capacity uint32 }
type paramsBlock struct {
	refs, generation, size, capacity uint32
	entries                          paramsEntriesRef
	live                             bool
}
type paramsText struct {
	ref     arena.VectorRef
	size    uint32
	present bool
}
type paramEntry struct {
	key    Symbol
	kind   ParamKind
	bits   uint64
	text   paramsText
	symbol Symbol
	rat    mpnum.OwnedRational
}
type paramsEntryState struct {
	generation uint32
	used       bool
}
type paramsEntryPage struct {
	entries []paramEntry
	states  []paramsEntryState
	class   uint32
}
type ParamsStats struct {
	LiveBlocks, References, Entries                                 uint32
	HeaderBytes, BlockBytes, PageBytes, DirectoryBytes, FormatBytes uint64
	Text                                                            arena.VectorStats
}

// This cold owner borrows both services. They must outlive it; budgets shared
// across owners require external confinement. All hot records are pointer-free.
type ParamsContext struct {
	mu          sync.Mutex
	self        *ParamsContext
	symbols     *SymbolManager
	numerals    *mpnum.RationalService
	budget      *arena.Budget
	texts       *arena.Words
	owner       uint64
	blocks      []paramsBlock
	pages       []paramsEntryPage
	workspace   []byte
	pageEntries uint32
	headerBytes uint64
	closed      bool
}

func paramsBytes(n int, t reflect.Type) uint64 { return uint64(n) * uint64(t.Size()) }
func paramsBlockBytes(n int) uint64            { return paramsBytes(n, reflect.TypeOf(paramsBlock{})) }
func paramsPageBytes(p paramsEntryPage) uint64 {
	return paramsBytes(cap(p.entries), reflect.TypeOf(paramEntry{})) + paramsBytes(cap(p.states), reflect.TypeOf(paramsEntryState{}))
}
func paramsDirectoryBytes(n int) uint64 { return paramsBytes(n, reflect.TypeOf(paramsEntryPage{})) }
func NewParamsContext(s *SymbolManager, q *mpnum.RationalService, c ParamsConfig) (*ParamsContext, error) {
	if s == nil || q == nil || c.Text.Descriptors.Budget != c.Budget {
		return nil, arena.ErrConfig
	}
	if _, e := s.Hash(Symbol{}); e != nil {
		return nil, e
	}
	r, e := q.NewValue()
	if e != nil {
		return nil, e
	}
	if e = q.Del(&r); e != nil {
		return nil, e
	}
	var id uint64
	for {
		old := atomic.LoadUint64(&paramsNextOwner)
		if old == math.MaxUint64 {
			return nil, arena.ErrBounds
		}
		if atomic.CompareAndSwapUint64(&paramsNextOwner, old, old+1) {
			id = old + 1
			break
		}
	}
	n := c.PageEntries
	if n == 0 {
		n = 64
	}
	if n > 1<<30 {
		return nil, arena.ErrBounds
	}
	h := uint64(reflect.TypeOf(ParamsContext{}).Size())
	if e = c.Budget.Reserve(h); e != nil {
		return nil, e
	}
	w, e := arena.NewWords(c.Text)
	if e != nil {
		c.Budget.Release(h)
		return nil, e
	}
	p := &ParamsContext{symbols: s, numerals: q, budget: c.Budget, texts: w, owner: id, pageEntries: n, headerBytes: h}
	p.self = p
	return p, nil
}
func (c *ParamsContext) open() error {
	if c == nil || c.self != c || c.closed {
		return ErrParamsOwner
	}
	if _, e := c.symbols.Hash(Symbol{}); e != nil {
		return e
	}
	return nil
}
func (c *ParamsContext) check(v *ParamsRef) error {
	if e := c.open(); e != nil {
		return e
	}
	if v == nil || v.owner != c.owner {
		return ErrParamsOwner
	}
	if v.slot == 0 {
		return nil
	}
	if int(v.slot) > len(c.blocks) {
		return ErrParamsOwner
	}
	b := &c.blocks[v.slot-1]
	if !b.live || b.generation != v.generation {
		return ErrParamsOwner
	}
	return nil
}
func (c *ParamsContext) NewValue() (ParamsRef, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.open(); e != nil {
		return ParamsRef{}, e
	}
	return ParamsRef{owner: c.owner}, nil
}
func (c *ParamsContext) entry(b *paramsBlock, i uint32) *paramEntry {
	r := b.entries
	p := &c.pages[r.page-1]
	return &p.entries[r.block*p.class+i]
}
func (c *ParamsContext) allocateEntries(n uint32) (paramsEntriesRef, error) {
	class := uint32(1)
	for class < n {
		if class >= 1<<30 {
			return paramsEntriesRef{}, arena.ErrBounds
		}
		class *= 2
	}
	for pi := range c.pages {
		p := &c.pages[pi]
		if p.class != class {
			continue
		}
		for bi := range p.states {
			s := &p.states[bi]
			if !s.used && s.generation != math.MaxUint32 {
				s.generation++
				s.used = true
				return paramsEntriesRef{uint32(pi + 1), uint32(bi), s.generation, class}, nil
			}
		}
	}
	count := c.pageEntries / class
	if count == 0 {
		count = 1
	}
	total := uint64(count) * uint64(class)
	if total > uint64(^uint(0)>>1)/uint64(reflect.TypeOf(paramEntry{}).Size()) {
		return paramsEntriesRef{}, arena.ErrBounds
	}
	p := paramsEntryPage{class: class}
	bytes := total*uint64(reflect.TypeOf(paramEntry{}).Size()) + uint64(count)*uint64(reflect.TypeOf(paramsEntryState{}).Size())
	if e := c.budget.Reserve(bytes); e != nil {
		return paramsEntriesRef{}, e
	}
	p.entries = make([]paramEntry, int(total))
	p.states = make([]paramsEntryState, count)
	if len(c.pages) == cap(c.pages) {
		n, e := pdGrowth(cap(c.pages))
		if e != nil {
			c.budget.Release(bytes)
			return paramsEntriesRef{}, e
		}
		if e = c.budget.Reserve(paramsDirectoryBytes(n)); e != nil {
			c.budget.Release(bytes)
			return paramsEntriesRef{}, e
		}
		next := make([]paramsEntryPage, len(c.pages), n)
		copy(next, c.pages)
		c.budget.Release(paramsDirectoryBytes(cap(c.pages)))
		c.pages = next
	}
	p.states[0] = paramsEntryState{1, true}
	c.pages = append(c.pages, p)
	return paramsEntriesRef{uint32(len(c.pages)), 0, 1, class}, nil
}
func (c *ParamsContext) freeEntries(r paramsEntriesRef) {
	if r.page != 0 {
		c.pages[r.page-1].states[r.block].used = false
	}
}
func (c *ParamsContext) newBlock() (uint32, error) {
	for i := range c.blocks {
		b := &c.blocks[i]
		if !b.live && b.generation != math.MaxUint32 {
			g := b.generation + 1
			*b = paramsBlock{refs: 1, generation: g, live: true}
			return uint32(i + 1), nil
		}
	}
	if len(c.blocks) == cap(c.blocks) {
		n, e := pdGrowth(cap(c.blocks))
		if e != nil {
			return 0, e
		}
		if e = c.budget.Reserve(paramsBlockBytes(n)); e != nil {
			return 0, e
		}
		next := make([]paramsBlock, len(c.blocks), n)
		copy(next, c.blocks)
		c.budget.Release(paramsBlockBytes(cap(c.blocks)))
		c.blocks = next
	}
	c.blocks = append(c.blocks, paramsBlock{refs: 1, generation: 1, live: true})
	return uint32(len(c.blocks)), nil
}
func (c *ParamsContext) delEntry(x *paramEntry) error {
	if x.kind == ParamNumeral {
		if e := c.numerals.Del(&x.rat); e != nil {
			return e
		}
	}
	if x.text.present {
		if e := c.texts.Free(x.text.ref); e != nil {
			return e
		}
	}
	*x = paramEntry{}
	return nil
}
func (c *ParamsContext) resetBlock(b *paramsBlock) error {
	for i := uint32(0); i < b.size; i++ {
		if e := c.delEntry(c.entry(b, i)); e != nil {
			return e
		}
	}
	c.freeEntries(b.entries)
	b.entries = paramsEntriesRef{}
	b.size = 0
	b.capacity = 0
	return nil
}
func (c *ParamsContext) dec(slot uint32) error {
	if slot == 0 {
		return nil
	}
	b := &c.blocks[slot-1]
	b.refs--
	if b.refs == 0 {
		if e := c.resetBlock(b); e != nil {
			return e
		}
		b.live = false
	}
	return nil
}
func (c *ParamsContext) assign(dst, src *ParamsRef) error {
	if e := c.check(dst); e != nil {
		return e
	}
	if e := c.check(src); e != nil {
		return e
	}
	if src.slot != 0 {
		b := &c.blocks[src.slot-1]
		if b.refs == math.MaxUint32 {
			return arena.ErrBounds
		}
		b.refs++
	}
	old := dst.slot
	*dst = *src
	return c.dec(old)
}
func (c *ParamsContext) Assign(dst, src *ParamsRef) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.assign(dst, src)
}
func (c *ParamsContext) Del(v *ParamsRef) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.check(v); e != nil {
		return e
	}
	e := c.dec(v.slot)
	if e == nil {
		*v = ParamsRef{}
	}
	return e
}
func (c *ParamsContext) init(v *ParamsRef) error {
	if e := c.check(v); e != nil {
		return e
	}
	if v.slot != 0 && c.blocks[v.slot-1].refs == 1 {
		return nil
	}
	old := v.slot
	slot, e := c.newBlock()
	if e != nil {
		return e
	}
	v.slot = slot
	v.generation = c.blocks[slot-1].generation
	if old != 0 {
		e = c.copyCore(slot, old)
		de := c.dec(old)
		if e == nil {
			e = de
		}
	}
	return e
}
func (c *ParamsContext) grow(b *paramsBlock) error {
	n, e := pdGrowth(int(b.capacity))
	if e != nil {
		return e
	}
	r, e := c.allocateEntries(uint32(n))
	if e != nil {
		return e
	}
	old := b.entries
	tmp := paramsBlock{entries: r}
	for i := uint32(0); i < b.size; i++ {
		*c.entry(&tmp, i) = *c.entry(b, i)
	}
	b.entries = r
	b.capacity = uint32(n)
	if old.page != 0 {
		prior := paramsBlock{entries: old}
		for i := uint32(0); i < b.size; i++ {
			*c.entry(&prior, i) = paramEntry{}
		}
		c.freeEntries(old)
	}
	return nil
}
func (c *ParamsContext) copyCore(dst, src uint32) error {
	if src == 0 {
		return nil
	}
	n := c.blocks[src-1].size
	for i := uint32(0); i < n; i++ {
		x := c.entry(&c.blocks[src-1], i)
		if e := c.setCore(dst, x.key, x.kind, x.bits, x.symbol, nil, &x.rat, &x.text); e != nil {
			return e
		}
	}
	return nil
}
func (c *ParamsContext) Copy(dst, src *ParamsRef) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.check(dst); e != nil {
		return e
	}
	if e := c.check(src); e != nil {
		return e
	}
	if dst.slot == 0 {
		return c.assign(dst, src)
	}
	if e := c.init(dst); e != nil {
		return e
	}
	return c.copyCore(dst.slot, src.slot)
}
func (c *ParamsContext) Append(dst, src *ParamsRef) error { return c.Copy(dst, src) }
func (c *ParamsContext) keyCheck(k Symbol) error          { _, e := c.symbols.Hash(k); return e }
func (c *ParamsContext) find(v *ParamsRef, k Symbol, name *string, byName bool, kind ParamKind, any bool) (*paramEntry, error) {
	if e := c.check(v); e != nil {
		return nil, e
	}
	if !byName {
		if e := c.keyCheck(k); e != nil {
			return nil, e
		}
	}
	if v.slot == 0 {
		return nil, nil
	}
	b := &c.blocks[v.slot-1]
	for i := uint32(0); i < b.size; i++ {
		x := c.entry(b, i)
		match := x.key == k
		if byName {
			var e error
			match, e = c.symbols.EqualString(x.key, name)
			if e != nil {
				return nil, e
			}
		}
		if match && (any || x.kind == kind || (kind == ParamNumeral && x.kind == ParamUInt)) {
			return x, nil
		}
	}
	return nil, nil
}
func (c *ParamsContext) Contains(v *ParamsRef, k Symbol) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	x, e := c.find(v, k, nil, false, 0, true)
	return x != nil, e
}
func (c *ParamsContext) ContainsName(v *ParamsRef, k *string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	x, e := c.find(v, Symbol{}, k, true, 0, true)
	return x != nil, e
}
func (c *ParamsContext) Empty(v *ParamsRef) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.check(v); e != nil {
		return false, e
	}
	return v.slot == 0 || c.blocks[v.slot-1].size == 0, nil
}
func (c *ParamsContext) Reset(v *ParamsRef) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.check(v); e != nil {
		return e
	}
	if v.slot == 0 {
		return nil
	}
	return c.resetBlock(&c.blocks[v.slot-1])
}
func (c *ParamsContext) resetKey(v *ParamsRef, k Symbol, name *string, byName bool) error {
	x, e := c.find(v, k, name, byName, 0, true)
	if e != nil || x == nil {
		return e
	}
	b := &c.blocks[v.slot-1]
	for i := uint32(0); i < b.size; i++ {
		if c.entry(b, i) != x {
			continue
		}
		if e = c.delEntry(x); e != nil {
			return e
		}
		for j := i + 1; j < b.size; j++ {
			*c.entry(b, j-1) = *c.entry(b, j)
		}
		b.size--
		*c.entry(b, b.size) = paramEntry{}
		break
	}
	return nil
}
func (c *ParamsContext) ResetKey(v *ParamsRef, k Symbol) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.resetKey(v, k, nil, false)
}
func (c *ParamsContext) ResetKeyName(v *ParamsRef, k *string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.resetKey(v, Symbol{}, k, true)
}
func (c *ParamsContext) storeText(s *string, source *paramsText) (paramsText, error) {
	n := uint32(0)
	present := s != nil
	if source != nil {
		n = source.size
		present = source.present
	} else if s != nil {
		prefix := len(*s)
		if i := strings.IndexByte(*s, 0); i >= 0 {
			prefix = i
		}
		var e error
		n, e = paramsCheckedTextLength(uint64(prefix))
		if e != nil {
			return paramsText{}, e
		}
	}
	if !present {
		return paramsText{}, nil
	}
	length := (uint64(n) + 3) / 4
	if length == 0 {
		length = 1
	}
	if length > math.MaxUint32 {
		return paramsText{}, arena.ErrBounds
	}
	r, e := c.texts.Allocate(uint32(length), uint32(length))
	if e != nil {
		return paramsText{}, e
	}
	out := paramsText{r, n, true}
	for i := uint32(0); i < uint32(length); i++ {
		var word uint32
		for j := uint32(0); j < 4 && i*4+j < n; j++ {
			var ch byte
			if source != nil {
				old, er := c.texts.Read(source.ref, (i*4+j)/4)
				if er != nil {
					c.texts.Free(r)
					return paramsText{}, er
				}
				ch = byte(old >> (8 * ((i*4 + j) % 4)))
			} else {
				ch = (*s)[i*4+j]
			}
			word |= uint32(ch) << (8 * j)
		}
		if e = c.texts.Write(r, i, word); e != nil {
			c.texts.Free(r)
			return paramsText{}, e
		}
	}
	return out, nil
}
func (c *ParamsContext) setCore(slot uint32, k Symbol, kind ParamKind, bits uint64, sym Symbol, str *string, rat *mpnum.OwnedRational, text *paramsText) error {
	b := &c.blocks[slot-1]
	var x *paramEntry
	for i := uint32(0); i < b.size; i++ {
		p := c.entry(b, i)
		if p.key == k {
			x = p
			break
		}
	}
	if kind == ParamNumeral && x != nil {
		if x.kind != ParamNumeral {
			if e := c.delEntry(x); e != nil {
				return e
			}
			r, e := c.numerals.NewValue()
			if e != nil {
				return e
			}
			*x = paramEntry{key: k, kind: kind, rat: r}
		}
		return c.numerals.Copy(&x.rat, rat)
	}
	// Prepare owned text before deleting a replaced value: adapting borrowed C
	// text to owned payload adds a failure point, without publishing a dangling tag.
	next := paramEntry{key: k, kind: kind, bits: bits, symbol: sym}
	if kind == ParamString {
		t, e := c.storeText(str, text)
		if e != nil {
			return e
		}
		next.text = t
	}
	if kind == ParamNumeral {
		r, e := c.numerals.NewValue()
		if e != nil {
			return e
		}
		next.rat = r
		if e = c.numerals.Copy(&next.rat, rat); e != nil {
			c.numerals.Del(&next.rat)
			return e
		}
	}
	if x != nil {
		if e := c.delEntry(x); e != nil {
			c.delEntry(&next)
			return e
		}
		*x = next
		return nil
	}
	if b.size == b.capacity {
		if e := c.grow(b); e != nil {
			c.delEntry(&next)
			return e
		}
	}
	*c.entry(b, b.size) = next
	b.size++
	return nil
}
func (c *ParamsContext) set(v *ParamsRef, k Symbol, name *string, byName bool, kind ParamKind, bits uint64, sym Symbol, str *string, rat *mpnum.OwnedRational) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.check(v); e != nil {
		return e
	}
	// Provenance guards precede source mutation; key construction follows init.
	if !byName {
		if e := c.keyCheck(k); e != nil {
			return e
		}
	}
	if kind == ParamSymbol {
		if sym.IsNumerical() {
			return ErrParamsPrecondition
		}
		if e := c.keyCheck(sym); e != nil {
			return e
		}
	}
	if kind == ParamNumeral {
		if _, e := c.numerals.IsRational(rat); e != nil {
			return e
		}
	}
	if e := c.init(v); e != nil {
		return e
	}
	if byName {
		var e error
		k, e = c.symbols.InternNullable(name)
		if e != nil {
			return e
		}
	}
	return c.setCore(v.slot, k, kind, bits, sym, str, rat, nil)
}
func (c *ParamsContext) SetBool(v *ParamsRef, k Symbol, b bool) error {
	var n uint64
	if b {
		n = 1
	}
	return c.set(v, k, nil, false, ParamBool, n, Symbol{}, nil, nil)
}
func (c *ParamsContext) SetBoolName(v *ParamsRef, k *string, b bool) error {
	var n uint64
	if b {
		n = 1
	}
	return c.set(v, Symbol{}, k, true, ParamBool, n, Symbol{}, nil, nil)
}
func (c *ParamsContext) SetUInt(v *ParamsRef, k Symbol, n uint32) error {
	return c.set(v, k, nil, false, ParamUInt, uint64(n), Symbol{}, nil, nil)
}
func (c *ParamsContext) SetUIntName(v *ParamsRef, k *string, n uint32) error {
	return c.set(v, Symbol{}, k, true, ParamUInt, uint64(n), Symbol{}, nil, nil)
}
func (c *ParamsContext) SetDouble(v *ParamsRef, k Symbol, n float64) error {
	return c.set(v, k, nil, false, ParamDouble, math.Float64bits(n), Symbol{}, nil, nil)
}
func (c *ParamsContext) SetDoubleName(v *ParamsRef, k *string, n float64) error {
	return c.set(v, Symbol{}, k, true, ParamDouble, math.Float64bits(n), Symbol{}, nil, nil)
}
func (c *ParamsContext) SetString(v *ParamsRef, k Symbol, s *string) error {
	return c.set(v, k, nil, false, ParamString, 0, Symbol{}, s, nil)
}
func (c *ParamsContext) SetStringName(v *ParamsRef, k, s *string) error {
	return c.set(v, Symbol{}, k, true, ParamString, 0, Symbol{}, s, nil)
}
func (c *ParamsContext) SetSymbol(v *ParamsRef, k, s Symbol) error {
	return c.set(v, k, nil, false, ParamSymbol, 0, s, nil, nil)
}
func (c *ParamsContext) SetSymbolName(v *ParamsRef, k *string, s Symbol) error {
	return c.set(v, Symbol{}, k, true, ParamSymbol, 0, s, nil, nil)
}
func (c *ParamsContext) SetRat(v *ParamsRef, k Symbol, r *mpnum.OwnedRational) error {
	return c.set(v, k, nil, false, ParamNumeral, 0, Symbol{}, nil, r)
}
func (c *ParamsContext) SetRatName(v *ParamsRef, k *string, r *mpnum.OwnedRational) error {
	return c.set(v, Symbol{}, k, true, ParamNumeral, 0, Symbol{}, nil, r)
}
func (c *ParamsContext) get(v *ParamsRef, k Symbol, name *string, byName bool, kind ParamKind, fallback *ParamsRef) (paramEntry, bool, error) {
	x, e := c.find(v, k, name, byName, kind, false)
	if e != nil {
		return paramEntry{}, false, e
	}
	if fallback != nil {
		if e = c.check(fallback); e != nil {
			return paramEntry{}, false, e
		}
		if x == nil {
			x, e = c.find(fallback, Symbol{}, name, true, kind, false)
			if e != nil {
				return paramEntry{}, false, e
			}
		}
	}
	if x == nil {
		return paramEntry{}, false, nil
	}
	return *x, true, nil
}
func (c *ParamsContext) getBool(v *ParamsRef, k Symbol, name *string, byName bool, f *ParamsRef, d bool) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	x, ok, e := c.get(v, k, name, byName, ParamBool, f)
	if e != nil || !ok {
		return d, e
	}
	return x.bits != 0, nil
}
func (c *ParamsContext) GetBool(v *ParamsRef, k Symbol, d bool) (bool, error) {
	return c.getBool(v, k, nil, false, nil, d)
}
func (c *ParamsContext) GetBoolName(v *ParamsRef, k *string, d bool) (bool, error) {
	return c.getBool(v, Symbol{}, k, true, nil, d)
}
func (c *ParamsContext) GetBoolFallbackName(v *ParamsRef, k *string, f *ParamsRef, d bool) (bool, error) {
	return c.getBool(v, Symbol{}, k, true, f, d)
}
func (c *ParamsContext) getUInt(v *ParamsRef, k Symbol, name *string, byName bool, f *ParamsRef, d uint32) (uint32, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	x, ok, e := c.get(v, k, name, byName, ParamUInt, f)
	if e != nil || !ok {
		return d, e
	}
	return uint32(x.bits), nil
}
func (c *ParamsContext) GetUInt(v *ParamsRef, k Symbol, d uint32) (uint32, error) {
	return c.getUInt(v, k, nil, false, nil, d)
}
func (c *ParamsContext) GetUIntName(v *ParamsRef, k *string, d uint32) (uint32, error) {
	return c.getUInt(v, Symbol{}, k, true, nil, d)
}
func (c *ParamsContext) GetUIntFallbackName(v *ParamsRef, k *string, f *ParamsRef, d uint32) (uint32, error) {
	return c.getUInt(v, Symbol{}, k, true, f, d)
}
func (c *ParamsContext) getDouble(v *ParamsRef, k Symbol, name *string, byName bool, f *ParamsRef, d float64) (float64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	x, ok, e := c.get(v, k, name, byName, ParamDouble, f)
	if e != nil || !ok {
		return d, e
	}
	return math.Float64frombits(x.bits), nil
}
func (c *ParamsContext) GetDouble(v *ParamsRef, k Symbol, d float64) (float64, error) {
	return c.getDouble(v, k, nil, false, nil, d)
}
func (c *ParamsContext) GetDoubleName(v *ParamsRef, k *string, d float64) (float64, error) {
	return c.getDouble(v, Symbol{}, k, true, nil, d)
}
func (c *ParamsContext) GetDoubleFallbackName(v *ParamsRef, k *string, f *ParamsRef, d float64) (float64, error) {
	return c.getDouble(v, Symbol{}, k, true, f, d)
}
func (c *ParamsContext) getSymbol(v *ParamsRef, k Symbol, name *string, byName bool, f *ParamsRef, d Symbol) (Symbol, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.keyCheck(d); e != nil {
		return Symbol{}, e
	}
	x, ok, e := c.get(v, k, name, byName, ParamSymbol, f)
	if e != nil || !ok {
		return d, e
	}
	return x.symbol, nil
}
func (c *ParamsContext) GetSymbol(v *ParamsRef, k, d Symbol) (Symbol, error) {
	return c.getSymbol(v, k, nil, false, nil, d)
}
func (c *ParamsContext) GetSymbolName(v *ParamsRef, k *string, d Symbol) (Symbol, error) {
	return c.getSymbol(v, Symbol{}, k, true, nil, d)
}
func (c *ParamsContext) GetSymbolFallbackName(v *ParamsRef, k *string, f *ParamsRef, d Symbol) (Symbol, error) {
	return c.getSymbol(v, Symbol{}, k, true, f, d)
}

// String results are caller-owned copies; bool distinguishes null from empty.
func (c *ParamsContext) getString(v *ParamsRef, k Symbol, name *string, byName bool, f *ParamsRef, d *string) (string, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	x, ok, e := c.get(v, k, name, byName, ParamString, f)
	if e != nil {
		return "", false, e
	}
	if !ok {
		if d == nil {
			return "", false, nil
		}
		return paramsCString(*d), true, nil
	}
	if !x.text.present {
		return "", false, nil
	}
	c.workspace = c.workspace[:0]
	if e = c.appendText(c.texts, x.text.ref, x.text.size); e != nil {
		return "", false, e
	}
	return string(c.workspace), true, nil
}
func (c *ParamsContext) GetString(v *ParamsRef, k Symbol, d *string) (string, bool, error) {
	return c.getString(v, k, nil, false, nil, d)
}
func (c *ParamsContext) GetStringName(v *ParamsRef, k, d *string) (string, bool, error) {
	return c.getString(v, Symbol{}, k, true, nil, d)
}
func (c *ParamsContext) GetStringFallbackName(v *ParamsRef, k *string, f *ParamsRef, d *string) (string, bool, error) {
	return c.getString(v, Symbol{}, k, true, f, d)
}
func (c *ParamsContext) getRat(v *ParamsRef, k Symbol, name *string, byName bool, d *mpnum.OwnedRational) (mpnum.OwnedRational, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, e := c.numerals.IsRational(d); e != nil {
		return mpnum.OwnedRational{}, e
	}
	x, ok, e := c.get(v, k, name, byName, ParamNumeral, nil)
	if e != nil {
		return mpnum.OwnedRational{}, e
	}
	r, e := c.numerals.NewValue()
	if e != nil {
		return r, e
	}
	if ok && x.kind == ParamUInt {
		e = c.numerals.AssignInt(&r, int32(uint32(x.bits)))
	} else {
		src := d
		if ok {
			src = &x.rat
		}
		e = c.numerals.Copy(&r, src)
	}
	if e != nil {
		c.numerals.Del(&r)
		return mpnum.OwnedRational{}, e
	}
	return r, nil
}
func (c *ParamsContext) GetRat(v *ParamsRef, k Symbol, d *mpnum.OwnedRational) (mpnum.OwnedRational, error) {
	return c.getRat(v, k, nil, false, d)
}
func (c *ParamsContext) GetRatName(v *ParamsRef, k *string, d *mpnum.OwnedRational) (mpnum.OwnedRational, error) {
	return c.getRat(v, Symbol{}, k, true, d)
}
func paramsCString(s string) string {
	if i := strings.IndexByte(s, 0); i >= 0 {
		return s[:i]
	}
	return s
}
func (c *ParamsContext) reserveFormat(extra int) error {
	if extra < 0 || extra > int(^uint(0)>>1)-len(c.workspace) {
		return arena.ErrBounds
	}
	want := len(c.workspace) + extra
	if want <= cap(c.workspace) {
		return nil
	}
	n := cap(c.workspace)
	if n == 0 {
		n = 64
	}
	for n < want {
		if n > int(^uint(0)>>1)/2 {
			return arena.ErrBounds
		}
		n *= 2
	}
	if e := c.budget.Reserve(uint64(n)); e != nil {
		return e
	}
	next := make([]byte, len(c.workspace), n)
	copy(next, c.workspace)
	c.budget.Release(uint64(cap(c.workspace)))
	c.workspace = next
	return nil
}
func (c *ParamsContext) appendString(s string) error {
	if e := c.reserveFormat(len(s)); e != nil {
		return e
	}
	c.workspace = append(c.workspace, s...)
	return nil
}
func (c *ParamsContext) appendByte(b byte) error {
	if e := c.reserveFormat(1); e != nil {
		return e
	}
	c.workspace = append(c.workspace, b)
	return nil
}
func (c *ParamsContext) appendText(w *arena.Words, r arena.VectorRef, n uint32) error {
	if e := c.reserveFormat(int(n)); e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		v, e := w.Read(r, i/4)
		if e != nil {
			return e
		}
		c.workspace = append(c.workspace, byte(v>>(8*(i%4))))
	}
	return nil
}
func (c *ParamsContext) appendSymbol(k Symbol, strMode, normalize bool) error {
	m := c.symbols
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.checkPlain(k); e != nil {
		return e
	}
	if k.IsNumerical() {
		var a [32]byte
		b := strconv.AppendInt(a[:0], int64(int32(k.value)), 10)
		if e := c.appendString("k!"); e != nil {
			return e
		}
		if e := c.reserveFormat(len(b)); e != nil {
			return e
		}
		c.workspace = append(c.workspace, b...)
		return nil
	}
	if k.value == 0 {
		if normalize {
			return ErrParamsPrecondition
		}
		if strMode {
			return c.appendString("<null>")
		}
		return c.appendString("null")
	}
	r := arena.BytesRegionRef{Slot: arena.Slot(k.value), Generation: k.generation}
	for i := m.pointerWidth; ; i++ {
		b, e := m.region.Read(r, i)
		if e != nil {
			return e
		}
		if b == 0 {
			return nil
		}
		if normalize {
			if b == '-' {
				b = '_'
			} else if b >= 'A' && b <= 'Z' {
				b = b - 'A' + 'a'
			}
		}
		if e = c.appendByte(b); e != nil {
			return e
		}
	}
}
func (c *ParamsContext) appendValue(x *paramEntry) error {
	var a [128]byte
	var b []byte
	switch x.kind {
	case ParamBool:
		if x.bits != 0 {
			return c.appendString("true")
		}
		return c.appendString("false")
	case ParamUInt:
		b = strconv.AppendUint(a[:0], uint64(uint32(x.bits)), 10)
	case ParamDouble:
		v := math.Float64frombits(x.bits)
		if math.IsNaN(v) {
			if math.Signbit(v) {
				return c.appendString("-nan")
			}
			return c.appendString("nan")
		}
		if math.IsInf(v, 1) {
			return c.appendString("inf")
		}
		if math.IsInf(v, -1) {
			return c.appendString("-inf")
		}
		b = strconv.AppendFloat(a[:0], v, 'g', 6, 64)
	case ParamSymbol:
		return c.appendSymbol(x.symbol, false, false)
	case ParamString:
		if !x.text.present {
			return ErrParamsStreamFailure
		}
		return c.appendText(c.texts, x.text.ref, x.text.size)
	case ParamNumeral:
		for {
			if e := c.reserveFormat(64); e != nil {
				return e
			}
			old := len(c.workspace)
			dst := c.workspace[old:cap(c.workspace)]
			n, e := c.numerals.FormatInto(&x.rat, dst)
			if errors.Is(e, mpnum.ErrShortBuffer) {
				if e = c.reserveFormat(cap(c.workspace) - old + 1); e != nil {
					return e
				}
				continue
			}
			if e != nil {
				return e
			}
			c.workspace = c.workspace[:old+n]
			return nil
		}
	default:
		return ErrParamsPrecondition
	}
	if e := c.reserveFormat(len(b)); e != nil {
		return e
	}
	c.workspace = append(c.workspace, b...)
	return nil
}
func (c *ParamsContext) format(v *ParamsRef, key *Symbol, module *string, d *ParamDescrs) error {
	if e := c.check(v); e != nil {
		return e
	}
	if key != nil {
		if e := c.keyCheck(*key); e != nil {
			return e
		}
	}
	if d != nil {
		if d.manager != c.symbols {
			return ErrParamsOwner
		}
		if e := d.open(); e != nil {
			return e
		}
	}
	c.workspace = c.workspace[:0]
	if d == nil && key == nil {
		if e := c.appendString("(params"); e != nil {
			return e
		}
	}
	if v.slot != 0 {
		b := &c.blocks[v.slot-1]
		for i := uint32(0); i < b.size; i++ {
			x := c.entry(b, i)
			if key != nil {
				if x.key != *key {
					continue
				}
				return c.appendValue(x)
			}
			if d != nil {
				idx, e := d.find(x.key)
				if e != nil {
					return e
				}
				if idx < 0 {
					continue
				}
				if e = c.appendString("(set-option :"); e != nil {
					return e
				}
				if module == nil {
					return ErrParamsStreamFailure
				}
				if e = c.appendString(paramsCString(*module)); e != nil {
					return e
				}
				if e = c.appendByte('.'); e != nil {
					return e
				}
			} else {
				if e := c.appendByte(' '); e != nil {
					return e
				}
			}
			if e := c.appendSymbol(x.key, false, false); e != nil {
				return e
			}
			if e := c.appendByte(' '); e != nil {
				return e
			}
			if e := c.appendValue(x); e != nil {
				return e
			}
			if d != nil {
				if e := c.appendString(")\n"); e != nil {
					return e
				}
			}
		}
	}
	if key != nil {
		return c.appendString("default")
	}
	if d == nil {
		return c.appendByte(')')
	}
	return nil
}
func (c *ParamsContext) Display(v *ParamsRef) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.format(v, nil, nil, nil); e != nil {
		if errors.Is(e, ErrParamsStreamFailure) {
			return string(c.workspace), e
		}
		return "", e
	}
	return string(c.workspace), nil
}
func (c *ParamsContext) FormatInto(v *ParamsRef, dst []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.format(v, nil, nil, nil); e != nil {
		return 0, e
	}
	if len(dst) < len(c.workspace) {
		return len(c.workspace), mpnum.ErrShortBuffer
	}
	return copy(dst, c.workspace), nil
}
func (c *ParamsContext) DisplayKey(v *ParamsRef, k Symbol) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.format(v, &k, nil, nil); e != nil {
		if errors.Is(e, ErrParamsStreamFailure) {
			return string(c.workspace), e
		}
		return "", e
	}
	return string(c.workspace), nil
}
func (c *ParamsContext) DisplaySMT2(v *ParamsRef, module *string, d *ParamDescrs) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if d == nil {
		return "", ErrParamsPrecondition
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := c.format(v, nil, module, d); e != nil {
		if errors.Is(e, ErrParamsStreamFailure) {
			return string(c.workspace), e
		}
		return "", e
	}
	return string(c.workspace), nil
}
func (c *ParamsContext) appendDescrs(d *ParamDescrs) error {
	var names []Symbol
	defer func() { c.budget.Release(pdNameBytes(cap(names))) }()
	for _, x := range d.entries {
		if x.state != 2 {
			continue
		}
		if len(names) == cap(names) {
			n, e := pdGrowth(cap(names))
			if e != nil {
				return e
			}
			if e = c.budget.Reserve(pdNameBytes(n)); e != nil {
				return e
			}
			next := make([]Symbol, len(names), n)
			copy(next, names)
			c.budget.Release(pdNameBytes(cap(names)))
			names = next
		}
		names = append(names, x.key)
	}
	for _, k := range names {
		if _, e := d.symbolCompare(k, k); e != nil {
			return e
		}
		i, e := d.find(k)
		if e != nil {
			return e
		}
		if d.entries[i].info.descr.ref == (arena.VectorRef{}) {
			return ErrParamDescrsPrecondition
		}
	}
	if e := d.sortSymbols(names); e != nil {
		return e
	}
	for _, k := range names {
		if e := c.appendString("  "); e != nil {
			return e
		}
		if e := c.appendSymbol(k, false, true); e != nil {
			return e
		}
		i, e := d.find(k)
		if e != nil {
			return e
		}
		info := d.entries[i].info
		if e = c.appendString(" ("); e != nil {
			return e
		}
		if e = c.appendString(info.kind.String()); e != nil {
			return e
		}
		if e = c.appendByte(')'); e != nil {
			return e
		}
		if info.def.ref != (arena.VectorRef{}) {
			if e = c.appendString(" (default: "); e != nil {
				return e
			}
			if e = c.appendText(d.texts, info.def.ref, info.def.length); e != nil {
				return e
			}
			if e = c.appendByte(')'); e != nil {
				return e
			}
		}
		if e = c.appendByte('\n'); e != nil {
			return e
		}
	}
	return nil
}
func (c *ParamsContext) Validate(v *ParamsRef, d *ParamDescrs) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.check(v); e != nil {
		return e
	}
	if d == nil || d.manager != c.symbols {
		return ErrParamsOwner
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return e
	}
	if v.slot == 0 {
		return nil
	}
	b := &c.blocks[v.slot-1]
	for i := uint32(0); i < b.size; i++ {
		x := c.entry(b, i)
		expected, e := paramsDescrKindInModule(d, &x.key)
		if e != nil {
			return e
		}
		if expected == x.kind || (x.kind == ParamUInt && expected == ParamNumeral) {
			continue
		}
		c.workspace = c.workspace[:0]
		if expected == ParamInvalid {
			if e = c.appendString("unknown parameter '"); e != nil {
				return e
			}
			if e = c.appendSymbol(x.key, true, false); e != nil {
				return e
			}
			if e = c.appendString("'\nLegal parameters are:\n"); e != nil {
				return e
			}
			e = c.appendDescrs(d)
			if e != nil {
				return e
			}
		} else {
			if e = c.appendString("Parameter "); e != nil {
				return e
			}
			if e = c.appendSymbol(x.key, true, false); e != nil {
				return e
			}
			if e = c.appendString(" was given argument of type "); e != nil {
				return e
			}
			if e = c.appendString(x.kind.String()); e != nil {
				return e
			}
			if e = c.appendString(", expected "); e != nil {
				return e
			}
			if e = c.appendString(expected.String()); e != nil {
				return e
			}
		}
		return NewDefaultException(string(c.workspace))
	}
	return nil
}
func (c *ParamsContext) Stats() ParamsStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	var s ParamsStats
	if c.closed {
		return s
	}
	s.HeaderBytes = c.headerBytes
	s.BlockBytes = paramsBlockBytes(cap(c.blocks))
	s.DirectoryBytes = paramsDirectoryBytes(cap(c.pages))
	s.FormatBytes = uint64(cap(c.workspace))
	s.Text = c.texts.Stats()
	for _, p := range c.pages {
		s.PageBytes += paramsPageBytes(p)
	}
	for _, b := range c.blocks {
		if b.live {
			s.LiveBlocks++
			s.References += b.refs
			s.Entries += b.size
		}
	}
	return s
}
func (c *ParamsContext) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.self != c {
		return ErrParamsOwner
	}
	if c.closed {
		return nil
	}
	for i := range c.blocks {
		if c.blocks[i].live {
			if e := c.resetBlock(&c.blocks[i]); e != nil {
				return e
			}
		}
	}
	c.texts.Close()
	for _, p := range c.pages {
		c.budget.Release(paramsPageBytes(p))
	}
	c.budget.Release(paramsDirectoryBytes(cap(c.pages)) + paramsBlockBytes(cap(c.blocks)) + uint64(cap(c.workspace)) + c.headerBytes)
	c.blocks = nil
	c.pages = nil
	c.workspace = nil
	c.closed = true
	return nil
}

// DisplayKeyName constructs the source temporary symbol even with null backing.
func (c *ParamsContext) DisplayKeyName(v *ParamsRef, name *string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.check(v); e != nil {
		return "", e
	}
	k, e := c.symbols.InternNullable(name)
	if e != nil {
		return "", e
	}
	if e = c.format(v, &k, nil, nil); e != nil {
		if errors.Is(e, ErrParamsStreamFailure) {
			return string(c.workspace), e
		}
		return "", e
	}
	return string(c.workspace), nil
}
func (c *ParamsContext) FormatKeyInto(v *ParamsRef, k Symbol, dst []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.format(v, &k, nil, nil); e != nil {
		return 0, e
	}
	if len(dst) < len(c.workspace) {
		return len(c.workspace), mpnum.ErrShortBuffer
	}
	return copy(dst, c.workspace), nil
}
func (c *ParamsContext) FormatSMT2Into(v *ParamsRef, module *string, d *ParamDescrs, dst []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if d == nil {
		return 0, ErrParamsPrecondition
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := c.format(v, nil, module, d); e != nil {
		return 0, e
	}
	if len(dst) < len(c.workspace) {
		return len(c.workspace), mpnum.ErrShortBuffer
	}
	return copy(dst, c.workspace), nil
}

// Caller holds d.mu throughout validation, including prefix publication.
func paramsDescrKindInModule(d *ParamDescrs, name *Symbol) (ParamKind, error) {
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

// DebugString is the source Z3DEBUG pp text boundary, including its newline.
func (c *ParamsContext) DebugString(v *ParamsRef) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.format(v, nil, nil, nil); e != nil {
		if errors.Is(e, ErrParamsStreamFailure) {
			return string(c.workspace), e
		}
		return "", e
	}
	if e := c.appendByte('\n'); e != nil {
		return "", e
	}
	return string(c.workspace), nil
}

// Check the actual host C-prefix length before any uint32 narrowing.
func paramsCheckedTextLength(n uint64) (uint32, error) {
	if n > math.MaxUint32 {
		return 0, arena.ErrBounds
	}
	return uint32(n), nil
}
