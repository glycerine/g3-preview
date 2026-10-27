// Source: src/util/lp/lar_solver.h:456-463,1090-1139,1158-1202,1242-1275.
package lp

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
)

func lpCheckConflictCertificate(t *testing.T, h *lpPairHarness, s *LARSolver, e *SparseRationalVector) {
	t.Helper()
	coefficients := map[uint32]*big.Rat{}
	rhs := new(big.Rat)
	var orientation int
	strict := false
	for i := range e.Data {
		cell := &e.Data[i]
		if uint64(cell.Index) >= uint64(len(s.Constraints)) {
			t.Fatal("missing witness", cell.Index)
		}
		con := s.Constraints[cell.Index]
		text, err := h.m.String(&cell.Value)
		h.must(err)
		a := mustLPRat(t, text)
		kind := con.Base().Kind
		if a.Sign() < 0 {
			kind = -kind
		}
		if kind != LConstraintEQ {
			sign := 1
			if kind < 0 {
				sign = -1
			}
			if orientation != 0 && orientation != sign {
				t.Fatal("mixed oriented inequalities")
			}
			orientation = sign
			strict = strict || kind == LConstraintLT || kind == LConstraintGT
		}
		v, err := con.CoefficientsAsVector()
		h.must(err)
		for k := range v.Data {
			term := &v.Data[k]
			text, err = h.m.String(&term.Value)
			h.must(err)
			p := new(big.Rat).Mul(a, mustLPRat(t, text))
			if coefficients[term.Index] == nil {
				coefficients[term.Index] = new(big.Rat)
			}
			coefficients[term.Index].Add(coefficients[term.Index], p)
		}
		h.must(v.Close())
		free := h.pair("0", "0")
		h.must(con.FreeCoefficient(&free.X))
		text, err = h.m.String(&free.X)
		h.must(err)
		f := mustLPRat(t, text)
		text, err = h.m.String(&con.Base().RightSide)
		h.must(err)
		rhs.Add(rhs, new(big.Rat).Mul(a, new(big.Rat).Sub(mustLPRat(t, text), f)))
	}
	for j, a := range coefficients {
		if a.Sign() != 0 {
			t.Fatal("independent zero LHS", j, a)
		}
	}
	valid := rhs.Sign() != 0
	if orientation < 0 {
		valid = rhs.Sign() < 0 || strict && rhs.Sign() == 0
	} else if orientation > 0 {
		valid = rhs.Sign() > 0 || strict && rhs.Sign() == 0
	}
	if !valid {
		t.Fatal("independent impossible relation", orientation, strict, rhs)
	}
}

