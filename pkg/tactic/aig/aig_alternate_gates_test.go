package aig

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestAIGAlternateGateEncodingTruthTablesAndCleanup(t *testing.T) {
	m, budget := aigFixture(t)
	h, e := m.MkTrue()
	aigMust(t, e)
	g, e := NewManagerWithGateEncoding(m, false)
	aigMust(t, e)
	a, e := g.MakeVariable(h)
	aigMust(t, e)
	b, e := g.MakeVariable(h)
	aigMust(t, e)
	c, e := g.MakeVariable(h)
	aigMust(t, e)
	encodedIff, e := g.Iff(a, b)
	aigMust(t, e)
	encodedIte, e := g.Ite(a, b, c)
	aigMust(t, e)
	if !encodedIff.lit.inverted || !encodedIte.lit.inverted {
		t.Fatal("alternate OR-of-AND gate topology was not selected")
	}
	aigMust(t, encodedIte.Close())
	aigMust(t, encodedIff.Close())
	for _, pair := range [][2]*Ref{{a, b}, {b, c}, {a, a}, {c, b}} {
		iff, e := g.Iff(pair[0], pair[1])
		aigMust(t, e)
		xor, e := g.Xor(pair[0], pair[1])
		aigMust(t, e)
		for mask := uint32(0); mask < 8; mask++ {
			x, y := aigEval(pair[0].lit, mask), aigEval(pair[1].lit, mask)
			if got := aigEval(iff.lit, mask); got != (x == y) {
				t.Fatalf("alternate iff %03b: %v", mask, got)
			}
			if got := aigEval(xor.lit, mask); got != (x != y) {
				t.Fatalf("alternate xor %03b: %v", mask, got)
			}
		}
		aigMust(t, xor.Close())
		aigMust(t, iff.Close())
	}
	for _, condition := range []*Ref{a, b, c} {
		for _, then := range []*Ref{a, b, c} {
			for _, otherwise := range []*Ref{a, b, c} {
				out, e := g.Ite(condition, then, otherwise)
				aigMust(t, e)
				for mask := uint32(0); mask < 8; mask++ {
					want := aigEval(otherwise.lit, mask)
					if aigEval(condition.lit, mask) {
						want = aigEval(then.lit, mask)
					}
					if got := aigEval(out.lit, mask); got != want {
						t.Fatalf("alternate ite %03b: %v want %v", mask, got, want)
					}
				}
				aigMust(t, out.Close())
			}
		}
	}
	baseline := budget.Used()
	succeeded := false
	for prefix := int64(0); prefix < 16; prefix++ {
		budget.FailAfter(prefix)
		out, failure := g.Iff(a, b)
		budget.FailAfter(-1)
		if failure == nil {
			aigMust(t, out.Close())
			succeeded = true
			break
		}
		if out != nil || !errors.Is(failure, arena.ErrInjectedFailure) {
			t.Fatalf("prefix %d: %v %v", prefix, out, failure)
		}
		if budget.Used() != baseline {
			t.Fatalf("prefix %d retained %d bytes", prefix, budget.Used()-baseline)
		}
	}
	if !succeeded {
		t.Fatal("alternate IFF allocation campaign never succeeded")
	}
	aigMust(t, c.Close())
	aigMust(t, b.Close())
	aigMust(t, a.Close())
	aigMust(t, g.Close())
}
