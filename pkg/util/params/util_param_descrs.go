// Source: src/util/params.cpp:59-86; src/util/params.h.
// Concrete translation of util/params.cpp param_descrs and dictionary<info>.
package params

import (
	"errors"
	"reflect"
	"strings"
	"sync"

	"github.com/glycerine/g3/pkg/arena"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	utext "github.com/glycerine/g3/pkg/util/text"
	"github.com/glycerine/insdict"
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

type ParamDescrsConfig struct {
	Budget          *arena.Budget
	InitialCapacity uint32
	Text            arena.VectorConfig
}
type paramDescrText struct {
	value   string
	present bool
}
type paramDescrInfo struct {
	kind               ParamKind
	descr, def, module paramDescrText
}
type ParamDescrsStats struct {
	Live, Deleted, Capacity, Names, NamesCapacity uint32
	OwnerBytes, TableBytes, NamesBytes            uint64
}

// Symbols are borrowed from one manager, which must outlive this owner and may
// not be torn down concurrently. Metadata retains immutable Go strings,
// truncated at the first NUL; nil and present-empty are distinct.
// The cold mutex protects this owner; a shared external Budget requires caller
// serialization. Owning Go values must not be shallow-copied.
type ParamDescrs struct {
	mu                   sync.Mutex
	Self                 *ParamDescrs
	manager              *usymbol.Manager
	Budget               *arena.Budget
	entries              *insdict.Dict[usymbol.Symbol, paramDescrInfo]
	Names                []usymbol.Symbol
	closed               bool
	constructorReady     bool
	constructorCompleted uint8
	constructorCause     error
	OwnerBytes           uint64
}

func prepareSourceParamDescrs(m *usymbol.Manager, c ParamDescrsConfig) (ParamDescrsConfig, error) {
	if m == nil {
		return c, ErrParamDescrsPrecondition
	}
	if _, e := m.Hash(usymbol.Symbol{}); e != nil {
		return c, e
	}
	n := c.InitialCapacity
	if n == 0 {
		n = 8
	}
	if uint64(n) > uint64(^uint(0)>>1) {
		return c, arena.ErrBounds
	}
	if c.Text.Descriptors.Budget != nil && c.Text.Descriptors.Budget != c.Budget {
		return c, ErrParamDescrsPrecondition
	}
	c.InitialCapacity = n
	return c, nil
}
func newSourceParamDescrs(m *usymbol.Manager, c ParamDescrsConfig) (*ParamDescrs, error) {
	c, e := prepareSourceParamDescrs(m, c)
	if e != nil {
		return nil, e
	}
	owner := uint64(reflect.TypeOf(ParamDescrs{}).Size())
	if e := c.Budget.Reserve(owner); e != nil {
		return nil, e
	}
	d := &ParamDescrs{manager: m, Budget: c.Budget, OwnerBytes: owner}
	d.Self = d
	if e := d.constructSourceMembers(c); e != nil {
		return d, e
	}
	return d, nil
}

// Same placement member engine; embedded callers charge the actual containing header.
func (d *ParamDescrs) constructSourceMembers(c ParamDescrsConfig) error {
	d.entries = insdict.NewDictFuncSize[usymbol.Symbol, paramDescrInfo](func(k usymbol.Symbol) uint64 {
		words := k.SymbolWords()
		h := ((uint64(words[0]) | uint64(words[1])<<32) ^ uint64(k.Value)<<32) | uint64(k.Tag)
		return h ^ uint64(k.Generation)*0x9e3779b97f4a7c15
	}, int(c.InitialCapacity))
	d.constructorCompleted = 1 // actual info dictionary completed
	d.Names = nil              // source null names vector default construction cannot fail
	d.constructorCompleted = 2
	d.constructorReady = true
	return nil
}
func (d *ParamDescrs) open() error {
	if d == nil || d.Self != d {
		return ErrParamDescrsPrecondition
	}
	if d.closed {
		return arena.ErrClosed
	}
	if !d.constructorReady {
		return ErrParamDescrsPrecondition
	}
	_, e := d.manager.Hash(usymbol.Symbol{})
	return e
}
func (d *ParamDescrs) find(k usymbol.Symbol) (paramDescrInfo, bool, error) {
	_, e := d.manager.Hash(k)
	if e != nil {
		return paramDescrInfo{}, false, e
	}
	v, ok := d.entries.Get2(k)
	return v, ok, nil
}
func paramDescrString(s *string) paramDescrText {
	if s == nil {
		return paramDescrText{}
	}
	return paramDescrText{value: (*s)[:utext.CStringLength(*s)], present: true}
}
func (d *ParamDescrs) load(r paramDescrText) (string, bool, error) {
	return r.value, r.present, nil
}
func (d *ParamDescrs) prepareInsert(k usymbol.Symbol, kind ParamKind) (bool, error) {
	_, e := d.manager.Hash(k)
	if e != nil {
		return false, e
	}
	if k.IsNumerical() {
		return false, ErrParamDescrsPrecondition
	}
	if info, found := d.entries.Get2(k); found {
		if info.kind != kind {
			return false, ErrParamDescrsPrecondition
		}
		return true, nil
	}
	return false, nil
}
func (d *ParamDescrs) publishInsert(k usymbol.Symbol, info paramDescrInfo) error {
	// Preserve source insertion publication before names.push_back. A names
	// history records the first successful insertion.
	d.entries.Put(k, info)
	if uint64(len(d.Names)) >= uint64(^uint32(0)) {
		d.entries.Del(k)
		return arena.ErrBounds
	}
	d.Names = append(d.Names, k)
	return nil
}
func (d *ParamDescrs) insert(k usymbol.Symbol, kind ParamKind, descr, def, module *string) error {
	existing, e := d.prepareInsert(k, kind)
	if e != nil || existing {
		return e
	}
	info := paramDescrInfo{
		kind: kind, descr: paramDescrString(descr),
		def: paramDescrString(def), module: paramDescrString(module),
	}
	return d.publishInsert(k, info)
}
func (d *ParamDescrs) insertCopy(key usymbol.Symbol, info paramDescrInfo) error {
	existing, e := d.prepareInsert(key, info.kind)
	if e != nil || existing {
		return e
	}
	return d.publishInsert(key, info)
}
func (d *ParamDescrs) Insert(k usymbol.Symbol, kind ParamKind, descr, def, module *string) error {
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
func (d *ParamDescrs) contains(k usymbol.Symbol) (bool, error) {
	_, found, e := d.find(k)
	return found, e
}
func (d *ParamDescrs) Contains(k usymbol.Symbol) (bool, error) {
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
func (d *ParamDescrs) erase(k usymbol.Symbol) error {
	_, found, e := d.find(k)
	if e != nil || !found {
		return e
	}
	d.entries.Del(k)
	return nil
}
func (d *ParamDescrs) Erase(k usymbol.Symbol) error {
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
func (d *ParamDescrs) kind(k usymbol.Symbol) (ParamKind, error) {
	info, found, e := d.find(k)
	if e != nil || !found {
		return ParamInvalid, e
	}
	return info.kind, nil
}
func (d *ParamDescrs) Kind(k usymbol.Symbol) (ParamKind, error) {
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
func (d *ParamDescrs) text(k usymbol.Symbol, which uint32) (string, bool, error) {
	info, found, e := d.find(k)
	if e != nil || !found {
		return "", false, e
	}
	r := info.descr
	if which == 1 {
		r = info.def
	}
	if which == 2 {
		r = info.module
	}
	return d.load(r)
}
func (d *ParamDescrs) getText(k usymbol.Symbol, which uint32) (string, bool, error) {
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
func (d *ParamDescrs) Description(k usymbol.Symbol) (string, bool, error) { return d.getText(k, 0) }
func (d *ParamDescrs) Default(k usymbol.Symbol) (string, bool, error)     { return d.getText(k, 1) }
func (d *ParamDescrs) Module(k usymbol.Symbol) (string, bool, error)      { return d.getText(k, 2) }
func (d *ParamDescrs) DescriptionName(n string) (string, bool, error)     { return d.getTextName(n, 0) }
func (d *ParamDescrs) DefaultName(n string) (string, bool, error)         { return d.getTextName(n, 1) }
func (d *ParamDescrs) ModuleName(n string) (string, bool, error)          { return d.getTextName(n, 2) }
func (d *ParamDescrs) Size() (uint32, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return 0, e
	}
	return uint32(len(d.Names)), nil
}
func (d *ParamDescrs) ParamName(i uint32) (usymbol.Symbol, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return usymbol.Symbol{}, e
	}
	if uint64(i) >= uint64(len(d.Names)) {
		return usymbol.Symbol{}, arena.ErrBounds
	}
	return d.Names[i], nil
}

// Checked spelling byte access does not allocate a borrowed spelling copy.
func (d *ParamDescrs) symbolByte(k usymbol.Symbol, index uint32) (byte, error) {
	m := d.manager
	if e := m.Validate(k); e != nil {
		return 0, e
	}
	if k.Value == 0 || k.IsNumerical() {
		return 0, ErrParamDescrsPrecondition
	}
	text, _, e := m.BareString(k)
	if e != nil {
		return 0, e
	}
	if uint64(index) > uint64(len(text)) {
		return 0, arena.ErrBounds
	}
	if uint64(index) == uint64(len(text)) {
		return 0, nil
	}
	return text[index], nil
}
func (d *ParamDescrs) internBytes(b []byte) (usymbol.Symbol, error) {
	return d.manager.Intern(string(b))
}
func (d *ParamDescrs) internSymbolRange(k usymbol.Symbol, start, n uint32) (usymbol.Symbol, error) {
	if uint64(n) > uint64(^uint(0)>>1) {
		return usymbol.Symbol{}, arena.ErrBounds
	}
	if n == 0 {
		return d.manager.Intern("")
	}
	b := make([]byte, n)
	for i := uint32(0); i < n; i++ {
		v, e := d.symbolByte(k, start+i)
		if e != nil {
			return usymbol.Symbol{}, e
		}
		b[i] = v
	}
	return d.internBytes(b)
}
func (d *ParamDescrs) internText(r paramDescrText) (usymbol.Symbol, error) {
	if !r.present {
		return usymbol.Symbol{}, nil
	}
	return d.manager.Intern(r.value)
}
func (d *ParamDescrs) splitName(name usymbol.Symbol) (usymbol.Symbol, usymbol.Symbol, bool, error) {
	n, e := d.manager.Size(name)
	if e != nil {
		return usymbol.Symbol{}, usymbol.Symbol{}, false, e
	}
	dot := uint32(0)
	for dot < n {
		ch, e := d.symbolByte(name, dot)
		if e != nil {
			return usymbol.Symbol{}, usymbol.Symbol{}, false, e
		}
		if ch == '.' {
			break
		}
		dot++
	}
	if dot == n {
		return usymbol.Symbol{}, usymbol.Symbol{}, false, nil
	}
	prefixBytes := make([]byte, dot)
	for i := uint32(0); i < dot; i++ {
		prefixBytes[i], e = d.symbolByte(name, i)
		if e != nil {
			return usymbol.Symbol{}, usymbol.Symbol{}, false, e
		}
	}
	prefix, e := d.internBytes(prefixBytes)
	if e != nil {
		return usymbol.Symbol{}, usymbol.Symbol{}, false, e
	}
	suffix, e := d.internSymbolRange(name, dot+1, n-dot-1)
	if e != nil {
		return usymbol.Symbol{}, usymbol.Symbol{}, false, e
	}
	return prefix, suffix, true, nil
}
func (d *ParamDescrs) KindInModule(name *usymbol.Symbol) (ParamKind, error) {
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
	if *name == (usymbol.Symbol{}) {
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
	info, found, e := d.find(suffix)
	if e != nil {
		return ParamInvalid, e
	}
	if !found {
		return ParamInvalid, ErrParamDescrsPrecondition
	}
	module, e := d.internText(info.module)
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
func (d *ParamDescrs) symbolCompare(a, b usymbol.Symbol) (int, error) {
	m := d.manager
	if e := m.Validate(a); e != nil {
		return 0, e
	}
	if e := m.Validate(b); e != nil {
		return 0, e
	}
	if a.Value == 0 || b.Value == 0 || a.IsNumerical() || b.IsNumerical() {
		return 0, ErrParamDescrsPrecondition
	}
	x, _, e := m.BareString(a)
	if e != nil {
		return 0, e
	}
	y, _, e := m.BareString(b)
	if e != nil {
		return 0, e
	}
	return strings.Compare(x, y), nil
}

// Concrete in-place sort realizes std::sort's comparator/output contract;
// it has no reflection swapper or additional heap-backed work arrays.
func (d *ParamDescrs) siftSymbols(a []usymbol.Symbol, root, end int) error {
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
func (d *ParamDescrs) sortSymbols(a []usymbol.Symbol) error {
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
func (d *ParamDescrs) writeName(out *strings.Builder, k usymbol.Symbol, smt2 bool) error {
	m := d.manager
	if e := m.Validate(k); e != nil {
		return e
	}
	if k.Value == 0 || k.IsNumerical() {
		return ErrParamDescrsPrecondition
	}
	text, _, e := m.BareString(k)
	if e != nil {
		return e
	}
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if smt2 && ch == '_' {
			ch = '-'
		} else if !smt2 && ch == '-' {
			ch = '_'
		} else if ch >= 'A' && ch <= 'Z' {
			ch = ch - 'A' + 'a'
		}
		out.WriteByte(ch)
	}
	return nil
}
func (d *ParamDescrs) writeText(out *strings.Builder, r paramDescrText) error {
	out.WriteString(r.value)
	return nil
}
func (d *ParamDescrs) Display(indent uint32, smt2, includeDescr bool) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return "", e
	}
	var names []usymbol.Symbol
	for key := range d.entries.All() {
		names = append(names, key)
	}
	for _, k := range names {
		if _, e := d.symbolCompare(k, k); e != nil {
			return "", e
		}
		info, found, e := d.find(k)
		if e != nil {
			return "", e
		}
		if !found {
			return "", ErrParamDescrsPrecondition
		}
		if paramDescrsStreamSourceDebug && !info.descr.present {
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
		info, found, e := d.find(key)
		if e != nil {
			return "", e
		}
		if !found {
			return "", ErrParamDescrsPrecondition
		}
		out.WriteString(" (")
		out.WriteString(info.kind.String())
		out.WriteByte(')')
		if includeDescr {
			out.WriteByte(' ')
			if !info.descr.present {
				return out.String(), utext.ErrNullCString
			}
			if e = d.writeText(&out, info.descr); e != nil {
				return "", e
			}
		}
		if info.def.present {
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
	for key, info := range other.entries.All() {
		if e := d.insertCopy(key, info); e != nil {
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
	live := uint32(0)
	if d.entries != nil {
		live = uint32(d.entries.Len())
	}
	return ParamDescrsStats{Live: live, Names: uint32(len(d.Names)), NamesCapacity: uint32(cap(d.Names)), OwnerBytes: d.OwnerBytes}
}
func (d *ParamDescrs) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.Self != d {
		return ErrParamDescrsPrecondition
	}
	if d.closed {
		return nil
	}
	if !d.constructorReady {
		return ErrParamDescrsPrecondition
	}
	d.Budget.Release(d.OwnerBytes)
	d.OwnerBytes = 0
	d.entries = nil
	d.Names = nil
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
// order. Immutable text shares the Go lifetime of its dictionary entries.
func (d *ParamDescrs) sourceFinalizeMembers() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.Self != d {
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
	d.Names = nil
	d.entries = nil
	d.Budget.Release(d.OwnerBytes)
	d.OwnerBytes = 0
	d.closed = true
	return nil
}

// InitEmbeddedSource preserves placement-new storage and failed member prefixes.
func (d *ParamDescrs) InitEmbeddedSource(m *usymbol.Manager, c ParamDescrsConfig) error {
	cfg, e := prepareSourceParamDescrs(m, c)
	if e != nil {
		return e
	}
	*d = ParamDescrs{manager: m, Budget: cfg.Budget}
	d.Self = d
	return d.constructSourceMembers(cfg)
}
func (d *ParamDescrs) ConstructorReady() bool       { return d.constructorReady }
func (d *ParamDescrs) Initialized() bool            { return d.Self != nil }
func (d *ParamDescrs) Closed() bool                 { return d.closed }
func (d *ParamDescrs) SourceFinalizeMembers() error { return d.sourceFinalizeMembers() }
func (d *ParamDescrs) AbortConstructor() error      { return d.abortConstructor() }
