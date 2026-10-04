// Licensed under the MIT license.
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
// Concrete cmd_context_types.h exception values; no command runtime scaffold.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"strconv"
	"strings"
)

type CmdException struct {
	DefaultException
	line, pos int32
}

func NewCmdException(message string) CmdException {
	return CmdException{NewDefaultException(message), -1, -1}
}
func NewCmdExceptionCString(message string) CmdException {
	return NewCmdException(message[:cSymbolLength(message)])
}
func NewCmdExceptionAt(message string, line, pos int32) CmdException {
	return CmdException{NewDefaultException(message), line, pos}
}
func NewCmdExceptionSymbol(m *SymbolManager, message string, s Symbol) (CmdException, error) {
	return NewCmdExceptionSymbolAt(m, message, s, -1, -1)
}

// The manager is borrowed during construction only. Checked spelling bytes go
// directly into one final caller-owned message; no Symbol.String allocation or
// owner mutation/charge occurs. Returned exceptions survive manager Close.
func NewCmdExceptionSymbolAt(m *SymbolManager, message string, s Symbol, line, pos int32) (CmdException, error) {
	if m == nil {
		return CmdException{}, ErrExceptionPrecondition
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.checkPlain(s); e != nil {
		return CmdException{}, e
	}
	prefix := message[:cSymbolLength(message)]
	var digits [12]byte
	var numeric []byte
	length := uint64(4)
	if s.IsNumerical() {
		numeric = strconv.AppendInt(digits[:0], int64(int32(s.value)), 10)
		length = uint64(2 + len(numeric))
	} else if s.value != 0 {
		k, e := m.key(s)
		if e != nil {
			return CmdException{}, e
		}
		length = uint64(k.Length())
	}
	maxInt := uint64(^uint(0) >> 1)
	if length > maxInt || uint64(len(prefix)) > maxInt-length {
		return CmdException{}, arena.ErrBounds
	}
	var out strings.Builder
	out.Grow(len(prefix) + int(length))
	out.WriteString(prefix)
	if s.IsNumerical() {
		out.WriteString("k!")
		out.Write(numeric)
	} else if s.value == 0 {
		out.WriteString("null")
	} else {
		r := arena.BytesRegionRef{Slot: arena.Slot(s.value), Generation: s.generation}
		for i := uint32(0); i < uint32(length); i++ {
			b, e := m.region.Read(r, m.pointerWidth+i)
			if e != nil {
				return CmdException{}, e
			}
			out.WriteByte(b)
		}
	}
	return NewCmdExceptionAt(out.String(), line, pos), nil
}
func (e CmdException) HasPos() bool { return e.line >= 0 }
func (e CmdException) Line() (int32, error) {
	if !e.HasPos() {
		return 0, ErrExceptionPrecondition
	}
	return e.line, nil
}
func (e CmdException) Pos() (int32, error) {
	if !e.HasPos() {
		return 0, ErrExceptionPrecondition
	}
	return e.pos, nil
}

// StopParserException has no source message/code and is not a Z3Exception. Its
// Error method supplies a Go-only transport label for this distinct signal.
type StopParserException struct{}

func (StopParserException) Error() string { return "g3: stop parser" }
