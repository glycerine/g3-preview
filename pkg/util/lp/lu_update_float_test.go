package lp

import (
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"testing"
)

func TestLPFloatLUUpdateIndependentBasisOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	rng := rand.New(rand.NewPCG(753819, 619))
	rowTails, noBumps := 0, 0
	for world := 0; world < 64; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			n := 1 + world%8
			b := lpExactIdentity(n)
			basis := make([]uint32, n)
			order := rng.Perm(n)
			for i := range b {
				basis[i] = uint32(i)
				for j := range b[i] {
					v := int64(rng.IntN(5) - 2)
					if j == order[i] {
						v = int64(20*n + i)
					}
					b[i][j].SetFrac64(v, 4)
				}
			}
			a := lpFloatLUSourceColumns(t, b, basis, uint32(n+12))
			density := math.Inf(1)
			if (world/8)%2 == 0 {
				density = 0
			}
			s := lpFloatLUSettings(density)
			s.PivotTolerance = 1e-6
			m, err := NewFloatLU(a, basis, s)
			lpStaticMust(t, err)
			defer m.Close()
			w, d := NewIndexedFloatVector(uint32(n)), NewIndexedFloatVector(uint32(n))
			for step := 0; step < 12; step++ {
				leaving := rng.IntN(n)
				entering := uint32(n + step)
				combination := make([]*big.Rat, n)
				for i := range combination {
					combination[i] = big.NewRat(int64(rng.IntN(5)-2), 8)
				}
				combination[leaving].SetInt64(1)
				column := lpExactMatrixVector(b, combination, false)
				for i := range column {
					v, _ := column[i].Float64()
					lpStaticMust(t, a.Set(uint32(i), entering, v))
				}
				lpStaticMust(t, m.SolveBd(entering, d, w))
				for i := range combination {
					lpFloatBlockNear(t, d.Data[i], combination[i])
				}
				previous := len(m.Tail)
				check := d.Data[leaving]
				if step%3 == 0 {
					check = 0
				}
				lpStaticMust(t, m.ReplaceColumn(check, w, uint32(leaving)))
				if m.GetStatus() != LUOK {
					t.Fatal("degenerated nonsingular update", world, step, check)
				}
				if len(m.Tail) == previous+2 {
					rowTails++
				} else if len(m.Tail) == previous+1 {
					noBumps++
				} else {
					t.Fatal("tail count")
				}
				if m.RefactorCounter != uint32(step+1) || len(w.Index) != 0 || len(m.rowEtaWork.Index) != 0 {
					t.Fatal("counter/workspace")
				}
				for i := range b {
					b[i][leaving] = new(big.Rat).Set(column[i])
				}
				basis[leaving] = entering
				for i := 0; i < n; i++ {
					for j := 0; j < n; j++ {
						v, e := m.U.Get(uint32(i), uint32(j))
						lpStaticMust(t, e)
						if i > j && v != 0 {
							t.Fatal("not triangular", i, j, v)
						}
						if i == j {
							lpFloatBlockNear(t, v, big.NewRat(1, 1))
						}
					}
				}
				x := make([]*big.Rat, n)
				for i := range x {
					x[i] = big.NewRat(int64(rng.IntN(17)-8), 4)
				}
				lpFloatLUCheckSolutions(t, m, b, x)
				for i, v := range m.rowEtaWork.Data {
					if v != 0 {
						t.Fatal("stale row workspace", i, v)
					}
				}
			}
		})
	}
	if rowTails == 0 || noBumps == 0 {
		t.Fatal("missing update routes", rowTails, noBumps)
	}
}

