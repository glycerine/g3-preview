package sat

// Source: src/sat/sat_simplifier.cpp:255.

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATSimplifierBlockedCleanupClausesSourceTrace(t *testing.T) {
	s, l := satSimplifierFixture(t, 3)
	_, err := l.MkClauseCore(3, []Literal{0, 2, 4}, false)
	satOK(t, err)
	for i := 0; i < 2; i++ {
		satOK(t, s.Run(false))
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("sat")
	l.traceTags = tags
	satOK(t, s.Run(false))
	if s.numBlocked != 1 {
		t.Fatal("real third-call blocked elimination", s.numBlocked)
	}
	want := ""
	if tags.ShouldTrace("sat") {
		record := "-------- [sat] cleanup_clauses src/sat/sat_simplifier.cpp:255 ---------\n" +
			"cleanup_clauses\n------------------------------------------------\n"
		want = record + record
	}
	if got := output.String(); got != want {
		t.Fatalf("source learned/original cleanup records: got %q, want %q", got, want)
	}
}
