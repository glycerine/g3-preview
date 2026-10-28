package nlsat

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func explanationMinimizerFixture(t *testing.T) (*Evaluator, *searchState, *explanationMinimizer) {
	t.Helper()
	e, r := searchStateFixture(t)
	m, err := newExplanationMinimizer(r)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, m.close()) })
	return e, r, m
}
func explanationThreePointCore(t *testing.T, e *Evaluator, r *searchState) ([]sat.Literal, []*Clause) {
	t.Helper()
	a, ca := searchInequality(t, e, r, AtomEQ, evaluatorTerm{1, 0, 2}, evaluatorTerm{-1, 0, 0})
	b, cb := searchInequality(t, e, r, AtomEQ, evaluatorTerm{1, 0, 1}, evaluatorTerm{1, 0, 0})
	c, cc := searchInequality(t, e, r, AtomEQ, evaluatorTerm{1, 0, 1}, evaluatorTerm{-1, 0, 0})
	return []sat.Literal{a, b.Negated(), c.Negated()}, []*Clause{ca, cb, cc}
}
func TestExplanationMinimizeOrderOwningAppendAndRegistryGrowth(t *testing.T) {
	e, r, m := explanationMinimizerFixture(t)
	// Grow actual atom storage after the component has borrowed its owner.
	for i := 0; i < 70; i++ {
		_, err := r.store.atoms.mkBoolVar()
		assignmentMust(t, err)
	}
	lits, clauses := explanationThreePointCore(t, e, r)
	want := []sat.Literal{lits[2], lits[0], lits[1]}
	out, err := m.minimize(lits)
	assignmentMust(t, err)
	if !reflect.DeepEqual(out, want) {
		t.Fatal("source conflict selection/reverse order", out, want)
	}
	for _, l := range lits {
		if h := arithmeticAtomHeader(r.store.atoms.bools.atoms[l.Variable()]); h.refs != 2 {
			t.Fatal("result did not acquire one atom lease", l, h.refs)
		}
	}
	out, err = m.minimize(lits)
	assignmentMust(t, err)
	if !reflect.DeepEqual(out, append(append([]sat.Literal(nil), want...), want...)) {
		t.Fatal("source append replaced or deduplicated older output", out)
	}
	// Clauses can disappear after the result takes its own atom references.
	for _, c := range clauses {
		assignmentMust(t, r.store.deleteClause(c))
	}
	for _, l := range lits {
		if h := arithmeticAtomHeader(r.store.atoms.bools.atoms[l.Variable()]); h.refs != 2 {
			t.Fatal("duplicate result occurrence leases", h.refs)
		}
	}
	assignmentMust(t, m.resetResult())
	for _, l := range lits {
		if !r.store.atoms.bools.dead[l.Variable()] {
			t.Fatal("last result DEC did not retire actual atom", l)
		}
	}
	if m.resetting || m.resetIndex != 0 || len(m.result) != 0 {
		t.Fatal("reset progress not completed")
	}
}
func TestExplanationMinimizeRedundantAndConstantBranches(t *testing.T) {
	for _, name := range []string{"redundant-tail", "singleton", "already-full"} {
		t.Run(name, func(t *testing.T) {
			e, r, m := explanationMinimizerFixture(t)
			if name == "singleton" || name == "already-full" {
				l, _ := searchInequality(t, e, r, AtomLT, evaluatorTerm{1, 0, 2}, evaluatorTerm{1, 0, 0})
				if name == "singleton" {
					// Source singleton appends without evaluating even this atom.
					r.config.Limit.SetCancel(1)
					out, err := m.minimize([]sat.Literal{l})
					r.config.Limit.SetCancel(0)
					assignmentMust(t, err)
					if len(out) != 1 || out[0] != l {
						t.Fatal("singleton result", out)
					}
				} else {
					other, _ := searchInequality(t, e, r, AtomGT, evaluatorTerm{1, 0, 1})
					m.core = []sat.Literal{l}
					m.todo = []sat.Literal{other}
					again, err := m.minimizeCore()
					assignmentMust(t, err)
					if again || !reflect.DeepEqual(m.core, []sat.Literal{l}) || len(m.todo) != 1 {
						t.Fatal("existing full core should finish before singleton", again, m.core)
					}
				}
				return
			}
			a, _ := searchInequality(t, e, r, AtomGT, evaluatorTerm{1, 0, 1})
			b, _ := searchInequality(t, e, r, AtomLT, evaluatorTerm{1, 0, 1})
			redundant, _ := searchInequality(t, e, r, AtomLT, evaluatorTerm{1, 0, 1}, evaluatorTerm{-4, 0, 0})
			out, err := m.minimize([]sat.Literal{a, b, redundant})
			assignmentMust(t, err)
			if !reflect.DeepEqual(out, []sat.Literal{b, a}) {
				t.Fatal("redundant tail was retained", out)
			}
		})
	}
}

