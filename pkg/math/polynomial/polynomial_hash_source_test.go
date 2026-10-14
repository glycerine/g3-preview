package polynomial

import "testing"

func TestPolynomialHashCacheKeysAndLexicalOrder(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	p := polynomialQueryTerms(t, m.core, []int32{3, 5, 7}, [][]MonomialPower{{{0, 2}}, {{1, 1}}, nil})
	q := polynomialQueryTerms(t, m.core, []int32{7, 3, 5}, [][]MonomialPower{nil, {{0, 2}}, {{1, 1}}})
	r := polynomialQueryTerms(t, m.core, []int32{7, 3, 6}, [][]MonomialPower{nil, {{0, 2}}, {{1, 1}}})
	equal, e := m.core.Equal(p, q)
	serviceMust(t, e)
	if !equal || p == q {
		t.Fatal("expected distinct equal polynomials")
	}
	hp, e := m.Hash(p)
	serviceMust(t, e)
	hq, e := m.Hash(q)
	serviceMust(t, e)
	if hp != hq {
		t.Fatal("term order changed equal polynomial hash", hp, hq)
	}
	info, e := m.core.Info(p)
	serviceMust(t, e)
	if !info.LexSorted {
		t.Fatal("source hash did not lexically sort its input")
	}
	again, e := m.Hash(p)
	serviceMust(t, e)
	if again != hp {
		t.Fatal("hash changed on repeated cache lookup", hp, again)
	}
	hr, e := m.Hash(r)
	serviceMust(t, e)
	if hr == hp {
		t.Fatal("changed coefficient retained the same fixture hash")
	}
	empty := polynomialQueryTerms(t, m.core, nil, nil)
	he, e := m.Hash(empty)
	serviceMust(t, e)
	if he != 31 {
		t.Fatal("source empty polynomial hash", he)
	}
}
