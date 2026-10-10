// Source: src/ast/rewriter/bv_rewriter.cpp:2141-2168.
package ast

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestASTBitVectorOriginalZeroBitStoredNumerals(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	for _, width := range []uint32{1, 8, 65} {
		sort, e := m.BitVectorSort(width)
		astTestMust(t, e)
		for _, literal := range []string{"0", "1", "12", "257", "-3", "-257", "340282366920938463463374607431768211457"} {
			input := bvRawOffsetTestNumeral(t, m, fid, sort, literal, width)
			v, ok := new(big.Int).SetString(literal, 10)
			if !ok {
				t.Fatal("fixture integer")
			}
			for _, idx := range []uint32{0, 1, 2, 7, 8, 64, 65, 128, 129} {
				got, e := m.BitVectorZeroBit(input, idx)
				astTestMust(t, e)
				mod := new(big.Int).Lsh(big.NewInt(1), uint(idx+1))
				bits := new(big.Int).Mod(v, mod)
				want := bits.Bit(int(idx)) == 0
				if got != want {
					t.Fatal("stored bit", width, literal, idx, got, want)
				}
			}
		}
	}
}

func TestASTBitVectorZeroBitConcatRoutingAndErasedReads(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	num := func(v, w uint32) ASTHandle {
		h, e := m.BitVectorNumeralUnsigned(v, w)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	concat := func(a, b ASTHandle) ASTHandle {
		h, e := m.MkBitVectorConcatPair(a, b)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	sort, e := m.BitVectorSort(4)
	astTestMust(t, e)
	x, e := m.MkVar(0, sort)
	astTestMust(t, e)
	astTestPin(t, m, x)
	input := concat(num(10, 4), concat(x, num(5, 4)))
	for idx := uint32(0); idx < 12; idx++ {
		got, e := m.BitVectorZeroBit(input, idx)
		astTestMust(t, e)
		want := false
		if idx < 4 {
			want = (5>>idx)&1 == 0
		}
		if idx >= 8 {
			want = (10>>(idx-8))&1 == 0
		}
		if got != want {
			t.Fatal("nested reverse concat routing", idx, got, want)
		}
	}
	before := m.Stats()
	used := m.Budget().Used()
	m.Budget().FailAfter(0)
	for _, idx := range []uint32{0, 3, 4, 7, ^uint32(0)} {
		got, e := m.BitVectorZeroBit(x, idx)
		astTestMust(t, e)
		if got {
			t.Fatal("unknown bit reported zero")
		}
	}
	for _, idx := range []uint32{4, 7} {
		got, e := m.BitVectorZeroBit(input, idx)
		astTestMust(t, e)
		if got {
			t.Fatal("routed unknown bit reported zero")
		}
	}
	m.Budget().FailAfter(-1)
	if before != m.Stats() || used != m.Budget().Used() {
		t.Fatal("symbolic path allocated numerical bookkeeping")
	}
	if _, e := m.BitVectorZeroBit(input, 12); !errors.Is(e, ErrASTDomain) {
		t.Fatal("source concat UNREACHABLE boundary", e)
	}
	zero := num(0, 8)
	got, e := m.BitVectorZeroBit(zero, ^uint32(0))
	astTestMust(t, e)
	if !got {
		t.Fatal("zero should return before enormous power construction")
	}
	if f := &m.bitVectorPlugin.numeralPrimary; !f.zeroBit || !f.retired || f.placed[1] || f.placed[2] || f.placed[3] {
		t.Fatal("zero reached power/two/modulo temporaries")
	}
}

func TestASTBitVectorZeroBitEveryActualReservation(t *testing.T) {
	success, failures, namedDivision := false, 0, false
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := bvOffsetRuntimeTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			fid, e := m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			sort, e := m.BitVectorSort(8)
			astTestMust(t, e)
			input := bvTrailingColdTestNumeral(t, m, fid, sort)
			var call ASTBitVectorNumeralInvocation
			m.Budget().FailAfter(ordinal)
			zero, body := m.BitVectorZeroBitWithInvocation(input, 65, &call)
			m.Budget().FailAfter(-1)
			if body != nil {
				failures++
				if !errors.Is(body, arena.ErrInjectedFailure) {
					var f umemory.Failure
					if !errors.As(body, &f) || f.Kind != umemory.FailureMemout {
						t.Fatal("unexpected refusal", body)
					}
				}
				if f := call.receipt.frame; f != nil {
					if f.zeroBit {
						if f.invocation == 0 || f.retired {
							t.Fatal("failed zero-bit frame disappeared")
						}
						childBody, _, _ := f.children[1].Causes()
						if !errors.Is(childBody, arena.ErrInvalidSlot) {
							if m.numerals.Manager.Z.DivisionRaw.Invocation == 0 {
								t.Fatal("entered negative division missing raw child")
							}
							namedDivision = true
						}
					}
					if f.placed[2] && !f.consumed[1] {
						t.Fatal("power return temporary survived into two construction")
					}
					if f.placed[3] && !f.placed[2] {
						t.Fatal("modulo return appeared before two")
					}
				}
				if _, e := m.BitVectorZeroBitWithInvocation(input, 65, &call); !errors.Is(e, ErrASTPending) {
					t.Fatal("stopped helper call site reused", e)
				}
			} else {
				success = true
				if !zero {
					t.Fatal("cold negative input bit 65 should be zero")
				}
			}
			ready := m.BitVectorNumeralInvocationSourceReady(&call)
			astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
			after := m.BitVectorNumeralInvocationSourceReady(&call)
			if (ready == nil) != (after == nil) || ready != nil && !errors.Is(after, ready) {
				t.Fatal("owning cleanup changed source readiness", ready, after)
			}
		})
		if success {
			break
		}
	}
	if !success || failures == 0 || !namedDivision {
		t.Fatal("cold zero-bit horizon", success, failures, namedDivision)
	}
}