func TestExplanationMinimizeRootAtomsAndParameterizedDomains(t *testing.T) {
	t.Run("root-complements", func(t *testing.T) {
		e, r, m := explanationMinimizerFixture(t)
		p := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 2}, evaluatorTerm{-2, 0, 0})
		b, err := r.store.atoms.mkRoot(AtomRootGT, 1, 2, p)
		assignmentMust(t, err)
		l := sat.MakeLiteral(b, false)
		searchClause(t, r, l)
		out, err := m.minimize([]sat.Literal{l, l.Negated()})
		assignmentMust(t, err)
		if !reflect.DeepEqual(out, []sat.Literal{l.Negated(), l}) || arithmeticAtomHeader(r.store.atoms.bools.atoms[b]).refs != 3 {
			t.Fatal("root complement core or repeated-variable leases", out)
		}
		assignmentMust(t, m.resetResult())
		if arithmeticAtomHeader(r.store.atoms.bools.atoms[b]).refs != 1 {
			t.Fatal("root result multiplicity not retired")
		}
	})
	for _, y := range []int64{-2, 0, 2} {
		t.Run(fmt.Sprint(y), func(t *testing.T) {
			e, r, m := explanationMinimizerFixture(t)
			evaluatorSetFraction(t, e, 0, y, 1)
			lower, _ := searchInequality(t, e, r, AtomGT, evaluatorTerm{1, 0, 1}, evaluatorTerm{-1, 1, 0})
			upper, _ := searchInequality(t, e, r, AtomLT, evaluatorTerm{1, 0, 1}, evaluatorTerm{-1, 1, 0})
			out, err := m.minimize([]sat.Literal{lower, upper})
			assignmentMust(t, err)
			if !reflect.DeepEqual(out, []sat.Literal{upper, lower}) {
				t.Fatal("current lower-variable interpretation was ignored", y, out)
			}
			value, err := r.assignment.Value(0)
			assignmentMust(t, err)
			assignmentWantInteger(t, e.am, value, int32(y))
		})
	}
}
func TestExplanationMinimizeSourceEmptyTodoGuard(t *testing.T) {
	atomDebugFixture(t)
	e, r, m := explanationMinimizerFixture(t)
	// This redundant order reaches the source's empty-todo assertion on the
	// next pass. Preserve that source guard instead of silently repairing it.
	a, _ := searchInequality(t, e, r, AtomGT, evaluatorTerm{1, 0, 1}, evaluatorTerm{2, 0, 0})
	b, _ := searchInequality(t, e, r, AtomGT, evaluatorTerm{1, 0, 1})
	c, _ := searchInequality(t, e, r, AtomLT, evaluatorTerm{1, 0, 1})
	out, err := m.minimize([]sat.Literal{a, b, c})
	if diagnostic.DebugBuild {
		if !errors.Is(err, diagnostic.ErrDebugAssertion) || len(m.todo) != 0 || len(m.core) != 2 {
			t.Fatal("source empty todo prefix", out, err, m.todo, m.core)
		}
		assignmentMust(t, m.cleanup())
	} else {
		assignmentMust(t, err)
		if !reflect.DeepEqual(out, []sat.Literal{c, b}) {
			t.Fatal("source release full-core termination", out)
		}
	}
}
func TestExplanationMinimizeAllocationStopsOwnIntervals(t *testing.T) {
	successes, stops := 0, 0
	for stop := int64(0); stop < 512; stop++ {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			e, r, m := explanationMinimizerFixture(t)
			lits, _ := explanationThreePointCore(t, e, r)
			beforeLive := len(r.ism.live)
			e.budget.FailAfter(stop)
			out, err := m.minimize(lits)
			e.budget.FailAfter(-1)
			if err == nil {
				successes++
				if len(out) != 3 {
					t.Fatal("successful reduced core", out)
				}
				return
			}
			stops++
			if err2 := r.newLevel(); !errors.Is(err2, ErrSearchPending) {
				t.Fatal("search accepted pending explanation operation", err2)
			}
			searchAcknowledgeNumericalCallees(t, e)
			assignmentMust(t, m.cleanup())
			if len(r.ism.live) != beforeLive || len(m.retired) != 0 || m.refs[0].value != nil || m.refs[1].value != nil || m.union.owner != nil {
				t.Fatal("minimizer retained interrupted interval scope", stop, len(r.ism.live), beforeLive)
			}
			out, err = m.minimize(lits)
			assignmentMust(t, err)
			if len(out) != 3 {
				t.Fatal("retry core", out)
			}
		})
		if successes > 0 {
			break
		}
	}
	if successes != 1 || stops < 2 {
		t.Fatal("did not reach interval construction stops through success", stops, successes)
	}
	t.Log("consecutive failed reservations", stops)
}
func TestExplanationMinimizeInfeasibleCoresPrebuiltCPPOracle(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	var script strings.Builder
	script.WriteString("(set-logic QF_NRA)\n(declare-const x Real)\n")
	queries := 0
	for shift := int32(-4); shift <= 4; shift++ {
		e, r, m := explanationMinimizerFixture(t)
		// Two distinct roots and both disequalities form a necessary three-way
		// conflict. A redundant tail must not be copied to the result.
		a := shift
		b := shift + 2
		poly, _ := searchInequality(t, e, r, AtomEQ, evaluatorTerm{1, 0, 2}, evaluatorTerm{-(a + b), 0, 1}, evaluatorTerm{a * b, 0, 0})
		left, _ := searchInequality(t, e, r, AtomEQ, evaluatorTerm{1, 0, 1}, evaluatorTerm{-a, 0, 0})
		right, _ := searchInequality(t, e, r, AtomEQ, evaluatorTerm{1, 0, 1}, evaluatorTerm{-b, 0, 0})
		tail, _ := searchInequality(t, e, r, AtomLT, evaluatorTerm{1, 0, 1}, evaluatorTerm{-(b + 5), 0, 0})
		lits := []sat.Literal{poly, left.Negated(), right.Negated(), tail}
		forms := map[sat.BoolVar]string{
			poly.Variable():  fmt.Sprintf("(= (* (- x %s) (- x %s)) 0.0)", intervalSMTInteger(a), intervalSMTInteger(b)),
			left.Variable():  fmt.Sprintf("(= x %s)", intervalSMTInteger(a)),
			right.Variable(): fmt.Sprintf("(= x %s)", intervalSMTInteger(b)),
			tail.Variable():  fmt.Sprintf("(< x %s)", intervalSMTInteger(b+5)),
		}
		out, err := m.minimize(lits)
		assignmentMust(t, err)
		if len(out) != 3 {
			t.Fatal("redundant constraint retained", shift, out)
		}
		for omit := -1; omit < len(out); omit++ {
			var parts []string
			for i, l := range out {
				if i == omit {
					continue
				}
				s := forms[l.Variable()]
				if l.Sign() {
					s = "(not " + s + ")"
				}
				parts = append(parts, s)
			}
			fmt.Fprintf(&script, "(push)(assert (and %s))(check-sat-using qfnra-nlsat)(pop)\n", strings.Join(parts, " "))
			queries++
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, reference, "-in", "-smt2")
	cmd.Stdin = strings.NewReader(script.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("C++ core oracle %v: %s", err, out)
	}
	answers := strings.Fields(string(out))
	if len(answers) != queries {
		t.Fatalf("C++ answers: %d want %d: %s", len(answers), queries, out)
	}
	for i, answer := range answers {
		want := "sat"
		if i%4 == 0 {
			want = "unsat"
		}
		if answer != want {
			t.Fatal("core unsatisfiability/minimality", i, answer, want)
		}
	}
}

func TestExplanationMinimizeConstructorAndParentClose(t *testing.T) {
	_, r := searchStateFixture(t)
	b := r.store.atoms.ineq.budget
	before := b.Used()
	b.FailAfter(0)
	m, err := newExplanationMinimizer(r)
	b.FailAfter(-1)
	if !errors.Is(err, arena.ErrInjectedFailure) || m != nil || r.minimizer != nil || b.Used() != before {
		t.Fatal("failed constructor published a borrower", m, err)
	}
	m, err = newExplanationMinimizer(r)
	assignmentMust(t, err)
	if other, err := newExplanationMinimizer(r); !errors.Is(err, arena.ErrConfig) || other != nil {
		t.Fatal("duplicate component", other, err)
	}
	// Search owns the actual subobject and closes it before dropping its own
	// borrowed-parent registrations. A separate explanation destructor is safe.
	assignmentMust(t, r.close())
	if !m.closed || r.minimizer != nil {
		t.Fatal("parent close abandoned explanation owner")
	}
	assignmentMust(t, m.close())
	if _, err := m.minimize(nil); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed component reused", err)
	}
}

