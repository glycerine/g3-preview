// Source: src/sat/sat_simplifier.cpp:949-1010.
package sat

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATSimplifierActualBlockedClauseSourceTraces(t *testing.T) {
	for _, binary := range []bool{false, true} {
		s, lifecycle := satSimplifierFixture(t, 3)
		if binary {
			satOK(t, lifecycle.mkBinClause(0, 2, false))
		} else {
			_, err := lifecycle.MkClauseCore(3, []Literal{0, 2, 4}, false)
			satOK(t, err)
		}
		var output bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		tags.Enable("blocked_clause")
		lifecycle.traceTags = tags
		for i := 0; i < 3; i++ {
			satOK(t, s.Run(false))
		}
		if !tags.ShouldTrace("blocked_clause") {
			if output.Len() != 0 {
				t.Fatal("build-erased blocked trace", output.String())
			}
			continue
		}
		got := output.String()
		first := "-------- [blocked_clause] process src/sat/sat_simplifier.cpp:951 ---------\n" +
			"processing: 0\n------------------------------------------------\n"
		blocked := "-------- [blocked_clause] process src/sat/sat_simplifier.cpp:967 ---------\n" +
			"new blocked clause: (0 1 2)\n------------------------------------------------\n"
		if binary {
			blocked = "-------- [blocked_clause] process src/sat/sat_simplifier.cpp:1008 ---------\n" +
				"new blocked clause: 1 0\n------------------------------------------------\n"
		}
		if !strings.HasPrefix(got, first+blocked) {
			t.Fatalf("source processing/publication prefix binary=%v: %q", binary, got)
		}
		if strings.Count(got, "src/sat/sat_simplifier.cpp:951") != 6 || s.numBlocked != 1 {
			t.Fatalf("source queue visits and blocked count binary=%v: count=%d trace=%q", binary, s.numBlocked, got)
		}
	}
}
