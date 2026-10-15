// Source: src/sat/sat_solver.cpp:1494,1519; retained util/trace TRACE sites.
package sat

import (
	"io"
	"strconv"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (g *satSolverGC) traceGC(function string, line int) {
	tags := g.traceTags
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	_ = tags.EmitConditional("sat", function, "src/sat/sat_solver.cpp", line,
		func() bool { return true }, func(w io.Writer) error {
			_, _ = io.WriteString(w, "gc\n")
			return nil
		})
}

func (g *satSolverGC) traceMinDTk() {
	tags := g.traceTags
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	_ = tags.EmitConditional("sat_frozen", "gc_dyn_psm", "src/sat/sat_solver.cpp", 1540,
		func() bool { return true }, func(w io.Writer) error {
			_, _ = io.WriteString(w, "m_min_d_tk: ")
			_, _ = io.WriteString(w, strconv.FormatFloat(g.minDTk, 'g', 6, 64))
			_, _ = io.WriteString(w, "\n")
			return nil
		})
}
