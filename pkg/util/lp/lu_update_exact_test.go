package lp

import (
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"
)

func lpExactLUUpdateCheckSolutions(t *testing.T, h *lpPairHarness, r *RationalLU, p *PairLU, b [][]*big.Rat, x, y []*big.Rat) {
	t.Helper()
	column := lpExactLUVector(h, lpExactMatrixVector(b, x, false))
	defer func() { h.must(column.Close()) }()
	if r != nil {
		h.must(r.SolveBy(column.Data))
		lpExactLUCheckVector(t, h, column.Data, x)
	} else {
		pairs := NewIndexedPairVector(h.m, uint32(len(x)))
		rhsY := lpExactMatrixVector(b, y, false)
		for i := range pairs.Data {
			h.must(h.m.Copy(&pairs.Data[i].X, &column.Data[i]))
			h.must(h.m.SetString(&pairs.Data[i].Y, rhsY[i].RatString()))
		}
		h.must(p.SolveBy(pairs.Data))
		for i := range pairs.Data {
			h.check(&pairs.Data[i], x[i], y[i])
		}
		h.must(pairs.Close())
	}
	row := lpExactLUVector(h, lpExactMatrixVector(b, x, true))
	indexed := lpExactLUVector(h, lpExactMatrixVector(b, y, true))
	defer func() { h.must(row.Close()); h.must(indexed.Close()) }()
	if r != nil {
		h.must(r.SolveYB(row.Data))
		h.must(r.SolveYBIndexed(indexed))
	} else {
		h.must(p.SolveYB(row.Data))
		h.must(p.SolveYBIndexed(indexed))
	}
	lpExactLUCheckVector(t, h, row.Data, x)
	lpExactLUCheckVector(t, h, indexed.Data, y)
}

func TestLPExactLUUpdateDegenerationAndGuards(t *testing.T) {
	for _, pair := range []bool{false, true} {
		for _, phase := range []string{"empty column", "zero diagonal", "tiny diagonal", "leaving bounds", "foreign caller", "Q composition", "R composition", "closed", "counter wrap"} {
			t.Run(fmt.Sprint(pair, "/", phase), func(t *testing.T) {
				h := newLPPairHarness(t)
				b := lpExactIdentity(2)
				a := lpExactLUStatic(t, h, b, []uint32{0, 1})
				pivot := h.pair("888", "0")
				var r *RationalLU
				var p *PairLU
				var state *exactLUState
				var err error
				s := lpFloatLUSettings(math.Inf(1))
				s.DropTolerance = 1e200
				s.PivotTolerance = 1e200
				if pair {
					p, err = NewPairLU(a, []uint32{0, 1}, s)
					state = p.exactLUState
				} else {
					r, err = NewRationalLU(a, []uint32{0, 1}, s)
					state = r.exactLUState
				}
				h.must(err)
				w := NewIndexedRationalVector(h.m, 2)
				defer func() {
					if r != nil {
						h.must(r.Close())
					}
					if p != nil {
						h.must(p.Close())
					}
					h.must(w.Close())
				}()
				leaving := uint32(0)
				input := w
				switch phase {
				case "empty column":
					leaving = 1
				case "zero diagonal":
				case "tiny diagonal":
					h.must(h.m.SetString(&w.Data[0], "1/1"+strings.Repeat("0", 160)))
					w.Index = []uint32{0}
				case "leaving bounds":
					leaving = 2
				case "foreign caller":
					input = NewIndexedRationalVector(nil, 0)
				case "Q composition":
					state.q.values[1] = 2
					h.must(h.m.SetInt64(&w.Data[0], 1))
					w.Index = []uint32{0}
				case "R composition":
					state.r.values[1] = 2
					h.must(h.m.SetInt64(&w.Data[0], 1))
					w.Index = []uint32{0}
				case "closed":
					if pair {
						h.must(p.Close())
					} else {
						h.must(r.Close())
					}
				case "counter wrap":
					state.RefactorCounter = ^uint32(0)
					h.must(h.m.SetInt64(&w.Data[0], 2))
					w.Index = []uint32{0}
				}
				if pair {
					err = p.ReplaceColumn(&pivot.X, input, leaving)
				} else {
					err = r.ReplaceColumn(&pivot.X, input, leaving)
				}
				switch phase {
				case "closed":
					if err != ErrLUClosed {
						t.Fatal(err)
					}
				case "foreign caller":
					if err != ErrExactVectorOwner || state.RefactorCounter != 0 || state.updateWork != nil {
						t.Fatal("foreign caller prefix", err)
					}
				case "leaving bounds", "Q composition", "R composition":
					if err != ErrBounds || !state.stopped || state.RefactorCounter != 1 {
						t.Fatal(err, state.RefactorCounter)
					}
				case "empty column", "zero diagonal":
					h.must(err)
					if state.status != LUDegenerated || state.stopped || state.pendingRowEta != nil {
						t.Fatal("degeneration")
					}
					for i := range state.rowEtaWork.Data {
						lpSparseRationalCheck(t, h, &state.rowEtaWork.Data[i], new(big.Rat))
					}
				case "tiny diagonal":
					h.must(err)
					if state.status != LUOK {
						t.Fatal("precise tiny pivot dropped")
					}
					v, e := state.U.Get(0, 0)
					h.must(e)
					lpSparseRationalCheck(t, h, v, big.NewRat(1, 1))
				case "counter wrap":
					h.must(err)
					if state.RefactorCounter != 0 || state.NeedToRefactor() {
						t.Fatal("counter wrap")
					}
				}
				if r != nil {
					h.must(r.Close())
				}
				if p != nil {
					h.must(p.Close())
				}
				h.must(w.Close())
				lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
			})
		}
	}
}

