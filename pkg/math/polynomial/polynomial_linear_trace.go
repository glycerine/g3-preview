// Source: linear_eq_solver.h:90,124.
package polynomial

import (
	"io"
	"strconv"
)

func (s *RationalLinearSolver) traceIteration(k uint32, line int) {
	tags := s.integer.BorrowTraceTags()
	if !tags.ShouldTrace("linear_eq_solver") {
		return
	}
	_ = tags.EmitConditional("linear_eq_solver", "solve", "src/math/polynomial/linear_eq_solver.h", line,
		func() bool { return true }, func(w io.Writer) error {
			if e := linearWriteText(w, "iteration "+strconv.FormatUint(uint64(k), 10)+"\n"); e != nil {
				return e
			}
			return s.Display(w)
		})
}
func (s *ModularLinearSolver) traceIteration(k uint32, line int) {
	tags := s.manager.BorrowTraceTags()
	if !tags.ShouldTrace("linear_eq_solver") {
		return
	}
	_ = tags.EmitConditional("linear_eq_solver", "solve", "src/math/polynomial/linear_eq_solver.h", line,
		func() bool { return true }, func(w io.Writer) error {
			if e := linearWriteText(w, "iteration "+strconv.FormatUint(uint64(k), 10)+"\n"); e != nil {
				return e
			}
			return s.Display(w)
		})
}
