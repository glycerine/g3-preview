// Source: src/math/realclosure/realclosure.cpp:3637-3702, 3930-4025,
// and 4031-4069 for rational coefficients and unbounded intervals.
package realclosure

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
)

var errRealClosureSequencePending = errors.New("realclosure: polynomial sequence cleanup pending")

func (s *realValueStore) sequenceReady() error {
	if s == nil || s.numbers == nil {
		return arena.ErrInvalidSlot
	}
	if s.pendingSequence != nil || len(s.pendingSturmCoefficients) != 0 {
		return errRealClosureSequencePending
	}
	return nil
}

func (s *realValueStore) closePolynomialSequence(rows []realPolynomial) error {
	if s == nil {
		return arena.ErrInvalidSlot
	}
	if s.pendingSequence != nil && (len(rows) == 0 || &rows[0] != &s.pendingSequence[0]) {
		return errRealClosureSequencePending
	}
	for i := range rows {
		if err := s.resetPolynomial(&rows[i]); err != nil {
			s.pendingSequence = rows
			return err
		}
	}
	s.pendingSequence = nil
	return nil
}

// releaseSturmCoefficients owns a completed temporary coefficient slice and
// retains its unconsumed suffix when one value's destructor refuses.
func (s *realValueStore) releaseSturmCoefficients(values []*realValue) error {
	for i, value := range values {
		if err := s.release(value); err != nil {
			s.pendingSturmCoefficients = append(s.pendingSturmCoefficients, values[i:])
			return err
		}
	}
	return nil
}

// releaseSturmScratch retains a standalone temporary when its final release
// refuses; it has no coefficient slice to carry it into later cleanup.
func (s *realValueStore) releaseSturmScratch(value *realValue) error {
	if err := s.release(value); err != nil {
		s.pendingSturmCoefficients = append(s.pendingSturmCoefficients, []*realValue{value})
		return err
	}
	return nil
}

// cleanupPendingPolynomialSequence finishes younger coefficient and scratch
// groups before the older retained row sequence.
func (s *realValueStore) cleanupPendingPolynomialSequence() error {
	if s == nil {
		return arena.ErrInvalidSlot
	}
	for len(s.pendingSturmCoefficients) != 0 {
		group := s.pendingSturmCoefficients[0]
		for i, value := range group {
			if err := s.release(value); err != nil {
				s.pendingSturmCoefficients[0] = group[i:]
				return err
			}
		}
		s.pendingSturmCoefficients[0] = nil
		s.pendingSturmCoefficients = s.pendingSturmCoefficients[1:]
	}
	s.pendingSturmCoefficients = nil
	if s.pendingSequence == nil {
		return nil
	}
	return s.closePolynomialSequence(s.pendingSequence)
}

func (s *realValueStore) sturmSequenceRational(first, second []*realValue, usePseudo bool) ([]realPolynomial, error) {
	if err := s.sequenceReady(); err != nil {
		return nil, err
	}
	if len(first) == 0 || len(second) == 0 {
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
			_ = s.releaseSturmCoefficients(remainder)
			_ = s.closePolynomialSequence(rows)
			return nil, err
		}
		if err := s.releaseSturmCoefficients(remainder); err != nil {
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
	if err := s.sequenceReady(); err != nil {
		return 0, err
	}
	derivative, err := s.derivativeRationalPolynomial(p)
	if err != nil {
		return 0, err
	}
	if len(derivative) == 0 {
		return 0, s.releaseSturmCoefficients(derivative)
	}
	rows, err := s.sturmSequenceRational(p, derivative, usePseudo)
	releaseErr := s.releaseSturmCoefficients(derivative)
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
	if err := s.sequenceReady(); err != nil {
		return 0, err
	}
	derivative, err := s.derivativeRationalPolynomial(p)
	if err != nil {
		return 0, err
	}
	if len(derivative) == 0 {
		return 0, s.releaseSturmCoefficients(derivative)
	}
	second, err := s.multiplyRationalPolynomials(derivative, q)
	releaseErr := s.releaseSturmCoefficients(derivative)
	if err != nil {
		return 0, err
	}
	if releaseErr != nil {
		_ = s.releaseSturmCoefficients(second)
		return 0, releaseErr
	}
	if len(second) == 0 {
		return 0, s.releaseSturmCoefficients(second)
	}
	rows, err := s.sturmSequenceRational(p, second, usePseudo)
	releaseErr = s.releaseSturmCoefficients(second)
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
