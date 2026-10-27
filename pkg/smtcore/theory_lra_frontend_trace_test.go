package smtcore

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

type lraFrontendTraceSnapshot struct {
	marker                             ast.ASTHandle
	bounds, trail, assignments, scopes int
	unassigned                         uint32
	disequalities, axioms              uint64
}
type lraFrontendTraceWriter struct {
	owner     *SMTLRATheory
	output    bytes.Buffer
	observed  bytes.Buffer
	refuse    bool
	snapshots map[int]lraFrontendTraceSnapshot
}

func (w *lraFrontendTraceWriter) Write(p []byte) (int, error) {
	w.observed.Write(p)
	for _, line := range []int{300, 595, 705, 714, 781, 787, 806, 856, 864} {
		if !strings.Contains(w.observed.String(), fmt.Sprintf("src/smt/theory_lra.cpp:%d ", line)) {
			continue
		}
		if _, exists := w.snapshots[line]; exists {
			continue
		}
		a := w.owner
		snapshot := lraFrontendTraceSnapshot{marker: a.notHandled, bounds: len(a.boundByBoolean), trail: len(a.boundTrail), assignments: len(a.assignments), scopes: len(a.scopes), disequalities: uint64(a.statistics.disequalities), axioms: a.equalityAdapter.NumAxioms()}
		if len(a.unassigned) > 0 {
			snapshot.unassigned = a.unassigned[0]
		}
		w.snapshots[line] = snapshot
	}
	if w.refuse {
		return 0, io.ErrClosedPipe
	}
	return w.output.Write(p)
}
func lraFrontendTrace(f *lraFrontendFixture, refuse bool) (*lraFrontendTraceWriter, *diagnostic.TraceTags) {
	w := &lraFrontendTraceWriter{owner: f.a, refuse: refuse, snapshots: make(map[int]lraFrontendTraceSnapshot)}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("arith")
	f.a.trace = tags
	return w, tags
}
func checkLRAFrontendOutput(t *testing.T, w *lraFrontendTraceWriter, tags *diagnostic.TraceTags, want string) {
	t.Helper()
	if !tags.ShouldTrace("arith") || w.refuse {
		want = ""
	}
	if w.output.String() != want {
		t.Fatalf("trace:\n%s\nwant:\n%s", w.output.String(), want)
	}
	if !tags.ShouldTrace("arith") && len(w.snapshots) != 0 {
		t.Fatal("native trace operands evaluated")
	}
}
func TestSMTLRAFrontendTraceRefusalOrderingAndBadbit(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		f := newLRAFrontendFixture(t, true)
		x, y := f.variable("trace-raw-x"), f.variable("trace-raw-y")
		atom := f.app(smtArithOpLE, x, y)
		w, tags := lraFrontendTrace(f, refuse)
		accepted, err := f.a.InternalizeAtom(atom, false)
		astTestMust(t, err)
		if !accepted || f.a.notHandled != atom || len(f.a.boundByBoolean) != 0 {
			t.Fatal("refusal bookkeeping")
		}
		body := "(<= trace-raw-x trace-raw-y)\n"
		want := lraObjectiveTraceRecord("arith", "internalize_atom_strict", 705, "Could not internalize "+body) + lraObjectiveTraceRecord("arith", "found_not_handled", 300, "Unhandled: "+body)
		checkLRAFrontendOutput(t, w, tags, want)
		if tags.ShouldTrace("arith") && !refuse {
			if !w.snapshots[705].marker.IsNull() || w.snapshots[300].marker != atom {
				t.Fatal("refusal trace moved across marker update", w.snapshots)
			}
		}
		// A nonlinear term reaches found_not_handled without the atom-refusal site.
		w, tags = lraFrontendTrace(f, refuse)
		product := f.app(smtArithOpMul, x, x)
		_, err = f.a.InternalizeTerm(product)
		astTestMust(t, err)
		if f.a.notHandled != product {
			t.Fatal("term refusal marker")
		}
		checkLRAFrontendOutput(t, w, tags, lraObjectiveTraceRecord("arith", "found_not_handled", 300, "Unhandled: (* trace-raw-x trace-raw-x)\n"))
	}
}
func TestSMTLRAFrontendTraceRegistrationAssignmentAndScope(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		f := newLRAFrontendFixture(t, true)
		x := f.variable("trace-bound-x")
		atom := f.app(smtArithOpGE, x, f.number(0))
		w, tags := lraFrontendTrace(f, refuse)
		accepted, err := f.a.InternalizeAtom(atom, false)
		astTestMust(t, err)
		if !accepted || len(f.a.boundTrail) != 1 || f.a.unassigned[0] != 1 {
			t.Fatal("bound registration")
		}
		want := lraObjectiveTraceRecord("arith", "updt_unassigned_bounds", 595, "v0 0 += 1\n") + lraObjectiveTraceRecord("arith", "internalize_atom_strict", 714, "Internalized (>= trace-bound-x 0.0)\n")
		checkLRAFrontendOutput(t, w, tags, want)
		if tags.ShouldTrace("arith") && !refuse {
			before, after := w.snapshots[595], w.snapshots[714]
			if before.unassigned != 0 || before.bounds != 0 || before.trail != 0 || after.unassigned != 1 || after.bounds != 1 || after.trail != 1 {
				t.Fatal("registration trace order", before, after)
			}
		}
		bv, ok := f.s.BoolVar(atom)
		if !ok {
			t.Fatal("missing bool")
		}
		for _, truth := range []bool{false, true} {
			w, tags = lraFrontendTrace(f, refuse)
			count := len(f.a.assignments)
			astTestMust(t, f.a.Assign(bv, truth))
			checkLRAFrontendOutput(t, w, tags, lraObjectiveTraceRecord("arith", "assign_eh", 787, fmt.Sprintf("(>= trace-bound-x 0.0) %t\n", truth)))
			if len(f.a.assignments) != count+1 || f.a.assignments[count].truth != truth {
				t.Fatal("assignment lost")
			}
			if tags.ShouldTrace("arith") && !refuse && w.snapshots[787].assignments != count {
				t.Fatal("assignment trace after enqueue")
			}
		}
		astTestMust(t, f.a.PushScope())
		astTestMust(t, f.a.PushScope())
		w, tags = lraFrontendTrace(f, refuse)
		astTestMust(t, f.a.PopScopes(2))
		checkLRAFrontendOutput(t, w, tags, lraObjectiveTraceRecord("arith", "pop_scope_eh", 856, "num scopes: 2 new scope level: 0\n"))
		if len(f.a.scopes) != 0 || len(f.a.assignments) != 2 || f.a.unassigned[0] != 1 {
			t.Fatal("scope restoration")
		}
		if tags.ShouldTrace("arith") && !refuse && w.snapshots[856].scopes != 0 {
			t.Fatal("pop trace before restoration")
		}
		w, tags = lraFrontendTrace(f, refuse)
		astTestMust(t, f.a.PopScopes(0))
		checkLRAFrontendOutput(t, w, tags, "")
	}
}
func TestSMTLRAFrontendTraceEqualityDisequalityAndRelevancy(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		f := newLRAFrontendFixture(t, true)
		x, y := f.variable("trace-eq-x"), f.variable("trace-eq-y")
		xv, err := f.a.internalizeDef(x)
		astTestMust(t, err)
		yv, err := f.a.internalizeDef(y)
		astTestMust(t, err)
		eq := f.pin(f.s.mkEqualityAtom(x, y))
		w, tags := lraFrontendTrace(f, refuse)
		astTestMust(t, f.s.Internalize(eq, true))
		if f.a.equalityAdapter.NumAxioms() != 1 {
			t.Fatal("equality adapter not called")
		}
		if tags.ShouldTrace("arith") && !refuse {
			record := lraObjectiveTraceRecord("arith", "internalize_eq_eh", 781, "(= trace-eq-x trace-eq-y)\n")
			if !strings.HasPrefix(w.output.String(), record) || w.snapshots[781].axioms != 0 {
				t.Fatal("equality trace after adapter", w.output.String(), w.snapshots)
			}
		} else {
			checkLRAFrontendOutput(t, w, tags, "")
		}
		w, tags = lraFrontendTrace(f, refuse)
		before := f.a.statistics.disequalities
		astTestMust(t, f.a.NewDisequality(yv, xv))
		checkLRAFrontendOutput(t, w, tags, lraObjectiveTraceRecord("arith", "new_diseq_eh", 806, fmt.Sprintf("v%d != v%d\n", yv, xv)))
		if f.a.statistics.disequalities != before+1 || f.a.equalityAdapter.NumAxioms() != 1 {
			t.Fatal("disequality cache/count")
		}
		if tags.ShouldTrace("arith") && !refuse && w.snapshots[806].disequalities != uint64(before) {
			t.Fatal("disequality trace after increment")
		}
		// Ordinary terms also emit relevant_eh before the operator dispatch returns.
		w, tags = lraFrontendTrace(f, refuse)
		astTestMust(t, f.a.Relevant(x))
		checkLRAFrontendOutput(t, w, tags, lraObjectiveTraceRecord("arith", "relevant_eh", 864, "trace-eq-x\n"))
	}
}
func TestSMTLRAFrontendTraceUnassignedDecrementAndScopeRestore(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		f := newLRAFrontendFixture(t, true)
		x := f.variable("trace-counter-x")
		// The nearest weaker bound is propagated by the source basic-bound path.
		weaker := f.app(smtArithOpGE, x, f.number(1))
		stronger := f.app(smtArithOpGE, x, f.number(2))
		astTestMust(t, f.s.Internalize(weaker, true))
		astTestMust(t, f.s.Internalize(stronger, true))
		astTestMust(t, f.a.Propagate())
		astTestMust(t, f.s.PushScope())
		boolean, _ := f.s.BoolVar(stronger)
		bound := f.a.boundByBoolean[boolean]
		before := f.a.unassigned[bound.variable]
		w, tags := lraFrontendTrace(f, refuse)
		astTestMust(t, f.a.propagateBasicBound(bound, true))
		if f.a.unassigned[bound.variable] != before-1 || f.a.basicPropagations != 1 {
			t.Fatal("basic propagation count")
		}
		checkLRAFrontendOutput(t, w, tags, lraObjectiveTraceRecord("arith", "propagate_bound", 1797, "(>= trace-counter-x 2.0) => (>= trace-counter-x 1.0)\n")+lraObjectiveTraceRecord("arith", "updt_unassigned_bounds", 595, fmt.Sprintf("v%d %d += -1\n", bound.variable, before)))
		if tags.ShouldTrace("arith") && !refuse && w.snapshots[595].unassigned != before {
			t.Fatal("counter trace after decrement")
		}
		f.a.trace = nil
		astTestMust(t, f.s.PopScopes(1))
		if f.a.unassigned[bound.variable] != before {
			t.Fatal("decrement not restored")
		}
	}
}

