package sat

// Source: src/sat/sat_simplifier.cpp:98-105.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type satSimplifierRemoveTraceObserver struct {
	output            bytes.Buffer
	simplifier        *satSimplifier
	clause            StoredClauseRef
	markedBeforeTrace bool
}

func (w *satSimplifierRemoveTraceObserver) Write(p []byte) (int, error) {
	if strings.HasPrefix(string(p), "del_clause: ") {
		removed, err := w.simplifier.lifecycle.propagation.clauses.Removed(w.clause)
		w.markedBeforeTrace = err == nil && removed && !w.simplifier.needCleanup
	}
	return w.output.Write(p)
}

func TestSATSimplifierRemoveClauseSourceTraceAfterMarkBeforeCleanup(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		s, l := satSimplifierFixture(t, 3)
		c, err := l.propagation.clauses.MkClause([]Literal{0, 2, 4}, false)
		satOK(t, err)
		satOK(t, l.pushClause(false, c))
		observer := &satSimplifierRemoveTraceObserver{simplifier: s, clause: c}
		tags := diagnostic.NewTraceTagsWithWriter(observer)
		if enabled {
			tags.Enable("resolution_bug")
		}
		l.traceTags = tags
		satOK(t, s.removeClauseCore(c))
		removed, err := l.propagation.clauses.Removed(c)
		satOK(t, err)
		if !removed || !s.needCleanup {
			t.Fatal("source clause removal and cleanup publication", removed, s.needCleanup)
		}
		want := ""
		if tags.ShouldTrace("resolution_bug") {
			want = "-------- [resolution_bug] remove_clause_core src/sat/sat_simplifier.cpp:103 ---------\n" +
				"del_clause: (0 1 2)x+\n------------------------------------------------\n"
			if !observer.markedBeforeTrace {
				t.Fatal("trace must follow removed flag and precede cleanup flag")
			}
		}
		if got := observer.output.String(); got != want {
			t.Fatalf("source removal trace = %q, want %q", got, want)
		}
	}
}
