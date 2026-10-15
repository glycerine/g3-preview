package euclid

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestEuclideanApplySolutionTraceBeforeAndAfterIntegerSwap(t *testing.T) {
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
	add := func(coefficients []int32, xs []uint32, constant int32, proof string, js []uint32) *equation {
		t.Helper()
		as := make([]numeral.Int, len(coefficients))
		for i, value := range coefficients {
			as[i] = numeral.NewInt(value)
		}
		c := numeral.NewInt(constant)
		var bs []numeral.Rational
		if proof != "" {
			bs = []numeral.Rational{numeral.NewRational()}
			euclidMust(t, m.SetString(&bs[0], proof))
		}
		eq, e := s.addEquation(as, xs, &c, bs, js, true)
		euclidMust(t, e)
		for i := range as {
			euclidMust(t, z.Del(&as[i]))
		}
		euclidMust(t, z.Del(&c))
		for i := range bs {
			euclidMust(t, m.Del(&bs[i]))
		}
		return eq
	}
	add([]int32{-1, 1}, []uint32{0, 1}, 2, "1/2", []uint32{j})
	euclidMust(t, s.promoteSolvedEquation(0, 0))
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("euclidean_solver_apply")
	s.trace = tags
	target := add([]int32{3}, []uint32{0}, 5, "", nil)
	euclidMust(t, s.applySolutionToEquation(target))
	if !equalUint32Slices(target.xs, []uint32{1}) || !equalUint32Slices(target.js, []uint32{j}) {
		t.Fatal("substitution did not publish the expected row", target.xs, target.js)
	}
	got, err := z.String(&target.c)
	euclidMust(t, err)
	if got != "11" {
		t.Fatal("substitution constant", got)
	}
	const footer = "------------------------------------------------\n"
	const before = "-------- [euclidean_solver_apply] apply_solution src/math/euclid/euclidean_solver.cpp:436 ---------\n" +
		"applying: 3 * 1/2*j0 |= -1*x0 + 1*x1 + 2 = 0\n3*x0 \n" + footer
	const after = "-------- [euclidean_solver_apply] apply_solution src/math/euclid/euclidean_solver.cpp:444 ---------\n" +
		"3*x1 \n" + footer
	if tags.ShouldTrace("euclidean_solver_apply") {
		if got := output.String(); got != before+after {
			t.Fatalf("source apply TRACE records: got %q, want %q", got, before+after)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased apply TRACE emitted", output.String())
	}
	refusing := diagnostic.NewTraceTagsWithWriter(refusedEuclideanTraceWriter{})
	refusing.Enable("euclidean_solver_apply")
	s.trace = refusing
	target = add([]int32{4}, []uint32{0}, 0, "", nil)
	euclidMust(t, s.applySolutionToEquation(target))
	got, err = z.String(&target.c)
	euclidMust(t, err)
	if got != "8" || !equalUint32Slices(target.xs, []uint32{1}) {
		t.Fatal("refused trace changed substitution", got, target.xs)
	}
	euclidMust(t, s.Close())
}
