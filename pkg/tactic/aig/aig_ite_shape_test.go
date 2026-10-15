package aig

import "testing"

func TestAIGITEShapeFourSourceComplementPositions(t *testing.T) {
	m, _ := aigFixture(t)
	h, e := m.MkTrue()
	aigMust(t, e)
	g, e := NewManager(m)
	aigMust(t, e)
	a, e := g.MakeVariable(h)
	aigMust(t, e)
	b, e := g.MakeVariable(h)
	aigMust(t, e)
	c, e := g.MakeVariable(h)
	aigMust(t, e)
	na, e := a.Not()
	aigMust(t, e)
	for _, row := range []struct{ left, right [2]*Ref }{
		{[2]*Ref{a, b}, [2]*Ref{na, c}},
		{[2]*Ref{a, b}, [2]*Ref{c, na}},
		{[2]*Ref{b, a}, [2]*Ref{na, c}},
		{[2]*Ref{b, a}, [2]*Ref{c, na}},
		{[2]*Ref{na, b}, [2]*Ref{a, c}},
		{[2]*Ref{b, na}, [2]*Ref{c, a}},
	} {
		l, e := g.InternAnd(row.left[0], row.left[1])
		aigMust(t, e)
		r, e := g.InternAnd(row.right[0], row.right[1])
		aigMust(t, e)
		nl, e := l.Not()
		aigMust(t, e)
		nr, e := r.Not()
		aigMust(t, e)
		root, e := g.InternAnd(nl, nr)
		aigMust(t, e)
		found, condition, then, otherwise := iteShape(root.lit.node)
		if !found {
			t.Fatal("source complementary-child pattern not recognized")
		}
		for mask := uint32(0); mask < 8; mask++ {
			want := aigEval(otherwise, mask)
			if aigEval(condition, mask) {
				want = aigEval(then, mask)
			}
			if got := aigEval(root.lit, mask); got != want {
				t.Fatalf("ITE shape mask %03b: got %v want %v", mask, got, want)
			}
		}
		aigMust(t, root.Close())
		aigMust(t, nr.Close())
		aigMust(t, nl.Close())
		aigMust(t, r.Close())
		aigMust(t, l.Close())
	}
	plain, e := g.InternAnd(a, b)
	aigMust(t, e)
	if found, _, _, _ := iteShape(plain.lit.node); found {
		t.Fatal("plain AND matched ITE shape")
	}
	aigMust(t, plain.Close())
	aigMust(t, na.Close())
	aigMust(t, c.Close())
	aigMust(t, b.Close())
	aigMust(t, a.Close())
	aigMust(t, g.Close())
}