func TestSMTLRAFrontendTraceLPBoundCounterAndBadbit(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		f := newLRAFrontendFixture(t, true)
		x, y := f.variable("trace-row-x"), f.variable("trace-row-y")
		target := f.app(smtArithOpGE, x, f.number(3))
		astTestMust(t, f.s.Internalize(target, true))
		astTestMust(t, f.a.Propagate())
		sum := f.app(smtArithOpAdd, x, y)
		f.assert(f.app(smtArithOpGE, sum, f.number(5)), true)
		f.assert(f.app(smtArithOpLE, y, f.number(2)), true)
		bv, _ := f.s.BoolVar(target)
		v := f.a.boundByBoolean[bv].variable
		before := f.a.unassigned[v]
		w, tags := lraFrontendTrace(f, refuse)
		f.propagate(true)
		assignment, err := f.s.Assignment(NewSMTLiteral(bv, false))
		astTestMust(t, err)
		if assignment != lbool.True || f.a.lpPropagations == 0 || f.a.unassigned[v] != before-1 || f.a.numeric.stopped {
			t.Fatal("LP bound propagation damaged", assignment, f.a.lpPropagations)
		}
		record := lraObjectiveTraceRecord("arith", "updt_unassigned_bounds", 595, fmt.Sprintf("v%d %d += -1\n", v, before))
		if tags.ShouldTrace("arith") && !refuse {
			if !strings.Contains(w.output.String(), record) {
				t.Fatal("LP decrement trace missing", w.output.String())
			}
		} else {
			checkLRAFrontendOutput(t, w, tags, "")
		}
	}
}
