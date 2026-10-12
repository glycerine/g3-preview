package lp

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func lpSparseRationalCheck(t *testing.T, h *lpPairHarness, value *numeral.Rational, want *big.Rat) {
	t.Helper()
	s, err := h.m.String(value)
	h.must(err)
	got, ok := new(big.Rat).SetString(s)
	if !ok || got.Cmp(want) != 0 {
		t.Fatalf("got %s want %s", s, want)
	}
}

func TestLPSparseRationalVectorIndependentOracle(t *testing.T) {
	h := newLPPairHarness(t)
	v := NewSparseRationalVector(h.m)
	t.Cleanup(func() { h.must(v.Close()) })
	p := h.pair("0", "0")
	r := rand.New(rand.NewPCG(715, 100))
	var expected []*big.Rat
	var indices []uint32
	for step := 0; step < 120; step++ {
		n := new(big.Int).Lsh(big.NewInt(int64(r.IntN(100)-50)), uint(r.IntN(100)))
		x := new(big.Rat).SetFrac(n, big.NewInt(int64(1+r.IntN(99))))
		h.must(h.m.SetString(&p.X, x.RatString()))
		index := uint32(r.IntN(9))
		h.must(v.Append(index, &p.X))
		expected = append(expected, x)
		indices = append(indices, index)
		if step%11 == 0 {
			div := new(big.Rat).SetFrac(big.NewInt(int64(1+r.IntN(30))), big.NewInt(int64(1+r.IntN(30))))
			h.must(h.m.SetString(&p.Y, div.RatString()))
			h.must(v.Divide(&p.Y))
			for _, x := range expected {
				x.Quo(x, div)
			}
		}
		if int(v.Size()) != len(expected) {
			t.Fatal(v.Size())
		}
		for i := range expected {
			if v.Data[i].Index != indices[i] {
				t.Fatal("reordered entries")
			}
			lpSparseRationalCheck(t, h, &v.Data[i].Value, expected[i])
		}
	}
	for index := uint32(0); index < 10; index++ {
		h.must(v.At(index, &p.X))
		want := new(big.Rat)
		for i, j := range indices {
			if j == index {
				want = expected[i]
				break
			}
		}
		lpSparseRationalCheck(t, h, &p.X, want)
	}
	h.must(v.Close())
	h.must(v.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPSparseRationalVectorLiteralAliasesAndZero(t *testing.T) {
	h := newLPPairHarness(t)
	v := NewSparseRationalVector(h.m)
	t.Cleanup(func() { h.must(v.Close()) })
	p := h.pair("2", "6")
	h.must(v.Append(99, &p.X))
	h.must(v.Append(99, &p.Y))
	h.must(v.Append(5, &v.Data[0].Value))
	h.must(v.Divide(&v.Data[0].Value))
	// The const-reference divisor becomes one after the first entry. Later
	// entries therefore divide by one rather than by its original value two.
	for i, w := range []int64{1, 6, 2} {
		lpSparseRationalCheck(t, h, &v.Data[i].Value, big.NewRat(w, 1))
	}
	h.must(v.At(99, &p.X))
	lpSparseRationalCheck(t, h, &p.X, big.NewRat(1, 1))
	h.must(v.At(100, &p.X))
	lpSparseRationalCheck(t, h, &p.X, new(big.Rat))
	h.must(v.Append(42, &p.X))
	// A tiny exact divisor must never trigger the floating epsilon assertion.
	h.must(h.m.SetString(&p.Y, "1/1000000000000000000000000000000000000000000000000"))
	h.must(v.Divide(&p.Y))
	err := v.Divide(&p.X)
	if sourceDebugTests {
		if err != ErrAssertion {
			t.Fatal(err)
		}
	} else if !errors.Is(err, numeral.ErrDivideByZero) {
		t.Fatal(err)
	}
	h.must(v.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPSparseRationalVectorAllocationFailureCensus(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"append", "lookup", "divide", "negative-divide", "alias-divide"} {
			failures, successes, rawStops := 0, 0, 0
			for ordinal := int64(0); ordinal < 150; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("direct=%v/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
					if op == "negative-divide" {
						h.must(h.m.Neg(&p.X))
					}
					out := h.pair("0", "0")
					v := NewSparseRationalVector(h.m)
					t.Cleanup(func() { h.budget.FailAfter(-1); h.must(v.Close()) })
					if op != "append" {
						h.must(v.Append(1, &p.X))
						h.must(v.Append(3, &p.Y))
					}
					h.budget.FailAfter(ordinal)
					var err error
					switch op {
					case "append":
						err = v.Append(7, &p.X)
					case "lookup":
						err = v.At(1, &out.X)
					case "divide", "negative-divide":
						err = v.Divide(&p.Y)
					case "alias-divide":
						err = v.Divide(&v.Data[0].Value)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						successes++
						done = true
					} else {
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						failures++
						if op == "append" && v.Size() != 0 {
							t.Fatal("published incomplete coefficient")
						}
					}
					if v.calls.division != 0 {
						rawStops++
					}
					if v.ready() != nil {
						if e := v.Append(9, &p.X); e != ErrExactVectorPending {
							t.Fatal(e)
						}
						if e := v.At(1, &out.X); e != ErrExactVectorPending {
							t.Fatal(e)
						}
						if e := v.Divide(&p.Y); e != ErrExactVectorPending {
							t.Fatal(e)
						}
					}
					h.must(v.Close())
					h.must(v.Close())
					lpExactVectorNamedCensus(t, h, nil)
					if h.m.DivisionInvocationState().Active || h.m.RawInvocationState().Active != [3]bool{} {
						t.Fatal("orphan numeral frame")
					}
				})
				if done {
					break
				}
			}
			if failures == 0 || successes != 1 {
				t.Fatal(op, failures, successes)
			}
			if op == "alias-divide" && rawStops == 0 {
				t.Fatal("no retained alias temporary exercised")
			}
		}
	}
}

