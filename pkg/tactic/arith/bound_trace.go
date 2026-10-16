// Source: src/tactic/arith/bound_propagator.cpp:279-305,331-357,382-411,457,473,523.
package arith

import (
	"io"
	"strconv"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// BindTrace borrows a local trace stream for the concrete bound owner. Nil
// restores the process trace stream used by the source global TRACE service.
func (s *BoundEquationStore) BindTrace(tags *diagnostic.TraceTags) error {
	if err := s.open(); err != nil {
		return err
	}
	s.trace = tags
	return nil
}

func (s *BoundEquationStore) boundTraceTags() *diagnostic.TraceTags {
	if s.trace != nil {
		return s.trace
	}
	return diagnostic.BorrowProcessTraceTags()
}

func (s *BoundEquationStore) emitBoundTrace(tag, function string, line int, body string) {
	_ = s.boundTraceTags().EmitConditional(tag, function, "src/tactic/arith/bound_propagator.cpp", line,
		func() bool { return true }, func(w io.Writer) error {
			_, err := io.WriteString(w, body)
			return err
		})
}

func boundTraceContext(lower bool) (function, direction string, entryLine, falseLine, derivedLine, approxLine, relaxedLine int) {
	if lower {
		return "assert_lower_core", "lower", 279, 286, 294, 302, 305
	}
	return "assert_upper_core", "upper", 331, 338, 347, 354, 357
}

func sourceBoolDigit(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

func (s *BoundEquationStore) traceBoundValue(tag string, lower bool, x uint32, k *numeral.Rational, strict bool, derived bool) error {
	tags := s.boundTraceTags()
	if !tags.ShouldTrace(tag) {
		return nil
	}
	value, err := s.manager.String(k)
	if err != nil {
		return err
	}
	function, direction, entryLine, _, derivedLine, _, _ := boundTraceContext(lower)
	line := entryLine
	if derived {
		line = derivedLine
	}
	body := "new " + direction + " x" + strconv.FormatUint(uint64(x), 10) + " " + value + " strict: " + sourceBoolDigit(strict) + "\n"
	s.emitBoundTrace(tag, function, line, body)
	return nil
}

func (s *BoundEquationStore) traceBoundFalseAlarm(lower bool) {
	if !s.boundTraceTags().ShouldTrace("bound_propagator_detail") {
		return
	}
	function, _, _, line, _, _, _ := boundTraceContext(lower)
	s.emitBoundTrace("bound_propagator_detail", function, line, "false alarm\n")
}

func (s *BoundEquationStore) traceBoundApprox(lower bool, x uint32, k *numeral.Rational, approx float64, relaxed bool) error {
	tags := s.boundTraceTags()
	if !tags.ShouldTrace("new_bound") {
		return nil
	}
	value, err := s.manager.String(k)
	if err != nil {
		return err
	}
	function, direction, _, _, _, approxLine, relaxedLine := boundTraceContext(lower)
	line, label := approxLine, " approx: "
	if relaxed {
		line, label = relaxedLine, " relaxed approx: "
	}
	body := "x" + strconv.FormatUint(uint64(x), 10) + " " + direction + ": " + value + label + strconv.FormatFloat(approx, 'g', 6, 64) + "\n"
	s.emitBoundTrace("new_bound", function, line, body)
	return nil
}

func (s *BoundEquationStore) traceRelevantBound(x uint32, candidate float64, lower bool, old *equationBound) error {
	if !s.boundTraceTags().ShouldTrace("bound_propagator_detail") {
		return nil
	}
	body := "relevant_bound x" + strconv.FormatUint(uint64(x), 10) + " " + strconv.FormatFloat(candidate, 'g', 6, 64) + " LOWER: " + sourceBoolDigit(lower) + "\n"
	if old != nil {
		value, err := s.manager.String(&old.value)
		if err != nil {
			return err
		}
		body += "old: " + value + " | " + strconv.FormatFloat(old.approx, 'g', 6, 64) + "\n"
	}
	s.emitBoundTrace("bound_propagator_detail", "relevant_bound", 382, body)
	return nil
}

func (s *BoundEquationStore) traceBoundSmallImprovement(candidate, old float64, lower bool) {
	if !s.boundTraceTags().ShouldTrace("bound_propagator") {
		return
	}
	label, line := "UPPER", 411
	if lower {
		label, line = "LOWER", 405
	}
	body := label + " new: " + strconv.FormatFloat(candidate, 'g', 6, 64) + " old: " + strconv.FormatFloat(old, 'g', 6, 64) + " improvement is too small\n"
	s.emitBoundTrace("bound_propagator", "relevant_bound", line, body)
}

func (s *BoundEquationStore) traceBoundConflict(x uint32) {
	tags := s.boundTraceTags()
	if !tags.ShouldTrace("bound_propagator") {
		return
	}
	_ = tags.EmitConditional("bound_propagator", "check_feasibility", "src/tactic/arith/bound_propagator.cpp", 457,
		func() bool { return true }, func(w io.Writer) error {
			if _, err := io.WriteString(w, "inconsistency detected: x"+strconv.FormatUint(uint64(x), 10)+"\n"); err != nil {
				return err
			}
			return s.Display(w)
		})
}

func (s *BoundEquationStore) tracePropagatingVariable(x uint32) {
	if !s.boundTraceTags().ShouldTrace("bound_propagator_detail") {
		return
	}
	s.emitBoundTrace("bound_propagator_detail", "propagate", 473,
		"propagating x"+strconv.FormatUint(uint64(x), 10)+"\n")
}

func (s *BoundEquationStore) tracePropagatingEquation(eq *LinearEquation) {
	tags := s.boundTraceTags()
	if !tags.ShouldTrace("bound_propagator_detail") {
		return
	}
	_ = tags.EmitConditional("bound_propagator_detail", "propagate_eq", "src/tactic/arith/bound_propagator.cpp", 523,
		func() bool { return true }, func(w io.Writer) error {
			if _, err := io.WriteString(w, "propagating using eq: "); err != nil {
				return err
			}
			if err := s.equations.Display(w, eq); err != nil {
				return err
			}
			_, err := io.WriteString(w, "\n")
			return err
		})
}
