// Source: src/util/lp/lar_core_solver.h:533-582.
package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

func lpLARTableauCatchUpFixture(t *testing.T, h *lpPairHarness, rows, cols, world int) (*LARCoreSolver, [][]*big.Rat) {
	t.Helper()
	c, a := lpLARCatchUpFixture(t, h, rows, cols, world)
	c.settings.SimplexStrategy = StrategyTableauRows
	h.must(c.PrefixR())
	h.must(c.RPrimal.Factorization.Close())
	c.RPrimal.Factorization = nil
	for row, j := range c.RBasis {
		ok, err := c.RPrimal.PivotColumnTableau(j, uint32(row))
		h.must(err)
		if !ok {
			t.Fatal("initial tableau basis")
		}
	}
	return c, a
}

// Independently characterize the complete result by B*T=A, where A is the
// original matrix and B consists of its final basic columns. No test pivot
// implementation or copy of the production queue algorithm is needed.
func lpCheckLARTableauCatchUpMatrix(t *testing.T, h *lpPairHarness, c *LARCoreSolver, a [][]*big.Rat) {
	t.Helper()
	rows, cols := len(a), len(a[0])
	tableau := make([][]*big.Rat, rows)
	for i := 0; i < rows; i++ {
		tableau[i] = make([]*big.Rat, cols)
		for j := range tableau[i] {
			tableau[i][j] = new(big.Rat)
		}
		for k := range c.RA.Rows[i] {
			cell := &c.RA.Rows[i][k]
			v, err := h.m.String(&cell.Value)
			h.must(err)
			tableau[i][cell.Column].Add(tableau[i][cell.Column], mustLPRat(t, v))
		}
	}
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			actual := new(big.Rat)
			for k, basic := range c.RBasis {
				actual.Add(actual, new(big.Rat).Mul(a[i][basic], tableau[k][j]))
			}
			if actual.Cmp(a[i][j]) != 0 {
				t.Fatal("independent B*T=A", i, j, actual, a[i][j], c.RBasis)
			}
		}
	}
	for i, basic := range c.RBasis {
		for row := 0; row < rows; row++ {
			want := int64(0)
			if row == i {
				want = 1
			}
			if tableau[row][basic].Cmp(big.NewRat(want, 1)) != 0 {
				t.Fatal("unit basis", row, basic, tableau[row][basic])
			}
		}
	}
	for j, cells := range c.RA.Columns {
		for k, link := range cells {
			if int(link.Row) >= len(c.RA.Rows) || int(link.Offset) >= len(c.RA.Rows[link.Row]) {
				t.Fatal("column link extent")
			}
			cell := c.RA.Rows[link.Row][link.Offset]
			if cell.Column != uint32(j) || cell.Offset != uint32(k) {
				t.Fatal("row/column reciprocal offsets")
			}
		}
	}
}

