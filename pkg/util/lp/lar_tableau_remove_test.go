package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

func lpNewRemovalModel(t *testing.T, h *lpPairHarness, c0, c1 string, two bool) *LARSolver {
	t.Helper()
	s, err := NewLARSolver(h.m)
	h.must(err)
	for _, ext := range []uint32{31, 67} {
		_, err = s.AddVar(ext)
		h.must(err)
	}
	p := h.pair(c0, c1)
	constant := h.pair("5", "0")
	count := 1
	if two {
		count = 2
	}
	for k := 0; k < count; k++ {
		_, err = s.AddTerm([]RationalSparseEntry{{Index: 0, Value: p.X}, {Index: 1, Value: p.Y}}, &constant.X)
		h.must(err)
	}
	return s
}

func TestLARTableauRemovalIndependentSurvivingModel(t *testing.T) {
	h := newLPPairHarness(t)
	rng := rand.New(rand.NewPCG(87421, 981))
	for world := 0; world < 160; world++ {
		a, b := int64(rng.IntN(11)+1), int64(rng.IntN(17)-8)
		if world%2 == 0 {
			a = -a
		}
		c0, c1 := big.NewRat(a, int64(rng.IntN(7)+1)), big.NewRat(b, int64(rng.IntN(11)+1))
		s := lpNewRemovalModel(t, h, c0.RatString(), c1.RatString(), true)
		c := s.Core
		for i := range c.RightSides.Data {
			h.must(h.m.SetInt64(&c.RightSides.Data[i].X, int64(i+10)))
			h.must(h.m.SetInt64(&c.RightSides.Data[i].Y, int64(i-2)))
		}
		rhs := make([]RationalPair, len(c.RightSides.Data))
		for i := range rhs {
			rhs[i] = NewRationalPair()
			h.must(rhs[i].Copy(h.m, &c.RightSides.Data[i]))
		}
		// Exercise a row-owning column that has left the basis and now appears
		// in several rows. Exact re-pivot/removal must restore the older term row.
		if world%3 == 0 {
			ok, e := c.RPrimal.PivotColumnTableau(0, 1)
			h.must(e)
			if !ok {
				t.Fatal("preparing nonbasic removal")
			}
			h.must(c.RPrimal.ChangeBasis(0, 3))
		}
		if world%3 == 1 {
			h.must(c.RPrimal.TransposeRowsTableau(0, 1))
		}
		d := []int64{17, -7, 19, 11}
		for j, x := range d {
			h.must(h.m.SetInt64(&c.RPrimal.D.Data[j], x))
		}
		cost := int64(world%7 - 3)
		h.must(h.m.SetInt64(&c.Costs.Data[3], cost))
		oldWitness := append([]LARBoundWitnesses(nil), s.Bounds.Witnesses.Values()...)
		h.must(s.RemoveColumnFromTableau(3))
		if c.RA.RowCount() != 1 || c.RA.ColumnCount() != 3 || len(c.RX.Data) != 3 || len(c.RPrimal.D.Data) != 3 || len(c.Costs.Data) != 3 {
			t.Fatal("numeric extents")
		}
		if !reflect.DeepEqual(c.RBasis, []uint32{2}) || len(c.RNBasis) != 2 {
			t.Fatal("surviving basis", c.RBasis, c.RNBasis)
		}
		want := map[IndexPair]*big.Rat{{0, 0}: new(big.Rat).Neg(c0), {0, 2}: big.NewRat(1, 1)}
		if c1.Sign() != 0 {
			want[IndexPair{0, 1}] = new(big.Rat).Neg(c1)
		}
		lpExactStaticCheck(t, h, c.RA, want)
		for j, x := range []*big.Rat{c0, c1} {
			expected := new(big.Rat).Sub(big.NewRat(d[j], 1), new(big.Rat).Mul(big.NewRat(cost, 1), x))
			lpSparseRationalCheck(t, h, &c.RPrimal.D.Data[j], expected)
		}
		lpSparseRationalCheck(t, h, &c.RPrimal.D.Data[2], big.NewRat(19, 1))
		if len(c.RightSides.Data) != 3 {
			t.Fatal("source pops final RHS element, not row index")
		}
		// Matrix row swaps intentionally leave RHS entries in their original order.
		for i := range c.RightSides.Data {
			h.check(&c.RightSides.Data[i], big.NewRat(int64(i+10), 1), big.NewRat(int64(i-2), 1))
		}
		if !reflect.DeepEqual(oldWitness, s.Bounds.Witnesses.Values()) || len(s.Terms) != 2 || len(s.Constraints) != 0 || len(s.ColumnsToExternal) != 4 {
			t.Fatal("numeric removal changed outer scopes")
		}
		for i := range rhs {
			h.must(rhs[i].Del(h.m))
		}
		h.must(s.Close())
	}
}

