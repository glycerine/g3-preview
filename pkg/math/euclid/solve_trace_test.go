package euclid

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestEuclideanSolveTraceEntryCandidateAndPostPivotDisplay(t *testing.T) {
	m := euclidNumeralManager(t)
	s, err := newSolverSetup(m)
	euclidMust(t, err)
	_, err = s.mkVar(false)
	euclidMust(t, err)
	euclideanAssertSmall(t, s, []int32{1}, []uint32{0}, 1, nullEquationIndex)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("euclidean_solver")
	s.trace = tags
	consistent, err := s.solve()
	euclidMust(t, err)
	if !consistent || len(s.solution) != 1 || len(s.equations) != 0 {
		t.Fatal("unit solver result changed")
	}
	if tags.ShouldTrace("euclidean_solver") {
		want := euclideanTraceRecord("solve", 711,
			"solving...\nsolution set:\ntodo:\n|= 1*x0 + 1 = 0\n") +
			euclideanTraceRecord("solve", 714, "eliminating x0\n") +
			euclideanTraceRecord("elim_unit", 605,
				"eliminating equation with unit coefficient:\n|= 1*x0 + 1 = 0\n") +
			euclideanTraceRecord("substitute_most_recent_solution", 583,
				"applying solution for x0\n|= -1*x0 + -1 = 0\n") +
			euclideanTraceRecord("solve", 719,
				"solution set:\n|= -1*x0 + -1 = 0\ntodo:\n")
		if got := output.String(); got != want {
			t.Fatalf("solver-loop source TRACE order: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased solver-loop TRACE emitted", output.String())
	}
	euclidMust(t, s.Close())
}

func TestEuclideanSolveTracePostPivotInconsistencyDisplay(t *testing.T) {
	m := euclidNumeralManager(t)
	s, err := newSolverSetup(m)
	euclidMust(t, err)
	_, err = s.mkVar(false)
	euclidMust(t, err)
	euclideanAssertSmall(t, s, []int32{1}, []uint32{0}, 0, nullEquationIndex)
	euclideanAssertSmall(t, s, []int32{1}, []uint32{0}, 1, nullEquationIndex)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("euclidean_solver")
	s.trace = tags
	consistent, err := s.solve()
	euclidMust(t, err)
	if consistent || s.inconsistent == nullEquationIndex {
		t.Fatal("contradictory fixture unexpectedly solved")
	}
	if tags.ShouldTrace("euclidean_solver") {
		got := output.String()
		post := euclideanTraceRecord("solve", 719,
			"inconsistent: |= -1 = 0\nsolution set:\n|= -1*x0 + -1 = 0\ntodo:\n|= -1 = 0\n")
		if !strings.HasSuffix(got, post) {
			t.Fatalf("post-pivot TRACE missed contradiction before false return: got %q, suffix %q", got, post)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased contradiction TRACE emitted", output.String())
	}
	euclidMust(t, s.Close())
}
