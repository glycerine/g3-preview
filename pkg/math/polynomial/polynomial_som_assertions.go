// Source: src/math/polynomial/polynomial.cpp:1898,1910,1919,1936,2149.
package polynomial

import (
	"errors"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// polynomialSOMSourceAssert models SASSERT at the source statement. A Continue
// debugger action returns to the following statement; Throw preserves both the
// polynomial assertion identity and the debugger's concrete cause.
func polynomialSOMSourceAssert(line int32, expression string, condition bool) error {
	if !polynomialSourceDebug {
		return nil
	}
	d := diagnostic.BorrowProcessDebugContext()
	enabled, e := diagnostic.SourceAssertionsEnabled(d)
	if e != nil || !enabled || condition {
		return e
	}
	if e = diagnostic.DebugAssertionFailure(d, "src/math/polynomial/polynomial.cpp", line, expression); e != nil {
		return errors.Join(ErrPolynomialAssertion, e)
	}
	return nil
}
