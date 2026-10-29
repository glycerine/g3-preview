package nlsat

import (
	"testing"

	"github.com/glycerine/g3/pkg/math/polynomial"
)

func TestExplanationSimplifyCoreWithEqualityPseudoRemainder(t *testing.T) {
	e, r, n := explanationAssumptionFixture(t)
	evaluatorSetFraction(t, e, 0, 1, 1)
	equality := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0}, evaluatorTerm{-1, 0, 0})
	consequence := evaluatorPolynomial(t, e, evaluatorTerm{1, 2, 0}, evaluatorTerm{-1, 0, 0})
	eq := normalizationLiteral(t, r, AtomEQ, []polynomial.Polynomial{equality}, []bool{false})
	gt := normalizationLiteral(t, r, AtomGT, []polynomial.Polynomial{consequence}, []bool{false})
	core := scopedLiteralVector{factory: r.store.atoms}
	assignmentMust(t, core.push(eq))
	assignmentMust(t, core.push(gt))
	t.Cleanup(func() { assignmentMust(t, core.close()) })
	assignmentMust(t, n.start())
	modified, err := n.simplifyCoreWithEquality(&core, equality, 0)
	assignmentMust(t, err)
	assignmentMust(t, n.finish(nil))
	if !modified || len(core.result) != 0 || len(n.output.result) != 0 {
		t.Fatal("equality pseudo-remainder did not expose false literal", modified, core.result, n.output.result)
	}
}

func TestExplanationSelectsMinimumDegreeCoreEquality(t *testing.T) {
	e, r, n := explanationAssumptionFixture(t)
	quadratic := evaluatorPolynomial(t, e, evaluatorTerm{1, 2, 0}, evaluatorTerm{-1, 0, 0})
	linear := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0}, evaluatorTerm{-1, 0, 0})
	q := normalizationLiteral(t, r, AtomEQ, []polynomial.Polynomial{quadratic}, []bool{false})
	l := normalizationLiteral(t, r, AtomEQ, []polynomial.Polynomial{linear}, []bool{false})
	core := scopedLiteralVector{factory: r.store.atoms}
	assignmentMust(t, core.push(q))
	assignmentMust(t, core.push(l))
	t.Cleanup(func() { assignmentMust(t, core.close()) })
	selected, err := n.selectCoreEquality(&core, 0)
	assignmentMust(t, err)
	equal, err := e.core.Equal(selected, linear)
	assignmentMust(t, err)
	if !equal {
		t.Fatal("did not select minimum-degree equality")
	}
}

func TestExplanationSimplifiesWithCachedLowerStageEquality(t *testing.T) {
	e, r, n := explanationAssumptionFixture(t)
	evaluatorSetFraction(t, e, 0, 1, 1)
	evaluatorSetFraction(t, e, 1, 0, 1)
	equality := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0}, evaluatorTerm{-1, 0, 0})
	mixed := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0}, evaluatorTerm{1, 0, 1}, evaluatorTerm{-2, 0, 0})
	eq := normalizationLiteral(t, r, AtomEQ, []polynomial.Polynomial{equality}, []bool{false})
	conflict := normalizationLiteral(t, r, AtomGT, []polynomial.Polynomial{mixed}, []bool{false})
	r.store.vars.equalities = make([]ArithmeticAtom, 2)
	r.store.vars.equalities[0] = r.store.atoms.bools.atoms[eq.Variable()]
	core := scopedLiteralVector{factory: r.store.atoms}
	assignmentMust(t, core.push(conflict))
	t.Cleanup(func() { assignmentMust(t, core.close()) })
	selected, err := n.selectLowerStageEquality(&core, 1)
	assignmentMust(t, err)
	if selected == nil {
		t.Fatal("cached lower-stage equality was not selected")
	}
	assignmentMust(t, n.start())
	assignmentMust(t, n.simplifyCoreEqualities(&core, 1))
	assignmentMust(t, n.finish(nil))
	if len(core.result) != 1 || len(n.output.result) != 1 || n.output.result[0] != eq.Negated() {
		t.Fatal("cached lower-stage equality rewrite", core.result, n.output.result)
	}
	rewritten := r.store.atoms.bools.atoms[core.result[0].Variable()].(*InequalityAtom)
	yDegree, err := e.core.Degree(rewritten.factors[0].Polynomial, 0)
	assignmentMust(t, err)
	xDegree, err := e.core.Degree(rewritten.factors[0].Polynomial, 1)
	assignmentMust(t, err)
	if yDegree != 0 || xDegree != 1 {
		t.Fatal("lower-stage variable was not eliminated", yDegree, xDegree)
	}
}
