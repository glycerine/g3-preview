// Source: src/math/subpaving/subpaving.cpp:198-250, concrete context_mpff_wrapper.
package subpaving

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

var ErrFastShellInexactInteger = errors.New("subpaving: integer coefficient cannot be represented exactly as fast float")

// FastShellCoefficients owns the MPFF wrapper's scalar, coefficient vector,
// and exact integer comparison scratch. Its context and managers are borrowed.
type FastShellCoefficients struct {
	self         *FastShellCoefficients
	context      *FastContext
	numbers      *numeral.FastFloatManager
	rational     *numeral.RationalManager
	constant     numeral.FastFloat
	coefficients []numeral.FastFloat
	z1, z2       numeral.Int
	closeStage   uint8
	closeIndex   int
	closed       bool
}

func NewFastShellCoefficients(context *FastContext, rational *numeral.RationalManager) (*FastShellCoefficients, error) {
	if context == nil || rational == nil || rational.Z == nil {
		return nil, arena.ErrInvalidSlot
	}
	if err := context.open(); err != nil {
		return nil, err
	}
	w := &FastShellCoefficients{context: context, numbers: context.numbers, rational: rational}
	w.self = w
	return w, nil
}

func (w *FastShellCoefficients) open() error {
	if w == nil || w.self != w || w.context == nil || w.numbers == nil || w.rational == nil || w.rational.Z == nil {
		return arena.ErrInvalidSlot
	}
	if w.closed {
		return arena.ErrClosed
	}
	if err := w.context.open(); err != nil {
		return err
	}
	if w.context.numbers != w.numbers {
		return arena.ErrInvalidSlot
	}
	return nil
}

// intToFast retains the source's copy, conversion, integer round trip and
// exact equality order. z1 and z2 remain owned scratch across calls.
func (w *FastShellCoefficients) intToFast(value *numeral.Int, out *numeral.FastFloat) error {
	if err := w.rational.Z.Copy(&w.z1, value); err != nil {
		return err
	}
	if err := w.numbers.SetInteger(out, w.rational.Z, &w.z1); err != nil {
		return err
	}
	if err := w.numbers.ToInteger(out, w.rational.Z, &w.z2); err != nil {
		return err
	}
	exact, err := w.rational.Z.Equal(&w.z1, &w.z2)
	if err != nil {
		return err
	}
	if !exact {
		return ErrFastShellInexactInteger
	}
	return nil
}

// PrepareSum converts every coefficient before the constant. The returned
// values are borrowed until the next wrapper mutation or Close.
func (w *FastShellCoefficients) PrepareSum(constant *numeral.Int, coefficients []*numeral.Int) (*numeral.FastFloat, []numeral.FastFloat, error) {
	if err := w.open(); err != nil {
		return nil, nil, err
	}
	if constant == nil {
		return nil, nil, arena.ErrInvalidSlot
	}
	if len(coefficients) > len(w.coefficients) {
		w.coefficients = append(w.coefficients, make([]numeral.FastFloat, len(coefficients)-len(w.coefficients))...)
	}
	for i, coefficient := range coefficients {
		if err := w.intToFast(coefficient, &w.coefficients[i]); err != nil {
			return nil, nil, err
		}
	}
	if err := w.intToFast(constant, &w.constant); err != nil {
		return nil, nil, err
	}
	return &w.constant, w.coefficients[:len(coefficients)], nil
}

func (w *FastShellCoefficients) MkVar(integer bool) (uint32, error) {
	if err := w.open(); err != nil {
		return 0, err
	}
	return w.context.NewVariable(integer)
}

func (w *FastShellCoefficients) MkMonomial(powers []FastPower) (uint32, error) {
	if err := w.open(); err != nil {
		return 0, err
	}
	return w.context.DefineMonomial(powers)
}

func (w *FastShellCoefficients) MkSum(constant *numeral.Int, coefficients []*numeral.Int, variables []uint32) (uint32, error) {
	if len(coefficients) != len(variables) {
		return 0, arena.ErrInvalidSlot
	}
	_, _, err := w.PrepareSum(constant, coefficients)
	if err != nil {
		return 0, err
	}
	terms := make([]FastPolynomialTerm, len(variables))
	for i, variable := range variables {
		terms[i] = FastPolynomialTerm{Variable: variable, Coefficient: &w.coefficients[i]}
	}
	return w.context.DefinePolynomial(&w.constant, terms)
}

func (w *FastShellCoefficients) PrepareInequality(value *numeral.Rational, lower bool) (*numeral.FastFloat, error) {
	if err := w.open(); err != nil {
		return nil, err
	}
	if err := w.numbers.SetRounding(!lower); err != nil {
		return nil, err
	}
	if err := w.numbers.SetRational(&w.constant, w.rational, value); err != nil {
		return nil, err
	}
	return &w.constant, nil
}

func (w *FastShellCoefficients) MkInequality(variable uint32, value *numeral.Rational, lower, open bool) (*FastInequality, error) {
	bound, err := w.PrepareInequality(value, lower)
	if err != nil {
		return nil, err
	}
	return w.context.NewInequality(variable, bound, lower, open)
}

func (w *FastShellCoefficients) AddClause(atoms []*FastInequality) (*FastClause, error) {
	if err := w.open(); err != nil {
		return nil, err
	}
	return w.context.AddClause(atoms, false, true)
}

// Close unwinds the source members in reverse order. A refused release keeps
// its member for a later retry; the borrowed context is closed by its owner.
func (w *FastShellCoefficients) Close() error {
	if w == nil || w.self != w || w.numbers == nil || w.rational == nil {
		return arena.ErrInvalidSlot
	}
	if w.closed {
		return nil
	}
	if w.closeStage == 0 {
		if err := w.rational.Z.Del(&w.z2); err != nil {
			return err
		}
		w.closeStage = 1
	}
	if w.closeStage == 1 {
		if err := w.rational.Z.Del(&w.z1); err != nil {
			return err
		}
		w.closeStage = 2
	}
	if w.closeStage == 2 {
		for w.closeIndex < len(w.coefficients) {
			if err := w.numbers.Del(&w.coefficients[w.closeIndex]); err != nil {
				return err
			}
			w.closeIndex++
		}
		w.coefficients = nil
		w.closeStage = 3
	}
	if w.closeStage == 3 {
		if err := w.numbers.Del(&w.constant); err != nil {
			return err
		}
		w.closeStage = 4
	}
	w.closed = true
	return nil
}
