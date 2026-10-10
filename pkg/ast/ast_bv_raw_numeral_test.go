// Source: src/ast/bv_decl_plugin.cpp:416-435,584-596,865-868.
package ast

import (
	"context"
	"errors"
	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestASTBitVectorRawNumeralDeclarationDoesNotConstructApp(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	n, e := m.numerals.NewValue()
	astTestMust(t, e)
	astTestMust(t, m.numerals.SetInt64(&n, -1))
	value, e := m.NewParameterRational(&n)
	astTestMust(t, e)
	bits, e := m.NewParameterInt(8)
	astTestMust(t, e)
	ps := []ASTParameter{value, bits}
	before := m.Stats().LiveNodes
	d, e := m.MkFamilyFuncDecl(fid, 0, ps, nil, ASTHandle{})
	astTestMust(t, e)
	astTestPin(t, m, d)
	if m.Stats().LiveNodes != before+1 {
		t.Fatal("raw factory must create only a declaration", before, m.Stats().LiveNodes)
	}
	astTestMust(t, m.DelParameter(&bits))
	astTestMust(t, m.DelParameter(&value))
	h, e := m.BitVectorNumeral(&n, 8)
	astTestMust(t, e)
	astTestPin(t, m, h)
	got, e := m.GetDecl(h)
	astTestMust(t, e)
	if got != d {
		t.Fatal("utility and raw factory declaration identity", got, d)
	}
	astTestMust(t, m.numerals.Del(&n))
}

func TestASTBitVectorRawNumeralFactoryNormalizationAndValidation(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	q := m.numerals
	n, e := q.NewValue()
	astTestMust(t, e)
	out, e := q.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, q.Del(&out)); astTestMust(t, q.Del(&n)) }()
	for _, row := range []struct {
		literal string
		width   int32
		want    string
	}{
		{"-1", 1, "1"}, {"257", 8, "1"}, {"256", 8, "0"},
		{"36893488147419103233", 65, "1"}, {"-1", 65, "36893488147419103231"},
	} {
		astTestMust(t, q.SetString(&n, row.literal))
		value, e := m.NewParameterRational(&n)
		astTestMust(t, e)
		bits, e := m.NewParameterInt(row.width)
		astTestMust(t, e)
		ps := []ASTParameter{value, bits}
		d, e := m.MkFamilyFuncDecl(fid, 0, ps, nil, ASTHandle{})
		astTestMust(t, e)
		astTestPin(t, m, d)
		h, e := m.MkFamilyApp(fid, 0, ps, nil)
		astTestMust(t, e)
		astTestPin(t, m, h)
		got, e := m.GetDecl(h)
		astTestMust(t, e)
		if got != d {
			t.Fatal("both plugin overloads must normalize identically", row)
		}
		width := uint32(0)
		found, e := m.BitVectorNumeralValue(h, &out, &width)
		astTestMust(t, e)
		text, e := q.String(&out)
		astTestMust(t, e)
		if !found || width != uint32(row.width) || text != row.want {
			t.Fatal("normalized raw numeral", row, found, width, text)
		}
		_, e = m.MkFamilyFuncDecl(fid, 0, ps, []ASTHandle{{}}, ASTHandle{})
		if e == nil || e.Error() != "invalid bit-vector numeral declaration" {
			t.Fatal("arity checked before domain read", e)
		}
		astTestMust(t, m.DelParameter(&bits))
		astTestMust(t, m.DelParameter(&value))
	}
	serial := m.bitVectorPlugin.numeralSerial
	for _, ps := range [][]ASTParameter{nil, {{}}, {{}, {}, {}}} {
		_, e := m.MkFamilyFuncDecl(fid, 0, ps, nil, ASTHandle{})
		if e == nil || e.Error() != "invalid bit-vector numeral declaration" {
			t.Fatal("count erases parameter reads", e)
		}
	}
	if serial != m.bitVectorPlugin.numeralSerial {
		t.Fatal("invalid shape must precede numerical construction")
	}
}

