// Concrete cmd_context_types.h exception values; no command runtime scaffold.
package smtcore

import (
	"strconv"
	"strings"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

type CmdException struct {
	util.DefaultException
	line, pos int32
}

func NewCmdException(message string) CmdException {
	return CmdException{util.NewDefaultException(message), -1, -1}
}
func NewCmdExceptionCString(message string) CmdException {
	if i := strings.IndexByte(message, 0); i >= 0 {
		message = message[:i]
	}
	return NewCmdException(message)
}
func NewCmdExceptionAt(message string, line, pos int32) CmdException {
	return CmdException{util.NewDefaultException(message), line, pos}
}
func NewCmdExceptionSymbol(m *util.SymbolManager, message string, s util.Symbol) (CmdException, error) {
	return NewCmdExceptionSymbolAt(m, message, s, -1, -1)
}

// The manager is borrowed during construction only. Checked spelling bytes go
// directly into one final caller-owned message; no Symbol.String allocation or
// owner mutation/charge occurs. Returned exceptions survive manager Close.
func NewCmdExceptionSymbolAt(m *util.SymbolManager, message string, s util.Symbol, line, pos int32) (CmdException, error) {
	if m == nil {
		return CmdException{}, util.ErrExceptionPrecondition
	}
	if _, e := m.FromToken(s.Token()); e != nil {
		return CmdException{}, e
	}
	if s.IsMarked() {
		return CmdException{}, util.ErrInvalidSymbol
	}
	prefix := message
	if i := strings.IndexByte(message, 0); i >= 0 {
		prefix = message[:i]
	}
	var digits [12]byte
	var numeric []byte
	length := uint64(4)
	var view util.ParamsStringView
	if s.IsNumerical() {
		number, e := s.Number()
		if e != nil {
			return CmdException{}, e
		}
		numeric = strconv.AppendInt(digits[:0], int64(int32(number)), 10)
		length = uint64(2 + len(numeric))
	} else if !s.IsNull() {
		var e error
		view, e = m.CStringView(s)
		if e != nil {
			return CmdException{}, e
		}
		length, e = view.Length()
		if e != nil {
			return CmdException{}, e
		}
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
	} else if s.IsNull() {
		out.WriteString("null")
	} else {
		for i := uint64(0); i < length; i++ {
			v, e := view.Byte(i)
			if e != nil {
				return CmdException{}, e
			}
			out.WriteByte(v)
		}
	}
	return NewCmdExceptionAt(out.String(), line, pos), nil
}
func (e CmdException) HasPos() bool { return e.line >= 0 }
func (e CmdException) Line() (int32, error) {
	if !e.HasPos() {
		return 0, util.ErrExceptionPrecondition
	}
	return e.line, nil
}
func (e CmdException) Pos() (int32, error) {
	if !e.HasPos() {
		return 0, util.ErrExceptionPrecondition
	}
	return e.pos, nil
}

// StopParserException has no source message/code and is not a Z3Exception. Its
// Error method supplies a Go-only transport label for this distinct signal.
type StopParserException struct{}

func (StopParserException) Error() string { return "g3: stop parser" }
