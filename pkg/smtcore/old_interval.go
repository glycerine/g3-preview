// Source: src/smt/old_interval.cpp and src/smt/old_interval.h retained
// rational interval operations used by nonlinear arithmetic.
package smtcore

import (
	"math/big"

	"github.com/glycerine/g3/pkg/ast"
)

type smtOldInterval struct {
	manager              *smtArithmeticBoundDependencyManager
	lower, upper         smtExtendedRational
	lowerOpen, upperOpen bool
	lowerDep, upperDep   *smtArithmeticBoundDependency
}

func newSMTOldInterval(manager *smtArithmeticBoundDependencyManager) (*smtOldInterval, error) {
	if manager == nil {
		return nil, ast.ErrASTDomain
	}
	return &smtOldInterval{manager: manager, lower: smtInfiniteExtended(false), upper: smtInfiniteExtended(true), lowerOpen: true, upperOpen: true}, nil
}

func newSMTOldIntervalBounds(manager *smtArithmeticBoundDependencyManager, lower *big.Rat, lowerOpen bool, lowerDep *smtArithmeticBoundDependency, upper *big.Rat, upperOpen bool, upperDep *smtArithmeticBoundDependency) (*smtOldInterval, error) {
	if manager == nil || lower == nil || upper == nil || lower.Cmp(upper) > 0 || lower.Cmp(upper) == 0 && lowerOpen && upperOpen {
		return nil, ast.ErrASTDomain
	}
	return &smtOldInterval{manager: manager, lower: smtFiniteExtended(lower), upper: smtFiniteExtended(upper), lowerOpen: lowerOpen, upperOpen: upperOpen, lowerDep: lowerDep, upperDep: upperDep}, nil
}

func newSMTOldIntervalPoint(manager *smtArithmeticBoundDependencyManager, value *big.Rat, lowerDep, upperDep *smtArithmeticBoundDependency) (*smtOldInterval, error) {
	return newSMTOldIntervalBounds(manager, value, false, lowerDep, value, false, upperDep)
}

func newSMTOldIntervalHalfBound(manager *smtArithmeticBoundDependencyManager, value *big.Rat, open, lower bool, dependency *smtArithmeticBoundDependency) (*smtOldInterval, error) {
	result, err := newSMTOldInterval(manager)
	if err != nil || value == nil {
		if err == nil {
			err = ast.ErrASTDomain
		}
		return nil, err
	}
	if lower {
		result.lower, result.lowerOpen, result.lowerDep = smtFiniteExtended(value), open, dependency
	} else {
		result.upper, result.upperOpen, result.upperDep = smtFiniteExtended(value), open, dependency
	}
	return result, nil
}

func (i *smtOldInterval) clone() *smtOldInterval {
	result := *i
	result.lower.set(&i.lower)
	result.upper.set(&i.upper)
	return &result
}
func (i *smtOldInterval) valid(other *smtOldInterval) bool {
	return i != nil && other != nil && i.manager != nil && i.manager == other.manager
}

func (i *smtOldInterval) add(other *smtOldInterval) error {
	if !i.valid(other) {
		return ast.ErrASTDomain
	}
	if err := i.lower.add(&other.lower); err != nil {
		return err
	}
	if err := i.upper.add(&other.upper); err != nil {
		return err
	}
	i.lowerOpen = i.lowerOpen || other.lowerOpen
	i.upperOpen = i.upperOpen || other.upperOpen
	if i.lower.infinite() {
		i.lowerDep = nil
	} else {
		i.lowerDep = i.manager.Join(i.lowerDep, other.lowerDep)
	}
	if i.upper.infinite() {
		i.upperDep = nil
	} else {
		i.upperDep = i.manager.Join(i.upperDep, other.upperDep)
	}
	return nil
}

func (i *smtOldInterval) neg() {
	i.lower, i.upper = i.upper, i.lower
	i.lowerOpen, i.upperOpen = i.upperOpen, i.lowerOpen
	i.lowerDep, i.upperDep = i.upperDep, i.lowerDep
	i.lower.neg()
	i.upper.neg()
}
func (i *smtOldInterval) sub(other *smtOldInterval) error {
	if !i.valid(other) {
		return ast.ErrASTDomain
	}
	copy := other.clone()
	copy.neg()
	return i.add(copy)
}

