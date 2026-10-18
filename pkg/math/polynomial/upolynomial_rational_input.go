// Source: src/math/polynomial/upolynomial.cpp:204-211.
package polynomial

import unumeral "github.com/glycerine/g3/pkg/util/numeral"

// Receiving numerical children outlive a stopped global-rational borrow. The
// source temporary remains with that service's existing lexical owner.
type upolyRationalInputScope struct {
	copy      unumeral.IntegerCopyAllocationReceipt
	normalize unumeral.IntegerNormalizeInvocationReceipt
	bindings  [4]unumeral.NumericCallerBinding
	bound     uint8
	running   bool
	source    *unumeral.IntegerManager
}

func (s *upolyRationalInputScope) NumericExecuting() bool { return s.running }
func (s *upolyRationalInputScope) pending(m *UPolynomialManager) bool {
	return m.config.Integer.MatchesCopyAllocationInvocation(&s.copy) || m.modular.MatchesPNormalizeInvocation(&s.normalize)
}
func (s *upolyRationalInputScope) bind(z, source *unumeral.IntegerManager) error {
	for i, owner := range [2]*unumeral.IntegerManager{z, source} {
		if owner == nil || i == 1 && owner == z {
			continue
		}
		if e := owner.BindCaller(&s.bindings[s.bound], s, 0); e != nil {
			s.unbind()
			return e
		}
		s.bound++
		if e := owner.Natural.BindCaller(&s.bindings[s.bound], s, 0); e != nil {
			s.unbind()
			return e
		}
		s.bound++
	}
	return nil
}
func (s *upolyRationalInputScope) unbind() {
	for s.bound != 0 {
		s.bound--
		if e := s.bindings[s.bound].Unbind(); e != nil {
			panic(e)
		}
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
		s.source = nil
		return nil
	}
	s.running = true
	if e := s.bind(m.config.Integer, s.source); e != nil {
		s.running = false
		return e
	}
	defer func() {
		s.unbind()
		s.running = false
		if !s.pending(m) {
			s.source = nil
		}
	}()
	if e := m.config.Integer.CleanupCopyAllocationInvocation(&s.copy); e != nil {
		return e
	}
	return m.modular.CleanupPNormalizeInvocation(&s.normalize, true)
}
