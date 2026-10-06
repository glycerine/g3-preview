// Licensed under the MIT license.
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
// Source: src/util/warning.cpp:105-169.
// Source author: Leonardo de Moura (2006-12-01).
// Explicit-argument integer/C-string subset of format2ostream.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"os"
	"reflect"
	"sync"
)

type PrintfProfile uint32

const (
	PrintfNonWindowsAMD64 PrintfProfile = iota
	PrintfNonWindows32
)

var (
	ErrPrintfOwner         = errors.New("g3: invalid printf owner")
	ErrPrintfProfile       = errors.New("g3: unsupported printf platform profile")
	ErrPrintfArgument      = errors.New("g3: printf argument type/count precondition")
	ErrPrintfUnsupported   = errors.New("g3: unsupported legitimate printf format domain")
	ErrPrintfMalformed     = errors.New("g3: persistent malformed printf backend format")
	ErrPrintfCountOverflow = errors.New("g3: printf signed count overflow")
)

const (
	pfInt32 uint32 = iota + 1
	pfUInt32
	pfInt64
	pfUInt64
	pfCString
	pfChar
)

// PrintfArg is a cold explicit va_list adaptation; args and nullable CString
// pointers are borrowed for one synchronous call and never retained in an owner.
type PrintfArg struct {
	kind         uint32
	bits         uint64
	text         *string
	view         ParamsStringView
	borrowedView bool
}

func PrintfInt32(v int32) PrintfArg     { return PrintfArg{kind: pfInt32, bits: uint64(uint32(v))} }
func PrintfUInt32(v uint32) PrintfArg   { return PrintfArg{kind: pfUInt32, bits: uint64(v)} }
func PrintfInt64(v int64) PrintfArg     { return PrintfArg{kind: pfInt64, bits: uint64(v)} }
func PrintfUInt64(v uint64) PrintfArg   { return PrintfArg{kind: pfUInt64, bits: v} }
func PrintfCString(v *string) PrintfArg { return PrintfArg{kind: pfCString, text: v} }

// PrintfCStringView borrows the actual source pointer for the reached %s leaf.
func PrintfCStringView(v ParamsStringView) PrintfArg {
	return PrintfArg{kind: pfCString, view: v, borrowedView: true}
}
func PrintfChar(v uint32) PrintfArg { return PrintfArg{kind: pfChar, bits: uint64(v)} }

type PrintfEventKind uint32

const (
	PrintfGrow PrintfEventKind = iota
	PrintfBegin
	PrintfCall
	PrintfEnd
)

type PrintfFailure uint32

const (
	PrintfNoFailure PrintfFailure = iota
	PrintfMalformedFailure
	PrintfCountFailure
	PrintfUnsupportedFailure
	PrintfArgumentFailure
)

// Events are Go-only diagnostics. Linux BEGIN/END macros have no callbacks.
type PrintfEvent struct {
	Kind                        PrintfEventKind
	OldCapacity, Capacity, Size uint32
	Count                       int64
	Failure                     PrintfFailure
}
type PrintfConfig struct {
	Budget  *arena.Budget
	Profile PrintfProfile
	Observe func(PrintfEvent)
}
type PrintfStats struct {
	OwnerBytes, WorkspaceBytes uint64
	Capacity, Size             uint32
	Calls                      uint64
}

// Each call owns and destroys the source vector scratch. The mutex protects
// only this owner; externally shared Budgets require caller serialization.
// Observe's environment is borrowed and must not reenter any owner methods.
// Final immutable ostream/exception text and borrowed args are caller-owned.
type PrintfFormatter struct {
	mu     sync.Mutex
	self   *PrintfFormatter
	config PrintfConfig
	buffer []byte
	size   uint32
	calls  uint64
	closed bool
}