func TestLPFloatLUUpdateUpstreamSmallLU(t *testing.T) {
	a := NewFloatStaticMatrix(3, 6)
	for _, e := range []struct {
		i, j uint32
		v    float64
	}{{0, 0, 1}, {0, 2, 3.9}, {2, 3, 11}, {0, 5, -3}, {1, 1, 4}, {1, 4, 7}, {2, 0, 1.8}, {2, 2, 5}, {2, 4, 2}, {2, 5, 8}} {
		lpStaticMust(t, a.Set(e.i, e.j, e.v))
	}
	basis := []uint32{0, 1, 3}
	s := lpFloatLUSettings(math.Inf(1))
	s.PivotTolerance = 1e-6
	m, err := NewFloatLU(a, basis, s)
	lpStaticMust(t, err)
	defer m.Close()
	w := NewIndexedFloatVector(3)
	for _, update := range []struct{ entering, leaving uint32 }{{2, 0}, {4, 3}, {5, 1}, {3, 2}} {
		position := -1
		for i, c := range basis {
			if c == update.leaving {
				position = i
			}
		}
		if position < 0 {
			t.Fatal("basis heading")
		}
		lpStaticMust(t, m.PrepareEntering(update.entering, w))
		lpStaticMust(t, m.ReplaceColumn(0, w, uint32(position)))
		if m.GetStatus() != LUOK {
			t.Fatal("source update degenerated")
		}
		basis[position] = update.entering
		b := lpExactIdentity(3)
		for i := range b {
			for j, c := range basis {
				v, e := a.Element(uint32(i), c)
				lpStaticMust(t, e)
				b[i][j].SetFloat64(v)
			}
		}
		lpFloatLUCheckSolutions(t, m, b, []*big.Rat{big.NewRat(3, 4), big.NewRat(-5, 2), big.NewRat(7, 8)})
	}
	// The source's subsequent dimension-extension scenario remains separate.
}

func lpFloatLUUpdateFixture(t *testing.T, n uint32) *FloatLU {
	t.Helper()
	s := lpFloatLUSettings(math.Inf(1))
	s.PivotTolerance = 1e-6
	m := &FloatLU{Dim: n, Q: NewFloatPermutation(n), R: NewFloatPermutation(n), RWave: NewFloatPermutation(n), U: newFloatSparseMatrix(n), Settings: s, rowEtaWork: NewIndexedFloatVector(n)}
	for i := uint32(0); i < n; i++ {
		lpStaticMust(t, m.U.AddNewElement(i, i, 1))
	}
	return m
}

func TestLPFloatLUUpdateDegenerationAndPublication(t *testing.T) {
	for _, phase := range []string{"empty column", "zero diagonal", "wrong pivot", "divide failure", "leaving bounds", "column bounds", "closed", "counter wrap"} {
		t.Run(phase, func(t *testing.T) {
			m := lpFloatLUUpdateFixture(t, 2)
			w := NewIndexedFloatVector(2)
			leaving := uint32(0)
			check := 0.0
			switch phase {
			case "empty column":
				leaving = 1
			case "zero diagonal":
			case "wrong pivot":
				lpStaticMust(t, w.SetValue(1, 0))
				lpStaticMust(t, w.SetValue(1, 1))
				check = 2
			case "divide failure":
				lpStaticMust(t, w.SetValue(2, 0))
				m.U.Rows[0][0].Other = 2
			case "leaving bounds":
				leaving = 2
			case "column bounds":
				m.U.ColumnPermutation.reverse[0] = 2
			case "closed":
				lpStaticMust(t, m.Close())
			case "counter wrap":
				m.RefactorCounter = ^uint32(0)
				lpStaticMust(t, w.SetValue(2, 0))
			}
			if phase == "divide failure" {
				// Test the last-row call directly to reach failure after tail
				// publication; earlier column replacement would inspect the link.
				err := m.finishUpdateBump(0, 0)
				if err != ErrBounds || len(m.Tail) != 1 {
					t.Fatal("tail-before-division", err, len(m.Tail))
				}
				return
			}
			err := m.ReplaceColumn(check, w, leaving)
			switch phase {
			case "closed":
				if err != ErrLUClosed {
					t.Fatal(err)
				}
			case "leaving bounds", "column bounds":
				if err != ErrBounds || m.RefactorCounter != 1 || !m.stopped {
					t.Fatal("failure prefix", err, m.RefactorCounter)
				}
				if e := m.ReplaceColumn(0, w, 0); e != ErrLUStopped {
					t.Fatal(e)
				}
			case "counter wrap":
				lpStaticMust(t, err)
				if m.RefactorCounter != 0 || m.NeedToRefactor() {
					t.Fatal("counter wrap")
				}
			default:
				lpStaticMust(t, err)
				if m.GetStatus() != LUDegenerated || m.RefactorCounter != 1 || m.stopped || len(m.Tail) != 0 {
					t.Fatal("degeneration", m.GetStatus(), len(m.Tail))
				}
				for _, v := range m.rowEtaWork.Data {
					if v != 0 {
						t.Fatal("degeneration cleanup")
					}
				}
			}
		})
	}
}

