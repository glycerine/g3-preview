// Source: src/util/string_buffer.h:48-188.
package text

import (
	"bytes"
	"errors"
	"strconv"
	"strings"

	"github.com/glycerine/g3/pkg/arena"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

var (
	ErrOwner     = errors.New("g3: invalid string buffer owner")
	ErrConfig    = errors.New("g3: invalid string buffer long-width profile")
	ErrLongRange = errors.New("g3: string buffer value outside source long domain")
)

// LongBits selects the source ABI long width, default LP64. The unsigned
// overload deliberately uses signed 32-bit interpretation matching this
// fork's %d baseline.
type Config struct {
	LongBits uint32
}

// Stats retains the reporting shape used by existing callers.
// bytes.Buffer owns ordinary Go memory, so the arena accounting fields are zero.
type Stats struct {
	Position, Capacity                     uint64
	OwnerBytes, InlineBytes, OverflowBytes uint64
	LongBits                               uint32
}

// Buffer is the live specialization of the source string_buffer.
// Its storage uses the standard library; the zero value is ready for use.
type Buffer struct {
	buffer   bytes.Buffer
	longBits uint32
	closed   bool
}

func stringBufferLongBits(bits uint32) (uint32, error) {
	if bits == 0 {
		return 64, nil
	}
	if bits != 32 && bits != 64 {
		return 0, ErrConfig
	}
	return bits, nil
}

func NewBuffer(c Config) (*Buffer, error) {
	b := new(Buffer)
	if err := b.Init(c); err != nil {
		return nil, err
	}
	return b, nil
}

// Init configures an embedded buffer. Calling it discards prior contents.
func (b *Buffer) Init(c Config) error {
	if b == nil {
		return ErrOwner
	}
	bits, err := stringBufferLongBits(c.LongBits)
	if err != nil {
		return err
	}
	*b = Buffer{longBits: bits}
	return nil
}

func (b *Buffer) open() error {
	if b == nil {
		return ErrOwner
	}
	if b.closed {
		return arena.ErrClosed
	}
	return nil
}

func (b *Buffer) bits() uint32 {
	if b.longBits == 0 {
		return 64
	}
	return b.longBits
}

func (b *Buffer) AppendByte(v byte) error {
	if err := b.open(); err != nil {
		return err
	}
	return b.buffer.WriteByte(v)
}

func (b *Buffer) AppendBytes(v []byte) error {
	if err := b.open(); err != nil {
		return err
	}
	_, err := b.buffer.Write(v)
	return err
}

func (b *Buffer) AppendCString(v string) error {
	if err := b.open(); err != nil {
		return err
	}
	if i := strings.IndexByte(v, 0); i >= 0 {
		v = v[:i]
	}
	_, err := b.buffer.WriteString(v)
	return err
}

func (b *Buffer) appendNumber(v int64) error {
	var text [24]byte
	return b.AppendBytes(strconv.AppendInt(text[:0], v, 10))
}

func (b *Buffer) AppendInt32(v int32) error   { return b.appendNumber(int64(v)) }
func (b *Buffer) AppendUInt32(v uint32) error { return b.appendNumber(int64(int32(v))) }

func (b *Buffer) AppendLong(v int64) error {
	if err := b.open(); err != nil {
		return err
	}
	if b.bits() == 32 && (v < -2147483648 || v > 2147483647) {
		return ErrLongRange
	}
	return b.appendNumber(v)
}

func (b *Buffer) AppendBool(v bool) error {
	if v {
		return b.AppendCString("true")
	}
	return b.AppendCString("false")
}

func (b *Buffer) AppendSymbol(m *usymbol.Manager, v usymbol.Symbol) error {
	if err := b.open(); err != nil {
		return err
	}
	if m == nil {
		return usymbol.ErrInvalid
	}
	if err := m.Validate(v); err != nil {
		return err
	}
	if v.IsNumerical() {
		if err := b.AppendCString("k!"); err != nil {
			return err
		}
		return b.appendNumber(int64(int32(v.Value)))
	}
	if v.Value == 0 {
		return b.AppendCString("null")
	}
	text, _, err := m.BareString(v)
	if err != nil {
		return err
	}
	return b.AppendCString(text)
}

func (b *Buffer) Reset() error {
	if err := b.open(); err != nil {
		return err
	}
	b.buffer.Reset()
	return nil
}

func (b *Buffer) CStr() (string, error) {
	if err := b.open(); err != nil {
		return "", err
	}
	data := b.buffer.Bytes()
	if n := bytes.IndexByte(data, 0); n >= 0 {
		data = data[:n]
	}
	return string(data), nil
}

// BytesCopy returns the counted contents, including bytes after embedded NULs.
func (b *Buffer) BytesCopy() ([]byte, error) {
	if err := b.open(); err != nil {
		return nil, err
	}
	return bytes.Clone(b.buffer.Bytes()), nil
}

func (b *Buffer) Read(i uint64) (byte, error) {
	if err := b.open(); err != nil {
		return 0, err
	}
	if i >= uint64(b.buffer.Len()) {
		return 0, arena.ErrBounds
	}
	return b.buffer.Bytes()[i], nil
}

func (b *Buffer) Size() uint32 { return uint32(b.Position()) }

func (b *Buffer) Position() uint64 {
	if b.open() != nil {
		return 0
	}
	return uint64(b.buffer.Len())
}

func (b *Buffer) Empty() bool { return b.Position() == 0 }

func (b *Buffer) Stats() Stats {
	if b.open() != nil {
		return Stats{}
	}
	return Stats{
		Position: uint64(b.buffer.Len()),
		Capacity: uint64(b.buffer.Cap()),
		LongBits: b.bits(),
	}
}

func (b *Buffer) Closed() bool { return b == nil || b.closed }

func (b *Buffer) Close() error {
	if b == nil {
		return ErrOwner
	}
	if b.closed {
		return nil
	}
	b.buffer = bytes.Buffer{}
	b.closed = true
	return nil
}

func (b *Buffer) AppendBuffer(src *Buffer) error {
	if err := b.open(); err != nil {
		return err
	}
	if err := src.open(); err != nil {
		return err
	}
	text, err := src.CStr()
	if err != nil {
		return err
	}
	return b.AppendCString(text)
}