func NewPrintfFormatter(c PrintfConfig) (*PrintfFormatter, error) {
	if c.Profile > PrintfNonWindows32 {
		return nil, ErrPrintfProfile
	}
	if e := c.Budget.Reserve(uint64(reflect.TypeOf(PrintfFormatter{}).Size())); e != nil {
		return nil, e
	}
	f := &PrintfFormatter{config: c}
	f.self = f
	return f, nil
}
func (f *PrintfFormatter) event(e PrintfEvent) {
	if f.config.Observe != nil {
		f.config.Observe(e)
	}
}
func (f *PrintfFormatter) resize(size uint32) error {
	for uint64(cap(f.buffer)) < uint64(size) {
		n, e := pdGrowth(cap(f.buffer))
		if e != nil {
			return e
		}
		if e = f.config.Budget.Reserve(uint64(n)); e != nil {
			return e
		}
		b := make([]byte, n)
		copy(b, f.buffer)
		old := uint32(cap(f.buffer))
		f.config.Budget.Release(uint64(old))
		f.buffer = b
		f.event(PrintfEvent{Kind: PrintfGrow, OldCapacity: old, Capacity: uint32(n), Size: f.size})
	}
	f.size = size
	return nil
}
func (f *PrintfFormatter) clear() {
	f.config.Budget.Release(uint64(cap(f.buffer)))
	f.buffer = nil
	f.size = 0
}
func (f *PrintfFormatter) Stats() PrintfStats {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.self != f || f.closed {
		return PrintfStats{}
	}
	return PrintfStats{uint64(reflect.TypeOf(PrintfFormatter{}).Size()), uint64(cap(f.buffer)), uint32(cap(f.buffer)), f.size, f.calls}
}
func (f *PrintfFormatter) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.self != f {
		return ErrPrintfOwner
	}
	if f.closed {
		return nil
	}
	f.clear()
	f.config.Budget.Release(uint64(reflect.TypeOf(PrintfFormatter{}).Size()))
	f.closed = true
	return nil
}
func (f *PrintfFormatter) Format(format string, args []PrintfArg) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.self != f {
		return "", ErrPrintfOwner
	}
	if f.closed {
		return "", arena.ErrClosed
	}
	defer f.clear()
	initial := uint32(1024)
	if f.config.Profile == PrintfNonWindows32 {
		initial = 128
	}
	if e := f.resize(initial); e != nil {
		return "", e
	}
	f.event(PrintfEvent{Kind: PrintfBegin, Capacity: uint32(cap(f.buffer)), Size: f.size})
	format = format[:cSymbolLength(format)]
	for {
		count, failure, e := pfRender(f.buffer[:f.size], f.config.Profile, format, args)
		f.calls++
		f.event(PrintfEvent{Kind: PrintfCall, Capacity: uint32(cap(f.buffer)), Size: f.size, Count: count, Failure: failure})
		if failure == PrintfUnsupportedFailure || failure == PrintfArgumentFailure {
			return "", e
		}
		if f.config.Profile == PrintfNonWindowsAMD64 {
			f.event(PrintfEvent{Kind: PrintfEnd, Capacity: uint32(cap(f.buffer)), Size: f.size, Count: count, Failure: failure})
			if count < 0 {
				return format, nil
			}
			n := uint32(count)
			if n >= f.size {
				n = f.size - 1
				f.buffer[n] = 0
			}
			for i := uint32(0); i < n; i++ {
				if f.buffer[i] == 0 {
					n = i
					break
				}
			}
			return string(f.buffer[:n]), nil
		}
		if count < 0 {
			return "", e
		} // Source would retry forever: explicit typed finite adapter.
		if uint64(count) < uint64(f.size) {
			f.event(PrintfEvent{Kind: PrintfEnd, Capacity: uint32(cap(f.buffer)), Size: f.size, Count: count})
			n := uint32(count)
			for i := uint32(0); i < n; i++ {
				if f.buffer[i] == 0 {
					n = i
					break
				}
			}
			return string(f.buffer[:n]), nil
		}
		if f.size > (^uint32(0)-1)/2 {
			return "", arena.ErrBounds
		}
		if e := f.resize(f.size*2 + 1); e != nil {
			return "", e
		}
	}
}
func NewDefaultExceptionFormatted(f *PrintfFormatter, format string, args []PrintfArg) (DefaultException, error) {
	if f == nil {
		return DefaultException{}, ErrPrintfOwner
	}
	s, e := f.Format(format, args)
	if e != nil {
		return DefaultException{}, e
	}
	return NewDefaultException(s), nil
}

// pfSink writes at most size-1 bytes but counts every conceptual output byte.
// Repeated width/precision spans are counted in O(1), never allocated as strings.
// Concrete call-local sinks share the original parser/number engine. Optional
// raw and stdio destinations never retain a second hot byte payload. The file
// sink is the explicit unbuffered Go adapter of fprintf, not an ostream.
type pfSink struct {
	buffer         []byte
	runtime        *memory.Runtime
	block          memory.Block
	offset, extent uint64
	file           *os.File
	fileByte       [1]byte
	count          uint64
	overflow       bool
	cause          error
}

