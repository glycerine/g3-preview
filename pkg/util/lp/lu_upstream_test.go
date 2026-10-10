package lp

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"testing"

	z3rand "github.com/glycerine/g3/pkg/util/random"
)

// Source: src/test/lp.cpp:318-510. The mathematical scenarios called after
// test_small_lu by --test_lu; diagnostic printing is not a numerical assertion.
func TestLPFloatLUUpstreamLargerScenarios(t *testing.T) {
	for _, scenario := range []string{"larger", "holes", "random"} {
		n, cols := 6, 12
		basis := []uint32{1, 3, 0, 4, 5, 6}
		type change struct{ entering, leaving uint32 }
		updates := []change{{9, 0}}
		var grid [][]int64
		switch scenario {
		case "larger":
			grid = make([][]int64, n)
			for i := range grid {
				grid[i] = make([]int64, cols)
				for j := 0; j < cols; j++ {
					grid[i][(j+i)%cols] = int64(j * j)
				}
			}
		case "random":
			g := z3rand.New(0)
			grid = make([][]int64, n)
			for i := range grid {
				grid[i] = make([]int64, cols)
				for j := range grid[i] {
					grid[i][j] = int64(g.Next() % 20)
				}
			}
			if !reflect.DeepEqual(grid[0], []int64{18, 19, 18, 17, 15, 17, 5, 5, 10, 12, 13, 0}) {
				t.Fatal("source default RNG/draw order", grid[0])
			}
			updates = []change{{8, 3}, {11, 0}}
		case "holes":
			n, cols = 8, 9
			basis = []uint32{0, 1, 2, 3, 4, 5, 6, 7}
			updates = []change{{8, 0}}
			grid = [][]int64{
				{1, 2, 3, 4, 5, 0, 0, 0, 99},
				{0, -6, 7, 8, 9, 0, 0, 0, 0},
				{0, 0, 10, 0, 0, 0, 0, 0, 0},
				{0, 0, 11, -12, 0, 0, 0, 0, 0},
				{0, 0, 13, 14, 15, 0, 0, 0, 0},
				{0, 0, 0, 0, 28, -18, 19, 25, 0},
				{0, 0, 0, 0, 0, 20, -21, 0, 0},
				{0, 0, 0, 0, 0, 22, 23, 24, 88},
			}
		}
		for _, density := range []float64{.7, 0, math.Inf(1)} {
			t.Run(fmt.Sprint(scenario, "/", density), func(t *testing.T) {
				a := NewFloatStaticMatrix(uint32(n), uint32(cols))
				for i, row := range grid {
					for j, v := range row {
						lpStaticMust(t, a.Set(uint32(i), uint32(j), float64(v)))
					}
				}
				active := append([]uint32(nil), basis...)
				s := lpFloatLUSettings(density)
				s.PivotTolerance = 1e-6
				var m *FloatLU
				lpStaticMust(t, InitFloatFactorization(&m, a, active, s))
				defer m.Close()
				w := NewIndexedFloatVector(uint32(n))
				x := make([]*big.Rat, n)
				for i := range x {
					x[i] = big.NewRat(int64(2*i-3), 11)
				}
				for step := 0; step <= len(updates); step++ {
					b := lpExactIdentity(n)
					for i := range b {
						for j, c := range active {
							b[i][j].SetInt64(grid[i][c])
						}
					}
					if m.GetStatus() != LUOK {
						t.Fatal("source nonsingular basis degenerated", step)
					}
					lpFloatLUCheckFactorization(t, m, b)
					lpFloatLUCheckSolutions(t, m, b, x)
					rhs := lpExactMatrixVector(b, x, true)
					y := make([]float64, n)
					indexed := NewIndexedFloatVector(uint32(n))
					for i := range y {
						y[i], _ = rhs[i].Float64()
						if y[i] != 0 {
							lpStaticMust(t, indexed.SetValue(y[i], uint32(i)))
						}
					}
					lpStaticMust(t, m.SolveYBWithErrorCheck(y, active))
					lpStaticMust(t, m.SolveYBWithErrorCheckIndexed(indexed, lpLUHeading(uint32(cols), active), active, s))
					for i := range y {
						lpFloatBlockNear(t, y[i], x[i])
						lpFloatBlockNear(t, indexed.Data[i], x[i])
					}
					if step == len(updates) {
						break
					}
					u := updates[step]
					pos := -1
					for j, c := range active {
						if c == u.leaving {
							pos = j
						}
					}
					if pos < 0 {
						t.Fatal("source leaving column absent")
					}
					lpStaticMust(t, m.PrepareEntering(u.entering, w))
					lpStaticMust(t, m.ReplaceColumn(0, w, uint32(pos)))
					active[pos] = u.entering
					if len(w.Index) != 0 {
						t.Fatal("update workspace not consumed")
					}
				}
				if m.RefactorCounter != uint32(len(updates)) {
					t.Fatal("source update count", m.RefactorCounter)
				}
			})
		}
	}
}
