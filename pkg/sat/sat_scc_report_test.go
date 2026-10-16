// Source: src/sat/sat_scc.cpp:43-58,61-65.
package sat

import (
	"regexp"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATSCCReportActualEliminationAndSourceGates(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	out := satInspectionStream(t, arena.NewBudget(4096))
	oldLevel, err := diagnostic.ProcessVerbosityLevel()
	satOK(t, err)
	oldStream, err := diagnostic.BorrowProcessVerboseStream()
	satOK(t, err)
	t.Cleanup(func() {
		satOK(t, diagnostic.SetProcessVerboseStream(oldStream))
		satOK(t, diagnostic.SetProcessVerbosityLevel(oldLevel))
	})
	satOK(t, diagnostic.SetProcessVerboseStream(out))
	satOK(t, diagnostic.SetProcessVerbosityLevel(10))
	for range 2 {
		_, err = s.MkVar(false, true)
		satOK(t, err)
	}
	satOK(t, s.Lifecycle.mkBinClause(1, 2, false))
	satOK(t, s.Lifecycle.mkBinClause(3, 0, false))
	s.scc.enabled = false
	count, err := s.scc.run()
	satOK(t, err)
	got, err := out.String()
	satOK(t, err)
	if count != 0 || got != "" {
		t.Fatal("disabled SCC reached source report", count, got)
	}
	s.scc.enabled = true
	count, err = s.scc.run()
	satOK(t, err)
	got, err = out.String()
	satOK(t, err)
	want := regexp.MustCompile(`^ \(sat-scc :elim-vars 1 :gc-clause 0 :memory [0-9]+\.[0-9]{2} :time [0-9]+\.[0-9]{2}\)\n$`)
	if count != 1 || !want.MatchString(got) {
		t.Fatalf("source SCC report: %d %q", count, got)
	}
	satOK(t, out.ResetText())
	count, err = s.scc.run()
	satOK(t, err)
	got, err = out.String()
	satOK(t, err)
	want = regexp.MustCompile(`^ \(sat-scc :elim-vars 0 :gc-clause 0 :memory [0-9]+\.[0-9]{2} :time [0-9]+\.[0-9]{2}\)\n$`)
	if count != 0 || !want.MatchString(got) {
		t.Fatalf("source SCC delta: %d %q", count, got)
	}
	satOK(t, out.ResetText())
	satOK(t, diagnostic.SetProcessVerbosityLevel(9))
	count, err = s.scc.run()
	satOK(t, err)
	got, err = out.String()
	satOK(t, err)
	if count != 0 || got != "" {
		t.Fatalf("SCC verbosity gate: %d %q", count, got)
	}
}
