// Source: src/sat/sat_simplifier.cpp:421-449.
package sat

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func satBackwardSubsumptionTraceRecord(tag, line, body string) string {
	return "-------- [" + tag + "] back_subsumption1 src/sat/sat_simplifier.cpp:" + line + " ---------\n" +
		body + "------------------------------------------------\n"
}

func TestSATSimplifierBackwardSubsumptionSourceTraces(t *testing.T) {
	for _, tc := range []struct {
		name, tag string
		other     []Literal
		want      string
	}{
		{"subsumed", "subsumption", []Literal{0, 2, 4, 6},
			satBackwardSubsumptionTraceRecord("subsumption", "435", "(0 1 2)+ subsumed (0 1 2 3)+\n")},
		{"resolved", "subsumption_resolution", []Literal{1, 2, 4, 6},
			satBackwardSubsumptionTraceRecord("subsumption_resolution", "441", "(0 1 2)+ sub-ref(-0) (-0 1 2 3)+\n") +
				satBackwardSubsumptionTraceRecord("subsumption_resolution", "444", "result: (1 2 3)+\n")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, lifecycle := satSimplifierFixture(t, 4)
			a := lifecycle.propagation.clauses
			satOK(t, s.useList.Init(4))
			satOK(t, s.kernel.InitVisited(4))
			c1, err := a.MkClause([]Literal{0, 2, 4}, false)
			satOK(t, err)
			c2, err := a.MkClause(tc.other, false)
			satOK(t, err)
			for _, c := range []StoredClauseRef{c1, c2} {
				satOK(t, lifecycle.pushClause(false, c))
				satOK(t, s.useList.Insert(c))
			}
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable(tc.tag)
			lifecycle.traceTags = tags
			satOK(t, s.backSubsumption1(c1))
			want := ""
			if tags.ShouldTrace(tc.tag) {
				want = tc.want
			}
			if got := output.String(); got != want {
				t.Fatalf("source %s trace: got %q, want %q", tc.name, got, want)
			}
			if tc.name == "subsumed" && s.numSubsumed != 1 {
				t.Fatal("source subsumed count", s.numSubsumed)
			}
			if tc.name == "resolved" && s.numSubRes != 1 {
				t.Fatal("source resolution count", s.numSubRes)
			}
		})
	}
	t.Run("binary-dummy-owner", func(t *testing.T) {
		s, lifecycle := satSimplifierFixture(t, 3)
		a := lifecycle.propagation.clauses
		satOK(t, s.useList.Init(3))
		satOK(t, s.kernel.InitVisited(3))
		c, err := a.MkClause([]Literal{0, 2, 4}, false)
		satOK(t, err)
		satOK(t, lifecycle.pushClause(false, c))
		satOK(t, s.useList.Insert(c))
		var output bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		tags.Enable("subsumption")
		lifecycle.traceTags = tags
		satOK(t, s.backSubsumptionBinary(0, 2, false))
		want := ""
		if tags.ShouldTrace("subsumption") {
			want = satBackwardSubsumptionTraceRecord("subsumption", "435", "(0 1)+ subsumed (0 1 2)+\n")
		}
		if got := output.String(); got != want {
			t.Fatalf("source dummy-clause trace: got %q, want %q", got, want)
		}
	})
}
