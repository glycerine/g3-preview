// Concrete INITIAL_SIZE32/64/128 translations of util/string_buffer.h.
package util

import (
	"errors"
	"reflect"
	"sync"

	"github.com/glycerine/g3/pkg/arena"
)

var (
	ErrStringBufferOwner     = errors.New("g3: invalid string buffer owner")
	ErrStringBufferConfig    = errors.New("g3: invalid string buffer long-width profile")
	ErrStringBufferLongRange = errors.New("g3: string buffer value outside source long domain")
	ErrStringBufferAlias     = errors.New("g3: source string buffer self-append would use freed heap storage")
)
var stringBufferAppendMutex sync.Mutex

// LongBits selects the source ABI long width, default LP64. The unsigned
// overload deliberately uses signed32-bit interpretation matching this fork's
// %d baseline, a defined Go adaptation of the source high-bit type mismatch.
type StringBufferConfig struct {
	Budget   *arena.Budget
	LongBits uint32
}
type StringBufferStats struct {
	Position, Capacity                     uint64
	OwnerBytes, InlineBytes, OverflowBytes uint64
	LongBits                               uint32
}

// InlineBytes is included in OwnerBytes, not a second budget charge. Each
// concrete owner physically embeds its own exact initial byte array.
type StringBufferState struct {
	Mu                        sync.Mutex
	Budget                    *arena.Budget
	Heap                      []byte
	Pos, Capacity, OwnerBytes uint64
	Initial, LongBits         uint32
	Closed                    bool
}

func sbConfig(c StringBufferConfig) (StringBufferConfig, error) {
	if c.LongBits == 0 {
		c.LongBits = 64
	}
	if c.LongBits != 32 && c.LongBits != 64 {
		return c, ErrStringBufferConfig
	}
	return c, nil
}
func (s *StringBufferState) Open(valid bool) error {
	if !valid {
		return ErrStringBufferOwner
	}
	if s.Closed {
		return arena.ErrClosed
	}
	return nil
}
func (s *StringBufferState) Data(inline []byte) []byte {
	if s.Heap != nil {
		return s.Heap
	}
	return inline
}
func (s *StringBufferState) Expand(inline []byte) error {
	if s.Capacity > uint64(^uint(0)>>1)/2 {
		return arena.ErrBounds
	}
	n := s.Capacity * 2
	if n == 0 {
		return arena.ErrBounds
	}
	if e := s.Budget.Reserve(n); e != nil {
		return e
	}
	next := make([]byte, int(n))
	copy(next, s.Data(inline)[:s.Pos])
	if s.Heap != nil {
		s.Budget.Release(uint64(len(s.Heap)))
	}
	s.Heap = next
	s.Capacity = n
	return nil
}
func (s *StringBufferState) AppendBytes(inline, bytes []byte) error {
	n := uint64(len(bytes))
	if n > ^uint64(0)-s.Pos {
		return arena.ErrBounds
	}
	next := s.Pos + n
	for next > s.Capacity {
		if e := s.Expand(inline); e != nil {
			return e
		}
	}
	copy(s.Data(inline)[s.Pos:next], bytes)
	s.Pos = next
	return nil
}
func (s *StringBufferState) AppendCString(inline []byte, text string) error {
	n := uint64(cSymbolLength(text))
	if n > ^uint64(0)-s.Pos {
		return arena.ErrBounds
	}
	next := s.Pos + n
	for next > s.Capacity {
		if e := s.Expand(inline); e != nil {
			return e
		}
	}
	copy(s.Data(inline)[s.Pos:next], text[:n])
	s.Pos = next
	return nil
}
func (s *StringBufferState) AppendByte(inline []byte, b byte) error {
	if s.Pos >= s.Capacity {
		if e := s.Expand(inline); e != nil {
			return e
		}
	}
	s.Data(inline)[s.Pos] = b
	s.Pos++
	return nil
}
func (s *StringBufferState) Cstr(inline []byte) ([]byte, error) {
	if s.Pos >= s.Capacity {
		if e := s.Expand(inline); e != nil {
			return nil, e
		}
	}
	data := s.Data(inline)
	data[s.Pos] = 0
	n := s.Pos
	for i := uint64(0); i < n; i++ {
		if data[i] == 0 {
			n = i
			break
		}
	}
	return data[:n], nil
}
func (s *StringBufferState) Number(inline []byte, value int64) error {
	var stack [24]byte
	at := len(stack) - 1
	negative := value < 0
	v := uint64(value)
	if negative {
		v = ^v + 1
	}
	for {
		at--
		stack[at] = '0' + byte(v%10)
		v /= 10
		if v == 0 {
			break
		}
	}
	if negative {
		at--
		stack[at] = '-'
	}
	return s.AppendBytes(inline, stack[at:len(stack)-1])
}
func (s *StringBufferState) Long(inline []byte, value int64) error {
	if s.LongBits == 32 && (value < -2147483648 || value > 2147483647) {
		return ErrStringBufferLongRange
	}
	return s.Number(inline, value)
}
func (s *StringBufferState) Boolean(inline []byte, b bool) error {
	if b {
		return s.AppendCString(inline, "true")
	}
	return s.AppendCString(inline, "false")
}

