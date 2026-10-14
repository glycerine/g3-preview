// Source: src/math/realclosure/realclosure.cpp:3773-3920 and 3930-4025,
// rational-coefficient exact evaluation at finite binary-rational bounds.
package realclosure

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func normalizedRationalSign(raw int32) int32 {
	if raw < 0 {
		return -1
	}
	if raw > 0 {
		return 1
	}
	return 0
}

// signAtBinaryBound follows the source's zero shortcut and exact Horner
// evaluation for rational coefficients. The source may first try interval
// approximations; those nonrational/refinement branches remain open.
func (s *realValueStore) signAtBinaryBound(coefficients []*realValue, bounds *numeral.BinaryRationalManager, bound *numeral.BinaryRational) (int32, error) {
	if s == nil || s.numbers == nil || bounds == nil || bound == nil {
		return 0, arena.ErrInvalidSlot
	}
	if len(coefficients) == 0 {
		return 0, nil
	}
	for _, value := range coefficients {
		if !s.rationalOperand(value) {
			return 0, arena.ErrInvalidSlot
		}
	}
	if coefficients[len(coefficients)-1] == nil {
		return 0, arena.ErrInvalidSlot
	}
	if len(coefficients) == 1 {
		raw, err := s.numbers.Sign(&coefficients[0].rational)
		return normalizedRationalSign(raw), err
	}
	zero, err := bounds.IsZero(bound)
	if err != nil {
		return 0, err
	}
	if zero {
		if coefficients[0] == nil {
			return 0, nil
		}
		raw, err := s.numbers.Sign(&coefficients[0].rational)
		return normalizedRationalSign(raw), err
	}
	x := numeral.NewRational()
	defer func() { _ = s.numbers.Del(&x) }()
	if err := bounds.ToRational(s.numbers, bound, &x); err != nil {
		return 0, err
	}
	acc := numeral.NewRational()
	defer func() { _ = s.numbers.Del(&acc) }()
	if err := s.numbers.Mul(&coefficients[len(coefficients)-1].rational, &x, &acc); err != nil {
		return 0, err
	}
	for i := len(coefficients) - 2; i >= 0; i-- {
		if coefficients[i] != nil {
			next := numeral.NewRational()
			if err := s.numbers.Add(&acc, &coefficients[i].rational, &next); err != nil {
				_ = s.numbers.Del(&next)
				return 0, err
			}
			if err := s.numbers.Del(&acc); err != nil {
				_ = s.numbers.Del(&next)
				return 0, err
			}
			acc = next
		}
		if i > 0 {
			next := numeral.NewRational()
			if err := s.numbers.Mul(&acc, &x, &next); err != nil {
				_ = s.numbers.Del(&next)
				return 0, err
			}
			if err := s.numbers.Del(&acc); err != nil {
				_ = s.numbers.Del(&next)
				return 0, err
			}
			acc = next
		}
	}
	raw, err := s.numbers.Sign(&acc)
	return normalizedRationalSign(raw), err
}

func (s *realValueStore) signVariationsAtBinaryEndpoint(rows []realPolynomial, interval *BinaryInterval, lower bool) (int, error) {
	if err := interval.open(); err != nil {
		return 0, err
	}
	if lower && interval.lowerInf {
		return s.signVariationsAtInfinity(rows, false)
	}
	if !lower && interval.upperInf {
		return s.signVariationsAtInfinity(rows, true)
	}
	bound := &interval.upper
	if lower {
		bound = &interval.lower
	}
	previous, variations := int32(0), 0
	for _, row := range rows {
		sign, err := s.signAtBinaryBound(row.coefficients, interval.numbers, bound)
		if err != nil {
			return 0, err
		}
		if sign == 0 {
			continue
		}
		if previous != 0 && sign != previous {
			variations++
		}
		previous = sign
	}
	return variations, nil
}

func (s *realValueStore) tarskiFromBinaryIntervalSequence(rows []realPolynomial, interval *BinaryInterval) (int, error) {
	lower, err := s.signVariationsAtBinaryEndpoint(rows, interval, true)
	if err != nil {
		return 0, err
	}
	upper, err := s.signVariationsAtBinaryEndpoint(rows, interval, false)
	if err != nil {
		return 0, err
	}
	return lower - upper, nil
}

// rootCountBinaryIntervalRational follows TaQ_1 across a binary-rational
// interval. As in the source, endpoint openness does not change the query.
func (s *realValueStore) rootCountBinaryIntervalRational(p []*realValue, interval *BinaryInterval, usePseudo bool) (int, error) {
	if err := s.sequenceReady(); err != nil {
		return 0, err
	}
	if err := interval.open(); err != nil {
		return 0, err
	}
	derivative, err := s.derivativeRationalPolynomial(p)
	if err != nil {
		return 0, err
	}
	if len(derivative) == 0 {
		return 0, s.releaseCoefficients(derivative)
	}
	rows, err := s.sturmSequenceRational(p, derivative, usePseudo)
	releaseErr := s.releaseCoefficients(derivative)
	if err != nil {
		return 0, err
	}
	if releaseErr != nil {
		_ = s.closePolynomialSequence(rows)
		return 0, releaseErr
	}
	count, err := s.tarskiFromBinaryIntervalSequence(rows, interval)
	closeErr := s.closePolynomialSequence(rows)
	if err != nil {
		return 0, err
	}
	return count, closeErr
}

func (s *realValueStore) tarskiBinaryIntervalRational(p, q []*realValue, interval *BinaryInterval, usePseudo bool) (int, error) {
	if err := s.sequenceReady(); err != nil {
		return 0, err
	}
	if err := interval.open(); err != nil {
		return 0, err
	}
	derivative, err := s.derivativeRationalPolynomial(p)
	if err != nil {
		return 0, err
	}
	if len(derivative) == 0 {
		return 0, s.releaseCoefficients(derivative)
	}
	second, err := s.multiplyRationalPolynomials(derivative, q)
	releaseErr := s.releaseCoefficients(derivative)
	if err != nil {
		return 0, err
	}
	if releaseErr != nil {
		_ = s.releaseCoefficients(second)
		return 0, releaseErr
	}
	if len(second) == 0 {
		return 0, s.releaseCoefficients(second)
	}
	rows, err := s.sturmSequenceRational(p, second, usePseudo)
	releaseErr = s.releaseCoefficients(second)
	if err != nil {
		return 0, err
	}
	if releaseErr != nil {
		_ = s.closePolynomialSequence(rows)
		return 0, releaseErr
	}
	query, err := s.tarskiFromBinaryIntervalSequence(rows, interval)
	closeErr := s.closePolynomialSequence(rows)
	if err != nil {
		return 0, err
	}
	return query, closeErr
}
