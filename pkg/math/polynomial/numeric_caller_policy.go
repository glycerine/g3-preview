// Source: src/math/polynomial/upolynomial.cpp; caller-owned numeric lifetimes.
package polynomial

import (
	"github.com/glycerine/g3/pkg/util"
)

// bufferOwner validates the saved higher invocation without requiring it to
// execute. Pending primitive buffers remain releasable after the source body.
func (r *polynomialNumericRoots) bufferOwner() (*[4]util.NaturalBufferReceipt, *util.NaturalVerificationReceipt, uint64) {
	if r == nil || r.integer == nil || !r.integer.ValidOwner() {
		return nil, nil, 0
	}
	c, z := r.caller, r.integer
	if q := c.factorFrontend; q != nil {
		if !(c.factorFrontendHistoryFrame(z) == q) || q.service.manager.config.Integer != r.integer || q.invocation == 0 || q.invocation != c.factorFrontendInvocation || &q.numeric != r {
			return nil, nil, 0
		}
		return &q.buffers, &q.verification, q.invocation
	}
	if q := c.rootUpperBound; q != nil {
		if !(q.matches()) || q.owner.config.Integer != r.integer || q.invocation == 0 || q.invocation != c.rootUpperBoundInvocation || &q.numeric != r {
			return nil, nil, 0
		}
		return &q.buffers, &q.verification, q.invocation
	}
	if q := c.removeOneHalfRoot; q != nil {
		if !(q.matches()) || q.owner.config.Integer != r.integer || q.invocation == 0 || q.invocation != c.removeOneHalfRootInvocation || &q.numeric != r {
			return nil, nil, 0
		}
		return &q.buffers, &q.verification, q.invocation
	}
	if q := c.primePreparation; q != nil {
		if !(q.matches()) || q.owner.manager.config.Integer != r.integer || q.invocation == 0 || q.invocation != c.primePreparationInvocation || &q.numeric != r {
			return nil, nil, 0
		}
		return &q.buffers, &q.verification, q.invocation
	}
	if q := c.recombination; q != nil {
		if !(q.matches()) || q.owner.manager.config.Integer != r.integer || q.invocation == 0 || q.invocation != c.recombinationInvocation || &q.numeric != r {
			return nil, nil, 0
		}
		return &q.buffers, &q.verification, q.invocation
	}
	if q := c.henselCaller; q != nil {
		if !(q.matches()) || q.owner.manager.config.Integer != r.integer || q.invocation == 0 || q.invocation != c.henselCallerInvocation || &q.numeric != r {
			return nil, nil, 0
		}
		return &q.buffers, &q.verification, q.invocation
	}
	if q := c.henselPair; q != nil {
		if !(q.service != nil && q.service.matchesHenselPairInvocation(q)) || q.service.manager.config.Integer != r.integer || q.invocation == 0 || q.invocation != c.henselPairInvocation || &q.frame.numeric != r {
			return nil, nil, 0
		}
		return &q.frame.buffers, &q.frame.verification, q.invocation
	}
	if q := c.divides; q != nil {
		if !(q.core != nil && q.core.self == q.core && q.core.matchesDividesInvocation(q)) || q.core.integer != r.integer || q.invocation == 0 || q.invocation != c.dividesInvocation || &q.frame.numeric != r {
			return nil, nil, 0
		}
		return &q.frame.buffers, &q.frame.verification, q.invocation
	}
	if q := c.scalarExact; q != nil {
		if !(q.core != nil && q.core.self == q.core && q.core.matchesScalarExactDivInvocation(q)) || q.core.integer != r.integer || q.invocation == 0 || q.invocation != c.scalarExactInvocation || &q.frame.numeric != r {
			return nil, nil, 0
		}
		return &q.frame.buffers, &q.frame.verification, q.invocation
	}
	if q := c.exactDiv; q != nil {
		if !(q.core != nil && q.core.self == q.core && q.core.matchesExactDivInvocation(q)) || q.core.integer != r.integer || q.invocation == 0 || q.invocation != c.exactDivInvocation || &q.frame.numeric != r {
			return nil, nil, 0
		}
		return &q.frame.buffers, &q.frame.verification, q.invocation
	}
	if q := c.somNormalize; q != nil {
		if !(q.manager != nil && q.manager.hasSOMNormalizeInvocationHistory(q) && q.frame.active) || q.manager.integer != r.integer || q.invocation == 0 || q.invocation != c.somNormalizeInvocation || &q.buffer.gcdNumeric != r {
			return nil, nil, 0
		}
		return &q.frame.buffers, &q.frame.verification, q.invocation
	}
	if q := c.normalization; q != nil {
		if !(q.manager != nil && q.manager.hasNormalizeInvocationHistory(q) && q.frame.active) || q.manager.core.integer != r.integer || q.invocation == 0 || q.invocation != c.normalizationInvocation || &q.frame.numeric != r {
			return nil, nil, 0
		}
		return &q.frame.buffers, &q.frame.verification, q.invocation
	}
	if q := c.content; q != nil {
		if !(q.manager != nil && q.manager.matchesContentInvocation(q)) || q.manager.integer != r.integer || q.invocation == 0 || q.invocation != c.contentInvocation || &q.frame.numeric != r {
			return nil, nil, 0
		}
		return &q.frame.buffers, &q.frame.verification, q.invocation
	}
	if q := c.scalarDiv; q != nil {
		if !(q.manager != nil && q.manager.self == q.manager && q.manager.matchesScalarDivInvocation(q)) || q.manager.integer != r.integer || q.invocation == 0 || q.invocation != c.scalarDiv.invocation || &q.frame.numeric != r {
			return nil, nil, 0
		}
		return &q.frame.buffers, &q.frame.verification, q.invocation
	}
	return nil, nil, 0
}
func (r *polynomialNumericRoots) NumericBound(executing bool) bool {
	buffers, _, _ := r.bufferOwner()
	return buffers != nil && (!executing || r.caller.roots(r.integer) == r)
}
func (r *polynomialNumericRoots) NumericBuffer(n *util.NaturalManager, index int) *util.NaturalBufferReceipt {
	if !r.NumericBound(true) || r.integer.Naturals() != n || index < 0 || index >= 4 {
		return nil
	}
	buffers, _, inv := r.bufferOwner()
	b := &buffers[index]
	if n.BindCallerBuffer(b, index, inv, &r.hooks) != nil {
		return nil
	}
	return b
}
func (r *polynomialNumericRoots) NumericVerification(n *util.NaturalManager) (*util.NaturalVerificationScope, error) {
	if !r.NumericBound(true) || r.integer.Naturals() != n {
		return nil, util.ErrNumericOwner
	}
	_, receipt, _ := r.bufferOwner()
	frame, e := n.BeginCallerVerification(&r.hooks)
	if e != nil {
		return nil, e
	}
	*receipt, e = frame.Receipt(n)
	return frame, e
}
func (r *polynomialNumericRoots) NumericPrepare(z *util.IntegerManager, phase string) error {
	if r.NumericRoots(z, nil) == nil {
		return util.ErrNumericOwner
	}
	if q := r.caller.upolyDisplayFrame(z); q != nil {
		if e := q.prepareStorage(); e != nil {
			return e
		}
	}
	if e := r.caller.prepareFactorHistory(z); e != nil {
		return e
	}
	return r.caller.prepareFactorFrontendConstantHistory(z)
}
func (r *polynomialNumericRoots) NumericNormalize(m *util.IntegerModularManager, out util.Int) (util.Int, error) {
	if m == nil || r.NumericRoots(m.Integers(), nil) == nil {
		return out, util.ErrNumericOwner
	}
	err := numericIntegerModularManager_normalizeWithSourceContext(m, &out, r.caller)
	return out, err
}
func (r *polynomialNumericRoots) NumericRecord(z *util.IntegerManager, event string, receipt util.NumericInvocationReceipt) {
	if r.NumericRoots(z, nil) == nil {
		return
	}
	c := r.caller
	if receipt.Divides != nil {
		if q := c.divides; q != nil {
			q.frame.boolNumeral = *receipt.Divides
		}
		if q := c.scalarExact; q != nil {
			q.frame.boolNumeral = *receipt.Divides
		}
		if q := c.exactDiv; q != nil {
			q.frame.boolNumeral = *receipt.Divides
		}
		if q := c.factorFrontend; q != nil {
			q.boolNumeral = *receipt.Divides
		}
		if q := c.recombination; q != nil {
			q.boolNumeral = *receipt.Divides
		}
		if q := c.trial; q != nil {
			q.boolNumeral = *receipt.Divides
		}
		if q := c.henselCaller; q != nil {
			q.boolNumeral = *receipt.Divides
		}
		if q := c.henselPair; q != nil {
			q.frame.boolNumeral = *receipt.Divides
		}
		return
	}
	if receipt.PerfectSquare != nil {
		if q := c.factorFrontend; q != nil {
			q.perfectSquare = *receipt.PerfectSquare
		}
		return
	}
	raw := receipt.Raw
	if raw == nil {
		return
	}

	switch event {
	case "division", "extended":
		c.recordRecombinationRaw(z, raw)
		c.recordRootUpperRaw(z, raw)
		c.recordRemoveOneHalfRaw(z, raw)
		c.recordPrimePreparationRaw(z, raw)
		c.recordFactorFrontendRaw(z, raw)
		c.recordContentRaw(z, raw)
		c.recordNormalizationRaw(z, raw)
		c.recordSOMNormalizationRaw(z, raw)
		division := event == "division"
		numericIntegerManager_bindHenselRaw(z, c, raw, division)
		numericIntegerManager_bindHenselCallerRaw(z, c, raw, division)
		numericIntegerManager_bindExactDivRaw(z, c, raw, division)
		numericIntegerManager_bindScalarExactDivRaw(z, c, raw, division)
		numericIntegerManager_bindDividesRaw(z, c, raw, division)
	case "gcd":
		c.recordRecombinationRaw(z, raw)
		c.recordPrimePreparationRaw(z, raw)
		c.recordFactorFrontendRaw(z, raw)
		c.recordPreparationRaw(z, raw)
		c.recordContentRaw(z, raw)
		c.recordNormalizationRaw(z, raw)
		c.recordSOMNormalizationRaw(z, raw)
	}
}

