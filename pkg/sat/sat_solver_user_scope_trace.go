// Source: src/sat/sat_solver.cpp:2625,2741; retained util/trace TRACE sites.
package sat

import (
	"io"
	"strings"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (s *Solver) traceUserScopeRecord(function string, line int, body func(io.Writer) error) {
	tags := s.traceTags
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	_ = tags.EmitConditional("sat", function, "src/sat/sat_solver.cpp", line,
		func() bool { return true }, body)
}

func (s *Solver) traceUserPush(l Literal) {
	s.traceUserScopeRecord("user_push", 2625, func(w io.Writer) error {
		_, _ = io.WriteString(w, "user_push: "+l.String()+"\n")
		return nil
	})
}

func (s *Solver) traceUserPopTrail() {
	s.traceUserScopeRecord("user_pop", 2741, func(w io.Writer) error {
		n, err := s.Variables.trail.Size(s.Variables.trailRef)
		if err != nil {
			return err
		}
		var b strings.Builder
		for i := uint32(0); i < n; i++ {
			l, err := s.Variables.trailRead(i)
			if err != nil {
				return err
			}
			if i != 0 {
				b.WriteByte(' ')
			}
			b.WriteString(l.String())
		}
		b.WriteByte('\n')
		_, _ = io.WriteString(w, b.String())
		return nil
	})
}
