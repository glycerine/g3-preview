// Source: src/ast/bv_decl_plugin.cpp:827-848.
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
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTBitVectorOriginalMultiplicativeInverse(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	q := m.numerals
	n, e := q.NewValue()
	astTestMust(t, e)
	out, e := q.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, q.Del(&n)); astTestMust(t, q.Del(&out)) }()
	for _, width := range []uint32{1, 8, 65, 257} {
		for _, text := range []string{"1", "0", "2", "3", "17", "-3", "-17", "18446744073709551617"} {
			astTestMust(t, q.SetString(&n, text))
			astTestMust(t, q.SetInt64(&out, 99))
			yes, e := m.BitVectorMultiplicativeInverse(&n, width, &out)
			astTestMust(t, e)
			integer, _ := new(big.Int).SetString(text, 10)
			want := new(big.Int).ModInverse(integer, new(big.Int).Lsh(big.NewInt(1), uint(width)))
			actual, e := q.String(&out)
			astTestMust(t, e)
			if want == nil {
				if yes || actual != "99" {
					t.Fatal("even branch changed output", width, text, yes, actual)
				}
			} else if !yes || actual != want.String() {
				t.Fatal("inverse", width, text, yes, actual, want)
			}
		}
	}
	astTestMust(t, q.SetInt64(&n, 17))
	yes, e := m.BitVectorMultiplicativeInverse(&n, 8, &n)
	astTestMust(t, e)
	actual, e := q.String(&n)
	astTestMust(t, e)
	if !yes || actual != "241" {
		t.Fatal("aliased result", yes, actual)
	}
	// One skips all width-dependent work, even at zero width.
	astTestMust(t, q.SetInt64(&n, 1))
	yes, e = m.BitVectorMultiplicativeInverse(&n, 0, &out)
	astTestMust(t, e)
	actual, e = q.String(&out)
	astTestMust(t, e)
	if !yes || actual != "1" {
		t.Fatal("one fast path", yes, actual)
	}
	astTestMust(t, q.SetInt64(&n, 2))
	yes, e = m.BitVectorMultiplicativeInverse(&n, ^uint32(0), nil)
	astTestMust(t, e)
	if yes {
		t.Fatal("even branch read unused result")
	}
}

func TestASTBitVectorInverseZeroWidthSourceAssertion(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	n, e := m.numerals.NewValue()
	astTestMust(t, e)
	out, e := m.numerals.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, m.numerals.Del(&n)); astTestMust(t, m.numerals.Del(&out)) }()
	astTestMust(t, m.numerals.SetInt64(&n, 3))
	astTestMust(t, m.numerals.SetInt64(&out, 99))
	yes, body := m.BitVectorMultiplicativeInverse(&n, 0, &out)
	text, e := m.numerals.String(&out)
	astTestMust(t, e)
	if ASTSourceDebug {
		if yes || !errors.Is(body, util.ErrGlobalParamsAssertion) || text != "99" {
			t.Fatal("source positivity assertion", yes, body, text)
		}
	} else if !yes || body != nil || text != "0" {
		t.Fatal("release erases source assertions", yes, body, text)
	}
}

func TestASTBitVectorInverseEveryNumericalReservation(t *testing.T) {
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		success := false
		func() {
			m := bvOffsetRuntimeTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			_, e = m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			q := m.numerals
			n, e := q.NewValue()
			astTestMust(t, e)
			out, e := q.NewValue()
			astTestMust(t, e)
			defer func() { astTestMust(t, q.CleanupValue(&out)); astTestMust(t, q.CleanupValue(&n)) }()
			astTestMust(t, q.SetString(&n, "18446744073709551617"))
			m.Budget().FailAfter(ordinal)
			yes, body := m.BitVectorMultiplicativeInverse(&n, 65, &out)
			m.Budget().FailAfter(-1)
			if body == nil {
				if !yes {
					t.Fatal("odd inverse rejected", ordinal)
				}
				text, e := q.String(&out)
				astTestMust(t, e)
				if text != "18446744073709551617" {
					t.Fatal("inverse result", ordinal, text)
				}
				success = true
			} else if !errors.Is(body, arena.ErrInjectedFailure) {
				var f umemory.Failure
				if !errors.As(body, &f) || f.Kind != umemory.FailureMemout {
					t.Fatal("unexpected refusal", ordinal, body)
				}
			}
		}()
		if success {
			if ordinal == 0 {
				t.Fatal("missing numerical reservations")
			}
			t.Log("first completed reservation", ordinal)
			return
		}
	}
	t.Fatal("inverse reservation horizon exhausted")
}

func TestASTBitVectorInverseTransportOwnerGuard(t *testing.T) {
	if os.Getenv("G3_BV_INVERSE_CALLBACK_CHILD") != "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestASTBitVectorInverseTransportOwnerGuard$")
		cmd.Env = append(os.Environ(), "G3_BV_INVERSE_CALLBACK_CHILD=1")
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
	out, e := q.NewValue()
	astTestMust(t, e)
	astTestMust(t, rt.SetMaxCount(uint64(rt.State().GlobalCount)+1))
	active = true
	_, body := m.BitVectorMultiplicativeInverse(&n, 65, &out)
	active = false
	if body == nil || calls == 0 || m.formatExecuting {
		t.Fatal("reached numerical transport/restoration", body, calls, m.formatExecuting)
	}
	astTestMust(t, rt.SetMaxCount(0))
	astTestMust(t, q.CleanupLexical())
	astTestMust(t, q.CleanupValue(&out))
	astTestMust(t, q.Del(&n))
	astTestMust(t, m.Close())
	astTestMust(t, q.Close())
	astTestMust(t, sy.Close())
	astTestMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("callback owner graph leak", b.Used())
	}
}
