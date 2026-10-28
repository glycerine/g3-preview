package sat

// Source: src/sat/sat_simplifier.cpp:785-806,1058-1077,1438-1458.

import (
	"regexp"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATSimplifierReportsSourceScopesAndVerbosityGate(t *testing.T) {
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
	satOK(t, diagnostic.SetProcessVerbosityLevel(10))
	for range 3 {
		satOK(t, s.simplifier.Run(false))
	}
	got, err := out.String()
	satOK(t, err)
	if strings.Count(got, " (sat-subsumer ") != 3 || strings.Count(got, " (sat-resolution ") != 3 || strings.Count(got, " (sat-blocked-clauses ") != 1 {
		t.Fatalf("three source invocation scopes: %q", got)
	}
	for _, line := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
		want := regexp.MustCompile(`^ \(sat-(subsumer|resolution|blocked-clauses) :.* :memory [0-9]+\.[0-9]{2} :time [0-9]+\.[0-9]{2}\)$`)
		if !want.MatchString(line) {
			t.Fatalf("source report record: %q", line)
		}
	}
	if !strings.Contains(got, " (sat-blocked-clauses :elim-blocked-clauses 0 :memory ") {
		t.Fatalf("source blocked report: %q", got)
	}
	satOK(t, out.ResetText())
	satOK(t, diagnostic.SetProcessVerbosityLevel(9))
	satOK(t, s.simplifier.Run(false))
	got, err = out.String()
	satOK(t, err)
	if got != "" {
		t.Fatalf("source verbosity gate emitted reports: %q", got)
	}
}

func TestSATSimplifierReportsActualSubsumptionAndResolution(t *testing.T) {
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
	satOK(t, diagnostic.SetProcessVerbosityLevel(10))
	for range 4 {
		_, err = s.MkVar(false, true)
		satOK(t, err)
	}
	for i := uint32(1); i < 4; i++ {
		satOK(t, s.Variables.setExternal(BoolVar(i), true))
	}
	first, err := s.Lifecycle.MkClauseCore(3, []Literal{0, 2, 4}, false)
	satOK(t, err)
	_, err = s.Lifecycle.MkClauseCore(3, []Literal{1, 2, 4}, false)
	satOK(t, err)
	_, err = s.Lifecycle.MkClauseCore(4, []Literal{0, 2, 4, 6}, false)
	satOK(t, err)
	satOK(t, s.Clauses.MarkStrengthened(first))
	satOK(t, s.simplifier.Run(false))
	if s.simplifier.numSubsumed == 0 || s.simplifier.numElimVars != 1 {
		t.Fatal("source simplification result", s.simplifier.numSubsumed, s.simplifier.numElimVars)
	}
	got, err := out.String()
	satOK(t, err)
	sub := regexp.MustCompile(`(?m)^ \(sat-subsumer :subsumed [1-9][0-9]* :subsumption-resolution [0-9]+ :threshold -?[0-9]+ :memory [0-9]+\.[0-9]{2} :time [0-9]+\.[0-9]{2}\)$`)
	elim := regexp.MustCompile(`(?m)^ \(sat-resolution :elim-bool-vars 1 :threshold -?[0-9]+ :memory [0-9]+\.[0-9]{2} :time [0-9]+\.[0-9]{2}\)$`)
	if !sub.MatchString(got) || !elim.MatchString(got) {
		t.Fatalf("source nonzero reports: %q", got)
	}
}
