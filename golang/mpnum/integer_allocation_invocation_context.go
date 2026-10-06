// Source: src/util/mpz.h:allocate; src/util/mpz.cpp scratch/publication callers.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

// Explicit passed invocation identity. It carries no allocation snapshot and
// creates no source numeral, reservation, or ambient callback owner hook.
type integerAllocationInvocationContext struct {
	trial                   *upolyRecombinationTrialScope
	recombination           *upolyRecombinationStepScope
	recombinationInvocation uint64
	recombinationManager    *UPolynomialManager
	trialInvocation         uint64
	trialManager            *UPolynomialManager
	henselCaller            *upolyHenselCallerScope
	henselCallerInvocation  uint64
	henselCallerManager     *UPolynomialManager
	henselPair              *upolyHenselPairInvocationReceipt
	henselPairInvocation    uint64
	henselManager           *UPolynomialManager
	iccpConstant            *polynomialICCPConstantInvocationReceipt
	iccpConstantInvocation  uint64
	primeTrial              *upolyPrimeTrialInvocationReceipt
	primeTrialInvocation    uint64
	pseudo                  *polynomialPseudoDivisionInvocationReceipt
	pseudoInvocation        uint64
	mignotte                *upolyMignotteInvocationReceipt
	mignotteInvocation      uint64
	divides                 *polynomialDividesInvocationReceipt
	dividesInvocation       uint64
	boolDivides             *integerDividesNumeralInvocationReceipt
	boolDividesInvocation   uint64
	neg                     *polynomialNegInvocationReceipt
	negInvocation           uint64
	arithmeticInvocation    uint64
	scalarExact             *polynomialScalarExactDivInvocationReceipt
	scalarExactInvocation   uint64
	exactDiv                *polynomialExactDivInvocationReceipt
	exactDivInvocation      uint64
	content                 *polynomialContentInvocationReceipt
	contentInvocation       uint64
	raw                     *integerRawInvocationReceipt
	rawInvocation           uint64
	pn                      *integerPNormalizeScope
	pnInvocation            uint64
	arithmetic              *polynomialArithmeticInvocationReceipt
	addMul                  *integerAddMulInvocationReceipt
	addMulInvocation        uint64
	derivative              *polynomialDerivativeInvocationReceipt
	scalarDiv               *polynomialScalarDivInvocationReceipt
}

func (c integerAllocationInvocationContext) withAddMulInvocation(r *integerAddMulInvocationReceipt) integerAllocationInvocationContext {
	c.addMul = r
	c.addMulInvocation = 0
	if r != nil {
		c.addMulInvocation = r.invocation
	}
	return c
}
func (c integerAllocationInvocationContext) withRawInvocation(r *integerRawInvocationReceipt) integerAllocationInvocationContext {
	c.raw = r
	c.rawInvocation = 0
	if r != nil {
		c.rawInvocation = r.invocation
	}
	return c
}
func (c integerAllocationInvocationContext) rawReceipt(m *IntegerManager) *integerRawInvocationReceipt {
	if c.raw != nil && c.rawInvocation != 0 && c.rawInvocation == c.raw.invocation && m.rawAllocationRoots(c.raw) != nil {
		return c.raw
	}
	return nil
}

