// Source: src/sat/sat_solver.cpp:3287,3289,3297,3307,3366; retained util/trace sites.
package sat

import (
	"io"
	"strings"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (s *Solver) traceConsequenceRecord(function string, line int, body func(io.Writer) error) {
	tags := s.traceTags
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	_ = tags.EmitConditional("sat", function, "src/sat/sat_solver.cpp", line,
		func() bool { return true }, body)
}

func (s *Solver) traceConsequenceCore(line int) {
	s.traceConsequenceRecord("fixup_consequence_core", line, func(w io.Writer) error {
		n, err := s.conflict.core.Size()
		if err != nil {
			return err
		}
		var b strings.Builder
		for i := uint32(0); i < n; i++ {
			v, err := s.conflict.core.Read(i)
			if err != nil {
				return err
			}
			if i != 0 {
				b.WriteByte(' ')
			}
			b.WriteString(Literal(v).String())
		}
		b.WriteByte('\n')
		_, _ = io.WriteString(w, b.String())
		return nil
	})
}

func (s *Solver) traceConsequenceAntecedents(l Literal, row *satSolverIndexSet) {
	s.traceConsequenceRecord("fixup_consequence_core", 3289, func(w io.Writer) error {
		if err := row.open(); err != nil {
			return err
		}
		var b strings.Builder
		b.WriteString(l.String())
		b.WriteString(": ")
		for key := range row.keys.All() {
			b.WriteString(Literal(key).String())
			b.WriteByte(' ')
		}
		b.WriteByte('\n')
		_, _ = io.WriteString(w, b.String())
		return nil
	})
}

func (s *Solver) traceConsequenceAssumptions(function string, line int, asms satConsequenceInput) {
	s.traceConsequenceRecord(function, line, func(w io.Writer) error {
		n, err := asms.size()
		if err != nil {
			return err
		}
		var b strings.Builder
		for i := uint32(0); i < n; i++ {
			v, err := asms.read(i)
			if err != nil {
				return err
			}
			if i != 0 {
				b.WriteByte(' ')
			}
			b.WriteString(Literal(v).String())
		}
		b.WriteByte('\n')
		_, _ = io.WriteString(w, b.String())
		return nil
	})
}
