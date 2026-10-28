package sat

// Source: src/sat/sat_cleaner.cpp:158-179.

import (
	"regexp"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATCleanerReportUsesProcessVerboseStreamAndSourceGate(t *testing.T) {
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
	_, err = s.Lifecycle.MkClauseCore(3, []Literal{0, 2, 4}, false)
	satOK(t, err)
	done, err := s.cleaner.run(true)
	satOK(t, err)
	if done {
		t.Fatal("unchanged trail reached report")
	}
	got, err := out.String()
	satOK(t, err)
	if got != "" {
		t.Fatal("early return emitted report", got)
	}
	satOK(t, s.propagation.assign(0, satJustification{}))
	done, err = s.cleaner.run(true)
	satOK(t, err)
	if !done {
		t.Fatal("assigned trail skipped cleaner")
	}
	got, err = out.String()
	satOK(t, err)
	want := regexp.MustCompile(`^ \(sat-cleaner :elim-literals 0 :elim-clauses 1 :cost 3 :gc-clause 0 :memory [0-9]+\.[0-9]{2} :time [0-9]+\.[0-9]{2}\)\n$`)
	if !want.MatchString(got) {
		t.Fatalf("source report: %q", got)
	}
	satOK(t, out.ResetText())
	satOK(t, diagnostic.SetProcessVerbosityLevel(9))
	_, err = s.MkVar(false, true)
	satOK(t, err)
	satOK(t, s.propagation.assign(2, satJustification{}))
	done, err = s.cleaner.run(true)
	satOK(t, err)
	if !done {
		t.Fatal("second assigned trail skipped cleaner")
	}
	got, err = out.String()
	satOK(t, err)
	if got != "" {
		t.Fatal("source verbosity gate emitted report", got)
	}
}
