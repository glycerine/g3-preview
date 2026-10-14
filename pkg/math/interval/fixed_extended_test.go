package interval

import (
	"errors"
	"io"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingFixedExtendedRetainedBranches(t *testing.T) {
	m, err := numeral.NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := m.New()
	b, _ := m.New()
	out, _ := m.New()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	defer func() { must(a.Close()); must(b.Close()); must(out.Close()); must(m.Close()) }()
	e := FixedExtended{Numbers: m}
	must(m.SetRatio(a, -3, 2))
	finite, err := m.String(a)
	must(err)
	assertExtendedShortWrites(t, finite, func(w io.Writer) error { return e.Display(w, a, Finite) })
	must(m.SetInt64(b, 2))
	must(m.SetInt64(out, 7))
	kind, err := e.Add(a, Finite, b, Finite, out)
	must(err)
	r, err := m.Rational(out)
	must(err)
	if kind != Finite || r.Cmp(big.NewRat(1, 2)) != 0 {
		t.Fatalf("finite extended sum = %d, %s", kind, r)
	}
	kind, err = e.Sub(a, Finite, b, Finite, out)
	must(err)
	r, err = m.Rational(out)
	must(err)
	if kind != Finite || r.Cmp(big.NewRat(-7, 2)) != 0 {
		t.Fatalf("finite extended difference = %d, %s", kind, r)
	}
	kind, err = e.Mul(a, Finite, b, Finite, out)
	must(err)
	r, err = m.Rational(out)
	must(err)
	if kind != Finite || r.Cmp(big.NewRat(-3, 1)) != 0 {
		t.Fatalf("finite extended product = %d, %s", kind, r)
	}
	cmp, err := e.Compare(a, Finite, b, Finite)
	must(err)
	if cmp >= 0 {
		t.Fatalf("-3/2 compared with 2 = %d", cmp)
	}
	cmp, err = e.Compare(a, MinusInfinity, b, PlusInfinity)
	must(err)
	if cmp >= 0 {
		t.Fatalf("infinity order = %d", cmp)
	}
	kind, err = e.Add(a, MinusInfinity, b, Finite, out)
	must(err)
	zero, err := m.IsZero(out)
	must(err)
	if kind != MinusInfinity || !zero {
		t.Fatalf("negative infinite sum = %d, zero=%t", kind, zero)
	}
	if _, err := e.Add(a, MinusInfinity, b, PlusInfinity, out); !errors.Is(err, ErrExtendedDomain) {
		t.Fatalf("opposing infinity sum = %v", err)
	}
	if _, err := e.Sub(a, PlusInfinity, b, PlusInfinity, out); !errors.Is(err, ErrExtendedDomain) {
		t.Fatalf("same infinity difference = %v", err)
	}
	must(m.Reset(out))
	kind, err = e.Mul(out, Finite, b, PlusInfinity, out)
	must(err)
	zero, err = m.IsZero(out)
	must(err)
	if kind != Finite || !zero {
		t.Fatalf("zero times infinity = %d, zero=%t", kind, zero)
	}
	kind, err = e.Mul(a, Finite, b, PlusInfinity, out)
	must(err)
	if kind != MinusInfinity {
		t.Fatalf("negative times infinity = %d", kind)
	}
	kind, err = e.Div(a, PlusInfinity, a, Finite, out)
	must(err)
	if kind != MinusInfinity {
		t.Fatalf("positive infinity / negative = %d", kind)
	}
	kind, err = e.Div(b, Finite, a, PlusInfinity, out)
	must(err)
	zero, err = m.IsZero(out)
	must(err)
	if kind != Finite || !zero {
		t.Fatalf("finite / infinity = %d, zero=%t", kind, zero)
	}
	if _, err := e.Div(a, Finite, out, Finite, out); !errors.Is(err, ErrExtendedDomain) {
		t.Fatalf("zero divisor = %v", err)
	}
	must(m.SetInt64(out, 4))
	kind, err = e.Inv(out, PlusInfinity)
	must(err)
	zero, err = m.IsZero(out)
	must(err)
	if kind != Finite || !zero {
		t.Fatalf("infinite inverse = %d, zero=%t", kind, zero)
	}
	kind, err = e.Power(a, MinusInfinity, 2)
	must(err)
	if kind != PlusInfinity {
		t.Fatalf("negative infinite square = %d", kind)
	}
	kind, err = e.Power(a, Finite, 2)
	must(err)
	r, err = m.Rational(a)
	must(err)
	if kind != Finite || r.Cmp(big.NewRat(9, 4)) != 0 {
		t.Fatalf("finite square = %d, %s", kind, r)
	}
	must(m.SetInt64(a, 1))
	must(m.SetInt64(b, 3))
	must(m.SetRounding(false))
	kind, err = e.Div(a, Finite, b, Finite, out)
	must(err)
	down, err := m.Rational(out)
	must(err)
	must(m.SetRounding(true))
	kind, err = e.Div(a, Finite, b, Finite, out)
	must(err)
	up, err := m.Rational(out)
	must(err)
	exact := big.NewRat(1, 3)
	if kind != Finite || down.Cmp(exact) >= 0 || up.Cmp(exact) <= 0 {
		t.Fatalf("directed finite quotient = %d, [%s,%s]", kind, down, up)
	}
	// Source ext_numeral.h resolves infinite kinds before reading their
	// numeral storage. Nil here represents a reset, unread endpoint.
	for _, tc := range []struct {
		kind           ExtendedKind
		zero, pos, neg bool
		negated        ExtendedKind
	}{
		{MinusInfinity, false, false, true, PlusInfinity},
		{PlusInfinity, false, true, false, MinusInfinity},
	} {
		zero, err := e.IsZero(nil, tc.kind)
		must(err)
		pos, err := e.IsPos(nil, tc.kind)
		must(err)
		neg, err := e.IsNeg(nil, tc.kind)
		must(err)
		negated, err := e.Neg(nil, tc.kind)
		must(err)
		if zero != tc.zero || pos != tc.pos || neg != tc.neg || negated != tc.negated {
			t.Fatalf("infinite kind %d: zero=%t pos=%t neg=%t negated=%d", tc.kind, zero, pos, neg, negated)
		}
		equal, err := e.Equal(nil, tc.kind, nil, tc.kind)
		must(err)
		if !equal {
			t.Fatalf("infinite kind %d unequal to itself", tc.kind)
		}
	}
	less, err := e.Less(nil, MinusInfinity, nil, PlusInfinity)
	must(err)
	greater, err := e.Greater(nil, PlusInfinity, nil, MinusInfinity)
	must(err)
	equal, err := e.Equal(nil, MinusInfinity, nil, PlusInfinity)
	must(err)
	if !less || !greater || equal {
		t.Fatal("infinite kind order and equality", less, greater, equal)
	}
	kind, err = e.Power(nil, MinusInfinity, 0)
	must(err)
	if kind != PlusInfinity {
		t.Fatal("source even-power infinity kind", kind)
	}
	kind, err = e.Power(nil, MinusInfinity, 3)
	must(err)
	if kind != MinusInfinity {
		t.Fatal("source odd-power infinity kind", kind)
	}
}
