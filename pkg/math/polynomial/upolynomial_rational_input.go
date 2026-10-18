// Source: src/math/polynomial/upolynomial.cpp:204-211.
package polynomial

import unumeral "github.com/glycerine/g3/pkg/util/numeral"

// Receiving numerical children outlive a stopped global-rational borrow. The
// source temporary remains with that service's existing lexical owner.
type upolyRationalInputScope struct {
	copy             unumeral.IntegerCopyAllocationReceipt
	normalize        unumeral.IntegerNormalizeInvocationReceipt
	integer, natural unumeral.NumericCallerBinding
	running          bool
}

func (s *upolyRationalInputScope) NumericExecuting() bool { return s.running }
func (s *upolyRationalInputScope) pending(m *UPolynomialManager) bool {
	return m.config.Integer.MatchesCopyAllocationInvocation(&s.copy) || m.modular.MatchesPNormalizeInvocation(&s.normalize)
}
func (s *upolyRationalInputScope) bind(z *unumeral.IntegerManager) error {
	if e := z.BindCaller(&s.integer, s, 0); e != nil {
		return e
	}
	if e := z.Natural.BindCaller(&s.natural, s, 0); e != nil {
		_ = s.integer.Unbind()
		return e
	}
	return nil
}
func (s *upolyRationalInputScope) unbind() {
	if e := s.natural.Unbind(); e != nil {
		panic(e)
	}
	if e := s.integer.Unbind(); e != nil {
		panic(e)
	}
}

// CleanupRationalInput acknowledges only the receiving children actually issued
// by SetRationals/the S-expression constant conversion, before vector cleanup.
func (m *UPolynomialManager) CleanupRationalInput() error {
	if m == nil || m.self != m {
		return ErrUPolynomialOwner
	}
	s := &m.rationalInput
	if s.running {
		return ErrUPolynomialCleanupPending
	}
	if !s.pending(m) {
		return nil
	}
	s.running = true
	if e := s.bind(m.config.Integer); e != nil {
		s.running = false
		return e
	}
	defer func() { s.unbind(); s.running = false }()
	if e := m.config.Integer.CleanupCopyAllocationInvocation(&s.copy); e != nil {
		return e
	}
	return m.modular.CleanupPNormalizeInvocation(&s.normalize, true)
}
