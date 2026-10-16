// Source: src/sat/sat_simplifier.cpp:951,967,1008; retained util/trace sites.
package sat

import (
	"io"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (s *satSimplifier) traceBlocked(line int, body func(io.Writer) error) {
	tags := s.lifecycle.traceTags
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	_ = tags.EmitConditional("blocked_clause", "process", "src/sat/sat_simplifier.cpp", line,
		func() bool { return true }, body)
}

func (s *satSimplifier) traceBlockedProcessing(l Literal) {
	s.traceBlocked(951, func(w io.Writer) error {
		_, err := io.WriteString(w, "processing: "+l.String()+"\n")
		return err
	})
}

func (s *satSimplifier) traceBlockedClause(c StoredClauseRef) {
	s.traceBlocked(967, func(w io.Writer) error {
		clause, err := satTraceClause(s.lifecycle.propagation.clauses, c)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, "new blocked clause: "+clause+"\n")
		return err
	})
}

func (s *satSimplifier) traceBlockedBinary(other, selected Literal) {
	s.traceBlocked(1008, func(w io.Writer) error {
		_, err := io.WriteString(w, "new blocked clause: "+other.String()+" "+selected.String()+"\n")
		return err
	})
}