func TestLARConflictCertificateActualSolves(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauRows, StrategyTableauCosts} {
		for world := 0; world < 30; world++ {
			for _, strict := range []bool{false, true} {
				t.Run(fmt.Sprintf("%d/%d/%v", strategy, world, strict), func(t *testing.T) {
					h := newLPPairHarness(t)
					s, _ := lpLARFullSolveFixture(t, h, strategy, world, true, strict)
					status, err := s.FindFeasibleSolution()
					h.must(err)
					if status != StatusInfeasible {
						t.Fatal(status)
					}
					e := NewSparseRationalVector(h.m)
					h.must(e.Append(999, &h.pair("123", "0").X))
					h.must(s.GetInfeasibilityExplanation(e))
					if len(e.Data) == 0 || e.Data[0].Index == 999 {
						t.Fatal("clear then extract")
					}
					lpCheckConflictCertificate(t, h, s, e)
					ok, err := s.ExplanationIsCorrect(e)
					h.must(err)
					if !ok {
						t.Fatal("source validation")
					}
					ok, err = s.InfExplanationIsCorrect()
					h.must(err)
					if !ok || s.validationExplanation != nil {
						t.Fatal("owned validation lexical vector")
					}
					h.must(e.Close())
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
			}
		}
	}
}

func TestLARConflictCertificateContradictoryColumn(t *testing.T) {
	for _, strict := range []bool{false, true} {
		h := newLPPairHarness(t)
		s, err := NewLARSolver(h.m)
		h.must(err)
		_, err = s.AddVar(17)
		h.must(err)
		low, upper := LConstraintGE, LConstraintLE
		if strict {
			low, upper = LConstraintGT, LConstraintLT
		}
		_, err = s.AddVarBound(0, low, &h.pair("2", "0").X)
		h.must(err)
		_, err = s.AddVarBound(0, upper, &h.pair("1", "0").X)
		h.must(err)
		e := NewSparseRationalVector(h.m)
		h.must(s.GetInfeasibilityExplanation(e))
		if len(e.Data) != 2 || e.Data[0].Index != 1 || e.Data[1].Index != 0 {
			t.Fatal("upper witness then lower witness")
		}
		lpSparseRationalCheck(t, h, &e.Data[0].Value, big.NewRat(1, 1))
		lpSparseRationalCheck(t, h, &e.Data[1].Value, big.NewRat(-1, 1))
		lpCheckConflictCertificate(t, h, s, e)
		// The direct helper appends; only GetInfeasibilityExplanation clears.
		h.must(s.FillExplanationFromInfeasibleColumn(e))
		if len(e.Data) != 4 {
			t.Fatal("append-only direct helper")
		}
		h.must(s.GetInfeasibilityExplanation(e))
		if len(e.Data) != 2 {
			t.Fatal("repeated extraction")
		}
		h.must(e.Close())
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARConflictCertificateRelationOrientation(t *testing.T) {
	for _, first := range []LConstraintKind{LConstraintLE, LConstraintLT, LConstraintGE, LConstraintGT, LConstraintEQ, LConstraintKind(37)} {
		for _, second := range []LConstraintKind{LConstraintLE, LConstraintLT, LConstraintGE, LConstraintGT, LConstraintEQ} {
			for _, a := range []int64{-1, 0, 1} {
				for _, b := range []int64{-1, 0, 1} {
					h := newLPPairHarness(t)
					s, err := NewLARSolver(h.m)
					h.must(err)
					for _, kind := range []LConstraintKind{first, second} {
						c, err := NewLARVarConstraint(h.m, 0, kind, &h.pair("0", "0").X)
						h.must(err)
						s.Constraints = append(s.Constraints, c)
					}
					e := NewSparseRationalVector(h.m)
					h.must(e.Append(0, &h.pair(fmt.Sprint(a), "0").X))
					h.must(e.Append(1, &h.pair(fmt.Sprint(b), "0").X))
					greater, less, strict := false, false, false
					for i, k := range []LConstraintKind{first, second} {
						if []int64{a, b}[i] <= 0 {
							k = -k
						}
						greater = greater || k == LConstraintGE || k == LConstraintGT
						less = less || k == LConstraintLE || k == LConstraintLT
						strict = strict || k == LConstraintGT || k == LConstraintLT
					}
					wantKind := LConstraintEQ
					if greater {
						wantKind = LConstraintGE
					} else if less {
						wantKind = LConstraintLE
					}
					if strict {
						wantKind /= 2
					}
					same, kind, err := s.TheRelationsAreOfSameType(e)
					h.must(err)
					if same == (greater && less) || kind != wantKind {
						t.Fatal(first, second, a, b, same, kind, wantKind)
					}
					h.must(e.Close())
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				}
			}
		}
	}
}

func TestLARConflictCertificatePhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"extract", "validate", "inf-validate", "left", "sum", "sum-constant", "diagnostic"} {
			done, failures := false, 0
			for ordinal := int64(0); ordinal < 1800 && !done; ordinal++ {
				t.Run(fmt.Sprintf("%v/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					var s *LARSolver
					var err error
					if op == "diagnostic" || op == "sum-constant" {
						s, err = NewLARSolver(h.m)
						h.must(err)
						_, err = s.AddVar(17)
						h.must(err)
						if op == "diagnostic" {
							c, x := NewLARVarConstraint(h.m, 0, LConstraintGE, &h.pair("0", "0").X)
							h.must(x)
							s.Constraints = append(s.Constraints, c)
						} else {
							v := NewSparseRationalVector(h.m)
							h.must(v.Append(0, &h.pair("3", "0").X))
							constant := h.pair(strings.Repeat("2718281828", 20)+"/19", "0")
							term, x := s.AddTerm(v.Data, &constant.X)
							h.must(x)
							h.must(v.Close())
							_, err = s.AddVarBound(0, LConstraintGE, &h.pair("1", "0").X)
							h.must(err)
							rhs := h.pair("0", "0")
							h.must(h.m.Add(&constant.X, &h.pair("2", "0").X, &rhs.X))
							_, err = s.AddVarBound(term, LConstraintLE, &rhs.X)
							h.must(err)
						}
					} else {
						s, _ = lpLARFullSolveNumericFixture(t, h, StrategyTableauRows, 4, true, true, true)
					}
					status := StatusInfeasible
					if op != "diagnostic" {
						status, err = s.FindFeasibleSolution()
					}
					h.must(err)
					if status != StatusInfeasible {
						t.Fatal(status)
					}
					e := NewSparseRationalVector(h.m)
					if op == "diagnostic" {
						h.must(e.Append(0, &h.pair(strings.Repeat("1234567890", 20)+"7/13", "0").X))
					} else if op != "extract" {
						h.must(s.GetInfeasibilityExplanation(e))
					}
					out := h.pair("123", "0")
					h.budget.FailAfter(ordinal)
					switch op {
					case "extract":
						err = s.GetInfeasibilityExplanation(e)
					case "validate":
						_, err = s.ExplanationIsCorrect(e)
					case "inf-validate":
						_, err = s.InfExplanationIsCorrect()
					case "left":
						_, err = s.TheLeftSidesSumToZero(e)
					case "sum", "sum-constant":
						err = s.SumOfRightSidesOfExplanation(e, &out.X)
					case "diagnostic":
						var zero bool
						output := lpCaptureFeasibleStdout(t, func() { zero, err = s.TheLeftSidesSumToZero(e) })
						expected := "left side = " + strings.Repeat("1234567890", 20) + "7/13v17\n"
						if !strings.HasPrefix(expected, output) || err == nil && (zero || output != expected) {
							t.Fatal("diagnostic publication prefix", zero, output, err)
						}
					}
					h.budget.FailAfter(-1)
					if err == nil {
						done = true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if x := s.GetInfeasibilityExplanation(e); x != ErrExactVectorPending {
							t.Fatal("stopped retry", x)
						}
					}
					h.must(e.Close())
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
			}
			// Native extraction and constant summation reuse pooled digits
			// warmed by the actual solve. Direct allocation must still refuse.
			warmNative := !sourceDebugEnabled && (op == "validate" || op == "inf-validate" || (op == "extract" || op == "sum-constant") && !direct)
			if !done || failures == 0 && !warmNative {
				t.Fatal("incomplete physical campaign", direct, op, done, failures)
			}
		}
	}
}

func TestLARConflictCertificateIndependentRightSums(t *testing.T) {
	for world := 0; world < 40; world++ {
		for _, variant := range []string{"variable", "term", "general"} {
			h := newLPPairHarness(t)
			s, model, _, lhs := lpLARConstraintEvalFixture(t, h, variant, world, false, LConstraintGE, 1)
			e := NewSparseRationalVector(h.m)
			weight := new(big.Rat)
			for _, a := range []*big.Rat{big.NewRat(1, 5), big.NewRat(-2, 7), big.NewRat(3, 11)} {
				h.must(e.Append(0, &h.pair(a.RatString(), "0").X))
				weight.Add(weight, a)
			}
			want := new(big.Rat).Add(lhs, big.NewRat(1, 29))
			if variant == "term" {
				want.Sub(want, big.NewRat(int64(world-17), 23))
			}
			want.Mul(want, weight)
			out := h.pair("97", "0")
			h.must(s.SumOfRightSidesOfExplanation(e, &out.X))
			lpSparseRationalCheck(t, h, &out.X, want)
			h.must(e.Close())
			h.must(s.Close())
			h.must(model.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLARConflictCertificateAllRelationAssertions(t *testing.T) {
	for _, kind := range []LConstraintKind{LConstraintLE, LConstraintLT, LConstraintGE, LConstraintGT, LConstraintEQ} {
		for _, rhs := range []int64{-1, 0, 1} {
			t.Run(fmt.Sprintf("%d/%d", kind, rhs), func(t *testing.T) {
				h := newLPPairHarness(t)
				s, err := NewLARSolver(h.m)
				h.must(err)
				for i, k := range []LConstraintKind{kind, -kind} {
					v := int64(0)
					if i == 0 {
						v = rhs
					}
					c, err := NewLARVarConstraint(h.m, 0, k, &h.pair(fmt.Sprint(v), "0").X)
					h.must(err)
					s.Constraints = append(s.Constraints, c)
				}
				e := NewSparseRationalVector(h.m)
				h.must(e.Append(0, &h.pair("1", "0").X))
				h.must(e.Append(1, &h.pair("-1", "0").X))
				valid := false
				switch kind {
				case LConstraintLE:
					valid = rhs < 0
				case LConstraintLT:
					valid = rhs <= 0
				case LConstraintGE:
					valid = rhs > 0
				case LConstraintGT:
					valid = rhs >= 0
				case LConstraintEQ:
					valid = rhs != 0
				}
				ok, err := s.ExplanationIsCorrect(e)
				want := error(nil)
				if !valid && sourceDebugEnabled {
					want = ErrAssertion
				}
				if err != want || ok != (valid || !sourceDebugEnabled) {
					t.Fatal("literal source contradiction assertion", ok, err, valid)
				}
				h.must(e.Close())
				h.must(s.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

func TestLARConflictCertificateZeroInsertionAndDiagnostics(t *testing.T) {
	for _, profile := range []string{"zero", "two-zeros", "three-zeros", "cancel", "nonzero", "empty"} {
		t.Run(profile, func(t *testing.T) {
			h := newLPPairHarness(t)
			s, err := NewLARSolver(h.m)
			h.must(err)
			_, err = s.AddVar(17)
			h.must(err)
			v := NewSparseRationalVector(h.m)
			values := []string{"0"}
			want := false
			text := "left side = 0v17\n"
			switch profile {
			case "two-zeros":
				values = []string{"0", "0"}
				want = true
				text = ""
			case "three-zeros":
				values = []string{"0", "0", "0"}
			case "cancel":
				values = []string{"3", "-3"}
				want = true
				text = ""
			case "nonzero":
				values = []string{"-1"}
				text = "left side =  - v17\n"
			case "empty":
				values = nil
				want = true
				text = ""
			}
			for _, value := range values {
				h.must(v.Append(0, &h.pair(value, "0").X))
			}
			con, err := NewLARConstraint(h.m, v.Data, LConstraintGE, &h.pair("0", "0").X)
			h.must(err)
			s.Constraints = append(s.Constraints, con)
			h.must(v.Close())
			e := NewSparseRationalVector(h.m)
			h.must(e.Append(0, &h.pair("1", "0").X))
			var zero bool
			output := lpCaptureFeasibleStdout(t, func() { zero, err = s.TheLeftSidesSumToZero(e) })
			h.must(err)
			if zero != want || output != text || s.explanationCoefficients != nil || s.validationCoefficients != nil {
				t.Fatal("source unordered-map insertion/erase", zero, want, output, text)
			}
			h.must(e.Close())
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARConflictCertificateClosedDiagnostic(t *testing.T) {
	h := newLPPairHarness(t)
	s, err := NewLARSolver(h.m)
	h.must(err)
	c, err := NewLARVarConstraint(h.m, 0, LConstraintGE, &h.pair("0", "0").X)
	h.must(err)
	s.Constraints = append(s.Constraints, c)
	e := NewSparseRationalVector(h.m)
	h.must(e.Append(0, &h.pair("1", "0").X))
	r, w, err := os.Pipe()
	h.must(err)
	h.must(w.Close())
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old; r.Close() }()
	zero, err := s.TheLeftSidesSumToZero(e)
	os.Stdout = old
	if zero || !errors.Is(err, os.ErrClosed) || s.explanationCoefficients == nil {
		t.Fatal("diagnostic failure retains dictionary", zero, err)
	}
	h.must(e.Close())
	h.must(s.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARConflictCertificatePreservesOlderComparison(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			s, _ := lpLARFullSolveNumericFixture(t, h, StrategyTableauRows, 4, true, true, true)
			_, err := s.FindFeasibleSolution()
			h.must(err)
			e := NewSparseRationalVector(h.m)
			h.must(s.GetInfeasibilityExplanation(e))
			out := h.pair("97", "0")

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
				err = s.SumOfRightSidesOfExplanation(e, &out.X)
				if s.calls.comparison != 0 {
					t.Fatal("adopted older comparison", err)
				}
			}
			h.must(e.Close())
			h.must(s.Close())
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("consumed older comparison")
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

func TestLARConflictCertificateEntryGuards(t *testing.T) {
	for _, op := range []string{"relations", "sum", "left", "validate", "extract", "sign", "column", "inf-validate"} {
		for _, phase := range []string{"stopped", "nil", "foreign", "pending"} {
			if op == "inf-validate" && phase != "stopped" {
				continue
			}
			t.Run(op+"/"+phase, func(t *testing.T) {
				h := newLPPairHarness(t)
				if phase == "pending" {
					h = lpNewDirectSparseHarness(t)
				}
				s, err := NewLARSolver(h.m)
				h.must(err)
				e := NewSparseRationalVector(h.m)
				argument := e
				other := newLPPairHarness(t)
				foreign := NewSparseRationalVector(other.m)
				out := h.pair("97", "0")
				want := error(ErrExactVectorOwner)
				switch phase {
				case "stopped":
					s.stopped = true
					want = ErrExactVectorPending
				case "nil":
					argument = nil
				case "foreign":
					argument = foreign
				case "pending":
					v := h.pair(strings.Repeat("1234567890", 20)+"/7", "0")
					h.budget.FailAfter(0)
					err = e.Append(0, &v.X)
					h.budget.FailAfter(-1)
					if !lpExactPriorityPhysicalRefusal(err) || e.ready() == nil {
						t.Fatal("no real unfinished input", err)
					}
					want = ErrExactVectorPending
				}
				if op == "validate" && !sourceDebugEnabled && phase != "stopped" {
					want = nil
				}
				switch op {
				case "relations":
					_, _, err = s.TheRelationsAreOfSameType(argument)
				case "sum":
					err = s.SumOfRightSidesOfExplanation(argument, &out.X)
				case "left":
					_, err = s.TheLeftSidesSumToZero(argument)
				case "validate":
					_, err = s.ExplanationIsCorrect(argument)
				case "extract":
					err = s.GetInfeasibilityExplanation(argument)
				case "sign":
					err = s.GetInfeasibilityExplanationForInfSign(argument, s.Core.Infeasibility, 1)
				case "column":
					err = s.FillExplanationFromInfeasibleColumn(argument)
				case "inf-validate":
					_, err = s.InfExplanationIsCorrect()
				}
				if err != want {
					t.Fatal("entry ownership prefix", err, want)
				}
				h.must(e.Close())
				other.must(foreign.Close())
				h.must(s.Close())
				lpExactVectorNamedCensus(t, h, nil)
				lpExactVectorNamedCensus(t, other, nil)
			})
		}
	}
}

func TestLARConflictCertificateWitnessSelectionAndEmpty(t *testing.T) {
	for _, sign := range []int32{-1, 0, 1} {
		h := newLPPairHarness(t)
		s, err := NewLARSolver(h.m)
		h.must(err)
		_, err = s.AddVar(17)
		h.must(err)
		_, err = s.AddVarBound(0, LConstraintGE, &h.pair("-1", "0").X)
		h.must(err)
		_, err = s.AddVarBound(0, LConstraintLE, &h.pair("1", "0").X)
		h.must(err)
		row := NewSparseRationalVector(h.m)
		e := NewSparseRationalVector(h.m)
		for _, value := range []string{"2", "-3", "0"} {
			h.must(row.Append(0, &h.pair(value, "0").X))
		}
		h.must(e.Append(99, &h.pair("7", "0").X))
		h.must(s.GetInfeasibilityExplanationForInfSign(e, row, sign))
		if len(e.Data) != 4 || e.Data[0].Index != 99 {
			t.Fatal("sign helper must append")
		}
		for i, a := range []int32{2, -3, 0} {
			adjusted := sign
			if a <= 0 {
				adjusted = -sign
			}
			want := uint32(0)
			if adjusted < 0 {
				want = 1
			}
			if e.Data[1+i].Index != want {
				t.Fatal("zero coefficient takes nonpositive sign branch", sign, a, e.Data[1+i].Index, want)
			}
		}
		// The source sign-zero return follows clearing and does not inspect
		// even a malformed retained core row.
		s.Core.InfeasibleSumSign = 0
		h.must(s.Core.Infeasibility.Append(999, &h.pair("1", "0").X))
		h.must(s.GetInfeasibilityExplanation(e))
		if len(e.Data) != 0 {
			t.Fatal("empty sign did not clear")
		}
		h.must(row.Close())
		h.must(e.Close())
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARConflictCertificateAssertionPrefixes(t *testing.T) {
	for _, phase := range []string{"mixed", "nonzero-left", "bad-left-index", "bad-sum-index", "bad-kind-index", "bad-witness", "bad-column", "nil-output"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			s, err := NewLARSolver(h.m)
			h.must(err)
			_, err = s.AddVar(17)
			h.must(err)
			for _, kind := range []LConstraintKind{LConstraintGE, LConstraintLE} {
				c, err := NewLARVarConstraint(h.m, 0, kind, &h.pair("0", "0").X)
				h.must(err)
				s.Constraints = append(s.Constraints, c)
			}
			e := NewSparseRationalVector(h.m)
			h.must(e.Append(0, &h.pair("1", "0").X))
			out := h.pair("97", "0")
			want := error(nil)
			if sourceDebugEnabled {
				want = ErrAssertion
			}
			var output string
			switch phase {
			case "mixed":
				h.must(e.Append(1, &h.pair("1", "0").X))
				_, err = s.ExplanationIsCorrect(e)
			case "nonzero-left":
				output = lpCaptureFeasibleStdout(t, func() { _, err = s.ExplanationIsCorrect(e) })
			case "bad-left-index":
				e.Data[0].Index = 999
				_, err = s.TheLeftSidesSumToZero(e)
				if !sourceDebugEnabled {
					want = ErrBounds
				}
			case "bad-sum-index":
				e.Data[0].Index = 999
				err = s.SumOfRightSidesOfExplanation(e, &out.X)
				if !sourceDebugEnabled {
					want = ErrBounds
				}
			case "bad-kind-index":
				e.Data[0].Index = 999
				_, _, err = s.TheRelationsAreOfSameType(e)
				want = ErrBounds
			case "bad-witness":
				h.must(s.Core.Infeasibility.Append(0, &h.pair("1", "0").X))
				err = s.GetInfeasibilityExplanationForInfSign(e, s.Core.Infeasibility, 1)
				if !sourceDebugEnabled && (len(e.Data) != 2 || e.Data[1].Index != InvalidLARIndex) {
					t.Fatal("native source preserves invalid witness")
				}
			case "bad-column":
				s.Bounds.InfeasibleColumn.Value = 999
				err = s.FillExplanationFromInfeasibleColumn(e)
				want = ErrBounds
			case "nil-output":
				err = s.SumOfRightSidesOfExplanation(e, nil)
				want = ErrBounds
			}
			if err != want {
				t.Fatal("source assertion prefix", err, want)
			}
			if phase == "nonzero-left" && sourceDebugEnabled && output != "left side = v17\n" {
				t.Fatal("actual source diagnostic", output)
			}
			h.must(e.Close())
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}
