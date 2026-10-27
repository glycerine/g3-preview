package lp

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

func lpCoreScopeAddColumn(t *testing.T, h *lpPairHarness, c *LARCoreSolver, basic bool) {
	t.Helper()
	j := c.RA.ColumnCount()
	h.must(c.RA.AddColumn())
	c.DA.AddColumn()
	c.Bounds.Types.Append(ColumnFree)
	h.must(c.Bounds.Low.IncreaseSizeByOne())
	h.must(c.Bounds.Upper.IncreaseSizeByOne())
	h.must(c.RX.resizeData(j + 1))
	h.must(c.Costs.resizeData(j + 1))
	h.must(c.RPrimal.D.resizeData(j + 1))
	c.DX = resizeFloatSolverValues(c.DX, int(j)+1)
	c.DLow = resizeFloatSolverValues(c.DLow, int(j)+1)
	c.DUpper = resizeFloatSolverValues(c.DUpper, int(j)+1)
	c.DCosts = resizeFloatSolverValues(c.DCosts, int(j)+1)
	c.RPrimal.ColumnsNZ = append(c.RPrimal.ColumnsNZ, j+11)
	if basic {
		row := c.RA.RowCount()
		h.must(c.RA.AddRow())
		c.DA.AddRow()
		h.must(c.RA.Set(row, j, &h.pair("1", "0").X))
		h.must(c.DA.Set(row, j, 1))
		if j != 0 {
			h.must(c.RA.Set(row, 0, &h.pair("2", "0").X))
			h.must(c.DA.Set(row, 0, 2))
		}
		c.RBasis = append(c.RBasis, j)
		c.DBasis = append(c.DBasis, j)
		c.RHeading = append(c.RHeading, int32(row))
		c.DHeading = append(c.DHeading, int32(row))
		c.RPrimal.RowsNZ = append(c.RPrimal.RowsNZ, row+21)
	} else {
		c.RHeading = append(c.RHeading, -int32(len(c.RNBasis))-1)
		c.DHeading = append(c.DHeading, -int32(len(c.DNBasis))-1)
		c.RNBasis = append(c.RNBasis, j)
		c.DNBasis = append(c.DNBasis, j)
	}
}

// Supply actual coupled core fields, without pretending that the outer model's
// terms, witnesses, directories or feasibility state have been scoped here.
func lpNewCoreScopeFixture(t *testing.T, h *lpPairHarness) *LARCoreSolver {
	t.Helper()
	s, err := NewSettings()
	h.must(err)
	s.SimplexStrategy = StrategyLU
	c, err := NewLARCoreSolver(h.m, s, nil)
	h.must(err)
	lpCoreScopeAddColumn(t, h, c, true)
	lpCoreScopeAddColumn(t, h, c, false)
	lpCoreScopeAddColumn(t, h, c, false)
	for j := uint32(1); j < 3; j++ {
		h.must(c.RA.Set(0, j, &h.pair(fmt.Sprint(j+1), "0").X))
		h.must(c.DA.Set(0, j, float64(j+1)))
	}
	h.must(InitPairFactorization(&c.RPrimal.Factorization, c.RA, c.RBasis, s))
	h.must(InitFloatFactorization(&c.DPrimal.Factorization, c.DA, c.DBasis, s))
	return c
}

type lpCoreScopeSnapshot struct {
	rows, cols               uint32
	types                    []ColumnType
	low, upper               [][2]string
	rBasis, dBasis, cnz, rnz []uint32
}

func lpCoreScopePairStrings(t *testing.T, h *lpPairHarness, v *StackedPairVector) [][2]string {
	t.Helper()
	out := make([][2]string, v.Size())
	for j := range out {
		pair, err := v.At(uint32(j))
		h.must(err)
		out[j][0], err = h.m.String(&pair.X)
		h.must(err)
		out[j][1], err = h.m.String(&pair.Y)
		h.must(err)
	}
	return out
}

func lpCoreScopeSnapshotOf(t *testing.T, h *lpPairHarness, c *LARCoreSolver) lpCoreScopeSnapshot {
	t.Helper()
	return lpCoreScopeSnapshot{c.RA.RowCount(), c.RA.ColumnCount(), append([]ColumnType(nil), c.Bounds.Types.Values()...), lpCoreScopePairStrings(t, h, c.Bounds.Low), lpCoreScopePairStrings(t, h, c.Bounds.Upper), append([]uint32(nil), c.RBasis...), append([]uint32(nil), c.DBasis...), append([]uint32(nil), c.RPrimal.ColumnsNZ...), append([]uint32(nil), c.RPrimal.RowsNZ...)}
}

