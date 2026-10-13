package subpaving

import (
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingHardwareShellCoefficientAndInequalityConversion(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	qm, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	if err != nil {
		t.Fatal(err)
	}
	m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewHardwareContext(m, &control.Reslimit{Limit: 1000}, 20, 1, 8192)
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewHardwareShellCoefficients(c, qm)
	if err != nil {
		t.Fatal(err)
	}
	values := [4]numeral.Int{numeral.NewInt(0), numeral.NewInt(0), numeral.NewInt(0), numeral.NewInt(0)}
	q := numeral.NewRational()
	t.Cleanup(func() {
		if err := qm.Del(&q); err != nil {
			t.Error(err)
		}
		for i := range values {
			if err := qm.Z.Del(&values[i]); err != nil {
				t.Error(err)
			}
		}
		if err := c.Close(); err != nil {
			t.Error(err)
		}
		if err := qm.CloseChecked(); err != nil {
			t.Error(err)
		}
		if b.Used() != 0 {
			t.Error("shell coefficient roots", b.Used())
		}
	})
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(qm.Z.SetInt64(&values[0], 3))
	must(qm.Z.SetInt64(&values[1], 4))
	must(qm.Z.SetInt64(&values[2], 5))
	must(w.PrepareSum(&values[2], []*numeral.Int{&values[0], &values[1]}))
	if len(w.coefficients) != 2 || m.ToFloat64(&w.coefficients[0]) != 3 || m.ToFloat64(&w.coefficients[1]) != 4 || m.ToFloat64(&w.constant) != 5 {
		t.Fatal("source sum coefficient order")
	}
	must(qm.Z.SetInt64(&values[0], 1<<53))
	must(qm.Z.SetInt64(&values[1], 1<<53+1))
	if err := w.PrepareSum(&values[2], []*numeral.Int{&values[0], &values[1]}); !errors.Is(err, ErrHardwareShellInexactInteger) || !errors.Is(err, ErrSubpavingFailure) {
		t.Fatal("rounded coefficient", err)
	}
	if m.ToFloat64(&w.coefficients[0]) != 1<<53 || m.ToFloat64(&w.coefficients[1]) != 1<<53 || m.ToFloat64(&w.constant) != 5 {
		t.Fatal("inexact sum publication prefix")
	}
	must(qm.Z.SetInt64(&values[0], math.MinInt64))
	must(w.PrepareSum(&values[2], []*numeral.Int{&values[0]}))
	if m.ToFloat64(&w.coefficients[0]) != float64(math.MinInt64) {
		t.Fatal("exact minimum int64")
	}
	must(qm.Z.SetInt64(&values[0], math.MaxInt64))
	if err := w.PrepareSum(&values[2], []*numeral.Int{&values[0]}); !errors.Is(err, ErrHardwareShellInexactInteger) || !errors.Is(err, ErrSubpavingFailure) {
		t.Fatal("maximum int64 rounds outside range", err)
	}
	must(qm.Z.SetString(&values[3], "9223372036854775808"))
	if err := w.PrepareSum(&values[2], []*numeral.Int{&values[3]}); !errors.Is(err, ErrHardwareShellInexactInteger) || !errors.Is(err, ErrSubpavingFailure) {
		t.Fatal("out-of-int64 coefficient", err)
	}
	must(qm.SetFractionSigned(&q, 1, 3))
	must(w.PrepareInequality(&q, true))
	lower := new(big.Rat).SetFloat64(m.ToFloat64(&w.constant))
	if lower.Cmp(big.NewRat(1, 3)) >= 0 {
		t.Fatal("lower inequality direction", lower)
	}
	must(w.PrepareInequality(&q, false))
	upper := new(big.Rat).SetFloat64(m.ToFloat64(&w.constant))
	if upper.Cmp(big.NewRat(1, 3)) <= 0 {
		t.Fatal("upper inequality direction", upper)
	}
	x0, err := c.NewVariable(true)
	must(err)
	x1, err := c.NewVariable(true)
	must(err)
	_, err = c.NewVariable(false)
	must(err)
	must(qm.Z.SetInt64(&values[0], 3))
	must(qm.Z.SetInt64(&values[1], 4))
	must(qm.Z.SetInt64(&values[2], 5))
	must(qm.Z.SetInt64(&values[3], 7))
	polyVar, err := w.MkSum(&values[3], []*numeral.Int{&values[0], &values[1], &values[2]}, []uint32{x1, x0, x1})
	must(err)
	p := c.polynomials[polyVar]
	if polyVar != 3 || c.isInt[polyVar] || m.ToFloat64(&p.constant) != 7 || len(p.terms) != 3 ||
		p.terms[0].variable != x0 || m.ToFloat64(&p.terms[0].coefficient) != 4 ||
		p.terms[1].variable != x1 || m.ToFloat64(&p.terms[1].coefficient) != 5 ||
		p.terms[2].variable != x1 || m.ToFloat64(&p.terms[2].coefficient) != 0 ||
		len(c.watchers[x1]) != 2 || c.watchers[x1][0] != polyVar || c.watchers[x1][1] != polyVar {
		t.Fatalf("source sorted polynomial and duplicate coefficient swaps: x=%d int=%v c=%v terms=%v watchers=%v", polyVar, c.isInt[polyVar], m.ToFloat64(&p.constant), p.terms, c.watchers[x1])
	}
	if _, err := w.MkSum(&values[3], []*numeral.Int{&values[0]}, []uint32{x0, x1}); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("mismatched sum vectors", err)
	}
	if _, err := w.MkSum(&values[3], []*numeral.Int{&values[0]}, []uint32{99}); !errors.Is(err, arena.ErrInvalidSlot) || len(c.isInt) != 4 {
		t.Fatal("invalid source variable", err)
	}
	lowerAtom, err := w.MkInequality(x0, &q, true, false)
	must(err)
	upperAtom, err := w.MkInequality(x0, &q, false, true)
	must(err)
	if !lowerAtom.lower || lowerAtom.open || upperAtom.lower || !upperAtom.open ||
		new(big.Rat).SetFloat64(m.ToFloat64(&lowerAtom.value)).Cmp(big.NewRat(1, 3)) >= 0 ||
		new(big.Rat).SetFloat64(m.ToFloat64(&upperAtom.value)).Cmp(big.NewRat(1, 3)) <= 0 ||
		m.ToFloat64(&p.constant) != 7 {
		t.Fatal("directed inequality copies and polynomial ownership")
	}
	if _, err := w.MkInequality(99, &q, true, false); !errors.Is(err, arena.ErrInvalidSlot) || len(c.inequalities) != 2 {
		t.Fatal("invalid inequality variable", err)
	}
	must(qm.Z.Power(&numeral.Int{Val: 2}, 2000, &values[0]))
	must(qm.SetInteger(&q, &values[0]))
	if err := w.PrepareInequality(&q, false); !errors.Is(err, numeral.ErrHardwareFloatNonRegular) || errors.Is(err, ErrSubpavingFailure) {
		t.Fatal("source hardware wrapper preserves uncaught hardware numeral exception", err)
	}
	root, err := c.CreateRootNode()
	must(err)
	if root.id != 0 || root.depth != 0 || len(root.lower) != len(c.isInt) || root.context != c {
		t.Fatal("context root bound views")
	}
	if _, err := c.NewVariable(true); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("variable after root construction", err)
	}
	child, err := c.NewChildNode(root)
	must(err)
	if child.id != 1 || child.depth != 1 || child.context != c {
		t.Fatal("context child node identity")
	}
	must(c.Close())
	if !child.closed || !root.closed || c.activeNodes != 0 {
		t.Fatal("context must release children before root")
	}
	if err := w.PrepareInequality(&q, true); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed wrapper context", err)
	}
	if _, err := c.NewVariable(true); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed context variable", err)
	}
}
