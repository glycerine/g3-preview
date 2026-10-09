// Source: src/util/symbol.h; src/util/symbol.cpp.
package util

import (
	"errors"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrInvalidSymbol = errors.New("g3: invalid source symbol precondition")
var ErrSymbolOwner = errors.New("g3: symbol belongs to another owner")

// Symbol is a pointer-free source symbol identity. Tag0 is a string or null,
// tag1 a uint32 numerical value, and tags2/3 the source marked forms. Physical
// IDs replace addresses; owner and IDs are never recycled. Generation remains
// in the encoded handle for compatibility and is validated on access.
type Symbol struct {
	owner      uint64
	Value      uint32
	Generation arena.Generation
	Tag        uint32
}

func NumericalSymbol(value uint32) Symbol { return Symbol{Value: value, Tag: 1} }
func DummySymbol() Symbol                 { return Symbol{Tag: 2} }
func (s Symbol) IsNumerical() bool        { return s.Tag == 1 }
func (s Symbol) IsMarked() bool           { return s.Tag > 1 }
func (s Symbol) Number() (uint32, error) {
	if !s.IsNumerical() {
		return 0, ErrInvalidSymbol
	}
	return s.Value, nil
}

// Marking is private in the source and is used by the scoped symbol table.
func (s Symbol) Mark() (Symbol, error) {
	if s.IsMarked() {
		return Symbol{}, ErrInvalidSymbol
	}
	s.Tag += 2
	return s, nil
}
func (s Symbol) unmark() (Symbol, error) {
	if !s.IsMarked() || s.Tag > 3 {
		return Symbol{}, ErrInvalidSymbol
	}
	s.Tag -= 2
	return s, nil
}

// SymbolToken replaces the source raw pointer conversion; it is opaque and
// comparable, and its string identity can only be restored by its live owner.
type SymbolToken struct{ symbol Symbol }

func (s Symbol) Token() SymbolToken { return SymbolToken{symbol: s} }

type SymbolConfig struct {
	Budget *arena.Budget
	// Legacy arena geometry is ignored: spellings use Go-managed storage.
	PageSize, Alignment, DescriptorPageSize uint32
	DirectOnly                              bool
}
type SymbolStats struct {
	OwnerBytes uint64
	Symbols    uint32
	Region     arena.BytesRegionStats
	Table      SymbolInternStats
}

// SymbolInternStats retains the existing accounting view while interning is
// implemented by Go maps. Map storage is managed by the Go runtime.
type SymbolInternStats struct {
	OwnerBytes, TableBytes, DirectoryBytes, RetainedBytes uint64
	Size, Deleted, Capacity                               uint32
}

// SymbolManager owns canonical immutable Go spellings. IDs are never recycled
// within an owner; the mutex serializes interning and owner teardown.
type SymbolManager struct {
	mu      sync.Mutex
	byText  map[string]uint32
	entries []symbolEntry
	Budget  *arena.Budget
	owner   uint64
	closed  bool
}
type symbolEntry struct {
	text string
	hash uint32
}

var symbolOwnerSequence atomic.Uint64

func NewSymbolManager(c SymbolConfig) (*SymbolManager, error) {
	header := uint64(reflect.TypeOf(SymbolManager{}).Size())
	if e := c.Budget.Reserve(header); e != nil {
		return nil, e
	}
	// Saturate instead of recycling identities on counter exhaustion.
	var owner uint64
	for {
		previous := symbolOwnerSequence.Load()
		if previous == math.MaxUint64 {
			c.Budget.Release(header)
			return nil, arena.ErrBounds
		}
		if symbolOwnerSequence.CompareAndSwap(previous, previous+1) {
			owner = previous + 1
			break
		}
	}
	return &SymbolManager{byText: make(map[string]uint32), Budget: c.Budget, owner: owner}, nil
}
func cSymbolLength(text string) int {
	if i := strings.IndexByte(text, 0); i >= 0 {
		return i
	}
	return len(text)
}
func (m *SymbolManager) check(s Symbol) error {
	if m.closed {
		return arena.ErrClosed
	}
	if s.Tag > 3 {
		return ErrInvalidSymbol
	}
	if s.Tag == 1 || s.Tag == 3 {
		if s.owner != 0 || s.Generation != 0 {
			return ErrInvalidSymbol
		}
		return nil
	}
	if s.Value == 0 {
		if s.owner != 0 || s.Generation != 0 {
			return ErrInvalidSymbol
		}
		return nil
	}
	if s.owner != m.owner {
		return ErrSymbolOwner
	}
	if s.Generation != 1 || uint64(s.Value) > uint64(len(m.entries)) {
		return arena.ErrInvalidSlot
	}
	return nil
}
func (m *SymbolManager) checkPlain(s Symbol) error {
	if e := m.check(s); e != nil {
		return e
	}
	if s.IsMarked() {
		return ErrInvalidSymbol
	}
	return nil
}
func (m *SymbolManager) symbol(id uint32) Symbol {
	return Symbol{owner: m.owner, Value: id, Generation: 1}
}

// Intern canonicalizes the C-string spelling using a Go map. The typed entry
// retains the source hash without reproducing its raw size_t prefix or NUL.
func (m *SymbolManager) Intern(text string) (Symbol, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.intern(text)
}

func (m *SymbolManager) intern(text string) (Symbol, error) {
	if m.closed {
		return Symbol{}, arena.ErrClosed
	}
	length := cSymbolLength(text)
	text = text[:length]
	if r, found := m.byText[text]; found {
		return m.symbol(r), nil
	}
	if uint64(length) > math.MaxUint32 || uint64(len(m.entries)) == math.MaxUint32 {
		return Symbol{}, arena.ErrBounds
	}
	// Own only the spelling, rather than retaining a possibly large input suffix.
	text = strings.Clone(text)
	id := uint32(len(m.entries)) + 1
	m.entries = append(m.entries, symbolEntry{text: text, hash: String(text, 17)})
	m.byText[text] = id
	return m.symbol(id), nil
}

// A null constructor needs no interned spelling, as in the source.
func (m *SymbolManager) InternNullable(text *string) (Symbol, error) {
	if text == nil {
		return Symbol{}, nil
	}
	return m.Intern(*text)
}
func (m *SymbolManager) AssignString(out *Symbol, text *string) error {
	if out == nil || text == nil {
		return ErrInvalidSymbol
	}
	s, e := m.Intern(*text)
	if e != nil {
		return e
	}
	*out = s
	return nil
}
func (m *SymbolManager) FromToken(t SymbolToken) (Symbol, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.check(t.symbol); e != nil {
		return Symbol{}, e
	}
	return t.symbol, nil
}
func (m *SymbolManager) spelling(s Symbol) (string, error) {
	if s.Value == 0 || uint64(s.Value) > uint64(len(m.entries)) || s.Generation != 1 {
		return "", arena.ErrInvalidSlot
	}
	return m.entries[s.Value-1].text, nil
}
func (m *SymbolManager) text(s Symbol, display bool) (string, error) {
	if e := m.checkPlain(s); e != nil {
		return "", e
	}
	if s.IsNumerical() {
		return "k!" + strconv.FormatInt(int64(int32(s.Value)), 10), nil
	}
	if s.Value == 0 {
		if display {
			return "null", nil
		}
		return "<null>", nil
	}
	return m.spelling(s)
}
func (m *SymbolManager) String(s Symbol) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.text(s, false)
}
func (m *SymbolManager) Display(s Symbol) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.text(s, true)
}

