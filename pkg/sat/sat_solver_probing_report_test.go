// Source: src/sat/sat_probing.cpp:171-188,190-240.
package sat

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATSolverProbingReportAssignedStoppedAndSourceGates(t *testing.T) {
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

	t.Run("assigned", func(t *testing.T) {
		s, _, _ := satSolverCheckFixture(t)
		for range 2 {
			_, err := s.MkVar(false, true)
			satOK(t, err)
		}
		satOK(t, s.Lifecycle.mkBinClause(0, 2, false))
		satOK(t, s.Lifecycle.mkBinClause(1, 2, false))
		s.probing.enabled = false
		answer, err := s.probing.run(true)
		satOK(t, err)
		got, err := out.String()
		satOK(t, err)
		if !answer || got != "" {
			t.Fatalf("disabled probing emitted report: %v %q", answer, got)
		}
		s.probing.enabled = true
		answer, err = s.probing.run(true)
		satOK(t, err)
		got, err = out.String()
		satOK(t, err)
		want := regexp.MustCompile(fmt.Sprintf(`^ \(sat-probing :probing-assigned 1 :cost %d :memory [0-9]+\.[0-9]{2} :time [0-9]+\.[0-9]{2}\)\n$`, s.propagation.probingCounter))
		if !answer || !want.MatchString(got) {
			t.Fatalf("source probing assigned report: %v %q", answer, got)
		}
	})
	satOK(t, out.ResetText())

	t.Run("stopped", func(t *testing.T) {
		s, _, _ := satSolverCheckFixture(t)
		for range 2 {
			_, err := s.MkVar(false, true)
			satOK(t, err)
		}
		s.probing.limit = 0
		answer, err := s.probing.run(true)
		satOK(t, err)
		got, err := out.String()
		satOK(t, err)
		want := regexp.MustCompile(`^ \(sat-probing :probing-assigned 0 :cost 4 :stopped-at 1 :memory [0-9]+\.[0-9]{2} :time [0-9]+\.[0-9]{2}\)\n$`)
		if answer || s.probing.stoppedAt != 1 || !want.MatchString(got) {
			t.Fatalf("source probing stopped report: %v %q", answer, got)
		}
	})
	satOK(t, out.ResetText())
	t.Run("canceled", func(t *testing.T) {
		s, _, _ := satSolverCheckFixture(t)
		_, err := s.MkVar(false, true)
		satOK(t, err)
		s.Limit.SetCancel(1)
		answer, err := s.probing.run(true)
		if answer || !errors.Is(err, ErrSATSolverCanceled) {
			t.Fatalf("source checkpoint: %v %v", answer, err)
		}
		got, err := out.String()
		satOK(t, err)
		want := regexp.MustCompile(`^ \(sat-probing :probing-assigned 0 :cost 0 :memory [0-9]+\.[0-9]{2} :time [0-9]+\.[0-9]{2}\)\n$`)
		if !want.MatchString(got) {
			t.Fatalf("source probing unwind report: %q", got)
		}
		s.Limit.ResetCancel()
	})
	satOK(t, out.ResetText())
	satOK(t, diagnostic.SetProcessVerbosityLevel(9))
	s, _, _ := satSolverCheckFixture(t)
	answer, err := s.probing.run(true)
	satOK(t, err)
	got, err := out.String()
	satOK(t, err)
	if !answer || got != "" {
		t.Fatalf("source verbosity gate emitted report: %v %q", answer, got)
	}
}
