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
	if err := w.PrepareSum(&values[2], []*numeral.Int{&values[0], &values[1]}); !errors.Is(err, ErrHardwareShellInexactInteger) {
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
	if err := w.PrepareSum(&values[2], []*numeral.Int{&values[0]}); !errors.Is(err, ErrHardwareShellInexactInteger) {
		t.Fatal("maximum int64 rounds outside range", err)
	}
	must(qm.Z.SetString(&values[3], "9223372036854775808"))
	if err := w.PrepareSum(&values[2], []*numeral.Int{&values[3]}); !errors.Is(err, ErrHardwareShellInexactInteger) {
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
	must(c.Close())
	if err := w.PrepareInequality(&q, true); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed wrapper context", err)
	}
}