func (r *polynomialNumericRoots) NumericBufferEnabled() bool {
	buffers, _, _ := r.bufferOwner()
	return buffers != nil
}
func (r *polynomialNumericRoots) NumericVerificationEnabled() bool { return r.NumericBufferEnabled() }
func (r *polynomialNumericRoots) NumericScalarRawEnabled() bool {
	return r != nil && r.caller.scalarDiv != nil
}
func (r *polynomialNumericRoots) NumericScalarRaw(z *util.IntegerManager, event string) (*util.IntegerScalarRawScope, error) {
	if r.NumericRoots(z, nil) == nil || r.caller.scalarDiv == nil {
		return nil, util.ErrNumericOwner
	}
	s := r.caller.scalarDiv.frame
	switch event {
	case "division":
		return &s.division, nil
	case "extended":
		return &s.extended, nil
	}
	return nil, util.ErrNumericOwner
}

func (r *polynomialNumericRoots) NumericPowerEnabled() bool {
	return r != nil && r.caller.factorFrontend != nil
}
func (r *polynomialNumericRoots) NumericPower(m *util.IntegerModularManager) (util.NumericPowerInvocation, error) {
	if m == nil || r.NumericRoots(m.Integers(), nil) == nil {
		return nil, util.ErrNumericOwner
	}
	q := r.caller.factorFrontendFrame(m.Integers())
	if q == nil || q.service.manager.modular != m {
		return nil, ErrUPolynomialOwner
	}
	s := &q.modularPower
	if s.active || s.executing || s.sourceExecuting || s.ownExecuting || s.invocation == ^uint64(0) {
		return nil, ErrUPolynomialCleanupPending
	}
	*s = upolyFactorModularPowerScope{owner: q, invocation: s.invocation + 1, active: true, executing: true}
	return s, nil
}
func (q *upolyFactorModularPowerScope) NumericPowerValue() *util.Int { return &q.power }
func (q *upolyFactorModularPowerScope) NumericPowerFinish() error {
	q.endEntered = true
	return q.finishEnd()
}
func (q *upolyFactorModularPowerScope) NumericPowerEnd(e error) {
	if e != nil && q.bodyCause == nil {
		q.bodyCause = e
	}
	q.executing = false
}
func (r *polynomialNumericRoots) NumericSourceCause(z *util.IntegerManager) error {
	if r == nil || r.integer != z {
		return nil
	}
	c := r.caller
	if q := c.factorFrontend; q != nil && q.actual() && q.invocation == c.factorFrontendInvocation {
		return q.numericSourceCause()
	}
	if c.divides != nil {
		return numericIntegerManager_dividesNumericSourceCause(z, c.divides)
	}
	if c.scalarExact != nil {
		return numericIntegerManager_scalarExactNumericSourceCause(z, c.scalarExact)
	}
	if c.exactDiv != nil {
		return numericIntegerManager_exactDivNumericSourceCause(z, c.exactDiv)
	}
	if c.somNormalize != nil {
		return numericIntegerManager_somNormalizeNumericSourceCause(z, c.somNormalize)
	}
	if c.normalization != nil {
		return numericIntegerManager_normalizationNumericSourceCause(z, c.normalization)
	}
	return nil
}

// Frontend NUM receipts authorize their original modular owner.
func (r *polynomialNumericRoots) NumericModularOwner(m *util.IntegerModularManager) bool {
	if q := r.caller.factorFrontend; q != nil {
		return q.service != nil && q.service.manager != nil && q.service.manager.modular == m
	}
	return true
}
