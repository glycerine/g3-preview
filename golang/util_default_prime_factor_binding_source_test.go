// Source: src/util/prime_generator.cpp:99-107;
// src/math/polynomial/polynomial.cpp:6683-6725,6737-6830.
// The borrowed-core accessor is explicit Go binding, not a C++ getter port.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"testing"
)

func TestDefaultPrimeBorrowFullPolynomialFactorEndToEnd(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	q, e := mpnum.NewRationalManager(mpnum.RationalConfig{Integers: mpnum.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	smallObjectMust(t, e)
	s, e := NewDefaultPrimeService(b)
	smallObjectMust(t, e)
	g, e := s.BorrowGenerator()
	smallObjectMust(t, e)
	mm, e := mpnum.NewMonomialManager(mpnum.MonomialManagerConfig{Budget: b, Memory: q.Integers().RuntimeOwner()})
	smallObjectMust(t, e)
	x, e := mm.MkVar()
	smallObjectMust(t, e)
	full, e := mpnum.NewMultivariatePolynomialManager(mpnum.MultivariatePolynomialManagerConfig{Core: mpnum.MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}, Factorization: mpnum.UPolynomialFactorizationConfig{Primes: g}})
	smallObjectMust(t, e)
	c, e := full.ArithmeticCore()
	smallObjectMust(t, e)
	copied := *full
	if _, e = copied.ArithmeticCore(); e != mpnum.ErrPolynomialOwner {
		t.Fatal("copied borrowed owner", e)
	}
	p, e := c.MkUnivariateTake(x, 2, []mpnum.Int{mpnum.NewInt(-1), mpnum.NewInt(0), mpnum.NewInt(1)})
	smallObjectMust(t, e)
	input, e := mpnum.NewPolynomialRef(c)
	smallObjectMust(t, e)
	smallObjectMust(t, input.Assign(p))
	fs, e := mpnum.NewPolynomialFactors(full)
	smallObjectMust(t, e)
	smallObjectMust(t, full.Factor(p, fs, mpnum.DefaultUPolynomialFactorParams()))
	n, e := fs.Distinct()
	smallObjectMust(t, e)
	if n != 2 {
		t.Fatal("literal borrowed-backend factors", n)
	}
	product, e := mpnum.NewPolynomialRef(c)
	smallObjectMust(t, e)
	smallObjectMust(t, fs.Multiply(product))
	v, e := product.Get()
	smallObjectMust(t, e)
	same, e := c.Equal(p, v)
	smallObjectMust(t, e)
	if !same {
		t.Fatal("borrowed backend reconstruction")
	}
	smallObjectMust(t, product.Close())
	smallObjectMust(t, fs.CloseChecked())
	smallObjectMust(t, input.Close())
	smallObjectMust(t, full.CloseChecked())
	if _, e = full.ArithmeticCore(); e != arena.ErrClosed {
		t.Fatal("closed borrowed owner", e)
	}
	again, e := s.BorrowGenerator()
	smallObjectMust(t, e)
	if again != g {
		t.Fatal("Factor acquired or replaced borrowed prime owner")
	}
	smallObjectMust(t, mm.CloseChecked())
	smallObjectMust(t, s.Close())
	smallObjectMust(t, q.CloseChecked())
	if b.Used() != 0 {
		t.Fatal("actual public binding teardown", b.Used())
	}
}