func TestLARTableauCatchUpIndependentTraceModels(t *testing.T) {
	rng := rand.New(rand.NewPCG(725319, 64155))
	for world := 0; world < 200; world++ {
		h := newLPPairHarness(t)
		rows, cols := 1+world%3, 5+world%3
		c, a := lpLARTableauCatchUpFixture(t, h, rows, cols, world)
		basis := make([]uint32, rows)
		non := make([]uint32, cols-rows)
		for i := range basis {
			basis[i] = uint32(i)
		}
		for i := range non {
			non[i] = uint32(rows + i)
		}
		var trace []uint32
		for step := 0; step < 8+world%29; step++ {
			r, n := rng.IntN(rows), rng.IntN(len(non))
			trace = append(trace, non[n], basis[r])
			basis[r], non[n] = non[n], basis[r]
		}
		target := make([]int32, cols)
		for i := range target {
			target[i] = -1
		}
		for i, j := range basis {
			target[j] = int32(i)
		}
		c.RPrimal.StartTracingBasisChanges()
		ok, err := c.CatchUpInLUTableau(trace, target)
		h.must(err)
		if !ok {
			t.Fatal("nonsingular final basis", world)
		}
		lpCheckLARCatchUpBasis(t, c.RBasis, c.RNBasis, c.RHeading, basis)
		lpCheckLARTableauCatchUpMatrix(t, h, c, a)
		for i := range c.RightSides.Data {
			h.check(&c.RightSides.Data[i], new(big.Rat), new(big.Rat))
		}
		if c.RPrimal.Factorization != nil {
			t.Fatal("tableau catch-up constructed an LU")
		}
		h.must(c.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARTableauCatchUpFalsePivotRestoresLastExchange(t *testing.T) {
	for _, late := range []bool{false, true} {
		h := newLPPairHarness(t)
		c, a := lpLARTableauCatchUpFixture(t, h, 2, 4, 1)
		trace, target := []uint32{2, 0}, []int32{-1, 1, 0, -2}
		want := []uint32{0, 1}
		if late {
			for i := 0; i < 2; i++ {
				a[i][3] = new(big.Rat).Set(a[i][2])
				text := "0"
				for k := range c.RA.Rows[i] {
					if c.RA.Rows[i][k].Column == 2 {
						var err error
						text, err = h.m.String(&c.RA.Rows[i][k].Value)
						h.must(err)
					}
				}
				lpSetLARCatchUpExactCell(t, h, c, uint32(i), 3, text)
			}
			trace, target = []uint32{2, 0, 3, 1}, []int32{-1, -2, 0, 1}
			want = []uint32{2, 1}
		} else {
			for i := 0; i < 2; i++ {
				a[i][2] = new(big.Rat)
				lpSetLARCatchUpExactCell(t, h, c, uint32(i), 2, "0")
			}
		}
		c.RPrimal.StartTracingBasisChanges()
		ok, err := c.CatchUpInLUTableau(trace, target)
		h.must(err)
		if ok || c.stopped {
			t.Fatal("source false is not a Go failure")
		}
		if !reflect.DeepEqual(c.RBasis, want) {
			t.Fatal("only last exchange rolls back", late, c.RBasis, want)
		}
		if late {
			if !reflect.DeepEqual(c.RPrimal.Trace, []uint32{2, 0}) {
				t.Fatal("retained earlier trace", c.RPrimal.Trace)
			}
		} else if len(c.RPrimal.Trace) != 0 {
			t.Fatal("inverse trace must cancel")
		}
		lpCheckLARTableauCatchUpMatrix(t, h, c, a)
		h.must(c.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARTableauCatchUpNonzeroRHSSourcePrefix(t *testing.T) {
	h := newLPPairHarness(t)
	c, _ := lpLARTableauCatchUpFixture(t, h, 2, 4, 1)
	h.must(c.RightSides.Data[0].Copy(h.m, h.pair("6", "0")))
	h.must(c.RightSides.Data[1].Copy(h.m, h.pair("10", "0")))
	p, q := new(big.Rat), new(big.Rat)
	for i := 0; i < 2; i++ {
		for k := range c.RA.Rows[i] {
			if c.RA.Rows[i][k].Column == 2 {
				text, err := h.m.String(&c.RA.Rows[i][k].Value)
				h.must(err)
				if i == 0 {
					p = mustLPRat(t, text)
				} else {
					q = mustLPRat(t, text)
				}
			}
		}
	}
	normalized := new(big.Rat).Quo(big.NewRat(6, 1), p)
	correctOther := new(big.Rat).Sub(big.NewRat(10, 1), new(big.Rat).Mul(q, normalized))
	if correctOther.Cmp(big.NewRat(10, 1)) == 0 {
		t.Fatal("nontrivial RHS oracle")
	}
	ok, err := c.CatchUpInLUTableau([]uint32{2, 0}, []int32{-1, 1, 0, -2})
	h.must(err)
	if !ok {
		t.Fatal("pivot")
	}
	h.check(&c.RightSides.Data[0], normalized, new(big.Rat))
	// Source normalizes the pivot RHS but omits nonpivot RHS elimination.
	h.check(&c.RightSides.Data[1], big.NewRat(10, 1), new(big.Rat))
	h.must(c.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARTableauCatchUpReachedPrefixes(t *testing.T) {
	for _, phase := range []string{"parent-stopped", "nil-primal", "primal-stopped", "short-target", "odd-first", "odd-late", "target-late", "short-basis", "bad-initial-basis"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			c, _ := lpLARTableauCatchUpFixture(t, h, 2, 4, 1)
			primal, basis := c.RPrimal, c.RBasis
			trace, target := []uint32{2, 0}, []int32{-1, 1, 0, -2}
			wantErr, wantBasic := error(ErrBounds), uint32(0)
			switch phase {
			case "parent-stopped":
				c.stopped = true
				wantErr = ErrExactVectorPending
			case "nil-primal":
				c.RPrimal = nil
			case "primal-stopped":
				c.RPrimal.stopped = true
				wantErr = ErrExactVectorPending
			case "short-target":
				target = target[:2]
			case "odd-first":
				trace = trace[:1]
			case "odd-late":
				trace = append(trace, 3)
				wantBasic = 2
			case "target-late":
				trace = append(trace, 4, 1)
				wantBasic = 2
			case "short-basis":
				c.RBasis = c.RBasis[:0]
			case "bad-initial-basis":
				lpSetLARCatchUpExactCell(t, h, c, 0, 0, "2")
				if sourceDebugEnabled {
					wantErr = ErrAssertion
				} else {
					wantErr = nil
					wantBasic = 2
				}
			}
			ok, err := c.CatchUpInLUTableau(trace, target)
			if err != wantErr || ok != (wantErr == nil) {
				t.Fatal("reached prefix", ok, err, wantErr)
			}
			if phase != "short-basis" && c.RBasis[0] != wantBasic {
				t.Fatal("basis prefix", c.RBasis, wantBasic)
			}
			if phase == "short-basis" && c.RHeading[2] != 0 {
				t.Fatal("entering heading publishes before failed basis access", c.RHeading)
			}
			c.RPrimal = primal
			c.RBasis = basis
			h.must(c.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARTableauCatchUpPhysicalOwnership(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, late := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1500; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%t/%t/%d", direct, late, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					c, _ := lpLARTableauCatchUpFixture(t, h, 2, 4, 1)
					column := uint32(2)
					trace, target := []uint32{2, 0}, []int32{-1, 1, 0, -2}
					if late {
						column = 3
						trace = []uint32{2, 0, 3, 1}
						target = []int32{-1, -2, 0, 1}
					}
					huge := strings.Repeat("1234567890", 20)
					lpSetLARCatchUpExactCell(t, h, c, 0, column, huge+"7/13")
					lpSetLARCatchUpExactCell(t, h, c, 1, column, huge+"3/17")
					h.budget.FailAfter(ordinal)
					ok, err := c.CatchUpInLUTableau(trace, target)
					h.budget.FailAfter(-1)
					if err == nil {
						if !ok {
							t.Fatal("nonsingular pivot returned false")
						}
						success, done = true, true
						want := []uint32{2, 1}
						if late {
							want = []uint32{2, 3}
						}
						lpCheckLARCatchUpBasis(t, c.RBasis, c.RNBasis, c.RHeading, want)
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						// Source exchanges metadata before allocating pivot arithmetic and
						// only reverses an ordinary false pivot, never a numerical error.
						if c.RBasis[0] != 2 || (!late && c.RBasis[1] != 1) {
							t.Fatal("basis prefix", c.RBasis)
						}
						if _, retry := c.CatchUpInLUTableau(nil, nil); retry != ErrExactVectorPending {
							t.Fatal("stopped retry", retry)
						}
					}
					h.must(c.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if !success || direct && failures == 0 {
				t.Fatal("physical campaign", direct, late, success, failures)
			}
		}
	}
}

func TestLARTableauCatchUpPreservesOlderComparison(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			c, _ := lpLARTableauCatchUpFixture(t, h, 2, 4, 1)
			lpSetLARCatchUpExactCell(t, h, c, 0, 2, strings.Repeat("1234567890", 20)+"7/13")
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
				_, err = c.CatchUpInLUTableau([]uint32{2, 0}, []int32{-1, 1, 0, -2})
				if err == nil || c.calls.comparison != 0 {
					t.Fatal("adopted older comparison", err)
				}
			}
			h.must(c.Close())
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

func TestLARTableauCatchUpReversePivotErrorPrefix(t *testing.T) {
	h := newLPPairHarness(t)
	c, _ := lpLARTableauCatchUpFixture(t, h, 2, 4, 1)
	for row := uint32(0); row < 2; row++ {
		lpSetLARCatchUpExactCell(t, h, c, row, 2, "0")
	}
	// The zero entering pivot does not reach RHS storage; the restored basic
	// pivot does. Keep owning values separately and restore them before Close.
	rhs := c.RightSides.Data
	c.RightSides.Data = nil
	c.RPrimal.StartTracingBasisChanges()
	ok, err := c.CatchUpInLUTableau([]uint32{2, 0}, []int32{-1, 1, 0, -2})
	c.RightSides.Data = rhs
	if ok || err != ErrBounds || !reflect.DeepEqual(c.RBasis, []uint32{0, 1}) || len(c.RPrimal.Trace) != 0 {
		t.Fatal("reverse pivot prefix", ok, err, c.RBasis, c.RPrimal.Trace)
	}
	if _, err = c.CatchUpInLUTableau(nil, nil); err != ErrExactVectorPending {
		t.Fatal("retry", err)
	}
	h.must(c.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARTableauCatchUpReversePhysicalOwnership(t *testing.T) {
	for _, direct := range []bool{false, true} {
		success, failures := false, 0
		for ordinal := int64(0); ordinal < 1500; ordinal++ {
			done := false
			t.Run(fmt.Sprintf("%t/%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				c, _ := lpLARTableauCatchUpFixture(t, h, 2, 4, 1)
				for row := uint32(0); row < 2; row++ {
					lpSetLARCatchUpExactCell(t, h, c, row, 2, "0")
				}
				h.must(c.RightSides.Data[0].Copy(h.m, h.pair(strings.Repeat("1234567890", 20)+"7/13", "0")))
				h.budget.FailAfter(ordinal)
				ok, err := c.CatchUpInLUTableau([]uint32{2, 0}, []int32{-1, 1, 0, -2})
				h.budget.FailAfter(-1)
				if ok || !reflect.DeepEqual(c.RBasis, []uint32{0, 1}) {
					t.Fatal("restored metadata precedes repivot arithmetic", ok, c.RBasis)
				}
				if err == nil {
					success, done = true, true
				} else {
					failures++
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					if _, err = c.CatchUpInLUTableau(nil, nil); err != ErrExactVectorPending {
						t.Fatal("retry", err)
					}
				}
				h.must(c.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
			if done {
				break
			}
		}
		if !success || direct && failures == 0 {
			t.Fatal("reverse physical campaign", direct, success, failures)
		}
	}
}

func TestLARTableauCatchUpReducedCostsIndependentModel(t *testing.T) {
	for world := 0; world < 80; world++ {
		h := newLPPairHarness(t)
		c, a := lpLARTableauCatchUpFixture(t, h, 2, 4, world)
		c.settings.SimplexStrategy = StrategyTableauCosts
		oldRow := make([]*big.Rat, 4)
		oldCosts := make([]*big.Rat, 4)
		for j := range oldRow {
			oldRow[j] = new(big.Rat)
			oldCosts[j] = big.NewRat(int64(world+j+1), 1)
			h.must(h.m.SetInt64(&c.RPrimal.D.Data[j], int64(world+j+1)))
		}
		for k := range c.RA.Rows[0] {
			cell := &c.RA.Rows[0][k]
			text, err := h.m.String(&cell.Value)
			h.must(err)
			oldRow[cell.Column] = mustLPRat(t, text)
		}
		ok, err := c.CatchUpInLUTableau([]uint32{2, 0}, []int32{-1, 1, 0, -2})
		h.must(err)
		if !ok {
			t.Fatal("pivot")
		}
		for j := range oldCosts {
			want := new(big.Rat)
			if j != 2 {
				want.Sub(oldCosts[j], new(big.Rat).Mul(oldCosts[2], new(big.Rat).Quo(oldRow[j], oldRow[2])))
			}
			lpSparseRationalCheck(t, h, &c.RPrimal.D.Data[j], want)
		}
		lpCheckLARTableauCatchUpMatrix(t, h, c, a)
		h.must(c.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}