func TestLPExactLUUpdateLeavesStoppedCallerOwned(t *testing.T) {
	for _, pair := range []bool{false, true} {
		matched := false
		for ordinal := int64(0); ordinal < 80 && !matched; ordinal++ {
			t.Run(fmt.Sprint(pair, "/", ordinal), func(t *testing.T) {
				h := lpNewDirectSparseHarness(t)
				a := lpExactLUStatic(t, h, lpExactIdentity(1), []uint32{0})
				v := h.pair("1"+strings.Repeat("0", 100)+"/7", "2"+strings.Repeat("0", 100)+"/13")
				var r *RationalLU
				var p *PairLU
				var state *exactLUState
				var err error
				if pair {
					p, err = NewPairLU(a, []uint32{0}, lpFloatLUSettings(math.Inf(1)))
					state = p.exactLUState
				} else {
					r, err = NewRationalLU(a, []uint32{0}, lpFloatLUSettings(math.Inf(1)))
					state = r.exactLUState
				}
				h.must(err)
				w := NewIndexedRationalVector(h.m, 1)
				defer func() {
					if r != nil {
						h.must(r.Close())
					}
					if p != nil {
						h.must(p.Close())
					}
					h.must(w.Close())
				}()
				h.must(w.SetValue(&v.X, 0))
				h.budget.FailAfter(ordinal)
				err = w.Add(0, &v.Y)
				h.budget.FailAfter(-1)
				if w.calls.pending() {
					matched = true
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					before := h.m.RawInvocationState()
					if pair {
						err = p.ReplaceColumn(&v.X, w, 0)
					} else {
						err = r.ReplaceColumn(&v.X, w, 0)
					}
					if err != ErrExactVectorPending || state.updateWork != nil || state.RefactorCounter != 0 {
						t.Fatal("stopped caller accepted", err)
					}
					if pair {
						h.must(p.Close())
					} else {
						h.must(r.Close())
					}
					if h.m.RawInvocationState() != before {
						t.Fatal("LU consumed caller receipt")
					}
				}
				if pair {
					h.must(p.Close())
				} else {
					h.must(r.Close())
				}
				h.must(w.Close())
				lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
			})
		}
		if !matched {
			t.Fatal("no actual caller receipt")
		}
	}
}

