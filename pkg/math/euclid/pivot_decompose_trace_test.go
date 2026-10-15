package euclid

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func euclideanPivotTraceRecord(function string, line int, body string) string {
	return "-------- [euclidean_solver] " + function + " src/math/euclid/euclidean_solver.cpp:" +
		strconv.Itoa(line) + " ---------\n" + body + "------------------------------------------------\n"
}

func TestEuclideanUnitPivotTraceSourceOrderAndEquations(t *testing.T) {
	m := euclidNumeralManager(t)
	s, err := newSolverSetup(m)
	euclidMust(t, err)
	for i := 0; i < 3; i++ {
		_, err = s.mkVar(false)
		euclidMust(t, err)
	}
	for i := 0; i < 2; i++ {
		_, err = s.mkJustification()
		euclidMust(t, err)
	}
	euclideanAssertSmall(t, s, []int32{1, 2}, []uint32{0, 1}, 1, 0)
	euclideanAssertSmall(t, s, []int32{2, 3}, []uint32{0, 2}, 4, 1)
	found, err := s.selectNextVar()
	euclidMust(t, err)
	if !found || s.nextX != 0 || s.nextEq != 0 {
		t.Fatal("unexpected unit pivot")
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("euclidean_solver")
	s.trace = tags
	euclidMust(t, s.eliminateUnit())
	if tags.ShouldTrace("euclidean_solver") {
		want := euclideanPivotTraceRecord("elim_unit", 605,
			"eliminating equation with unit coefficient:\n1*j0 |= 1*x0 + 2*x1 + 1 = 0\n") +
			euclideanPivotTraceRecord("substitute_most_recent_solution", 583,
				"applying solution for x0\n-1*j0 |= -1*x0 + -2*x1 + -1 = 0\n") +
			euclideanPivotTraceRecord("substitute_most_recent_solution", 592,
				"eq before substituting x0\n1*j1 |= 2*x0 + 3*x2 + 4 = 0\n") +
			euclideanPivotTraceRecord("substitute_most_recent_solution", 594,
				"eq after substituting x0\n-2*j0 1*j1 |= -4*x1 + 3*x2 + 2 = 0\n") +
			euclideanTraceRecord("normalize_eq", 184,
				"normalizing:\n-2*j0 1*j1 |= -4*x1 + 3*x2 + 2 = 0\n")
		if got := output.String(); got != want {
			t.Fatalf("unit pivot source TRACE order: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased unit pivot TRACE emitted", output.String())
	}
	euclidMust(t, s.Close())
}

type euclideanAuxPublicationWriter struct {
	output  bytes.Buffer
	solver  *solverSetup
	pending []bool
	sizes   []int
}

func (w *euclideanAuxPublicationWriter) Write(p []byte) (int, error) {
	if string(p) == "-------- [" {
		w.pending = append(w.pending, w.solver.pendingEq != nil)
		w.sizes = append(w.sizes, len(w.solver.solution))
	}
	return w.output.Write(p)
}

func TestEuclideanDecompositionTraceBeforeAuxiliaryPublication(t *testing.T) {
	m := euclidNumeralManager(t)
	s, err := newSolverSetup(m)
	euclidMust(t, err)
	for i := 0; i < 2; i++ {
		_, err = s.mkVar(false)
		euclidMust(t, err)
	}
	euclideanAssertSmall(t, s, []int32{2, 3}, []uint32{0, 1}, 4, nullEquationIndex)
	found, err := s.selectNextVar()
	euclidMust(t, err)
	if !found || s.nextX != 0 || s.nextEq != 0 {
		t.Fatal("unexpected non-unit pivot")
	}
	w := &euclideanAuxPublicationWriter{solver: s}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("euclidean_solver")
	s.trace = tags
	euclidMust(t, s.decomposeAndEliminate())
	if tags.ShouldTrace("euclidean_solver") {
		want := euclideanPivotTraceRecord("decompose_and_elim", 653,
			"decompositing equation for x0\n|= 2*x0 + 3*x1 + 4 = 0\n") +
			euclideanPivotTraceRecord("decompose_and_elim", 699,
				"decomposition: new parameter x2 aux eq:\n|= -1*x0 + -1*x1 + 1*x2 + -2 = 0\n|= 1*x1 + 2*x2 + 0 = 0\n") +
			euclideanPivotTraceRecord("substitute_most_recent_solution", 583,
				"applying solution for x0\n|= -1*x0 + -1*x1 + 1*x2 + -2 = 0\n")
		if got := w.output.String(); got != want {
			t.Fatalf("decomposition source TRACE order: got %q, want %q", got, want)
		}
		if len(w.sizes) != 3 || w.sizes[0] != 0 || w.sizes[1] != 0 || !w.pending[1] ||
			w.sizes[2] != 1 || w.pending[2] {
			t.Fatal("auxiliary trace missed source publication boundary", w.sizes, w.pending)
		}
	} else if w.output.Len() != 0 || len(w.sizes) != 0 {
		t.Fatal("erased decomposition TRACE emitted")
	}
	if s.pendingEq != nil {
		t.Fatal("published auxiliary equation retained a pending owner")
	}
	euclidMust(t, s.Close())
}
