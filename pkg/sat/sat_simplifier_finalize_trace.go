// Source: src/sat/sat_simplifier.cpp:210,216; retained util/trace sites.
package sat

import (
	"io"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (s *satSimplifier) traceFinalizeCleanupBranch() {
	tags := s.lifecycle.traceTags
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	line, message := 216, "skipping cleanup...\n"
	if s.needCleanup {
		line, message = 210, "cleanning watches...\n"
	}
	_ = tags.EmitConditional("after_simplifier", "scoped_finalize_fn", "src/sat/sat_simplifier.cpp", line,
		func() bool { return true }, func(w io.Writer) error {
			_, err := io.WriteString(w, message)
			return err
		})
}
