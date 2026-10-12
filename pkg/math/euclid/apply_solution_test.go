package euclid

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestEuclideanApplySolutionChainedSolvedQueueAndOwnedSwap(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	for i := 0; i < 3; i++ {
		_, e := s.mkVar(false)
		euclidMust(t, e)
		_, e = s.mkJustification()
		euclidMust(t, e)
	}
	add := func(coefficients []int32, xs []uint32, constant int32, rational string, j uint32) *equation {
		t.Helper()
		as := make([]numeral.Int, len(coefficients))
		for i, coefficient := range coefficients {
			as[i] = numeral.NewInt(coefficient)
		}
		c := numeral.NewInt(constant)
		bs := []numeral.Rational{numeral.NewRational()}
		euclidMust(t, m.SetString(&bs[0], rational))
		eq, e := s.addEquation(as, xs, &c, bs, []uint32{j}, true)
		euclidMust(t, e)
		for i := range as {
			euclidMust(t, z.Del(&as[i]))
		}
		euclidMust(t, z.Del(&c))
		euclidMust(t, m.Del(&bs[0]))
		return eq
	}
	// x0 = x1+1 and x1 = x2+2. The target contains only x0 initially;
	// substitution must enqueue the newly exposed solved x1.
	sol0 := add([]int32{-1, 1}, []uint32{0, 1}, 1, "1/2", 0)
	euclidMust(t, s.promoteSolvedEquation(0, 0))
	sol1 := add([]int32{-1, 1}, []uint32{1, 2}, 2, "1/3", 1)
	euclidMust(t, s.promoteSolvedEquation(1, 1))
	target := add([]int32{3}, []uint32{0}, 5, "1/5", 2)
	euclidMust(t, s.applySolutionToEquation(target))
	if !equalUint32Slices(target.xs, []uint32{2}) || !equalUint32Slices(target.js, []uint32{0, 1, 2}) || !s.queue.Empty() {
		t.Fatalf("substituted IDs or queue: %v, %v, empty=%v", target.xs, target.js, s.queue.Empty())
	}
	for i, want := range []string{"3"} {
		got, e := z.String(&target.as[i])
		euclidMust(t, e)
		if got != want {
			t.Fatalf("target coefficient[%d] = %s", i, got)
		}
	}
	constant, e := z.String(&target.c)
	euclidMust(t, e)
	if constant != "14" {
		t.Fatalf("substituted constant = %s", constant)
	}
	for i, want := range []string{"3/2", "1", "1/5"} {
		got, e := m.String(&target.bs[i])
		euclidMust(t, e)
		if got != want {
			t.Fatalf("justification[%d] = %s, want %s", i, got, want)
		}
	}
	if sol0.closed || sol1.closed || s.equations[0] != nil || s.equations[1] != nil {
		t.Fatal("solution ownership transfer did not preserve live equations")
	}
	// Reapplying the same solution set must leave the normalized target alone.
	euclidMust(t, s.applySolutionToEquation(target))
	if !equalUint32Slices(target.xs, []uint32{2}) || !equalUint32Slices(target.js, []uint32{0, 1, 2}) {
		t.Fatal("repeat substitution changed target")
	}
	euclidMust(t, s.Close())
	if !sol0.closed || !sol1.closed || !target.closed {
		t.Fatal("source equation/solution destructor missed an owner")
	}
}
