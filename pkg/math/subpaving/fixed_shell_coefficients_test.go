// Source: src/math/subpaving/subpaving.cpp:198-260.
package subpaving

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestFixedShellExactCoefficientOwnerAndDirectedBound(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	q, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	m, err := numeral.NewFixedPointManager(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewFixedContext(m, &control.Reslimit{Limit: 1000}, nil, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewFixedShellCoefficients(c, q)
	if err != nil {
		t.Fatal(err)
	}
	one, two, three := numeral.NewInt(1), numeral.NewInt(2), numeral.NewInt(3)
	tooBig := numeral.Int{}
	bound := numeral.NewRational()
	t.Cleanup(func() {
		for _, e := range []error{w.Close(), c.Close(), q.Del(&bound), q.Z.Del(&tooBig), m.Close(), q.CloseChecked()} {
			if e != nil {
				t.Error(e)
			}
		}
		if budget.Used() != 0 {
			t.Error("fixed shell retained allocation", budget.Used())
		}
	})
	x, err := w.MkVar(true)
	if err != nil || x != 0 {
		t.Fatal("first source variable", x, err)
	}
	y, err := w.MkVar(false)
	if err != nil || y != 1 || len(c.root.lower) != 2 || len(c.watchers) != 2 {
		t.Fatal("second source variable and bound views", y, err)
	}
	constant, values, err := w.PrepareSum(&three, []*numeral.Int{&one, &two})
	if err != nil || len(values) != 2 {
		t.Fatal("source coefficient conversion", err, len(values))
	}
	for i, tc := range []struct {
		value *numeral.FixedPoint
		want  int64
	}{{constant, 3}, {values[0], 1}, {values[1], 2}} {
		got, e := m.Rational(tc.value)
		if e != nil || got.Cmp(bigRat(tc.want)) != 0 {
			t.Fatalf("converted coefficient %d: %v, %v", i, got, e)
		}
	}
	_, _, err = w.PrepareSum(&three, []*numeral.Int{&one})
	if err != nil || len(w.coefficients) != 2 {
		t.Fatal("source reserve high slot", len(w.coefficients), err)
	}
	high, err := m.Rational(w.coefficients[1])
	if err != nil || high.Cmp(bigRat(2)) != 0 {
		t.Fatal("smaller sum changed retained high coefficient", high, err)
	}
	z, err := w.MkSum(&three, []*numeral.Int{&one, &two}, []uint32{y, x})
	if err != nil || z != 2 || len(c.polynomials[z].terms) != 2 || c.polynomials[z].terms[0].Variable != x {
		t.Fatal("fixed polynomial publication", z, err)
	}
	monomial, err := w.MkMonomial([]FixedPower{{Variable: x, Degree: 2}, {Variable: y, Degree: 1}})
	if err != nil || monomial != 3 || c.monomials[monomial] == nil {
		t.Fatal("fixed wrapper monomial forwarding", monomial, err)
	}
	if err := q.Z.SetString(&tooBig, "4294967296"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := w.PrepareSum(&three, []*numeral.Int{&tooBig}); !errors.Is(err, ErrSubpavingFailure) {
		t.Fatal("fixed-point overflow exception boundary", err)
	}
	if err := q.SetFractionSigned(&bound, 1, 3); err != nil {
		t.Fatal(err)
	}
	lower, err := w.MkInequality(x, &bound, true, false)
	if err != nil {
		t.Fatal(err)
	}
	upper, err := w.MkInequality(x, &bound, false, true)
	if err != nil {
		t.Fatal(err)
	}
	lo, err := m.Rational(lower.value)
	if err != nil || lo.Cmp(big.NewRat(1, 3)) >= 0 {
		t.Fatal("lower directed fixed bound", lo, err)
	}
	hi, err := m.Rational(upper.value)
	if err != nil || hi.Cmp(big.NewRat(1, 3)) <= 0 || !upper.open {
		t.Fatal("upper directed fixed bound", hi, err)
	}
	clause, err := w.AddClause([]*FixedInequality{upper, lower})
	if err != nil || clause == nil || len(clause.atoms) != 2 || !clause.watched || clause.lemma {
		t.Fatal("fixed wrapper clause forwarding", clause, err)
	}
}

func bigRat(n int64) *big.Rat { return big.NewRat(n, 1) }
