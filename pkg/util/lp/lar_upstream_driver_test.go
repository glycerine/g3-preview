package lp

import (
	"fmt"
	"math/big"
	"testing"
)

func TestLARAddConstraintExecutableSourceProfiles(t *testing.T) {
	for _, kind := range []LConstraintKind{LConstraintGE, LConstraintGT, LConstraintLE, LConstraintLT, LConstraintEQ, LConstraintKind(99)} {
		for _, malformed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%v/%v", kind, malformed), func(t *testing.T) {
				h := newLPPairHarness(t)
				s, err := NewLARSolver(h.m)
				h.must(err)
				_, err = s.AddVar(17)
				h.must(err)
				ci, err := s.AddVarBound(0, LConstraintGE, &h.pair("3/7", "0").X)
				h.must(err)
				entries := []RationalSparseEntry{{Index: 0, Value: h.pair("7", "0").X}}
				rhs := &h.pair("-123/17", "0").X
				if malformed {
					// Neither the unregistered index nor the nil RHS is reached by
					// the executable source body.
					entries[0].Index = InvalidLARIndex
					rhs = nil
				}
				used := h.budget.Used()
				h.budget.FailAfter(0)
				got, err := s.AddConstraint(entries, kind, rhs)
				h.budget.FailAfter(-1)
				want := sourceAssertionFailure()
				if got != 0 || err != want || s.NumberOfConstraints() != 1 || h.budget.Used() != used {
					t.Fatal("source assertion/zero-return prefix", got, err, want)
				}
				w, err := s.Bounds.Witnesses.At(0)
				h.must(err)
				if w.LowBoundWitness != ci || w.UpperBoundWitness != InvalidLARIndex || s.Core.RA.RowCount() != 0 {
					t.Fatal("no-op changed preexisting constraints or matrix", w)
				}
				if sourceDebugEnabled {
					if _, err := s.AddConstraint(nil, kind, nil); err != ErrExactVectorPending {
						t.Fatal("assertion must stop the owner", err)
					}
				} else {
					witness, strict := uint32(99), true
					out := h.pair("0", "0")
					found, err := s.HasLowerBound(0, &witness, &out.X, &strict)
					h.must(err)
					if !found || strict || witness != ci {
						t.Fatal("native owner cannot continue", found, strict, witness)
					}
					lpSparseRationalCheck(t, h, &out.X, big.NewRat(3, 7))
				}
				h.must(s.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
	h := newLPPairHarness(t)
	s, err := NewLARSolver(h.m)
	h.must(err)
	s.stopped = true
	if _, err := s.AddConstraint(nil, LConstraintEQ, nil); err != ErrExactVectorPending {
		t.Fatal("stopped entry", err)
	}
	h.must(s.Close())
	if _, err := s.AddConstraint(nil, LConstraintEQ, nil); err != ErrExactVectorPending {
		t.Fatal("closed entry", err)
	}
	lpExactVectorNamedCensus(t, h, nil)
}

// Source: src/test/lp.cpp:2711-2739, test_term. The source does not install its
// displayed equality: debug stops at add_constraint, native solves the two
// unconstrained variables. Do not credit this driver as a constrained solve.
func TestLARUpstreamTermDriver(t *testing.T) {
	h := newLPPairHarness(t)
	s, err := NewLARSolver(h.m)
	h.must(err)
	x, err := s.AddVar(0)
	h.must(err)
	y, err := s.AddVar(1)
	h.must(err)
	one := h.pair("1", "0")
	term, err := s.AddTerm([]RationalSparseEntry{{Index: x, Value: one.X}, {Index: y, Value: one.X}}, &h.pair("3", "0").X)
	h.must(err)
	_, err = s.AddConstraint([]RationalSparseEntry{{Index: x, Value: one.X}, {Index: y, Value: one.X}, {Index: term, Value: one.X}}, LConstraintEQ, &h.pair("0", "0").X)
	if err != sourceAssertionFailure() || s.NumberOfConstraints() != 0 || len(s.OriginalTerms) != 1 {
		t.Fatal("upstream add_constraint prefix", err)
	}
	if !sourceDebugEnabled {
		status, err := s.Solve()
		h.must(err)
		if status != StatusOptimal {
			t.Fatal(status)
		}
		model := NewLARRationalModel(h.m)
		h.must(s.GetModel(model))
		// Solve materializes the deferred homogeneous term column as well.
		if len(model.Values) != 3 {
			t.Fatal("missing materialized term", len(model.Values))
		}
		for _, j := range []uint32{x, y, 2} {
			lpSparseRationalCheck(t, h, model.Values[j], new(big.Rat))
		}
		// With x=y=0 the original term is3, so x+y+term is3, not0.
		// This explicitly exposes the disabled source normalization body.
		h.must(model.Close())
	}
	h.must(s.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

// Source: src/test/lp.cpp:2741-2759, test_evidence_for_total_inf_simple. Its two
// general inequalities disappear in native; the remaining x<=-1,y>=0 are feasible.
func TestLARUpstreamTotalInfeasibilityDriver(t *testing.T) {
	h := newLPPairHarness(t)
	s, err := NewLARSolver(h.m)
	h.must(err)
	x, err := s.AddVar(0)
	h.must(err)
	y, err := s.AddVar(1)
	h.must(err)
	_, err = s.AddVarBound(x, LConstraintLE, &h.pair("-1", "0").X)
	h.must(err)
	_, err = s.AddVarBound(y, LConstraintGE, &h.pair("0", "0").X)
	h.must(err)
	entries := []RationalSparseEntry{{Index: x, Value: h.pair("1", "0").X}, {Index: y, Value: h.pair("1", "0").X}}
	_, err = s.AddConstraint(entries, LConstraintGE, &h.pair("1", "0").X)
	if err != sourceAssertionFailure() {
		t.Fatal(err)
	}
	if !sourceDebugEnabled {
		entries[1].Value = h.pair("-1", "0").X
		_, err = s.AddConstraint(entries, LConstraintGE, &h.pair("0", "0").X)
		h.must(err)
		status, err := s.Solve()
		h.must(err)
		if status != StatusOptimal || s.NumberOfConstraints() != 2 {
			t.Fatal("source native result", status, s.NumberOfConstraints())
		}
		model := NewLARRationalModel(h.m)
		h.must(s.GetModel(model))
		lpSparseRationalCheck(t, h, model.Values[x], big.NewRat(-1, 1))
		lpSparseRationalCheck(t, h, model.Values[y], new(big.Rat))
		h.must(model.Close())
	}
	h.must(s.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

// The same mathematical system installed through the retained SMT consumer's
// AddTerm/AddVarBound path really is infeasible. Keep this distinct from the
// literal upstream driver, whose AddConstraint calls have no native effect.
func TestLARUpstreamTotalInfeasibilityThroughRetainedBounds(t *testing.T) {
	h := newLPPairHarness(t)
	s, err := NewLARSolver(h.m)
	h.must(err)
	x, err := s.AddVar(0)
	h.must(err)
	y, err := s.AddVar(1)
	h.must(err)
	_, err = s.AddVarBound(x, LConstraintLE, &h.pair("-1", "0").X)
	h.must(err)
	_, err = s.AddVarBound(y, LConstraintGE, &h.pair("0", "0").X)
	h.must(err)
	for i, sign := range []string{"1", "-1"} {
		term, err := s.AddTerm([]RationalSparseEntry{{Index: x, Value: h.pair("1", "0").X}, {Index: y, Value: h.pair(sign, "0").X}}, &h.pair("0", "0").X)
		h.must(err)
		_, err = s.AddVarBound(term, LConstraintGE, &h.pair([]string{"1", "0"}[i], "0").X)
		h.must(err)
	}
	status, err := s.Solve()
	h.must(err)
	if status != StatusInfeasible {
		t.Fatal("x<=-1 and y>=0 contradict x-y>=0", status)
	}
	evidence := NewSparseRationalVector(h.m)
	h.must(s.GetInfeasibilityExplanation(evidence))
	ok, err := s.ExplanationIsCorrect(evidence)
	h.must(err)
	if !ok || len(evidence.Data) == 0 {
		t.Fatal("missing contradiction certificate")
	}
	lpCheckConflictCertificate(t, h, s, evidence)
	h.must(evidence.Close())
	h.must(s.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

// Source: src/test/lp.cpp:2760-2807, test_bound_propagation_one_small_sample1.
func TestLARUpstreamTrianglePropagationDriver(t *testing.T) {
	h := newLPPairHarness(t)
	s, err := NewLARSolver(h.m)
	h.must(err)
	for ext := uint32(0); ext < 3; ext++ {
		_, err := s.AddVar(ext)
		h.must(err)
	}
	one, minusOne, zero := h.pair("1", "0"), h.pair("-1", "0"), h.pair("0", "0")
	for _, j := range []uint32{2, 1} {
		_, err = s.AddTerm([]RationalSparseEntry{{Index: 0, Value: one.X}, {Index: j, Value: minusOne.X}}, &zero.X)
		h.must(err)
	}
	_, err = s.AddConstraint([]RationalSparseEntry{{Index: 0, Value: one.X}, {Index: 1, Value: minusOne.X}}, LConstraintLE, &zero.X)
	if err != sourceAssertionFailure() {
		t.Fatal(err)
	}
	if !sourceDebugEnabled {
		_, err = s.AddConstraint([]RationalSparseEntry{{Index: 1, Value: one.X}, {Index: 2, Value: minusOne.X}}, LConstraintLE, &zero.X)
		h.must(err)
		_, err = s.AddVarBound(0, LConstraintLE, &one.X)
		h.must(err)
		status, err := s.Solve()
		h.must(err)
		if status != StatusOptimal || s.NumberOfConstraints() != 1 || s.Core.RA.RowCount() != 2 {
			t.Fatal("triangle driver native state", status, s.NumberOfConstraints(), s.Core.RA.RowCount())
		}
		p := NewLARBoundPropagator(s.Bounds, &s.ColumnsToExternal, &s.Terms)
		h.must(s.PropagateBoundsForTouchedRows(p))
		// Both triangle inequalities were ignored. Unbounded b,c and the
		// unconstrained term columns permit no new interval endpoint.
		if p.NumberOfFoundBounds() != 0 {
			t.Fatal("invented source triangle constraint", p.NumberOfFoundBounds())
		}
		h.must(p.Close())
	}
	h.must(s.Close())
	lpExactVectorNamedCensus(t, h, nil)
}
