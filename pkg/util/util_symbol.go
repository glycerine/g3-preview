// Translation of src/util/symbol.h/.cpp, Leonardo de Moura, 2006-09-11.
package util

import (
	"errors"
	"math"
	"reflect"
	"strconv"
	"sync"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrInvalidSymbol = errors.New("g3: invalid source symbol precondition")
var ErrSymbolOwner = errors.New("g3: symbol belongs to another owner")

// Symbol is a pointer-free source symbol identity. Tag0 is a string or null,
// tag1 a uint32 numerical value, and tags2/3 the source marked forms. Physical
// slots and generations replace addresses; owner is never recycled.
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
	Budget                                  *arena.Budget
	PageSize, Alignment, DescriptorPageSize uint32
	DirectOnly                              bool
}
type SymbolStats struct {
	OwnerBytes uint64
	Symbols    uint32
	Region     arena.BytesRegionStats
	Table      StrHashtableStats
}

// SymbolManager explicitly owns the source global interning service. It has no
// reset: all spelling identities survive until Close. The cold mutex preserves
// source cr_symbol serialization; concurrent owner teardown is serialized too.
// Returned Go strings are caller-owned outputs, not retained manager storage.
type SymbolManager struct {
	mu           sync.Mutex
	region       *arena.BytesRegion
	byText       map[string]arena.BytesRegionRef
	Budget       *arena.Budget
	owner        uint64
	pointerWidth uint32
	closed       bool
}

func NewSymbolManager(c SymbolConfig) (*SymbolManager, error) {
	header := uint64(reflect.TypeOf(SymbolManager{}).Size())
	if e := c.Budget.Reserve(header); e != nil {
		return nil, e
	}
	width := c.Alignment
	if width == 0 {
		width = 8
	}
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{PageSize: c.PageSize, Alignment: width, DirectOnly: c.DirectOnly, Descriptors: arena.Config{PageSize: c.DescriptorPageSize, Budget: c.Budget}})
	if e != nil {
		c.Budget.Release(header)
		return nil, e
	}
	return &SymbolManager{region: r, byText: make(map[string]arena.BytesRegionRef), Budget: c.Budget, owner: r.OwnerID(), pointerWidth: width}, nil
}
func cSymbolLength(text string) int {
	for i := 0; i < len(text); i++ {
		if text[i] == 0 {
			return i
		}
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
	if !m.region.IsLive(arena.BytesRegionRef{Slot: arena.Slot(s.Value), Generation: s.Generation}) {
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
func (m *SymbolManager) symbol(r arena.BytesRegionRef) Symbol {
	return Symbol{owner: m.owner, Value: uint32(r.Slot), Generation: r.Generation}
}

// Intern performs source threshold growth before duplicate lookup. On a miss,
// it allocates hash-prefix+spelling+NUL, writes the source cached hash, then
// publishes the borrowed region key. Unlike the source temporary query entry,
// no caller string is published before owned spelling allocation succeeds.
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
	wide := uint64(length) + 1 + uint64(m.pointerWidth)
	if wide > math.MaxUint32 {
		return Symbol{}, arena.ErrBounds
	}
	r, e := m.region.Allocate(uint32(wide))
	if e != nil {
		return Symbol{}, e
	}
	hash := String(text, 17)
	for i := uint32(0); i < m.pointerWidth; i++ {
		b := byte(0)
		if i < 4 {
			b = byte(hash >> (i * 8))
		}
		if e := m.region.Write(r, i, b); e != nil {
			return Symbol{}, e
		}
	}
	for i := 0; i < length; i++ {
		if e := m.region.Write(r, m.pointerWidth+uint32(i), text[i]); e != nil {
			return Symbol{}, e
		}
	}
	if e := m.region.Write(r, m.pointerWidth+uint32(length), 0); e != nil {
		return Symbol{}, e
	}
	m.byText[text] = r
	return m.symbol(r), nil
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
func (m *SymbolManager) key(s Symbol) (StrKey, error) {
	return BorrowCString(m.region, arena.BytesRegionRef{Slot: arena.Slot(s.Value), Generation: s.Generation}, m.pointerWidth)
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
	k, e := m.key(s)
	if e != nil {
		return "", e
	}
	return m.symbolText(k)
}

func (m *SymbolManager) symbolText(k StrKey) (string, error) {
	b := make([]byte, k.Length())
	for i := range b {
		v, e := m.region.Read(k.Allocation(), k.Offset()+uint32(i))
		if e != nil {
			return "", e
		}
		b[i] = v
	}
	return string(b), nil
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

// BareString preserves the source nullable bare pointer through present=false
// for null. Its text is a caller-owned copy, never a borrowed backing slice.
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
	k, e := m.key(s)
	if e != nil {
		return "", false, e
	}
	text, e = m.symbolText(k)
	return text, e == nil, e
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
	r := arena.BytesRegionRef{Slot: arena.Slot(s.Value), Generation: s.Generation}
	var hash uint32
	for i := uint32(0); i < 4; i++ {
		b, e := m.region.Read(r, i)
		if e != nil {
			return 0, e
		}
		hash |= uint32(b) << (8 * i)
	}
	return hash, nil
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
	k, e := m.key(s)
	if e != nil {
		return false, e
	}
	for i := uint32(0); i <= k.Length(); i++ {
		b, e := m.region.Read(k.Allocation(), k.Offset()+i)
		if e != nil {
			return false, e
		}
		if b == ch {
			return true, nil
		}
	}
	return false, nil
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
	k, e := m.key(s)
	return k.Length(), e
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
	k, e := m.key(s)
	if e != nil {
		return false, e
	}
	if uint64(k.Length()) != uint64(len(v)) {
		return false, nil
	}
	for i := uint32(0); i < k.Length(); i++ {
		b, e := m.region.Read(k.Allocation(), k.Offset()+i)
		if e != nil {
			return false, e
		}
		if b != v[i] {
			return false, nil
		}
	}
	return true, nil
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
	x, e := m.key(a)
	if e != nil {
		return false, e
	}
	y, e := m.key(b)
	if e != nil {
		return false, e
	}
	for i := uint32(0); ; i++ {
		p, e := m.region.Read(x.Allocation(), x.Offset()+i)
		if e != nil {
			return false, e
		}
		q, e := m.region.Read(y.Allocation(), y.Offset()+i)
		if e != nil {
			return false, e
		}
		if int8(p) < int8(q) {
			return true, nil
		}
		if p != q {
			return false, nil
		}
		// Different interned strings cannot have equal complete spellings.
		nextX, e := m.region.Read(x.Allocation(), x.Offset()+i+1)
		if e != nil {
			return false, e
		}
		if nextX == 0 {
			return true, nil
		}
		nextY, e := m.region.Read(y.Allocation(), y.Offset()+i+1)
		if e != nil {
			return false, e
		}
		if nextY == 0 {
			return false, nil
		}
	}
}
func (m *SymbolManager) Stats() SymbolStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := SymbolStats{Region: m.region.Stats()}
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
	if e := m.region.Close(); e != nil {
		return e
	}
	m.Budget.Release(uint64(reflect.TypeOf(SymbolManager{}).Size()))
	m.byText = nil
	m.closed = true
	return nil
}