// Source symbol insertion is a bulk CString action for names/null, and two
// actions k! then signed UNBOXINT for numeric values. Borrowed manager lifetime
// is confined to this synchronous call; no spelling copy/owner pointer escapes.
func (s *StringBufferState) Symbol(inline []byte, m *SymbolManager, k Symbol) error {
	if m == nil {
		return ErrInvalidSymbol
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.checkPlain(k); e != nil {
		return e
	}
	if k.IsNumerical() {
		if e := s.AppendCString(inline, "k!"); e != nil {
			return e
		}
		return s.Number(inline, int64(int32(k.Value)))
	}
	if k.Value == 0 {
		return s.AppendCString(inline, "null")
	}
	text, e := m.spelling(k)
	if e != nil {
		return e
	}
	n := uint64(len(text))
	if n > ^uint64(0)-s.Pos {
		return arena.ErrBounds
	}
	next := s.Pos + n
	for next > s.Capacity {
		if e := s.Expand(inline); e != nil {
			return e
		}
	}
	copy(s.Data(inline)[s.Pos:next], text)
	s.Pos = next
	return nil
}
func sbAppendBuffer(dst, src *StringBufferState, dinline, sinline []byte, dvalid, svalid bool) error {
	stringBufferAppendMutex.Lock()
	defer stringBufferAppendMutex.Unlock()
	dst.Mu.Lock()
	defer dst.Mu.Unlock()
	if src != dst {
		src.Mu.Lock()
		defer src.Mu.Unlock()
	}
	if e := dst.Open(dvalid); e != nil {
		return e
	}
	if e := src.Open(svalid); e != nil {
		return e
	}
	// Self append can invalidate the original heap CString when append expands.
	// Full buffers first expand inside c_str; inline old storage remains embedded.
	if src == dst && src.Heap != nil && src.Pos < src.Capacity {
		n := src.Pos
		data := src.Data(sinline)
		for i := uint64(0); i < n; i++ {
			if data[i] == 0 {
				n = i
				break
			}
		}
		if n > src.Capacity-src.Pos {
			return ErrStringBufferAlias
		}
	}
	bytes, e := src.Cstr(sinline)
	if e != nil {
		return e
	}
	return dst.AppendBytes(dinline, bytes)
}
func (s *StringBufferState) Stats(valid bool) StringBufferStats {
	if !valid || s.Closed {
		return StringBufferStats{}
	}
	return StringBufferStats{s.Pos, s.Capacity, s.OwnerBytes, uint64(s.Initial), uint64(len(s.Heap)), s.LongBits}
}
func (s *StringBufferState) Close(valid bool) error {
	if !valid {
		return ErrStringBufferOwner
	}
	if s.Closed {
		return nil
	}
	s.Budget.Release(uint64(len(s.Heap)) + s.OwnerBytes)
	s.Heap = nil
	s.Pos = 0
	s.Capacity = 0
	s.Closed = true
	return nil
}

// StringBuffer is the source default INITIAL_SIZE64 specialization.
type StringBuffer = StringBuffer64

func NewStringBuffer(c StringBufferConfig) (*StringBuffer, error) { return NewStringBuffer64(c) }

// StringBuffer32 owns a physical [32]byte inline payload plus optional overflow.
// It must not be shallow-copied. No backing slice/pointer is publicly borrowed.
type StringBuffer32 struct {
	State  StringBufferState
	Inline [32]byte
	Self   *StringBuffer32
}

func NewStringBuffer32(c StringBufferConfig) (*StringBuffer32, error) {
	c, e := sbConfig(c)
	if e != nil {
		return nil, e
	}
	owner := uint64(reflect.TypeOf(StringBuffer32{}).Size())
	if e = c.Budget.Reserve(owner); e != nil {
		return nil, e
	}
	b := &StringBuffer32{State: StringBufferState{Budget: c.Budget, Capacity: 32, OwnerBytes: owner, Initial: 32, LongBits: c.LongBits}}
	b.Self = b
	return b, nil
}
func (b *StringBuffer32) AppendByte(v byte) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.AppendByte(b.Inline[:], v)
}
func (b *StringBuffer32) AppendCString(v string) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.AppendCString(b.Inline[:], v)
}
func (b *StringBuffer32) AppendInt32(v int32) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.Number(b.Inline[:], int64(v))
}
func (b *StringBuffer32) AppendUInt32(v uint32) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.Number(b.Inline[:], int64(int32(v)))
}
func (b *StringBuffer32) AppendLong(v int64) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.Long(b.Inline[:], v)
}
func (b *StringBuffer32) AppendBool(v bool) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.Boolean(b.Inline[:], v)
}
func (b *StringBuffer32) AppendSymbol(m *SymbolManager, v Symbol) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.Symbol(b.Inline[:], m, v)
}
func (b *StringBuffer32) Reset() error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	b.State.Pos = 0
	return nil
}
func (b *StringBuffer32) CStr() (string, error) {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return "", e
	}
	bytes, e := b.State.Cstr(b.Inline[:])
	if e != nil {
		return "", e
	}
	return string(bytes), nil
}
func (b *StringBuffer32) Read(i uint64) (byte, error) {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return 0, e
	}
	if i >= b.State.Pos {
		return 0, arena.ErrBounds
	}
	return b.State.Data(b.Inline[:])[i], nil
}
func (b *StringBuffer32) Size() uint32 { return uint32(b.Position()) }
func (b *StringBuffer32) Position() uint64 {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if b.State.Open(b.Self == b) != nil {
		return 0
	}
	return b.State.Pos
}
func (b *StringBuffer32) Empty() bool { return b.Position() == 0 }
func (b *StringBuffer32) Stats() StringBufferStats {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	return b.State.Stats(b.Self == b)
}
func (b *StringBuffer32) Close() error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	return b.State.Close(b.Self == b)
}
func (b *StringBuffer32) AppendBuffer32(s *StringBuffer32) error {
	if s == nil {
		return ErrStringBufferOwner
	}
	return sbAppendBuffer(&b.State, &s.State, b.Inline[:], s.Inline[:], b.Self == b, s.Self == s)
}
func (b *StringBuffer32) AppendBuffer64(s *StringBuffer64) error {
	if s == nil {
		return ErrStringBufferOwner
	}
	return sbAppendBuffer(&b.State, &s.State, b.Inline[:], s.Inline[:], b.Self == b, s.Self == s)
}
func (b *StringBuffer32) AppendBuffer128(s *StringBuffer128) error {
	if s == nil {
		return ErrStringBufferOwner
	}
	return sbAppendBuffer(&b.State, &s.State, b.Inline[:], s.Inline[:], b.Self == b, s.Self == s)
}