func (s *pfSink) limit() uint64 {
	if s.runtime != nil {
		if s.extent == 0 {
			return 0
		}
		return s.extent - 1
	}
	if s.file != nil {
		return ^uint64(0)
	}
	return uint64(len(s.buffer) - 1)
}
func (s *pfSink) put(i uint64, v byte) {
	if s.cause != nil {
		return
	}
	if s.runtime != nil {
		if s.extent == 0 {
			return
		}
		if i > ^uint64(0)-s.offset {
			s.cause = arena.ErrBounds
			return
		}
		s.cause = s.runtime.Write(s.block, s.offset+i, v)
		return
	}
	if s.file != nil {
		s.fileByte[0] = v
		_, _ = s.file.Write(s.fileByte[:])
		return
	}
	s.buffer[i] = v
}

func (s *pfSink) add(n uint64) bool {
	if n > uint64(2147483647)-s.count {
		s.overflow = true
		return false
	}
	return true
}
func (s *pfSink) repeat(ch byte, n uint64) {
	if !s.add(n) {
		return
	}
	limit := s.limit()
	for i := s.count; i < limit && i < s.count+n; i++ {
		s.put(i, ch)
	}
	s.count += n
}
func (s *pfSink) bytes(b []byte) {
	if !s.add(uint64(len(b))) {
		return
	}
	limit := s.limit()
	for i, ch := range b {
		if s.count+uint64(i) < limit {
			s.put(s.count+uint64(i), ch)
		}
	}
	s.count += uint64(len(b))
}
func (s *pfSink) text(b string) {
	if !s.add(uint64(len(b))) {
		return
	}
	limit := s.limit()
	n := uint64(len(b))
	if s.count < limit {
		if n > limit-s.count {
			n = limit - s.count
		}
		for i := uint64(0); i < n; i++ {
			s.put(s.count+i, b[i])
		}
	}
	s.count += uint64(len(b))
}
func (s *pfSink) finish() {
	if s.file != nil || s.runtime != nil && s.extent == 0 {
		return
	}
	i := s.count
	limit := s.limit()
	if i > limit {
		i = limit
	}
	s.put(i, 0)
}
func (s *pfSink) cstring(v ParamsStringView, precision int64) error {
	var n uint64
	for precision < 0 || n < uint64(precision) {
		b, e := v.Byte(n)
		if e != nil {
			return e
		}
		if b == 0 {
			break
		}
		n++
	}
	if !s.add(n) {
		s.overflow = true
		return nil
	}
	limit := s.limit()
	for i := uint64(0); i < n && s.count+i < limit; i++ {
		b, e := v.Byte(i)
		if e != nil {
			return e
		}
		s.put(s.count+i, b)
		if s.cause != nil {
			return s.cause
		}
	}
	s.count += n
	return nil
}

type pfFlags struct{ left, plus, space, alternate, zero bool }

