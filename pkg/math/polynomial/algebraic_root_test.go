package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// Observable bounded unit hooks; main-package source fixtures bind real
// reslimit/cooperation owners instead. Cancellation has no hidden default.
type algebraicTestControl struct {
	visits, cooperations, refused uint64
	limit                         uint64
}

func (c *algebraicTestControl) checkpoint() bool {
	c.visits++
	if c.visits > c.limit {
		c.refused++
		return false
	}
	return true
}
func (c *algebraicTestControl) cooperate() error { c.cooperations++; return nil }
func algebraicFixture(t *testing.T) (*AlgebraicRootCore, *UPolynomialManager) {
	c, p, _ := algebraicControlledFixture(t)
	return c, p
}
func algebraicControlledFixture(t *testing.T) (*AlgebraicRootCore, *UPolynomialManager, *algebraicTestControl) {

	t.Helper()
	s, _ := serviceNew(t)
	control := &algebraicTestControl{limit: 1000000}
	p, e := NewUPolynomialManagerFromRationalService(s, UPolynomialConfig{Checkpoint: control.checkpoint, Cooperation: control.cooperate})
	serviceMust(t, e)
	primes, e := util.NewPrimeGenerator(s.Manager.BudgetField)
	serviceMust(t, e)
	f, e := NewUPolynomialFactorizationService(p, UPolynomialFactorizationConfig{Primes: primes})
	serviceMust(t, e)
	b, e := util.NewBinaryRationalManager(s.Manager.Z, util.BinaryRationalConfig{Diagnostics: s})
	serviceMust(t, e)
	bi, e := util.NewBinaryIntervalManager(b)
	serviceMust(t, e)
	a, e := util.NewSmallAllocator(util.SmallAllocatorConfig{Runtime: s.Manager.Z.Memory, Profile: util.SmallAllocatorProfile{PointerBits: 64}})
	serviceMust(t, e)
	c, e := NewAlgebraicRootCore(AlgebraicRootConfig{Rationals: s.Manager, Polynomials: p, Factors: f, Binary: b, Intervals: bi, Allocator: a, Checkpoint: p.config.Checkpoint, Cooperation: p.config.Cooperation})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, c.Close())
		serviceMust(t, a.Close())
		bi.Close()
		serviceMust(t, b.CloseChecked())
		serviceMust(t, f.Close())
		serviceMust(t, primes.Close())
		serviceMust(t, p.Close())
		if control.visits != control.cooperations+control.refused {
			t.Fatal("checkpoint/cooperation pairing", control)
		}
	})
	return c, p, control
}
func algebraicPolynomial(t *testing.T, p *UPolynomialManager, coeff ...int32) UPolynomial {
	t.Helper()
	v, e := p.NewPolynomial()
	serviceMust(t, e)
	values := make([]util.Int, len(coeff))
	for i, n := range coeff {
		values[i] = util.NewInt(n)
	}
	serviceMust(t, p.SetInts(v, values))
	t.Cleanup(func() { serviceMust(t, p.DelPolynomial(&v)) })
	return v
}

// Selected active tst1 assertions: algebraic.cpp:57,65,91,107. Original
// multivariate polynomial-expression construction remains a separate dependency.
func TestAlgebraicSourceRootCounts(t *testing.T) {
	c, p := algebraicFixture(t)
	for _, x := range []struct {
		a []int32
		n uint32
	}{{[]int32{-2, 3}, 1}, {[]int32{-2, 0, 1}, 2}, {[]int32{-2, 1, 2, -1, 0, 2, -3, 1}, 3}, {[]int32{6, 0, -5, 0, 1}, 4}} {
		q := algebraicPolynomial(t, p, x.a...)
		v, e := p.wholeView(q)
		serviceMust(t, e)
		r, e := (c).NewVector()
		serviceMust(t, e)
		serviceMust(t, c.IsolateRoots(v, r))
		if r.Len() != x.n {
			t.Fatalf("%v roots %d want %d", x.a, r.Len(), x.n)
		}
		for i := uint32(1); i < r.Len(); i++ {
			d, e := r.Compare(i-1, i)
			serviceMust(t, e)
			if d >= 0 {
				t.Fatal("root order")
			}
		}
		serviceMust(t, r.Close())
	}
}
func TestAlgebraicRootCopyRefineAndIndex(t *testing.T) {
	c, p := algebraicFixture(t)
	q := algebraicPolynomial(t, p, -2, 0, 1)
	v, e := p.wholeView(q)
	serviceMust(t, e)
	var a, b Algebraic
	serviceMust(t, c.MkRoot(v, 2, &a))
	serviceMust(t, c.Copy(&b, &a))
	serviceMust(t, c.Refine(&a))
	d, e := c.Compare(&a, &b)
	serviceMust(t, e)
	if d != 0 {
		t.Fatal("deep root copy")
	}
	rat, e := c.IsRational(&a)
	serviceMust(t, e)
	if rat {
		t.Fatal("sqrt2 irrational")
	}
	if e = c.MkRoot(v, 0, &b); e == nil || e.Error() != "invalid root object, root index must be greater than 0" {
		t.Fatal(e)
	}
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.Del(&b))
}
func TestAlgebraicNullZeroAndRationalCells(t *testing.T) {
	c, _ := algebraicFixture(t)
	var a, b Algebraic
	z, e := c.IsZero(&a)
	serviceMust(t, e)
	if !z {
		t.Fatal("null zero")
	}
	r := util.NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&r, 2, 3))
	serviceMust(t, c.SetRational(&a, &r))
	serviceMust(t, c.Copy(&b, &a))
	serviceMust(t, c.Del(&a))
	d, e := c.CompareRational(&b, &r)
	serviceMust(t, e)
	if d != 0 {
		t.Fatal("basic deep copy")
	}
	serviceMust(t, c.Del(&b))
	serviceMust(t, c.config.Rationals.Del(&r))
}
