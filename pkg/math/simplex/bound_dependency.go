package simplex

import "math/big"

// BoundDependency is a weighted Boolean literal index supplied by the SMT
// arithmetic consumer. Tableau pivots preserve bounds and their dependencies.
type BoundDependency struct {
	Literal     int32
	Coefficient big.Rat
}

// ArithmeticBoundUse identifies an endpoint used by an arithmetic deduction.
// Its coefficient is relative to the normalized tableau row.
type ArithmeticBoundUse struct {
	Column      uint32
	Lower       bool
	Coefficient big.Rat
}

func cloneBoundDependencies(dependencies []BoundDependency) []BoundDependency {
	result := make([]BoundDependency, len(dependencies))
	for i := range dependencies {
		result[i].Literal = dependencies[i].Literal
		result[i].Coefficient.Set(&dependencies[i].Coefficient)
	}
	return result
}

// SetLowerWithDependencies installs a logical bound. An empty dependency
// list denotes an unconditional bound; ordinary SetLower leaves it unexplained.
func (s *Solver) SetLowerWithDependencies(column uint32, bound Infinitesimal, dependencies []BoundDependency) error {
	if err := s.SetLower(column, bound); err != nil {
		return err
	}
	x := &s.vars[column]
	x.lowerDependencies = cloneBoundDependencies(dependencies)
	x.lowerExplained = true
	return nil
}

func (s *Solver) SetUpperWithDependencies(column uint32, bound Infinitesimal, dependencies []BoundDependency) error {
	if err := s.SetUpper(column, bound); err != nil {
		return err
	}
	x := &s.vars[column]
	x.upperDependencies = cloneBoundDependencies(dependencies)
	x.upperExplained = true
	return nil
}

// BoundDependencies returns an owning copy, and distinguishes an unconditional
// endpoint from a bound without a logical explanation (such as a model guess).
func (s *Solver) BoundDependencies(column uint32, lower bool) ([]BoundDependency, bool) {
	if int(column) >= len(s.vars) {
		return nil, false
	}
	x := &s.vars[column]
	if lower {
		return cloneBoundDependencies(x.lowerDependencies), x.lowerValid && x.lowerExplained
	}
	return cloneBoundDependencies(x.upperDependencies), x.upperValid && x.upperExplained
}