func TestLARTableauPopEmptyColumnsAndReverseOrder(t *testing.T) {
	h := newLPPairHarness(t)
	s := lpNewRemovalModel(t, h, "2", "-3", true)
	for _, ext := range []uint32{101, 103} {
		_, err := s.AddVar(ext)
		h.must(err)
	}
	c := s.Core
	// Put the last empty column first in the nonbasis list, forcing swap-pop.
	c.RNBasis[0], c.RNBasis[3] = c.RNBasis[3], c.RNBasis[0]
	for i, j := range c.RNBasis {
		c.RHeading[j] = -int32(i) - 1
	}
	s.ColumnsToExternal = s.ColumnsToExternal[:2]
	h.must(s.PopTableau())
	if c.RA.RowCount() != 0 || c.RA.ColumnCount() != 2 || len(c.RX.Data) != 2 || len(c.RPrimal.D.Data) != 2 || len(c.Costs.Data) != 2 || len(c.RBasis) != 0 || len(c.RHeading) != 2 {
		t.Fatal("reverse removal extents")
	}
	for _, row := range []uint32{0, 1} {
		rep, e := s.ColumnRepresentsRowInTableau(row)
		h.must(e)
		if rep {
			t.Fatal("ordinary column marked row")
		}
	}
	if len(s.Terms) != 2 || s.Bounds.Witnesses.Size() != 6 {
		t.Fatal("outer state prematurely popped")
	}
	h.must(s.PopTableau())
	h.must(s.Close())
}

func TestLARTableauBasisSwapPopAndBottomRightFirstLink(t *testing.T) {
	h := newLPPairHarness(t)
	// The source reverse scan stores every index and therefore chooses the
	// first column link when the last row has no entry in that column.
	s := lpNewRemovalModel(t, h, "2", "3", true)
	_, err := s.AddTerm([]RationalSparseEntry{{Index: 1, Value: h.pair("7", "0").X}}, &h.pair("0", "0").X)
	h.must(err)
	c := s.Core
	// Last column originally has only row2. Move its link to row0 and append
	// another row1 link without touching the last row.
	h.must(c.RPrimal.TransposeRowsTableau(0, 2))
	h.must(c.RA.Set(1, 4, &h.pair("9", "0").X))
	h.must(s.MakeSureBottomRightElementIsNonzeroInTableau(2, 4))
	if c.RBasis[2] != 4 || c.RA.Columns[4][0].Row != 2 {
		t.Fatal("did not select first link", c.RBasis, c.RA.Columns[4])
	}
	h.must(s.Close())
	// Direct source helper removes a basic column from a nonfinal basis slot.
	s = lpNewRemovalModel(t, h, "2", "3", true)
	c = s.Core
	c.RBasis = []uint32{3, 2}
	c.RHeading = []int32{-1, -2, 1, 0}
	savedRow, savedColumn := c.RA.Rows[1], c.RA.Columns[3]
	c.RA.Rows = c.RA.Rows[:1]
	c.RA.Columns = c.RA.Columns[:3]
	h.must(s.RemoveLastColumnFromBasisTableau(3))
	if !reflect.DeepEqual(c.RBasis, []uint32{2}) || !reflect.DeepEqual(c.RHeading, []int32{-1, -2, 0}) {
		t.Fatal("basic swap-pop")
	}
	// Restore borrowed geometry so the owning rows remain reachable for Close.
	c.RA.Rows = append(c.RA.Rows, savedRow)
	c.RA.Columns = append(c.RA.Columns, savedColumn)
	h.must(s.Close())
}

func TestLARTableauRemovalPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"cost", "pivot", "transpose", "pop"} {
			failures, own, child := 0, 0, 0
			success := false
			for ordinal := int64(0); ordinal < 260; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("direct=%t/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					x := strings.Repeat("1234567890", 100) + "/7"
					y := strings.Repeat("9876543210", 100) + "/13"
					s := lpNewRemovalModel(t, h, x, y, op == "transpose")
					c := s.Core
					j := c.RA.ColumnCount() - 1
					if op == "pivot" {
						ok, e := c.RPrimal.PivotColumnTableau(0, 0)
						h.must(e)
						if !ok {
							t.Fatal("setup pivot")
						}
						h.must(c.RPrimal.ChangeBasis(0, j))
						h.must(h.m.SetString(&c.RightSides.Data[0].X, x))
					}
					if op == "transpose" {
						h.must(c.RPrimal.TransposeRowsTableau(0, 1))
					}
					h.must(h.m.SetString(&c.Costs.Data[j], x))
					h.budget.FailAfter(ordinal)
					var err error
					if op == "pop" {
						s.ColumnsToExternal = s.ColumnsToExternal[:2]
						err = s.PopTableau()
					} else {
						err = s.RemoveColumnFromTableau(j)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success = true
						done = true
						if c.RA.ColumnCount() != j || len(c.RPrimal.D.Data) != int(j) {
							t.Fatal("successful tail extents")
						}
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if s.calls.pending() {
							own++
						}
						if c.RA.work != nil || c.RA.transposeRow != nil || c.RPrimal.PairCoreSolverBase.stopped {
							child++
						}
						if (c.RA.work != nil || c.RA.transposeRow != nil || c.RPrimal.PairCoreSolverBase.stopped) && s.calls.pending() {
							t.Fatal("outer owner adopted child failure")
						}
						if s.RemoveColumnFromTableau(j) != ErrExactVectorPending || s.PopTableau() != ErrExactVectorPending {
							t.Fatal("lost stopped outer owner")
						}
					}
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if failures == 0 || !success {
				t.Fatal("incomplete physical campaign", direct, op, failures, success)
			}
			if own == 0 {
				t.Fatal("no outer expression refusal", direct, op)
			}
			if op != "cost" && op != "pop" && child == 0 {
				t.Fatal("no child refusal", direct, op)
			}
		}
	}
}