// BareString preserves source nullability through present=false. Its immutable
// Go string remains safe to use after the manager closes.
func (m *SymbolManager) BareString(s Symbol) (text string, present bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.checkPlain(s); e != nil {
		return "", false, e
	}
	if s.IsNumerical() {
		return "", false, ErrInvalidSymbol
	}
	if s.Value == 0 {
		return "", false, nil
	}
	text, err = m.spelling(s)
	return text, err == nil, err
}
func (m *SymbolManager) Hash(s Symbol) (uint32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.checkPlain(s); e != nil {
		return 0, e
	}
	if s.IsNumerical() {
		return s.Value, nil
	}
	if s.Value == 0 {
		return 0x9e3779d9, nil
	}
	return m.entries[s.Value-1].hash, nil
}
func (m *SymbolManager) Contains(s Symbol, ch byte) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.checkPlain(s); e != nil {
		return false, e
	}
	if s.IsNumerical() {
		return false, nil
	}
	if s.Value == 0 {
		return false, ErrInvalidSymbol
	}
	text, e := m.spelling(s)
	if e != nil {
		return false, e
	}
	return ch == 0 || strings.IndexByte(text, ch) >= 0, nil
}
func (m *SymbolManager) Size(s Symbol) (uint32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.checkPlain(s); e != nil {
		return 0, e
	}
	if s.IsNumerical() {
		n, v := uint32(4), s.Value>>1
		for v > 0 {
			n++
			v >>= 1
		}
		return n, nil
	}
	if s.Value == 0 {
		return 0, ErrInvalidSymbol
	}
	text, e := m.spelling(s)
	return uint32(len(text)), e
}
func (m *SymbolManager) EqualString(s Symbol, text *string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.checkPlain(s); e != nil {
		return false, e
	}
	if s.Tag == 0 && s.Value == 0 {
		return text == nil, nil
	}
	if text == nil {
		return false, nil
	}
	v := (*text)[:cSymbolLength(*text)]
	if s.IsNumerical() {
		rendered, e := m.text(s, false)
		return rendered == v, e
	}
	spelling, e := m.spelling(s)
	if e != nil {
		return false, e
	}
	return spelling == v, nil
}
func (m *SymbolManager) Less(a, b Symbol) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.check(a); e != nil {
		return false, e
	}
	if e := m.check(b); e != nil {
		return false, e
	}
	if a == b {
		return false, nil
	}
	if a.IsMarked() || b.IsMarked() {
		return false, ErrInvalidSymbol
	}
	if a.IsNumerical() {
		if !b.IsNumerical() {
			return true, nil
		}
		return a.Value < b.Value, nil
	}
	if b.IsNumerical() {
		return false, nil
	}
	if a.Value == 0 || b.Value == 0 {
		return false, ErrInvalidSymbol
	}
	x, e := m.spelling(a)
	if e != nil {
		return false, e
	}
	y, e := m.spelling(b)
	if e != nil {
		return false, e
	}
	for i := 0; ; i++ {
		var p, q byte
		if i < len(x) {
			p = x[i]
		}
		if i < len(y) {
			q = y[i]
		}
		if int8(p) < int8(q) {
			return true, nil
		}
		if p != q {
			return false, nil
		}
		if i+1 >= len(x) {
			return true, nil
		}
		if i+1 >= len(y) {
			return false, nil
		}
	}
}
func (m *SymbolManager) Stats() SymbolStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := SymbolStats{}
	if !m.closed {
		s.OwnerBytes = uint64(reflect.TypeOf(SymbolManager{}).Size())
		s.Symbols = uint32(len(m.byText))
		s.Table.Size = s.Symbols
	}
	return s
}
func (m *SymbolManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.Budget.Release(uint64(reflect.TypeOf(SymbolManager{}).Size()))
	m.byText = nil
	m.entries = nil
	m.closed = true
	return nil
}
