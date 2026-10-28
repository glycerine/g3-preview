package lp

// Source: src/util/lp/lar_solver.h:1069-1081,1347-1357.

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func lpLARConstraintEvalFixture(t *testing.T, h *lpPairHarness, variant string, world int, large bool, kind LConstraintKind, offset int64) (*LARSolver, *LARRationalModel, LARStoredConstraint, *big.Rat) {
	t.Helper()
	s, err := NewLARSolver(h.m)
	h.must(err)
	model := NewLARRationalModel(h.m)
	x, y := big.NewRat(int64(world-19), 7), big.NewRat(int64(world%11-5), 13)
	a, b, extra, constant := big.NewRat(2, 3), big.NewRat(-3, 5), big.NewRat(7, 11), big.NewRat(int64(world-17), 23)
	if large {
		x = mustLPRat(t, strings.Repeat("1234567890", 20)+"/7")
		y = mustLPRat(t, strings.Repeat("9876543210", 20)+"/13")
		a = mustLPRat(t, strings.Repeat("3141592653", 20)+"/17")
		constant = mustLPRat(t, strings.Repeat("2718281828", 20)+"/19")
	}
	for j, value := range []*big.Rat{x, y, big.NewRat(11, 17)} {
		h.must(model.set(uint32(j), &h.pair(value.RatString(), "0").X))
	}
	lhs := new(big.Rat).Add(new(big.Rat).Mul(new(big.Rat).Add(a, extra), x), new(big.Rat).Mul(b, y))
	if variant == "variable" {
		lhs.Set(y)
	} else if variant == "term" {
		lhs.Add(lhs, constant)
	}
	rhs := h.pair(new(big.Rat).Add(lhs, big.NewRat(offset, 29)).RatString(), "0")
	input := NewSparseRationalVector(h.m)
	for _, entry := range []struct {
		j uint32
		v *big.Rat
	}{{0, a}, {1, b}, {0, extra}, {2, new(big.Rat)}} {
		h.must(input.Append(entry.j, &h.pair(entry.v.RatString(), "0").X))
	}
	var c LARStoredConstraint
	switch variant {
	case "variable":
		c, err = NewLARVarConstraint(h.m, 1, kind, &rhs.X)
	case "term":
		term, e := NewLARTermFromCoefficients(h.m, input.Data, &h.pair(constant.RatString(), "0").X)
		h.must(e)
		s.Terms = append(s.Terms, term)
		c, err = NewLARTermConstraint(h.m, term, kind, &rhs.X)
	case "general":
		c, err = NewLARConstraint(h.m, input.Data, kind, &rhs.X)
	default:
		t.Fatal(variant)
	}
	h.must(err)
	s.Constraints = append(s.Constraints, c)
	h.must(input.Close())
	return s, model, c, lhs
}

func TestLARConstraintEvalIndependentRelations(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for world := 0; world < 40; world++ {
		for _, variant := range []string{"variable", "term", "general"} {
			for _, kind := range []LConstraintKind{LConstraintLE, LConstraintLT, LConstraintGE, LConstraintGT, LConstraintEQ} {
				for _, offset := range []int64{-1, 0, 1} {
					t.Run(fmt.Sprintf("%d/%s/%d/%d", world, variant, kind, offset), func(t *testing.T) {
						h := newLPPairHarness(t)
						s, model, c, lhs := lpLARConstraintEvalFixture(t, h, variant, world, false, kind, offset)
						out := h.pair("97", "0")
						h.must(s.GetLeftSideVal(c, model, &out.X))
						lpSparseRationalCheck(t, h, &out.X, lhs)
						holds, err := s.ConstraintHolds(c, model)
						h.must(err)
						want := false
						switch kind {
						case LConstraintLE:
							want = offset >= 0
						case LConstraintLT:
							want = offset > 0
						case LConstraintGE:
							want = offset <= 0
						case LConstraintGT:
							want = offset < 0
						case LConstraintEQ:
							want = offset == 0
						}
						if holds != want || s.rationalWork != nil || s.validationCoefficients != nil {
							t.Fatal("independent relation and completed lexical owners", holds, want)
						}
						h.must(s.Close())
						h.must(model.Close())
						lpExactVectorNamedCensus(t, h, nil)
					})
				}
			}
		}
	}
}

