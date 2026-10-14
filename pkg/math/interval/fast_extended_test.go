package interval

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingFastFloatExtendedRetainedBranches(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	cfg := numeral.DefaultFastFloatConfig(arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	cfg.InitialCapacity = 4
	m, err := numeral.NewFastFloatManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	vars := make([]*numeral.ScopedFastFloat, 6)
	for i := range vars {
		vars[i], err = numeral.NewScopedFastFloat(m)
		if err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		for i := len(vars) - 1; i >= 0; i-- {
			if e := vars[i].Close(); e != nil {
				t.Error(e)
			}
		}
		if e := m.CloseChecked(); e != nil {
			t.Error(e)
		}
		if b.Used() != 0 {
			t.Error("retained fast-float roots", b.Used())
		}
	}()
	a, _ := vars[0].Get()
	d, _ := vars[1].Get()
	out, _ := vars[2].Get()
	expected, _ := vars[3].Get()
	down, _ := vars[4].Get()
	up, _ := vars[5].Get()
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	eq := func(x, y *numeral.FastFloat) bool { t.Helper(); v, e := m.Equal(x, y); must(e); return v }
	e := FastExtended{Numbers: m}
	must(m.SetInt32(a, -3))
	must(m.SetInt32(d, 2))
	for _, tc := range []struct {
		kind ExtendedKind
		want string
	}{
		{MinusInfinity, "-oo"},
		{PlusInfinity, "+oo"},
	} {
		var rendered bytes.Buffer
		must(e.Display(&rendered, nil, tc.kind))
		if rendered.String() != tc.want {
			t.Fatalf("infinite display: got %q, want %q", rendered.String(), tc.want)
		}
	}
	var rendered bytes.Buffer
	must(e.Display(&rendered, a, Finite))
	finite, err := m.String(a)
	must(err)
	if rendered.String() != finite {
		t.Fatalf("finite display: got %q, want %q", rendered.String(), finite)
	}
	assertExtendedShortWrites(t, finite, func(w io.Writer) error { return e.Display(w, a, Finite) })
	positive, err := e.IsPos(d, Finite)
	must(err)
	negative, err := e.IsNeg(a, Finite)
	must(err)
	if !positive || !negative {
		t.Fatal("finite signs", positive, negative)
	}
	k, err := e.Neg(a, Finite)
	must(err)
	must(m.SetInt32(expected, 3))
	if k != Finite || !eq(a, expected) {
		t.Fatal("finite negation", k)
	}
	must(m.SetInt32(a, -3))
	k, err = e.Add(a, Finite, d, Finite, out)
	must(err)
	must(m.SetInt32(expected, -1))
	if k != Finite || !eq(out, expected) {
		t.Fatal("finite sum", k)
	}
	k, err = e.Sub(a, Finite, d, Finite, out)
	must(err)
	must(m.SetInt32(expected, -5))
	if k != Finite || !eq(out, expected) {
		t.Fatal("finite difference", k)
	}
	k, err = e.Mul(a, Finite, d, Finite, out)
	must(err)
	must(m.SetInt32(expected, -6))
	if k != Finite || !eq(out, expected) {
		t.Fatal("finite product", k)
	}
	k, err = e.Div(a, Finite, d, Finite, out)
	must(err)
	must(m.SetFraction32(expected, -3, 2))
	if k != Finite || !eq(out, expected) {
		t.Fatal("finite quotient", k)
	}
	cmp, err := e.Compare(a, Finite, d, Finite)
	must(err)
	if cmp >= 0 {
		t.Fatal("finite order", cmp)
	}
	cmp, err = e.Compare(a, MinusInfinity, d, PlusInfinity)
	must(err)
	if cmp >= 0 {
		t.Fatal("infinity order", cmp)
	}
	k, err = e.Add(a, MinusInfinity, d, Finite, out)
	must(err)
	zero, err := m.IsZero(out)
	must(err)
	if k != MinusInfinity || !zero {
		t.Fatal("infinite sum did not reset output", k, zero)
	}
	if _, err = e.Add(a, MinusInfinity, d, PlusInfinity, out); !errors.Is(err, ErrExtendedDomain) {
		t.Fatal("opposite infinities", err)
	}
	if _, err = e.Sub(a, PlusInfinity, d, PlusInfinity, out); !errors.Is(err, ErrExtendedDomain) {
		t.Fatal("same infinity subtraction", err)
	}
	must(m.Reset(out))
	k, err = e.Mul(out, Finite, d, PlusInfinity, out)
	must(err)
	zero, err = m.IsZero(out)
	must(err)
	if k != Finite || !zero {
		t.Fatal("zero times infinity", k, zero)
	}
	k, err = e.Mul(a, Finite, d, PlusInfinity, out)
	must(err)
	if k != MinusInfinity {
		t.Fatal("negative times infinity", k)
	}
	k, err = e.Div(a, PlusInfinity, a, Finite, out)
	must(err)
	if k != MinusInfinity {
		t.Fatal("signed infinite quotient", k)
	}
	k, err = e.Div(d, Finite, a, PlusInfinity, out)
	must(err)
	zero, err = m.IsZero(out)
	must(err)
	if k != Finite || !zero {
		t.Fatal("finite over infinity", k, zero)
	}
	if _, err = e.Div(a, Finite, out, Finite, out); !errors.Is(err, ErrExtendedDomain) {
		t.Fatal("zero divisor", err)
	}
	must(m.SetInt32(out, 4))
	k, err = e.Inv(out, PlusInfinity)
	must(err)
	zero, err = m.IsZero(out)
	must(err)
	if k != Finite || !zero {
		t.Fatal("infinite inverse", k, zero)
	}
	k, err = e.Power(a, MinusInfinity, 2)
	must(err)
	if k != PlusInfinity {
		t.Fatal("negative infinite square", k)
	}
	k, err = e.Power(a, Finite, 2)
	must(err)
	must(m.SetInt32(expected, 9))
	if k != Finite || !eq(a, expected) {
		t.Fatal("finite square", k)
	}
	must(m.SetInt32(a, 1))
	must(m.SetInt32(d, 3))
	must(m.SetRounding(false))
	k, err = e.Div(a, Finite, d, Finite, down)
	must(err)
	must(m.SetRounding(true))
	k, err = e.Div(a, Finite, d, Finite, up)
	must(err)
	less, err := m.Less(down, up)
	must(err)
	if k != Finite || !less {
		t.Fatal("directed fast-float quotient", k)
	}
	// Source ext_numeral.h resolves infinite kinds without consulting the
	// reset numeral. A nil numeral makes an accidental read visible here.
	for _, tc := range []struct {
		kind               ExtendedKind
		positive, negative bool
		negated            ExtendedKind
	}{
		{MinusInfinity, false, true, PlusInfinity},
		{PlusInfinity, true, false, MinusInfinity},
	} {
		zero, err := e.IsZero(nil, tc.kind)
		must(err)
		positive, err := e.IsPos(nil, tc.kind)
		must(err)
		negative, err := e.IsNeg(nil, tc.kind)
		must(err)
		negated, err := e.Neg(nil, tc.kind)
		must(err)
		equal, err := e.Equal(nil, tc.kind, nil, tc.kind)
		must(err)
		if zero || positive != tc.positive || negative != tc.negative || negated != tc.negated || !equal {
			t.Fatalf("infinite kind %d: zero=%t pos=%t neg=%t negated=%d equal=%t", tc.kind, zero, positive, negative, negated, equal)
		}
	}
	less, err = e.Less(nil, MinusInfinity, nil, PlusInfinity)
	must(err)
	greater, err := e.Greater(nil, PlusInfinity, nil, MinusInfinity)
	must(err)
	equal, err := e.Equal(nil, MinusInfinity, nil, PlusInfinity)
	must(err)
	if !less || !greater || equal {
		t.Fatal("infinite kind order and equality", less, greater, equal)
	}
	k, err = e.Power(nil, MinusInfinity, 0)
	must(err)
	if k != PlusInfinity {
		t.Fatal("source even-power infinity kind", k)
	}
	k, err = e.Power(nil, MinusInfinity, 3)
	must(err)
	if k != MinusInfinity {
		t.Fatal("source odd-power infinity kind", k)
	}
}