func (i *smtOldInterval) containsZero() bool {
	return i != nil && (i.lower.sign() < 0 || i.lower.zero() && !i.lowerOpen) && (i.upper.sign() > 0 || i.upper.zero() && !i.upperOpen)
}
func (i *smtOldInterval) contains(value *big.Rat) bool {
	if i == nil || value == nil {
		return false
	}
	if !i.lower.infinite() {
		c := value.Cmp(&i.lower.value)
		if c < 0 || c == 0 && i.lowerOpen {
			return false
		}
	}
	if !i.upper.infinite() {
		c := value.Cmp(&i.upper.value)
		if c > 0 || c == 0 && i.upperOpen {
			return false
		}
	}
	return true
}
func (i *smtOldInterval) zero() bool { return i != nil && i.lower.zero() && i.upper.zero() }
func (i *smtOldInterval) positiveOne() bool {
	return i.lower.sign() > 0 || i.lower.zero() && i.lowerOpen
}
func (i *smtOldInterval) negativeOne() bool {
	return i.upper.sign() < 0 || i.upper.zero() && i.upperOpen
}

func (i *smtOldInterval) inverse() error {
	if i == nil || i.manager == nil || i.containsZero() {
		return ast.ErrASTDomain
	}
	if i.positiveOne() {
		newLower := i.upper
		if err := newLower.inverse(); err != nil {
			return err
		}
		newUpper := smtInfiniteExtended(true)
		if !i.lower.zero() {
			newUpper = i.lower
			if err := newUpper.inverse(); err != nil {
				return err
			}
		}
		i.lower, i.upper = newLower, newUpper
		i.lowerOpen, i.upperOpen = i.upperOpen, i.lowerOpen
		newUpperDep := i.lowerDep
		i.lowerDep = i.manager.Join(i.lowerDep, i.upperDep)
		i.upperDep = newUpperDep
		return nil
	}
	if i.negativeOne() {
		newUpper := i.lower
		if err := newUpper.inverse(); err != nil {
			return err
		}
		newLower := smtInfiniteExtended(false)
		if !i.upper.zero() {
			newLower = i.upper
			if err := newLower.inverse(); err != nil {
				return err
			}
		}
		i.lower, i.upper = newLower, newUpper
		i.lowerOpen, i.upperOpen = i.upperOpen, i.lowerOpen
		newLowerDep := i.upperDep
		i.upperDep = i.manager.Join(i.lowerDep, i.upperDep)
		i.lowerDep = newLowerDep
		return nil
	}
	return ast.ErrASTDomain
}

func (i *smtOldInterval) power(exponent uint32) {
	if i == nil || exponent == 1 {
		return
	}
	if exponent%2 != 0 {
		i.lower.power(exponent)
		i.upper.power(exponent)
		return
	}
	if i.lower.sign() > 0 {
		i.lower.power(exponent)
		i.upper.power(exponent)
		if !i.upper.infinite() {
			i.upperDep = i.manager.Join(i.lowerDep, i.upperDep)
		}
		return
	}
	if i.upper.sign() < 0 {
		i.lower, i.upper = i.upper, i.lower
		i.lowerOpen, i.upperOpen = i.upperOpen, i.lowerOpen
		i.lowerDep, i.upperDep = i.upperDep, i.lowerDep
		i.lower.power(exponent)
		i.upper.power(exponent)
		if !i.upper.infinite() {
			i.upperDep = i.manager.Join(i.lowerDep, i.upperDep)
		}
		return
	}
	i.lower.power(exponent)
	i.upper.power(exponent)
	if i.lower.cmp(&i.upper) > 0 || i.lower.equal(&i.upper) && !i.lowerOpen && i.upperOpen {
		i.upper, i.upperOpen = i.lower, i.lowerOpen
	}
	if !i.upper.infinite() {
		i.upperDep = i.manager.Join(i.lowerDep, i.upperDep)
	}
	i.lower = smtFiniteExtended(nil)
	i.lowerOpen, i.lowerDep = false, nil
}

func (i *smtOldInterval) String() string {
	if i == nil {
		return "<nil>"
	}
	left, right := "[", "]"
	if i.lowerOpen {
		left = "("
	}
	if i.upperOpen {
		right = ")"
	}
	return left + i.lower.String() + ", " + i.upper.String() + right
}
