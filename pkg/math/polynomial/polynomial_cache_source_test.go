package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestPolynomialCacheConstructorBudgetAndRefusal(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	b := m.core.integer.Budget()
	before := b.Used()
	b.FailAfter(0)
	_, e := NewPolynomialCache(m)
	b.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != before {
		t.Fatal("failed cache construction changed owner budget", e, b.Used(), before)
	}
	c, e := NewPolynomialCache(m)
	serviceMust(t, e)
	if b.Used() <= before {
		t.Fatal("cache owner was not charged")
	}
	serviceMust(t, c.Close())
	if b.Used() != before {
		t.Fatal("cache owner charge survived close", b.Used(), before)
	}
}

func TestPolynomialCacheCanonicalizationAndReset(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.Close()) })
	p := polynomialQueryTerms(t, m.core, []int32{3, 5, 7}, [][]MonomialPower{{{0, 2}}, {{1, 1}}, nil})
	q := polynomialQueryTerms(t, m.core, []int32{7, 3, 5}, [][]MonomialPower{nil, {{0, 2}}, {{1, 1}}})
	r := polynomialQueryTerms(t, m.core, []int32{7, 3, 6}, [][]MonomialPower{nil, {{0, 2}}, {{1, 1}}})
	first, e := c.MkUnique(q)
	serviceMust(t, e)
	if first != q {
		t.Fatal("first equal polynomial was not retained")
	}
	canonical, e := c.MkUnique(p)
	serviceMust(t, e)
	if canonical != q {
		t.Fatal("equal polynomial did not reuse first cache entry")
	}
	distinct, e := c.MkUnique(r)
	serviceMust(t, e)
	if distinct != r {
		t.Fatal("different polynomial reused an equal bucket entry")
	}
	for _, item := range []Polynomial{q, r} {
		info, e := m.core.Info(item)
		serviceMust(t, e)
		if info.RefCount != 2 {
			t.Fatal("cache did not own exactly one reference", info)
		}
	}
	again, e := c.MkUnique(q)
	serviceMust(t, e)
	if again != q {
		t.Fatal("cached ID did not return itself")
	}
	info, e := m.core.Info(q)
	serviceMust(t, e)
	if info.RefCount != 2 {
		t.Fatal("repeated lookup added a reference", info)
	}
	serviceMust(t, c.Reset())
	for _, item := range []Polynomial{q, r} {
		info, e := m.core.Info(item)
		serviceMust(t, e)
		if info.RefCount != 1 {
			t.Fatal("reset did not release the cached reference", info)
		}
	}
	first, e = c.MkUnique(p)
	serviceMust(t, e)
	if first != p {
		t.Fatal("reset retained the former representative")
	}
	canonical, e = c.MkUnique(q)
	serviceMust(t, e)
	if canonical != p {
		t.Fatal("new representative did not win after reset")
	}
	serviceMust(t, c.Close())
	if _, e := c.MkUnique(p); e != arena.ErrClosed {
		t.Fatal("closed cache accepted lookup", e)
	}
}