func TestLPSparseRationalVectorLeavesOlderDivisionOwned(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
			z := h.pair("0", "1")
			var caller exactEtaCalls
			before, err := caller.begin(h.m)
			h.must(err)
			h.budget.FailAfter(ordinal)
			err = h.m.Div(&p.X, &p.Y, &p.Y)
			h.budget.FailAfter(-1)
			err = caller.after(h.m, before, err)
			state := h.m.DivisionInvocationState()
			if state.Active {
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				matched = true
				v := NewSparseRationalVector(h.m)
				h.must(v.Append(0, &z.X))
				h.must(v.Divide(&z.Y))
				h.must(v.At(0, &z.X))
				h.must(v.Close())
				if h.m.DivisionInvocationState() != state {
					t.Fatal("borrower consumed older division")
				}
				// The owner's own stopped call refuses further mutations.
				if _, e := caller.begin(h.m); e != ErrExactVectorPending {
					t.Fatal(e)
				}
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no older division reached")
	}
}

func TestLPExactEtaCallsRetainedIntegerDivision(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair("-"+strings.Repeat("1234567890", 100)+"7", strings.Repeat("9876543210", 40)+"3")
			out := h.pair("0", "0")
			var caller exactEtaCalls
			before, err := caller.begin(h.m)
			h.must(err)
			h.budget.FailAfter(ordinal)
			err = h.m.Z.Div(&p.X.Num, &p.Y.Num, &out.X.Num)
			h.budget.FailAfter(-1)
			err = caller.after(h.m, before, err)
			if caller.integerDivision.Matches(h.m.Z) {
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				matched = true
				// A second borrower observes the older invocation without
				// acquiring it; cleanup must leave the original receipt live.
				var borrower exactEtaCalls
				b, e := borrower.begin(h.m)
				h.must(e)
				h.must(borrower.after(h.m, b, nil))
				h.must(borrower.cleanupRaw(h.m))
				h.must(borrower.cleanupRoots(h.m))
				if !caller.integerDivision.Matches(h.m.Z) {
					t.Fatal("older integer division consumed")
				}
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no actual retained integer division reached")
	}
}
