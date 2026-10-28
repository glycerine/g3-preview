package nlsat

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/sat"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"
)

func normalizationLiteral(t *testing.T, r *searchState, k AtomKind, ps []polynomial.Polynomial, even []bool) sat.Literal {
	t.Helper()
	l, err := r.store.atoms.mkInequalityLiteral(k, ps, even)
	assignmentMust(t, err)
	normalizationOwnLiteral(t, r, l)
	return l
}
func normalizationOwnLiteral(t *testing.T, r *searchState, l sat.Literal) {
	t.Helper()
	assignmentMust(t, r.store.atoms.incRef(l.Variable()))
	t.Cleanup(func() { assignmentMust(t, r.store.atoms.decRef(l.Variable())) })
}
func TestExplanationNormalizeVanishingAndLowerStage(t *testing.T) {
	for _, kind := range []AtomKind{AtomEQ, AtomLT, AtomGT} {
		for _, negative := range []bool{false, true} {
			e, r, n := explanationAssumptionFixture(t)
			evaluatorSetFraction(t, e, 0, -1, 1)
			// (y+1)*x^3 + y*x + 1 reduces to y*x+1, with a vanishing assumption.
			p := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 3}, evaluatorTerm{1, 0, 3}, evaluatorTerm{1, 1, 1}, evaluatorTerm{1, 0, 0})
			lower := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0})
			l := normalizationLiteral(t, r, kind, []polynomial.Polynomial{p, lower}, []bool{false, false})
			if negative {
				l = l.Negated()
			}
			result, err := n.collectNormalize(l, 1)
			assignmentMust(t, err)
			normalizationOwnLiteral(t, r, result)
			a, ok := r.store.atoms.bools.atoms[result.Variable()].(*InequalityAtom)
			if !ok || len(a.factors) != 1 || a.factors[0].Even || result.Sign() != negative {
				t.Fatal("normalized factors/polarity", result, a)
			}
			want, err := kind.Flip()
			assignmentMust(t, err)
			if a.kind != want {
				t.Fatal("negative odd lower-stage factor did not flip relation", kind, a.kind)
			}
			degree, err := e.core.Degree(a.factors[0].Polynomial, 1)
			assignmentMust(t, err)
			if degree != 1 || len(n.output.result) != 2 || n.normalization.active || n.vanishing.active {
				t.Fatal("vanishing and lower-stage assumption composition", degree, n.output.result)
			}
		}
	}
}
func TestExplanationNormalizeEvenZeroConstantAndIdentity(t *testing.T) {
	for _, value := range []int64{-3, -2, -1, 0, 2} {
		for _, even := range []bool{false, true} {
			for _, kind := range []AtomKind{AtomEQ, AtomLT, AtomGT} {
				for _, negative := range []bool{false, true} {
					e, r, n := explanationAssumptionFixture(t)
					evaluatorSetFraction(t, e, 0, value, 1)
					y := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0})
					l := normalizationLiteral(t, r, kind, []polynomial.Polynomial{y}, []bool{even})
					if negative {
						l = l.Negated()
					}
					result, err := n.collectNormalize(l, 1)
					assignmentMust(t, err)
					truth := false
					switch kind {
					case AtomEQ:
						truth = value == 0
					case AtomLT:
						truth = value < 0 && !even
					case AtomGT:
						truth = value > 0 || (value < 0 && even)
					}
					if result != explanationTruthLiteral(truth != negative) || len(n.output.result) != 1 {
						t.Fatal("all lower-stage factors did not become source truth literal", value, even, kind, negative, result)
					}
				}
			}
		}
	}
	e, r, n := explanationAssumptionFixture(t)
	x := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1}, evaluatorTerm{2, 0, 0})
	l := normalizationLiteral(t, r, AtomGT, []polynomial.Polynomial{x}, []bool{false})
	result, err := n.collectNormalize(l, 1)
	assignmentMust(t, err)
	if result != l || len(n.output.result) != 0 {
		t.Fatal("unchanged factor lost canonical literal")
	}
	for _, l := range []sat.Literal{TrueLiteral, FalseLiteral} {
		result, err = n.collectNormalize(l, 1)
		assignmentMust(t, err)
		if result != l {
			t.Fatal("Boolean constant changed")
		}
	}
	root, err := r.store.atoms.mkRoot(AtomRootLT, 1, 1, x)
	assignmentMust(t, err)
	rootLiteral := sat.MakeLiteral(root, true)
	normalizationOwnLiteral(t, r, rootLiteral)
	result, err = n.collectNormalize(rootLiteral, 1)
	assignmentMust(t, err)
	if result != rootLiteral {
		t.Fatal("root atom normalized")
	}
}
func TestExplanationVanishingVectorStableCompaction(t *testing.T) {
	e, _, n := explanationAssumptionFixture(t)
	evaluatorSetFraction(t, e, 0, 0, 1)
	ps := projectionOutput(t, e)
	gone := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 2})
	keep := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1}, evaluatorTerm{1, 0, 0})
	reduced := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 3}, evaluatorTerm{2, 0, 1})
	for _, p := range []polynomial.Polynomial{gone, keep, reduced, gone, keep} {
		assignmentMust(t, ps.Push(p))
	}
	assignmentMust(t, n.collectVanishingVector(ps))
	values := projectionValues(t, ps)
	if len(values) != 3 || values[0] != keep || values[2] != keep {
		t.Fatal("stable compaction/multiplicity", values)
	}
	expected := evaluatorPolynomial(t, e, evaluatorTerm{2, 0, 1})
	equal, err := e.core.Equal(values[1], expected)
	assignmentMust(t, err)
	if !equal || len(n.output.result) != 1 || n.normalization.active {
		t.Fatal("actual reduced vector and canonical assumption", values, n.output.result)
	}
}
func TestExplanationNormalizeCoreCompactionAndFalseReset(t *testing.T) {
	e, r, n := explanationAssumptionFixture(t)
	evaluatorSetFraction(t, e, 0, 1, 1)
	y := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0})
	x := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1})
	yes := normalizationLiteral(t, r, AtomGT, []polynomial.Polynomial{y}, []bool{false})
	no := normalizationLiteral(t, r, AtomLT, []polynomial.Polynomial{y}, []bool{false})
	keep := normalizationLiteral(t, r, AtomGT, []polynomial.Polynomial{x}, []bool{false})
	core := scopedLiteralVector{factory: r.store.atoms}
	t.Cleanup(func() { assignmentMust(t, core.close()) })
	for _, l := range []sat.Literal{yes, keep, yes, keep} {
		assignmentMust(t, core.push(l))
	}
	assignmentMust(t, n.collectNormalizeCore(&core, 1))
	if len(core.result) != 2 || core.result[0] != keep || core.result[1] != keep {
		t.Fatal("true literals not removed stably", core.result)
	}
	assignmentMust(t, core.push(no))
	assignmentMust(t, n.collectNormalizeCore(&core, 1))
	if len(core.result) != 0 {
		t.Fatal("false normalization did not reset conflict core", core.result)
	}
}
func TestExplanationNormalizeRawNegativeSign(t *testing.T) {
	e, r, n := explanationAssumptionFixture(t)
	evaluatorSetFraction(t, e, 0, -2, 1)
	y := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0})
	x := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1})
	l := normalizationLiteral(t, r, AtomGT, []polynomial.Polynomial{y, x}, []bool{false, false})
	result, err := n.collectNormalize(l, 1)
	assignmentMust(t, err)
	normalizationOwnLiteral(t, r, result)
	a := r.store.atoms.bools.atoms[result.Variable()].(*InequalityAtom)
	raw, err := n.sign(y)
	assignmentMust(t, err)
	if raw != -2 || a.kind != AtomLT || len(n.output.result) != 1 {
		t.Fatal("raw negative sign did not flip odd-factor polarity", raw, a.kind)
	}
}