// StringBuffer64 owns a physical [64]byte inline payload plus optional overflow.
// It must not be shallow-copied. No backing slice/pointer is publicly borrowed.
type StringBuffer64 struct {
	State  StringBufferState
	Inline [64]byte
	Self   *StringBuffer64
}

func NewStringBuffer64(c StringBufferConfig) (*StringBuffer64, error) {
	c, e := sbConfig(c)
	if e != nil {
		return nil, e
	}
	owner := uint64(reflect.TypeOf(StringBuffer64{}).Size())
	if e = c.Budget.Reserve(owner); e != nil {
		return nil, e
	}
	b := &StringBuffer64{State: StringBufferState{Budget: c.Budget, Capacity: 64, OwnerBytes: owner, Initial: 64, LongBits: c.LongBits}}
	b.Self = b
	return b, nil
}
func (b *StringBuffer64) AppendByte(v byte) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.AppendByte(b.Inline[:], v)
}
func (b *StringBuffer64) AppendCString(v string) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.AppendCString(b.Inline[:], v)
}
func (b *StringBuffer64) AppendInt32(v int32) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.Number(b.Inline[:], int64(v))
}
func (b *StringBuffer64) AppendUInt32(v uint32) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.Number(b.Inline[:], int64(int32(v)))
}
func (b *StringBuffer64) AppendLong(v int64) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.Long(b.Inline[:], v)
}
func (b *StringBuffer64) AppendBool(v bool) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.Boolean(b.Inline[:], v)
}
func (b *StringBuffer64) AppendSymbol(m *SymbolManager, v Symbol) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.Symbol(b.Inline[:], m, v)
}
func (b *StringBuffer64) Reset() error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	b.State.Pos = 0
	return nil
}
func (b *StringBuffer64) CStr() (string, error) {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return "", e
	}
	bytes, e := b.State.Cstr(b.Inline[:])
	if e != nil {
		return "", e
	}
	return string(bytes), nil
}
func (b *StringBuffer64) Read(i uint64) (byte, error) {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return 0, e
	}
	if i >= b.State.Pos {
		return 0, arena.ErrBounds
	}
	return b.State.Data(b.Inline[:])[i], nil
}
func (b *StringBuffer64) Size() uint32 { return uint32(b.Position()) }
func (b *StringBuffer64) Position() uint64 {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if b.State.Open(b.Self == b) != nil {
		return 0
	}
	return b.State.Pos
}
func (b *StringBuffer64) Empty() bool { return b.Position() == 0 }
func (b *StringBuffer64) Stats() StringBufferStats {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	return b.State.Stats(b.Self == b)
}
func (b *StringBuffer64) Close() error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	return b.State.Close(b.Self == b)
}
func (b *StringBuffer64) AppendBuffer32(s *StringBuffer32) error {
	if s == nil {
		return ErrStringBufferOwner
	}
	return sbAppendBuffer(&b.State, &s.State, b.Inline[:], s.Inline[:], b.Self == b, s.Self == s)
}
func (b *StringBuffer64) AppendBuffer64(s *StringBuffer64) error {
	if s == nil {
		return ErrStringBufferOwner
	}
	return sbAppendBuffer(&b.State, &s.State, b.Inline[:], s.Inline[:], b.Self == b, s.Self == s)
}
func (b *StringBuffer64) AppendBuffer128(s *StringBuffer128) error {
	if s == nil {
		return ErrStringBufferOwner
	}
	return sbAppendBuffer(&b.State, &s.State, b.Inline[:], s.Inline[:], b.Self == b, s.Self == s)
}

