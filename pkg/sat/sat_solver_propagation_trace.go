// Source: src/sat/sat_solver.cpp:506,564; retained util/trace TRACE sites.
package sat

import (
	"io"
	"strconv"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: src/sat/sat_justification.h:56-77.
func satJustificationTraceString(j satJustification) string {
	switch j.Kind() {
	case 0:
		return "none"
	case 1:
		return "binary " + Literal(j.val1).String()
	case 2:
		return "ternary " + Literal(j.val1).String() + " " + Literal(j.val2>>3).String()
	case 3:
		return "clause"
	case 4:
		return "external"
	}
	return ""
}

func (s *satSolverPropagationState) traceRecord(tag, function string, line int, body func(io.Writer) error) {
	tags := s.traceTags
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	_ = tags.EmitConditional(tag, function, "src/sat/sat_solver.cpp", line,
		func() bool { return true }, body)
}

func (s *satSolverPropagationState) traceAssignCore(l Literal, j satJustification) {
	s.traceRecord("sat_assign_core", "assign_core", 506, func(w io.Writer) error {
		_, _ = io.WriteString(w, l.String()+" "+satJustificationTraceString(j)+
			" level: "+strconv.FormatUint(uint64(s.variables.scopeLevel), 10)+"\n")
		return nil
	})
}

func (s *satSolverPropagationState) tracePropagating(l Literal) {
	s.traceRecord("sat_propagate", "propagate_core", 564, func(w io.Writer) error {
		j, err := s.variables.justification.read(uint32(l.variable()))
		if err != nil {
			return err
		}
		_, _ = io.WriteString(w, "propagating: "+l.String()+" "+satJustificationTraceString(j)+"\n")
		return nil
	})
}
