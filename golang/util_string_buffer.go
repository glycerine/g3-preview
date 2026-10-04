// Licensed under the MIT license.
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
// Concrete INITIAL_SIZE32/64/128 translations of util/string_buffer.h.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"sync"
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
type stringBufferState struct {
	mu                        sync.Mutex
	budget                    *arena.Budget
	heap                      []byte
	pos, capacity, ownerBytes uint64
	initial, longBits         uint32
	closed                    bool
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
func (s *stringBufferState) open(valid bool) error {
	if !valid {
		return ErrStringBufferOwner
	}
	if s.closed {
		return arena.ErrClosed
	}
	return nil
}
func (s *stringBufferState) data(inline []byte) []byte {
	if s.heap != nil {
		return s.heap
	}
	return inline
}
func (s *stringBufferState) expand(inline []byte) error {
	if s.capacity > uint64(^uint(0)>>1)/2 {
		return arena.ErrBounds
	}
	n := s.capacity * 2
	if n == 0 {
		return arena.ErrBounds
	}
	if e := s.budget.Reserve(n); e != nil {
		return e
	}
	next := make([]byte, int(n))
	copy(next, s.data(inline)[:s.pos])
	if s.heap != nil {
		s.budget.Release(uint64(len(s.heap)))
	}
	s.heap = next
	s.capacity = n
	return nil
}
func (s *stringBufferState) appendBytes(inline, bytes []byte) error {
	n := uint64(len(bytes))
	if n > ^uint64(0)-s.pos {
		return arena.ErrBounds
	}
	next := s.pos + n
	for next > s.capacity {
		if e := s.expand(inline); e != nil {
			return e
		}
	}
	copy(s.data(inline)[s.pos:next], bytes)
	s.pos = next
	return nil
}
func (s *stringBufferState) appendCString(inline []byte, text string) error {
	n := uint64(cSymbolLength(text))
	if n > ^uint64(0)-s.pos {
		return arena.ErrBounds
	}
	next := s.pos + n
	for next > s.capacity {
		if e := s.expand(inline); e != nil {
			return e
		}
	}
	copy(s.data(inline)[s.pos:next], text[:n])
	s.pos = next
	return nil
}
func (s *stringBufferState) appendByte(inline []byte, b byte) error {
	if s.pos >= s.capacity {
		if e := s.expand(inline); e != nil {
			return e
		}
	}
	s.data(inline)[s.pos] = b
	s.pos++
	return nil
}
func (s *stringBufferState) cstr(inline []byte) ([]byte, error) {
	if s.pos >= s.capacity {
		if e := s.expand(inline); e != nil {
			return nil, e
		}
	}
	data := s.data(inline)
	data[s.pos] = 0
	n := s.pos
	for i := uint64(0); i < n; i++ {
		if data[i] == 0 {
			n = i
			break
		}
	}
	return data[:n], nil
}
func (s *stringBufferState) number(inline []byte, value int64) error {
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
	return s.appendBytes(inline, stack[at:len(stack)-1])
}
func (s *stringBufferState) long(inline []byte, value int64) error {
	if s.longBits == 32 && (value < -2147483648 || value > 2147483647) {
		return ErrStringBufferLongRange
	}
	return s.number(inline, value)
}
func (s *stringBufferState) boolean(inline []byte, b bool) error {
	if b {
		return s.appendCString(inline, "true")
	}
	return s.appendCString(inline, "false")
}

// Source symbol insertion is a bulk CString action for names/null, and two
// actions k! then signed UNBOXINT for numeric values. Borrowed manager lifetime
// is confined to this synchronous call; no spelling copy/owner pointer escapes.
func (s *stringBufferState) symbol(inline []byte, m *SymbolManager, k Symbol) error {
	if m == nil {
		return ErrInvalidSymbol
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.checkPlain(k); e != nil {
		return e
	}
	if k.IsNumerical() {
		if e := s.appendCString(inline, "k!"); e != nil {
			return e
		}
		return s.number(inline, int64(int32(k.value)))
	}
	if k.value == 0 {
		return s.appendCString(inline, "null")
	}
	key, e := m.key(k)
	if e != nil {
		return e
	}
	n := uint64(key.Length())
	if n > ^uint64(0)-s.pos {
		return arena.ErrBounds
	}
	next := s.pos + n
	for next > s.capacity {
		if e := s.expand(inline); e != nil {
			return e
		}
	}
	r := arena.BytesRegionRef{Slot: arena.Slot(k.value), Generation: k.generation}
	for i := uint32(0); uint64(i) < n; i++ {
		b, e := m.region.Read(r, m.pointerWidth+i)
		if e != nil {
			return e
		}
		s.data(inline)[s.pos+uint64(i)] = b
	}
	s.pos = next
	return nil
}
func sbAppendBuffer(dst, src *stringBufferState, dinline, sinline []byte, dvalid, svalid bool) error {
	stringBufferAppendMutex.Lock()
	defer stringBufferAppendMutex.Unlock()
	dst.mu.Lock()
	defer dst.mu.Unlock()
	if src != dst {
		src.mu.Lock()
		defer src.mu.Unlock()
	}
	if e := dst.open(dvalid); e != nil {
		return e
	}
	if e := src.open(svalid); e != nil {
		return e
	}
	// Self append can invalidate the original heap CString when append expands.
	// Full buffers first expand inside c_str; inline old storage remains embedded.
	if src == dst && src.heap != nil && src.pos < src.capacity {
		n := src.pos
		data := src.data(sinline)
		for i := uint64(0); i < n; i++ {
			if data[i] == 0 {
				n = i
				break
			}
		}
		if n > src.capacity-src.pos {
			return ErrStringBufferAlias
		}
	}
	bytes, e := src.cstr(sinline)
	if e != nil {
		return e
	}
	return dst.appendBytes(dinline, bytes)
}
func (s *stringBufferState) stats(valid bool) StringBufferStats {
	if !valid || s.closed {
		return StringBufferStats{}
	}
	return StringBufferStats{s.pos, s.capacity, s.ownerBytes, uint64(s.initial), uint64(len(s.heap)), s.longBits}
}
func (s *stringBufferState) close(valid bool) error {
	if !valid {
		return ErrStringBufferOwner
	}
	if s.closed {
		return nil
	}
	s.budget.Release(uint64(len(s.heap)) + s.ownerBytes)
	s.heap = nil
	s.pos = 0
	s.capacity = 0
	s.closed = true
	return nil
}

// StringBuffer is the source default INITIAL_SIZE64 specialization.
type StringBuffer = StringBuffer64

func NewStringBuffer(c StringBufferConfig) (*StringBuffer, error) { return NewStringBuffer64(c) }

// StringBuffer32 owns a physical [32]byte inline payload plus optional overflow.
// It must not be shallow-copied. No backing slice/pointer is publicly borrowed.
type StringBuffer32 struct {
	state  stringBufferState
	inline [32]byte
	self   *StringBuffer32
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
	b := &StringBuffer32{state: stringBufferState{budget: c.Budget, capacity: 32, ownerBytes: owner, initial: 32, longBits: c.LongBits}}
	b.self = b
	return b, nil
}
func (b *StringBuffer32) AppendByte(v byte) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.appendByte(b.inline[:], v)
}
func (b *StringBuffer32) AppendCString(v string) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.appendCString(b.inline[:], v)
}
func (b *StringBuffer32) AppendInt32(v int32) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.number(b.inline[:], int64(v))
}
func (b *StringBuffer32) AppendUInt32(v uint32) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.number(b.inline[:], int64(int32(v)))
}
func (b *StringBuffer32) AppendLong(v int64) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.long(b.inline[:], v)
}
func (b *StringBuffer32) AppendBool(v bool) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.boolean(b.inline[:], v)
}
func (b *StringBuffer32) AppendSymbol(m *SymbolManager, v Symbol) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.symbol(b.inline[:], m, v)
}
func (b *StringBuffer32) Reset() error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	b.state.pos = 0
	return nil
}
func (b *StringBuffer32) CStr() (string, error) {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return "", e
	}
	bytes, e := b.state.cstr(b.inline[:])
	if e != nil {
		return "", e
	}
	return string(bytes), nil
}
func (b *StringBuffer32) Read(i uint64) (byte, error) {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return 0, e
	}
	if i >= b.state.pos {
		return 0, arena.ErrBounds
	}
	return b.state.data(b.inline[:])[i], nil
}
func (b *StringBuffer32) Size() uint32 { return uint32(b.Position()) }
func (b *StringBuffer32) Position() uint64 {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if b.state.open(b.self == b) != nil {
		return 0
	}
	return b.state.pos
}
func (b *StringBuffer32) Empty() bool { return b.Position() == 0 }
func (b *StringBuffer32) Stats() StringBufferStats {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	return b.state.stats(b.self == b)
}
func (b *StringBuffer32) Close() error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	return b.state.close(b.self == b)
}
func (b *StringBuffer32) AppendBuffer32(s *StringBuffer32) error {
	if s == nil {
		return ErrStringBufferOwner
	}
	return sbAppendBuffer(&b.state, &s.state, b.inline[:], s.inline[:], b.self == b, s.self == s)
}
func (b *StringBuffer32) AppendBuffer64(s *StringBuffer64) error {
	if s == nil {
		return ErrStringBufferOwner
	}
	return sbAppendBuffer(&b.state, &s.state, b.inline[:], s.inline[:], b.self == b, s.self == s)
}
func (b *StringBuffer32) AppendBuffer128(s *StringBuffer128) error {
	if s == nil {
		return ErrStringBufferOwner
	}
	return sbAppendBuffer(&b.state, &s.state, b.inline[:], s.inline[:], b.self == b, s.self == s)
}

