// Source: src/math/realclosure/realclosure.cpp:3637-3702, 3930-4025,
// and 4031-4069 for rational coefficients and unbounded intervals.
package realclosure

import "github.com/glycerine/g3/pkg/arena"

func (s *realValueStore) closePolynomialSequence(rows []realPolynomial) error {
	for i := range rows {
		if err := s.resetPolynomial(&rows[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *realValueStore) sturmSequenceRational(first, second []*realValue, usePseudo bool) ([]realPolynomial, error) {
	if s == nil || len(first) == 0 || len(second) == 0 {
		return nil, arena.ErrInvalidSlot
	}
	rows := make([]realPolynomial, 2)
	if err := s.setPolynomial(&rows[0], first); err != nil {
		return nil, err
	}
	if err := s.setPolynomial(&rows[1], second); err != nil {
		_ = s.closePolynomialSequence(rows)
		return nil, err
	}
	for {
		previous := rows[len(rows)-2].coefficients
		current := rows[len(rows)-1].coefficients
		var remainder []*realValue
		var err error
		if usePseudo {
			remainder, err = s.signedPseudoRemainderRational(previous, current)
		} else {
			remainder, err = s.signedRemainderRational(previous, current)
		}
		if err != nil {
			_ = s.closePolynomialSequence(rows)
			return nil, err
		}
		if len(remainder) == 0 {
			return rows, nil
		}
		var next realPolynomial
		if err := s.setPolynomial(&next, remainder); err != nil {
			_ = s.releaseCoefficients(remainder)
			_ = s.closePolynomialSequence(rows)
			return nil, err
		}
		if err := s.releaseCoefficients(remainder); err != nil {
			_ = s.resetPolynomial(&next)
			_ = s.closePolynomialSequence(rows)
			return nil, err
		}
		rows = append(rows, next)
	}
}

func (s *realValueStore) signVariationsAtInfinity(rows []realPolynomial, plus bool) (int, error) {
	previous, variations := int32(0), 0
	for _, row := range rows {
		if len(row.coefficients) == 0 || row.coefficients[len(row.coefficients)-1] == nil {
			return 0, arena.ErrInvalidSlot
		}
		sign, err := s.numbers.Sign(&row.coefficients[len(row.coefficients)-1].rational)
		if err != nil {
			return 0, err
		}
		if sign > 0 {
			sign = 1
		} else if sign < 0 {
			sign = -1
		}
		if !plus && (len(row.coefficients)-1)%2 != 0 {
			sign = -sign
		}
		if sign != 0 {
			if previous != 0 && sign != previous {
				variations++
			}
			previous = sign
		}
	}
	return variations, nil
}

func (s *realValueStore) tarskiFromUnboundedSequence(rows []realPolynomial) (int, error) {
	lower, err := s.signVariationsAtInfinity(rows, false)
	if err != nil {
		return 0, err
	}
	upper, err := s.signVariationsAtInfinity(rows, true)
	if err != nil {
		return 0, err
	}
	return lower - upper, nil
}

// rootCountUnboundedRational follows TaQ_1 for the whole real line.
func (s *realValueStore) rootCountUnboundedRational(p []*realValue, usePseudo bool) (int, error) {
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
	count, err := s.tarskiFromUnboundedSequence(rows)
	closeErr := s.closePolynomialSequence(rows)
	if err != nil {
		return 0, err
	}
	return count, closeErr
}

// tarskiUnboundedRational follows TaQ(p,q) for an unbounded interval: it
// starts at p and p'*q, then subtracts the endpoint sign variations.
func (s *realValueStore) tarskiUnboundedRational(p, q []*realValue, usePseudo bool) (int, error) {
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
	query, err := s.tarskiFromUnboundedSequence(rows)
	closeErr := s.closePolynomialSequence(rows)
	if err != nil {
		return 0, err
	}
	return query, closeErr
}
