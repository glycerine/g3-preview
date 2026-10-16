// Source: src/sat/sat_solver.cpp:1678; src/util/debug.h:64.
package sat

import (
	"errors"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (s *satSolverConflictResolution) assertNoMarks() error {
	if !satStorageSourceDebug {
		return nil
	}
	d := diagnostic.BorrowProcessDebugContext()
	enabled, err := d.ConditionalAssertionEnabled("sat_check_marks")
	if err != nil || !enabled {
		return err
	}
	if err = s.checkMarks(); err == nil {
		return nil
	}
	if !errors.Is(err, ErrSATAssertion) {
		return err
	}
	return diagnostic.DebugAssertionFailure(d, "src/sat/sat_solver.cpp", 1678, "check_marks()")
}
