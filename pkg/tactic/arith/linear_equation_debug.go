// Source: src/tactic/arith/linear_equation.cpp:93-97.
package arith

import "github.com/glycerine/g3/pkg/util/diagnostic"

// The source checks this invariant only inside DEBUG_CODE, before the first
// mk_core TRACE record or numeral operation.
func (l *LinearEquationManager) assertSortedEquationVars(xs []uint32) error {
	d := diagnostic.BorrowProcessDebugContext()
	on, err := diagnostic.SourceAssertionsEnabled(d)
	if err != nil || !on {
		return err
	}
	for i := 1; i < len(xs); i++ {
		if xs[i-1] >= xs[i] {
			return diagnostic.DebugAssertionFailure(d, "src/tactic/arith/linear_equation.cpp", 95, "xs[i-1] < xs[i]")
		}
	}
	return nil
}
