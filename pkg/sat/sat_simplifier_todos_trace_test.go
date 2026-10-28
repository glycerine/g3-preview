package sat

// Source: src/sat/sat_simplifier.cpp:810-872.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func satSubsumptionTodoTraceRecord(line, body string) string {
	return "-------- [subsumption] subsume src/sat/sat_simplifier.cpp:" + line + " ---------\n" +
		body + "------------------------------------------------\n"
}

func TestSATSimplifierActualSubsumptionTodoSourceTraces(t *testing.T) {
	for _, binary := range []bool{false, true} {
		s, lifecycle := satSimplifierFixture(t, 3)
		p := lifecycle.propagation
		satOK(t, s.useList.Init(3))
		satOK(t, s.kernel.InitVisited(3))
		s.kernel.subCounter = 100
		c, err := p.clauses.MkClause([]Literal{0, 2, 4}, false)
		satOK(t, err)
		satOK(t, lifecycle.pushClause(false, c))
		satOK(t, s.useList.Insert(c))
		satOK(t, s.subTodo.Insert(c))
		if binary {
			satOK(t, p.assign(5, satJustification{}))
		}
		var output bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		tags.Enable("subsumption")
		lifecycle.traceTags = tags
		satOK(t, s.subsume())
		if !tags.ShouldTrace("subsumption") {
			if output.Len() != 0 {
				t.Fatal("build-erased source trace", output.String())
			}
			continue
		}
		got := output.String()
		prefix := satSubsumptionTodoTraceRecord("815", "sub_todo size: 1\n") +
			satSubsumptionTodoTraceRecord("845", "next: (0 1 2)\n")
		if !strings.HasPrefix(got, prefix) {
			t.Fatalf("source queue pop order binary=%v: %q", binary, got)
		}
		if binary {
			want := satSubsumptionTodoTraceRecord("863", "clause became binary: (0 1)+\n")
			if !strings.Contains(got, want) || strings.Contains(got, "using: ") {
				t.Fatalf("source binary branch: %q", got)
			}
		} else {
			want := satSubsumptionTodoTraceRecord("870", "using: (0 1 2)\n") +
				satSubsumptionTodoTraceRecord("815", "sub_todo size: 0\n")
			if !strings.HasSuffix(got, want) || strings.Contains(got, "clause became binary") {
				t.Fatalf("source ordinary branch: %q", got)
			}
		}
	}
}
