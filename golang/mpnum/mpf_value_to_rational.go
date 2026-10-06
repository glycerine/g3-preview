// Source: src/util/mpf.cpp:1695-1711,1922-1945; src/util/mpq.h:696-710.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// ToRational follows the source copy/unpack/scale route. The output belongs to
// the explicitly supplied rational manager; it can have a different Integer
// descriptor namespace from the MPF manager.
func (m *ArbitraryFloatManager) ToRational(x *ArbitraryFloat, qm *RationalManager, out *Rational) (cause error) {
	if e := m.valueGuard(x); e != nil {
		return e
	}
	if qm == nil || qm.self != qm {
		return memory.ErrRuntimeOwner
	}
	if e := qm.checkSetter(out); e != nil {
		return e
	}
	// Actual lexical declarations: scoped_mpf a, then scoped_mpz n,d.
	s := &m.rationalConversion
	if s.active || s.executing || s.finishing {
		return ErrPolynomialPending
	}
	if m.rationalConversionInvocation == ^uint64(0) {
		return ErrPolynomialPending
	}
	m.rationalConversionInvocation++
	*s = arbitraryFloatRationalConversionScope{manager: m, outputManager: qm, invocation: m.rationalConversionInvocation, active: true, executing: true, constructed: 3}
	s.enterLifecycle()
	defer s.leaveLifecycle()
	a, n, d := &s.a, &s.n, &s.d
	defer func() {
		s.executing = false
		s.bodyCause = cause
		source := m.finishRationalConversionSource()
		cause = polynomialNewtonJoin(cause, source)
		s.recordHistory()
	}()
	s.numericPhase = 1
	if e := m.copyWithValueReceiptAndContext(a, x, &s.copy, s.allocationContext()); e != nil {
		return e
	}
	s.numericPhase = 2
	if e := m.unpackRationalNumeral(a, true, s.allocationContext()); e != nil {
		return e
	}
	s.numericPhase = 3
	if e := m.integer.copyFromOwnerWithAllocationContext(n, m.integer, &a.significand, nil, s.allocationContext()); e != nil {
		return e
	}
	if a.Sign() {
		if e := m.integer.negWithAllocationContext(n, s.allocationContext()); e != nil {
			return e
		}
	}
	s.numericPhase = 4
	two := NewInt(2)
	if e := m.integer.powerWithAllocationContext(&two, a.SBits()-1, d, nil, s.allocationContext()); e != nil {
		return e
	}
	s.numericPhase = 5
	if a.exponent >= 0 {
		if e := m.integer.shiftWithAllocationContext(n, uint32(a.exponent), n, true, s.allocationContext()); e != nil {
			return e
		}
	} else {
		// C++ signed unary negation has no defined INT64_MIN result. Refuse
		// this represented Go domain only at the reached source expression.
		if a.exponent == -1<<63 {
			return arena.ErrBounds
		}
		if e := m.integer.shiftWithAllocationContext(d, uint32(-a.exponent), d, true, s.allocationContext()); e != nil {
			return e
		}
	}
	s.numericPhase = 6
	return qm.setPartsFromIntegerOwnerWithAllocationContext(out, n, d, m.integer, s.outputAllocationContext())
}

// This is the reached private unpack body, including its literal special and
// zero behavior. The rewriter's finite-input check belongs to that caller.
func (m *ArbitraryFloatManager) unpackRationalNumeral(a *ArbitraryFloat, normalize bool, context integerAllocationInvocationContext) error {
	zero, e := m.IsZero(a)
	if e != nil || zero {
		return e
	}
	normal, e := m.IsNormal(a)
	if e != nil {
		return e
	}
	if normal {
		w, e := m.powerTwoWrapper(a.SBits()-1, false)
		if e != nil {
			return e
		}
		power, e := w.capture()
		if e != nil {
			return e
		}
		return m.integer.addSubWithAllocationContext(&a.significand, &power, &a.significand, false, context)
	}
	a.exponent, e = m.bottomExponent(a.EBits())
	if e != nil {
		return e
	}
	a.exponent++
	if !normalize {
		return nil
	}
	empty, e := m.integer.IsZero(&a.significand)
	if e != nil || empty {
		return e
	}
	w, e := m.powerTwoWrapper(a.SBits()-1, false)
	if e != nil {
		return e
	}
	power, e := w.capture()
	if e != nil {
		return e
	}
	for {
		cmp, e := m.integer.Compare(&power, &a.significand)
		if e != nil {
			return e
		}
		if cmp <= 0 {
			return nil
		}
		a.exponent--
		if e := m.integer.shiftWithAllocationContext(&a.significand, 1, &a.significand, true, context); e != nil {
			return e
		}
	}
}
