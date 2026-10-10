package sat

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSATSolverLemmaMinimizationIndependentFiniteDAG(t *testing.T) {
	for seed := 0; seed < 64; seed++ {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			s, p, v, _ := satLemmaFixture(t, 8)
			parents := [8][]uint32{}
			decision := [8]bool{}
			decision[2] = true
			for id := uint32(0); id < 8; id++ {
				level := uint32(1)
				if id == 0 {
					level = 0
				}
				v.scopeLevel = level
				j := satJustification{}
				if id >= 3 {
					a := uint32((seed + int(id)*3) % int(id))
					b := uint32((seed*7 + int(id)) % int(id))
					parents[id] = []uint32{a, b}
					j = newSATTernaryJustification(MakeLiteral(BoolVar(a), true), MakeLiteral(BoolVar(b), true))
				}
				satOK(t, p.assignCore(MakeLiteral(BoolVar(id), false), j))
			}
			v.mark[1] = true
			s.levelSet.insert(1)
			var implied func(uint32) bool
			implied = func(id uint32) bool {
				if id <= 1 {
					return true
				}
				if decision[id] {
					return false
				}
				for _, a := range parents[id] {
					if !implied(a) {
						return false
					}
				}
				return true
			}
			for id := uint32(3); id < 8; id++ {
				got, e := s.impliedByMarked(MakeLiteral(BoolVar(id), false))
				satOK(t, e)
				if got != implied(id) {
					t.Fatal("independent decision-leaf reachability", id, got, parents)
				}
				satOK(t, s.resetUnmark(0))
				if !v.mark[1] {
					t.Fatal("preexisting anchor mark")
				}
			}
			satLemmaAudit(t, s)
		})
	}
}
func TestSATSolverLemmaMinimizationSourceCompactionAndApproxCollision(t *testing.T) {
	s, p, v, _ := satLemmaFixture(t, 6)
	for id := uint32(0); id < 6; id++ {
		v.scopeLevel = 1
		j := satJustification{}
		if id == 2 {
			j = newSATBinaryJustification(MakeLiteral(1, true))
		}
		if id == 3 {
			v.scopeLevel = 0
		}
		if id == 5 {
			v.scopeLevel = 33
			j = newSATBinaryJustification(MakeLiteral(4, true))
		}
		satOK(t, p.assignCore(MakeLiteral(BoolVar(id), false), j))
	}
	for _, id := range []uint32{1, 2, 3, 5} {
		v.mark[id] = true
	}
	for _, id := range []uint32{0, 1, 2, 3, 5} {
		satOK(t, s.lemma.Push(uint32(MakeLiteral(BoolVar(id), true))))
	}
	satOK(t, s.minimizeLemma())
	want := []uint32{1, 3, 11}
	n, e := s.lemma.Size()
	satOK(t, e)
	if n != uint32(len(want)) || s.minimizedLits != 2 {
		t.Fatal(n, s.minimizedLits)
	}
	for i, l := range want {
		got, e := s.lemma.Read(uint32(i))
		satOK(t, e)
		if got != l {
			t.Fatal(i, got, l)
		}
	}
	// level33 collides with1 but its unmarked decision4 must still be visited,
	// so var5 stays essential. Removed var2/basevar3 are unmarked ascending.
	if v.mark[2] || v.mark[3] || !v.mark[1] || !v.mark[5] {
		t.Fatal("source mark cleanup")
	}
	satOK(t, s.resetLemmaVarMarks())
	satLemmaAudit(t, s)
}
func TestSATSolverLemmaMinimizationClauseBranchesAndRejectedSuffix(t *testing.T) {
	for position := 0; position < 2; position++ {
		t.Run(fmt.Sprint(position), func(t *testing.T) {
			s, p, v, _ := satLemmaFixture(t, 5)
			lits := []Literal{8, 2, 4, 6}
			if position == 1 {
				lits[0], lits[1] = lits[1], lits[0]
			}
			c, e := p.clauses.MkClause(lits, false)
			satOK(t, e)
			o, e := p.clauses.Offset(c)
			satOK(t, e)
			for id := uint32(0); id < 5; id++ {
				v.scopeLevel = 1
				j := satJustification{}
				if id == 4 {
					j = newSATClauseJustification(o)
				}
				satOK(t, p.assignCore(MakeLiteral(BoolVar(id), false), j))
			}
			s.levelSet.insert(1)
			v.mark[1] = true
			satOK(t, s.unmark.Push(1))
			got, e := s.impliedByMarked(8)
			satOK(t, e)
			if got {
				t.Fatal("decision antecedents remain essential")
			}
			n, e := s.unmark.Size()
			satOK(t, e)
			if n != 1 || v.mark[2] || v.mark[3] || !v.mark[1] {
				t.Fatal("only attempted suffix reset")
			}
			satLemmaAudit(t, s)
			satOK(t, p.clauses.Delete(c))
		})
	}
}
func TestSATSolverLemmaMinimizationTernaryShortCircuitAndExternalGuard(t *testing.T) {
	s, p, v, _ := satLemmaFixture(t, 4)
	for id := uint32(0); id < 4; id++ {
		v.scopeLevel = 1
		j := satJustification{}
		if id == 2 {
			j = newSATTernaryJustification(2, 0xffffffff)
		}
		if id == 3 {
			j = newSATExternalJustification(4)
		}
		satOK(t, p.assignCore(MakeLiteral(BoolVar(id), false), j))
	}
	// Missing level mask rejects first antecedent without touching invalid second.
	got, e := s.impliedByMarked(4)
	satOK(t, e)
	if got {
		t.Fatal(got)
	}
	satOK(t, s.extAntecedents.Push(2))
	_, e = s.impliedByMarked(6)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
		n, e := s.extAntecedents.Size()
		satOK(t, e)
		if n != 1 {
			t.Fatal("assertion before reset")
		}
	} else {
		if !errors.Is(e, ErrSATMissingSourceExtBinding) {
			t.Fatal(e)
		}
		n, e := s.extAntecedents.Size()
		satOK(t, e)
		if n != 0 {
			t.Fatal("source reset before actual missing call")
		}
	}
	p.extensionPresent = true
	_, e = s.impliedByMarked(6)
	if !errors.Is(e, ErrSATMissingSourceExtBinding) {
		t.Fatal(e)
	}
	satLemmaAudit(t, s)
}
func TestSATSolverLemmaMinimizationWarmPhysicalPlateau(t *testing.T) {
	s, p, v, b := satLemmaFixture(t, 3)
	for id := uint32(0); id < 3; id++ {
		v.scopeLevel = 1
		j := satJustification{}
		if id == 2 {
			j = newSATBinaryJustification(2)
		}
		satOK(t, p.assignCore(MakeLiteral(BoolVar(id), false), j))
	}
	v.mark[1] = true
	s.levelSet.insert(1)
	got, e := s.impliedByMarked(4)
	satOK(t, e)
	if !got {
		t.Fatal(got)
	}
	before := b.Used()
	physical := v.memory.Stats()
	b.FailAfter(0)
	allocations := testing.AllocsPerRun(30, func() {
		got, e := s.impliedByMarked(4)
		satOK(t, e)
		if !got {
			t.Fatal(got)
		}
		satOK(t, s.resetUnmark(0))
	})
	if allocations != 0 || b.Used() != before || v.memory.Stats() != physical {
		t.Fatal("actual physical plateau", allocations, b.Used(), before)
	}
	satLemmaAudit(t, s)
	b.FailAfter(-1)
}

