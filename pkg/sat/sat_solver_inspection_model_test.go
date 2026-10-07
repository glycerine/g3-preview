// Source: src/sat/sat_solver.cpp:723-735,1289-1346,2852-3017.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2011-05-21.
package sat

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverInspectionDIMACSActualCheckBindingAndAssumptions(t *testing.T) {
	s, c, p := satSolverCheckFixture(t)
	for i := 0; i < 2; i++ {
		_, e := s.MkVar(false, true)
		satOK(t, e)
	}
	satOK(t, s.MkClause([]Literal{0, 2}))
	k := "dimacs.display"
	satOK(t, c.SetBoolName(&p, &k, true))
	satOK(t, s.UpdateParams(&p))
	knownBefore, decisionsBefore := s.Variables.modelKnown, s.decision.decisions
	got, e := s.Check([]Literal{3})
	if got != util.LUndef || !errors.Is(e, ErrSATMissingDimacsBinding) {
		t.Fatal(got, e)
	}
	o := satInspectionStream(t, arena.NewBudget(math.MaxUint64))
	satOK(t, s.bindDIMACS(o))
	got, e = s.Check([]Literal{3})
	satOK(t, e)
	if got != util.LUndef {
		t.Fatal(got)
	}
	satInspectionText(t, o, "p cnf 2 1\n1 2 0\n-2 0\n")
	if s.decision.decisions != decisionsBefore || s.Variables.modelKnown != knownBefore {
		t.Fatal("display unexpectedly searched")
	}
	satFullSolverRuntimeAudit(t, s)
}
func TestSATSolverInspectionIndependentDIMACSFiniteModels(t *testing.T) {
	for mask := uint32(0); mask < 32; mask++ {
		s, _, _ := satSolverCheckFixture(t)
		for i := 0; i < 3; i++ {
			_, e := s.MkVar(false, true)
			satOK(t, e)
		}
		input := [][]Literal{}
		for j := uint32(0); j < 5; j++ {
			if mask&(1<<j) == 0 {
				continue
			}
			c := []Literal{Literal(j & 1), Literal(2 + (j >> 1 & 1))}
			if j == 4 {
				c = []Literal{1, 3, 4}
			}
			input = append(input, c)
			_, e := s.Lifecycle.MkClauseCore(uint32(len(c)), c, false)
			satOK(t, e)
		}
		o := satInspectionStream(t, arena.NewBudget(math.MaxUint64))
		satOK(t, s.displayDIMACS(o))
		text, e := o.String()
		satOK(t, e)
		lines := strings.Split(strings.TrimSpace(text), "\n")
		head := strings.Fields(lines[0])
		if head[0] != "p" || head[1] != "cnf" || head[2] != "3" {
			t.Fatal(text)
		}
		n, e := strconv.Atoi(head[3])
		satOK(t, e)
		if n != len(lines)-1 {
			t.Fatal("actual source count", text)
		}
		for bits := uint32(0); bits < 8; bits++ {
			want := true
			for _, c := range input {
				yes := false
				for _, l := range c {
					yes = yes || ((bits&(1<<l.variable()) != 0) != l.sign())
				}
				want = want && yes
			}
			got := true
			for _, line := range lines[1:] {
				yes := false
				words := strings.Fields(line)
				for _, word := range words[:len(words)-1] {
					v, e := strconv.Atoi(word)
					satOK(t, e)
					sign := v < 0
					if sign {
						v = -v
					}
					yes = yes || ((bits&(1<<uint(v-1)) != 0) != sign)
				}
				got = got && yes
			}
			if want != got {
				t.Fatal(mask, bits, text)
			}
		}
		satFullSolverRuntimeAudit(t, s)
	}
}
func TestSATSolverInspectionBaseUnitsVsAllTrailAndMarkProfiles(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 4; i++ {
		_, e := s.MkVar(false, true)
		satOK(t, e)
	}
	satOK(t, s.propagation.assign(3, satJustification{}))
	_, e := s.propagation.propagateCore(false)
	satOK(t, e)
	satOK(t, s.Lifecycle.push())
	satOK(t, s.propagation.assign(4, satJustification{}))
	o := satInspectionStream(t, arena.NewBudget(math.MaxUint64))
	satOK(t, s.displayUnits(o))
	satInspectionText(t, o, "-1 \n")
	satOK(t, o.ResetText())
	satOK(t, s.displayAssignment(o))
	satInspectionText(t, o, "-1 2\n")
	satOK(t, s.conflict.analysis.mark(0))
	e = s.checkMarks()
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else {
		satOK(t, e)
	}
	satOK(t, s.conflict.analysis.resetMark(0))
}
func TestSATSolverInspectionSoftReadAndClauseJustificationPrefixes(t *testing.T) {
	s := satInspectionFixture(t)
	o := satInspectionStream(t, arena.NewBudget(math.MaxUint64))
	e := s.displayWCNF(o, 1, nil, []uint32{2})
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	satInspectionText(t, o, "p wcnf 4 4 3\nc soft 1\n3 -2 0\n3 1 2 0\n3 -1 3 4 0\n2 ")
	satOK(t, o.ResetText())
	e = s.displayWCNF(o, 1, []Literal{0}, nil)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	satInspectionText(t, o, "")
	satOK(t, o.ResetText())
	off, e := s.Clauses.Offset(s.Lifecycle.Original[0])
	satOK(t, e)
	satOK(t, s.displayJustification(o, newSATClauseJustification(off)))
	satInspectionText(t, o, "clause(-0 2 3)+")
}

func TestSATSolverInspectionBinaryCollectionAppendsOriginalAndLearned(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 2; i++ {
		_, e := s.MkVar(false, true)
		satOK(t, e)
	}
	_, e := s.Lifecycle.MkClauseCore(2, []Literal{0, 2}, false)
	satOK(t, e)
	_, e = s.Lifecycle.MkClauseCore(2, []Literal{1, 3}, true)
	satOK(t, e)
	out, e := NewBinClauseBuffer(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.Close()) })
	satOK(t, out.Push(newSATSolverBinPair(2, 2)))
	satOK(t, s.collectBinClauses(out, false))
	n, e := out.Size()
	satOK(t, e)
	if n != 2 {
		t.Fatal(n)
	}
	satOK(t, s.collectBinClauses(out, true))
	n, e = out.Size()
	satOK(t, e)
	if n != 4 {
		t.Fatal(n)
	}
	satOK(t, out.Close())
	satFullSolverRuntimeAudit(t, s)
}

func TestSATSolverInspectionExactSourceWatchDisplay(t *testing.T) {
	s := satInspectionFixture(t)
	o := satInspectionStream(t, arena.NewBudget(math.MaxUint64))
	satOK(t, s.displayWatches(o))
	satInspectionText(t, o, "0: (2 3)\n-0: 1\n1: \n-1: 0\n2: \n-2: (-0 3)\n3: \n-3: (-0 2)\n")
}
