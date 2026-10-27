package lp

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

func TestLARScopesIndependentNestedTableauModels(t *testing.T) {
	h := newLPPairHarness(t)
	rng := rand.New(rand.NewPCG(81783, 431))
	for world := 0; world < 160; world++ {
		a := big.NewRat(int64(rng.IntN(11)-5), int64(rng.IntN(7)+1))
		b := big.NewRat(int64(rng.IntN(13)-6), int64(rng.IntN(11)+1))
		s := lpNewRemovalModel(t, h, a.RatString(), b.RatString(), false)
		c := s.Core
		lo := h.pair("1/3", "0")
		_, err := s.AddVarBound(0, LConstraintGE, &lo.X)
		h.must(err)
		initialConstraint := s.Constraints[0]
		initialTerm, initialOriginal := s.Terms[0], s.OriginalTerms[0]
		type snap struct {
			cols, rows, terms, constraints int
			low                            *big.Rat
			status                         Status
			infeasible                     int32
			witness                        LARBoundWitnesses
		}
		var stack []snap
		var removedTerms, removedOriginals []*LARTerm
		depth := 1 + world%4
		for level := 0; level < depth; level++ {
			low, e := s.Bounds.Low.At(0)
			h.must(e)
			text, e := h.m.String(&low.X)
			h.must(e)
			witness, e := s.Bounds.Witnesses.At(0)
			h.must(e)
			stack = append(stack, snap{len(s.ColumnsToExternal), int(c.RA.RowCount()), len(s.Terms), len(s.Constraints), mustLPRat(t, text), s.Bounds.Status.Value, s.Bounds.InfeasibleColumn.Value, witness})
			h.must(s.Push())
			_, err = s.AddVar(uint32(1000 + level))
			h.must(err)
			p := h.pair("2/5", "0")
			termIndex, e := s.AddTerm([]RationalSparseEntry{{Index: 0, Value: p.X}}, &p.Y)
			h.must(e)
			_, err = s.AddVarBound(termIndex, LConstraintGE, &p.Y)
			h.must(err)
			removedTerms = append(removedTerms, s.Terms[len(s.Terms)-1])
			removedOriginals = append(removedOriginals, s.OriginalTerms[len(s.OriginalTerms)-1])
			h.must(h.m.SetInt32(&lo.X, int32(level+2)))
			_, err = s.AddVarBound(0, LConstraintGE, &lo.X)
			h.must(err)
			s.Bounds.Status.Value = StatusInfeasible
			s.Bounds.InfeasibleColumn.Value = 0
			h.must(s.Bounds.ChangedColumns.Insert(c.RA.ColumnCount() - 1))
			h.must(s.RowsWithChangedBounds.Resize(c.RA.RowCount()))
			h.must(s.RowsWithChangedBounds.Insert(c.RA.RowCount() - 1))
		}
		current := big.NewRat(0, 1)
		for len(stack) > 0 {
			k := 1
			if world%3 == 0 && len(stack) > 1 {
				k = 2
			}
			target := stack[len(stack)-k]
			h.must(c.RPrimal.InfSet.Insert(0))
			h.must(s.Pop(uint32(k)))
			if current.Cmp(target.low) < 0 {
				current.Set(target.low)
			}
			stack = stack[:len(stack)-k]
			h.check(&c.RX.Data[0], current, big.NewRat(0, 1))
			h.check(&c.RX.Data[2], new(big.Rat).Mul(a, current), big.NewRat(0, 1))
			if len(s.ColumnsToExternal) != target.cols || len(s.ExternalToColumns) != target.cols || int(c.RA.RowCount()) != target.rows || len(s.Terms) != target.terms || len(s.OriginalTerms) != target.terms || len(s.Constraints) != target.constraints {
				t.Fatal("scope extents", world, k, target)
			}
			if s.Bounds.Status.Value != target.status || s.Bounds.InfeasibleColumn.Value != target.infeasible {
				t.Fatal("status restoration")
			}
			witness, e := s.Bounds.Witnesses.At(0)
			h.must(e)
			if witness != target.witness {
				t.Fatal("witness restoration", witness, target.witness)
			}
			for _, j := range s.Bounds.ChangedColumns.Indices() {
				if int(j) >= target.cols {
					t.Fatal("changed column retained")
				}
			}
			for _, j := range s.RowsWithChangedBounds.Indices() {
				if int(j) >= target.rows {
					t.Fatal("changed row retained")
				}
			}
			if s.SimplexStrategy.StackSize() != uint32(len(stack)) || s.TermCount.StackSize() != uint32(len(stack)) || s.ConstraintCount.StackSize() != uint32(len(stack)) {
				t.Fatal("scope depths")
			}
			if s.Terms[0] != initialTerm || s.OriginalTerms[0] != initialOriginal || s.Constraints[0] != initialConstraint {
				t.Fatal("surviving owner replaced")
			}
			ok, e := s.AxIsCorrect()
			h.must(e)
			if !ok {
				t.Fatal("equation oracle")
			}
			if !c.RPrimal.InfSet.Empty() {
				t.Fatal("infeasibility repair")
			}
		}
		for _, term := range append(removedTerms, removedOriginals...) {
			if term.Size() != 0 {
				t.Fatal("scoped term not destroyed")
			}
		}
		if !reflect.DeepEqual(s.ColumnsToExternal, []uint32{31, 67, LARTermStartIndex}) {
			t.Fatal(s.ColumnsToExternal)
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARScopesActualNonunitLURestoration(t *testing.T) {
	h := newLPPairHarness(t)
	for world := 0; world < 100; world++ {
		a1 := big.NewRat(int64(world%9+1), 7)
		basis := big.NewRat(int64(world%11+1), 13)
		if world%2 == 0 {
			basis.Neg(basis)
		}
		s := lpPreparedLARPivot(t, h, "-2/3", a1.RatString(), basis.RatString())
		c := s.Core
		if c.RPrimal.Factorization == nil {
			t.Fatal("missing actual LU")
		}
		h.must(h.m.SetInt32(&c.RX.Data[1].X, 2))
		h.must(h.m.SetString(&c.RX.Data[2].X, new(big.Rat).Mul(big.NewRat(-2, 1), new(big.Rat).Quo(a1, basis)).RatString()))
		oldConstraints := len(s.Constraints)
		oldHeading := append([]int32(nil), c.RHeading...)
		h.must(s.Push())
		_, err := s.AddVar(uint32(100 + world))
		h.must(err)
		rhs := h.pair("1/3", "0")
		_, err = s.AddVarBound(0, LConstraintGE, &rhs.X)
		h.must(err)
		h.must(c.RPrimal.InfSet.Insert(0))
		s.Bounds.Status.Value = StatusInfeasible
		h.must(s.Pop(1))
		if len(s.Constraints) != oldConstraints || len(s.ColumnsToExternal) != 3 || c.RA.ColumnCount() != 3 || c.RA.RowCount() != 1 || c.DA.ColumnCount() != 3 || c.DA.RowCount() != 0 {
			t.Fatal("LU scope extents")
		}
		if c.RPrimal.Factorization != nil || c.DPrimal.Factorization != nil {
			t.Fatal("source factorization destruction")
		}
		if !reflect.DeepEqual(c.RHeading, oldHeading) {
			t.Fatal(c.RHeading, oldHeading)
		}
		// Restored low=-3 leaves x0=0 unchanged; source values are not snapshots.
		h.check(&c.RX.Data[0], big.NewRat(0, 1), big.NewRat(0, 1))
		if !c.RPrimal.InfSet.Empty() {
			t.Fatal("feasible nonbasic retained")
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARScopesSourceFailurePrefixes(t *testing.T) {
	for _, which := range []string{"zero", "too-deep", "directory", "directory-growth", "tableau", "witness", "core", "cleanup", "ax", "inf", "constraint-count", "term-count", "term-pair", "nil-constraint", "nil-term", "strategy", "costs"} {
		t.Run(which, func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpNewRemovalModel(t, h, "2", "3", false)
			c := s.Core
			s.Bounds.Status.Value = StatusFeasible
			s.Bounds.InfeasibleColumn.Value = -1
			h.must(s.Push())
			s.Bounds.Status.Value = StatusInfeasible
			s.Bounds.InfeasibleColumn.Value = 0
			v := h.pair("1", "0")
			_, err := s.AddVarBound(0, LConstraintGE, &v.X)
			h.must(err)
			_, err = s.AddTerm([]RationalSparseEntry{{Index: 0, Value: v.X}}, &v.Y)
			h.must(err)
			originalTerms := append([]*LARTerm(nil), s.Terms...)
			originalOriginals := append([]*LARTerm(nil), s.OriginalTerms...)
			originalConstraints := append([]LARStoredConstraint(nil), s.Constraints...)
			k := uint32(1)
			var want error = ErrBounds
			switch which {
			case "zero":
				k = 0
				want = ErrScope
			case "too-deep":
				k = 2
				want = ErrScope
			case "directory":
				s.ColumnsToExternal = nil
			case "directory-growth":
				clear(s.ExternalToColumns)
				s.ColumnsToExternal = nil
				want = nil
			case "tableau":
				c.RHeading = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "witness":
				s.Bounds.Witnesses.vectorSizes = nil
				want = ErrScope
			case "core":
				c.Bounds.Low.vectorSizes = nil
				want = ErrScope
			case "cleanup":
				h.must(c.RPrimal.InfSet.Insert(0))
				c.RHeading[0] = -99
				// Native ignores nonbasic inverse consistency; debug rejects during tableau removal.
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "ax":
				h.must(h.m.SetInt32(&c.RX.Data[2].X, 7))
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "inf":
				h.must(s.Bounds.Low.Set(0, h.pair("1", "0"))) // Scoped bound disappears; no stale membership required.
				h.must(c.RPrimal.InfSet.Insert(2))
				want = nil
			case "constraint-count":
				s.ConstraintCount.stack[0] = 99
			case "term-count":
				s.TermCount.stack[0] = 99
			case "term-pair":
				s.OriginalTerms = s.OriginalTerms[:1]
			case "nil-constraint":
				s.Constraints[0] = nil
			case "nil-term":
				s.Terms[1] = nil
			case "strategy":
				s.SimplexStrategy.stack[0] = StrategyTableauCosts
				want = nil
			case "costs":
				h.must(h.m.SetInt32(&c.Costs.Data[3], 1))
				want = nil
			}
			stdout := lpCaptureFeasibleStdout(t, func() { err = s.Pop(k) })
			if !errors.Is(err, want) {
				t.Fatal(err, want, stdout)
			}
			if which == "zero" {
				if s.Bounds.Status.Value != StatusInfeasible || s.Bounds.InfeasibleColumn.Value != 0 {
					t.Fatal("zero-depth scalar prefix")
				}
			}
			if which == "too-deep" || which == "witness" {
				if s.Bounds.Status.Value != StatusFeasible || s.Bounds.InfeasibleColumn.Value != -1 {
					t.Fatal("scalar prefix before witness refusal")
				}
			}
			if which == "term-pair" {
				if s.Terms[1] != nil || originalTerms[1].Size() != 0 {
					t.Fatal("current term was not destroyed before original guard")
				}
			}
			if err != nil {
				if s.Push() != ErrExactVectorPending || s.Pop(1) != ErrExactVectorPending {
					t.Fatal("stopped scope entry")
				}
			}
			// Restore only intentionally removed owning headers/tokens for destruction.
			if which == "term-pair" {
				s.OriginalTerms = originalOriginals
			}
			if which == "nil-constraint" {
				s.Constraints = originalConstraints
			}
			if which == "nil-term" {
				s.Terms = originalTerms
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARScopesPhysicalPopOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"tableau", "bound", "cleanup", "ax", "LU", "costs"} {
			failures, owned := 0, 0
			success := false
			for ordinal := int64(0); ordinal < 340; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("direct=%t/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					a := strings.Repeat("9876543210", 100) + "/7"
					b := strings.Repeat("1234567890", 100) + "/13"
					var s *LARSolver
					if op == "LU" {
						s = lpPreparedLARPivot(t, h, "-2/3", "5/7", "3/11")
						h.must(h.m.SetInt32(&s.Core.RX.Data[1].X, 2))
						h.must(h.m.SetString(&s.Core.RX.Data[2].X, "-110/21"))
					} else {
						s = lpNewRemovalModel(t, h, "2/3", "-5/7", false)
					}
					c := s.Core
					r := c.RPrimal
					low := h.pair(a, "1/5")
					if op == "bound" || op == "cleanup" || op == "LU" {
						h.must(s.Bounds.Low.Set(0, low))
						h.must(s.Bounds.Types.Set(0, ColumnLowBound))
					}
					if op == "ax" || op == "cleanup" {
						h.must(h.m.SetString(&c.RX.Data[0].X, b))
						h.must(h.m.SetString(&c.RX.Data[2].X, new(big.Rat).Mul(big.NewRat(2, 3), mustLPRat(t, b)).RatString()))
					}
					if op == "ax" {
						h.must(h.m.SetString(&c.RX.Data[0].Y, b))
						h.must(h.m.SetString(&c.RX.Data[2].Y, new(big.Rat).Mul(big.NewRat(2, 3), mustLPRat(t, b)).RatString()))
					}
					if op == "costs" {
						s.Settings.SimplexStrategy = StrategyTableauCosts
						h.must(h.m.SetString(&c.Costs.Data[2], b))
						h.must(h.m.SetString(&r.D.Data[0], new(big.Rat).Mul(big.NewRat(2, 3), mustLPRat(t, b)).RatString()))
						h.must(h.m.SetString(&r.D.Data[1], new(big.Rat).Mul(big.NewRat(-5, 7), mustLPRat(t, b)).RatString()))
					}
					h.must(s.Push())
					if op == "tableau" {
						_, e := s.AddTerm([]RationalSparseEntry{{Index: 0, Value: low.X}}, &low.Y)
						h.must(e)
						h.must(h.m.Copy(&c.Costs.Data[3], &low.X))
					}
					if op == "bound" {
						h.must(s.Bounds.Low.Set(0, h.pair("1", "0")))
					}
					if op == "bound" || op == "cleanup" || op == "LU" {
						h.must(r.InfSet.Insert(0))
					}
					h.budget.FailAfter(ordinal)
					err := s.Pop(1)
					h.budget.FailAfter(-1)
					if err == nil {
						success = true
						done = true
						if op == "bound" || op == "cleanup" {
							h.check(&c.RX.Data[0], mustLPRat(t, a), big.NewRat(1, 5))
							h.check(&c.RX.Data[2], new(big.Rat).Mul(big.NewRat(2, 3), mustLPRat(t, a)), big.NewRat(2, 15))
						}
						if op == "LU" {
							h.check(&c.RX.Data[2], new(big.Rat).Add(big.NewRat(-110, 21), new(big.Rat).Mul(big.NewRat(22, 9), mustLPRat(t, a))), big.NewRat(22, 45))
							if r.Factorization == nil {
								t.Fatal("cleanup did not rebuild actual LU")
							}
						}
						if s.SimplexStrategy.StackSize() != 0 || s.TermCount.StackSize() != 0 || s.ConstraintCount.StackSize() != 0 {
							t.Fatal("scope tail not completed")
						}
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						child := c.Bounds.Low.unfinished() || c.Bounds.Upper.unfinished() || r.PairCoreSolverBase.calls.pending() || r.PairCoreSolverBase.pivotWork != nil || r.calls.pending()
						if r.Factorization != nil && r.Factorization.stopped {
							child = true
						}
						if child {
							if s.calls.pending() {
								t.Fatal("outer adopted child invocation")
							}
							owned++
						} else if s.calls.pending() || s.rationalWork != nil || s.work != nil {
							owned++
						}
						if s.Push() != ErrExactVectorPending || s.Pop(1) != ErrExactVectorPending {
							t.Fatal("stopped scope")
						}
					}
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if !success || (((op != "ax" && op != "costs") || sourceDebugEnabled) && (failures == 0 || owned == 0)) {
				t.Fatal("incomplete physical campaign", direct, op, success, failures, owned)
			}
		}
	}
}

func TestLARScopesPushActualPendingBoundPrefix(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		matched := false
		for ordinal := int64(0); ordinal < 80 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("direct=%t/%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				s := lpNewRemovalModel(t, h, "2/3", "-5/7", false)
				large := "1" + strings.Repeat("0", 999) + "7"
				v := h.pair(large+"/7", large+"/11")
				h.must(s.Bounds.Low.Set(0, v))
				zero := h.pair("0", "0")
				h.budget.FailAfter(ordinal)
				err := s.Bounds.Low.Set(0, zero)
				h.budget.FailAfter(-1)
				if s.Bounds.Low.unfinished() {
					matched = true
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					if e := s.Push(); e != ErrPairVectorPending {
						t.Fatal(e)
					}
					if s.SimplexStrategy.StackSize() != 1 || s.Bounds.Status.StackSize() != 1 || s.Bounds.InfeasibleColumn.StackSize() != 1 || len(s.Bounds.Witnesses.vectorSizes) != 1 {
						t.Fatal("outer prefix missing")
					}
					if s.TermCount.StackSize() != 0 || s.ConstraintCount.StackSize() != 0 || s.calls.pending() {
						t.Fatal("outer count/capture publication passed child failure")
					}
					if s.Push() != ErrExactVectorPending {
						t.Fatal("stopped push")
					}
				} else if err != nil && !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				h.must(s.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !matched {
			t.Fatal("no actual pending bound", direct)
		}
	}
}

func TestLARScopesActualUndecidedDeferredTerms(t *testing.T) {
	h := newLPPairHarness(t)
	for world := 0; world < 100; world++ {
		s, err := NewLARSolver(h.m)
		h.must(err)
		s.Settings.SimplexStrategy = StrategyUndecided
		for _, ext := range []uint32{31, 67} {
			_, err = s.AddVar(ext)
			h.must(err)
		}
		v := h.pair("2/3", "5/7")
		_, err = s.AddTerm([]RationalSparseEntry{{Index: 0, Value: v.X}}, &v.Y)
		h.must(err)
		old, original := s.Terms[0], s.OriginalTerms[0]
		h.must(s.Push())
		_, err = s.AddVar(uint32(100 + world))
		h.must(err)
		_, err = s.AddTerm([]RationalSparseEntry{{Index: 1, Value: v.Y}}, &v.X)
		h.must(err)
		removed, removedOriginal := s.Terms[1], s.OriginalTerms[1]
		_, err = s.AddVarBound(0, LConstraintGE, &v.X)
		h.must(err)
		h.must(s.Pop(1))
		if s.Settings.SimplexStrategy != StrategyUndecided || s.Core.RA.ColumnCount() != 2 || s.Core.RA.RowCount() != 0 || s.Core.DA.ColumnCount() != 0 || s.Bounds.Witnesses.Size() != 2 || len(s.Terms) != 1 || len(s.OriginalTerms) != 1 || len(s.Constraints) != 0 {
			t.Fatal("undecided source extents")
		}
		if s.Terms[0] != old || s.OriginalTerms[0] != original || removed.Size() != 0 || removedOriginal.Size() != 0 {
			t.Fatal("deferred owners")
		}
		if !reflect.DeepEqual(s.ColumnsToExternal, []uint32{31, 67}) || !reflect.DeepEqual(s.Core.RNBasis, []uint32{1, 0}) {
			t.Fatal("source nonbasic restoration", s.ColumnsToExternal, s.Core.RNBasis)
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARScopesPivotedTableauKeepsSurvivingBasis(t *testing.T) {
	for _, row := range []uint32{0, 1} {
		t.Run(fmt.Sprint(row), func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpNewRemovalModel(t, h, "2", "3", false)
			c := s.Core
			r := c.RPrimal
			for j, p := range []*RationalPair{h.pair("2", "1/3"), h.pair("2", "-1/5"), h.pair("10", "1/15")} {
				h.must(c.RX.Data[j].Copy(h.m, p))
			}
			h.must(s.Push())
			v := h.pair("5/7", "0")
			_, err := s.AddTerm([]RationalSparseEntry{{Index: 0, Value: v.X}}, &v.Y)
			h.must(err)
			ok, err := r.PivotColumnTableau(0, row)
			h.must(err)
			if !ok {
				t.Fatal("actual tableau pivot refused")
			}
			leaving := c.RBasis[row]
			h.must(r.ChangeBasis(0, leaving))
			h.must(s.BasicColumnsWithChangedCost.Resize(4))
			h.must(s.BasicColumnsWithChangedCost.Insert(3))
			h.must(s.Pop(1))
			wantBasis := uint32(0)
			if row == 1 {
				wantBasis = 2
			}
			if !reflect.DeepEqual(c.RBasis, []uint32{wantBasis}) || !reflect.DeepEqual(c.DBasis, c.RBasis) {
				t.Fatal("tableau basis was snapshotted", c.RBasis, c.DBasis, wantBasis)
			}
			h.check(&c.RX.Data[0], big.NewRat(2, 1), big.NewRat(1, 3))
			h.check(&c.RX.Data[1], big.NewRat(2, 1), big.NewRat(-1, 5))
			h.check(&c.RX.Data[2], big.NewRat(10, 1), big.NewRat(1, 15))
			if !s.BasicColumnsWithChangedCost.Contains(3) {
				t.Fatal("source does not clean this member set during Pop")
			}
			if len(s.Terms) != 1 || len(s.OriginalTerms) != 1 || len(s.ColumnsToExternal) != 3 || s.Bounds.Witnesses.Size() != 3 {
				t.Fatal("pivoted scope extents")
			}
			ok, err = s.AxIsCorrect()
			h.must(err)
			if !ok {
				t.Fatal("surviving equation")
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}