func TestLPExactLUUpdateReachedBounds(t *testing.T) {
	for _, phase := range []string{"wave start", "wave loop", "wave columns", "scan row map", "scan row extent", "scan column map", "scan workspace", "solve workspace", "solve row map", "solve row extent", "solve column map", "solve column extent", "eta scan", "eta solve", "bump workspace", "bump row map", "restore workspace", "diagonal read"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			a := lpExactLUStatic(t, h, lpExactIdentity(3), []uint32{0, 1, 2})
			value := h.pair("1", "0")
			m, err := NewRationalLU(a, []uint32{0, 1, 2}, lpFloatLUSettings(math.Inf(1)))
			h.must(err)
			// Constructor tie-breaking may permute an identity basis. These
			// physical-coordinate fault fixtures use its equivalent identity maps.
			m.U.RowPermutation.Init(3)
			m.U.ColumnPermutation.Init(3)
			m.Q.Init(3)
			m.R.Init(3)
			w := NewIndexedRationalVector(h.m, 3)
			defer func() { h.must(m.Close()); h.must(w.Close()) }()
			h.must(m.beginUpdate(&value.X, w))
			switch phase {
			case "wave start":
				m.RWave.values = nil
				err = m.calculateExactRWave(0, 2, &m.RWave.Permutation)
			case "wave loop":
				m.RWave.values = m.RWave.values[:1]
				err = m.calculateExactRWave(0, 2, &m.RWave.Permutation)
			case "wave columns":
				m.U.ColumnPermutation.values[2] = 3
				err = m.calculateExactRWave(0, 2, &m.RWave.Permutation)
			case "scan row map":
				m.U.RowPermutation.values = nil
				err = m.scanExactUpdateRow(2)
			case "scan row extent":
				m.U.Rows = m.U.Rows[:1]
				err = m.scanExactUpdateRow(2)
			case "scan column map":
				m.U.ColumnPermutation.values = nil
				err = m.scanExactUpdateRow(2)
			case "scan workspace":
				m.rowEtaWork.Data = m.rowEtaWork.Data[:1]
				err = m.scanExactUpdateRow(2)
			case "solve workspace":
				m.rowEtaWork.Data = nil
				err = m.solveExactUpdateRow(0, 2)
			case "solve row map":
				h.must(h.m.Copy(&m.rowEtaWork.Data[0], &value.X))
				m.U.RowPermutation.values = nil
				err = m.solveExactUpdateRow(0, 2)
			case "solve row extent":
				h.must(h.m.Copy(&m.rowEtaWork.Data[0], &value.X))
				m.U.Rows = nil
				err = m.solveExactUpdateRow(0, 2)
			case "solve column map":
				h.must(h.m.Copy(&m.rowEtaWork.Data[0], &value.X))
				m.U.ColumnPermutation.values = nil
				err = m.solveExactUpdateRow(0, 2)
			case "solve column extent":
				h.must(h.m.Copy(&m.rowEtaWork.Data[0], &value.X))
				h.must(m.U.AddNewElement(0, 2, &value.X))
				m.rowEtaWork.Data = m.rowEtaWork.Data[:2]
				err = m.solveExactUpdateRow(0, 2)
			case "eta scan":
				m.U.RowPermutation.values = nil
				err = m.makeExactUpdateRowEta(0, 2)
			case "eta solve":
				h.must(m.U.AddNewElement(2, 0, &value.X))
				m.U.RowPermutation.values[0] = 3
				err = m.makeExactUpdateRowEta(0, 2)
			case "bump workspace":
				m.rowEtaWork.Data = nil
				err = m.restoreExactUpdateBump(0, 2)
			case "bump row map":
				m.U.RowPermutation.values = nil
				err = m.restoreExactUpdateBump(0, 2)
			case "restore workspace":
				m.rowEtaWork.Index = []uint32{3}
				h.must(h.m.SetInt64(&m.rowEtaWork.Data[2], 2))
				err = m.restoreExactUpdateBump(0, 2)
			case "diagonal read":
				m.U.ColumnPermutation.reverse = nil
				err = m.restoreExactUpdateBump(2, 2)
			}
			if err != ErrBounds {
				t.Fatal(phase, err)
			}
			m.finishUpdate(&err)
			h.must(m.Close())
			h.must(w.Close())
			lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
		})
	}
}

func TestLPExactLUUpdateSparseColumnIntegration(t *testing.T) {
	for _, pair := range []bool{false, true} {
		t.Run(fmt.Sprint(pair), func(t *testing.T) {
			h := newLPPairHarness(t)
			const n = 21
			b := lpExactIdentity(n)
			basis := make([]uint32, n)
			for i := range basis {
				basis[i] = uint32(i)
			}
			a := lpExactLUStatic(t, h, b, basis)
			value := h.pair("1/7", "0")
			one := h.pair("1", "0")
			h.must(a.Set(0, n, &one.X))
			h.must(a.Set(n-1, n, &value.X))
			var r *RationalLU
			var p *PairLU
			var err error
			if pair {
				p, err = NewPairLU(a, basis, lpFloatLUSettings(math.Inf(1)))
			} else {
				r, err = NewRationalLU(a, basis, lpFloatLUSettings(math.Inf(1)))
			}
			h.must(err)
			d, w := NewIndexedRationalVector(h.m, n), NewIndexedRationalVector(h.m, n)
			defer func() {
				if r != nil {
					h.must(r.Close())
				}
				if p != nil {
					h.must(p.Close())
				}
				h.must(d.Close())
				h.must(w.Close())
			}()
			if pair {
				h.must(p.SolveBd(n, d, w))
			} else {
				h.must(r.SolveBd(n, d, w))
			}
			if w.Size() != 2 || !luSparseColumnRoute(w.Size(), d.DataSize(), 10) {
				t.Fatal("sparse column route")
			}
			if pair {
				h.must(p.ReplaceColumn(&one.X, w, 0))
			} else {
				h.must(r.ReplaceColumn(&one.X, w, 0))
			}
			b[n-1][0].SetFrac64(1, 7)
			x, y := make([]*big.Rat, n), make([]*big.Rat, n)
			for i := range x {
				x[i] = big.NewRat(int64(i-3), 11)
				y[i] = big.NewRat(int64(2*i-1), 17)
			}
			lpExactLUUpdateCheckSolutions(t, h, r, p, b, x, y)
			if pair {
				h.must(p.Close())
			} else {
				h.must(r.Close())
			}
			h.must(d.Close())
			h.must(w.Close())
			lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
		})
	}
}

