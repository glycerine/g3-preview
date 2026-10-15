// Source: src/sat/sat_solver.cpp:1825,1892; retained util/trace TRACE sites.
package sat

import (
	"io"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (s *satSolverConflictResolution) traceAssumptionCoreRecord(function string, line int, body func(io.Writer) error) {
	tags := s.traceTags
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	_ = tags.EmitConditional("sat", function, "src/sat/sat_solver.cpp", line,
		func() bool { return true }, body)
}

func (s *satSolverConflictResolution) traceUnsatCoreAntecedent(l Literal, marked bool) {
	s.traceAssumptionCoreRecord("process_antecedent_for_unsat_core", 1825, func(w io.Writer) error {
		flag := "-"
		if marked {
			flag = "+"
		}
		_, _ = io.WriteString(w, l.String()+" "+flag+"\n")
		return nil
	})
}

func (s *satSolverConflictResolution) traceInitNotLiteral(l Literal) {
	s.traceAssumptionCoreRecord("resolve_conflict_for_init", 1892, func(w io.Writer) error {
		_, _ = io.WriteString(w, "not_l: "+l.String()+"\n")
		return nil
	})
}
