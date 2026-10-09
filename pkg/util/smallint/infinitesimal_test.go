// Source: src/util/inf_s_integer.h:25-347; src/util/inf_s_integer.cpp:22-24.
package smallint

import (
	"errors"
	"testing"
)

func TestInfinitesimalConstructionPredicatesAndText(t *testing.T) {
	for _, tc := range []struct {
		whole, epsilon int32
		text           string
	}{{3, 0, "3"}, {3, 2, "(3 +e*2)"}, {3, -2, "(3 -e*-2)"}, {-3, -2, "(-3 -e*-2)"}} {
		n := NewPair(New(tc.whole), New(tc.epsilon))
		if n.Whole() != New(tc.whole) || n.Epsilon() != New(tc.epsilon) || n.String() != tc.text {
			t.Fatal(tc, n, n.String())
		}
		if n.IsInteger() != (tc.epsilon == 0) || n.IsZero() != (tc.whole == 0 && tc.epsilon == 0) || n.IsPositive() != (tc.whole > 0 || tc.whole == 0 && tc.epsilon > 0) || n.IsNegative() != (tc.whole < 0 || tc.whole == 0 && tc.epsilon < 0) {
			t.Fatal("predicate", tc)
		}
	}
	if NewOffset(7, true) != NewPair(7, 1) || NewOffset(7, false) != NewPair(7, -1) {
		t.Fatal("offset")
	}
}

func TestInfinitesimalLexicographicAlgebra(t *testing.T) {
	values := []Infinitesimal{
		NewPair(-2, -1), NewPair(-2, 0), NewPair(-2, 1), NewPair(0, -1), {}, NewPair(0, 1), NewPair(2, -1), NewPair(2, 0), NewPair(2, 1),
	}
	for i, x := range values {
		for j, y := range values {
			want := comparePair(x.whole, x.epsilon, y.whole, y.epsilon)
			if x.Compare(y) != want {
				t.Fatal("order", i, j)
			}
			if x.Add(y) != NewPair(x.whole+y.whole, x.epsilon+y.epsilon) || x.Sub(y) != NewPair(x.whole-y.whole, x.epsilon-y.epsilon) {
				t.Fatal("pair arithmetic", i, j)
			}
		}
		for _, scalar := range []Integer{-3, 0, 4} {
			if x.CompareInteger(scalar) != comparePair(x.whole, x.epsilon, scalar, 0) {
				t.Fatal("scalar compare", i, scalar)
			}
			if x.AddInteger(scalar) != NewPair(x.whole+scalar, x.epsilon) || x.SubInteger(scalar) != NewPair(x.whole-scalar, x.epsilon) {
				t.Fatal("scalar arithmetic", i, scalar)
			}
			if scalar != 0 && x.Scale(scalar).Quo(scalar) != x {
				t.Fatal("scale quotient", i, scalar)
			}
		}
	}
}

func TestInfinitesimalFloorCeilFusedAndGuards(t *testing.T) {
	for _, tc := range []struct {
		n           Infinitesimal
		floor, ceil Integer
	}{{NewPair(3, -2), 2, 3}, {NewPair(3, 0), 3, 3}, {NewPair(3, 2), 3, 4}} {
		if tc.n.Floor() != tc.floor || tc.n.Ceil() != tc.ceil {
			t.Fatal(tc)
		}
		if tc.n.Abs().IsNegative() || tc.n.Neg().Neg() != tc.n {
			t.Fatal("sign", tc)
		}
		if tc.n.AddMul(2, tc.n) != tc.n.Scale(3) || tc.n.SubMul(2, tc.n) != tc.n.Neg() {
			t.Fatal("fused", tc)
		}
	}
	if got, err := NewFraction(4, 2); sourceDebug {
		if got != (Infinitesimal{}) || !errors.Is(err, ErrAssertion) {
			t.Fatal(got, err)
		}
	} else if err != nil || got != NewInfinitesimal(4) {
		t.Fatal(got, err)
	}
	n := NewPair(4, 1)
	_, numeratorErr := n.Numerator()
	_, denominatorErr := n.Denominator()
	if sourceDebug != (errors.Is(numeratorErr, ErrAssertion) && errors.Is(denominatorErr, ErrAssertion)) {
		t.Fatal(numeratorErr, denominatorErr)
	}
}

func TestInfinitesimalWarmValueOperationsDoNotAllocate(t *testing.T) {
	a, b := NewPair(17, 1), NewPair(5, -1)
	if got := testing.AllocsPerRun(1000, func() {
		a = a.Add(b).Sub(b).AddMul(2, b).SubMul(2, b)
		if a.Compare(b) == 0 {
			t.Fatal("unexpected equality")
		}
	}); got != 0 {
		t.Fatal("allocations", got)
	}
}

func comparePair(aw, ae, bw, be Integer) int {
	if aw != bw {
		return aw.Compare(bw)
	}
	return ae.Compare(be)
}
