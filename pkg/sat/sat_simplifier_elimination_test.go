// Source: src/sat/sat_simplifier.cpp:1085–1480.
package sat

import (
	"testing"
)

func TestSATSimplifierEliminationCostAndStableOrder(t *testing.T) {
	s, l := satSimplifierFixture(t, 3)
	satOK(t, s.useList.Init(3))
	for _, ls := range [][]Literal{{0, 2, 4}, {1, 2, 5}} {
		c, e := l.MkClauseCore(uint32(len(ls)), ls, false)
		satOK(t, e)
		satOK(t, s.useList.Insert(c))
	}
	c, e := s.eliminationCost(0)
	satOK(t, e)
	if c != 2 {
		t.Fatal("source pair cost", c)
	}
	// Variable 1 costs zero. Equal-cost variables 2 and 0 retain insertion order.
	satOK(t, l.propagation.variables.elimTodo.Reset())
	for _, v := range []uint32{2, 0, 1} {
		satOK(t, l.propagation.variables.elimTodo.Insert(v))
	}
	s.elimination.vars.Bind(l.propagation.memory)
	satOK(t, s.orderVariables())
	for i, want := range []uint32{1, 2, 0} {
		got, e := s.elimination.vars.Read(uint32(i))
		satOK(t, e)
		if got != want {
			t.Fatal("stable source order", i, got, want)
		}
	}
	if !l.propagation.variables.elimTodo.Empty() {
		t.Fatal("source reset before sorted publication")
	}
	satOK(t, s.closeElimination())
}

func TestSATSimplifierActualTernaryAndNaryResolventPublication(t *testing.T) {
	for _, last := range [][]Literal{{1, 4, 6}, {1, 6, 8}} {
		t.Run(string(rune('a'+last[1])), func(t *testing.T) {
			s, l := satSimplifierFixture(t, 5)
			p := l.propagation
			satOK(t, s.useList.Init(5))
			satOK(t, s.kernel.InitVisited(5))
			s.kernel.subCounter = 1000
			s.kernel.elimCounter = 1000
			old := make([]StoredClauseRef, 0, 2)
			for _, ls := range [][]Literal{{0, 2, 4}, last} {
				c, e := l.MkClauseCore(3, ls, false)
				satOK(t, e)
				old = append(old, c)
				satOK(t, s.useList.Insert(c))
			}
			beforeTer, beforeNary := l.mkTernary, l.mkNary
			yes, e := s.tryEliminate(0)
			satOK(t, e)
			if !yes {
				t.Fatal("finite source elimination")
			}
			if p.converter.Size() != 1 {
				t.Fatal("real elimination converter")
			}
			if len(l.Original) != 3 {
				t.Fatal("source publishes new owner before final deletion", len(l.Original))
			}
			c := l.Original[2]
			n, e := p.clauses.Size(c)
			satOK(t, e)
			want := []Literal{2, 4, 6}
			if last[1] == 6 {
				want = []Literal{2, 4, 6, 8}
			}
			if n != uint32(len(want)) {
				t.Fatal(n, want)
			}
			for i, w := range want {
				x, e := p.clauses.Literal(c, uint32(i))
				satOK(t, e)
				if x != w {
					t.Fatal("source literal order", x, w)
				}
			}
			if n == 3 {
				if l.mkTernary != beforeTer+1 || l.mkNary != beforeNary {
					t.Fatal("source ternary stat before allocation")
				}
			} else if l.mkNary != beforeNary+1 || l.mkTernary != beforeTer {
				t.Fatal("source nary stat before allocation")
			}
			for _, r := range old {
				removed, e := p.clauses.Removed(r)
				satOK(t, e)
				if !removed {
					t.Fatal("source removed prefix")
				}
			}
			satSimplifierAudit(t, s)
			// The owner enters the real enclosing destructor boundary; no fake attach.
			s.scopeActive = true
			l.simplifierContinuation = s
			s.numElimVars = 1
			s.oldNumElimVars = 0
			satOK(t, s.scopedFinalize())
			if len(l.Original) != 1 || l.Original[0] != c {
				t.Fatal("actual kept resolvent")
			}
			satSimplifierAudit(t, s)
		})
	}
}

func TestSATSimplifierSourceCutoffsAndTooManyResolventsPrefix(t *testing.T) {
	for cutoff := 0; cutoff < 4; cutoff++ {
		t.Run(string(rune('a'+cutoff)), func(t *testing.T) {
			s, l := satSimplifierFixture(t, 5)
			satOK(t, s.useList.Init(5))
			satOK(t, s.kernel.InitVisited(5))
			for _, ls := range [][]Literal{{0, 2, 4}, {1, 6, 8}} {
				c, e := l.MkClauseCore(3, ls, false)
				satOK(t, e)
				satOK(t, s.useList.Insert(c))
			}
			s.occCutoff = ^uint32(0)
			s.occRanges = [3]uint32{^uint32(0), ^uint32(0), ^uint32(0)}
			if cutoff == 0 {
				s.occCutoff = 1
			} else {
				s.occRanges[cutoff-1] = 1
				s.litRanges[cutoff-1] = 0
				switch cutoff {
				case 1:
					s.clsCutoffs = [2]uint32{3, 5}
				case 2:
					s.clsCutoffs = [2]uint32{0, 5}
				case 3:
					s.clsCutoffs = [2]uint32{0, 0}
				}
			}
			s.kernel.elimCounter = 1000
			yes, e := s.tryEliminate(0)
			satOK(t, e)
			eliminated, e := l.propagation.variables.eliminated.Read(0)
			satOK(t, e)
			if yes || eliminated != 0 || l.propagation.converter.Size() != 0 || s.kernel.elimCounter != 1000 {
				t.Fatal("literal source cutoff before resolve/MC publication")
			}
			satSimplifierAudit(t, s)
		})
	}
	s, l := satSimplifierFixture(t, 9)
	satOK(t, s.useList.Init(9))
	satOK(t, s.kernel.InitVisited(9))
	s.kernel.elimCounter = 1000
	for _, base := range []uint32{0, 1} {
		for i := uint32(0); i < 3; i++ {
			a := Literal(2 + 2*i)
			last := Literal(14)
			if base == 1 {
				a += 6
				last = 16
			}
			c, e := l.MkClauseCore(3, []Literal{Literal(base), a, last}, false)
			satOK(t, e)
			satOK(t, s.useList.Insert(c))
		}
	}
	yes, e := s.tryEliminate(0)
	satOK(t, e)
	if yes || l.propagation.converter.Size() != 0 || s.kernel.elimCounter != 958 {
		t.Fatal("seventh resolvent exceeds six originals after exact seven resolve costs", yes, s.kernel.elimCounter)
	}
	for _, b := range s.kernel.visited {
		if b != 0 {
			t.Fatal("normal resolve unmarks input after too-many cutoff")
		}
	}
	satSimplifierAudit(t, s)
}