func TestLARConstraintEvalReachedPrefixes(t *testing.T) {
	for _, phase := range []string{"stopped", "nil-constraint", "nil-model", "nil-output", "foreign-model", "stopped-model", "stopped-constraint", "missing", "unknown"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			s, model, c, _ := lpLARConstraintEvalFixture(t, h, "general", 0, false, LConstraintGE, 0)
			constraint, argument := c, model
			result := h.pair("97", "0")
			out := &result.X
			want := error(ErrExactVectorPending)
			switch phase {
			case "stopped":
				s.stopped = true
			case "nil-constraint":
				constraint = nil
				want = ErrExactVectorOwner
			case "nil-model":
				argument = nil
				want = ErrExactVectorOwner
			case "nil-output":
				out = nil
				want = ErrBounds
			case "foreign-model":
				argument = NewLARRationalModel(nil)
				want = ErrExactVectorOwner
			case "stopped-model":
				model.stopped = true
			case "stopped-constraint":
				c.Base().stopped = true
			case "missing":
				h.must(h.m.Del(model.Values[2]))
				delete(model.Values, 2) // General constraint retains this zero coefficient.
				want = ErrBounds
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "unknown":
				c.Base().Kind = LConstraintKind(255)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			}
			var err error
			if phase == "unknown" {
				var holds bool
				holds, err = s.ConstraintHolds(constraint, argument)
				if holds {
					t.Fatal("unknown relation holds")
				}
			} else {
				err = s.GetLeftSideVal(constraint, argument, out)
			}
			if err != want {
				t.Fatal("reached prefix", phase, err, want)
			}
			lpSparseRationalCheck(t, h, &result.X, big.NewRat(97, 1))
			if err != nil {
				if _, err = s.ConstraintHolds(c, model); err != ErrExactVectorPending {
					t.Fatal("failed evaluator accepted retry", err)
				}
			}
			h.must(s.Close())
			h.must(model.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARConstraintEvalPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, variant := range []string{"variable", "term", "general"} {
		for _, direct := range []bool{false, true} {
			for _, export := range []bool{false, true} {
				done, failures := false, 0
				for ordinal := int64(0); ordinal < 2000 && !done; ordinal++ {
					t.Run(fmt.Sprintf("%s/%v/%v/%d", variant, direct, export, ordinal), func(t *testing.T) {
						var h *lpPairHarness
						if direct {
							h = lpNewDirectSparseHarness(t)
						} else {
							h = newLPPairHarness(t)
						}
						s, model, c, lhs := lpLARConstraintEvalFixture(t, h, variant, 0, true, LConstraintGE, -1)
						out := h.pair("97", "0")
						h.budget.FailAfter(ordinal)
						var err error
						var holds bool
						if export {
							err = s.GetLeftSideVal(c, model, &out.X)
						} else {
							holds, err = s.ConstraintHolds(c, model)
						}
						h.budget.FailAfter(-1)
						if err == nil {
							done = true
							if export {
								lpSparseRationalCheck(t, h, &out.X, lhs)
							} else if !holds {
								t.Fatal("independent strict RHS margin")
							}
						} else {
							failures++
							if !lpExactPriorityPhysicalRefusal(err) || s.GetLeftSideVal(c, model, &out.X) != ErrExactVectorPending {
								t.Fatal("numerical refusal/stopped owner", err)
							}
						}
						h.must(s.Close())
						h.must(model.Close())
						lpExactVectorNamedCensus(t, h, nil)
					})
				}
				if !done || direct && failures == 0 {
					t.Fatal("physical campaign", variant, direct, export, failures)
				}
			}
		}
	}
}

func TestLARConstraintEvalExtractedStrictTermModel(t *testing.T) {
	h := newLPPairHarness(t)
	s, err := NewLARSolver(h.m)
	h.must(err)
	_, err = s.AddVar(17)
	h.must(err)
	_, err = s.AddVar(31)
	h.must(err)
	h.must(s.Core.RX.Data[0].Copy(h.m, h.pair("2", "1")))
	h.must(s.Core.RX.Data[1].Copy(h.m, h.pair("3", "-1")))
	input := NewSparseRationalVector(h.m)
	h.must(input.Append(0, &h.pair("2", "0").X))
	h.must(input.Append(1, &h.pair("-3", "0").X))
	term, err := s.AddTerm(input.Data, &h.pair("5", "0").X)
	h.must(err)
	_, err = s.AddVarBound(0, LConstraintGE, &h.pair("2", "0").X)
	h.must(err)
	_, err = s.AddVarBound(1, LConstraintLE, &h.pair("3", "0").X)
	h.must(err)
	_, err = s.AddVarBound(term, LConstraintGT, &h.pair("0", "0").X)
	h.must(err)
	model := NewLARRationalModel(h.m)
	h.must(s.GetModel(model))
	// At epsilon1/2, (2,1) and (3,-1) collide; the actual extractor halves
	// epsilon to1/4. The original term includes its free coefficient5.
	lpSparseRationalCheck(t, h, model.Values[0], big.NewRat(9, 4))
	lpSparseRationalCheck(t, h, model.Values[1], big.NewRat(11, 4))
	lpSparseRationalCheck(t, h, model.Values[2], big.NewRat(-15, 4))
	h.must(s.Terms[0].SetValue(&h.pair("99", "0").X))
	for _, c := range s.Constraints {
		holds, err := s.ConstraintHolds(c, model)
		h.must(err)
		if !holds {
			t.Fatal("actual variable and strict original-term constraints")
		}
	}
	result := h.pair("97", "0")
	h.must(s.GetLeftSideVal(s.Constraints[2], model, &result.X))
	lpSparseRationalCheck(t, h, &result.X, big.NewRat(5, 4))
	h.must(input.Close())
	h.must(s.Close())
	h.must(model.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARConstraintEvalPreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			s, model, c, _ := lpLARConstraintEvalFixture(t, h, "term", 0, true, LConstraintGE, 0)
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			var caller exactEtaCalls
			before, err := caller.begin(h.m)
			h.must(err)
			h.budget.FailAfter(ordinal)
			_, err = h.m.Compare(&v.X, &v.Y)
			h.budget.FailAfter(-1)
			err = caller.after(h.m, before, err)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				matched = true
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				if _, err = s.ConstraintHolds(c, model); err == nil || s.calls.comparison != 0 || c.Base().calls.comparison != 0 {
					t.Fatal("adopted caller comparison", err)
				}
			}
			h.must(s.Close())
			h.must(model.Close())
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("consumed caller comparison")
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no actual older comparison")
	}
}
