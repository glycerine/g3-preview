// Source: src/sat/sat_asymm_branch.cpp:35-52,54-70.
package sat

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATAsymmReportActualReductionAndSourceGates(t *testing.T) {
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
	for range 3 {
		_, err = s.MkVar(false, true)
		satOK(t, err)
	}
	_, err = s.Lifecycle.MkClauseCore(2, []Literal{0, 2}, false)
	satOK(t, err)
	r := satAsymmClause(t, s.Lifecycle, []Literal{0, 2, 4})
	s.asymm.enabled = false
	satOK(t, s.asymm.run(true))
	got, err := out.String()
	satOK(t, err)
	if got != "" {
		t.Fatal("disabled asymmetric branch emitted report", got)
	}
	s.asymm.enabled = true
	satOK(t, s.asymm.run(true))
	if s.Clauses.IsLive(r) || s.asymm.elimLiterals != 1 {
		t.Fatal("actual source reduction", s.asymm.elimLiterals)
	}
	got, err = out.String()
	satOK(t, err)
	want := regexp.MustCompile(fmt.Sprintf(`^ \(sat-asymm-branch :elim-literals 1 :cost %d :memory [0-9]+\.[0-9]{2} :time [0-9]+\.[0-9]{2}\)\n$`, s.propagation.asymmCounter))
	if !want.MatchString(got) {
		t.Fatalf("source asymmetric report: %q", got)
	}
	satOK(t, out.ResetText())
	satOK(t, s.asymm.run(true))
	got, err = out.String()
	satOK(t, err)
	want = regexp.MustCompile(`^ \(sat-asymm-branch :elim-literals 0 :cost 0 :memory [0-9]+\.[0-9]{2} :time [0-9]+\.[0-9]{2}\)\n$`)
	if !want.MatchString(got) {
		t.Fatalf("source asymmetric second-run delta: %q", got)
	}
	satOK(t, out.ResetText())
	satOK(t, diagnostic.SetProcessVerbosityLevel(9))
	satOK(t, s.asymm.run(true))
	got, err = out.String()
	satOK(t, err)
	if got != "" {
		t.Fatal("source verbosity gate emitted report", got)
	}
}

func TestSATAsymmReportSurvivesCanceledSourceCatch(t *testing.T) {
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
	for range 3 {
		_, err = s.MkVar(false, true)
		satOK(t, err)
	}
	satAsymmClause(t, s.Lifecycle, []Literal{0, 2, 4})
	s.Limit.SetCancel(1)
	err = s.asymm.run(true)
	if !errors.Is(err, ErrSATSolverCanceled) {
		t.Fatalf("source catch: %v", err)
	}
	got, err := out.String()
	satOK(t, err)
	want := regexp.MustCompile(`^ \(sat-asymm-branch :elim-literals 0 :cost 1 :memory [0-9]+\.[0-9]{2} :time [0-9]+\.[0-9]{2}\)\n$`)
	if !want.MatchString(got) {
		t.Fatalf("report on source catch: %q", got)
	}
	s.Limit.ResetCancel()
}
