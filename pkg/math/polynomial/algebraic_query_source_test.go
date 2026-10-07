// Source: src/math/polynomial/algebraic_numbers.cpp:217-240,350-356,2807-2828.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// These are independent API assertions. Existing registered algebraic root
// drivers reach the cells but do not contain corresponding query ENSUREs.
func TestAlgebraicSignAndDegreeLiteralBranches(t *testing.T) {
	c, p := algebraicFixture(t)
	t.Cleanup(func() { c.config.Rationals.Budget().FailAfter(-1) })
	check := func(a *Algebraic, pos, neg bool, degree uint32) {
		t.Helper()
		used := c.config.Rationals.Budget().Used()
		physical := c.config.Rationals.Integers().RuntimeOwner().Stats()
		c.config.Rationals.Budget().FailAfter(0)
		actual, e := c.IsPos(a)
		serviceMust(t, e)
		if actual != pos {
			t.Fatal("positive", actual, pos)
		}
		actual, e = c.IsNeg(a)
		serviceMust(t, e)
		if actual != neg {
			t.Fatal("negative", actual, neg)
		}
		d, e := c.Degree(a)
		serviceMust(t, e)
		if d != degree {
			t.Fatal("degree", d, degree)
		}
		if c.config.Rationals.Budget().Used() != used || physical != c.config.Rationals.Integers().RuntimeOwner().Stats() {
			t.Fatal("readonly query allocated")
		}
		c.config.Rationals.Budget().FailAfter(-1)
	}
	var zero, a Algebraic
	check(&zero, false, false, 0)
	q := util.NewRational()
	for _, n := range []int32{-3, 3} {
		serviceMust(t, c.config.Rationals.SetFraction(&q, int64(n), 2))
		serviceMust(t, c.SetRational(&a, &q))
		check(&a, n > 0, n < 0, 1)
		serviceMust(t, c.Del(&a))
	}
	serviceMust(t, c.config.Rationals.Del(&q))
	poly := algebraicPolynomial(t, p, -2, 0, 1)
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	for i := uint32(1); i <= 2; i++ {
		serviceMust(t, c.MkRoot(v, i, &a))
		check(&a, i == 2, i == 1, 2)
		serviceMust(t, c.Del(&a))
	}
}

// Invalid Go ownership guards are distinct from native numerical assertions.
func TestAlgebraicQueriesRejectCopiedForeignAndStaleOwners(t *testing.T) {
	c, _ := algebraicFixture(t)
	q := util.NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&q, 3, 2))
	var a Algebraic
	serviceMust(t, c.SetRational(&a, &q))
	stale := a
	copied := *c
	used, physical := c.config.Rationals.Budget().Used(), c.config.Rationals.Integers().RuntimeOwner().Stats()
	if _, e := copied.IsPos(&a); !errors.Is(e, ErrAlgebraicOwner) {
		t.Fatal(e)
	}
	if _, e := copied.IsNeg(&a); !errors.Is(e, ErrAlgebraicOwner) {
		t.Fatal(e)
	}
	if _, e := copied.Degree(&a); !errors.Is(e, ErrAlgebraicOwner) {
		t.Fatal(e)
	}
	foreign := a
	foreign.owner++
	if _, e := c.IsPos(&foreign); !errors.Is(e, ErrAlgebraicOwner) {
		t.Fatal(e)
	}
	if _, e := c.IsNeg(&foreign); !errors.Is(e, ErrAlgebraicOwner) {
		t.Fatal(e)
	}
	if _, e := c.Degree(&foreign); !errors.Is(e, ErrAlgebraicOwner) {
		t.Fatal(e)
	}
	if c.config.Rationals.Budget().Used() != used || c.config.Rationals.Integers().RuntimeOwner().Stats() != physical {
		t.Fatal("guard mutated actual graph")
	}
	serviceMust(t, c.Del(&a))
	if _, e := c.IsPos(&stale); !errors.Is(e, ErrAlgebraicOwner) {
		t.Fatal(e)
	}
	if _, e := c.IsNeg(&stale); !errors.Is(e, ErrAlgebraicOwner) {
		t.Fatal(e)
	}
	if _, e := c.Degree(&stale); !errors.Is(e, ErrAlgebraicOwner) {
		t.Fatal(e)
	}
	serviceMust(t, c.config.Rationals.Del(&q))
}

func TestAlgebraicDegreeUsesActualNonminimalPolynomial(t *testing.T) {
	c, p := algebraicFixture(t)
	poly := algebraicPolynomial(t, p, -4, 0, 0, 0, 1)
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	var lower, upper util.BinaryRational
	serviceMust(t, c.config.Binary.SetInt32(&lower, 1))
	serviceMust(t, c.config.Binary.SetInt32(&upper, 2))
	var a Algebraic
	serviceMust(t, c.SetRoot(&a, v, &lower, &upper, false))
	d, e := c.Degree(&a)
	serviceMust(t, e)
	if d != 4 {
		t.Fatal("source stored polynomial degree", d)
	}
	pos, e := c.IsPos(&a)
	serviceMust(t, e)
	if !pos {
		t.Fatal("actual positive interval")
	}
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.config.Binary.Del(&upper))
	serviceMust(t, c.config.Binary.Del(&lower))
}
