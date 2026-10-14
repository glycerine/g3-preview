package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRationalShellPersistentCoefficientGrowthAndContextTransfer(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	m, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewRationalContext(m, &control.Reslimit{Limit: 1000})
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewRationalShellCoefficients(c)
	if err != nil {
		t.Fatal(err)
	}
	var constant, coefficient numeral.Int
	t.Cleanup(func() {
		for _, e := range []error{w.Close(), c.Close(), m.Z.Del(&coefficient), m.Z.Del(&constant), m.CloseChecked()} {
			if e != nil {
				t.Error(e)
			}
		}
		if budget.Used() != 0 {
			t.Error("rational shell retained numeral owners", budget.Used())
		}
	})
	if err := m.Z.SetInt32(&constant, 3); err != nil {
		t.Fatal(err)
	}
	if err := m.Z.SetString(&coefficient, "18446744073709551617"); err != nil {
		t.Fatal(err)
	}
	variables := make([]uint32, 20)
	coefficients := make([]*numeral.Int, len(variables))
	for i := range variables {
		variables[i], err = c.NewVariable(false)
		if err != nil {
			t.Fatal(err)
		}
		coefficients[i] = &coefficient
	}
	first, err := w.MkSum(&constant, coefficients, variables)
	if err != nil || len(w.coefficients) != 20 || len(c.polynomials[first].terms) != 20 {
		t.Fatal("source wrapper reserve and first definition", first, err)
	}
	if err := m.Z.SetInt32(&coefficient, 5); err != nil {
		t.Fatal(err)
	}
	second, err := w.MkSum(&constant, coefficients[:1], variables[:1])
	if err != nil || len(w.coefficients) != 20 || len(c.polynomials[second].terms) != 1 {
		t.Fatal("source wrapper reused coefficient slots", second, err)
	}
	for _, sample := range []struct {
		value *numeral.Rational
		want  string
	}{
		{&c.polynomials[first].terms[0].coefficient, "18446744073709551617"},
		{&c.polynomials[second].terms[0].coefficient, "5"},
		{&w.coefficients[0], "5"},
		{&w.coefficients[19], "18446744073709551617"},
	} {
		got, e := m.String(sample.value)
		if e != nil || got != sample.want {
			t.Fatal("owned coefficient changed after wrapper reuse", got, sample.want, e)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := w.MkSum(&constant, coefficients[:1], variables[:1]); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed wrapper accepted a sum", err)
	}
	got, err := m.String(&c.polynomials[first].terms[0].coefficient)
	if err != nil || got != "18446744073709551617" {
		t.Fatal("context polynomial depended on deleted wrapper scratch", got, err)
	}
}
