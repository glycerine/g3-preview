// Source: src/ast/bv_decl_plugin.cpp:793-830.
package ast

import (
	"context"
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestASTBitVectorOriginalZeroAndAllOneRecognizers(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	n, e := m.numerals.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, m.numerals.Del(&n)) }()
	for _, width := range []uint32{1, 8, 65, 257} {
		for _, literal := range []string{"0", "1", "-1", "2"} {
			astTestMust(t, m.numerals.SetString(&n, literal))
			h, e := m.BitVectorNumeral(&n, width)
			astTestMust(t, e)
			astTestPin(t, m, h)
			zero, e := m.IsBitVectorZero(h)
			astTestMust(t, e)
			all, e := m.IsBitVectorAllOne(h)
			astTestMust(t, e)
			wantZero := literal == "0" || width == 1 && literal == "2"
			wantAll := literal == "-1" || width == 1 && literal == "1"
			if zero != wantZero || all != wantAll {
				t.Fatal("source numeral recognizers", width, literal, zero, all)
			}
		}
	}
}

func TestASTBitVectorOriginalRecognizerUnmatchedOutputsUnread(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	h, e := m.MkTrue()
	astTestMust(t, e)
	bit0, e := m.MkFamilyApp(fid, 2, nil, nil)
	astTestMust(t, e)
	astTestPin(t, m, bit0)
	for _, h := range []ASTHandle{h, bit0} {
		m.Budget().FailAfter(0)
		z, e := m.IsBitVectorZero(h)
		astTestMust(t, e)
		a, e := m.IsBitVectorAllOne(h)
		astTestMust(t, e)
		extract, e := m.BitVectorExtractValue(h, nil, nil, nil)
		astTestMust(t, e)
		converted, e := m.BitVectorToIntValue(h, nil)
		astTestMust(t, e)
		m.Budget().FailAfter(-1)
		if z || a || extract || converted {
			t.Fatal("unmatched branch must erase outputs/numerical operations")
		}
	}
}

func TestASTBitVectorOriginalExtractAndToIntRecognizers(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkConst(astTestSymbol(t, m, "bv-recognizer-base"), s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	hi, e := m.NewParameterInt(7)
	astTestMust(t, e)
	lo, e := m.NewParameterInt(2)
	astTestMust(t, e)
	extracted, e := m.MkFamilyApp(fid, 40, []ASTParameter{hi, lo}, []ASTHandle{x})
	astTestMust(t, e)
	astTestPin(t, m, extracted)
	converted, e := m.MkFamilyApp(fid, 58, nil, []ASTHandle{x})
	astTestMust(t, e)
	astTestPin(t, m, converted)
	astTestMust(t, m.DelParameter(&lo))
	astTestMust(t, m.DelParameter(&hi))
	low, high := uint32(99), uint32(100)
	var base ASTHandle
	m.Budget().FailAfter(0)
	found, e := m.BitVectorExtractValue(extracted, &low, &high, &base)
	astTestMust(t, e)
	if !found || low != 2 || high != 7 || base != x {
		t.Fatal("extract original low/high/base order", found, low, high, base)
	}
	base = ASTHandle{}
	found, e = m.BitVectorToIntValue(converted, &base)
	astTestMust(t, e)
	m.Budget().FailAfter(-1)
	if !found || base != x {
		t.Fatal("bv2int original operand", found, base)
	}
}

func TestASTBitVectorRecognizersDoNotNormalizeStoredParameters(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	for _, literal := range []string{"0", "255", "256", "-1"} {
		h := bvRawOffsetTestNumeral(t, m, fid, s, literal, 8)
		m.Budget().FailAfter(0)
		zero, e := m.IsBitVectorZero(h)
		m.Budget().FailAfter(-1)
		astTestMust(t, e)
		all, e := m.IsBitVectorAllOne(h)
		astTestMust(t, e)
		if zero != (literal == "0") || all != (literal == "255") {
			t.Fatal("source recognizer must not invent modulo normalization", literal, zero, all)
		}
	}
}

func TestASTBitVectorAllOneEveryActualNumericalReservation(t *testing.T) {
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
			value := bvRawOffsetTestNumeral(t, m, fid, s, "36893488147419103231", 65)
			m.Budget().FailAfter(ordinal)
			yes, body := m.IsBitVectorAllOne(value)
			m.Budget().FailAfter(-1)
			if body == nil {
				if !yes {
					t.Fatal("successful all-one result", ordinal)
				}
				success = true
				yes, e = m.IsBitVectorAllOne(value)
				astTestMust(t, e)
				if !yes {
					t.Fatal("successful context reuse")
				}
			} else if !errors.Is(body, arena.ErrInjectedFailure) {
				var source umemory.Failure
				if !errors.As(body, &source) || source.Kind != umemory.FailureMemout {
					t.Fatal("actual numeric refusal", ordinal, body)
				}
			}
		}()
		if success {
			if ordinal == 0 {
				t.Fatal("missing actual numeric allocation refusal")
			}
			t.Log("first completed numerical reservation", ordinal)
			return
		}
	}
	t.Fatal("unreached all-one recognition")
}

func TestASTBitVectorExtractRecognizerPreservesSignedToUnsignedBounds(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkConst(astTestSymbol(t, m, "negative-extract-recognizer"), s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	hi, e := m.NewParameterInt(0)
	astTestMust(t, e)
	lo, e := m.NewParameterInt(-1)
	astTestMust(t, e)
	h, e := m.MkFamilyApp(fid, 40, []ASTParameter{hi, lo}, []ASTHandle{x})
	astTestMust(t, e)
	astTestPin(t, m, h)
	astTestMust(t, m.DelParameter(&lo))
	astTestMust(t, m.DelParameter(&hi))
	low, high := uint32(0), uint32(99)
	var base ASTHandle
	found, e := m.BitVectorExtractValue(h, &low, &high, &base)
	astTestMust(t, e)
	if !found || low != ^uint32(0) || high != 0 || base != x {
		t.Fatal("source unsigned extract bounds", found, low, high, base)
	}
}

func TestASTBitVectorAllOneTransportOwnerGuard(t *testing.T) {
	if os.Getenv("G3_BV_ALLONE_CALLBACK_CHILD") != "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestASTBitVectorAllOneTransportOwnerGuard$")
		cmd.Env = append(os.Environ(), "G3_BV_ALLONE_CALLBACK_CHILD=1")
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
	q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	astTestMust(t, e)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
	astTestMust(t, e)
	m, e = NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	astTestMust(t, e)
	_, e = m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.BitVectorSort(65)
	astTestMust(t, e)
	n, e := q.NewValue()
	astTestMust(t, e)
	astTestMust(t, q.SetString(&n, "36893488147419103231"))
	value, e := m.BitVectorNumeral(&n, 65)
	astTestMust(t, e)
	ref, e := m.NewRef(value)
	astTestMust(t, e)
	astTestMust(t, rt.SetMaxCount(uint64(rt.State().GlobalCount)+1))
	active = true
	_, body := m.IsBitVectorAllOne(value)
	active = false
	if body == nil || calls == 0 || m.formatExecuting {
		t.Fatal("reached numerical transport/restoration", body, calls, m.formatExecuting)
	}
	astTestMust(t, rt.SetMaxCount(0))
	astTestMust(t, q.CleanupLexical())
	astTestMust(t, m.DelRef(&ref))
	astTestMust(t, q.Del(&n))
	astTestMust(t, m.Close())
	astTestMust(t, q.Close())
	astTestMust(t, sy.Close())
	astTestMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("callback owner graph leak", b.Used())
	}
}
