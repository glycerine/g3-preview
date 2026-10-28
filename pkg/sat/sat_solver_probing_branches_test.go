package sat

// Source: src/sat/sat_probing.cpp; src/sat/sat_probing.h.

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATSolverProbingCachedTryLitAndFailedBranchStatistic(t *testing.T) {
	t.Run("cached", func(t *testing.T) {
		s, life := satProbingFixture(t, 2)
		p := life.propagation
		satOK(t, p.assignCore(2, satJustification{}))
		satOK(t, s.cacheBins(0, 0))
		satOK(t, life.unassignVars(0))
		satOK(t, s.assigned.Insert(2))
		p.probingCounter = 17
		keep, e := s.tryLit(0, false)
		satOK(t, e)
		value, e := p.variables.valueVar(1)
		satOK(t, e)
		if !keep || value != lbool.True || s.numAssigned != 1 || p.probingCounter != 17 {
			t.Fatal(keep, value, s.numAssigned, p.probingCounter)
		}
		satProbingAudit(t, s, nil, nil)
	})
	t.Run("failedMiss", func(t *testing.T) {
		s, life := satProbingFixture(t, 2)
		satOK(t, life.mkBinClause(1, 2, false))
		satOK(t, life.mkBinClause(1, 3, false))
		keep, e := s.tryLit(0, true)
		satOK(t, e)
		if keep || s.numAssigned != 0 || life.propagation.variables.scopeLevel != 0 {
			t.Fatal(keep, s.numAssigned)
		}
		value, e := life.propagation.variables.valueVar(0)
		satOK(t, e)
		if value != lbool.False {
			t.Fatal(value)
		}
		satProbingAudit(t, s, nil, nil)
	})
}
func TestSATSolverProbingWarmProcessAndUnsignedDec(t *testing.T) {
	s, _ := satProbingFixture(t, 1)
	satOK(t, s.process(0))
	before := s.budget.Used()
	state := s.propagation().memory.Stats()
	s.budget.FailAfter(0)
	allocs := testing.AllocsPerRun(40, func() {
		if e := s.process(0); e != nil {
			t.Fatal(e)
		}
	})
	if allocs != 0 || s.budget.Used() != before || s.propagation().memory.Stats() != state {
		t.Fatal("warm actual storage", allocs, before, s.budget.Used())
	}
	satProbingAudit(t, s, nil, nil)
	s.budget.FailAfter(-1)
	s.propagation().probingCounter = -1 << 31
	satOK(t, s.dec(1))
	if s.propagation().probingCounter != 1<<31-1 {
		t.Fatal("unsigned subtraction conversion")
	}
	s.propagation().probingCounter = -1 << 31
	if e := s.process(0); !errors.Is(e, ErrSATSignedCounterDomain) {
		t.Fatal(e)
	}
	if s.propagation().variables.scopeLevel != 0 {
		t.Fatal("guard at actual decrement")
	}
}
func TestSATSolverProbingDisabledAndPositiveCounterEntryOrder(t *testing.T) {
	s, life := satProbingFixture(t, 1)
	s.enabled = false
	life.propagation.limit.SetCancel(1)
	ok, e := s.run(true)
	satOK(t, e)
	if !ok || life.propagation.limit.Count() != 0 {
		t.Fatal("disabled no propagation")
	}
	s.enabled = true
	life.propagation.limit.ResetCancel()
	life.propagation.probingCounter = 7
	satOK(t, life.propagation.assignCore(0, satJustification{}))
	ok, e = s.run(false)
	satOK(t, e)
	if !ok || life.propagation.variables.qhead != 1 || life.propagation.probingCounter != 7 {
		t.Fatal("propagate before positive-counter return")
	}
}
func TestSATSolverProbingConstructorRefusalAndCohort(t *testing.T) {
	life, _, _, b := satLifecycleFixture(t, 0)
	ctx, _, _, _ := satMainConfigTestRuntime(t)
	p, e := ctx.NewValue()
	satOK(t, e)
	defer func() { satOK(t, ctx.Del(&p)) }()
	for fault := int64(0); fault < 1; fault++ {
		before := b.Used()
		b.FailAfter(fault)
		s, e := newSATSolverProbing(life, ctx, &p, b)
		if !errors.Is(e, arena.ErrInjectedFailure) || s != nil || b.Used() != before {
			t.Fatal(fault, e, b.Used(), before)
		}
		b.FailAfter(-1)
	}
	// Actual embedded source members allocate no separate cold heap headers.
	// The one successful endpoint reserves precisely the actual probing header.
	before := b.Used()
	b.FailAfter(1)
	constructed, e := newSATSolverProbing(life, ctx, &p, b)
	satOK(t, e)
	if b.Used() != before+uint64(reflect.TypeOf(satSolverProbing{}).Size()) {
		t.Fatal("actual once-charged cold header")
	}
	b.FailAfter(-1)
	satOK(t, constructed.Close())
	if b.Used() != before {
		t.Fatal("constructor endpoint cleanup")
	}
	foreign := arena.NewBudget(^uint64(0))
	s, e := newSATSolverProbing(life, ctx, &p, foreign)
	if !errors.Is(e, ErrSATOwner) || s != nil || foreign.Used() != 0 {
		t.Fatal(e)
	}
}
func TestSATSolverProbingConditionalDiagnosticsSourceSites(t *testing.T) {
	s, life := satProbingFixture(t, 0)
	debug, _, _ := satIntegrityActionFixture(t, "")
	before := life.propagation.limit.Count()
	_, e := s.run(true)
	satOK(t, e)
	if life.propagation.limit.Count() != before {
		t.Fatal("disabled probing tag reached checker")
	}
	satOK(t, debug.EnableDebug("probing"))
	_, e = s.run(true)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATMissingIntegrityBinding) || life.propagation.limit.Count() != before+1 {
			t.Fatal(e)
		}
		satOK(t, debug.EnableAssertions(false))
		_, e = s.run(true)
		satOK(t, e)
		if life.propagation.limit.Count() != before+1 {
			t.Fatal("disabled process assertions reached checker")
		}
	} else {
		satOK(t, e)
		if life.propagation.limit.Count() != before {
			t.Fatal("release ignores CASSERT")
		}
	}
}

