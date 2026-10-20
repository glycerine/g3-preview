// Source: src/math/polynomial/polynomial.cpp:6438-6439.
package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"io"
)

// Entry TRACE finishes before the lexical aux declaration and any recursive
// evaluation. One actual diagnostic per root owner can therefore be reused
// after successful completion; a pending child prevents that reuse.
type polynomialEvaluationTraceScope struct {
	self                                 *polynomialEvaluationTraceScope
	owner                                *MultivariatePolynomialManager
	tags                                 *diagnostic.TraceTags
	invocation                           uint64
	active, executing                    bool
	bindings                             [2]unumeral.NumericCallerBinding
	bound                                uint8
	display                              polynomialDisplayInvocationReceipt
	bodyCause, sourceCause, cleanupCause error
}

func (s *polynomialEvaluationTraceScope) NumericExecuting() bool { return s.executing }
func (m *MultivariatePolynomialCore) evaluationTraceCallback() bool {
	if m == nil || m.self != m || m.evaluationTrace == nil {
		return false
	}
	s := m.evaluationTrace
	if m.callbackDepth != 0 || s.tags.OutputCallbackActive() {
		return true
	}
	rt := m.integer.RuntimeOwner()
	return rt != nil && rt.FailureTransportActive()
}
func (m *MultivariatePolynomialManager) evaluationTraceActual(s *polynomialEvaluationTraceScope) bool {
	return m != nil && m.self == m && s != nil && (s == &m.evaluation.trace || s == &m.algebraicEvaluation.trace) && s.owner == m && s.self == s && s.invocation != 0
}
func (m *MultivariatePolynomialManager) traceEvaluationEntry(p Polynomial, start, end, x uint32, s *polynomialEvaluationTraceScope, algebraic bool) (err error) {
	tags := m.core.monomials.monomialTraceTags()
	if !tags.ShouldTrace("eval_bug") {
		return nil
	}
	if (!algebraic && s != &m.evaluation.trace) || (algebraic && s != &m.algebraicEvaluation.trace) {
		return ErrPolynomialOwner
	}
	if s.active || s.executing {
		return ErrPolynomialPending
	}
	if s.invocation == ^uint64(0) {
		return arena.ErrBounds
	}
	*s = polynomialEvaluationTraceScope{self: s, owner: m, tags: tags, invocation: s.invocation + 1, active: true, executing: true}
	previous := m.core.evaluationTrace
	m.core.evaluationTrace = s
	defer func() {
		for s.bound != 0 {
			s.bound--
			if e := s.bindings[s.bound].Unbind(); e != nil {
				panic(e)
			}
		}
		m.core.evaluationTrace = previous
		s.executing = false
		s.bodyCause = err
		s.sourceCause = m.core.displayReceiptSourceCause(&s.display)
		s.active = m.core.displayReceiptPending(&s.display) || s.sourceCause != nil
		if s.sourceCause != nil {
			if algebraic {
				if m.algebraicEvaluation.sourceCause == nil {
					m.algebraicEvaluation.sourceCause = s.sourceCause
				}
			} else if m.evaluation.sourceCause == nil {
				m.evaluation.sourceCause = s.sourceCause
			}
		}
		err = errors.Join(err, s.sourceCause)
	}()
	if err = m.core.integer.BindCaller(&s.bindings[0], s, 0); err != nil {
		return err
	}
	s.bound = 1
	if err = m.core.integer.Natural.BindCaller(&s.bindings[1], s, 0); err != nil {
		return err
	}
	s.bound = 2
	var computation error
	_ = tags.EmitConditional("eval_bug", "t_eval_core", "src/math/polynomial/polynomial.cpp", 6438, func() bool { return true }, func(w io.Writer) error {
		out := &polynomialTraceSink{writer: w}
		_ = out.WriteLiteral("p: ")
		computation = m.core.displayPolynomialDirectWithReceipt(p, out, false, &s.display)
		if computation != nil {
			return computation
		}
		_ = out.WriteLiteral("\nstart: ")
		_ = out.WriteUnsigned(start)
		_ = out.WriteLiteral(", end: ")
		_ = out.WriteUnsigned(end)
		_ = out.WriteLiteral(", x: ")
		_ = out.WriteUnsigned(x)
		return out.WriteLiteral("\n")
	})
	return computation
}
func (m *MultivariatePolynomialManager) cleanupEvaluationTrace(s *polynomialEvaluationTraceScope) (err error) {
	if !m.evaluationTraceActual(s) {
		return nil
	}
	if s.executing || m.core.evaluationTraceCallback() {
		return ErrPolynomialPending
	}
	if !s.active {
		return nil
	}
	defer func() {
		if err != nil && s.cleanupCause == nil {
			s.cleanupCause = err
		}
	}()
	if err = m.core.cleanupDisplayReceipt(&s.display); err != nil {
		return err
	}
	s.active = false
	return nil
}
