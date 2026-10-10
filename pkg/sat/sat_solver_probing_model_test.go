// Source: src/sat/sat_probing.cpp; src/sat/sat_probing.h.
package sat

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func satProbingAudit(t *testing.T, s *satSolverProbing, analysis *satSolverConflictAnalysis, d *satSolverDynamicSubsumption, external ...util.Block) {
	t.Helper()
	p := s.propagation()
	v := p.variables
	l := s.lifecycle
	roots := append([]util.Block(nil), external...)
	satScalarAuditRoots(t, v, roots...)
	// Exact named physical graph, once-only Runtime and all genuine cold backing.
	want := s.ownerBytes

	if analysis != nil {
		want += analysis.ownerBytes
	}
	if d != nil {
		want += d.ownerBytes
	}
	want += l.ownerBytes + uint64(cap(l.Original)+cap(l.learned))*uint64(reflect.TypeOf(StoredClauseRef{}).Size()) + uint64(cap(l.reinit))*uint64(reflect.TypeOf(satClauseWrapper{}).Size()) + uint64(cap(l.scopes))*uint64(reflect.TypeOf(satSolverScope{}).Size())
	want += p.ownerBytes + uint64(cap(p.diffLevels.data)) + v.Stats().TotalBytes
	todo := v.elimTodo
	want += todo.ownerBytes
	a := p.clauses
	small := a.storage.Stats()
	want += a.ownerBytes + 4*uint64(a.ids.FreeCapacity()) + a.largeBytes + a.largeDirectory + small.OwnerBytes + small.PayloadBytes + small.MetadataBytes + small.DirectoryBytes
	mc := p.converter
	want += mc.ownerBytes
	r := p.limit
	want += uint64(reflect.TypeOf(control.Reslimit{}).Size()) + 8*uint64(cap(r.Limits)) + uint64(reflect.TypeOf((*control.Reslimit)(nil)).Size())*uint64(cap(r.Children))
	want += p.memory.Stats().TotalBytes
	if want != s.budget.Used() {
		t.Fatal("named once-only physical graph", want, s.budget.Used())
	}
	if small.Live != uint32(len(l.Original)+len(l.learned))+func() uint32 {
		if l.pendingCandidate != (StoredClauseRef{}) {
			return 1
		}
		return 0
	}() {
		t.Fatal("unnamed clause owner")
	}
}
func satProbingExpectedRefusal(t *testing.T, p *satSolverPropagationState, e error) {
	t.Helper()
	if errors.Is(e, arena.ErrInjectedFailure) {
		return
	}
	var failure util.Failure
	if errors.As(e, &failure) && failure.ErrorCode() == 101 && p.memory.State().OutOfMemory {
		return
	}
	t.Fatal("unexpected refusal", e)
}
func TestSATSolverProbingIndependentFiniteImplications(t *testing.T) {
	for seed := 0; seed < 32; seed++ {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			s, life := satProbingFixture(t, 4)
			// Deterministic source binary-clause input; independent complete truth table.
			clauses := [6][2]Literal{}
			for i := range clauses {
				a := uint32((seed*3 + i*5) % 8)
				b := uint32((seed*7 + i*3 + 2) % 8)
				if a>>1 == b>>1 {
					b = (b + 2) % 8
				}
				clauses[i] = [2]Literal{Literal(a), Literal(b)}
				satOK(t, life.mkBinClause(Literal(a), Literal(b), false))
			}
			models := []uint32{}
			truth := func(mask uint32, l Literal) bool {
				return ((mask>>uint32(l.variable()))&1 != 0) != (l.index()&1 != 0)
			}
			for mask := uint32(0); mask < 16; mask++ {
				ok := true
				for _, c := range clauses {
					ok = ok && (truth(mask, c[0]) || truth(mask, c[1]))
				}
				if ok {
					models = append(models, mask)
				}
			}
			_, e := s.run(true)
			satOK(t, e)
			p := life.propagation
			if p.inconsistent && len(models) > 0 {
				t.Fatal("unsound conflict", models)
			}
			for id := uint32(0); id < 4; id++ {
				value, e := p.variables.valueVar(BoolVar(id))
				satOK(t, e)
				if value == lbool.Undef {
					continue
				}
				for _, mask := range models {
					if (mask>>id&1 != 0) != (value == lbool.True) {
						t.Fatal("unsound deduction", id, value, models)
					}
				}
			}
			satProbingAudit(t, s, nil, nil)
		})
	}
}
func TestSATSolverProbingAllPhysicalRefusalPrefixes(t *testing.T) {
	failed, success := 0, 0
	for fault := int64(0); fault < 256; fault++ {
		done := false
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			s, life := satProbingFixture(t, 4)
			for _, c := range [][2]Literal{{0, 2}, {1, 2}, {3, 4}, {5, 6}} {
				satOK(t, life.mkBinClause(c[0], c[1], false))
			}
			s.budget.FailAfter(fault)
			finished, e := s.run(true)
			satProbingAudit(t, s, nil, nil) // before disabling refusal, any formatting or teardown.
			if e != nil {
				failed++
				satProbingExpectedRefusal(t, life.propagation, e)
			} else {
				success++
				done = true
				if !finished {
					t.Fatal("quota unexpectedly exhausted")
				}
				value, e := life.propagation.variables.valueVar(1)
				satOK(t, e)
				if value != lbool.True {
					t.Fatal(value)
				}
			}
			s.budget.FailAfter(-1)
		})
		if done {
			break
		}
	}
	if failed == 0 || success != 1 {
		t.Fatal(failed, success)
	}
	t.Log("real refusal prefixes", failed, "first success", success)
}
func TestSATSolverProbingNativeCacheAndStrictMemoryCutoff(t *testing.T) {
	s, life := satProbingFixture(t, 3)
	p := life.propagation
	satOK(t, p.assignCore(2, satJustification{}))
	satOK(t, p.assignCore(4, satJustification{}))
	a, e := p.memory.AllocationSize()
	satOK(t, e)
	s.cacheLimit = a // equality must permit cache allocation.
	satOK(t, s.cacheBins(0, 0))
	borrow, yes, e := s.cachedImpliedLits(0)
	satOK(t, e)
	if !yes {
		t.Fatal("strict > guard")
	}
	block, n, e := borrow.capture()
	satOK(t, e)
	if n != 2 {
		t.Fatal(n)
	}
	for i, w := range []Literal{2, 4} {
		x, e := borrow.readCaptured(block, uint32(i))
		satOK(t, e)
		if x != w {
			t.Fatal(x)
		}
	}
	if !s.cache.rows[0].available {
		t.Fatal("native availability flag")
	}
	satProbingAudit(t, s, nil, nil)
	prefix, e := p.memory.Allocate(1)
	satOK(t, e)
	s.cacheLimit = 0
	satOK(t, s.cacheBins(2, 0))
	_, yes, e = s.cachedImpliedLits(2)
	satOK(t, e)
	if yes {
		t.Fatal("cutoff should not publish")
	}
	satOK(t, p.memory.Deallocate(&prefix))
	satOK(t, s.resetCache(0))
	got, e := borrow.readCaptured(block, 0)
	satOK(t, e)
	if got != 2 {
		t.Fatal("captured native backing changed on cache reset", got)
	}
	satProbingAudit(t, s, nil, nil)
}
func TestSATSolverProbingInterruptedRoundRetainsSourceCache(t *testing.T) {
	s, life := satProbingFixture(t, 2)
	s.binaryEnabled = false
	// One process consumes two assumption checkpoints plus the outer checkpoint.
	// Abort a subsequent outer process only after actual previous cached entries.
	satOK(t, s.propagation().limit.Push(3))
	_, e := s.run(true)
	if !errors.Is(e, ErrSATSolverCanceled) {
		t.Fatal(e)
	}
	satProbingAudit(t, s, nil, nil)
	if !s.cache.blockLive() {
		t.Fatal("exception does not execute normal finalize")
	}
	if life.propagation.variables.scopeLevel != 0 {
		t.Fatal("outer checkpoint interruption")
	}
	s.propagation().limit.Pop()
}

