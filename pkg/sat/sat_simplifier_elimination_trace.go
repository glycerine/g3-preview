// Source: src/sat/sat_simplifier.cpp:1296,1302,1315; retained util/trace sites.
package sat

import (
	"io"
	"strconv"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (s *satSimplifier) traceElimination(tag, method string, line int, body func(io.Writer) error) {
	tags := s.lifecycle.traceTags
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	_ = tags.EmitConditional(tag, method, "src/sat/sat_simplifier.cpp", line,
		func() bool { return true }, body)
}

func (s *satSimplifier) traceEliminationProcessing(v BoolVar) {
	s.traceElimination("resolution_bug", "try_eliminate", 1302, func(w io.Writer) error {
		_, err := io.WriteString(w, "processing: "+strconv.FormatUint(uint64(v), 10)+"\n")
		return err
	})
}

func (s *satSimplifier) traceEliminationOccurrences(v BoolVar, pos, neg uint32) {
	s.traceElimination("resolution", "try_eliminate", 1315, func(w io.Writer) error {
		_, err := io.WriteString(w, strconv.FormatUint(uint64(v), 10)+
			" num_pos: "+strconv.FormatUint(uint64(pos), 10)+
			" neg_pos: "+strconv.FormatUint(uint64(neg), 10)+"\n")
		return err
	})
}

func (s *satSimplifier) traceEliminatedClause(c StoredClauseRef) {
	s.traceElimination("resolution_bug", "remove_clauses", 1296, func(w io.Writer) error {
		clause, err := satTraceClause(s.lifecycle.propagation.clauses, c)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, "del_clause (elim_var): "+clause+"\n")
		return err
	})
}