// StringBuffer64 owns a physical [64]byte inline payload plus optional overflow.
// It must not be shallow-copied. No backing slice/pointer is publicly borrowed.
type StringBuffer64 struct {
	state  stringBufferState
	inline [64]byte
	self   *StringBuffer64
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
	b := &StringBuffer64{state: stringBufferState{budget: c.Budget, capacity: 64, ownerBytes: owner, initial: 64, longBits: c.LongBits}}
	b.self = b
	return b, nil
}
func (b *StringBuffer64) AppendByte(v byte) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.appendByte(b.inline[:], v)
}
func (b *StringBuffer64) AppendCString(v string) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.appendCString(b.inline[:], v)
}
func (b *StringBuffer64) AppendInt32(v int32) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.number(b.inline[:], int64(v))
}
func (b *StringBuffer64) AppendUInt32(v uint32) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.number(b.inline[:], int64(int32(v)))
}
func (b *StringBuffer64) AppendLong(v int64) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.long(b.inline[:], v)
}
func (b *StringBuffer64) AppendBool(v bool) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.boolean(b.inline[:], v)
}
func (b *StringBuffer64) AppendSymbol(m *SymbolManager, v Symbol) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.symbol(b.inline[:], m, v)
}
func (b *StringBuffer64) Reset() error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	b.state.pos = 0
	return nil
}
func (b *StringBuffer64) CStr() (string, error) {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return "", e
	}
	bytes, e := b.state.cstr(b.inline[:])
	if e != nil {
		return "", e
	}
	return string(bytes), nil
}
func (b *StringBuffer64) Read(i uint64) (byte, error) {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return 0, e
	}
	if i >= b.state.pos {
		return 0, arena.ErrBounds
	}
	return b.state.data(b.inline[:])[i], nil
}
func (b *StringBuffer64) Size() uint32 { return uint32(b.Position()) }
func (b *StringBuffer64) Position() uint64 {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if b.state.open(b.self == b) != nil {
		return 0
	}
	return b.state.pos
}
func (b *StringBuffer64) Empty() bool { return b.Position() == 0 }
func (b *StringBuffer64) Stats() StringBufferStats {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	return b.state.stats(b.self == b)
}
func (b *StringBuffer64) Close() error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	return b.state.close(b.self == b)
}
func (b *StringBuffer64) AppendBuffer32(s *StringBuffer32) error {
	if s == nil {
		return ErrStringBufferOwner
	}
	return sbAppendBuffer(&b.state, &s.state, b.inline[:], s.inline[:], b.self == b, s.self == s)
}
func (b *StringBuffer64) AppendBuffer64(s *StringBuffer64) error {
	if s == nil {
		return ErrStringBufferOwner
	}
	return sbAppendBuffer(&b.state, &s.state, b.inline[:], s.inline[:], b.self == b, s.self == s)
}
func (b *StringBuffer64) AppendBuffer128(s *StringBuffer128) error {
	if s == nil {
		return ErrStringBufferOwner
	}
	return sbAppendBuffer(&b.state, &s.state, b.inline[:], s.inline[:], b.self == b, s.self == s)
}