func TestExplanationMinimizeConsumedResultDECContinuation(t *testing.T) {
	atomDebugFixture(t)
	e, r, m := explanationMinimizerFixture(t)
	lits, clauses := explanationThreePointCore(t, e, r)
	out, err := m.minimize(lits)
	assignmentMust(t, err)
	first := out[0].Variable()
	a := r.store.atoms.bools.atoms[first]
	for _, c := range clauses {
		assignmentMust(t, r.store.deleteClause(c))
	}
	// Qualified invalid Boolean-watch state, reaching the same source deletion
	// guard as the atom/clause owner tests. Do not repair its reference count.
	r.store.atoms.bools.watches[first] = []*Clause{clauses[0]}
	err = m.resetResult()
	want := ErrAtomCacheLive
	if diagnostic.DebugBuild {
		want = diagnostic.ErrDebugAssertion
	}
	if !errors.Is(err, want) || m.resetIndex != 1 || m.retiringAtom != a || arithmeticAtomHeader(a).refs != 0 || !m.resetting {
		t.Fatal("consumed result DEC lost exact continuation", err, m.resetIndex)
	}
	if err := r.newLevel(); !errors.Is(err, ErrSearchPending) {
		t.Fatal("search progressed during result destruction", err)
	}
	r.store.atoms.bools.watches[first] = nil
	assignmentMust(t, m.resetResult())
	for _, l := range lits {
		if !r.store.atoms.bools.dead[l.Variable()] {
			t.Fatal("result atom survived continued destruction", l)
		}
	}
}

