package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"
)

func TestLPImpliedBoundKindsAndStrings(t *testing.T) {
	for _, tc := range []struct {
		low, strict bool
		kind        LConstraintKind
		text        string
	}{
		{false, false, LConstraintLE, "<="},
		{false, true, LConstraintLT, "<"},
		{true, false, LConstraintGE, ">="},
		{true, true, LConstraintGT, ">"},
	} {
		b := ImpliedBound{IsLowBound: tc.low, Strict: tc.strict}
		if b.Kind() != tc.kind {
			t.Fatal(b.Kind(), tc.kind)
		}
		text, err := LConstraintKindString(b.Kind())
		if err != nil || text != tc.text {
			t.Fatal(text, err)
		}
		b.IsLowBound = !b.IsLowBound
		if FlipConstraintKind(tc.kind) != b.Kind() {
			t.Fatal("reversed relation")
		}
	}
	text, err := LConstraintKindString(LConstraintEQ)
	if text != "=" || err != nil || FlipConstraintKind(LConstraintEQ) != LConstraintEQ {
		t.Fatal(text, err)
	}
	for _, kind := range []LConstraintKind{-3, 3, -2147483648, 2147483647} {
		text, err = LConstraintKindString(kind)
		if text != "" || err != sourceAssertionFailure() {
			t.Fatal(kind, text, err)
		}
		if FlipConstraintKind(FlipConstraintKind(kind)) != kind {
			t.Fatal("int32 wrap")
		}
	}
}

func TestLPImpliedBoundIndependentExactAndEvidence(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	h := newLPPairHarness(t)
	p := h.pair("0", "0")
	rng := rand.New(rand.NewPCG(5191, 221))
	for i := 0; i < 160; i++ {
		x := new(big.Rat).SetFrac(new(big.Int).Lsh(big.NewInt(int64(rng.IntN(2001)-1000)), uint(rng.IntN(160))), big.NewInt(int64(rng.IntN(101)+1)))
		h.must(h.m.SetString(&p.X, x.RatString()))
		column, row := rng.Uint32(), rng.Uint32()
		low, sign, strict := rng.IntN(2) == 0, rng.IntN(2) == 0, rng.IntN(2) == 0
		b, err := NewImpliedBound(h.m, &p.X, column, low, sign, row, strict)
		h.must(err)
		other, err := NewImpliedBound(h.m, &p.X, column, low, sign, row, strict)
		h.must(err)
		h.must(h.m.SetInt32(&p.X, 19))
		lpSparseRationalCheck(t, h, &b.Bound, x)
		for field := 0; field < 7; field++ {
			switch field {
			case 1:
				other.Column++
			case 2:
				other.IsLowBound = !low
			case 3:
				h.must(h.m.SetString(&other.Bound, new(big.Rat).Add(x, big.NewRat(1, 1)).RatString()))
			case 4:
				other.CoefficientIsPositive = !sign
			case 5:
				other.RowOrTermIndex++
			case 6:
				other.Strict = !strict
			}
			equal, err := b.Equal(other)
			h.must(err)
			if equal != (field == 0) {
				t.Fatal(i, field, equal)
			}
			if field != 0 {
				h.must(other.Assign(b))
				equal, err = b.Equal(other)
				h.must(err)
				if !equal {
					t.Fatal("assignment did not restore evidence", field)
				}
			}
			other.Column, other.IsLowBound, other.CoefficientIsPositive, other.RowOrTermIndex, other.Strict = column, low, sign, row, strict
			h.must(h.m.SetString(&other.Bound, x.RatString()))
		}
		h.must(b.Assign(b))
		h.must(other.Close())
		lpSparseRationalCheck(t, h, &b.Bound, x)
		h.must(b.Close())
		h.must(b.Close())
	}
	foreign := newLPPairHarness(t)
	q := foreign.pair("1", "0")
	b, err := NewImpliedBound(h.m, &p.X, 0, false, false, 0, false)
	h.must(err)
	c, err := NewImpliedBound(foreign.m, &q.X, 0, false, false, 0, false)
	h.must(err)
	if _, err = b.Equal(c); err != ErrExactVectorOwner {
		t.Fatal(err)
	}
	if err = b.Assign(c); err != ErrExactVectorOwner {
		t.Fatal(err)
	}
	h.must(c.Close())
	h.must(b.Close())
}

func TestLPImpliedBoundConstructionPhysicalRefusals(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	value := strings.Repeat("1234567890", 100) + "/" + strings.Repeat("9876543210", 99) + "1"
	failures := 0
	for ordinal := int64(0); ordinal < 150; ordinal++ {
		done := false
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair(value, "0")
			h.budget.FailAfter(ordinal)
			b, err := NewImpliedBound(h.m, &p.X, 71, true, true, 91, true)
			h.budget.FailAfter(-1)
			if b == nil {
				t.Fatal("missing partial owner")
			}
			if err != nil {
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				failures++
				if b.Column != 0 || b.IsLowBound || b.CoefficientIsPositive || b.RowOrTermIndex != 0 || b.Strict {
					t.Fatal("evidence published before copy")
				}
				if _, err = b.Equal(b); err != ErrExactVectorPending {
					t.Fatal("stopped copy", err)
				}
				if err = b.Assign(b); err != ErrExactVectorPending {
					t.Fatal(err)
				}
			} else {
				done = true
				want, _ := new(big.Rat).SetString(value)
				lpSparseRationalCheck(t, h, &b.Bound, want)
				if b.Column != 71 || !b.IsLowBound || !b.CoefficientIsPositive || b.RowOrTermIndex != 91 || !b.Strict {
					t.Fatal("evidence")
				}
			}
			h.must(b.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
		if done {
			if failures == 0 {
				t.Fatal("no physical refusal exercised")
			}
			return
		}
	}
	t.Fatal("construction never succeeded")
}

func TestLPImpliedBoundAssignmentPhysicalRefusals(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	value := strings.Repeat("2345678901", 100) + "/" + strings.Repeat("7654321098", 99) + "1"
	failures := 0
	for ordinal := int64(0); ordinal < 150; ordinal++ {
		done := false
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair(value, "7/11")
			source, err := NewImpliedBound(h.m, &p.X, 73, true, true, 99, true)
			h.must(err)
			dest, err := NewImpliedBound(h.m, &p.Y, 18, false, false, 12, false)
			h.must(err)
			h.budget.FailAfter(ordinal)
			err = dest.Assign(source)
			h.budget.FailAfter(-1)
			if err != nil {
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				failures++
				if dest.Column != 18 || dest.IsLowBound || dest.CoefficientIsPositive || dest.RowOrTermIndex != 12 || dest.Strict {
					t.Fatal("partial evidence assignment")
				}
				if _, err = source.Equal(dest); err != ErrExactVectorPending {
					t.Fatal(err)
				}
				if err = source.Assign(dest); err != ErrExactVectorPending {
					t.Fatal(err)
				}
				if err = dest.Assign(source); err != ErrExactVectorPending {
					t.Fatal(err)
				}
			} else {
				done = true
				equal, err := dest.Equal(source)
				h.must(err)
				if !equal {
					t.Fatal("copy differs")
				}
			}
			h.must(dest.Close())
			want, _ := new(big.Rat).SetString(value)
			lpSparseRationalCheck(t, h, &source.Bound, want)
			h.must(source.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
		if done {
			if failures == 0 {
				t.Fatal("no physical refusal exercised")
			}
			return
		}
	}
	t.Fatal("assignment never succeeded")
}