func TestLARTableauRemovalSourceGuardsAndPrefixes(t *testing.T) {
	for _, op := range []string{"bottom-dim", "bottom-column", "bottom-empty", "empty-column", "last-index", "row-cost-extent", "row-heading-short", "row-d-short", "row-rhs-empty", "element-link", "basis-index", "basis-empty", "nonbasis-empty", "basis-slot", "basic-head-foreign", "nb-slot", "nb-head-foreign", "basis-count", "remove-column-cost", "remove-column-witness", "remove-Xempty", "remove-Dempty", "pop-cost-extent", "pop-basis-count", "pop-heading-wrong", "after-basis-unit", "basis-heading", "row-nonempty-finalcol"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpNewRemovalModel(t, h, "2", "3", false)
			c := s.Core
			want := ErrBounds
			var err error
			switch op {
			case "bottom-dim":
				err = s.MakeSureBottomRightElementIsNonzeroInTableau(9, 2)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "bottom-column":
				err = s.MakeSureBottomRightElementIsNonzeroInTableau(0, 99)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "bottom-empty":
				c.RA.Columns[2] = nil
				err = s.MakeSureBottomRightElementIsNonzeroInTableau(0, 2)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "empty-column":
				err = s.RemoveLastColumnFromTableau(2)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "last-index":
				err = s.RemoveLastColumnFromTableau(99)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "row-cost-extent":
				c.Costs.Data = c.Costs.Data[:2]
				err = s.RemoveLastRowAndColumnFromTableau(2)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "row-heading-short":
				c.RHeading = c.RHeading[:2]
				err = s.RemoveLastRowAndColumnFromTableau(2)
			case "row-d-short":
				h.must(h.m.SetInt64(&c.Costs.Data[2], 1))
				c.RPrimal.D.Data = nil
				err = s.RemoveLastRowAndColumnFromTableau(2)
			case "row-rhs-empty":
				c.RightSides.Data = nil
				err = s.RemoveLastRowAndColumnFromTableau(2)
				if c.RA.RowCount() != 0 || c.RA.ColumnCount() != 2 {
					t.Fatal("geometry must shrink before RHS guard")
				}
			case "element-link":
				row := c.RA.Rows[0]
				row[len(row)-1].Offset = 99
				err = s.RemoveLastRowAndColumnFromTableau(2)
			case "basis-index":
				err = s.RemoveLastColumnFromBasisTableau(99)
			case "basis-empty":
				c.RBasis = nil
				err = s.RemoveLastColumnFromBasisTableau(2)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "nonbasis-empty":
				c.RHeading[2] = -1
				c.RNBasis = nil
				err = s.RemoveLastColumnFromBasisTableau(2)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "basis-slot":
				c.RHeading[2] = 9
				err = s.RemoveLastColumnFromBasisTableau(2)
			case "basic-head-foreign":
				c.RBasis = []uint32{2, 99}
				err = s.RemoveLastColumnFromBasisTableau(2)
				if c.RBasis[0] != 99 {
					t.Fatal("basic swap prefix")
				}
			case "nb-slot":
				c.RHeading[2] = -99
				err = s.RemoveLastColumnFromBasisTableau(2)
			case "nb-head-foreign":
				c.RHeading[2] = -1
				c.RNBasis = []uint32{0, 99}
				err = s.RemoveLastColumnFromBasisTableau(2)
				if c.RNBasis[0] != 99 {
					t.Fatal("nonbasic swap prefix")
				}
			case "basis-count":
				err = s.RemoveLastColumnFromBasisTableau(2)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "remove-column-cost":
				c.Costs.Data = c.Costs.Data[:2]
				err = s.RemoveColumnFromTableau(2)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "remove-column-witness":
				s.Bounds.Witnesses.values = s.Bounds.Witnesses.values[:2]
				err = s.RemoveColumnFromTableau(2)
			case "remove-Xempty":
				c.RX.Data = nil
				err = s.RemoveColumnFromTableau(2)
			case "remove-Dempty":
				c.RPrimal.D.Data = nil
				err = s.RemoveColumnFromTableau(2)
			case "pop-cost-extent":
				c.Costs.Data = c.Costs.Data[:2]
				err = s.PopTableau()
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "pop-basis-count":
				c.RBasis = nil
				err = s.PopTableau()
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "after-basis-unit":
				_, err = s.AddVar(89)
				h.must(err)
				value, e := c.RA.ColumnValue(c.RA.Columns[2][0])
				h.must(e)
				h.must(h.m.SetInt64(value, 2))
				err = s.RemoveColumnFromTableau(3)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
				if c.RA.ColumnCount() != 3 || len(c.RX.Data) != 3 {
					t.Fatal("basis validation must follow numeric pops")
				}
			case "basis-heading":
				savedRows, savedCols := c.RA.Rows, c.RA.Columns
				c.RA.Rows = nil
				c.RA.Columns = c.RA.Columns[:2]
				c.RHeading[0] = -9
				err = s.RemoveLastColumnFromBasisTableau(2)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
				c.RA.Rows, c.RA.Columns = savedRows, savedCols
			case "row-nonempty-finalcol":
				_, e := s.AddTerm([]RationalSparseEntry{{Index: 1, Value: h.pair("7", "0").X}}, &h.pair("0", "0").X)
				h.must(e)
				h.must(c.RA.Set(0, 3, &h.pair("9", "0").X))
				err = s.RemoveLastRowAndColumnFromTableau(3)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "pop-heading-wrong":
				c.RHeading[2] = -3
				err = s.PopTableau()
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			}
			if err != want {
				t.Fatal(op, err, want)
			}
			if err != nil {
				if s.MakeSureBottomRightElementIsNonzeroInTableau(0, 2) != ErrExactVectorPending || s.RemoveLastRowAndColumnFromTableau(2) != ErrExactVectorPending || s.RemoveLastColumnFromTableau(2) != ErrExactVectorPending || s.RemoveLastColumnFromBasisTableau(2) != ErrExactVectorPending || s.RemoveColumnFromTableau(2) != ErrExactVectorPending || s.PopTableau() != ErrExactVectorPending {
					t.Fatal("stopped mutation accepted")
				}
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARCoreTableauBasisValidatorSourceProfiles(t *testing.T) {
	for _, op := range []string{"valid", "LU", "multi", "wrong-value", "bad-column", "bad-offset", "short-types", "fixed-skip", "bad-nonbasis", "int-min"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpNewRemovalModel(t, h, "2", "3", false)
			c := s.Core
			want := error(nil)
			switch op {
			case "LU":
				s.Settings.SimplexStrategy = StrategyLU
			case "multi":
				h.must(c.RA.AddRow())
				h.must(c.RA.Set(1, 2, &h.pair("3", "0").X))
				want = ErrAssertion
			case "wrong-value":
				h.must(h.m.SetInt64(&c.RA.Rows[0][len(c.RA.Rows[0])-1].Value, 2))
				want = ErrAssertion
			case "bad-column":
				c.RBasis[0] = 99
				want = ErrAssertion
			case "bad-offset":
				c.RA.Columns[2][0].Offset = 99
				want = ErrBounds
			case "short-types":
				c.RBasis = nil
				c.Bounds.Types.values = nil
				want = ErrBounds
			case "fixed-skip":
				c.RBasis = nil
				c.Bounds.Types.values[0] = ColumnFixed
				c.RHeading[0] = -99
			case "bad-nonbasis":
				c.RHeading[0] = -99
				want = ErrAssertion
			case "int-min":
				c.RHeading[0] = -1 << 31
				want = ErrBounds
			}
			if !sourceDebugEnabled {
				want = nil
			}
			ok, err := c.RBasisIsOK()
			if err != want || ok != (want == nil) {
				t.Fatal(ok, err, want)
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}
