package aig

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestAIGMaxSharingReassociatesTowardExistingPair(t *testing.T) {
	m, budget := aigFixture(t)
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
	ab, e := g.And(a, b)
	aigMust(t, e)
	abc, e := g.And(ab, c)
	aigMust(t, e)
	aigMust(t, ab.Close()) // only abc now owns the one-use nested AND
	bc, e := g.And(b, c)
	aigMust(t, e)
	result, e := g.MaxSharing(abc)
	aigMust(t, e)
	if result.Same(abc) {
		t.Fatal("source sharing pass did not reassociate")
	}
	if !result.lit.node.and || result.lit.node.children[0].node != bc.lit.node && result.lit.node.children[1].node != bc.lit.node {
		t.Fatal("reassociation did not reuse the preexisting b/c node")
	}
	for mask := uint32(0); mask < 8; mask++ {
		if got, want := aigEval(result.lit, mask), aigEval(abc.lit, mask); got != want {
			t.Fatalf("sharing mask %03b: %v want %v", mask, got, want)
		}
	}
	aigMust(t, result.Close())
	before := budget.Used()
	reached := false
	for prefix := int64(0); prefix < 32; prefix++ {
		budget.FailAfter(prefix)
		trial, failure := g.MaxSharing(abc)
		budget.FailAfter(-1)
		if failure == nil {
			aigMust(t, trial.Close())
			reached = true
			break
		}
		if trial != nil || !errors.Is(failure, arena.ErrInjectedFailure) {
			t.Fatalf("sharing prefix %d: %v %v", prefix, trial, failure)
		}
		if budget.Used() != before {
			t.Fatalf("sharing prefix %d retained %d bytes", prefix, budget.Used()-before)
		}
	}
	if !reached {
		t.Fatal("sharing allocation campaign never succeeded")
	}
	aigMust(t, bc.Close())
	aigMust(t, abc.Close())
	aigMust(t, c.Close())
	aigMust(t, b.Close())
	aigMust(t, a.Close())
	aigMust(t, g.Close())
}

func TestAIGMaxSharingUnchangedAndInvertedRoot(t *testing.T) {
	m, _ := aigFixture(t)
	h, e := m.MkTrue()
	aigMust(t, e)
	g, e := NewManager(m)
	aigMust(t, e)
	a, e := g.MakeVariable(h)
	aigMust(t, e)
	b, e := g.MakeVariable(h)
	aigMust(t, e)
	ab, e := g.And(a, b)
	aigMust(t, e)
	notAB, e := ab.Not()
	aigMust(t, e)
	plain, e := g.MaxSharing(ab)
	aigMust(t, e)
	negative, e := g.MaxSharing(notAB)
	aigMust(t, e)
	if !plain.Same(ab) || !negative.Same(notAB) {
		t.Fatal("unchanged root lost identity or polarity")
	}
	aigMust(t, negative.Close())
	aigMust(t, plain.Close())
	aigMust(t, notAB.Close())
	aigMust(t, ab.Close())
	aigMust(t, b.Close())
	aigMust(t, a.Close())
	aigMust(t, g.Close())
}

func TestAIGMaxSharingCachesSharedOldNode(t *testing.T) {
	m, _ := aigFixture(t)
	h, e := m.MkTrue()
	aigMust(t, e)
	g, e := NewManager(m)
	aigMust(t, e)
	a, e := g.MakeVariable(h)
	aigMust(t, e)
	b, e := g.MakeVariable(h)
	aigMust(t, e)
	ab, e := g.And(a, b)
	aigMust(t, e)
	duplicated, e := g.InternAnd(ab, ab)
	aigMust(t, e)
	result, e := g.MaxSharing(duplicated)
	aigMust(t, e)
	if !result.Same(ab) {
		t.Fatal("shared old-node cache did not collapse the duplicate root")
	}
	count, e := g.NumNodes()
	aigMust(t, e)
	if count != 5 {
		t.Fatalf("shared graph changed node ownership: %d", count)
	}
	aigMust(t, result.Close())
	aigMust(t, duplicated.Close())
	aigMust(t, ab.Close())
	aigMust(t, b.Close())
	aigMust(t, a.Close())
	aigMust(t, g.Close())
}