// StringBuffer128 owns a physical [128]byte inline payload plus optional overflow.
// It must not be shallow-copied. No backing slice/pointer is publicly borrowed.
type StringBuffer128 struct {
	State  StringBufferState
	Inline [128]byte
	Self   *StringBuffer128
}

func NewStringBuffer128(c StringBufferConfig) (*StringBuffer128, error) {
	c, e := sbConfig(c)
	if e != nil {
		return nil, e
	}
	owner := uint64(reflect.TypeOf(StringBuffer128{}).Size())
	if e = c.Budget.Reserve(owner); e != nil {
		return nil, e
	}
	b := &StringBuffer128{State: StringBufferState{Budget: c.Budget, Capacity: 128, OwnerBytes: owner, Initial: 128, LongBits: c.LongBits}}
	b.Self = b
	return b, nil
}
func (b *StringBuffer128) AppendByte(v byte) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.AppendByte(b.Inline[:], v)
}
func (b *StringBuffer128) AppendCString(v string) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.AppendCString(b.Inline[:], v)
}
func (b *StringBuffer128) AppendInt32(v int32) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.Number(b.Inline[:], int64(v))
}
func (b *StringBuffer128) AppendUInt32(v uint32) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.Number(b.Inline[:], int64(int32(v)))
}
func (b *StringBuffer128) AppendLong(v int64) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.Long(b.Inline[:], v)
}
func (b *StringBuffer128) AppendBool(v bool) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.Boolean(b.Inline[:], v)
}
func (b *StringBuffer128) AppendSymbol(m *SymbolManager, v Symbol) error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	return b.State.Symbol(b.Inline[:], m, v)
}
func (b *StringBuffer128) Reset() error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return e
	}
	b.State.Pos = 0
	return nil
}
func (b *StringBuffer128) CStr() (string, error) {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return "", e
	}
	bytes, e := b.State.Cstr(b.Inline[:])
	if e != nil {
		return "", e
	}
	return string(bytes), nil
}
func (b *StringBuffer128) Read(i uint64) (byte, error) {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if e := b.State.Open(b.Self == b); e != nil {
		return 0, e
	}
	if i >= b.State.Pos {
		return 0, arena.ErrBounds
	}
	return b.State.Data(b.Inline[:])[i], nil
}
func (b *StringBuffer128) Size() uint32 { return uint32(b.Position()) }
func (b *StringBuffer128) Position() uint64 {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if b.State.Open(b.Self == b) != nil {
		return 0
	}
	return b.State.Pos
}
func (b *StringBuffer128) Empty() bool { return b.Position() == 0 }
func (b *StringBuffer128) Stats() StringBufferStats {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	return b.State.Stats(b.Self == b)
}
func (b *StringBuffer128) Close() error {
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	return b.State.Close(b.Self == b)
}
func (b *StringBuffer128) AppendBuffer32(s *StringBuffer32) error {
	if s == nil {
		return ErrStringBufferOwner
	}
	return sbAppendBuffer(&b.State, &s.State, b.Inline[:], s.Inline[:], b.Self == b, s.Self == s)
}
func (b *StringBuffer128) AppendBuffer64(s *StringBuffer64) error {
	if s == nil {
		return ErrStringBufferOwner
	}
	return sbAppendBuffer(&b.State, &s.State, b.Inline[:], s.Inline[:], b.Self == b, s.Self == s)
}
func (b *StringBuffer128) AppendBuffer128(s *StringBuffer128) error {
	if s == nil {
		return ErrStringBufferOwner
	}
	return sbAppendBuffer(&b.State, &s.State, b.Inline[:], s.Inline[:], b.Self == b, s.Self == s)
}