func TestExplanationMinimizeCleanupPreservesOlderMarks(t *testing.T) {
	e, r, m := explanationMinimizerFixture(t)
	_, _, _ = searchReasonDomain(t, e, r)
	var older []sat.Literal
	r.ism.budget.FailAfter(1)
	err := r.ism.GetJustifications(r.store.vars.infeasible[1], &older)
	r.ism.budget.FailAfter(-1)
	if !errors.Is(err, arena.ErrInjectedFailure) || r.ism.justifications == nil || len(older) != 1 {
		t.Fatal("older mark scope not qualified", err)
	}
	scope := r.ism.justifications
	visited := append([]bool(nil), r.ism.visited...)
	used, stats := r.ism.budget.Used(), r.ism.allocator.RuntimeOwner().Stats()
	if _, err := m.minimize([]sat.Literal{TrueLiteral}); err == nil {
		t.Fatal("minimization accepted older marked work")
	}
	assignmentMust(t, m.cleanup())
	if r.ism.justifications != scope || !reflect.DeepEqual(visited, r.ism.visited) || r.ism.budget.Used() != used || r.ism.allocator.RuntimeOwner().Stats() != stats {
		t.Fatal("component cleanup adopted older independent mark scope")
	}
	assignmentMust(t, r.ism.Cleanup())
}