func TestExplanationNormalizeAllocationStopsAndRetry(t *testing.T) {
	success, failures := false, 0
	for stop := int64(0); stop < 256 && !success; stop++ {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			e, r, n := explanationAssumptionFixture(t)
			evaluatorSetFraction(t, e, 0, 1, 1)
			p := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 3}, evaluatorTerm{-1, 0, 3}, evaluatorTerm{1, 1, 1}, evaluatorTerm{2, 0, 1}, evaluatorTerm{1, 0, 0})
			y := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0})
			l := normalizationLiteral(t, r, AtomGT, []polynomial.Polynomial{p, y}, []bool{false, false})
			e.budget.FailAfter(stop)
			result, err := n.collectNormalize(l, 1)
			e.budget.FailAfter(-1)
			if err == nil {
				success = true
				normalizationOwnLiteral(t, r, result)
				return
			}
			failures++
			var oom umemory.Failure
			if !errors.Is(err, arena.ErrInjectedFailure) && !(errors.As(err, &oom) && oom.Kind == umemory.FailureMemout && oom.ErrorCode() == 101) {
				t.Fatal("unexpected source stop", stop, err)
			}
			if err := r.newLevel(); !errors.Is(err, ErrSearchPending) {
				t.Fatal("failed normalization allowed search", err)
			}
			assignmentMust(t, e.am.CleanupRetained())
			assignmentMust(t, r.store.atoms.ineq.polynomials.CleanupFactor())
			assignmentMust(t, r.store.atoms.ineq.polynomials.CleanupUnique())
			assignmentMust(t, r.store.atoms.ineq.pm.CleanupFlipSignIfLeadingMonomialNegative())
			assignmentMust(t, r.store.atoms.cleanup())
			assignmentMust(t, n.cleanup())
			if n.normalization.active || n.vanishing.active || n.localFactor != nil {
				t.Fatal("lexical normalization scope remained", stop)
			}
			assignmentMust(t, r.store.atoms.ineq.polynomials.Reset())
			result, err = n.collectNormalize(l, 1)
			assignmentMust(t, err)
			normalizationOwnLiteral(t, r, result)
			if result == TrueLiteral || result == FalseLiteral || result == l {
				t.Fatal("retry did not produce actual normalized literal", stop, result)
			}
		})
	}
	if !success || failures < 3 {
		t.Fatal("normalization allocation horizon incomplete", success, failures)
	}
	t.Log("consecutive failures", failures)
}
func TestExplanationVanishingVectorPreservesBorrowedSET(t *testing.T) {
	e, _, n := explanationAssumptionFixture(t)
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1})
	ps := projectionOutput(t, e)
	assignmentMust(t, ps.Push(p))
	err := ps.Set(9, p)
	if err == nil {
		t.Fatal("actual borrowed SET did not stop")
	}
	before, err := e.core.Info(p)
	assignmentMust(t, err)
	err = n.collectVanishingVector(ps)
	if !errors.Is(err, polynomial.ErrPolynomialPending) {
		t.Fatal("borrowed pending SET not guarded", err)
	}
	assignmentMust(t, n.cleanup())
	after, err := e.core.Info(p)
	assignmentMust(t, err)
	if after.RefCount != before.RefCount || len(projectionValues(t, ps)) != 1 {
		t.Fatal("outer cleanup adopted borrowed SET")
	}
	if err := ps.Set(0, p); !errors.Is(err, polynomial.ErrPolynomialPending) {
		t.Fatal("borrowed SET authority was consumed", err)
	}
	assignmentMust(t, ps.CleanupRetained())
}

