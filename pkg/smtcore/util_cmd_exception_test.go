package smtcore

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestCmdExceptionSourceConstructorRoutesAndLocationGuards(t *testing.T) {
	for _, text := range []string{"message", "x\x00y", "", "\xff"} {
		a := NewCmdException(text)
		b := NewCmdExceptionCString(text)
		if a.MessageField != text || b.MessageField != strings.SplitN(text, "\x00", 2)[0] || a.HasPos() || b.HasPos() {
			t.Fatal(a, b)
		}
		line, lineErr := a.Line()
		pos, posErr := a.Pos()
		if cmdExceptionSourceDebug {
			if lineErr != z3exception.ErrExceptionPrecondition || posErr != z3exception.ErrExceptionPrecondition {
				t.Fatal(lineErr, posErr)
			}
		} else if lineErr != nil || posErr != nil || line != -1 || pos != -1 {
			t.Fatal(line, pos, lineErr, posErr)
		}
		for _, line := range []int32{-2147483648, -1, 0, 1, 2147483647} {
			for _, pos := range []int32{-2147483648, -1, 0, 5, 2147483647} {
				x := NewCmdExceptionAt(text, line, pos)
				if x.HasPos() != (line >= 0) || x.ErrorCode() != z3exception.ErrorOK || x.HasErrorCode() {
					t.Fatal(x)
				}
				l, le := x.Line()
				p, pe := x.Pos()
				if line >= 0 {
					if le != nil || pe != nil || l != line || p != pos {
						t.Fatal(x, l, p, le, pe)
					}
				} else if cmdExceptionSourceDebug {
					if le != z3exception.ErrExceptionPrecondition || pe != z3exception.ErrExceptionPrecondition {
						t.Fatal(le, pe)
					}
				} else if le != nil || pe != nil || l != line || p != pos {
					t.Fatal(l, p, le, pe)
				}
			}
		}
	}
	var err error = StopParserException{}
	var stop StopParserException
	if !errors.As(err, &stop) {
		t.Fatal(err)
	}
	var def z3exception.DefaultException
	if errors.As(err, &def) {
		t.Fatal("stop parser distinct")
	}
}
func TestCmdExceptionSourceSymbolCompositionAndBorrowedLifetime(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	other := symbolManagerTest(t, nil, false)
	for _, s := range []usymbol.Symbol{usymbol.Symbol{}, usymbol.Numerical(0), usymbol.Numerical(17), usymbol.Numerical(0x7fffffff), usymbol.Numerical(0x80000000), usymbol.Numerical(^uint32(0)), internSymbol(t, m, "name\x00suffix"), internSymbol(t, m, "\xff-name")} {
		expected := "null"
		if s.IsNumerical() {
			expected = fmt.Sprintf("k!%d", int32(s.Value))
		} else if s != (usymbol.Symbol{}) {
			expected = symbolText(t, m, s)
		}
		x, e := NewCmdExceptionSymbol(m, "prefix\x00ignored", s)
		if e != nil || x.Message() != "prefix"+expected || x.HasPos() {
			t.Fatal(s, x, e)
		}
		y, e := NewCmdExceptionSymbolAt(m, "text:", s, 0, -1)
		if e != nil || y.Message() != "text:"+expected || !y.HasPos() {
			t.Fatal(y, e)
		}
		if pos, e := y.Pos(); e != nil || pos != -1 {
			t.Fatal(pos, e)
		}
	}
	foreign := internSymbol(t, other, "foreign")
	if _, e := NewCmdExceptionSymbol(m, "x", foreign); e != usymbol.ErrOwner {
		t.Fatal(e)
	}
	if _, e := NewCmdExceptionSymbol(m, "x", usymbol.Dummy()); e != usymbol.ErrInvalid {
		t.Fatal(e)
	}
	if _, e := NewCmdExceptionSymbol(nil, "x", usymbol.Symbol{}); e != z3exception.ErrExceptionPrecondition {
		t.Fatal(e)
	}
	x, e := NewCmdExceptionSymbol(m, "owner:", internSymbol(t, m, "alive"))
	if e != nil {
		t.Fatal(e)
	}
	m.Close()
	if x.Message() != "owner:alive" {
		t.Fatal(x)
	}
	if _, e = NewCmdExceptionSymbol(m, "", usymbol.Symbol{}); e != arena.ErrClosed {
		t.Fatal(e)
	}
}
func TestCmdExceptionIndependentByteDomainsAndOwnerAccounting(t *testing.T) {
	for _, width := range []uint32{4, 8} {
		b := arena.NewBudget(^uint64(0))
		m, e := usymbol.NewManager(usymbol.Config{Budget: b, Alignment: width, PageSize: 64, DescriptorPageSize: 2})
		if e != nil {
			t.Fatal(e)
		}
		for i := 1; i < 256; i++ {
			text := string([]byte{byte(i), 'X'})
			s, e := m.Intern(text)
			if e != nil {
				t.Fatal(e)
			}
			base := b.Used()
			b.FailAfter(0)
			x, e := NewCmdExceptionSymbolAt(m, "A", s, int32(i), -int32(i))
			b.FailAfter(-1)
			if e != nil || x.Message() != "A"+text || b.Used() != base {
				t.Fatal(width, i, x, e, b.Used(), base)
			}
		}
		m.Close()
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
}