func pfStar(args []PrintfArg, index *int) (int64, error) {
	if *index >= len(args) {
		return 0, ErrPrintfArgument
	}
	a := args[*index]
	*index++
	if a.kind != pfInt32 {
		return 0, ErrPrintfArgument
	}
	return int64(int32(a.bits)), nil
}
func pfDecimal(format string, index *int) (int64, bool) {
	var v int64
	for *index < len(format) && format[*index] >= '0' && format[*index] <= '9' {
		d := int64(format[*index] - '0')
		if v > (2147483647-d)/10 {
			return 0, false
		}
		v = v*10 + d
		*index++
	}
	return v, true
}
func pfNumber(a PrintfArg, signed bool, width uint32) (uint64, bool, error) {
	var bits uint64
	if width <= 32 {
		if a.kind != pfInt32 && a.kind != pfUInt32 {
			return 0, false, ErrPrintfArgument
		}
		if signed && width == 32 && a.kind == pfUInt32 && a.bits > 2147483647 {
			return 0, false, ErrPrintfArgument
		}
		if !signed && width == 32 && a.kind == pfInt32 && int32(a.bits) < 0 {
			return 0, false, ErrPrintfArgument
		}
		bits = a.bits & ((uint64(1) << width) - 1)
	} else {
		if signed {
			if a.kind != pfInt64 {
				return 0, false, ErrPrintfArgument
			}
		} else {
			if a.kind != pfUInt64 {
				return 0, false, ErrPrintfArgument
			}
		}
		bits = a.bits
	}
	if signed && bits&(uint64(1)<<(width-1)) != 0 {
		if width == 64 {
			return ^bits + 1, true, nil
		}
		return (uint64(1) << width) - bits, true, nil
	}
	return bits, false, nil
}
func pfInteger(s *pfSink, a PrintfArg, verb byte, length uint32, profile PrintfProfile, flags pfFlags, width, precision int64) error {
	signed := verb == 'd' || verb == 'i'
	bits := uint32(32)
	switch length {
	case 1:
		bits = 8
	case 2:
		bits = 16
	case 3, 6, 7:
		if profile == PrintfNonWindowsAMD64 {
			bits = 64
		}
	case 4, 5:
		bits = 64
	}
	value, negative, e := pfNumber(a, signed, bits)
	if e != nil {
		return e
	}
	base := uint64(10)
	if verb == 'o' {
		base = 8
	}
	if verb == 'x' || verb == 'X' {
		base = 16
	}
	var digits [64]byte
	at := len(digits)
	v := value
	if v != 0 || precision != 0 {
		for {
			at--
			d := byte(v % base)
			if d < 10 {
				digits[at] = '0' + d
			} else if verb == 'X' {
				digits[at] = 'A' + d - 10
			} else {
				digits[at] = 'a' + d - 10
			}
			v /= base
			if v == 0 {
				break
			}
		}
	}
	count := len(digits) - at
	explicitPrecision := precision >= 0
	if flags.alternate && base == 8 && (value != 0 || count == 0) && precision < int64(count+1) {
		precision = int64(count + 1)
	}
	zeros := uint64(0)
	if precision > int64(count) {
		zeros = uint64(precision) - uint64(count)
	}
	var prefix [2]byte
	prefixN := 0
	if signed {
		if negative {
			prefix[0] = '-'
			prefixN = 1
		} else if flags.plus {
			prefix[0] = '+'
			prefixN = 1
		} else if flags.space {
			prefix[0] = ' '
			prefixN = 1
		}
	} else if flags.alternate && base == 16 && value != 0 {
		prefix[0] = '0'
		prefix[1] = verb
		prefixN = 2
	}
	size := uint64(prefixN+count) + zeros
	padding := uint64(0)
	if width > 0 && uint64(width) > size {
		padding = uint64(width) - size
	}
	zeroWidth := flags.zero && !flags.left && !explicitPrecision
	if !flags.left && !zeroWidth {
		s.repeat(' ', padding)
	}
	s.bytes(prefix[:prefixN])
	if zeroWidth {
		s.repeat('0', padding)
	}
	s.repeat('0', zeros)
	s.bytes(digits[at:])
	if flags.left {
		s.repeat(' ', padding)
	}
	return nil
}