func TestExplanationNormalizeCorePreservesBorrowedPendingSET(t *testing.T) {
	e, r, n := explanationAssumptionFixture(t)
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1})
	l := normalizationLiteral(t, r, AtomGT, []polynomial.Polynomial{p}, []bool{false})
	core := scopedLiteralVector{factory: r.store.atoms}
	t.Cleanup(func() { assignmentMust(t, core.close()) })
	assignmentMust(t, core.push(l))
	err := core.set(7, l)
	if !errors.Is(err, arena.ErrBounds) || !core.incomingOwned {
		t.Fatal("actual pending SET prefix missing", err)
	}
	refs := arithmeticAtomHeader(r.store.atoms.bools.atoms[l.Variable()]).refs
	err = n.collectNormalizeCore(&core, 1)
	if !errors.Is(err, ErrSearchPending) {
		t.Fatal("normalization adopted borrowed SET", err)
	}
	assignmentMust(t, n.cleanup())
	if !core.setActive || !core.incomingOwned || core.result[0] != l || arithmeticAtomHeader(r.store.atoms.bools.atoms[l.Variable()]).refs != refs {
		t.Fatal("outer cleanup changed borrowed vector")
	}
	assignmentMust(t, core.cleanup())
}

func TestExplanationNormalizeRetainedFactorParity(t *testing.T) {
	e, r, n := explanationAssumptionFixture(t)
	evaluatorSetFraction(t, e, 0, -1, 1)
	y := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0})
	x := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1})
	l := normalizationLiteral(t, r, AtomGT, []polynomial.Polynomial{y, x}, []bool{false, true})
	result, err := n.collectNormalize(l, 1)
	assignmentMust(t, err)
	normalizationOwnLiteral(t, r, result)
	a := r.store.atoms.bools.atoms[result.Variable()].(*InequalityAtom)
	if a.kind != AtomLT || len(a.factors) != 1 || !a.factors[0].Even {
		t.Fatal("retained maximal-stage even exponent lost", a.kind, a.factors)
	}
}
func TestExplanationNormalizeLexicalVectorDestruction(t *testing.T) {
	e, r, n := explanationAssumptionFixture(t)
	evaluatorSetFraction(t, e, 0, 0, 1)
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 3}, evaluatorTerm{1, 0, 1}, evaluatorTerm{1, 0, 0})
	y := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0})
	l := normalizationLiteral(t, r, AtomGT, []polynomial.Polynomial{p, y}, []bool{false, false})
	stop := errors.New("stop actual normalization local vector destructor")
	calls := 0
	sourceOrder := false
	var reentry error
	h, err := polynomial.NewPolynomialDeletionHandler(e.budget, func(_ *polynomial.PolynomialDeletionHandler, _ *polynomial.MultivariatePolynomialCore, p polynomial.Polynomial) error {
		degree, err := e.core.Degree(p, 1)
		if err != nil {
			return err
		}
		_, scopeErr := n.normalization.ps.Size()
		if degree == 1 && errors.Is(scopeErr, arena.ErrClosed) {
			calls++
			sourceOrder = n.normalization.pFinished && !n.normalization.psFinished
			reentry = n.cleanup()
			return stop
		}
		return nil
	})
	assignmentMust(t, err)
	assignmentMust(t, e.core.AddDeletionHandler(h))
	t.Cleanup(func() { assignmentMust(t, e.core.RemoveDeletionHandler(h)); assignmentMust(t, h.CloseChecked()) })
	result, err := n.collectNormalize(l, 1)
	if !errors.Is(err, stop) || result != FalseLiteral || calls != 1 || !sourceOrder || !errors.Is(reentry, ErrSearchPending) || !n.normalization.active {
		t.Fatal("source p/ps destruction or callback guard", result, err, calls, sourceOrder, reentry)
	}
	assignmentMust(t, n.cleanup())
	if calls != 1 || n.normalization.active {
		t.Fatal("owning vector cleanup replayed source DEC", calls)
	}
}
