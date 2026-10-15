package euclid

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func euclideanTraceRecord(function string, line int, body string) string {
	return fmt.Sprintf("-------- [euclidean_solver] %s src/math/euclid/euclidean_solver.cpp:%d ---------\n%s------------------------------------------------\n", function, line, body)
}

type euclideanPublicationTraceWriter struct {
	output  bytes.Buffer
	solver  *solverSetup
	counts  []int
	pending []bool
}

func (w *euclideanPublicationTraceWriter) Write(p []byte) (int, error) {
	if string(p) == "-------- [" {
		w.counts = append(w.counts, len(w.solver.equations))
		w.pending = append(w.pending, w.solver.pendingEq != nil)
	}
	return w.output.Write(p)
}

func TestEuclideanNewEquationTraceBeforeListPublication(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, err := newSolverSetup(m)
	euclidMust(t, err)
	_, err = s.mkVar(false)
	euclidMust(t, err)
	w := &euclideanPublicationTraceWriter{solver: s}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("euclidean_solver")
	s.trace = tags
	coefficient, constant := numeral.NewInt(2), numeral.NewInt(4)
	euclidMust(t, s.assertEquation([]numeral.Int{coefficient}, []uint32{0}, &constant, nullEquationIndex))
	if tags.ShouldTrace("euclidean_solver") {
		if len(w.counts) != 4 || w.counts[0] != 0 || !w.pending[0] {
			t.Fatal("new-eq TRACE ran after equation-list publication", w.counts, w.pending)
		}
		for i := 1; i < len(w.counts); i++ {
			if w.counts[i] != 1 || w.pending[i] {
				t.Fatal("later TRACE missed published equation", w.counts, w.pending)
			}
		}
	} else if len(w.counts) != 0 || w.output.Len() != 0 {
		t.Fatal("erased trace observed equation publication")
	}
	if s.pendingEq != nil || len(s.equations) != 1 {
		t.Fatal("completed assertion retained local equation owner")
	}
	euclidMust(t, s.Close())
	euclidMust(t, z.Del(&coefficient))
	euclidMust(t, z.Del(&constant))
}

func TestEuclideanAssertNormalizeTraceDivisionAndGCDContradiction(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, err := newSolverSetup(m)
	euclidMust(t, err)
	for i := 0; i < 2; i++ {
		_, err = s.mkVar(false)
		euclidMust(t, err)
	}
	j, err := s.mkJustification()
	euclidMust(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("euclidean_solver")
	s.trace = tags
	as := []numeral.Int{numeral.NewInt(-6), numeral.NewInt(9)}
	c := numeral.NewInt(12)
	euclidMust(t, s.assertEquation(as, []uint32{0, 1}, &c, j))
	bad, badC := numeral.NewInt(4), numeral.NewInt(6)
	euclidMust(t, s.assertEquation([]numeral.Int{bad}, []uint32{0}, &badC, nullEquationIndex))
	if s.inconsistent != 1 {
		t.Fatal("gcd contradiction not published")
	}
	if tags.ShouldTrace("euclidean_solver") {
		const initial = "1*j0 |= -6*x0 + 9*x1 + 12 = 0\n"
		const divided = "1/3*j0 |= -2*x0 + 3*x1 + 12 = 0\n"
		const contradictory = "|= 4*x0 + 6 = 0\n"
		want := euclideanTraceRecord("assert_eq", 559, "new-eq:\n"+initial) +
			euclideanTraceRecord("normalize_eq", 184, "normalizing:\n"+initial) +
			euclideanTraceRecord("normalize_eq", 222, "after normalization:\n"+divided) +
			euclideanTraceRecord("assert_eq", 565, "asserted:\n"+divided) +
			euclideanTraceRecord("assert_eq", 559, "new-eq:\n"+contradictory) +
			euclideanTraceRecord("normalize_eq", 184, "normalizing:\n"+contradictory) +
			euclideanTraceRecord("normalize_eq", 214, "gcd inconsistency detected\n") +
			euclideanTraceRecord("assert_eq", 565, "asserted:\n"+contradictory)
		if got := output.String(); got != want {
			t.Fatalf("assert/normalize source TRACE order: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased assert/normalize TRACE emitted", output.String())
	}
	euclidMust(t, s.Close())
	for i := range as {
		euclidMust(t, z.Del(&as[i]))
	}
	for _, value := range []*numeral.Int{&c, &bad, &badC} {
		euclidMust(t, z.Del(value))
	}
}

func TestEuclideanConstantOnlyTraceContradictionAndDeletedZero(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, err := newSolverSetup(m)
	euclidMust(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("euclidean_solver")
	s.trace = tags
	zero, two := numeral.NewInt(0), numeral.NewInt(2)
	euclidMust(t, s.assertEquation(nil, nil, &zero, nullEquationIndex))
	euclidMust(t, s.assertEquation(nil, nil, &two, nullEquationIndex))
	if s.equations[0] != nil || s.inconsistent != 1 {
		t.Fatal("constant-only source branches changed")
	}
	if tags.ShouldTrace("euclidean_solver") {
		want := euclideanTraceRecord("assert_eq", 559, "new-eq:\n|= 0 = 0\n") +
			euclideanTraceRecord("normalize_eq", 184, "normalizing:\n|= 0 = 0\n") +
			euclideanTraceRecord("assert_eq", 559, "new-eq:\n|= 2 = 0\n") +
			euclideanTraceRecord("normalize_eq", 184, "normalizing:\n|= 2 = 0\n") +
			euclideanTraceRecord("normalize_eq", 189, "c = 0 inconsistency detected\n") +
			euclideanTraceRecord("assert_eq", 565, "asserted:\n|= 2 = 0\n")
		if got := output.String(); got != want {
			t.Fatalf("constant-only source TRACE order: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased constant TRACE emitted", output.String())
	}
	euclidMust(t, s.Close())
	euclidMust(t, z.Del(&zero))
	euclidMust(t, z.Del(&two))
}
