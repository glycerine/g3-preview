// Source: src/math/polynomial/polynomial.cpp:228,322-348,356-373.
package polynomial

import (
	"errors"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func monomialSourceAssertionsEnabled() (bool, error) {
	if !polynomialSourceDebug {
		return false, nil
	}
	return diagnostic.SourceAssertionsEnabled(diagnostic.BorrowProcessDebugContext())
}

func (m *MonomialManager) sourceAssertionFailure(line int32, expression string) error {
	m.assertionRunning = true
	defer func() { m.assertionRunning = false }()
	if e := diagnostic.DebugAssertionFailure(diagnostic.BorrowProcessDebugContext(), "src/math/polynomial/polynomial.cpp", line, expression); e != nil {
		return errors.Join(ErrMonomialAssertion, e)
	}
	return nil
}

func (m *MonomialManager) powerSourceIndexAssert(a Monomial, i uint32) error {
	enabled, e := monomialSourceAssertionsEnabled()
	if e != nil || !enabled {
		return e
	}
	n, e := m.sizeCore(a)
	if e != nil || i < n {
		return e
	}
	return m.sourceAssertionFailure(228, "idx < size()")
}

// The condition uses only already-loaded scalar locals. Assertions containing
// source object reads use concrete helpers below, after the runtime switch.
func (m *MonomialManager) sourceScalarAssert(line int32, expression string, condition bool) error {
	enabled, e := monomialSourceAssertionsEnabled()
	if e != nil || !enabled || condition {
		return e
	}
	return m.sourceAssertionFailure(line, expression)
}

func (m *MonomialManager) searchVariableAssert(a Monomial, i uint32, x uint32, variableFirst bool, line int32, expression string) error {
	enabled, e := monomialSourceAssertionsEnabled()
	if e != nil || !enabled {
		return e
	}
	v, e := m.varCore(a, i)
	if e != nil {
		return e
	}
	if variableFirst && v < x || !variableFirst && x < v {
		return nil
	}
	return m.sourceAssertionFailure(line, expression)
}

func (m *MonomialManager) searchWindowAssert(a Monomial, low, high, mid int32, midpoint bool) error {
	enabled, e := monomialSourceAssertionsEnabled()
	if e != nil || !enabled {
		return e
	}
	line, expression := int32(326), "0 <= low && high < static_cast<int>(m_size)"
	valid := low >= 0
	if midpoint {
		line, expression = 340, "low < mid && mid < high && high < static_cast<int>(m_size)"
		valid = low < mid && mid < high
	}
	if valid {
		n, e := m.sizeCore(a)
		if e != nil {
			return e
		}
		valid = high < int32(n)
	}
	if !valid {
		return m.sourceAssertionFailure(line, expression)
	}
	return nil
}

func (m *MonomialManager) searchPreviousAssert(a Monomial, mid, x uint32) error {
	enabled, e := monomialSourceAssertionsEnabled()
	if e != nil || !enabled {
		return e
	}
	previous, e := m.varCore(a, mid-1)
	if e != nil {
		return e
	}
	valid := previous < x
	if valid {
		current, e := m.varCore(a, mid)
		if e != nil {
			return e
		}
		valid = x == current
	}
	if !valid {
		return m.sourceAssertionFailure(341, "get_var(mid-1) < x && x == get_var(mid)")
	}
	return nil
}

func (m *MonomialManager) searchResultAssert(a Monomial, i, y uint32) error {
	enabled, e := monomialSourceAssertionsEnabled()
	if e != nil || !enabled {
		return e
	}
	valid := y != ^uint32(0)
	if valid {
		v, e := m.varCore(a, i)
		if e != nil {
			return e
		}
		valid = v <= y
	}
	if !valid {
		return m.sourceAssertionFailure(368, "y != null_var && get_var(i) <= y")
	}
	return nil
}
