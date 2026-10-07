// Source: src/math/polynomial/polynomial.cpp:3405-3430.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"github.com/glycerine/g3/pkg/util"
)

type polynomialContentInvocationReceipt struct {
	manager    *MultivariatePolynomialCore
	frame      *polynomialContentScope
	invocation uint64
}

func (m *MultivariatePolynomialCore) matchesContentInvocation(r *polynomialContentInvocationReceipt) bool {
	return m != nil && m.self == m && r != nil && r.manager == m && r.frame == &m.contentLocal && r.invocation != 0 && r.invocation == r.frame.invocation && r.frame.active
}
func numericIntegerManager_contentAllocationRoots(m *util.IntegerManager, r *polynomialContentInvocationReceipt) *polynomialNumericRoots {
	if m == nil || !m.ValidOwner() || r == nil || r.manager == nil || r.manager.integer != m || !r.manager.matchesContentInvocation(r) || !r.frame.executing || r.manager.callbackDepth != 0 {
		return nil
	}
	return &r.frame.numeric
}
func numericIntegerModularManager_contentAllocationContext(m *util.IntegerModularManager, r *polynomialContentInvocationReceipt) integerAllocationInvocationContext {
	if m == nil || m.Integers() == nil || numericIntegerManager_contentAllocationRoots(m.Integers(), r) == nil || r.manager.modular != m {
		return integerAllocationInvocationContext{}
	}
	return integerAllocationInvocationContext{content: r, contentInvocation: r.invocation}
}

func (c integerAllocationInvocationContext) contentReceipt(m *util.IntegerManager) *polynomialContentInvocationReceipt {
	if c.content != nil && c.contentInvocation != 0 && c.contentInvocation == c.content.invocation && numericIntegerManager_contentAllocationRoots(m, c.content) != nil {
		return c.content
	}
	return nil
}
func (c integerAllocationInvocationContext) recordContentRaw(m *util.IntegerManager, r *util.IntegerRawInvocationReceipt) {
	p := c.contentReceipt(m)
	if p == nil || r == nil || !m.MatchesRawInvocation(r) {
		return
	}
	if r.Kind() == util.IntegerRawGCD {
		p.frame.gcd = *r
	}

	if r.Kind() == util.IntegerRawDivision {
		p.frame.division = *r
	}
	if r.Kind() == util.IntegerRawExtendedGCD {
		p.frame.extended = *r
	}
}

func (m *MultivariatePolynomialCore) cleanupContentInvocation(r *polynomialContentInvocationReceipt, abort bool) error {
	if !m.matchesContentInvocation(r) {
		return nil
	}
	return m.finishIntegerContentPhase(true, abort)
}
func numericIntegerModularManager_gcdManyIndexedWithContentReceipt(m *util.IntegerModularManager, as util.IntegerArrayView, g *util.Int, r *polynomialContentInvocationReceipt) error {
	if e := m.Check(); e != nil {
		return e
	}
	if e := m.Integers().WithAllocationContext((numericIntegerModularManager_contentAllocationContext(m, r)).utility(m.Integers())).GcdManyIndexed(as, g); e != nil {
		return e
	}
	return m.AssertNormalized(g)
}

func numericIntegerModularManager_copyWithContentReceipt(m *util.IntegerModularManager, out, value *util.Int, r *polynomialContentInvocationReceipt) error {
	if e := m.Check(); e != nil {
		return e
	}
	context := numericIntegerModularManager_contentAllocationContext(m, r)
	if e := m.Integers().WithAllocationContext((context).utility(m.Integers())).CopyFromOwnerTracked(out, m.Integers(), value, nil); e != nil {
		return e
	}
	return numericIntegerModularManager_normalizeWithSourceContext(m, out, context)
}

func (m *MultivariatePolynomialCore) contentInvocationSourceCause(r *polynomialContentInvocationReceipt) error {
	if m == nil || m.self != m || r == nil || r.manager != m || r.frame != &m.contentLocal || r.invocation == 0 || r.invocation != r.frame.invocation {
		return nil
	}
	return r.frame.cleanupCause
}