func pnAllocationContext(pn *integerPNormalizeScope) integerAllocationInvocationContext {
	c := integerAllocationInvocationContext{pn: pn}
	if pn != nil {
		c.pnInvocation = pn.invocation
	}
	return c
}
func (c integerAllocationInvocationContext) roots(m *IntegerManager) *polynomialNumericRoots {
	if c.recombination != nil && c.recombinationFrame(m) == nil {
		return nil
	}
	if c.trial != nil && c.trialFrame(m) == nil {
		return nil
	}
	if c.henselCaller != nil && c.henselCallerFrame(m) == nil {
		return nil
	}
	if c.henselPair != nil && c.henselPairReceipt(m) == nil {
		return nil
	}
	if c.iccpConstant != nil && m.iccpConstantContextRoots(c) == nil {
		return nil
	}
	if c.primeTrial != nil && c.primeTrialReceipt(m) == nil {
		return nil
	}
	if (c.pseudo != nil && m.pseudoContextRoots(c) == nil) || (c.mignotte != nil && c.mignotteReceipt(m) == nil) {
		return nil
	}
	if c.boolDivides != nil {
		return m.dividesNumeralContextRoots(c)
	}
	if c.neg != nil {
		if c.negInvocation == 0 || c.negInvocation != c.neg.invocation || m.negAllocationRoots(c.neg) == nil {
			return nil
		}
	}
	if c.content != nil && c.contentReceipt(m) == nil {
		return nil
	}
	if c.raw != nil {
		if r := c.rawReceipt(m); r != nil {
			return m.rawAllocationRoots(r)
		}
		return nil
	}
	if s := c.pn; s != nil && s.owner != nil && s.owner.manager == m && s == &s.owner.normalization && s.active && s.executing && c.pnInvocation != 0 && c.pnInvocation == s.invocation {
		return &s.roots
	}
	if c.iccpConstant != nil {
		return m.iccpConstantContextRoots(c)
	}
	if c.pseudo != nil {
		return m.pseudoContextRoots(c)
	}
	if c.divides != nil {
		return m.dividesContextRoots(c)
	}
	if c.scalarExact != nil {
		return m.scalarExactContextRoots(c)
	}
	if c.exactDiv != nil {
		return m.exactDivContextRoots(c)
	}
	if p := c.contentReceipt(m); p != nil {
		return m.contentAllocationRoots(p)
	}
	if c.neg != nil {
		return m.negAllocationRoots(c.neg)
	}
	if c.scalarDiv != nil {
		return m.scalarDivAllocationRoots(c.scalarDiv)
	}
	if c.derivative != nil {
		return m.derivativeAllocationRoots(c.derivative)
	}
	if c.addMul != nil && c.addMulInvocation != 0 && c.addMulInvocation == c.addMul.invocation && m.matchesAddMulInvocation(c.addMul) && c.addMul.frame.executing {
		s := c.addMul.frame
		if s.parentContext != (integerAllocationInvocationContext{}) && s.parentContext.roots(m) == nil {
			return nil
		}
		if s.parent.manager != nil && m.arithmeticAllocationRoots(&s.parent) == nil {
			return nil
		}
		return &s.numeric
	}
	if r := c.recombinationFrame(m); r != nil {
		return &r.numeric
	}
	if q := c.trialFrame(m); q != nil {
		return &q.numeric
	}
	if r := c.primeTrialReceipt(m); r != nil {
		return &r.frame.numeric
	}
	if r := c.henselPairReceipt(m); r != nil {
		return &r.frame.numeric
	}
	if r := c.henselCallerFrame(m); r != nil {
		return &r.numeric
	}
	if r := c.mignotteReceipt(m); r != nil {
		return &r.frame.numeric
	}
	return m.arithmeticAllocationRoots(c.arithmeticReceipt(m))
}

func (m *IntegerManager) copyFromOwnerWithArithmeticReceipt(out *Int, owner *IntegerManager, value *Int, receipt *polynomialArithmeticInvocationReceipt) error {
	return m.copyFromOwnerWithAllocationContext(out, owner, value, nil, (integerAllocationInvocationContext{}).withArithmeticInvocation(receipt))
}
func (m *IntegerModularManager) arithmeticReceiptForReceiver(r *polynomialArithmeticInvocationReceipt) *polynomialArithmeticInvocationReceipt {
	if m == nil || m.manager == nil || r == nil || r.manager == nil || r.manager.modular != m || m.manager.arithmeticAllocationRoots(r) == nil {
		return nil
	}
	return r
}
func (m *IntegerModularManager) arithmeticAllocationContext(r *polynomialArithmeticInvocationReceipt) integerAllocationInvocationContext {
	return (integerAllocationInvocationContext{}).withArithmeticInvocation(m.arithmeticReceiptForReceiver(r))
}
func (m *IntegerModularManager) arithmeticAddMulReceipt(r *polynomialArithmeticInvocationReceipt) *integerAddMulInvocationReceipt {
	if m.arithmeticReceiptForReceiver(r) == nil {
		return nil
	}
	return &r.frame.addMul
}
func (m *IntegerModularManager) normalizeWithArithmeticReceipt(out *Int, r *polynomialArithmeticInvocationReceipt) error {
	if m.arithmeticReceiptForReceiver(r) == nil {
		return m.PNormalize(out)
	}
	e := m.pNormalizeWithReceipt(out, &r.frame.normalize)
	if r.frame.sourceCause == nil {
		r.frame.sourceCause = m.pNormalizeInvocationSourceCause(&r.frame.normalize)
	}
	return e
}

func (c integerAllocationInvocationContext) withArithmeticInvocation(r *polynomialArithmeticInvocationReceipt) integerAllocationInvocationContext {
	c.arithmetic = r
	c.arithmeticInvocation = 0
	if r != nil {
		c.arithmeticInvocation = r.invocation
	}
	return c
}
func (c integerAllocationInvocationContext) arithmeticReceipt(m *IntegerManager) *polynomialArithmeticInvocationReceipt {
	if c.arithmetic != nil && c.arithmeticInvocation != 0 && c.arithmeticInvocation == c.arithmetic.invocation && m.arithmeticAllocationRoots(c.arithmetic) != nil {
		return c.arithmetic
	}
	return nil
}
