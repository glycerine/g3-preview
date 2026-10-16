// Source: src/sat/sat_simplifier.cpp:206-224.
package sat

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type satSimplifierFinalizeTraceObserver struct {
	output bytes.Buffer
	s      *satSimplifier
	want   bool
	before bool
}

func (w *satSimplifierFinalizeTraceObserver) Write(p []byte) (int, error) {
	if strings.HasPrefix(string(p), "cleanning watches...") || strings.HasPrefix(string(p), "skipping cleanup...") {
		w.before = w.s.scopeActive && w.s.scopedStage == 0 && w.s.needCleanup == w.want
	}
	return w.output.Write(p)
}

func TestSATSimplifierActualFinalizeCleanupBranchSourceTrace(t *testing.T) {
	for _, cleanup := range []bool{false, true} {
		s, l := satSimplifierFixture(t, 4)
		if cleanup {
			p := l.propagation
			for i := uint32(1); i < 4; i++ {
				satOK(t, p.variables.setExternal(BoolVar(i), true))
			}
			c, err := l.MkClauseCore(3, []Literal{0, 2, 4}, false)
			satOK(t, err)
			_, err = l.MkClauseCore(3, []Literal{1, 2, 4}, false)
			satOK(t, err)
			_, err = l.MkClauseCore(4, []Literal{0, 2, 4, 6}, false)
			satOK(t, err)
			satOK(t, p.clauses.MarkStrengthened(c))
		}
		observer := &satSimplifierFinalizeTraceObserver{s: s, want: cleanup}
		tags := diagnostic.NewTraceTagsWithWriter(observer)
		tags.Enable("after_simplifier")
		l.traceTags = tags
		satOK(t, s.Run(false))
		want := ""
		if tags.ShouldTrace("after_simplifier") {
			line, message := "216", "skipping cleanup...\n"
			if cleanup {
				line, message = "210", "cleanning watches...\n"
			}
			want = "-------- [after_simplifier] scoped_finalize_fn src/sat/sat_simplifier.cpp:" + line + " ---------\n" +
				message + "------------------------------------------------\n"
			if !observer.before {
				t.Fatal("trace must precede selected source cleanup branch", cleanup)
			}
		}
		if got := observer.output.String(); got != want {
			t.Fatalf("source finalizer trace cleanup=%v: got %q, want %q", cleanup, got, want)
		}
	}
}
