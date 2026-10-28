package ast

// Source: src/ast/bv_decl_plugin.cpp:632-634,746-751.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTBitVectorOriginalValueCallbacks(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	q := m.numerals
	n, e := q.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, q.Del(&n)) }()
	for _, bits := range []uint32{1, 8, 65, 257} {
		s, e := m.BitVectorSort(bits)
		astTestMust(t, e)
		got, e := m.GetSomeValue(s)
		astTestMust(t, e)
		astTestPin(t, m, got)
		width := uint32(0)
		found, e := m.BitVectorNumeralValue(got, &n, &width)
		astTestMust(t, e)
		zero, e := q.IsZero(&n)
		astTestMust(t, e)
		if !found || !zero || width != bits {
			t.Fatal("source bit-vector completion is zero numeral", bits, found, zero, width)
		}
		want, e := m.BitVectorNumeral(&n, bits)
		astTestMust(t, e)
		if got != want {
			t.Fatal("canonical numeral identity", bits)
		}
		for _, read := range []func(ASTHandle) (bool, error){m.IsValue, m.IsUniqueValue} {
			value, e := read(got)
			astTestMust(t, e)
			if !value {
				t.Fatal("bit-vector numeral is a unique value", bits)
			}
		}
		astTestMust(t, q.SetInt64(&n, 1))
		one, e := m.BitVectorNumeral(&n, bits)
		astTestMust(t, e)
		astTestPin(t, m, one)
		distinct, e := m.AreDistinct(got, one)
		astTestMust(t, e)
		if !distinct {
			t.Fatal("different unique numeral values", bits)
		}
		proc := arrayValueProc{values: map[ASTHandle]ASTHandle{s: one}}
		override, e := m.GetSomeValueWithProc(s, &proc)
		astTestMust(t, e)
		if override != one || len(proc.sorts) != 1 || m.someValueProc != nil {
			t.Fatal("source lexical override precedes plugin", bits)
		}
		proc = arrayValueProc{failureSort: s, failure: errors.New("numeric override refusal")}
		_, e = m.GetSomeValueWithProc(s, &proc)
		if !errors.Is(e, proc.failure) || m.someValueProc != nil {
			t.Fatal("override restoration", e)
		}
	}
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkConst(astTestSymbol(t, m, "bv-nonvalue"), s)
	astTestMust(t, e)
	value, e := m.IsValue(x)
	astTestMust(t, e)
	if value {
		t.Fatal("uninterpreted bit-vector constant is not a plugin value")
	}
}

func TestASTBitVectorSomeValueNumericalTransportOwnerGuard(t *testing.T) {
	if os.Getenv("G3_BV_VALUE_CALLBACK_CHILD") != "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestASTBitVectorSomeValueNumericalTransportOwnerGuard$")
		cmd.Env = append(os.Environ(), "G3_BV_VALUE_CALLBACK_CHILD=1")
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
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(257)
	astTestMust(t, e)
	astTestMust(t, rt.SetMaxCount(uint64(rt.State().GlobalCount)+1))
	active = true
	_, body := m.GetSomeValue(s)
	active = false
	if body == nil || calls == 0 || m.formatExecuting {
		t.Fatal("reached numerical transport/restoration", body, calls, m.formatExecuting)
	}
	astTestMust(t, rt.SetMaxCount(0))
	astTestMust(t, q.CleanupLexical())
	astTestMust(t, m.Close())
	astTestMust(t, q.Close())
	astTestMust(t, sy.Close())
	astTestMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("callback owner graph leak", b.Used())
	}
}

func TestASTBitVectorSomeValueEveryWarmReservation(t *testing.T) {
	succeeded := false
	for fault := int64(0); fault < 128; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			m := astTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			_, e = m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			s, e := m.BitVectorSort(65)
			astTestMust(t, e)
			want, e := m.GetSomeValue(s)
			astTestMust(t, e)
			astTestPin(t, m, want)
			proc := arrayValueProc{}
			m.Budget().FailAfter(fault)
			got, e := m.GetSomeValueWithProc(s, &proc)
			m.Budget().FailAfter(-1)
			if m.someValueProc != nil {
				t.Fatal("lexical callback retained", fault)
			}
			if e == nil {
				if got != want {
					t.Fatal("canonical zero changed", fault)
				}
				succeeded = true
			} else if !errors.Is(e, arena.ErrInjectedFailure) {
				var source umemory.Failure
				if !errors.As(e, &source) || source.Kind != umemory.FailureMemout {
					t.Fatal("warm reservation refusal", fault, e)
				}
			}
			// Failed numeric construction owners stay at their actual addresses
			// until manager cleanup. Retrying uses a separate continuation.
			got, e = m.GetSomeValue(s)
			astTestMust(t, e)
			if got != want {
				t.Fatal("zero callback retry", fault)
			}
		})
		if succeeded {
			return
		}
	}
	t.Fatal("warm zero construction never succeeded")
}
