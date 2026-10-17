// Source: linear_eq_solver.h:90,124.
package polynomial

import (
	"io"
	"strconv"
)

func (s *RationalLinearSolver) traceIteration(k uint32, line int) error {
	tags := s.integer.BorrowTraceTags()
	if !tags.ShouldTrace("linear_eq_solver") {
		return nil
	}
	var computation error
	_ = tags.EmitConditional("linear_eq_solver", "solve", "src/math/polynomial/linear_eq_solver.h", line,
		func() bool { return true }, func(w io.Writer) error {
			out := linearTraceWriter{w}
			_ = linearWriteText(out, "iteration "+strconv.FormatUint(uint64(k), 10)+"\n")
			computation = s.displaySource(out)
			return computation
		})
	return computation
}
func (s *ModularLinearSolver) traceIteration(k uint32, line int) error {
	tags := s.manager.BorrowTraceTags()
	if !tags.ShouldTrace("linear_eq_solver") {
		return nil
	}
	var computation error
	_ = tags.EmitConditional("linear_eq_solver", "solve", "src/math/polynomial/linear_eq_solver.h", line,
		func() bool { return true }, func(w io.Writer) error {
			out := linearTraceWriter{w}
			_ = linearWriteText(out, "iteration "+strconv.FormatUint(uint64(k), 10)+"\n")
			computation = s.displaySource(out)
			return computation
		})
	return computation
}

// The diagnostic sticky writer retains ostream failure state. Suppress only
// insertion errors here so later numerical/display operands still execute;
// Display computation errors remain separate and stop the enclosing solver.
type linearTraceWriter struct{ io.Writer }

func (w linearTraceWriter) Write(p []byte) (int, error) {
	_, _ = w.Writer.Write(p)
	return len(p), nil
}