func TestASTBitVectorRawNumeralEveryActualNumericalReservation(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		func() {
			m := bvOffsetRuntimeTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			fid, e := m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			_, e = m.BitVectorSort(65)
			astTestMust(t, e)
			n, e := m.numerals.NewValue()
			astTestMust(t, e)
			astTestMust(t, m.numerals.SetString(&n, "36893488147419103231"))
			value, e := m.NewParameterRational(&n)
			astTestMust(t, e)
			bits, e := m.NewParameterInt(65)
			astTestMust(t, e)
			ps := []ASTParameter{value, bits}
			m.Budget().FailAfter(ordinal)
			d, body := m.MkFamilyFuncDecl(fid, 0, ps, nil, ASTHandle{})
			m.Budget().FailAfter(-1)
			if body == nil {
				success = true
				astTestPin(t, m, d)
			} else if !errors.Is(body, arena.ErrInjectedFailure) {
				var source umemory.Failure
				if !errors.As(body, &source) || source.Kind != umemory.FailureMemout {
					t.Fatal(ordinal, body)
				}
			}
			astTestMust(t, m.DelParameter(&bits))
			astTestMust(t, m.DelParameter(&value))
			astTestMust(t, m.numerals.Del(&n))
		}()
		if success {
			if ordinal == 0 {
				t.Fatal("no numerical reservation reached")
			}
			return
		}
	}
	t.Fatal("unreached raw numeral declaration")
}

func TestASTBitVectorRawNumeralParameterTypeAndZeroWidthGuards(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	n, e := m.numerals.NewValue()
	astTestMust(t, e)
	rational, e := m.NewParameterRational(&n)
	astTestMust(t, e)
	integer, e := m.NewParameterInt(0)
	astTestMust(t, e)
	for _, row := range []struct {
		params  []ASTParameter
		message string
	}{
		{[]ASTParameter{integer, {}}, "invalid bit-vector numeral declaration"},
		{[]ASTParameter{rational, rational}, "invalid bit-vector numeral declaration"},
		{[]ASTParameter{rational, integer}, "bit-vector size must be greater than zero"},
	} {
		before := m.bitVectorPlugin.numeralSerial
		_, e := m.MkFamilyFuncDecl(fid, 0, row.params, nil, ASTHandle{})
		if e == nil || e.Error() != row.message {
			t.Fatal(row.message, e)
		}
		if m.bitVectorPlugin.numeralSerial != before {
			t.Fatal("guard must precede numerical operations")
		}
	}
	_, e = m.MkFamilyApp(fid, 0, []ASTParameter{rational, integer}, []ASTHandle{{}})
	if e == nil || e.Error() != "invalid bit-vector numeral declaration" {
		t.Fatal("expression overload must check arity before argument", e)
	}
	astTestMust(t, m.DelParameter(&integer))
	astTestMust(t, m.DelParameter(&rational))
	astTestMust(t, m.numerals.Del(&n))
}

func TestASTBitVectorRawNumeralTransportOwnerGuard(t *testing.T) {
	if os.Getenv("G3_BV_RAW_NUMERAL_CALLBACK_CHILD") != "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestASTBitVectorRawNumeralTransportOwnerGuard$")
		cmd.Env = append(os.Environ(), "G3_BV_RAW_NUMERAL_CALLBACK_CHILD=1")
		out, e := cmd.CombinedOutput()
		if e != nil {
			t.Fatal("bounded numerical callback", e, string(out))
		}
		return
	}
	b := arena.NewBudget(1 << 28)
	var m *ASTManager
	active, calls := false, 0
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if active {
			calls++
			if e := m.Close(); !errors.Is(e, ErrASTPending) {
				t.Fatal("executing AST owner cleanup", e)
			}
			if _, e := m.MkTrue(); !errors.Is(e, ErrASTPending) {
				t.Fatal("executing AST owner reentry", e)
			}
		}
		return f
	}})
	astTestMust(t, e)
	q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Memory: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	astTestMust(t, e)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
	astTestMust(t, e)
	m, e = NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	astTestMust(t, e)
	_, e = m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.BitVectorSort(65)
	astTestMust(t, e)
	n, e := q.NewValue()
	astTestMust(t, e)
	astTestMust(t, q.SetString(&n, "36893488147419103231"))
	value, e := m.NewParameterRational(&n)
	astTestMust(t, e)
	bits, e := m.NewParameterInt(65)
	astTestMust(t, e)
	astTestMust(t, rt.SetMaxCount(uint64(rt.State().GlobalCount)+1))
	active = true
	_, body := m.MkFamilyFuncDecl(fid, 0, []ASTParameter{value, bits}, nil, ASTHandle{})
	active = false
	if body == nil || calls == 0 || m.formatExecuting {
		t.Fatal("reached numerical transport/restoration", body, calls, m.formatExecuting)
	}
	astTestMust(t, rt.SetMaxCount(0))
	astTestMust(t, q.CleanupLexical())
	astTestMust(t, m.DelParameter(&bits))
	astTestMust(t, m.DelParameter(&value))
	astTestMust(t, q.Del(&n))
	astTestMust(t, m.Close())
	astTestMust(t, q.Close())
	astTestMust(t, sy.Close())
	astTestMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("callback owner graph leak", b.Used())
	}
}
