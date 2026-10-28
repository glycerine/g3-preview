package nlsat

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/messages"
)

func searchAcknowledgeNumericalCallees(t *testing.T, e *Evaluator) {
	t.Helper()
	assignmentMust(t, e.am.CleanupRetained())
	if e.failed {
		assignmentMust(t, e.Cleanup())
	}
}
func TestSearchValuePrecedenceAndResourceFailures(t *testing.T) {
	e, r := searchStateFixture(t)
	l, c := searchInequality(t, e, r, AtomGT, evaluatorTerm{1, 0, 1}, evaluatorTerm{-1, 1, 0})
	r.config.Limit.SetCancel(1)
	before := r.config.Limit.Count()
	v, err := r.value(l)
	assignmentMust(t, err)
	if v != lbool.Undef || r.config.Limit.Count() != before {
		t.Fatal("unassigned maximum reached numerical evaluation")
	}
	r.config.Limit.SetCancel(0)
	evaluatorSetFraction(t, e, 0, 2, 1)
	evaluatorSetFraction(t, e, 1, -1, 1)
	v, err = r.value(l)
	assignmentMust(t, err)
	if v != lbool.False {
		t.Fatal("actual algebraic assignment value")
	}
	j, err := NewClauseJustification(c)
	assignmentMust(t, err)
	assignmentMust(t, r.assign(l, j))
	v, err = r.value(l)
	assignmentMust(t, err)
	if v != lbool.True {
		t.Fatal("explicit Boolean assignment did not precede arithmetic evaluation")
	}
	assignmentMust(t, r.undoUntilUnassigned(l.Variable()))
	v, err = r.value(l)
	assignmentMust(t, err)
	if v != lbool.False {
		t.Fatal("numerical value not restored after Boolean undo")
	}
	// Actual cancellation takes priority over an independently violated memory cap.
	r.config.MaxMemory = 0
	r.config.Limit.SetCancel(1)
	err = r.checkpoint()
	if err == nil || err.Error() != messages.Canceled {
		t.Fatal("resource cancellation ordering", err)
	}
	r.config.Limit.SetCancel(0)
	err = r.checkpoint()
	if err == nil || err.Error() != messages.MaxMemory {
		t.Fatal("actual synchronized memory size was not checked", err)
	}
	r.config.MaxMemory = uint64(^uint32(0))
	assignmentMust(t, r.config.Limit.Push(1))
	t.Cleanup(func() { r.config.Limit.Pop() })
	assignmentMust(t, r.checkpoint())
	err = r.checkpoint()
	if err == nil || err.Error() != messages.MaxResource {
		t.Fatal("actual shared resource counter was bypassed", err)
	}
}

func TestSearchFailedUnionKeepsRecordedTrailAndCurrentOwnership(t *testing.T) {
	success := false
	for stop := int64(0); stop < 64; stop++ {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			e, r := searchStateFixture(t)
			searchAtRealStage(t, r)
			_, c := searchInequality(t, e, r, AtomLT, evaluatorTerm{1, 0, 1})
			ok, err := r.processClause(c, false)
			assignmentMust(t, err)
			if !ok {
				t.Fatal("initial domain")
			}
			old := r.store.vars.infeasible[1]
			l, c2 := searchInequality(t, e, r, AtomLT, evaluatorTerm{1, 0, 1}, evaluatorTerm{1, 0, 0})
			j, err := NewClauseJustification(c2)
			assignmentMust(t, err)
			assignmentMust(t, r.assign(l, j))
			set, err := e.InfeasibleIntervals(r.store.atoms.bools.atoms[l.Variable()], false)
			assignmentMust(t, err)
			base := uint32(len(r.trail))
			r.store.atoms.ineq.budget.FailAfter(stop)
			err = r.updateInfeasible(set)
			r.store.atoms.ineq.budget.FailAfter(-1)
			if err != nil {
				var physical umemory.Failure
				if !errors.Is(err, arena.ErrInjectedFailure) && !(errors.As(err, &physical) && physical.Kind == umemory.FailureMemout) {
					t.Fatal("unexpected union refusal", err)
				}
				if len(r.trail) != int(base+1) || r.trail[base].transferred || r.store.vars.infeasible[1] != old || old.refs != 1 {
					t.Fatal("failed union changed the current reference or erased source trail publication")
				}
			}
			searchAcknowledgeNumericalCallees(t, e)
			assignmentMust(t, r.cleanup())
			assignmentMust(t, r.undoUntilSize(base))
			if old.closed || old.refs != 1 || r.store.vars.infeasible[1] != old {
				t.Fatal("undo of an untransferred/aliased entry decremented unchanged current ownership")
			}
			assignmentMust(t, r.updateInfeasible(set))
			assignmentMust(t, r.undoUntilSize(base))
			assignmentMust(t, r.ism.DecRef(set))
			success = err == nil
		})
		if success {
			break
		}
	}
	if !success {
		t.Fatal("union reservation campaign never reached success")
	}
}

