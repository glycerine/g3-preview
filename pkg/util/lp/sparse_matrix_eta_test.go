package lp

import (
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func lpSparseEtaOwningCells(m *RationalSparseMatrix, eta *RationalEtaMatrix) []*numeral.Rational {
	var out []*numeral.Rational
	for i := range m.Rows {
		for j := range m.Rows[i] {
			out = append(out, &m.Rows[i][j].Value)
		}
	}
	for i := range m.Columns {
		for j := range m.Columns[i].Values {
			out = append(out, &m.Columns[i].Values[j].Value)
		}
	}
	if eta != nil {
		out = append(out, &eta.Diagonal)
		for i := range eta.Column.Data {
			out = append(out, &eta.Column.Data[i].Value)
		}
	}
	return out
}

func TestLPSparseEtaConstructionIndependentOracle(t *testing.T) {
	rng := rand.New(rand.NewPCG(1517, 817))
	for world := 0; world < 180; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			value := h.pair("0", "0")
			n := uint32(2 + rng.IntN(5))
			m := lpNewExactSparse(t, h, n)
			f := newFloatSparseMatrix(n)
			cells := map[IndexPair]*big.Rat{}
			floats := map[IndexPair]float64{}
			rowMap, colMap := make([]uint32, n), make([]uint32, n)
			for i := range rowMap {
				rowMap[i] = uint32(i)
				colMap[i] = uint32(i)
			}
			for k := 0; k < 7; k++ {
				a := rng.Uint32N(n)
				b := (a + 1 + rng.Uint32N(n-1)) % n
				h.must(m.SwapRows(a, b))
				lpStaticMust(t, f.SwapRows(a, b))
				rowMap[a], rowMap[b] = rowMap[b], rowMap[a]
				a = rng.Uint32N(n)
				b = (a + 1 + rng.Uint32N(n-1)) % n
				h.must(m.SwapColumns(a, b))
				lpStaticMust(t, f.SwapColumns(a, b))
				colMap[a], colMap[b] = colMap[b], colMap[a]
			}
			// Retain an independent ordered column list, rather than reading the matrix
			// to derive expected coefficient values or insertion order.
			type entry struct {
				logical uint32
				value   *big.Rat
			}
			columns := make([][]entry, n)
			j := rng.Uint32N(n)
			for _, off := range rng.Perm(int(n * n)) {
				i, c := uint32(off)/n, uint32(off)%n
				if i != c && rng.IntN(3) == 0 {
					continue
				}
				v := math.Ldexp(float64(rng.IntN(17)-8), rng.IntN(9)-4)
				if i == c {
					v = float64(1 + rng.IntN(9))
				}
				x := new(big.Rat).SetFloat64(v)
				h.must(h.m.SetString(&value.X, x.RatString()))
				h.must(m.AddNewElement(rowMap[i], colMap[c], &value.X))
				lpStaticMust(t, f.AddNewElement(rowMap[i], colMap[c], v))
				cells[IndexPair{rowMap[i], colMap[c]}] = x
				floats[IndexPair{rowMap[i], colMap[c]}] = v
				columns[c] = append(columns[c], entry{i, x})
			}
			unit := true
			diagonal := new(big.Rat)
			var expected []entry
			for _, v := range columns[j] {
				if v.logical > j || (v.logical == j && v.value.Cmp(big.NewRat(1, 1)) != 0) {
					unit = false
				}
				if v.logical == j {
					diagonal.Set(v.value)
				}
			}
			for _, v := range columns[j] {
				if v.logical > j {
					expected = append(expected, entry{v.logical, new(big.Rat).Neg(new(big.Rat).Quo(v.value, diagonal))})
				}
			}
			eta, ok, err := m.FillEtaMatrix(j)
			h.must(err)
			if !ok {
				t.Fatal("exact constructor")
			}
			feta, ok, err := f.FillEtaMatrix(j)
			lpStaticMust(t, err)
			if !ok {
				t.Fatal("floating constructor")
			}
			if unit {
				if eta != nil || feta != nil {
					t.Fatal("unit column allocated eta")
				}
			} else {
				if eta == nil || feta == nil || eta.ColumnIndex() != j || feta.RowCount() != n || len(eta.Column.Data) != len(expected) || len(feta.Column.Data) != len(expected) {
					t.Fatal("eta shape")
				}
				lpSparseRationalCheck(t, h, &eta.Diagonal, diagonal)
				for k, v := range expected {
					if eta.Column.Data[k].Index != v.logical || feta.Column.Data[k].Index != v.logical {
						t.Fatal("eta order")
					}
					lpSparseRationalCheck(t, h, &eta.Column.Data[k].Value, v.value)
					want, _ := v.value.Float64()
					if feta.Column.Data[k].Value != want {
						t.Fatal("floating coefficient")
					}
				}
				lpExactVectorNamedCensus(t, h, lpSparseEtaOwningCells(m, eta))
				h.must(eta.Close())
			}
			lpExactSparseCheck(t, h, m, cells)
			lpFloatSparseCheck(t, f, floats)
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPSparseEtaConstructionLiteralColumns(t *testing.T) {
	type entry struct {
		row   uint32
		value string
	}
	for _, tc := range []struct {
		name           string
		j              uint32
		entries        []entry
		unit, accepted bool
		diagonal       string
		rows           []uint32
		values         []string
	}{
		{name: "empty", j: 1, unit: true, accepted: true},
		{name: "upper only", j: 1, entries: []entry{{0, "99"}}, unit: true, accepted: true},
		{name: "unit with upper", j: 1, entries: []entry{{0, "17"}, {1, "1"}}, unit: true, accepted: true},
		{name: "duplicate units", j: 1, entries: []entry{{1, "1"}, {1, "1"}}, unit: true, accepted: true},
		{name: "diagonal last wins", j: 1, entries: []entry{{2, "6"}, {1, "2"}, {0, "19"}, {1, "3"}, {2, "0"}, {2, "-9"}}, accepted: true, diagonal: "3", rows: []uint32{2, 2, 2}, values: []string{"-2", "0", "3"}},
		{name: "zero diagonal early", j: 1, entries: []entry{{2, "6"}, {1, "0"}, {1, "2"}}, accepted: false},
		{name: "zero final diagonal", j: 1, entries: []entry{{1, "2"}, {2, "6"}, {1, "0"}}, accepted: false},
		{name: "explicit zero lower", j: 1, entries: []entry{{1, "1"}, {2, "0"}}, accepted: true, diagonal: "1", rows: []uint32{2}, values: []string{"0"}},
		{name: "negative diagonal", j: 1, entries: []entry{{2, "3"}, {1, "-2"}}, accepted: true, diagonal: "-2", rows: []uint32{2}, values: []string{"3/2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newLPPairHarness(t)
			v := h.pair("0", "0")
			m := lpNewExactSparse(t, h, 3)
			f := newFloatSparseMatrix(3)
			for _, e := range tc.entries {
				h.must(h.m.SetString(&v.X, e.value))
				h.must(m.AddNewElement(e.row, tc.j, &v.X))
				x, _ := new(big.Rat).SetString(e.value)
				xf, _ := x.Float64()
				lpStaticMust(t, f.AddNewElement(e.row, tc.j, xf))
			}
			eta, ok, err := m.FillEtaMatrix(tc.j)
			h.must(err)
			if ok != tc.accepted || (eta == nil) != (tc.unit || !tc.accepted) {
				t.Fatal("exact status", ok, eta)
			}
			feta, ok, err := f.FillEtaMatrix(tc.j)
			lpStaticMust(t, err)
			if ok != tc.accepted || (feta == nil) != (tc.unit || !tc.accepted) {
				t.Fatal("float status", ok, feta)
			}
			if eta != nil {
				lpSparseRationalCheck(t, h, &eta.Diagonal, mustLPRat(t, tc.diagonal))
				if len(eta.Column.Data) != len(tc.rows) || len(feta.Column.Data) != len(tc.rows) {
					t.Fatal("literal terms")
				}
				for k, row := range tc.rows {
					if eta.Column.Data[k].Index != row || feta.Column.Data[k].Index != row {
						t.Fatal("literal row")
					}
					want := mustLPRat(t, tc.values[k])
					lpSparseRationalCheck(t, h, &eta.Column.Data[k].Value, want)
					wf, _ := want.Float64()
					if feta.Column.Data[k].Value != wf {
						t.Fatal("literal float")
					}
				}
				h.must(eta.Close())
			}
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactSparseEtaConstructionAllocationFailureCensus(t *testing.T) {
	for _, campaign := range []string{"diagonal-first", "lower-first", "diagonal-first-direct", "lower-first-direct"} {
		t.Run(campaign, func(t *testing.T) {
			failures, successes, matrixStops, etaStops, divisionStops := 0, 0, 0, 0, 0
			for ordinal := int64(0); ordinal < 600; ordinal++ {
				done := false
				t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if strings.HasSuffix(campaign, "-direct") {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					m := lpNewExactSparse(t, h, 3)
					xs, ys := strings.Repeat("1234567890", 50)+"/7", strings.Repeat("9876543210", 60)+"/13"
					p := h.pair(xs, ys)
					x := mustLPRat(t, xs)
					y := mustLPRat(t, ys)
					order := []uint32{0, 1, 2}
					if strings.HasPrefix(campaign, "lower-first") {
						order = []uint32{1, 0, 2}
					}
					want := map[IndexPair]*big.Rat{}
					for _, i := range order {
						value, rat := &p.Y, y
						if i == 0 {
							value, rat = &p.X, x
						}
						h.must(m.AddNewElement(i, 0, value))
						want[IndexPair{i, 0}] = rat
					}
					h.budget.FailAfter(ordinal)
					eta, ok, err := m.FillEtaMatrix(0)
					h.budget.FailAfter(-1)
					if err == nil {
						if !ok || eta == nil {
							t.Fatal("construct status")
						}
						successes++
						done = true
						lpSparseRationalCheck(t, h, &eta.Diagonal, x)
						expected := new(big.Rat).Neg(new(big.Rat).Quo(y, x))
						if len(eta.Column.Data) != 2 {
							t.Fatal("successful column")
						}
						for k := range eta.Column.Data {
							lpSparseRationalCheck(t, h, &eta.Column.Data[k].Value, expected)
						}
					} else {
						if !lpExactPriorityPhysicalRefusal(err) || ok || eta == nil {
							t.Fatal("construct refusal", ok, err)
						}
						failures++
						if m.calls.pending() {
							matrixStops++
						}
						if eta.ready() != nil {
							etaStops++
						}
						if eta.Column.calls.division != 0 || eta.Column.calls.integerDivision.Invocation != 0 || eta.Column.calls.gcd.Invocation != 0 {
							divisionStops++
						}
						// Matrix expressions and eta children retain separate numerical receipts.
						if _, _, e := m.FillEtaMatrix(0); e != ErrExactVectorPending {
							t.Fatal("stopped constructor", e)
						}
						if _, e := m.ShortenActiveMatrix(0, nil); e != ErrExactVectorPending {
							t.Fatal("stopped shortening", e)
						}
						if m.calls.division != 0 || m.calls.integerDivision.Invocation != 0 || m.calls.gcd.Invocation != 0 {
							t.Fatal("matrix adopted eta division")
						}
					}
					// The temporary passed into a stopped eta copy stays alive until the child
					// is closed. Destroy the independently returned child before its caller.
					h.must(eta.Close())
					if err == nil {
						lpExactSparseCheck(t, h, m, want)
					}
					h.must(m.Close())
					lpExactVectorNamedCensus(t, h, nil)
					if h.m.RawInvocationState().Active != [3]bool{} || h.m.DivisionInvocationState().Active || h.m.ComparisonInvocationState().Active {
						t.Fatal("orphan eta invocation")
					}
				})
				if done {
					break
				}
			}
			if failures == 0 || successes != 1 || etaStops == 0 || divisionStops == 0 {
				t.Fatal("incomplete construction campaign", failures, successes, matrixStops, etaStops, divisionStops)
			}
		})
	}
}

func TestLPSparseEtaConstructionBoundsAndPartialResult(t *testing.T) {
	for _, phase := range []string{"logical column", "physical column", "unit scan", "fill scan"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair("2", "1")
			m := lpNewExactSparse(t, h, 3)
			f := newFloatSparseMatrix(3)
			h.must(m.AddNewElement(0, 0, &p.X))
			h.must(m.AddNewElement(1, 0, &p.Y))
			lpStaticMust(t, f.AddNewElement(0, 0, 2))
			lpStaticMust(t, f.AddNewElement(1, 0, 1))
			j := uint32(0)
			saved := m.Columns
			switch phase {
			case "logical column":
				j = 3
			case "physical column":
				m.Columns = nil
				f.Columns = nil
			case "unit scan":
				m.Columns[0].Values[0].Index = 3
				f.Columns[0].Values[0].Index = 3
			case "fill scan":
				m.Columns[0].Values[1].Index = 3
				f.Columns[0].Values[1].Index = 3
			}
			eta, ok, err := m.FillEtaMatrix(j)
			feta, fok, ferr := f.FillEtaMatrix(j)
			if err != ErrBounds || ferr != ErrBounds || ok || fok {
				t.Fatal("bounds status", err, ferr)
			}
			if phase == "fill scan" {
				if eta == nil || feta == nil {
					t.Fatal("missing published partial eta")
				}
				lpSparseRationalCheck(t, h, &eta.Diagonal, big.NewRat(2, 1))
				if feta.Diagonal != 2 {
					t.Fatal("partial diagonal")
				}
				h.must(eta.Close())
			} else if eta != nil || feta != nil {
				t.Fatal("eta published before nonunit column")
			}
			m.Columns = saved
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPSparseEtaConstructionToleranceAndIEEE(t *testing.T) {
	for _, diagonal := range []float64{0, math.Copysign(0, -1), 0.999e-12, -0.999e-12, 1e-12, -1e-12, math.Inf(1), math.Inf(-1), math.NaN()} {
		t.Run(fmt.Sprint(diagonal), func(t *testing.T) {
			m := newFloatSparseMatrix(2)
			lpStaticMust(t, m.AddNewElement(1, 0, 3))
			lpStaticMust(t, m.AddNewElement(0, 0, diagonal))
			eta, ok, err := m.FillEtaMatrix(0)
			lpStaticMust(t, err)
			accepted := !(diagonal < 1e-12 && diagonal > -1e-12)
			if ok != accepted || (eta != nil) != accepted {
				t.Fatal("strict diagonal threshold", ok, diagonal)
			}
			if accepted {
				expected := -3 / diagonal
				got := eta.Column.Data[0].Value
				if math.IsNaN(expected) {
					if !math.IsNaN(got) {
						t.Fatal("NaN divide")
					}
				} else if got != expected || (got == 0 && math.Signbit(got) != math.Signbit(expected)) {
					t.Fatal("IEEE divide", got, expected)
				}
			}
		})
	}
	// Exact traits accept tiny nonzero pivots independently of floating epsilon.
	h := newLPPairHarness(t)
	m := lpNewExactSparse(t, h, 2)
	p := h.pair("1/10000000000000000000000000000000000000000", "3")
	h.must(m.AddNewElement(1, 0, &p.Y))
	h.must(m.AddNewElement(0, 0, &p.X))
	eta, ok, err := m.FillEtaMatrix(0)
	h.must(err)
	if !ok || eta == nil {
		t.Fatal("exact tiny diagonal rejected")
	}
	lpSparseRationalCheck(t, h, &eta.Column.Data[0].Value, mustLPRat(t, "-30000000000000000000000000000000000000000"))
	h.must(eta.Close())
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func lpDenseDeterminantByPermutations(a [][]*big.Rat) *big.Rat {
	n := len(a)
	sum := new(big.Rat)
	perm := make([]int, n)
	used := make([]bool, n)
	var walk func(int)
	walk = func(k int) {
		if k < n {
			for j := 0; j < n; j++ {
				if !used[j] {
					used[j] = true
					perm[k] = j
					walk(k + 1)
					used[j] = false
				}
			}
			return
		}
		product := big.NewRat(1, 1)
		inversions := 0
		for i, j := range perm {
			product.Mul(product, a[i][j])
			for h := 0; h < i; h++ {
				if perm[h] > j {
					inversions++
				}
			}
		}
		if inversions%2 != 0 {
			product.Neg(product)
		}
		sum.Add(sum, product)
	}
	walk(0)
	return sum
}

func TestLPSparseEtaRepeatedPivotIndependentDeterminant(t *testing.T) {
	rng := rand.New(rand.NewPCG(11317, 825))
	for world := 0; world < 120; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			value := h.pair("0", "0")
			n := uint32(1 + rng.IntN(5))
			m := lpNewExactSparse(t, h, n)
			f := newFloatSparseMatrix(n)
			dense := make([][]*big.Rat, n)
			for i := uint32(0); i < n; i++ {
				dense[i] = make([]*big.Rat, n)
				for j := uint32(0); j < n; j++ {
					v := int64(rng.IntN(13) - 6)
					if i == j {
						v += 100
					}
					dense[i][j] = big.NewRat(v, 1)
					if v == 0 {
						continue
					}
					h.must(h.m.SetInt64(&value.X, v))
					h.must(m.AddNewElement(i, j, &value.X))
					lpStaticMust(t, f.AddNewElement(i, j, float64(v)))
				}
			}
			determinant := lpDenseDeterminantByPermutations(dense)
			pivotProduct := big.NewRat(1, 1)
			sign := int64(1)
			h.must(m.PrepareForFactorization())
			lpStaticMust(t, f.PrepareForFactorization())
			for k := uint32(0); k < n; k++ {
				i, j := uint32(77), uint32(88)
				ok, err := m.GetPivotForColumn(&i, &j, 10, k)
				h.must(err)
				if !ok {
					t.Fatal("nonsingular matrix lost pivot", k)
				}
				if i != k {
					h.must(m.SwapRows(k, i))
					lpStaticMust(t, f.SwapRows(k, i))
					dense[k], dense[i] = dense[i], dense[k]
					sign = -sign
				}
				if j != k {
					h.must(m.SwapColumns(k, j))
					lpStaticMust(t, f.SwapColumns(k, j))
					for r := range dense {
						dense[r][k], dense[r][j] = dense[r][j], dense[r][k]
					}
					sign = -sign
				}
				pivot := new(big.Rat).Set(dense[k][k])
				if pivot.Sign() == 0 {
					t.Fatal("selected zero pivot")
				}
				pivotProduct.Mul(pivotProduct, pivot)
				eta, ok, err := m.FillEtaMatrix(k)
				h.must(err)
				if !ok {
					t.Fatal("exact degenerate pivot", k)
				}
				feta, ok, err := f.FillEtaMatrix(k)
				lpStaticMust(t, err)
				if !ok {
					t.Fatal("floating degenerate pivot", k)
				}
				for r := k + 1; r < n; r++ {
					alpha := new(big.Rat).Neg(new(big.Rat).Quo(dense[r][k], pivot))
					for c := uint32(0); c < n; c++ {
						dense[r][c] = new(big.Rat).Add(dense[r][c], new(big.Rat).Mul(alpha, dense[k][c]))
					}
				}
				for c := uint32(0); c < n; c++ {
					dense[k][c] = new(big.Rat).Quo(dense[k][c], pivot)
				}
				if eta == nil {
					ok, err = m.ShortenActiveMatrix(k, nil)
				} else {
					ok, err = m.PivotWithEta(k, eta, nil)
				}
				h.must(err)
				if !ok {
					t.Fatal("exact shortening failed", k)
				}
				settings := &Settings{DropTolerance: 1e-14, ZeroTolerance: 1e-12}
				if feta == nil {
					ok, err = f.ShortenActiveMatrix(k, nil)
				} else {
					ok, err = f.PivotWithEta(k, feta, settings)
				}
				lpStaticMust(t, err)
				if !ok {
					t.Fatal("floating shortening failed", k)
				}
				for r := uint32(0); r < n; r++ {
					for c := uint32(0); c < n; c++ {
						x, e := m.Get(r, c)
						h.must(e)
						lpSparseRationalCheck(t, h, x, dense[r][c])
						y, e := f.Get(r, c)
						lpStaticMust(t, e)
						want, _ := dense[r][c].Float64()
						if math.Abs(y-want) > 1e-9*(1+math.Abs(want)) {
							t.Fatal("floating dense residual", k, r, c, y, want)
						}
					}
				}
				lpExactVectorNamedCensus(t, h, lpSparseEtaOwningCells(m, eta))
				if eta != nil {
					h.must(eta.Close())
				}
			}
			if sign < 0 {
				pivotProduct.Neg(pivotProduct)
			}
			if pivotProduct.Cmp(determinant) != 0 {
				t.Fatal("independent determinant mismatch", pivotProduct, determinant)
			}
			if m.active != 0 {
				t.Fatal("final active elements", m.active)
			}
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactSparseEtaConstructionOlderComparisonOwnership(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			m := lpNewExactSparse(t, h, 2)
			p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
			h.must(m.AddNewElement(0, 0, &p.X))
			h.must(m.AddNewElement(1, 0, &p.Y))
			var caller exactPriorityCalls
			before, err := caller.begin(h.m)
			h.must(err)
			h.budget.FailAfter(ordinal)
			_, err = h.m.Compare(&p.X, &p.Y)
			h.budget.FailAfter(-1)
			err = caller.after(h.m, before, err)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				matched = true
				// Scalar equality/copy can proceed with the older comparison stopped.
				// The later division reaches the manager's actual invalid-slot
				// refusal for its occupied numerical scratch state; preserve that
				// prefix and the caller's receipt rather than inventing an early
				// busy guard or treating this as valid-domain arithmetic.
				eta, ok, e := m.FillEtaMatrix(0)
				if eta == nil || ok || e != arena.ErrInvalidSlot || m.calls.comparison != 0 || eta.calls.comparison != 0 || eta.Column.calls.comparison != 0 {
					t.Fatal("borrowed comparison adopted", eta, ok, e, m.calls.comparison)
				}
				h.must(eta.Close())
				h.must(m.Close())
				if h.m.ComparisonInvocationState() != older {
					t.Fatal("closed older comparison")
				}
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no actual comparison refusal")
	}
}

func TestLPExactSparseEtaConstructionUnitWithoutCopies(t *testing.T) {
	h := lpNewDirectSparseHarness(t)
	m := lpNewExactSparse(t, h, 2)
	p := h.pair(strings.Repeat("1234567890", 100)+"/7", "1")
	h.must(m.AddNewElement(0, 1, &p.X))
	h.must(m.AddNewElement(1, 1, &p.Y))
	h.budget.FailAfter(0)
	eta, ok, err := m.FillEtaMatrix(1)
	h.budget.FailAfter(-1)
	h.must(err)
	if !ok || eta != nil || m.work != nil || m.calls.pending() {
		t.Fatal("unit scan copied upper coefficient")
	}
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactSparseEtaConstructionZeroDiagonalDestroysLargePrefix(t *testing.T) {
	for _, direct := range []bool{false, true} {
		t.Run(fmt.Sprint(direct), func(t *testing.T) {
			var h *lpPairHarness
			if direct {
				h = lpNewDirectSparseHarness(t)
			} else {
				h = newLPPairHarness(t)
			}
			m := lpNewExactSparse(t, h, 2)
			text := strings.Repeat("1234567890", 100) + "/7"
			p := h.pair(text, "0")
			h.must(m.AddNewElement(1, 0, &p.X))
			h.must(m.AddNewElement(0, 0, &p.Y))
			eta, ok, err := m.FillEtaMatrix(0)
			h.must(err)
			if eta != nil || ok || m.work != nil || m.calls.pending() {
				t.Fatal("zero pivot did not destroy partial eta")
			}
			// The large lower coefficient was already copied into eta before the zero
			// diagonal was reached. Its unpublished owner must have been fully destroyed.
			lpExactSparseCheck(t, h, m, map[IndexPair]*big.Rat{{1, 0}: mustLPRat(t, text), {0, 0}: new(big.Rat)})
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}
