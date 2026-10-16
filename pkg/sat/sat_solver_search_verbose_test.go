// Source: src/sat/sat_solver.cpp:1191-1196,1347-1355.
package sat

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSATSolverSimplifyAndRestartSourceVerboseLevels(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	out := satInspectionStream(t, arena.NewBudget(8192))
	oldLevel, err := diagnostic.ProcessVerbosityLevel()
	satOK(t, err)
	oldStream, err := diagnostic.BorrowProcessVerboseStream()
	satOK(t, err)
	t.Cleanup(func() {
		satOK(t, diagnostic.SetProcessVerboseStream(oldStream))
		satOK(t, diagnostic.SetProcessVerbosityLevel(oldLevel))
	})
	satOK(t, diagnostic.SetProcessVerboseStream(out))
	satOK(t, diagnostic.SetProcessVerbosityLevel(2))
	satOK(t, s.simplifyProblem())
	got, err := out.String()
	satOK(t, err)
	if got != "(sat.simplify)\n" {
		t.Fatalf("source simplify notice: %q", got)
	}
	satOK(t, out.ResetText())
	s.nextSimplify = s.conflict.conflicts + 1
	satOK(t, s.simplifyProblem())
	got, err = out.String()
	satOK(t, err)
	if got != "" {
		t.Fatalf("early simplify exit emitted notice: %q", got)
	}
	satOK(t, diagnostic.SetProcessVerbosityLevel(1))
	satOK(t, s.gc.initSearch())
	s.conflict.conflicts = 7
	s.decision.decisions = 3
	satOK(t, s.gc.restart())
	got, err = out.String()
	satOK(t, err)
	want := regexp.MustCompile(`^\(sat-restart :conflicts 7 :decisions 3 :restarts 1 :gc-clause 0 :memory [0-9]+\.[0-9]{2} :time [0-9]+\.[0-9]{2}\)\n$`)
	if !want.MatchString(got) || !s.gc.watch.Running() {
		t.Fatalf("source restart report and running stopwatch: %q", got)
	}
	satOK(t, out.ResetText())
	satOK(t, diagnostic.SetProcessVerbosityLevel(30))
	satOK(t, s.gc.restart())
	got, err = out.String()
	satOK(t, err)
	want = regexp.MustCompile(`^\(sat-restart :conflicts 7 :decisions 3 :restarts 2 :gc-clause 0 :memory [0-9]+\.[0-9]{2} :time [0-9]+\.[0-9]{2}\)\n`)
	if !want.MatchString(got) || !strings.Contains(got, "(sat-status\n  :inconsistent    false\n") {
		t.Fatalf("source level-thirty status after restart: %q", got)
	}
}

func TestSATSolverCheckSourceEntryAndZeroConflictAbortVerbosity(t *testing.T) {
	for _, level := range []uint32{0, 2, 10} {
		t.Run(strconv.FormatUint(uint64(level), 10), func(t *testing.T) {
			s, c, p := satSolverCheckFixture(t)
			out := satInspectionStream(t, arena.NewBudget(8192))
			oldLevel, err := diagnostic.ProcessVerbosityLevel()
			satOK(t, err)
			oldStream, err := diagnostic.BorrowProcessVerboseStream()
			satOK(t, err)
			t.Cleanup(func() {
				satOK(t, diagnostic.SetProcessVerboseStream(oldStream))
				satOK(t, diagnostic.SetProcessVerbosityLevel(oldLevel))
			})
			satOK(t, diagnostic.SetProcessVerboseStream(out))
			satOK(t, diagnostic.SetProcessVerbosityLevel(level))
			k := "max_conflicts"
			satOK(t, c.SetUIntName(&p, &k, 0))
			satOK(t, s.UpdateParams(&p))
			result, err := s.Check(nil)
			satOK(t, err)
			if result != lbool.Undef {
				t.Fatal("zero-conflict source return", result)
			}
			got, err := out.String()
			satOK(t, err)
			if level == 0 && got != "" {
				t.Fatal("disabled verbosity emitted output", got)
			}
			if level >= 2 && !strings.HasPrefix(got, "(sat.sat-solver)\n") {
				t.Fatal("check entry missing before search", got)
			}
			abort := "(sat \"abort: max-conflicts = 0\")\n"
			if (level >= 10) != strings.Contains(got, abort) {
				t.Fatal("source abort gate or text", got)
			}
		})
	}
}
