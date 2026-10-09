// Source: src/ast/bv_decl_plugin.cpp:636-669.
package ast

import (
	"context"
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestASTBitVectorOriginalOffsetDistinctness(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	q := m.numerals
	n, e := q.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, q.Del(&n)) }()
	for _, width := range []uint32{1, 8, 65, 257} {
		s, e := m.BitVectorSort(width)
		astTestMust(t, e)
		x, e := m.MkConst(astTestSymbol(t, m, "bv-offset-base"), s)
		astTestMust(t, e)
		astTestPin(t, m, x)
		y, e := m.MkConst(astTestSymbol(t, m, "bv-offset-other"), s)
		astTestMust(t, e)
		astTestPin(t, m, y)
		values := make([]ASTHandle, 3)
		for i, literal := range []string{"0", "1", "-1"} {
			astTestMust(t, q.SetString(&n, literal))
			values[i], e = m.BitVectorNumeral(&n, width)
			astTestMust(t, e)
			astTestPin(t, m, values[i])
		}
		sums := make([]ASTHandle, 3)
		for i, v := range values {
			sums[i], e = m.MkFamilyApp(fid, 4, nil, []ASTHandle{v, x})
			astTestMust(t, e)
			astTestPin(t, m, sums[i])
		}
		for i, a := range sums {
			for j, b := range sums {
				got, e := m.AreDistinct(a, b)
				astTestMust(t, e)
				want := i != j && !(width == 1 && i != 0 && j != 0)
				if got != want {
					t.Fatal("original normalized offset comparison", width, i, j, got, want)
				}
			}
			for _, pair := range [][2]ASTHandle{{a, x}, {x, a}} {
				got, e := m.AreDistinct(pair[0], pair[1])
				astTestMust(t, e)
				if got != (i != 0) {
					t.Fatal("bare base and offset", width, i, got)
				}
			}
			got, e := m.AreDistinct(a, y)
			astTestMust(t, e)
			if got {
				t.Fatal("distinct bases do not prove distinctness")
			}
		}
		reverse, e := m.MkFamilyApp(fid, 4, nil, []ASTHandle{x, values[1]})
		astTestMust(t, e)
		astTestPin(t, m, reverse)
		three, e := m.MkFamilyApp(fid, 4, nil, []ASTHandle{values[1], x, values[1]})
		astTestMust(t, e)
		astTestPin(t, m, three)
		symbolic, e := m.MkFamilyApp(fid, 4, nil, []ASTHandle{y, x})
		astTestMust(t, e)
		astTestPin(t, m, symbolic)
		for _, a := range []ASTHandle{reverse, three, symbolic} {
			for _, b := range []ASTHandle{x, sums[1]} {
				got, e := m.AreDistinct(a, b)
				astTestMust(t, e)
				if got {
					t.Fatal("original positional/arity limitation", width, a, b)
				}
			}
		}
		got, e := m.AreDistinct(values[0], values[1])
		astTestMust(t, e)
		if !got {
			t.Fatal("base unique-value fallback")
		}
	}
}

func bvRawOffsetTestNumeral(t *testing.T, m *ASTManager, fid FamilyID, s ASTHandle, literal string, width uint32) ASTHandle {
	t.Helper()
	n, e := m.numerals.NewValue()
	astTestMust(t, e)
	astTestMust(t, m.numerals.SetString(&n, literal))
	value, e := m.NewParameterRational(&n)
	astTestMust(t, e)
	bits, e := m.NewParameterUint(width)
	astTestMust(t, e)
	info, e := m.NewFuncDeclInfo(fid, 0, []ASTParameter{value, bits}, ASTFuncFlags{}, false)
	astTestMust(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "bv"), nil, s, info)
	astTestMust(t, e)
	astTestMust(t, m.DelFuncDeclInfo(&info))
	astTestMust(t, m.DelParameter(&bits))
	astTestMust(t, m.DelParameter(&value))
	astTestMust(t, m.numerals.Del(&n))
	h, e := m.MkApp(d, nil)
	astTestMust(t, e)
	astTestPin(t, m, h)
	return h
}