func searchReasonDomain(t *testing.T, e *Evaluator, r *searchState) (sat.Literal, *Clause, []sat.Literal) {
	t.Helper()
	searchAtRealStage(t, r)
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1})
	b, err := r.store.atoms.mkInequality(AtomLT, []polynomial.Polynomial{p}, []bool{false})
	assignmentMust(t, err)
	left := sat.MakeLiteral(b, true)
	leftClause := searchClause(t, r, left)
	ok, err := r.processClause(leftClause, false)
	assignmentMust(t, err)
	if !ok {
		t.Fatal("lower reason domain")
	}
	for i := 0; i < 20; i++ {
		_, err := r.store.atoms.mkBoolVar()
		assignmentMust(t, err)
	}
	right, c := searchInequality(t, e, r, AtomLT, evaluatorTerm{1, 0, 1}, evaluatorTerm{-1, 0, 0})
	ok, err = r.processClause(c, false)
	assignmentMust(t, err)
	if !ok {
		t.Fatal("upper reason domain")
	}
	target, c := searchInequality(t, e, r, AtomLT, evaluatorTerm{1, 0, 2}, evaluatorTerm{-4, 0, 0})
	return target, c, []sat.Literal{left, right}
}
func TestSearchJustificationMarksAreOwnedByAcceptedCall(t *testing.T) {
	_, r := searchStateFixture(t)
	e := r.evaluator
	_, c, _ := searchReasonDomain(t, e, r)
	set := r.store.vars.infeasible[1]
	var older []sat.Literal
	r.store.atoms.ineq.budget.FailAfter(1)
	err := r.ism.GetJustifications(set, &older)
	r.store.atoms.ineq.budget.FailAfter(-1)
	if !errors.Is(err, arena.ErrInjectedFailure) || r.ism.justifications == nil || len(older) != 1 || len(r.ism.justifications.marked) != 1 {
		t.Fatal("older justification call did not qualify its marked prefix", err, older)
	}
	scope := r.ism.justifications
	visited := append([]bool(nil), r.ism.visited...)
	stats, used := r.ism.allocator.RuntimeOwner().Stats(), r.ism.budget.Used()
	if _, err := r.processClause(c, false); err == nil {
		t.Fatal("new operation accepted another caller's marked scratch")
	}
	assignmentMust(t, r.cleanup())
	if r.ism.justifications != scope || !reflect.DeepEqual(r.ism.visited, visited) || r.ism.allocator.RuntimeOwner().Stats() != stats || r.ism.budget.Used() != used {
		t.Fatal("search cleanup adopted independently older justification work")
	}
	assignmentMust(t, r.ism.Cleanup())
	ok, err := r.processClause(c, false)
	assignmentMust(t, err)
	if !ok {
		t.Fatal("reason propagation after owning mark cleanup")
	}
}

func TestSearchStaleMarkReceiptPreservesNewerCollection(t *testing.T) {
	e, r := searchStateFixture(t)
	_, _, _ = searchReasonDomain(t, e, r)
	set := r.store.vars.infeasible[1]
	r.ism.budget.FailAfter(1)
	err := r.ism.getJustificationsWithReceipt(set, &r.core, &r.justificationScope)
	r.ism.budget.FailAfter(-1)
	if !errors.Is(err, arena.ErrInjectedFailure) || r.justificationScope == nil || len(r.core) != 1 {
		t.Fatal("own collection did not accept a marked prefix", err)
	}
	old := r.justificationScope
	assignmentMust(t, r.ism.Cleanup()) // explicit actual owner acknowledges this invocation.
	var newer []sat.Literal
	r.ism.budget.FailAfter(0)
	err = r.ism.GetJustifications(set, &newer)
	r.ism.budget.FailAfter(-1)
	if !errors.Is(err, arena.ErrInjectedFailure) || r.ism.justifications == nil || r.ism.justifications == old || len(newer) != 1 {
		t.Fatal("newer collection did not qualify a distinct scope", err)
	}
	scope := r.ism.justifications
	visited := append([]bool(nil), r.ism.visited...)
	stats, used := r.ism.allocator.RuntimeOwner().Stats(), r.ism.budget.Used()
	assignmentMust(t, r.cleanup())
	if r.ism.justifications != scope || !reflect.DeepEqual(r.ism.visited, visited) || r.ism.allocator.RuntimeOwner().Stats() != stats || r.ism.budget.Used() != used || len(r.core) != 1 {
		t.Fatal("stale receipt adopted newer scratch or reset caller output")
	}
	assignmentMust(t, r.ism.Cleanup())
}