func lpCheckCoreScopeSnapshot(t *testing.T, h *lpPairHarness, c *LARCoreSolver, want lpCoreScopeSnapshot) {
	t.Helper()
	if got := lpCoreScopeSnapshotOf(t, h, c); !reflect.DeepEqual(got, want) || c.DA.RowCount() != want.rows || c.DA.ColumnCount() != want.cols {
		t.Fatalf("core scope state\ngot %+v\nwant %+v", got, want)
	}
	if c.RX.DataSize() != want.cols || c.Costs.DataSize() != want.cols || c.RPrimal.D.DataSize() != want.cols || len(c.DX) != int(want.cols) {
		t.Fatal("source exact value/cost/D and floating X extents")
	}
	for _, data := range []struct {
		basis, nb []uint32
		heading   []int32
	}{{c.RBasis, c.RNBasis, c.RHeading}, {c.DBasis, c.DNBasis, c.DHeading}} {
		basisRows := make(map[uint32]int32)
		for i, j := range data.basis {
			basisRows[j] = int32(i)
		}
		var nonbasic []uint32
		for j := want.cols; j > 0; {
			j--
			row, basic := basisRows[j]
			if basic {
				if data.heading[j] != row {
					t.Fatal("restored basis heading")
				}
			} else {
				if data.heading[j] != -int32(len(nonbasic))-1 {
					t.Fatal("restored nonbasis heading")
				}
				nonbasic = append(nonbasic, j)
			}
		}
		if !reflect.DeepEqual(data.nb, nonbasic) {
			t.Fatal("nonbasis must be reconstructed in descending column order")
		}
	}
}

