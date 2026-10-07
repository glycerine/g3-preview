// Source: src/util/prime_generator.cpp:99-107;
// src/math/polynomial/polynomial.cpp:6683-6725,6737-6830.
// The borrowed-core accessor is explicit Go binding, not a C++ getter port.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestDefaultPrimeBorrowFullPolynomialFactorEndToEnd(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	q, e := util.NewRationalManager(util.RationalConfig{Integers: util.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	polynomialBindingMust(t, e)
	s, e := util.NewDefaultPrimeService(b)
	polynomialBindingMust(t, e)
	g, e := s.BorrowGenerator()
	polynomialBindingMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: q.Integers().RuntimeOwner()})
	polynomialBindingMust(t, e)
	x, e := mm.MkVar()
	polynomialBindingMust(t, e)
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}, Factorization: UPolynomialFactorizationConfig{Primes: g}})
	polynomialBindingMust(t, e)
	c, e := full.ArithmeticCore()
	polynomialBindingMust(t, e)
	copied := *full
	if _, e = copied.ArithmeticCore(); e != ErrPolynomialOwner {
		t.Fatal("copied borrowed owner", e)
	}
	p, e := c.MkUnivariateTake(x, 2, []util.Int{util.NewInt(-1), util.NewInt(0), util.NewInt(1)})
	polynomialBindingMust(t, e)
	input, e := NewPolynomialRef(c)
	polynomialBindingMust(t, e)
	polynomialBindingMust(t, input.Assign(p))
	fs, e := NewPolynomialFactors(full)
	polynomialBindingMust(t, e)
	polynomialBindingMust(t, full.Factor(p, fs, DefaultUPolynomialFactorParams()))
	n, e := fs.Distinct()
	polynomialBindingMust(t, e)
	if n != 2 {
		t.Fatal("literal borrowed-backend factors", n)
	}
	product, e := NewPolynomialRef(c)
	polynomialBindingMust(t, e)
	polynomialBindingMust(t, fs.Multiply(product))
	v, e := product.Get()
	polynomialBindingMust(t, e)
	same, e := c.Equal(p, v)
	polynomialBindingMust(t, e)
	if !same {
		t.Fatal("borrowed backend reconstruction")
	}
	polynomialBindingMust(t, product.Close())
	polynomialBindingMust(t, fs.CloseChecked())
	polynomialBindingMust(t, input.Close())
	polynomialBindingMust(t, full.CloseChecked())
	if _, e = full.ArithmeticCore(); e != arena.ErrClosed {
		t.Fatal("closed borrowed owner", e)
	}
	again, e := s.BorrowGenerator()
	polynomialBindingMust(t, e)
	if again != g {
		t.Fatal("Factor acquired or replaced borrowed prime owner")
	}
	polynomialBindingMust(t, mm.CloseChecked())
	polynomialBindingMust(t, s.Close())
	polynomialBindingMust(t, q.CloseChecked())
	if b.Used() != 0 {
		t.Fatal("actual public binding teardown", b.Used())
	}
}

func polynomialBindingMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
