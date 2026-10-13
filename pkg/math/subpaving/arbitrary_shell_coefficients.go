// Source: src/math/subpaving/subpaving.cpp:85-136, concrete context_mpf_wrapper
// coefficient conversion. The enclosing context_mpf graph remains separate.
package subpaving

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

var ErrArbitraryShellInexactInteger = errors.New("subpaving: integer coefficient cannot be represented exactly as arbitrary float")

// ArbitraryShellCoefficients owns the source wrapper's constant, coefficient
// numerals and rational comparison scratch. Its managers are borrowed.
type ArbitraryShellCoefficients struct {
	core         *numeral.ArbitraryFloatManager
	numbers      *numeral.ArbitraryFloatNumeralManager
	rational     *numeral.RationalManager
	constant     numeral.ArbitraryFloat
	coefficients []numeral.ArbitraryFloat
	used         int
	before, back numeral.Rational
	closeStage   uint8
	closeIndex   int
	closed       bool
}

func NewArbitraryShellCoefficients(core *numeral.ArbitraryFloatManager, numbers *numeral.ArbitraryFloatNumeralManager) (*ArbitraryShellCoefficients, error) {
	if core == nil || numbers == nil {
		return nil, arena.ErrInvalidSlot
	}
	q, err := core.BorrowRationalManager()
	if err != nil {
		return nil, err
	}
	return &ArbitraryShellCoefficients{core: core, numbers: numbers, rational: q}, nil
}

func (w *ArbitraryShellCoefficients) open() error {
	if w == nil || w.core == nil || w.numbers == nil || w.rational == nil {
		return arena.ErrInvalidSlot
	}
	if w.closed {
		return arena.ErrClosed
	}
	_, err := w.core.BorrowRationalManager()
	return err
}

func (w *ArbitraryShellCoefficients) intToArbitrary(value *numeral.Int, out *numeral.ArbitraryFloat) error {
	if err := w.rational.SetInteger(&w.before, value); err != nil {
		return err
	}
	if err := w.numbers.SetRational(out, &w.before); err != nil {
		return err
	}
	if err := w.core.ToRational(out, w.rational, &w.back); err != nil {
		return err
	}
	exact, err := w.rational.Equal(&w.before, &w.back)
	if err != nil {
		return err
	}
	if !exact {
		return ErrArbitraryShellInexactInteger
	}
	return nil
}

// PrepareSum preserves the source's logical reserve and coefficient-first
// conversion order. The caller borrows the returned values until the next
// mutation or Close; the eventual context_mpf sum owns its own copies.
func (w *ArbitraryShellCoefficients) PrepareSum(constant *numeral.Int, coefficients []*numeral.Int) (*numeral.ArbitraryFloat, []numeral.ArbitraryFloat, error) {
	if err := w.open(); err != nil {
		return nil, nil, err
	}
	if constant == nil {
		return nil, nil, arena.ErrInvalidSlot
	}
	if len(coefficients) > len(w.coefficients) {
		w.coefficients = append(w.coefficients, make([]numeral.ArbitraryFloat, len(coefficients)-len(w.coefficients))...)
	}
	w.used = len(coefficients)
	for i, coefficient := range coefficients {
		if err := w.intToArbitrary(coefficient, &w.coefficients[i]); err != nil {
			return nil, nil, err
		}
	}
	if err := w.intToArbitrary(constant, &w.constant); err != nil {
		return nil, nil, err
	}
	return &w.constant, w.coefficients[:w.used], nil
}

// PrepareInequality chooses directed rounding before converting the bound.
// The returned scalar is borrowed until the next operation or Close.
func (w *ArbitraryShellCoefficients) PrepareInequality(value *numeral.Rational, lower bool) (*numeral.ArbitraryFloat, error) {
	if err := w.open(); err != nil {
		return nil, err
	}
	w.numbers.SetRounding(!lower)
	if err := w.numbers.SetRational(&w.constant, value); err != nil {
		return nil, err
	}
	return &w.constant, nil
}

// Close follows reverse member order: back, before, coefficient vector,
// constant. A stopped cleanup resumes at its failing member.
func (w *ArbitraryShellCoefficients) Close() error {
	if w == nil || w.core == nil || w.numbers == nil || w.rational == nil {
		return arena.ErrInvalidSlot
	}
	if w.closed {
		return nil
	}
	if w.closeStage == 0 {
		if err := w.rational.Del(&w.back); err != nil {
			return err
		}
		w.closeStage++
	}
	if w.closeStage == 1 {
		if err := w.rational.Del(&w.before); err != nil {
			return err
		}
		w.closeStage++
	}
	if w.closeStage == 2 {
		for w.closeIndex < len(w.coefficients) {
			if err := w.core.Del(&w.coefficients[w.closeIndex]); err != nil {
				return err
			}
			w.closeIndex++
		}
		w.closeStage++
	}
	if err := w.core.Del(&w.constant); err != nil {
		return err
	}
	w.coefficients = nil
	w.closed = true
	return nil
}
