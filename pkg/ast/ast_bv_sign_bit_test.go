// Source: src/ast/bv_decl_plugin.cpp:767-773.
package ast

import (
	"context"
	"errors"
	"math/big"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTBitVectorOriginalSignBitNormalizesInput(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	n, e := m.numerals.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, m.numerals.Del(&n)) }()
	for _, width := range []uint32{1, 8, 65, 257} {
		modulus := new(big.Int).Lsh(big.NewInt(1), uint(width))
		threshold := new(big.Int).Rsh(new(big.Int).Set(modulus), 1)
		for _, integer := range []*big.Int{big.NewInt(0), big.NewInt(-1), big.NewInt(-257), new(big.Int).Sub(new(big.Int).Set(threshold), big.NewInt(1)), threshold, new(big.Int).Sub(new(big.Int).Set(modulus), big.NewInt(1)), modulus, new(big.Int).Add(new(big.Int).Set(modulus), threshold)} {
			text := integer.String()
			astTestMust(t, m.numerals.SetString(&n, text))
			got, e := m.BitVectorHasSignBit(&n, width)
			astTestMust(t, e)
			want := new(big.Int).Mod(integer, modulus).Cmp(threshold) >= 0
			if got != want {
				t.Fatal(width, text, got, want)
			}
			after, e := m.numerals.String(&n)
			astTestMust(t, e)
			if after != text {
				t.Fatal("modified borrowed input", text, after)
			}
		}
	}
}

func TestASTBitVectorSignBitZeroWidthAssertionBeforeInput(t *testing.T) {
	if !ASTSourceDebug {
		return
	}
	m := astTestManager(t)
	yes, e := m.BitVectorHasSignBit(nil, 0)
	if yes || !errors.Is(e, uparams.ErrGlobalParamsAssertion) {
		t.Fatal("zero-width assertion must precede input/plugin reads", yes, e)
	}
}

func TestASTBitVectorSignBitEveryNestedNumericalReservation(t *testing.T) {
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		success := false
		func() {
			m := bvOffsetRuntimeTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			_, e = m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			n, e := m.numerals.NewValue()
			astTestMust(t, e)
			defer func() { astTestMust(t, m.numerals.CleanupValue(&n)) }()
			astTestMust(t, m.numerals.SetString(&n, "-36893488147419103233"))
			m.Budget().FailAfter(ordinal)
			yes, body := m.BitVectorHasSignBit(&n, 65)
			m.Budget().FailAfter(-1)
			if body == nil {
				if !yes {
					t.Fatal("normalization lost sign bit", ordinal)
				}
				if m.bitVectorPlugin.numeralFrames != nil {
					t.Fatal("successful nested caller retained frames")
				}
				success = true
			} else if !errors.Is(body, arena.ErrInjectedFailure) {
				var f umemory.Failure
				if !errors.As(body, &f) || f.Kind != umemory.FailureMemout {
					t.Fatal(ordinal, body)
				}
			}
		}()
		if success {
			if ordinal == 0 {
				t.Fatal("missing numerical allocations")
			}
			t.Log("first completed nested reservation", ordinal)
			return
		}
	}
	t.Fatal("nested sign-bit reservation horizon exhausted")
}

func TestASTBitVectorSignBitTransportOwnerGuard(t *testing.T) {
	if os.Getenv("G3_BV_SIGNBIT_CALLBACK_CHILD") != "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestASTBitVectorSignBitTransportOwnerGuard$")
		cmd.Env = append(os.Environ(), "G3_BV_SIGNBIT_CALLBACK_CHILD=1")
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
	_, e = m.BitVectorSort(65)
	astTestMust(t, e)
	n, e := q.NewValue()
	astTestMust(t, e)
	astTestMust(t, q.SetString(&n, "18446744073709551617"))
	astTestMust(t, rt.SetMaxCount(uint64(rt.State().GlobalCount)+1))
	active = true
	_, body := m.BitVectorHasSignBit(&n, 65)
	active = false
	if body == nil || calls == 0 || m.formatExecuting {
		t.Fatal("reached numerical transport/restoration", body, calls, m.formatExecuting)
	}
	astTestMust(t, rt.SetMaxCount(0))
	astTestMust(t, q.CleanupLexical())
	astTestMust(t, q.Del(&n))
	astTestMust(t, m.Close())
	astTestMust(t, q.Close())
	astTestMust(t, sy.Close())
	astTestMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("callback owner graph leak", b.Used())
	}
}
