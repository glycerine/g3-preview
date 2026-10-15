package aig

import "testing"

func TestAIGTwoLevelAndRules(t *testing.T) {
	m, b := aigFixture(t)
	h, e := m.MkTrue()
	aigMust(t, e)
	g, e := NewManager(m)
	aigMust(t, e)
	var owned []*Ref
	keep := func(r *Ref, e error) *Ref { t.Helper(); aigMust(t, e); owned = append(owned, r); return r }
	a := keep(g.MakeVariable(h))
	c := keep(g.MakeVariable(h))
	d := keep(g.MakeVariable(h))
	truth := keep(g.True())
	falsehood := keep(g.False())
	notA := keep(a.Not())
	notC := keep(c.Not())
	ac := keep(g.And(a, c))
	ca := keep(g.And(c, a))
	if !ac.Same(ca) {
		t.Fatal("mk_node did not sort children")
	}
	for _, pair := range []struct{ l, r, want *Ref }{
		{truth, a, a}, {falsehood, a, falsehood}, {a, notA, falsehood},
		{a, ac, ac}, {ac, a, ac},
	} {
		got := keep(g.And(pair.l, pair.r))
		if !got.Same(pair.want) {
			t.Fatal("one-level identity, contradiction or absorption failed")
		}
	}
	if got := keep(g.And(ac, notA)); !got.Same(falsehood) {
		t.Fatal("child contradiction")
	}
	ad := keep(g.And(a, d))
	if got := keep(g.And(ac, ad)); got.Same(ac) || got.Same(ad) {
		t.Fatal("two-level shared child lost third conjunct")
	}
	notAC := keep(ac.Not())
	if got := keep(g.And(notAC, a)); got.Same(a) || got.Same(falsehood) {
		t.Fatal("inverted child substitution")
	}
	aNotC := keep(g.And(a, notC))
	notANotC := keep(aNotC.Not())
	wantNotA := keep(notA.Clone())
	if got := keep(g.And(notAC, notANotC)); !got.Same(wantNotA) {
		t.Fatal("two-level resolution")
	}
	before := b.Used()
	b.FailAfter(0)
	got := keep(g.And(truth, a))
	b.FailAfter(-1)
	if !got.Same(a) || b.Used() != before {
		t.Fatal("identity path allocated a node")
	}
	for i := len(owned) - 1; i >= 0; i-- {
		aigMust(t, owned[i].Close())
	}
	aigMust(t, g.Close())
}

func aigEval(l Literal, mask uint32) bool {
	var value bool
	if l.node.id == 0 {
		value = true
	} else if l.node.and {
		value = aigEval(l.node.children[0], mask) && aigEval(l.node.children[1], mask)
	} else {
		value = mask&(1<<(l.node.id-1)) != 0
	}
	if l.inverted {
		return !value
	}
	return value
}

func TestAIGTwoLevelAndTruthTables(t *testing.T) {
	m, _ := aigFixture(t)
	h, e := m.MkTrue()
	aigMust(t, e)
	g, e := NewManager(m)
	aigMust(t, e)
	var owned []*Ref
	keep := func(r *Ref, e error) *Ref { t.Helper(); aigMust(t, e); owned = append(owned, r); return r }
	a, b, c := keep(g.MakeVariable(h)), keep(g.MakeVariable(h)), keep(g.MakeVariable(h))
	na, nb, nc := keep(a.Not()), keep(b.Not()), keep(c.Not())
	ab, anb := keep(g.And(a, b)), keep(g.And(a, nb))
	bc, bnc := keep(g.And(b, c)), keep(g.And(b, nc))
	nab, nbc := keep(ab.Not()), keep(bc.Not())
	items := []*Ref{a, b, c, na, nb, nc, ab, anb, bc, bnc, nab, nbc}
	for i, left := range items {
		for j, right := range items {
			result, e := g.And(left, right)
			aigMust(t, e)
			for mask := uint32(0); mask < 8; mask++ {
				got := aigEval(result.lit, mask)
				want := aigEval(left.lit, mask) && aigEval(right.lit, mask)
				if got != want {
					t.Fatalf("AND pair %d,%d assignment %03b: got %v want %v", i, j, mask, got, want)
				}
			}
			aigMust(t, result.Close())
		}
	}
	for i := len(owned) - 1; i >= 0; i-- {
		aigMust(t, owned[i].Close())
	}
	aigMust(t, g.Close())
}