func TestLPExactLUUpdateRowSolveLiteral(t *testing.T) {
	for _, phase := range []string{"cancel", "hidden cancel", "new tiny", "zero coefficient", "lower coefficient", "scan zero"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			a := lpExactLUStatic(t, h, lpExactIdentity(3), []uint32{0, 1, 2})
			one := h.pair("1", "0")
			m, err := NewRationalLU(a, []uint32{0, 1, 2}, lpFloatLUSettings(math.Inf(1)))
			h.must(err)
			input := NewIndexedRationalVector(h.m, 3)
			defer func() { h.must(m.Close()); h.must(input.Close()) }()
			m.U.RowPermutation.Init(3)
			m.U.ColumnPermutation.Init(3)
			h.must(m.beginUpdate(&one.X, input))
			w := m.rowEtaWork
			h.must(h.m.SetInt64(&w.Data[0], 2))
			w.Index = []uint32{0}
			coefficient := h.pair("3", "0")
			switch phase {
			case "scan zero":
				h.must(w.ClearAll())
				zero := h.pair("0", "0")
				h.must(m.U.AddNewElement(2, 0, &zero.X))
				h.must(m.scanExactUpdateRow(2))
				if len(w.Index) != 1 || w.Index[0] != 2 {
					t.Fatal("stored zero scanned", w.Index)
				}
				err = nil
				m.finishUpdate(&err)
				h.must(err)
				h.must(m.Close())
				h.must(input.Close())
				lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
				return
			case "cancel", "hidden cancel":
				h.must(h.m.SetInt64(&w.Data[2], -6))
				if phase == "cancel" {
					w.Index = append(w.Index, 2)
				}
			case "new tiny":
				h.must(h.m.SetString(&coefficient.X, "1/1"+strings.Repeat("0", 160)))
				m.Settings.DropTolerance = 1e200
			case "zero coefficient":
				h.must(h.m.SetInt64(&coefficient.X, 0))
			case "lower coefficient":
				h.must(w.ClearAll())
				h.must(h.m.SetInt64(&w.Data[1], 2))
				w.Index = []uint32{1}
				h.must(m.U.AddNewElement(1, 0, &coefficient.X))
				err = m.solveExactUpdateRow(1, 2)
				if err != sourceAssertionFailure() {
					t.Fatal(err)
				}
				if err == nil {
					lpSparseRationalCheck(t, h, &w.Data[0], big.NewRat(-6, 1))
				}
				m.finishUpdate(&err)
				h.must(m.Close())
				h.must(input.Close())
				lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
				return
			}
			h.must(m.U.AddNewElement(0, 2, &coefficient.X))
			h.must(m.solveExactUpdateRow(0, 2))
			want := new(big.Rat)
			if phase == "new tiny" {
				want.SetFrac(big.NewInt(2), new(big.Int).Exp(big.NewInt(10), big.NewInt(160), nil))
			}
			lpSparseRationalCheck(t, h, &w.Data[2], want)
			if (phase == "new tiny" && len(w.Index) != 2) || (phase != "new tiny" && len(w.Index) != 1) {
				t.Fatal("literal index", w.Index)
			}
			err = nil
			m.finishUpdate(&err)
			h.must(err)
			h.must(m.Close())
			h.must(input.Close())
			lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
		})
	}
}

func TestLPExactLUUpdateIndependentBasisOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	rng := rand.New(rand.NewPCG(193751, 779))
	for world := 0; world < 48; world++ {
		for _, pair := range []bool{false, true} {
			t.Run(fmt.Sprint(world, "/", pair), func(t *testing.T) {
				h := newLPPairHarness(t)
				n := 1 + world%6
				b := lpExactIdentity(n)
				order := rng.Perm(n)
				basis := make([]uint32, n)
				for i := range b {
					basis[i] = uint32(i)
					for j := range b[i] {
						v := int64(rng.IntN(5) - 2)
						if j == order[i] {
							v = int64(20*n + i)
						}
						b[i][j].SetFrac64(v, 7)
					}
				}
				a := NewRationalStaticMatrix(h.m, uint32(n), uint32(n+8))
				defer func() { h.must(a.Close()) }()
				value := h.pair("0", "0")
				pivot := h.pair("999", "0")
				source := make([][]*big.Rat, n)
				allColumns := make([]uint32, n+8)
				for j := range allColumns {
					allColumns[j] = uint32(j)
				}
				for i := range source {
					source[i] = make([]*big.Rat, n+8)
					for j := range source[i] {
						source[i][j] = new(big.Rat)
					}
					for j := range b[i] {
						source[i][j].Set(b[i][j])
						h.must(h.m.SetString(&value.X, b[i][j].RatString()))
						h.must(a.Set(uint32(i), uint32(j), &value.X))
					}
				}
				density := math.Inf(1)
				if (world/6)%2 == 0 {
					density = 0
				}
				s := lpFloatLUSettings(density)
				var r *RationalLU
				var p *PairLU
				var err error
				if pair {
					p, err = NewPairLU(a, basis, s)
				} else {
					r, err = NewRationalLU(a, basis, s)
				}
				h.must(err)
				w, d := NewIndexedRationalVector(h.m, uint32(n)), NewIndexedRationalVector(h.m, uint32(n))
				defer func() {
					if r != nil {
						h.must(r.Close())
					}
					if p != nil {
						h.must(p.Close())
					}
					h.must(w.Close())
					h.must(d.Close())
				}()
				for step := 0; step < 8; step++ {
					leaving := rng.IntN(n)
					entering := uint32(n + step)
					combination := make([]*big.Rat, n)
					for i := range combination {
						combination[i] = big.NewRat(int64(rng.IntN(7)-3), 13)
					}
					combination[leaving].SetFrac64(7, 11)
					column := lpExactMatrixVector(b, combination, false)
					for i := range column {
						source[i][entering].Set(column[i])
						h.must(h.m.SetString(&value.X, column[i].RatString()))
						h.must(a.Set(uint32(i), entering, &value.X))
					}
					var state *exactLUState
					if pair {
						h.must(p.SolveBd(entering, d, w))
						state = p.exactLUState
						h.must(p.ReplaceColumn(&pivot.X, w, uint32(leaving)))
					} else {
						h.must(r.SolveBd(entering, d, w))
						state = r.exactLUState
						h.must(r.ReplaceColumn(&pivot.X, w, uint32(leaving)))
					}
					lpExactLUCheckVector(t, h, d.Data, combination)
					if state.status != LUOK || state.RefactorCounter != uint32(step+1) || len(w.Index) != 0 || len(state.rowEtaWork.Index) != 0 || state.updateWork != nil || state.pendingRowEta != nil || state.pendingDiagonal != nil {
						t.Fatal("update completion", step, state.status)
					}
					for i := range b {
						b[i][leaving] = new(big.Rat).Set(column[i])
					}
					basis[leaving] = entering
					for i := range b {
						for j := range b {
							if i < j {
								continue
							}
							v, e := state.U.Get(uint32(i), uint32(j))
							h.must(e)
							want := new(big.Rat)
							if i == j {
								want.SetInt64(1)
							}
							lpSparseRationalCheck(t, h, v, want)
						}
					}
					x, y := make([]*big.Rat, n), make([]*big.Rat, n)
					for i := range x {
						x[i] = new(big.Rat).SetFrac(new(big.Int).Lsh(big.NewInt(int64(rng.IntN(17)-8)), 80), big.NewInt(17))
						y[i] = big.NewRat(int64(rng.IntN(19)-9), 19)
					}
					lpExactLUUpdateCheckSolutions(t, h, r, p, b, x, y)
				}
				if r != nil {
					h.must(r.Close())
				}
				if p != nil {
					h.must(p.Close())
				}
				h.must(w.Close())
				h.must(d.Close())
				lpExactLUCheckSource(t, h, a, source, allColumns)
				lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
				h.must(a.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}
