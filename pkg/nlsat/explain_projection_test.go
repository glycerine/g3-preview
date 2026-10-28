package nlsat

import "testing"

func TestExplanationProjectionBuildsLowerCell(t *testing.T) {
	e, r, n, todo := projectionCoefficientFixture(t, true)
	evaluatorSetFraction(t, e, 0, 0, 1)
	evaluatorSetFraction(t, e, 1, 1, 1)

	// x1^2 + x0 - 2 has discriminant 8 - 4*x0. Projection first
	// selects that lower-stage polynomial and then constructs the cell x0 < 2
	// containing the current x0 = 0 interpretation.
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 2}, evaluatorTerm{1, 1, 0}, evaluatorTerm{-2, 0, 0})
	ps := projectionOutput(t, e)
	assignmentMust(t, ps.Push(p))
	assignmentMust(t, n.collectProjection(ps, 1, false))

	values := projectionValues(t, ps)
	if len(values) != 1 {
		t.Fatal("projection did not finish at one lower-variable stage", values)
	}
	max, err := e.core.MaxVariable(values[0])
	assignmentMust(t, err)
	if max != 0 {
		t.Fatal("projection did not descend to x0", max)
	}
	empty, err := todo.empty()
	assignmentMust(t, err)
	if !empty {
		t.Fatal("completed projection retained worklist entries")
	}
	if len(n.output.result) != 1 {
		t.Fatal("projection cell boundary", n.output.result)
	}
	literal := n.output.result[0]
	if !literal.Sign() {
		t.Fatal("projection explanation must negate the selected boundary", literal)
	}
	root, ok := r.store.atoms.bools.atoms[literal.Variable()].(*RootAtom)
	if !ok || root.kind != AtomRootLT || root.Variable() != 0 || root.index != 1 || root.polynomial != values[0] {
		t.Fatal("wrong projected cell root", root)
	}
	value, err := e.Eval(root, literal.Sign())
	assignmentMust(t, err)
	if value {
		t.Fatal("projected explanation is true in the current interpretation")
	}
}

func TestExplanationProjectionEmptyInputPreservesWorklist(t *testing.T) {
	e, _, n, todo := projectionCoefficientFixture(t, false)
	queued := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0}, evaluatorTerm{1, 0, 0})
	assignmentMust(t, todo.insert(queued))
	ps := projectionOutput(t, e)
	assignmentMust(t, n.collectProjection(ps, 1, false))
	values := projectionValues(t, todo.set)
	if len(values) != 1 || values[0] != queued {
		t.Fatal("empty source projection changed its existing worklist", values)
	}
}
