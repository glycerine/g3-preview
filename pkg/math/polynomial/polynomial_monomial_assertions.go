// Source: src/math/polynomial/polynomial.cpp:228 monomial::get_power.
package polynomial

import (
	"errors"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (m *MonomialManager) powerSourceIndexAssert(a Monomial, i uint32) error {
	if !polynomialSourceDebug {
		return nil
	}
	d := diagnostic.BorrowProcessDebugContext()
	enabled, e := diagnostic.SourceAssertionsEnabled(d)
	if e != nil || !enabled {
		return e
	}
	n, e := m.sizeCore(a)
	if e != nil || i < n {
		return e
	}
	m.assertionRunning = true
	defer func() { m.assertionRunning = false }()
	if e := diagnostic.DebugAssertionFailure(d, "src/math/polynomial/polynomial.cpp", 228, "idx < size()"); e != nil {
		return errors.Join(ErrMonomialAssertion, e)
	}
	return nil
}
