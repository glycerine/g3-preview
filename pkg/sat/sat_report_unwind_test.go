// Source: src/sat/sat_cleaner.cpp:158-179; src/sat/sat_scc.cpp:43-58.
package sat

import (
	"errors"
	"regexp"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/stopwatch"
)

func TestSATReportsReadCompactStatsDuringStoppedContinuation(t *testing.T) {
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

	// An interrupted owner bars new public solver operations while a report
	// destructor still reads the source's already constructed stat members.
	s.Lifecycle.cleanerContinuation = s.cleaner
	s.cleaner.interrupted = true
	t.Cleanup(func() {
		s.Lifecycle.cleanerContinuation = nil
		s.cleaner.interrupted = false
	})
	if err = s.displayCompactStat(out); !errors.Is(err, arena.ErrClosed) {
		t.Fatalf("public inspection accepted stopped continuation: %v", err)
	}
	var watch stopwatch.Stopwatch
	satOK(t, watch.Start())
	satOK(t, watch.Stop())
	satOK(t, s.cleaner.emitReport(&watch, 0, 0))
	satOK(t, s.scc.emitReport())
	got, err := out.String()
	satOK(t, err)
	want := regexp.MustCompile(`^ \(sat-cleaner :elim-literals 0 :elim-clauses 0 :cost 0 :gc-clause 0 :memory [0-9]+\.[0-9]{2} :time [0-9]+\.[0-9]{2}\)\n \(sat-scc :elim-vars 0 :gc-clause 0 :memory [0-9]+\.[0-9]{2} :time 0\.00\)\n$`)
	if !want.MatchString(got) {
		t.Fatalf("source reports during stopped continuation: %q", got)
	}
}
