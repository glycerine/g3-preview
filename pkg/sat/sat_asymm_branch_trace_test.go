// Source: src/sat/sat_asymm_branch.cpp:118,143,162,187.
package sat

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func satAsymmTraceRecord(tag string, line int, body string) string {
	return "-------- [" + tag + "] process src/sat/sat_asymm_branch.cpp:" +
		strconv.Itoa(line) + " ---------\n" + body + "------------------------------------------------\n"
}

func TestSATAsymmBranchProcessSourceTraces(t *testing.T) {
	for _, unit := range []bool{false, true} {
		a, l := satAsymmFixture(t, 4)
		if unit {
			_, e := l.MkClauseCore(2, []Literal{0, 2}, false)
			satOK(t, e)
			_, e = l.MkClauseCore(2, []Literal{0, 3}, false)
			satOK(t, e)
		} else {
			_, e := l.MkClauseCore(2, []Literal{0, 2}, false)
			satOK(t, e)
		}
		r := satAsymmClause(t, l, []Literal{0, 2, 4})
		var output bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		tags.Enable("asymm_branch_detail")
		tags.Enable("asymm_branch")
		l.traceTags = tags
		satOK(t, a.run(true))
		want := ""
		if tags.ShouldTrace("asymm_branch_detail") {
			want += satAsymmTraceRecord("asymm_branch_detail", 118, "processing: (0 1 2)+\n")
			want += satAsymmTraceRecord("asymm_branch_detail", 143, "assigning: -0\n")
			if !unit {
				want += satAsymmTraceRecord("asymm_branch_detail", 143, "assigning: -1\n")
			}
		}
		if tags.ShouldTrace("asymm_branch") {
			newSize := "2"
			if unit {
				newSize = "1"
			}
			want += satAsymmTraceRecord("asymm_branch", 162, "(0 1 2)+\nnew_size: "+newSize+"\n0 0 0 \n")
			if unit {
				want += satAsymmTraceRecord("asymm_branch", 187, "produced unit clause: 0\n")
			}
		}
		if got := output.String(); got != want {
			t.Fatalf("unit=%v: got %q, want %q", unit, got, want)
		}
		if l.propagation.clauses.IsLive(r) || a.elimLiterals == 0 {
			t.Fatalf("unit=%v: no source reduction", unit)
		}
	}
}

func TestSATAsymmBranchDisabledSourceTrace(t *testing.T) {
	a, l := satAsymmFixture(t, 3)
	_, e := l.MkClauseCore(2, []Literal{0, 2}, false)
	satOK(t, e)
	satAsymmClause(t, l, []Literal{0, 2, 4})
	var output bytes.Buffer
	l.traceTags = diagnostic.NewTraceTagsWithWriter(&output)
	satOK(t, a.run(true))
	if strings.Contains(output.String(), "asymm_branch") {
		t.Fatal("disabled source trace emitted", output.String())
	}
}

func TestSATAsymmBranchFullSolverDisplaySourceTrace(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	_, err := s.MkVar(true, true)
	satOK(t, err)
	observer := &satSimplifierRunDisplayObserver{solver: s}
	tags := diagnostic.NewTraceTagsWithWriter(observer)
	tags.Enable("asymm_branch_detail")
	s.Lifecycle.traceTags = tags
	satOK(t, s.asymm.run(true))
	if !tags.ShouldTrace("asymm_branch_detail") {
		if observer.output.Len() != 0 {
			t.Fatal("build-erased source display", observer.output.String())
		}
		return
	}
	wantHeader := "-------- [asymm_branch_detail] operator() src/sat/sat_asymm_branch.cpp:68 ---------\n"
	if len(observer.match) != 1 || !observer.match[0] || !strings.HasPrefix(observer.output.String(), wantHeader) ||
		strings.Count(observer.output.String(), "------------------------------------------------\n") != 1 {
		t.Fatal("source solver display at run entry", observer.match, observer.output.String())
	}
}