func TestSATSolverProbingEveryActualCancellationPrefix(t *testing.T) {
	failures, success := 0, 0
	inside := false
	for quota := uint32(1); quota < 32; quota++ {
		done := false
		t.Run(fmt.Sprint(quota), func(t *testing.T) {
			s, life := satProbingFixture(t, 3)
			p := life.propagation
			satOK(t, p.limit.Push(quota))
			_, e := s.run(true)
			satProbingAudit(t, s, nil, nil)
			if e != nil {
				if !errors.Is(e, ErrSATSolverCanceled) {
					t.Fatal(e)
				}
				failures++
				inside = inside || p.variables.scopeLevel != 0
				if p.limit.Count() != uint64(quota)+1 {
					t.Fatal("real first over-limit source inc", quota, p.limit.Count())
				}
			} else {
				success++
				done = true
			}
			p.limit.Pop()
		})
		if done {
			break
		}
	}
	if failures == 0 || success != 1 || !inside {
		t.Fatal(failures, success, inside)
	}
	t.Log("actual canceled prefixes", failures, "success", success)
}
func TestSATSolverProbingNativeAvailableAndAppendWithoutReservations(t *testing.T) {
	s, life := satProbingFixture(t, 2)
	satOK(t, s.cacheBins(0, 0)) // actual empty entry, no inner payload.
	satOK(t, life.propagation.assignCore(2, satJustification{}))
	s.budget.FailAfter(0)
	before := s.budget.Used()
	e := s.cacheBins(0, 0)
	satOK(t, e)
	if s.budget.Used() != before {
		t.Fatal("native cache used arena", s.budget.Used(), before)
	}
	borrow, found, readErr := s.cachedImpliedLits(0)
	satOK(t, readErr)
	if !found {
		t.Fatal("available published before append")
	}
	n, readErr := borrow.size()
	satOK(t, readErr)
	if n != 1 {
		t.Fatal("native cached assignment missing", n)
	}
	values, _, readErr := borrow.capture()
	satOK(t, readErr)
	lit, readErr := borrow.readCaptured(values, 0)
	satOK(t, readErr)
	if lit != 2 {
		t.Fatal("actual native cached literal", lit)
	}
	satProbingAudit(t, s, nil, nil)
	s.budget.FailAfter(-1)
	satOK(t, s.resetCache(0))
	satProbingAudit(t, s, nil, nil)
}
