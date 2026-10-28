package sat

// Source: src/sat/sat_scc.cpp:68-69,138,156,168,191,216,221.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATSCCSourceCycleAndRootTraces(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		c, l := satSCCFixture(t, 2)
		satOK(t, l.mkBinClause(1, 2, false))
		satOK(t, l.mkBinClause(3, 0, false))
		var output bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		if enabled {
			for _, tag := range []string{"scc", "scc_detail", "scc_cycle"} {
				tags.Enable(tag)
			}
		}
		l.traceTags = tags
		n, err := c.run()
		satOK(t, err)
		if n != 1 {
			t.Fatalf("source equivalence elimination count %d", n)
		}
		got := output.String()
		if !tags.ShouldTrace("scc") {
			if got != "" {
				t.Fatalf("disabled source trace %q", got)
			}
			continue
		}
		for _, body := range []string{
			"cycle: 1 0 \n", "cycle: -1 -0 \n",
			"assigning roots...\n", "consuming stack...\n", "r: 0\n",
			"0 -> 0\n1 -> 0\nto_elim: 1 \n",
		} {
			if !strings.Contains(got, body) {
				t.Fatalf("missing source SCC body %q in %q", body, got)
			}
		}
		if strings.Count(got, "[scc_cycle] operator() src/sat/sat_scc.cpp:138") != 2 ||
			strings.Count(got, "[scc] operator() src/sat/sat_scc.cpp:216") != 1 {
			t.Fatalf("source SCC trace count/order %q", got)
		}
	}
}

func TestSATSCCFullSolverDisplaySourceTraces(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	_, err := s.MkVar(true, true)
	satOK(t, err)
	observer := &satSimplifierRunDisplayObserver{solver: s}
	tags := diagnostic.NewTraceTagsWithWriter(observer)
	for _, tag := range []string{"scc", "scc_details", "scc_detail"} {
		tags.Enable(tag)
	}
	s.Lifecycle.traceTags = tags
	_, err = s.scc.run()
	satOK(t, err)
	if !tags.ShouldTrace("scc") {
		if observer.output.Len() != 0 {
			t.Fatal("build-erased SCC display", observer.output.String())
		}
		return
	}
	got := observer.output.String()
	if len(observer.match) != 2 || !observer.match[0] || !observer.match[1] {
		t.Fatalf("source solver display at both SCC sites: %v %q", observer.match, got)
	}
	for _, header := range []string{
		"[scc] operator() src/sat/sat_scc.cpp:68",
		"[scc_details] operator() src/sat/sat_scc.cpp:69",
		"[scc_detail] operator() src/sat/sat_scc.cpp:221",
	} {
		if !strings.Contains(got, header) {
			t.Fatalf("missing full solver SCC display %q in %q", header, got)
		}
	}
}