func TestLPFloatLUUpdateRowSolveLiteral(t *testing.T) {
	for _, phase := range []string{"zero coefficient", "cancel", "new small", "existing small", "underflow", "lower coefficient", "scan tiny", "NaN check"} {
		t.Run(phase, func(t *testing.T) {
			m := lpFloatLUUpdateFixture(t, 3)
			w := m.rowEtaWork
			lpStaticMust(t, w.SetValue(2, 0))
			coefficient := 3.0
			switch phase {
			case "zero coefficient":
				coefficient = 0
			case "cancel":
				lpStaticMust(t, w.SetValue(-6, 2))
			case "new small":
				coefficient = 1e-16
			case "existing small":
				coefficient = 1e-16
				lpStaticMust(t, w.SetValue(1e-16, 2))
			case "underflow":
				w.Data[0] = math.SmallestNonzeroFloat64
				coefficient = math.SmallestNonzeroFloat64
			case "lower coefficient":
				w.ClearAll()
				lpStaticMust(t, w.SetValue(2, 1))
				lpStaticMust(t, m.U.AddNewElement(1, 0, 3))
				if err := m.solveUpdateRow(1, 2); err != sourceAssertionFailure() {
					t.Fatal(err)
				}
				return
			case "scan tiny":
				w.ClearAll()
				lpStaticMust(t, m.U.AddNewElement(2, 0, 1e-16))
				if err := m.scanUpdateRow(2); err != sourceAssertionFailure() {
					t.Fatal(err)
				}
				return
			case "NaN check":
				w.ClearAll()
				eta, err := m.makeUpdateRowEta(0, 2, math.NaN())
				lpStaticMust(t, err)
				if eta != nil || m.status != LUDegenerated {
					t.Fatal("NaN pivot check")
				}
				return
			}
			lpStaticMust(t, m.U.AddNewElement(0, 2, coefficient))
			err := m.solveUpdateRow(0, 2)
			if phase == "underflow" {
				if err != sourceAssertionFailure() {
					t.Fatal(err)
				}
				return
			}
			lpStaticMust(t, err)
			if w.Data[2] != 0 {
				t.Fatal(phase, w.Data[2])
			}
		})
	}
}

