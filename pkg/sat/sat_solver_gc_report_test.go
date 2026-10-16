// Source: src/sat/sat_solver.cpp:1460-1512,1518-1607.
package sat

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATSolverGCReportsStrategiesDynamicCountsAndGate(t *testing.T) {
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
	for _, tc := range []struct {
		strategy satGCStrategy
		name     string
	}{
		{satGCGlue, "glue"}, {satGCPSM, "psm"},
		{satGCGluePSM, "glue-psm"}, {satGCPSMGlue, "psm-glue"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, conflict := satGCFixture(t, 3)
			g.lifecycle().propagation.config.state.GCStrategy = tc.strategy
			for _, glue := range []uint32{4, 1, 3, 2} {
				satGCClause(t, g, []Literal{0, 2, 4}, glue)
			}
			conflict.conflictsSinceGC = g.gcThreshold + 1
			satOK(t, g.gc())
			got, err := out.String()
			satOK(t, err)
			want := fmt.Sprintf("(sat-gc :strategy %s :deleted 2)\n", tc.name)
			if got != want {
				t.Fatalf("source strategy report: got %q, want %q", got, want)
			}
			satOK(t, out.ResetText())
		})
	}
	t.Run("dynamic", func(t *testing.T) {
		g, conflict := satGCFixture(t, 3)
		p := g.lifecycle().propagation
		p.config.state.GCStrategy = satGCDynPSM
		p.config.state.GCK = 0
		p.config.state.Initialized |= satConfigGCKMask
		r := satGCClause(t, g, []Literal{0, 2, 4}, 5)
		satOK(t, g.lifecycle().detachClause(r))
		satOK(t, p.clauses.Freeze(r))
		g.numFrozen = 1
		g.minDTk = 0
		for i := uint32(0); i < 3; i++ {
			satOK(t, p.variables.setPhase(BoolVar(i), 0))
			satOK(t, p.variables.setPreviousPhase(BoolVar(i), 0))
		}
		conflict.conflictsSinceGC = g.gcThreshold + 1
		satOK(t, g.gc())
		got, err := out.String()
		satOK(t, err)
		want := regexp.MustCompile(`^\(sat-gc :d_tk 4(?:\.0+)? :min-d_tk 0(?:\.0+)? :frozen 0 :activated 0 :deleted 1\)\n$`)
		if !want.MatchString(got) {
			t.Fatalf("source dynamic report: %q", got)
		}
		satOK(t, out.ResetText())
	})
	t.Run("dynamic-activated", func(t *testing.T) {
		g, conflict := satGCFixture(t, 3)
		p := g.lifecycle().propagation
		p.config.state.GCStrategy = satGCDynPSM
		r := satGCClause(t, g, []Literal{0, 2, 4}, 5)
		satOK(t, g.lifecycle().detachClause(r))
		satOK(t, p.clauses.Freeze(r))
		g.numFrozen, g.minDTk = 1, 0
		conflict.conflictsSinceGC = g.gcThreshold + 1
		satOK(t, g.gc())
		got, err := out.String()
		satOK(t, err)
		want := regexp.MustCompile(`^\(sat-gc :d_tk 4(?:\.0+)? :min-d_tk 0(?:\.0+)? :frozen 0 :activated 1 :deleted 0\)\n$`)
		if !want.MatchString(got) {
			t.Fatalf("source dynamic activation report: %q", got)
		}
		satOK(t, out.ResetText())
	})
	t.Run("dynamic-frozen", func(t *testing.T) {
		g, conflict := satGCFixture(t, 3)
		p := g.lifecycle().propagation
		p.config.state.GCStrategy = satGCDynPSM
		p.config.state.GCK = 10
		p.config.state.GCSmallLBD = 2
		p.config.state.Initialized |= satConfigGCKMask | satConfigGCSmallLBDMask
		satGCClause(t, g, []Literal{0, 2, 4}, 5)
		g.minDTk = 0
		for i := uint32(0); i < 3; i++ {
			satOK(t, p.variables.setPhase(BoolVar(i), 0))
			satOK(t, p.variables.setPreviousPhase(BoolVar(i), 0))
		}
		conflict.conflictsSinceGC = g.gcThreshold + 1
		satOK(t, g.gc())
		got, err := out.String()
		satOK(t, err)
		want := regexp.MustCompile(`^\(sat-gc :d_tk 4(?:\.0+)? :min-d_tk 0(?:\.0+)? :frozen 1 :activated 0 :deleted 0\)\n$`)
		if !want.MatchString(got) {
			t.Fatalf("source dynamic freeze report: %q", got)
		}
		satOK(t, out.ResetText())
	})
	satOK(t, diagnostic.SetProcessVerbosityLevel(9))
	g, conflict := satGCFixture(t, 3)
	satGCClause(t, g, []Literal{0, 2, 4}, 1)
	conflict.conflictsSinceGC = g.gcThreshold + 1
	satOK(t, g.gc())
	got, err := out.String()
	satOK(t, err)
	if got != "" {
		t.Fatalf("source GC verbosity gate: %q", got)
	}
}
