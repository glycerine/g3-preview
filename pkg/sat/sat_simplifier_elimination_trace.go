// Source: src/sat/sat_simplifier.cpp:1296-1401; retained util/trace sites.
package sat

import (
	"io"
	"strconv"
	"strings"

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

func (s *satSimplifier) traceEliminationBeforeLiterals(v BoolVar, pos, neg, before uint32) {
	s.traceElimination("resolution", "try_eliminate", 1338, func(w io.Writer) error {
		_, err := io.WriteString(w, strconv.FormatUint(uint64(v), 10)+
			" num_pos: "+strconv.FormatUint(uint64(pos), 10)+
			" neg_pos: "+strconv.FormatUint(uint64(neg), 10)+
			" before_lits: "+strconv.FormatUint(uint64(before), 10)+"\n")
		return err
	})
}

func (s *satSimplifier) traceEliminationCollecting() {
	s.traceElimination("resolution_detail", "try_eliminate", 1354, func(w io.Writer) error {
		_, err := io.WriteString(w, "collecting number of after_clauses\n")
		return err
	})
}

func (s *satSimplifier) traceEliminationTooMany(after uint32) {
	s.traceElimination("resolution", "try_eliminate", 1369, func(w io.Writer) error {
		_, err := io.WriteString(w, "too many after clauses: "+strconv.FormatUint(uint64(after), 10)+"\n")
		return err
	})
}

func (s *satSimplifier) traceEliminationFound(before, after uint32) {
	s.traceElimination("resolution", "try_eliminate", 1375, func(w io.Writer) error {
		_, err := io.WriteString(w, "found var to eliminate, before: "+strconv.FormatUint(uint64(before), 10)+
			" after: "+strconv.FormatUint(uint64(after), 10)+"\n")
		return err
	})
}

func (s *satSimplifier) traceEliminationWrapper(c satClauseWrapper) (string, error) {
	a := s.lifecycle.propagation.clauses
	n, err := c.Size(a)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteByte('(')
	for i := uint32(0); i < n; i++ {
		if i > 0 {
			b.WriteByte(' ')
		}
		l, err := c.Literal(a, i)
		if err != nil {
			return "", err
		}
		b.WriteString(l.String())
	}
	b.WriteByte(')')
	return b.String(), nil
}

func (s *satSimplifier) traceEliminationInputsAndResult(w io.Writer, first, second satClauseWrapper, trailingSpace bool) error {
	a, err := s.traceEliminationWrapper(first)
	if err != nil {
		return err
	}
	b, err := s.traceEliminationWrapper(second)
	if err != nil {
		return err
	}
	if _, err = io.WriteString(w, a+"\n"+b+"\n-->\n"); err != nil {
		return err
	}
	n, err := s.literalStore.Size(s.newCls)
	if err != nil {
		return err
	}
	for i := uint32(0); i < n; i++ {
		l, err := s.literalStore.Read(s.newCls, i)
		if err != nil {
			return err
		}
		if i > 0 && !trailingSpace {
			if _, err = io.WriteString(w, " "); err != nil {
				return err
			}
		}
		if _, err = io.WriteString(w, l.String()); err != nil {
			return err
		}
		if trailingSpace {
			if _, err = io.WriteString(w, " "); err != nil {
				return err
			}
		}
	}
	_, err = io.WriteString(w, "\n")
	return err
}

func (s *satSimplifier) traceEliminationResolventDetail(first, second satClauseWrapper) {
	s.traceElimination("resolution_detail", "try_eliminate", 1365, func(w io.Writer) error {
		return s.traceEliminationInputsAndResult(w, first, second, true)
	})
}

func (s *satSimplifier) traceEliminationNewClause(first, second satClauseWrapper) {
	s.traceElimination("resolution_new_cls", "try_eliminate", 1401, func(w io.Writer) error {
		return s.traceEliminationInputsAndResult(w, first, second, false)
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
