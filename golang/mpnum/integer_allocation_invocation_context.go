// Source: src/math/polynomial/algebraic_numbers.cpp:1565-1577; src/util/mpz.h:124-149.
// Source: src/util/mpz.h:allocate; src/util/mpz.cpp scratch/publication callers.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

// Explicit passed invocation identity. It carries no allocation snapshot and
// creates no source numeral, reservation, or ambient callback owner hook.
type integerAllocationInvocationContext struct {
	algebraicNegEvalInvocation uint64
	algebraicNeg               *algebraicNegScope
	algebraicNegInvocation     uint64
	algebraicNegEval           bool
	mpfInput                   *arbitraryFloatInputBinding
	mpfInputInvocation         uint64
	mpfConversion              *arbitraryFloatRationalConversionScope
	mpfConversionInvocation    uint64
	mpfConversionOutput        bool
	mpfFormat                  *ArbitraryFloatFormatInvocation
	mpfFormatInvocation        uint64
	formatDecimal              *rationalDecimalScope
	mpfPower                   *arbitraryFloatPowerWrapper
	mpfPowerInvocation         uint64
	upolyDisplay               *upolyDisplayScope
	upolyDisplayInvocation     uint64
	mpfRound                   *arbitraryFloatRoundScope
	mpfRoundInvocation         uint64

	factorNumeralVector              *polynomialFactorNumeralVectorReceipt
	factorNumeralVectorInvocation    uint64
	toPolynomial                     *polynomialToPolynomialInvocationReceipt
	toPolynomialInvocation           uint64
	factorMemberSet                  *polynomialFactorMemberSetReceipt
	factorMemberSetInvocation        uint64
	factorConstant                   *polynomialFactorConstantInvocationReceipt
	factorConstantInvocation         uint64
	primePreparation                 *upolyFactorPrimePreparationScope
	primePreparationInvocation       uint64
	primePreparationManager          *UPolynomialManager
	preparation                      *upolyFactorPreparationScope
	preparationInvocation            uint64
	trial                            *upolyRecombinationTrialScope
	recombination                    *upolyRecombinationStepScope
	recombinationInvocation          uint64
	recombinationManager             *UPolynomialManager
	somNormalize                     *polynomialSOMNormalizeInvocationReceipt
	somNormalizeInvocation           uint64
	normalization                    *polynomialNormalizeInvocationReceipt
	normalizationInvocation          uint64
	trialInvocation                  uint64
	trialManager                     *UPolynomialManager
	henselCaller                     *upolyHenselCallerScope
	henselCallerInvocation           uint64
	henselCallerManager              *UPolynomialManager
	henselPair                       *upolyHenselPairInvocationReceipt
	henselPairInvocation             uint64
	henselManager                    *UPolynomialManager
	iccpConstant                     *polynomialICCPConstantInvocationReceipt
	iccpConstantInvocation           uint64
	primeTrial                       *upolyPrimeTrialInvocationReceipt
	primeTrialInvocation             uint64
	pseudo                           *polynomialPseudoDivisionInvocationReceipt
	pseudoInvocation                 uint64
	mignotte                         *upolyMignotteInvocationReceipt
	mignotteInvocation               uint64
	divides                          *polynomialDividesInvocationReceipt
	dividesInvocation                uint64
	boolDivides                      *integerDividesNumeralInvocationReceipt
	boolDividesInvocation            uint64
	neg                              *polynomialNegInvocationReceipt
	negInvocation                    uint64
	arithmeticInvocation             uint64
	scalarExact                      *polynomialScalarExactDivInvocationReceipt
	scalarExactInvocation            uint64
	exactDiv                         *polynomialExactDivInvocationReceipt
	exactDivInvocation               uint64
	content                          *polynomialContentInvocationReceipt
	contentInvocation                uint64
	raw                              *integerRawInvocationReceipt
	rawInvocation                    uint64
	pn                               *integerPNormalizeScope
	pnInvocation                     uint64
	arithmetic                       *polynomialArithmeticInvocationReceipt
	addMul                           *integerAddMulInvocationReceipt
	addMulInvocation                 uint64
	derivative                       *polynomialDerivativeInvocationReceipt
	scalarDiv                        *polynomialScalarDivInvocationReceipt
	factorFrontendConstant           *upolyFactorConstantBinding
	factorFrontendConstantInvocation uint64
	perfectSquare                    *integerPerfectSquareInvocationReceipt
	perfectSquareInvocation          uint64
	factorFrontend                   *upolyFactorFrontendScope
	factorFrontendInvocation         uint64
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
	if c.algebraicNeg != nil && c.algebraicNegFrame(m) == nil {
		return nil
	}
	if c.upolyDisplay != nil && c.upolyDisplayFrame(m) == nil {
		return nil
	}
	if c.mpfRound != nil {
		if s := c.roundFrame(m, true); s != nil {
			return &s.numeric
		}
		return nil
	}
	if c.mpfInput != nil {
		return c.inputRoots(m)
	}
	if c.mpfConversion != nil && c.conversionFrame(m, true) == nil {
		return nil
	}
	if c.mpfFormat != nil && c.formatFrame(m, true) == nil {
		return nil
	}
	if c.factorNumeralVector != nil && m.factorNumeralVectorRoots(c) == nil {
		return nil
	}
	if c.toPolynomial != nil && m.toPolynomialAllocationRoots(c) == nil {
		return nil
	}
	if c.factorMemberSet != nil && m.factorMemberSetRoots(c) == nil {
		return nil
	}
	if c.factorConstant != nil && m.factorConstantContextRoots(c) == nil {
		return nil
	}
	if c.factorFrontendConstant != nil && c.factorFrontendConstantFrame(m) == nil {
		return nil
	}
	if c.perfectSquare != nil && m.perfectSquareContextRoots(c) == nil {
		return nil
	}
	if c.factorFrontend != nil && c.factorFrontendFrame(m) == nil {
		return nil
	}
	if c.mpfPower != nil {
		return c.mpfPowerRoots(m)
	}
	if c.primePreparation != nil && c.primePreparationFrame(m) == nil {
		return nil
	}
	if c.preparation != nil && c.preparationFrame(m) == nil {
		return nil
	}
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
	if c.somNormalize != nil && c.somNormalizeReceipt(m) == nil {
		return nil
	}
	if c.normalization != nil && c.normalizationReceipt(m) == nil {
		return nil
	}
	if c.content != nil && c.contentReceipt(m) == nil {
		return nil
	}
	if c.factorMemberSet != nil {
		return m.factorMemberSetRoots(c)
	}
	if c.factorConstant != nil {
		return m.factorConstantContextRoots(c)
	}
	if c.raw != nil {
		if r := c.rawReceipt(m); r != nil {
			return m.rawAllocationRoots(r)
		}
		return nil
	}
	if c.mpfConversion != nil {
		if c.mpfConversionOutput {
			return &c.mpfConversion.outputAllocation.numeric
		}
		return &c.mpfConversion.numeric
	}
	if c.mpfFormat != nil {
		if c.formatDecimal != nil {
			return &c.formatDecimal.numeric
		}
		return &c.mpfFormat.numeric
	}
	if s := c.pn; s != nil && s.owner != nil && s.owner.manager == m && s == &s.owner.normalization && s.active && s.executing && c.pnInvocation != 0 && c.pnInvocation == s.invocation {
		return &s.roots
	}
	if c.pn != nil {
		return nil
	}
	if c.factorNumeralVector != nil {
		return m.factorNumeralVectorRoots(c)
	}
	if c.toPolynomial != nil {
		return m.toPolynomialAllocationRoots(c)
	}
	if c.perfectSquare != nil {
		return m.perfectSquareContextRoots(c)
	}
	if q := c.upolyDisplayFrame(m); q != nil {
		return &q.numeric
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
	if r := c.somNormalizeReceipt(m); r != nil {
		return m.somNormalizeAllocationRoots(r)
	}
	if p := c.normalizationReceipt(m); p != nil {
		return m.normalizationAllocationRoots(p)
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
	if q := c.algebraicNegFrame(m); q != nil {
		if c.algebraicNegEval {
			return &q.eval.numeric
		}
		return &q.numeric
	}
	if r := c.primePreparationFrame(m); r != nil {
		return &r.numeric
	}
	if b := c.factorFrontendConstantFrame(m); b != nil {
		if b.record != nil {
			return &b.record.numeric
		}
		return &b.numeric
	}
	if r := c.factorFrontendFrame(m); r != nil {
		return &r.numeric
	}
	if r := c.preparationFrame(m); r != nil {
		return &r.numeric
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
