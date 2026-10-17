// Source: src/math/polynomial/algebraic_numbers.cpp:1905-1924,2721-2730,
// 2748-2769,2983-3040.
package polynomial

import (
	"errors"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// GetPolynomial copies the defining polynomial, without minimizing a root's
// polynomial. A rational n/d is represented by d*x-n; zero is represented by x.
func (m *AlgebraicRootCore) GetPolynomial(a *Algebraic, out UPolynomial) error {
	l, e := m.location(a)
	if e != nil {
		return e
	}
	if m.config.Rationals.Integers().MatchesCopyAllocationInvocation(&m.representationCopy) || m.config.Polynomials.modular.MatchesPNormalizeInvocation(&m.representationNormalize) || unumeral.NumericAllocationPending(&m.representationNeg.roots) {
		return ErrAlgebraicPending
	}
	p := m.config.Polynomials
	d, e := p.descriptor(out)
	if e != nil {
		return e
	}
	m.bounds.executing = true
	defer func() { m.bounds.executing = false }()
	if l != nil && l.kind == 1 {
		r, e := m.load(l)
		if e != nil {
			return e
		}
		v, e := m.sourcePolynomial(&r)
		if e != nil {
			return e
		}
		return m.copyDefiningCoefficients(&d.state, v)
	}
	if e = p.reserve(&d.state, 2); e != nil {
		return e
	}
	values := p.values(&d.state)
	z := m.config.Rationals.Integers()
	if l == nil {
		if e = z.SetInt32(&values[0], 0); e != nil {
			return e
		}
		if e = z.SetInt32(&values[1], 1); e != nil {
			return e
		}
	} else {
		r, e := m.load(l)
		if e != nil {
			return e
		}
		if e = z.CopyFromOwnerWithReceipt(&values[0], z, r.value.BorrowNumerator(), &m.representationCopy); e != nil {
			return e
		}
		if e = z.CopyFromOwnerWithReceipt(&values[1], z, r.value.BorrowDenominator(), &m.representationCopy); e != nil {
			return e
		}
		s := &m.representationNeg
		s.owner = m
		s.hooks.Policy = s
		context := unumeral.NumericAllocationContext{}.WithCaller(&s.hooks)
		if e = z.WithAllocationContext(context).Neg(&values[0]); e != nil {
			return e
		}
	}
	return p.setSize(&d.state, 2)
}

// These source overloads require an irrational root. The precision overloads
// below additionally accept basic values, including zero.
func (m *AlgebraicRootCore) GetLower(a *Algebraic, out *unumeral.BinaryRational) error {
	return m.copyBinaryBound(a, out, false)
}
func (m *AlgebraicRootCore) GetUpper(a *Algebraic, out *unumeral.BinaryRational) error {
	return m.copyBinaryBound(a, out, true)
}
func (m *AlgebraicRootCore) irrationalBoundRecord(a *Algebraic, upper bool) (algebraicRecord, error) {
	if e := m.check(); e != nil {
		return algebraicRecord{}, e
	}
	if polynomialSourceDebug {
		d := diagnostic.BorrowProcessDebugContext()
		enabled, e := diagnostic.SourceAssertionsEnabled(d)
		if e != nil {
			return algebraicRecord{}, e
		}
		if enabled {
			rational, e := m.IsRational(a)
			if e != nil {
				return algebraicRecord{}, e
			}
			if rational {
				line := int32(2988)
				if upper {
					line = 3015
				}
				if e = diagnostic.DebugAssertionFailure(d, "src/math/polynomial/algebraic_numbers.cpp", line, "!is_rational(a)"); e != nil {
					return algebraicRecord{}, e
				}
			}
		}
	}
	l, e := m.location(a)
	if e != nil {
		return algebraicRecord{}, e
	}
	if l == nil || l.kind != 1 {
		return algebraicRecord{}, ErrAlgebraicOwner
	}
	return m.load(l)
}
func (m *AlgebraicRootCore) copyBinaryBound(a *Algebraic, out *unumeral.BinaryRational, upper bool) error {
	if e := m.check(); e != nil {
		return e
	}
	if m.bounds.remaining != 0 || m.config.Rationals.Integers().MatchesCopyAllocationInvocation(&m.bounds.copy) || m.bounds.conversion.pending(m) {
		return ErrAlgebraicPending
	}
	r, e := m.irrationalBoundRecord(a, upper)
	if e != nil {
		return e
	}
	bound := r.interval.BorrowLower()
	if upper {
		bound = r.interval.BorrowUpper()
	}
	m.bounds.executing = true
	defer func() { m.bounds.executing = false }()
	return m.config.Binary.CopyWithAllocationReceipt(out, bound, &m.bounds.copy)
}

type algebraicBoundScope struct {
	conversion algebraicBoundConversion
	copy       unumeral.IntegerCopyAllocationReceipt
	values     [2]unumeral.BinaryRational
	remaining  uint8
	executing  bool
}

func (m *AlgebraicRootCore) cleanupBounds() error { return m.finishBounds(true) }
func (m *AlgebraicRootCore) finishBounds(owning bool) error {
	s := &m.bounds
	if s.executing {
		return ErrAlgebraicPending
	}
	s.executing = true
	defer func() { s.executing = false }()
	if !owning && s.conversion.stop != nil {
		return s.conversion.stop
	}
	if owning {
		if e := s.conversion.cleanup(); e != nil {
			return e
		}
		if e := m.config.Rationals.Integers().CleanupCopyAllocationInvocation(&s.copy); e != nil {
			return e
		}
	}
	for s.remaining != 0 {
		if e := m.config.Binary.Del(&s.values[s.remaining-1]); e != nil {
			return e
		}
		s.remaining--
	}
	return nil
}

func (m *AlgebraicRootCore) GetLowerRational(a *Algebraic, out *unumeral.Rational) error {
	return m.rationalBound(a, out, false, false, 0)
}
func (m *AlgebraicRootCore) GetUpperRational(a *Algebraic, out *unumeral.Rational) error {
	return m.rationalBound(a, out, true, false, 0)
}
func (m *AlgebraicRootCore) GetLowerRationalAtPrecision(a *Algebraic, out *unumeral.Rational, precision uint32) error {
	return m.rationalBound(a, out, false, true, precision)
}
func (m *AlgebraicRootCore) GetUpperRationalAtPrecision(a *Algebraic, out *unumeral.Rational, precision uint32) error {
	return m.rationalBound(a, out, true, true, precision)
}

func (m *AlgebraicRootCore) rationalBound(a *Algebraic, out *unumeral.Rational, upper, refine bool, precision uint32) (cause error) {
	l, e := m.location(a)
	if e != nil {
		return e
	}
	s := &m.bounds
	if s.remaining != 0 || m.config.Rationals.Integers().MatchesCopyAllocationInvocation(&s.copy) || s.conversion.pending(m) {
		return ErrAlgebraicPending
	}
	r := algebraicRecord{value: unumeral.NewRational()}
	if refine {
		if l != nil {
			r, e = m.load(l)
		}
	} else {
		r, e = m.irrationalBoundRecord(a, upper)
	}
	if e != nil {
		return e
	}
	s.executing = true
	defer func() {
		s.executing = false
		cause = errors.Join(cause, m.finishBounds(false))
	}()
	if refine && (l == nil || l.kind == 0) {
		return m.config.Rationals.CopyWithAllocationReceipt(out, &r.value, &s.copy)
	}
	b := m.config.Binary
	s.values = [2]unumeral.BinaryRational{}
	s.remaining = 1
	if refine {
		s.remaining = 2
		if _, e = m.intervalAtPrecisionWithReceipt(&r, &s.values[0], &s.values[1], precision, &s.copy); e != nil {
			return e
		}
	} else {
		bound := r.interval.BorrowLower()
		if upper {
			bound = r.interval.BorrowUpper()
		}
		if e = b.CopyWithAllocationReceipt(&s.values[0], bound, &s.copy); e != nil {
			return e
		}
	}
	selected := &s.values[0]
	if upper && refine {
		selected = &s.values[1]
	}
	return m.boundToRational(selected, out)
}

func (m *AlgebraicRootCore) intervalAtPrecision(r *algebraicRecord, lower, upper *unumeral.BinaryRational, precision uint32) (bool, error) {
	return m.intervalAtPrecisionWithReceipt(r, lower, upper, precision, nil)
}
func (m *AlgebraicRootCore) intervalAtPrecisionWithReceipt(r *algebraicRecord, lower, upper *unumeral.BinaryRational, precision uint32, receipt *unumeral.IntegerCopyAllocationReceipt) (bool, error) {
	b := m.config.Binary
	if e := b.CopyWithAllocationReceipt(lower, r.interval.BorrowLower(), receipt); e != nil {
		return false, e
	}
	if e := b.CopyWithAllocationReceipt(upper, r.interval.BorrowUpper(), receipt); e != nil {
		return false, e
	}
	v, e := m.sourcePolynomial(r)
	if e != nil {
		return false, e
	}
	return m.config.Polynomials.RefineToPrecision(v, b, lower, upper, precision*4)
}

// Preserve reserve, ascending numeral SET and final size adjustment from
// upm().set, retaining the actual receiving manager and its normalization.
func (m *AlgebraicRootCore) copyDefiningCoefficients(out *upolyVectorState, v UPolynomialView) error {
	p := m.config.Polynomials
	if p.alias(v, out) {
		if polynomialSourceDebug && out.size != v.count {
			return ErrUPolynomialAssertion
		}
		return nil
	}
	if e := p.reserve(out, v.count); e != nil {
		return e
	}
	z := m.config.Rationals.Integers()
	for i := uint32(0); i < v.count; i++ {
		value, e := p.read(v, i)
		if e != nil {
			return e
		}
		if e = p.modular.CopyFromOwnerTracked(&p.values(out)[i], z, value, &m.representationCopy, &m.representationNormalize); e != nil {
			return e
		}
	}
	return p.setSize(out, v.count)
}
