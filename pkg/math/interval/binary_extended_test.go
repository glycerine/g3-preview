package interval

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRealClosureBinaryExtendedNumeralBranches(t *testing.T) {
	budget := arena.NewBudget(1 << 28)
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	m, err := numeral.NewBinaryRationalManager(q.Manager.Z, numeral.BinaryRationalConfig{Diagnostics: q})
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	a, b, out := numeral.NewBinaryRational(), numeral.NewBinaryRational(), numeral.NewBinaryRational()
	defer func() {
		must(m.Del(&a))
		must(m.Del(&b))
		must(m.Del(&out))
		must(m.CloseChecked())
		must(q.Close())
		if budget.Used() != 0 {
			t.Fatalf("binary extended owner retained %d bytes", budget.Used())
		}
	}()
	must(m.SetInt32Power(&a, -3, 1))
	must(m.SetInt32(&b, 2))
	e := BinaryExtended{Numbers: m, DivisionPrecision: 64}
	rawOne, rawTwoOverTwo := numeral.NewBinaryInt(1), numeral.NewBinaryIntPower(2, 1)
	if equal, err := e.Equal(&rawOne, Finite, &rawTwoOverTwo, Finite); err != nil || equal {
		t.Fatalf("source binary-rational equality compares raw fields: %v, %v", equal, err)
	}
	if cmp, err := e.Compare(&rawOne, Finite, &rawTwoOverTwo, Finite); err != nil || cmp != 0 {
		t.Fatalf("numeric ordering of the same raw pair: %d, %v", cmp, err)
	}
	if equal, err := e.Equal(nil, PlusInfinity, nil, PlusInfinity); err != nil || !equal {
		t.Fatalf("matching infinities must not read numerals: %v, %v", equal, err)
	}
	for _, tc := range []struct {
		ak, bk ExtendedKind
		want   int
	}{
		{MinusInfinity, Finite, -1}, {Finite, PlusInfinity, -1}, {PlusInfinity, MinusInfinity, 1},
		{MinusInfinity, MinusInfinity, 0}, {Finite, Finite, -1},
	} {
		got, err := e.Compare(&a, tc.ak, &b, tc.bk)
		must(err)
		if got != tc.want {
			t.Fatalf("binary compare (%d,%d) = %d", tc.ak, tc.bk, got)
		}
	}
	assert := func(k ExtendedKind, err error, wantKind ExtendedKind, wantText string) {
		t.Helper()
		must(err)
		got, err := m.String(&out)
		must(err)
		if k != wantKind || got != wantText {
			t.Fatalf("binary extended result = (%d,%s), want (%d,%s)", k, got, wantKind, wantText)
		}
	}
	k, err := e.Add(&a, Finite, &b, Finite, &out)
	assert(k, err, Finite, "1/2")
	k, err = e.Sub(&a, Finite, &b, MinusInfinity, &out)
	assert(k, err, PlusInfinity, "0")
	if _, err = e.Add(&a, MinusInfinity, &b, PlusInfinity, &out); !errors.Is(err, ErrExtendedDomain) {
		t.Fatalf("opposite infinities accepted: %v", err)
	}
	k, err = e.Mul(&a, Finite, &b, PlusInfinity, &out)
	assert(k, err, MinusInfinity, "0")
	must(m.SetInt32(&a, 0))
	k, err = e.Mul(&a, Finite, nil, Finite, &out)
	assert(k, err, Finite, "0")
	must(m.SetInt32Power(&a, -3, 1))
	k, err = e.Div(&a, Finite, &b, Finite, &out)
	assert(k, err, Finite, "-3/2^2")
	k, err = e.Div(&a, MinusInfinity, &b, Finite, &out)
	assert(k, err, MinusInfinity, "0")
	must(m.SetInt32(&a, 2))
	k, err = e.Inv(&a, Finite)
	must(err)
	got, err := m.String(&a)
	must(err)
	if k != Finite || got != "1/2" {
		t.Fatalf("binary reciprocal = (%d,%s)", k, got)
	}
	k, err = e.Power(&a, Finite, 2)
	must(err)
	got, err = m.String(&a)
	must(err)
	if k != Finite || got != "1/2^2" {
		t.Fatalf("binary square = (%d,%s)", k, got)
	}
	k, err = e.Power(&a, MinusInfinity, 2)
	if err != nil || k != PlusInfinity {
		t.Fatalf("negative infinite even power = %d,%v", k, err)
	}
	k, err = e.Inv(&a, PlusInfinity)
	must(err)
	got, err = m.String(&a)
	must(err)
	if k != Finite || got != "0" {
		t.Fatalf("infinite reciprocal = (%d,%s)", k, got)
	}
	must(m.SetInt32(&a, 1))
	must(m.SetInt32(&b, 3))
	down := BinaryExtended{Numbers: m, DivisionPrecision: 12, RoundUp: false}
	up := BinaryExtended{Numbers: m, DivisionPrecision: 12, RoundUp: true}
	lo, hi := numeral.NewBinaryRational(), numeral.NewBinaryRational()
	defer func() { must(m.Del(&lo)); must(m.Del(&hi)) }()
	_, err = down.Div(&a, Finite, &b, Finite, &lo)
	must(err)
	_, err = up.Div(&a, Finite, &b, Finite, &hi)
	must(err)
	less, err := m.Less(&lo, &hi)
	must(err)
	if !less {
		t.Fatal("directed binary division did not bracket non-dyadic quotient")
	}
}
