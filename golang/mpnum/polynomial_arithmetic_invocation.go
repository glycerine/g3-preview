// Source: src/math/polynomial/polynomial.cpp:1885-2156,2217-2300,2665-2744;
// src/util/obj_ref.h. Licensed under the MIT license.
// Source author: Leonardo de Moura (leonardo), 2011-11-15.
package mpnum

// A concrete owning continuation records acquisition by this actual source
// buffer. It is embedded in its charged core header; no lexical Runtime ticket
// or duplicate numeral array is constructed for this metadata.
type polynomialArithmeticScope struct {
	buffer                                            *polynomialSOM
	invocation                                        uint64
	active, executing, claimed, foreignMixed, stopped bool
	numeric                                           polynomialNumericRoots
	normalize                                         integerPNormalizeInvocationReceipt
	addMul                                            integerAddMulInvocationReceipt
	factory                                           polynomialFactoryInvocationReceipt
	constantGeneration                                uint64
	bodyCause, sourceCause                            error
}
type polynomialArithmeticInvocationReceipt struct {
	manager    *MultivariatePolynomialCore
	frame      *polynomialArithmeticScope
	invocation uint64
}

func (m *MultivariatePolynomialCore) matchesArithmeticInvocation(r *polynomialArithmeticInvocationReceipt) bool {
	return m != nil && m.self == m && r != nil && r.manager == m && r.frame != nil && m.ownsArithmeticBuffer(r.frame.buffer) && r.frame == &r.frame.buffer.arithmeticScope && r.invocation != 0 && r.frame.invocation == r.invocation && r.frame.active
}

// Direct numeric engines validate the concrete passed invocation. An executing
// high-level operation alone cannot acquire roots produced by an external hook.
func (m *IntegerManager) arithmeticAllocationRoots(r *polynomialArithmeticInvocationReceipt) *polynomialNumericRoots {
	if r == nil || r.manager == nil || r.manager.integer != m || !r.manager.matchesArithmeticInvocation(r) || !r.frame.executing || r.manager.callbackDepth != 0 {
		return nil
	}
	return &r.frame.numeric
}

func (m *MultivariatePolynomialCore) ownsArithmeticBuffer(b *polynomialSOM) bool {
	return m != nil && m.self == m && b != nil && b.owner == m && (b == &m.som || b == &m.som2 || b == &m.cheap || b == &m.cheap2)
}