// Length codes: none0,hh1,h2,l3,ll4,j5,z6,t7. Unsupported standard
// domains are typed errors, not fake negative vsnprintf results/raw fallback.
func pfRender(buffer []byte, profile PrintfProfile, format string, args []PrintfArg) (int64, PrintfFailure, error) {
	return pfRenderSource(pfSink{buffer: buffer}, profile, format, args)
}
func pfRenderSource(sink pfSink, profile PrintfProfile, format string, args []PrintfArg) (count int64, failure PrintfFailure, result error) {
	defer func() {
		if sink.cause != nil {
			count = int64(sink.count)
			failure = PrintfArgumentFailure
			result = sink.cause
		}
	}()

	arg := 0
	malformed := func() (int64, PrintfFailure, error) {
		sink.finish()
		return -1, PrintfMalformedFailure, ErrPrintfMalformed
	}
	countOverflow := func() (int64, PrintfFailure, error) {
		sink.finish()
		return -1, PrintfCountFailure, ErrPrintfCountOverflow
	}
	typed := func(f PrintfFailure, e error) (int64, PrintfFailure, error) {
		sink.finish()
		return int64(sink.count), f, e
	}
	for i := 0; i < len(format); {
		if sink.cause != nil {
			return typed(PrintfArgumentFailure, sink.cause)
		}
		if format[i] != '%' {
			start := i
			for i < len(format) && format[i] != '%' {
				i++
			}
			sink.text(format[start:i])
			if sink.overflow {
				return countOverflow()
			}
			continue
		}
		i++
		if i == len(format) {
			return malformed()
		}
		if format[i] == '%' {
			sink.repeat('%', 1)
			if sink.overflow {
				return countOverflow()
			}
			i++
			continue
		}
		flags := pfFlags{}
		for i < len(format) {
			ch := format[i]
			switch ch {
			case '-':
				flags.left = true
			case '+':
				flags.plus = true
			case ' ':
				flags.space = true
			case '#':
				flags.alternate = true
			case '0':
				flags.zero = true
			case 'I': // GNU alternative digits: identity in the fixed C locale.
			case '\'':
				return typed(PrintfUnsupportedFailure, ErrPrintfUnsupported)
			default:
				goto parsedFlags
			}
			i++
		}
	parsedFlags:
		width := int64(0)
		precision := int64(-1)
		if i < len(format) && format[i] == '*' {
			i++
			if i < len(format) && format[i] >= '0' && format[i] <= '9' {
				return typed(PrintfUnsupportedFailure, ErrPrintfUnsupported)
			}
			n, e := pfStar(args, &arg)
			if e != nil {
				return typed(PrintfArgumentFailure, e)
			}
			width = n
			if width < 0 {
				flags.left = true
				width = -width
			}
		} else {
			n, ok := pfDecimal(format, &i)
			if !ok {
				return countOverflow()
			}
			width = n
		}
		if i < len(format) && format[i] == '$' {
			return typed(PrintfUnsupportedFailure, ErrPrintfUnsupported)
		}
		if i < len(format) && format[i] == '.' {
			i++
			precision = 0
			if i < len(format) && format[i] == '*' {
				i++
				if i < len(format) && format[i] >= '0' && format[i] <= '9' {
					return typed(PrintfUnsupportedFailure, ErrPrintfUnsupported)
				}
				n, e := pfStar(args, &arg)
				if e != nil {
					return typed(PrintfArgumentFailure, e)
				}
				precision = n
				if precision < 0 {
					precision = -1
				}
			} else {
				n, ok := pfDecimal(format, &i)
				if !ok {
					return countOverflow()
				}
				precision = n
			}
		}
		length := uint32(0)
		if i < len(format) {
			switch format[i] {
			case 'h':
				length = 2
				i++
				if i < len(format) && format[i] == 'h' {
					length = 1
					i++
				}
			case 'l':
				length = 3
				i++
				if i < len(format) && format[i] == 'l' {
					length = 4
					i++
				}
			case 'j':
				length = 5
				i++
			case 'z':
				length = 6
				i++
			case 't':
				length = 7
				i++
			case 'L':
				return typed(PrintfUnsupportedFailure, ErrPrintfUnsupported)
			}
		}
		if i >= len(format) {
			return malformed()
		}
		verb := format[i]
		i++
		if verb != 'd' && verb != 'i' && verb != 'u' && verb != 'o' && verb != 'x' && verb != 'X' && verb != 's' && verb != 'c' {
			return typed(PrintfUnsupportedFailure, ErrPrintfUnsupported)
		}
		if (verb == 's' || verb == 'c') && length != 0 {
			return typed(PrintfUnsupportedFailure, ErrPrintfUnsupported)
		}
		if arg >= len(args) {
			return typed(PrintfArgumentFailure, ErrPrintfArgument)
		}
		a := args[arg]
		arg++
		if verb == 's' {
			if a.kind != pfCString {
				return typed(PrintfArgumentFailure, ErrPrintfArgument)
			}
			if a.borrowedView {
				var n uint64
				for precision < 0 || n < uint64(precision) {
					v, e := a.view.Byte(n)
					if e != nil {
						return typed(PrintfArgumentFailure, e)
					}
					if v == 0 {
						break
					}
					n++
				}
				padding := uint64(0)
				if uint64(width) > n {
					padding = uint64(width) - n
				}
				if !flags.left {
					sink.repeat(' ', padding)
				}
				if e := sink.cstring(a.view, precision); e != nil {
					return typed(PrintfArgumentFailure, e)
				}
				if flags.left {
					sink.repeat(' ', padding)
				}
				if sink.overflow {
					return countOverflow()
				}
				continue
			}
			text := "(null)"
			if a.text != nil {
				text = (*a.text)[:cSymbolLength(*a.text)]
			} else if precision >= 0 && precision < 6 {
				text = ""
			}
			if precision >= 0 && uint64(precision) < uint64(len(text)) {
				text = text[:precision]
			}
			padding := uint64(0)
			if uint64(width) > uint64(len(text)) {
				padding = uint64(width) - uint64(len(text))
			}
			if !flags.left {
				sink.repeat(' ', padding)
			}
			sink.text(text)
			if flags.left {
				sink.repeat(' ', padding)
			}
		} else if verb == 'c' {
			if a.kind != pfChar && a.kind != pfInt32 && a.kind != pfUInt32 {
				return typed(PrintfArgumentFailure, ErrPrintfArgument)
			}
			padding := uint64(0)
			if width > 1 {
				padding = uint64(width - 1)
			}
			if !flags.left {
				sink.repeat(' ', padding)
			}
			sink.repeat(byte(a.bits), 1)
			if flags.left {
				sink.repeat(' ', padding)
			}
		} else {
			if e := pfInteger(&sink, a, verb, length, profile, flags, width, precision); e != nil {
				return typed(PrintfArgumentFailure, e)
			}
		}
		if sink.overflow {
			return countOverflow()
		}
	}
	sink.finish()
	return int64(sink.count), PrintfNoFailure, nil
}