// StringBuffer128 owns a physical [128]byte inline payload plus optional overflow.
// It must not be shallow-copied. No backing slice/pointer is publicly borrowed.
type StringBuffer128 struct {
	state  stringBufferState
	inline [128]byte
	self   *StringBuffer128
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
	b := &StringBuffer128{state: stringBufferState{budget: c.Budget, capacity: 128, ownerBytes: owner, initial: 128, longBits: c.LongBits}}
	b.self = b
	return b, nil
}
func (b *StringBuffer128) AppendByte(v byte) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.appendByte(b.inline[:], v)
}
func (b *StringBuffer128) AppendCString(v string) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.appendCString(b.inline[:], v)
}
func (b *StringBuffer128) AppendInt32(v int32) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.number(b.inline[:], int64(v))
}
func (b *StringBuffer128) AppendUInt32(v uint32) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.number(b.inline[:], int64(int32(v)))
}
func (b *StringBuffer128) AppendLong(v int64) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.long(b.inline[:], v)
}
func (b *StringBuffer128) AppendBool(v bool) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.boolean(b.inline[:], v)
}
func (b *StringBuffer128) AppendSymbol(m *SymbolManager, v Symbol) error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	return b.state.symbol(b.inline[:], m, v)
}
func (b *StringBuffer128) Reset() error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return e
	}
	b.state.pos = 0
	return nil
}
func (b *StringBuffer128) CStr() (string, error) {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return "", e
	}
	bytes, e := b.state.cstr(b.inline[:])
	if e != nil {
		return "", e
	}
	return string(bytes), nil
}
func (b *StringBuffer128) Read(i uint64) (byte, error) {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return 0, e
	}
	if i >= b.state.pos {
		return 0, arena.ErrBounds
	}
	return b.state.data(b.inline[:])[i], nil
}
func (b *StringBuffer128) Size() uint32 { return uint32(b.Position()) }
func (b *StringBuffer128) Position() uint64 {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if b.state.open(b.self == b) != nil {
		return 0
	}
	return b.state.pos
}
func (b *StringBuffer128) Empty() bool { return b.Position() == 0 }
func (b *StringBuffer128) Stats() StringBufferStats {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	return b.state.stats(b.self == b)
}
func (b *StringBuffer128) Close() error {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	return b.state.close(b.self == b)
}
func (b *StringBuffer128) AppendBuffer32(s *StringBuffer32) error {
	if s == nil {
		return ErrStringBufferOwner
	}
	return sbAppendBuffer(&b.state, &s.state, b.inline[:], s.inline[:], b.self == b, s.self == s)
}
func (b *StringBuffer128) AppendBuffer64(s *StringBuffer64) error {
	if s == nil {
		return ErrStringBufferOwner
	}
	return sbAppendBuffer(&b.state, &s.state, b.inline[:], s.inline[:], b.self == b, s.self == s)
}
func (b *StringBuffer128) AppendBuffer128(s *StringBuffer128) error {
	if s == nil {
		return ErrStringBufferOwner
	}
	return sbAppendBuffer(&b.state, &s.state, b.inline[:], s.inline[:], b.self == b, s.self == s)
}