func TestASTBitVectorOffsetRecomputesRawNumeralModulo(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkConst(astTestSymbol(t, m, "raw-bv-offset-base"), s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	for _, row := range []struct {
		a, b string
		want bool
	}{{"257", "1", false}, {"-1", "255", false}, {"256", "0", false}, {"257", "2", true}} {
		a := bvRawOffsetTestNumeral(t, m, fid, s, row.a, 8)
		b := bvRawOffsetTestNumeral(t, m, fid, s, row.b, 8)
		left, e := m.MkFamilyApp(fid, 4, nil, []ASTHandle{a, x})
		astTestMust(t, e)
		astTestPin(t, m, left)
		right, e := m.MkFamilyApp(fid, 4, nil, []ASTHandle{b, x})
		astTestMust(t, e)
		astTestPin(t, m, right)
		got, e := m.AreDistinct(left, right)
		astTestMust(t, e)
		if got != row.want {
			t.Fatal("original mod must be recomputed", row, got)
		}
		// Without ADD, the original base callback compares unique declaration identity.
		got, e = m.AreDistinct(a, b)
		astTestMust(t, e)
		if !got {
			t.Fatal("raw unequal numeral declarations retain base callback behavior", row)
		}
	}
}

func TestASTBitVectorOffsetEveryWarmNumericalReservation(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		func() {
			m := bvOffsetRuntimeTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			fid, e := m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			s, e := m.BitVectorSort(65)
			astTestMust(t, e)
			x, e := m.MkConst(astTestSymbol(t, m, "fault-bv-offset-base"), s)
			astTestMust(t, e)
			astTestPin(t, m, x)
			n := bvRawOffsetTestNumeral(t, m, fid, s, "36893488147419103231", 65)
			sum, e := m.MkFamilyApp(fid, 4, nil, []ASTHandle{n, x})
			astTestMust(t, e)
			astTestPin(t, m, sum)
			next := bvRawOffsetTestNumeral(t, m, fid, s, "36893488147419103230", 65)
			right, e := m.MkFamilyApp(fid, 4, nil, []ASTHandle{next, x})
			astTestMust(t, e)
			astTestPin(t, m, right)
			got, e := m.AreDistinct(sum, right)
			astTestMust(t, e)
			if !got {
				t.Fatal("warm offset setup")
			}
			m.Budget().FailAfter(ordinal)
			got, body := m.AreDistinct(sum, right)
			m.Budget().FailAfter(-1)
			if body == nil {
				if !got {
					t.Fatal("successful offset result", ordinal)
				}
				success = true
			} else if !errors.Is(body, arena.ErrInjectedFailure) {
				var source util.Failure
				if !errors.As(body, &source) || source.Kind != util.FailureMemout {
					t.Fatal("numerical reservation refusal", ordinal, body)
				}
			}
			if body == nil {
				got, e = m.AreDistinct(sum, right)
				astTestMust(t, e)
				if !got {
					t.Fatal("completed context reuse", ordinal)
				}
			}
			// Runtime memout is process-fatal in C++. This returning transport
			// profile retains interrupted raw numeric owners for whole-context
			// owning cleanup; it does not authorize another source operation.

		}()
		if success {
			if ordinal == 0 {
				t.Fatal("no actual numeric allocation refusal reached")
			}
			t.Log("first completed numeric reservation", ordinal)
			return
		}
	}
	t.Fatal("unreached successful numeric continuation")
}

func TestASTBitVectorOffsetIdenticalOperandsStillReachNumericalCopy(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(65)
	astTestMust(t, e)
	x, e := m.MkConst(astTestSymbol(t, m, "identical-bv-offset"), s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	n := bvRawOffsetTestNumeral(t, m, fid, s, "36893488147419103231", 65)
	sum, e := m.MkFamilyApp(fid, 4, nil, []ASTHandle{n, x})
	astTestMust(t, e)
	astTestPin(t, m, sum)
	serial := m.bitVectorPlugin.numeralSerial
	got, e := m.AreDistinct(sum, sum)
	astTestMust(t, e)
	if m.bitVectorPlugin.numeralSerial != serial+1 {
		t.Fatal("identity must not bypass original offset extraction")
	}
	for i := 0; i < 8; i++ {
		body, source, own := m.bitVectorPlugin.numeralPrimary.children[i].Causes()
		if body != nil || source != nil || own != nil {
			t.Fatal("unreached or incomplete original numeric child", i, body, source, own)
		}
	}
	if got {
		t.Fatal("identical offsets")
	}
}

func TestASTBitVectorOffsetNumericalTransportOwnerGuard(t *testing.T) {
	if os.Getenv("G3_BV_OFFSET_CALLBACK_CHILD") != "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestASTBitVectorOffsetNumericalTransportOwnerGuard$")
		cmd.Env = append(os.Environ(), "G3_BV_OFFSET_CALLBACK_CHILD=1")
		out, e := cmd.CombinedOutput()
		if e != nil {
			t.Fatal("bounded numerical callback", e, string(out))
		}
		return
	}
	b := arena.NewBudget(1 << 28)
	var m *ASTManager
	active, calls := false, 0
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64, Transport: func(f util.Failure) error {
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
	q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: rt, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	astTestMust(t, e)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
	astTestMust(t, e)
	m, e = NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	astTestMust(t, e)
	_, e = m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(65)
	astTestMust(t, e)
	x, e := m.MkConst(astTestSymbol(t, m, "transport-bv-offset"), s)
	astTestMust(t, e)
	ref, e := m.NewRef(x)
	astTestMust(t, e)
	n, e := q.NewValue()
	astTestMust(t, e)
	astTestMust(t, q.SetString(&n, "36893488147419103231"))
	value, e := m.BitVectorNumeral(&n, 65)
	astTestMust(t, e)
	astTestMust(t, q.Del(&n))
	sum, e := m.MkFamilyApp(fid, 4, nil, []ASTHandle{value, x})
	astTestMust(t, e)
	sumRef, e := m.NewRef(sum)
	astTestMust(t, e)
	astTestMust(t, rt.SetMaxCount(uint64(rt.State().GlobalCount)+1))
	active = true
	_, body := m.AreDistinct(sum, x)
	active = false
	if body == nil || calls == 0 || m.formatExecuting {
		t.Fatal("reached numerical transport/restoration", body, calls, m.formatExecuting)
	}
	astTestMust(t, rt.SetMaxCount(0))
	astTestMust(t, q.CleanupLexical())
	astTestMust(t, m.DelRef(&sumRef))
	astTestMust(t, m.DelRef(&ref))
	astTestMust(t, m.Close())
	astTestMust(t, q.Close())
	astTestMust(t, sy.Close())
	astTestMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("callback owner graph leak", b.Used())
	}
}

func bvOffsetRuntimeTestManager(t *testing.T) *ASTManager {
	t.Helper()
	b := arena.NewBudget(1 << 28)
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64, Transport: func(f util.Failure) error { return f }})
	astTestMust(t, e)
	q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: rt, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	astTestMust(t, e)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
	astTestMust(t, e)
	m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	astTestMust(t, e)
	t.Cleanup(func() {
		astTestMust(t, q.CleanupLexical())
		astTestMust(t, m.Close())
		astTestMust(t, q.Close())
		astTestMust(t, sy.Close())
		astTestMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("actual Runtime graph retained", b.Used())
		}
	})
	return m
}