func TestLARCoreScopesIndependentSnapshots(t *testing.T) {
	h := newLPPairHarness(t)
	c := lpNewCoreScopeFixture(t, h)
	rng := rand.New(rand.NewPCG(827491, 177))
	var scopes []lpCoreScopeSnapshot
	for step := 0; step < 1600; step++ {
		op := rng.IntN(5)
		if len(scopes) == 0 || op == 0 && len(scopes) < 6 {
			scopes = append(scopes, lpCoreScopeSnapshotOf(t, h, c))
			h.budget.FailAfter(0)
			err := c.Push()
			h.budget.FailAfter(-1)
			h.must(err)
			lpCoreScopeAddColumn(t, h, c, rng.IntN(2) == 0)
		} else if op == 1 {
			k := 1 + rng.IntN(len(scopes))
			want := scopes[len(scopes)-k]
			h.must(c.Pop(uint32(k)))
			scopes = scopes[:len(scopes)-k]
			lpCheckCoreScopeSnapshot(t, h, c, want)
			if c.RPrimal.Factorization != nil || c.DPrimal.Factorization != nil || c.StackedSimplexStrategy.StackSize() != uint32(len(scopes)) {
				t.Fatal("factorization retirement/scope depth")
			}
		} else {
			j := uint32(rng.IntN(int(c.RA.ColumnCount())))
			v := h.pair(fmt.Sprint(step+1), fmt.Sprint(-step-1))
			h.must(c.Bounds.Low.Set(j, v))
			h.must(c.Bounds.Upper.Set(j, h.pair(fmt.Sprint(step+10), fmt.Sprint(step+2))))
			h.must(c.Bounds.Types.Set(j, ColumnBoxed))
			c.RPrimal.ColumnsNZ[j] = uint32(step + 31)
			c.RPrimal.RowsNZ[0] = uint32(step + 41)
			// Both original basis alternatives have nonzero row coefficients.
			c.RBasis[0], c.DBasis[0] = uint32(step%2), uint32((step+1)%2)
			h.must(c.RPrimal.InitBasisHeadingAndNonBasicColumns())
			h.must(c.DPrimal.InitBasisHeadingAndNonBasicColumns())
		}
	}
	if len(scopes) != 0 {
		want := scopes[0]
		h.must(c.Pop(uint32(len(scopes))))
		lpCheckCoreScopeSnapshot(t, h, c, want)
	}
	h.must(c.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARCoreScopesTableauAndSurvivingValues(t *testing.T) {
	h := newLPPairHarness(t)
	for _, strategy := range []SimplexStrategy{StrategyTableauRows, StrategyTableauCosts, StrategyUndecided} {
		c := lpNewCoreScopeFixture(t, h)
		c.settings.SimplexStrategy = strategy
		rFactor, dFactor := c.RPrimal.Factorization, c.DPrimal.Factorization
		h.must(c.Push())
		if (len(c.RA.stack) == 0) != c.settings.UseTableau() || (len(c.RPushedBasis.vectorSizes) == 0) != c.settings.UseTableau() {
			t.Fatal("source conditional rational/basis snapshots")
		}
		h.must(c.RX.Data[1].Copy(h.m, h.pair("17/19", "2/23")))
		h.must(h.m.Copy(&c.Costs.Data[1], &h.pair("29/31", "0").X))
		h.must(h.m.Copy(&c.RPrimal.D.Data[1], &h.pair("37/41", "0").X))
		h.must(h.m.Copy(&c.RA.Rows[0][0].Value, &h.pair("43/47", "0").X))
		c.DA.Rows[0][0].Value = 3.25
		c.DX[1] = 5.5
		c.DLow[1], c.DUpper[1], c.DCosts[1] = 6.5, 7.5, 8.5
		c.RPrimal.ColumnsNZ[0] = 301
		c.RPrimal.RowsNZ[0] = 302
		c.RBasis[0], c.DBasis[0] = 1, 2
		h.must(c.RPrimal.InitBasisHeadingAndNonBasicColumns())
		h.must(c.DPrimal.InitBasisHeadingAndNonBasicColumns())
		c.Bounds.Status.Value = StatusInfeasible
		h.must(c.Pop(1))
		if c.RPrimal.Factorization != nil || c.DPrimal.Factorization != nil || !errors.Is(rFactor.ready(), ErrLUClosed) || !errors.Is(dFactor.ready(), ErrLUClosed) {
			t.Fatal("concrete factorization owners not retired")
		}
		h.check(&c.RX.Data[1], mustLPRat(t, "17/19"), mustLPRat(t, "2/23"))
		lpSparseRationalCheck(t, h, &c.Costs.Data[1], mustLPRat(t, "29/31"))
		lpSparseRationalCheck(t, h, &c.RPrimal.D.Data[1], mustLPRat(t, "37/41"))
		lpSparseRationalCheck(t, h, &c.RA.Rows[0][0].Value, mustLPRat(t, "43/47"))
		if c.DA.Rows[0][0].Value != 3.25 || c.DX[1] != 5.5 || c.DLow[1] != 6.5 || c.DUpper[1] != 7.5 || c.DCosts[1] != 8.5 || c.Bounds.Status.Value != StatusInfeasible {
			t.Fatal("unscoped surviving values/outer status were restored")
		}
		if strategy != StrategyUndecided {
			if c.RBasis[0] != 1 || !reflect.DeepEqual(c.DBasis, c.RBasis) || !reflect.DeepEqual(c.DNBasis, c.RNBasis) || !reflect.DeepEqual(c.DHeading, c.RHeading) || c.RPrimal.ColumnsNZ[0] != 301 || c.RPrimal.RowsNZ[0] != 302 {
				t.Fatal("tableau copies current exact basis; counts remain current")
			}
		} else if c.RBasis[0] != 0 || c.DBasis[0] != 0 || c.RPrimal.ColumnsNZ[0] != 11 || c.RPrimal.RowsNZ[0] != 21 {
			t.Fatal("undecided restores independent basis/count snapshots")
		}
		if c.settings.SimplexStrategy != strategy {
			t.Fatal("strategy restoration")
		}
		h.must(c.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARCoreScopesPhysicalBoundRestoration(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	large := "1" + strings.Repeat("0", 999) + "7"
	for _, direct := range []bool{false, true} {
		for _, upper := range []bool{false, true} {
			failures, success := 0, false
			for ordinal := int64(0); ordinal < 100; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%v/%v/%d", direct, upper, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					c := lpNewCoreScopeFixture(t, h)
					v := c.Bounds.Low
					if upper {
						v = c.Bounds.Upper
					}
					h.must(v.Set(0, h.pair(large+"/7", large+"/11")))
					h.must(c.Push())
					// Release the destination's big payload so journal restoration
					// must really allocate rather than reuse an equal-size buffer.
					h.must(v.Set(0, h.pair("0", "0")))
					lpCoreScopeAddColumn(t, h, c, true)
					h.budget.FailAfter(ordinal)
					err := c.Pop(1)
					h.budget.FailAfter(-1)
					if err != nil {
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						failures++
						if c.RA.ColumnCount() != 3 || c.RA.RowCount() != 1 || c.DA.ColumnCount() != 4 || len(c.Bounds.Types.vectorSizes) != 1 || c.StackedSimplexStrategy.StackSize() != 1 || c.RPrimal.Factorization == nil || c.DPrimal.Factorization == nil {
							t.Fatal("source bound-restore failure prefix")
						}
						if upper && len(c.Bounds.Low.vectorSizes) != 0 || !upper && len(c.Bounds.Upper.vectorSizes) != 1 {
							t.Fatal("lower-before-upper scope publication")
						}
						raw, div, cmp := h.m.RawInvocationState(), h.m.DivisionInvocationState(), h.m.ComparisonInvocationState()
						if e := c.Pop(1); !errors.Is(e, ErrExactVectorPending) {
							t.Fatal("stopped pop", e)
						}
						if e := c.Push(); !errors.Is(e, ErrExactVectorPending) {
							t.Fatal("stopped push", e)
						}
						if raw != h.m.RawInvocationState() || div != h.m.DivisionInvocationState() || cmp != h.m.ComparisonInvocationState() {
							t.Fatal("stopped core changed numeric ownership")
						}
					} else {
						done = true
						if c.RA.ColumnCount() != 3 || c.DA.ColumnCount() != 3 || c.RPrimal.Factorization != nil || c.DPrimal.Factorization != nil {
							t.Fatal("successful full core restore")
						}
					}
					h.must(c.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					success = true
					break
				}
			}
			if !success || failures == 0 {
				t.Fatal(direct, upper, success, failures)
			}
		}
	}
}

func TestLARCoreScopesPendingPushPublication(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	large := "1" + strings.Repeat("0", 999) + "7"
	for _, direct := range []bool{false, true} {
		for _, upper := range []bool{false, true} {
			var h *lpPairHarness
			if direct {
				h = lpNewDirectSparseHarness(t)
			} else {
				h = newLPPairHarness(t)
			}
			c := lpNewCoreScopeFixture(t, h)
			v := c.Bounds.Low
			if upper {
				v = c.Bounds.Upper
			}
			input := h.pair(large+"/7", large+"/11")
			h.must(v.Set(0, input))
			zero := h.pair("0", "0")
			h.budget.FailAfter(0)
			err := v.Set(0, zero)
			h.budget.FailAfter(-1)
			if !lpExactPriorityPhysicalRefusal(err) || !v.unfinished() {
				t.Fatal("no real child copy refusal", err)
			}
			raw, div, cmp := h.m.RawInvocationState(), h.m.DivisionInvocationState(), h.m.ComparisonInvocationState()
			if e := c.Push(); !errors.Is(e, ErrPairVectorPending) {
				t.Fatal("core push ignored pending bound child", e)
			}
			if len(c.RA.stack) != 1 || len(c.Bounds.Types.vectorSizes) != 1 || c.StackedSimplexStrategy.StackSize() != 1 || len(c.RPushedBasis.vectorSizes) != 0 || len(c.DA.stack) != 0 {
				t.Fatal("partial push order")
			}
			wantLowDepth := 0
			if upper {
				wantLowDepth = 1
			}
			if len(c.Bounds.Low.vectorSizes) != wantLowDepth || len(c.Bounds.Upper.vectorSizes) != 0 || raw != h.m.RawInvocationState() || div != h.m.DivisionInvocationState() || cmp != h.m.ComparisonInvocationState() {
				t.Fatal("bound child scope/receipt prefix")
			}
			h.must(c.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLARCoreScopesCurrentStrategyAndSnapshotOwnership(t *testing.T) {
	h := newLPPairHarness(t)
	for _, tableau := range []bool{false, true} {
		c := lpNewCoreScopeFixture(t, h)
		saved := StrategyUndecided
		if tableau {
			saved = StrategyLU
		}
		c.settings.SimplexStrategy = saved
		h.must(c.Push())
		c.settings.SimplexStrategy = StrategyLU
		if tableau {
			// Explicit current-mode profile with equal matrix dimensions; this
			// does not port the solver's missing strategy-migration algorithm.
			c.settings.SimplexStrategy = StrategyTableauRows
			c.RBasis[0] = 1
			h.must(c.RPrimal.InitBasisHeadingAndNonBasicColumns())
		}
		h.must(c.Pop(1))
		if c.settings.SimplexStrategy != saved {
			t.Fatal("strategy must restore after basis/count branches")
		}
		if tableau {
			if c.RBasis[0] != 1 || len(c.RA.stack) != 1 || len(c.RPushedBasis.vectorSizes) != 1 || len(c.RColumnsNZ.vectorSizes) != 1 || len(c.DA.stack) != 0 {
				t.Fatal("current tableau mode must skip rational/basis/count pops")
			}
		} else {
			if len(c.RA.stack) != 0 || len(c.RPushedBasis.vectorSizes) != 0 {
				t.Fatal("current LU mode did not pop saved scopes")
			}
			old := c.RPushedBasis.Values()[0]
			c.RBasis[0] = 1
			if c.RPushedBasis.Values()[0] != old {
				t.Fatal("restored basis aliases snapshot owner")
			}
			old = c.RColumnsNZ.Values()[0]
			c.RPrimal.ColumnsNZ[0] = 981
			if c.RColumnsNZ.Values()[0] != old {
				t.Fatal("restored counts alias snapshot owner")
			}
		}
		h.must(c.Close())
	}
	var snapshot StackedUintVector
	snapshot.Append(7)
	snapshot.Append(8)
	snapshot.Append(9)
	err := pushLARUintVector(&snapshot, []uint32{11, 12})
	if sourceDebugEnabled {
		if !errors.Is(err, ErrAssertion) || len(snapshot.vectorSizes) != 0 || !reflect.DeepEqual(snapshot.Values(), []uint32{7, 8, 9}) {
			t.Fatal("source oversized snapshot assertion prefix", err)
		}
	} else {
		h.must(err)
		if !reflect.DeepEqual(snapshot.Values(), []uint32{11, 12, 9}) {
			t.Fatal("native source helper must not truncate old tail")
		}
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARCoreScopesInvalidDepthAndDebugGuards(t *testing.T) {
	h := newLPPairHarness(t)
	for _, k := range []uint32{0, 2} {
		c := lpNewCoreScopeFixture(t, h)
		h.must(c.Push())
		lpCoreScopeAddColumn(t, h, c, true)
		if err := c.Pop(k); !errors.Is(err, ErrScope) {
			t.Fatal("invalid bound-scope depth", err)
		}
		wantColumns, wantStack := uint32(4), 1
		if k == 2 {
			wantColumns, wantStack = 3, 0
		}
		if c.RA.ColumnCount() != wantColumns || len(c.RA.stack) != wantStack || len(c.Bounds.Low.vectorSizes) != 1 || c.DA.ColumnCount() != 4 {
			t.Fatal("matrix pop precedes invalid bound depth guard")
		}
		if err := c.Pop(1); !errors.Is(err, ErrExactVectorPending) {
			t.Fatal("stopped invalid pop", err)
		}
		h.must(c.Close())
	}
	for _, mode := range []string{"r-heading", "d-heading", "types", "r-size", "d-size"} {
		c := lpNewCoreScopeFixture(t, h)
		switch mode {
		case "r-heading":
			c.RHeading[0] = -1
		case "d-heading":
			c.DHeading[0] = -1
		case "types":
			c.Bounds.Types.values = c.Bounds.Types.values[:2]
		case "r-size":
			c.RHeading = nil
		case "d-size":
			c.DHeading = nil
		}
		err := c.Push()
		if sourceDebugEnabled {
			if !errors.Is(err, ErrAssertion) || c.StackedSimplexStrategy.StackSize() != 0 || len(c.RA.stack) != 0 {
				t.Fatal("source debug assertion before publication", mode, err)
			}
		} else {
			h.must(err)
		}
		h.must(c.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARCoreScopesActualTableauBindings(t *testing.T) {
	h := newLPPairHarness(t)
	s, _ := lpDispatchModel(t, h, false, "2", "3")
	c := s.Core
	before := lpCoreScopePairStrings(t, h, c.Bounds.Low)
	h.must(c.Push())
	h.must(c.Bounds.Low.Set(0, h.pair("19/23", "1/29")))
	h.must(c.Pop(1))
	if !reflect.DeepEqual(lpCoreScopePairStrings(t, h, c.Bounds.Low), before) || c.DA.ColumnCount() != 0 || len(c.DX) != 0 || !reflect.DeepEqual(c.DBasis, c.RBasis) || !reflect.DeepEqual(c.DHeading, c.RHeading) {
		t.Fatal("actual tableau core bindings")
	}
	if len(c.RA.stack) != 0 || len(c.Bounds.Witnesses.vectorSizes) != 0 || len(c.Bounds.Status.stack) != 0 || len(s.Terms) != 1 || len(s.Constraints) != 4 {
		t.Fatal("core scope changed outer model scope state")
	}
	h.must(s.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARCoreScopesStoppedChildAndMatrixPrefixes(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	large := "1" + strings.Repeat("0", 999) + "7"
	for _, mode := range []string{"push-primal", "pop-primal", "push-matrix", "pop-matrix", "pop-r-links", "pop-d-links"} {
		h := newLPPairHarness(t)
		c := lpNewCoreScopeFixture(t, h)
		popping := strings.HasPrefix(mode, "pop-")
		if popping {
			h.must(c.Push())
		}
		want := ErrExactVectorPending
		switch mode {
		case "push-primal", "pop-primal":
			if e := c.RPrimal.CalculatePivotRowOfBInverse(99); !errors.Is(e, ErrBounds) {
				t.Fatal("no actual precise-core stop", e)
			}
		case "push-matrix", "pop-matrix":
			v := h.pair(large, "0")
			h.budget.FailAfter(0)
			err := c.RA.Set(0, 1, &v.X)
			h.budget.FailAfter(-1)
			if !lpExactPriorityPhysicalRefusal(err) {
				t.Fatal("no actual matrix copy refusal", err)
			}
		case "pop-r-links", "pop-d-links":
			lpCoreScopeAddColumn(t, h, c, true)
			want = ErrBounds
			// Explicit invalid matrix links exercise source bounds guards;
			// these are not allocator refusals or ordinary scope profiles.
			if mode == "pop-r-links" {
				c.RA.Rows[1][0].Column = 99
			} else {
				c.DA.Rows[1][0].Column = 99
			}
		}
		var err error
		if popping {
			err = c.Pop(1)
		} else {
			err = c.Push()
		}
		if !errors.Is(err, want) {
			t.Fatal(mode, err, want)
		}
		switch mode {
		case "push-primal":
			if c.StackedSimplexStrategy.StackSize() != 0 || len(c.Bounds.Types.vectorSizes) != 0 {
				t.Fatal("primal-ready guard precedes scope publication")
			}
		case "push-matrix":
			if c.StackedSimplexStrategy.StackSize() != 1 || len(c.Bounds.Types.vectorSizes) != 1 || len(c.Bounds.Low.vectorSizes) != 0 || len(c.RA.stack) != 0 {
				t.Fatal("matrix push failure follows strategy/type publication")
			}
		case "pop-primal", "pop-matrix", "pop-r-links":
			if len(c.Bounds.Low.vectorSizes) != 1 || len(c.Bounds.Types.vectorSizes) != 1 || c.RPrimal.Factorization == nil {
				t.Fatal("early pop failure advanced later fields")
			}
		case "pop-d-links":
			if len(c.Bounds.Low.vectorSizes) != 0 || len(c.Bounds.Types.vectorSizes) != 0 || c.RPrimal.Factorization != nil || c.DPrimal.Factorization == nil {
				t.Fatal("floating matrix failure must follow exact retirement")
			}
		}
		if mode == "pop-r-links" {
			c.RA.Rows[1][0].Column = 3
		} else if mode == "pop-d-links" {
			c.DA.Rows[1][0].Column = 3
		}
		h.must(c.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARCoreScopesCachedSnapshotSizePrefixes(t *testing.T) {
	h := newLPPairHarness(t)
	for _, mode := range []string{"r-basis", "columns", "rows", "d-basis"} {
		c := lpNewCoreScopeFixture(t, h)
		dst, sourceSize := &c.RPushedBasis, len(c.RBasis)
		switch mode {
		case "columns":
			dst, sourceSize = &c.RColumnsNZ, len(c.RPrimal.ColumnsNZ)
		case "rows":
			dst, sourceSize = &c.RRowsNZ, len(c.RPrimal.RowsNZ)
		case "d-basis":
			dst, sourceSize = &c.DPushedBasis, len(c.DBasis)
		}
		for i := 0; i <= sourceSize; i++ {
			dst.Append(791)
		}
		err := c.Push()
		if sourceDebugEnabled {
			if !errors.Is(err, ErrAssertion) || len(dst.vectorSizes) != 0 || len(c.Bounds.Low.vectorSizes) != 1 || len(c.Bounds.Upper.vectorSizes) != 1 || len(c.RA.stack) != 1 {
				t.Fatal("cached snapshot assertion prefix", mode, err)
			}
			wantDDepth := 0
			if mode == "d-basis" {
				wantDDepth = 1
			}
			if len(c.DA.stack) != wantDDepth {
				t.Fatal("double matrix precedes final basis snapshot")
			}
		} else {
			h.must(err)
			if int(dst.Size()) != sourceSize+1 || dst.Values()[sourceSize] != 791 {
				t.Fatal("native helper must retain extra cached tail")
			}
		}
		h.must(c.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARCoreScopesUnscopedExtentsAndModeMismatch(t *testing.T) {
	h := newLPPairHarness(t)
	c := lpNewCoreScopeFixture(t, h)
	h.must(c.RightSides.Resize(2))
	h.must(c.RightSides.Data[1].Copy(h.m, h.pair("17/19", "23/29")))
	h.must(c.Push())
	lpCoreScopeAddColumn(t, h, c, true)
	c.DLow[3], c.DUpper[3], c.DCosts[3] = 5.5, 6.5, 7.5
	h.must(c.Pop(1))
	if len(c.DX) != 3 || len(c.DLow) != 4 || len(c.DUpper) != 4 || len(c.DCosts) != 4 || c.DLow[3] != 5.5 || c.DUpper[3] != 6.5 || c.DCosts[3] != 7.5 || c.RightSides.DataSize() != 2 {
		t.Fatal("only floating X and named exact vectors shrink")
	}
	h.check(&c.RightSides.Data[1], mustLPRat(t, "17/19"), mustLPRat(t, "23/29"))
	h.must(c.Close())

	c = lpNewCoreScopeFixture(t, h)
	c.settings.SimplexStrategy = StrategyTableauRows
	h.must(c.Push())
	c.settings.SimplexStrategy = StrategyLU
	// Explicit unmatched mode profile, without inventing a migration adapter:
	// current LU attempts count scopes that tableau Push never created.
	if err := c.Pop(1); !errors.Is(err, ErrScope) {
		t.Fatal("missing current-mode count scope", err)
	}
	if len(c.Bounds.Low.vectorSizes) != 0 || len(c.Bounds.Types.vectorSizes) != 0 || c.RPrimal.Factorization != nil || c.DPrimal.Factorization == nil || len(c.DA.stack) != 1 || c.StackedSimplexStrategy.StackSize() != 1 || c.settings.SimplexStrategy != StrategyLU {
		t.Fatal("mode mismatch failure must precede floating pop and strategy restoration")
	}
	h.must(c.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARCoreScopesRetiresUpdatedFactorizations(t *testing.T) {
	h := newLPPairHarness(t)
	c := lpNewCoreScopeFixture(t, h)
	h.must(c.Push())
	lpCoreScopeAddColumn(t, h, c, true)
	// Factorizations created inside the scope have the enlarged geometry.
	// Rational matrix rollback precedes their destruction in the source.
	h.must(InitPairFactorization(&c.RPrimal.Factorization, c.RA, c.RBasis, c.settings))
	h.must(InitFloatFactorization(&c.DPrimal.Factorization, c.DA, c.DBasis, c.settings))
	rFactor, dFactor := c.RPrimal.Factorization, c.DPrimal.Factorization
	h.must(c.Pop(1))
	if c.RA.RowCount() != 1 || c.DA.RowCount() != 1 || c.RPrimal.Factorization != nil || c.DPrimal.Factorization != nil || !errors.Is(rFactor.ready(), ErrLUClosed) || !errors.Is(dFactor.ready(), ErrLUClosed) {
		t.Fatal("enlarged factorization owners survived matrix rollback")
	}
	h.must(c.Close())
	lpExactVectorNamedCensus(t, h, nil)
}