func TestSearchPropagationRefusalsOwnPartialScopes(t *testing.T) {
	qualified := false
	success := false
	partialLazy := false
	for stop := int64(0); stop < 64; stop++ {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			e, r := searchStateFixture(t)
			l, c, reasons := searchReasonDomain(t, e, r)
			base := uint32(len(r.trail))
			r.store.atoms.ineq.budget.FailAfter(stop)
			_, err := r.processClause(c, false)
			r.store.atoms.ineq.budget.FailAfter(-1)
			success = err == nil
			if r.pendingLazy.IsLazy() && !r.pendingLazy.lazy.ready {
				partialLazy = true
			}
			partial := r.justificationScope != nil && r.ism.justifications == r.justificationScope && len(r.justificationScope.marked) > 0
			searchAcknowledgeNumericalCallees(t, e)
			assignmentMust(t, r.cleanup())
			assignmentMust(t, r.undoUntilSize(base))
			if partial {
				qualified = true
				if err == nil || len(r.core) != 1 {
					t.Fatal("marked-prefix refusal did not preserve caller output", err, r.core)
				}
				for _, flag := range r.ism.visited {
					if flag {
						t.Fatal("own partial marks survived cleanup")
					}
				}
			}
			ok, retry := r.processClause(c, false)
			assignmentMust(t, retry)
			if !ok {
				t.Fatal("propagation did not recover")
			}
			j := r.store.atoms.bools.justifications[l.Variable()]
			why, err := j.lazy.Literals()
			assignmentMust(t, err)
			if len(why) != 3 || why[len(why)-1] != l.Negated() {
				t.Fatal("recovered reason lost an antecedent or its propagated-literal negation", why, reasons)
			}
		})
		if qualified && success {
			break
		}
	}
	if !qualified || !success || !partialLazy {
		t.Fatal("did not qualify visited-mark and lazy-constructor refusals through success", qualified, partialLazy, success)
	}
}

func TestSearchBorrowedLifetimeAndFullDomainAssertion(t *testing.T) {
	atomDebugFixture(t)
	e, r := searchStateFixture(t)
	for _, closeOwner := range []func() error{r.store.close, e.Close, r.assignment.Close, r.ism.Close} {
		if err := closeOwner(); err != ErrSearchLive {
			t.Fatal("borrowed parent progressed into destruction", err)
		}
	}
	searchAtRealStage(t, r)
	_, c := searchInequality(t, e, r, AtomLT, evaluatorTerm{1, 0, 1})
	ok, err := r.processClause(c, false)
	assignmentMust(t, err)
	if !ok {
		t.Fatal("base domain")
	}
	old := r.store.vars.infeasible[1]
	l, _ := searchInequality(t, e, r, AtomGT, evaluatorTerm{1, 0, 1})
	set, err := e.InfeasibleIntervals(r.store.atoms.bools.atoms[l.Variable()], false)
	assignmentMust(t, err)
	base := uint32(len(r.trail))
	err = r.updateInfeasible(set) // qualified invalid full-domain update reaches the source assertion.
	if diagnostic.DebugBuild {
		if !errors.Is(err, diagnostic.ErrDebugAssertion) || r.store.vars.infeasible[1] != old || r.trail[base].transferred {
			t.Fatal("full-domain assertion lost its reached source prefix", err)
		}
	} else {
		assignmentMust(t, err)
	}
	assignmentMust(t, r.cleanup())
	assignmentMust(t, r.undoUntilSize(base))
	if old.closed || old.refs != 1 || r.store.vars.infeasible[1] != old {
		t.Fatal("full-domain guard cleanup consumed the unchanged prior set")
	}
	assignmentMust(t, r.ism.DecRef(set))
	if r.store.atoms.bools.values[l.Variable()] != lbool.Undef {
		t.Fatal("guarded update invented a Boolean assignment")
	}
}
