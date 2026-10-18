package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Both concrete C++ get overloads use the process runtime SASSERT switch.
func berlekampAssertionFailure(line int32) error {
	if e := diagnostic.DebugAssertionFailure(diagnostic.BorrowProcessDebugContext(), "src/math/polynomial/upolynomial_factorization.cpp", line, "i < m_size && j < m_size"); e != nil {
		return errors.Join(ErrUPolynomialAssertion, e)
	}
	return nil
}