func TestLPFloatLUUpdateReachedBounds(t *testing.T) {
	for _, phase := range []string{"wave start", "wave loop", "wave columns", "scan row map", "scan row extent", "scan column map", "scan workspace", "solve workspace", "solve row map", "solve row extent", "solve column map", "solve column extent", "bump workspace", "bump row map", "restore workspace", "diagonal read", "Q composition", "R composition", "public scan"} {
		t.Run(phase, func(t *testing.T) {
			m := lpFloatLUUpdateFixture(t, 3)
			w := m.rowEtaWork
			var err error
			switch phase {
			case "wave start":
				m.RWave.values = nil
				err = m.calculateRWave(0, 2)
			case "wave loop":
				m.RWave.values = m.RWave.values[:1]
				err = m.calculateRWave(0, 2)
			case "wave columns":
				m.U.ColumnPermutation.values[2] = 3
				err = m.calculateRWave(0, 2)
			case "scan row map":
				m.U.RowPermutation.values = nil
				err = m.scanUpdateRow(2)
			case "scan row extent":
				m.U.Rows = m.U.Rows[:1]
				err = m.scanUpdateRow(2)
			case "scan column map":
				m.U.ColumnPermutation.values = nil
				err = m.scanUpdateRow(2)
			case "scan workspace":
				w.Data = w.Data[:1]
				err = m.scanUpdateRow(2)
			case "solve workspace":
				w.Data = nil
				err = m.solveUpdateRow(0, 2)
			case "solve row map":
				w.Data[0] = 1
				m.U.RowPermutation.values = nil
				err = m.solveUpdateRow(0, 2)
			case "solve row extent":
				w.Data[0] = 1
				m.U.Rows = nil
				err = m.solveUpdateRow(0, 2)
			case "solve column map":
				w.Data[0] = 1
				m.U.ColumnPermutation.values = nil
				err = m.solveUpdateRow(0, 2)
			case "solve column extent":
				w.Data[0] = 1
				lpStaticMust(t, m.U.AddNewElement(0, 2, 3))
				w.Data = w.Data[:2]
				err = m.solveUpdateRow(0, 2)
			case "bump workspace":
				w.Data = nil
				err = m.finishUpdateBump(0, 2)
			case "bump row map":
				m.U.RowPermutation.values = nil
				err = m.finishUpdateBump(0, 2)
			case "restore workspace":
				w.Index = []uint32{3}
				w.Data[2] = 2
				err = m.finishUpdateBump(0, 2)
			case "diagonal read":
				m.U.ColumnPermutation.reverse = nil
				err = m.finishUpdateBump(2, 2)
			case "Q composition", "R composition", "public scan":
				input := NewIndexedFloatVector(3)
				lpStaticMust(t, input.SetValue(1, 0))
				if phase == "Q composition" {
					m.Q.values[2] = 3
				} else if phase == "R composition" {
					m.R.values[2] = 3
				} else {
					m.rowEtaWork.Data = nil
					lpStaticMust(t, input.SetValue(1, 2))
				}
				err = m.ReplaceColumn(0, input, 0)
				if !m.stopped || m.RefactorCounter != 1 {
					t.Fatal("public stop prefix")
				}
			}
			if err != ErrBounds {
				t.Fatal(phase, err)
			}
		})
	}
}

func TestLPFloatLUUpdateSparseColumnSolveIntegration(t *testing.T) {
	const n = uint32(241)
	a := NewFloatStaticMatrix(n, n+1)
	basis := make([]uint32, n)
	for i := uint32(0); i < n; i++ {
		basis[i] = i
		lpStaticMust(t, a.Set(i, i, 1))
	}
	lpStaticMust(t, a.Set(0, n, 1))
	lpStaticMust(t, a.Set(n-1, n, 0.25))
	s := lpFloatLUSettings(math.Inf(1))
	s.PivotTolerance = 1e-6
	m, err := NewFloatLU(a, basis, s)
	lpStaticMust(t, err)
	defer m.Close()
	d, w := NewIndexedFloatVector(n), NewIndexedFloatVector(n)
	lpStaticMust(t, m.SolveBd(n, d, w))
	if !luSparseColumnRoute(w.Size(), d.DataSize(), 120) || w.Size() != 2 {
		t.Fatal("indexed column route")
	}
	lpStaticMust(t, m.ReplaceColumn(d.Data[0], w, 0))
	if m.status != LUOK {
		t.Fatal(m.status)
	}
	x := make([]float64, n)
	x[0] = 3
	x[n-1] = -1.25
	lpStaticMust(t, m.SolveBy(x))
	y := NewIndexedFloatVector(n)
	lpStaticMust(t, y.SetValue(2.5, 0))
	lpStaticMust(t, y.SetValue(-2, n-1))
	lpStaticMust(t, m.SolveYBIndexed(y))
	for i := uint32(0); i < n; i++ {
		want := int64(0)
		if i == 0 {
			want = 3
		}
		if i == n-1 {
			want = -2
		}
		lpFloatBlockNear(t, x[i], big.NewRat(want, 1))
		lpFloatBlockNear(t, y.Data[i], big.NewRat(want, 1))
	}
}
