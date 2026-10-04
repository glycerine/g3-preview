// Source: src/math/polynomial/polynomial.cpp:6237-6425,3185; src/math/polynomial/polynomial.h:942-966.
// Licensed under the MIT license.
// Author: Leonardo de Moura.
package mpnum

// This is the actually used cold virtual-map surface. The source manager()
// virtual is not called by substitution; array wrappers likewise do not bind
// a fabricated rational manager to the source UNREACHABLE implementation.
type PolynomialRationalSubstitution interface {
	Contains(uint32) (bool, error)
	Rational(uint32) (*Rational, error)
}
type polynomialSubstitutionArray struct {
	positions *PolynomialVariablePositions
	values    []Rational
}

// Distinct actual persistent m_var_max_degree, m_var_pos and m_found_vars.
// Embedded cold metadata is included in the reflected core charge. The source
// imp temporary itself is core.tmp1, never a monomial-manager scratch slot.
type polynomialSubstitutionState struct {
	max                     PolynomialMaxDegrees
	positions               PolynomialVariablePositions
	found                   polynomialFoundVariables
	array                   polynomialSubstitutionArray
	scope                   polynomialSubstitutionScope
	orphans                 []Monomial
	orphanDrain             uint32
	generatedReceipt        Monomial
	borrowedReceipt         uint64
	generatedCalleeReceipt  uint64
	bodyCause, cleanupCause error
}
type polynomialSubstitutionScope struct {
	active, executing                                bool
	kind                                             uint8
	newA, tmp                                        Int
	newAConstructed, tmpConstructed                  bool
	maxConstructed, posConstructed                   bool
	xs                                               PolynomialVariablesView
	xsCount                                          uint32
	single                                           bool
	singleX                                          uint32
	p1, q1, product                                  polynomialReference
	p1Constructed, q1Constructed, productConstructed bool
	monomial                                         polynomialDivisionReference
	monomialConstructed                              bool
	stage                                            uint8
	resetStopped                                     bool
	returned                                         Polynomial
	bodyCause, cleanupCause                          error
}