func TestSATProbingConditionalInvariantChecksBothReachedSites(t *testing.T) {
	g, _ := satGCFixture(t, 0)
	s := g.conflict.dynamic.probing
	debug, _, _ := satIntegrityActionFixture(t, "")
	satOK(t, debug.EnableDebug("probing"))
	before := g.lifecycle().propagation.limit.Count()
	yes, err := s.run(true)
	satOK(t, err)
	if !yes {
		t.Fatal("empty probing round")
	}
	want := uint64(0)
	if satStorageSourceDebug {
		want = 2
	}
	if got := g.lifecycle().propagation.limit.Count() - before; got != want {
		t.Fatal("entry and exit CASSERT reachability", got, want)
	}
}

func TestSATSolverProbingMissedPropagationTruthTable(t *testing.T) {
	for pattern := 0; pattern < 27; pattern++ {
		t.Run(fmt.Sprint(pattern), func(t *testing.T) {
			s, life := satProbingFixture(t, 3)
			satIntegrityActionFixture(t, "T")
			c, e := life.propagation.clauses.MkClause([]Literal{0, 2, 4}, true)
			satOK(t, e)
			satOK(t, life.pushClause(true, c))
			code := pattern
			undef, trues := 0, 0
			for i := uint32(0); i < 3; i++ {
				state := code % 3
				code /= 3
				value := []lbool.LBool{lbool.False, lbool.Undef, lbool.True}[state]
				satOK(t, life.propagation.variables.setAssignment(Literal(2*i), value))
				if value == lbool.Undef {
					undef++
				}
				if value == lbool.True {
					trues++
				}
			}
			empty, e := s.clauseEmpty(c)
			satOK(t, e)
			unit, e := s.clauseUnit(c)
			satOK(t, e)
			if empty != (trues == 0 && undef == 0) || unit != (trues == 0 && undef == 1) {
				t.Fatal(pattern, empty, unit)
			}
			e = s.checkMissedPropagation()
			bad := empty || unit
			if bad {
				var ex z3exception.DefaultException
				if satStorageSourceDebug && !errors.As(e, &ex) || !satStorageSourceDebug && !errors.Is(e, ErrSATAssertion) {
					t.Fatal(pattern, e)
				}
			} else {
				satOK(t, e)
			}
			satOK(t, life.propagation.clauses.Freeze(c))
			satOK(t, s.checkMissedPropagation())
			life.propagation.inconsistent = true
			satOK(t, s.checkMissedPropagation())
		})
	}
}

func TestSATSolverProbingBinaryWatchFiltersAndNonbinaryStop(t *testing.T) {
	t.Run("index", func(t *testing.T) {
		s, life := satProbingFixture(t, 3)
		satOK(t, life.mkBinClause(4, 0, false))
		satOK(t, s.process(2))
		if s.propagation().probingCounter != -3 || s.numAssigned != 0 {
			t.Fatal("lower literal index skipped", s.propagation().probingCounter)
		}
	})
	t.Run("assigned", func(t *testing.T) {
		s, life := satProbingFixture(t, 3)
		satOK(t, life.propagation.assignCore(0, satJustification{}))
		satOK(t, s.propagate())
		satOK(t, life.mkBinClause(4, 0, false))
		satOK(t, s.process(2))
		if s.propagation().probingCounter != -3 || s.numAssigned != 0 {
			t.Fatal("assigned literal skipped", s.propagation().probingCounter)
		}
	})
	t.Run("nonbinary", func(t *testing.T) {
		s, life := satProbingFixture(t, 4)
		_, e := life.MkClauseCore(3, []Literal{0, 2, 4}, false)
		satOK(t, e)
		satOK(t, life.mkBinClause(0, 6, false))
		satOK(t, life.mkBinClause(0, 4, false))
		satOK(t, s.process(0))
		if s.propagation().probingCounter != -5 || s.numAssigned != 0 {
			t.Fatal("first nonbinary ends binary scan", s.propagation().probingCounter)
		}
	})
	t.Run("binary", func(t *testing.T) {
		s, life := satProbingFixture(t, 4)
		satOK(t, life.mkBinClause(0, 6, false))
		satOK(t, life.mkBinClause(0, 4, false))
		satOK(t, s.process(0))
		if s.propagation().probingCounter != -6 || s.numAssigned != 0 {
			t.Fatal("both binary candidates tried", s.propagation().probingCounter)
		}
	})
}

func TestSATSolverProbingStatisticsAndDescriptorNoOp(t *testing.T) {
	s, _ := satProbingFixture(t, 0)
	st := statFixture(t)
	key := statKey("probing assigned")
	s.numAssigned = 9
	satOK(t, s.collectStatistics(st, key))
	if st.Size() != 1 || st.Key(0) != key || st.UintValue(0) != 9 {
		t.Fatal("actual source stat contribution")
	}
	s.resetStatistics()
	satOK(t, s.collectStatistics(st, key))
	if st.Size() != 1 {
		t.Fatal("source zero stat is omitted")
	}
	d := pdnew(t, symbolManagerTest(t, nil, false), nil)
	before, e := d.Size()
	satOK(t, e)
	satProbingCollectParamDescriptions(d)
	after, e := d.Size()
	satOK(t, e)
	if before != after {
		t.Fatal("source TODO body is no-op")
	}
}
