// Source: src/math/polynomial/polynomial.cpp:3461-3465;
// src/util/mpzzp.h:111-114,229-235. Licensed under the MIT license.
// Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// This embedded cold continuation names only the reached SET invocation. The
// source constant branch constructs no ai, ci, GCD, filter or powers local.
type polynomialICCPConstantScope struct {
	invocation                           uint64
	active, executing                    bool
	numeric                              polynomialNumericRoots
	normalize                            integerPNormalizeInvocationReceipt
	directReceipt                        polynomialICCPConstantInvocationReceipt
	bodyCause, sourceCause, cleanupCause error
}
type polynomialICCPConstantInvocationReceipt struct {
	manager    *MultivariatePolynomialManager
	frame      *polynomialICCPConstantScope
	invocation uint64
}

func (m *MultivariatePolynomialManager) hasICCPConstantHistory(r *polynomialICCPConstantInvocationReceipt) bool {
	return m != nil && m.self == m && r != nil && r.manager == m && r.frame == &m.iccpConstant && r.invocation != 0 && r.frame.invocation == r.invocation
}
func (m *IntegerManager) iccpConstantContextRoots(context integerAllocationInvocationContext) *polynomialNumericRoots {
	r := context.iccpConstant
	if m == nil || m.self != m || r == nil || r.manager == nil || r.manager.core == nil || r.manager.core.self != r.manager.core || r.manager.core.integer != m || !r.manager.hasICCPConstantHistory(r) || context.iccpConstantInvocation == 0 || context.iccpConstantInvocation != r.invocation || !r.frame.active || !r.frame.executing || r.manager.core.callbackDepth != 0 {
		return nil
	}
	return &r.frame.numeric
}
func (m *MultivariatePolynomialManager) iccpConstantSET(i, a *Int, parent *polynomialICCPInvocationReceipt) (err error) {
	c := m.core
	if e := c.modular.check(); e != nil {
		return e
	}
	if e := c.integer.check(i); e != nil {
		return e
	}
	if e := c.integer.check(a); e != nil {
		return e
	}
	s := &m.iccpConstant
	if s.active {
		return ErrPolynomialPending
	}
	if s.invocation == ^uint64(0) {
		return arena.ErrBounds
	}
	*s = polynomialICCPConstantScope{invocation: s.invocation + 1, active: true, executing: true}
	s.directReceipt = polynomialICCPConstantInvocationReceipt{m, s, s.invocation}
	if parent != nil {
		*parent = polynomialICCPInvocationReceipt{manager: m, shortcutConstant: s.directReceipt}
	}
	defer func() {
		s.executing = false
		s.bodyCause = err
		if s.sourceCause == nil {
			s.sourceCause = c.modular.pNormalizeInvocationSourceCause(&s.normalize)
		}
		err = errors.Join(err, s.sourceCause)
		pending := s.numeric.cell.object != (memory.ObjectRef{}) || c.modular.matchesPNormalizeInvocation(&s.normalize)
		for _, raw := range s.numeric.allocator {
			pending = pending || raw != (memory.Block{})
		}
		if !pending {
			s.active = false
		}
	}()
	return c.modular.copyWithICCPConstantReceipt(i, a, &s.directReceipt)
}
func (m *MultivariatePolynomialManager) cleanupICCPConstantInvocation(r *polynomialICCPConstantInvocationReceipt, abort bool) error {
	if !m.hasICCPConstantHistory(r) || !r.frame.active {
		return nil
	}
	s := r.frame
	c := m.core
	if s.executing {
		return ErrPolynomialPending
	}
	if e := c.modular.cleanupPNormalizeInvocation(&s.normalize, abort); e != nil {
		s.cleanupCause = e
		return e
	}
	if !abort {
		return nil
	}
	if e := s.numeric.cleanup(c.integer); e != nil {
		s.cleanupCause = e
		return e
	}
	s.active = false
	return nil
}

func (m *IntegerModularManager) iccpConstantAllocationContext(r *polynomialICCPConstantInvocationReceipt) integerAllocationInvocationContext {
	context := integerAllocationInvocationContext{}
	if m == nil || m.manager == nil || r == nil || r.manager == nil || r.manager.core == nil || r.manager.core.modular != m {
		return context
	}
	context = integerAllocationInvocationContext{iccpConstant: r, iccpConstantInvocation: r.invocation}
	if m.manager.iccpConstantContextRoots(context) == nil {
		return integerAllocationInvocationContext{}
	}
	return context
}
func (m *IntegerModularManager) copyWithICCPConstantReceipt(out, a *Int, r *polynomialICCPConstantInvocationReceipt) error {
	return m.copyWithSourceContext(out, a, m.iccpConstantAllocationContext(r))
}