func TestSATSolverLemmaMinimizationClauseDebugSiteAndStaleBorrow(t *testing.T) {
	s, p, v, _ := satLemmaFixture(t, 4)
	c, e := p.clauses.MkClause([]Literal{2, 4, 0}, false)
	satOK(t, e)
	o, e := p.clauses.Offset(c)
	satOK(t, e)
	for id := uint32(0); id < 4; id++ {
		v.scopeLevel = 1
		j := satJustification{}
		if id == 3 {
			j = newSATClauseJustification(o)
		}
		satOK(t, p.assignCore(MakeLiteral(BoolVar(id), false), j))
	}
	got, e := s.impliedByMarked(6)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal("c[1] assertion before antecedent0", e)
		}
	} else {
		satOK(t, e)
		if got {
			t.Fatal("source release first antecedent missing level", got)
		}
	}
	if v.mark[1] || v.mark[2] {
		t.Fatal("no mark before source assertion/reject")
	}
	n, e := s.minStack.Size()
	satOK(t, e)
	if n != 0 {
		t.Fatal("stack popped before actual reason read", n)
	}
	satOK(t, p.clauses.Delete(c))
	_, e = s.impliedByMarked(6)
	if !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("real stale clause guard at dereference", e)
	}
	satLemmaAudit(t, s)
}
func TestSATSolverLemmaMinimizationEmptyAndFirstUIPDebugRoutes(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s, _, _, _ := satLemmaFixture(t, 1)
		e := s.minimizeLemma()
		if satStorageSourceDebug {
			if !errors.Is(e, ErrSATAssertion) || s.minimizedLits != 0 {
				t.Fatal(e, s.minimizedLits)
			}
		} else {
			if !errors.Is(e, arena.ErrBounds) || s.minimizedLits != 0 {
				t.Fatal("empty lemma bounds guard before counters", e, s.minimizedLits)
			}
		}
		n, e := s.lemma.Size()
		satOK(t, e)
		if n != 0 {
			t.Fatal("empty lemma unchanged", n)
		}
		e = s.resetLemmaVarMarks()
		if satStorageSourceDebug {
			if !errors.Is(e, ErrSATAssertion) {
				t.Fatal("Go guard on source undefined empty iterator dereference", e)
			}
		} else {
			satOK(t, e)
		}
		satLemmaAudit(t, s)
	})
	t.Run("first", func(t *testing.T) {
		s, _, v, _ := satLemmaFixture(t, 2)
		satOK(t, s.lemma.Push(0))
		satOK(t, s.lemma.Push(2))
		v.mark[0] = true
		v.mark[1] = true
		e := s.resetLemmaVarMarks()
		if satStorageSourceDebug {
			if !errors.Is(e, ErrSATAssertion) || !v.mark[1] {
				t.Fatal("firstUIP assertion before suffix reset", e)
			}
		} else {
			satOK(t, e)
			if v.mark[1] {
				t.Fatal("release actual suffix reset")
			}
		}
		if !v.mark[0] {
			t.Fatal("does not reset firstUIP")
		}
		satLemmaAudit(t, s)
	})
}
