package polynomial

import (
	"errors"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"testing"
)

func TestUPolynomialSequenceOwnsAcceptedBinaryEvaluation(t *testing.T) {
	c, m, runtime := algebraicNegTLSFixture(t, nil)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	s, e := m.NewSequence()
	serviceMust(t, e)
	serviceMust(t, s.PushCopy(v))
	serviceMust(t, s.PushCopy(v))
	pad := algebraicCellSourceTLSBoundary(t, c)
	_, body := m.SignVariationsBinary(s, r.interval.BorrowLower())
	if body == nil || !m.matchesBinaryEvaluationReceipt(&s.evaluation) || !s.evaluationPending() || s.evaluating {
		t.Fatal("sequence lost accepted Horner child", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	before, used, stats := s.Stats(), m.config.Integer.Budget().Used(), runtime.Stats()
	for _, e := range []error{s.Reset(), s.PushCopy(v), s.PushMove(v), m.SturmSequence(v, s), m.SturmSequencePair(v, v, s), m.SturmTarskiSequence(v, v, s), m.FourierSequence(v, s)} {
		if !errors.Is(e, ErrUPolynomialCleanupPending) {
			t.Fatal("mutation discarded retained Horner", e)
		}
	}
	if before != s.Stats() || used != m.config.Integer.Budget().Used() || stats != runtime.Stats() {
		t.Fatal("refused mutation changed sequence")
	}
	serviceMust(t, s.CleanupEvaluation())
	if s.evaluationPending() || m.binaryEvaluation.active {
		t.Fatal("sequence did not acknowledge accepted child")
	}
	count, e := m.SignVariationsBinary(s, r.interval.BorrowLower())
	serviceMust(t, e)
	if count != 0 {
		t.Fatal("two identical polynomial rows must have zero sign variations", count)
	}
	// Close is the owning boundary and also acknowledges its accepted child.
	pad = algebraicCellSourceTLSBoundary(t, c)
	_, body = m.SignVariationsBinary(s, r.interval.BorrowLower())
	if body == nil || !s.evaluationPending() {
		t.Fatal("retry did not reach actual Horner refusal", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	serviceMust(t, s.Close())
	if m.binaryEvaluation.active {
		t.Fatal("sequence close lost child")
	}
	algebraicComparisonAudit(t, c, &root)
	serviceMust(t, c.Del(&root))
}

func TestUPolynomialSequenceRefusedEvaluationPreservesOlderHorner(t *testing.T) {
	c, m, runtime := algebraicNegTLSFixture(t, nil)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	s, e := m.NewSequence()
	serviceMust(t, e)
	serviceMust(t, s.PushCopy(v))
	serviceMust(t, s.PushCopy(v))
	pad := algebraicCellSourceTLSBoundary(t, c)
	_, body := m.EvalSignBinary(v, r.interval.BorrowLower())
	if body == nil || !m.binaryEvaluation.active {
		t.Fatal("older Horner did not retain real allocation", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	before, used, stats := m.binaryEvaluation, m.config.Integer.Budget().Used(), runtime.Stats()
	_, body = m.SignVariationsBinary(s, r.interval.BorrowLower())
	if body == nil || s.evaluation != (upolyBinaryEvaluationInvocationReceipt{}) || s.evaluationPending() {
		t.Fatal("refused sequence adopted older Horner", body)
	}
	serviceMust(t, s.CleanupEvaluation())
	if !reflect.DeepEqual(before, m.binaryEvaluation) || used != m.config.Integer.Budget().Used() || stats != runtime.Stats() {
		t.Fatal("sequence cleanup consumed older evaluation")
	}
	serviceMust(t, s.Close())
	if !reflect.DeepEqual(before, m.binaryEvaluation) {
		t.Fatal("sequence close consumed older evaluation")
	}
	serviceMust(t, m.CleanupBinaryEvaluation())
	serviceMust(t, c.Del(&root))
}

func TestUPolynomialSequenceExecutingEvaluationRejectsMutation(t *testing.T) {
	var m *UPolynomialManager
	var s *UPolynomialSequence
	var v UPolynomialView
	var lower *unumeral.BinaryRational
	probed, armed := false, false
	transport := func(f umemory.Failure) error {
		if !armed {
			return f
		}
		armed = false
		probed = true
		used := m.config.Integer.Budget().Used()
		for _, e := range []error{s.Close(), s.CleanupEvaluation(), s.Reset(), s.PushCopy(v), s.PushMove(v), m.SturmSequence(v, s)} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				t.Error("failure callback acquired executing sequence", e)
			}
		}
		if _, e := m.SignVariationsBinary(s, lower); !errors.Is(e, ErrUPolynomialCleanupPending) {
			t.Error("failure callback replaced evaluation", e)
		}
		if !s.evaluating || used != m.config.Integer.Budget().Used() {
			t.Error("failure callback changed sequence owner")
		}
		return f
	}
	c, owner, runtime := algebraicNegTLSFixture(t, transport)
	m = owner
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e = c.sourcePolynomial(&r)
	serviceMust(t, e)
	lower = r.interval.BorrowLower()
	s, e = m.NewSequence()
	serviceMust(t, e)
	serviceMust(t, s.PushCopy(v))
	serviceMust(t, s.PushCopy(v))
	pad := algebraicCellSourceTLSBoundary(t, c)
	armed = true
	_, body := m.SignVariationsBinary(s, lower)
	if !probed || body == nil || s.evaluating {
		t.Fatal("did not return actual executing Horner refusal", body, probed)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	serviceMust(t, s.CleanupEvaluation())
	count, e := m.SignVariationsBinary(s, lower)
	serviceMust(t, e)
	if count != 0 {
		t.Fatal("sequence did not resume legitimate evaluation", count)
	}
	serviceMust(t, s.Close())
	serviceMust(t, c.Del(&root))
}
