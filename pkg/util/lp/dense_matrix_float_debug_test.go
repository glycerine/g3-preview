//go:build g3debug

package lp

import (
	"math"
	"math/big"
	"math/rand"
	"reflect"
	"testing"
)

func lpDebugDense(t *testing.T, values [][]float64) *FloatDenseMatrix {
	t.Helper()
	cols := 0
	if len(values) > 0 {
		cols = len(values[0])
	}
	m, err := NewFloatDenseMatrix(uint32(len(values)), uint32(cols))
	lpStaticMust(t, err)
	for i, row := range values {
		for j, v := range row {
			lpStaticMust(t, m.SetElement(uint32(i), uint32(j), v))
		}
	}
	return m
}
func lpDebugDenseRows(t *testing.T, m *FloatDenseMatrix) [][]float64 {
	t.Helper()
	rows := make([][]float64, m.rows)
	for i := range rows {
		rows[i] = make([]float64, m.columns)
		for j := range rows[i] {
			v, err := m.Element(uint32(i), uint32(j))
			lpStaticMust(t, err)
			rows[i][j] = v
		}
	}
	return rows
}
func TestLPDebugDenseOriginalSource(t *testing.T) {
	// Source test_dense_matrix appears after test_lp_local's unconditional
	// return. Preserve it as a direct fixture, without registered-driver credit.
	d, err := NewFloatDenseMatrix(3, 2)
	lpStaticMust(t, err)
	lpStaticMust(t, d.SetElement(0, 0, 1))
	lpStaticMust(t, d.SetElement(1, 1, 2))
	lpStaticMust(t, d.SetElement(2, 0, 3))
	unit, err := NewFloatDenseMatrix(2, 2)
	lpStaticMust(t, err)
	// Source mistakenly sets d again, leaving unit zero; do not repair it.
	lpStaticMust(t, d.SetElement(0, 0, 1))
	lpStaticMust(t, d.SetElement(1, 1, 1))
	c, err := MultiplyFloatDebugMatrices(d, unit)
	lpStaticMust(t, err)
	if !reflect.DeepEqual(c.values, make([]float64, 6)) {
		t.Fatal(c.values)
	}
	perm := lpDebugDense(t, [][]float64{{0, 1, 0}, {1, 0, 0}, {0, 0, 1}})
	c1, err := MultiplyFloatDebugMatrices(perm, d)
	lpStaticMust(t, err)
	if !reflect.DeepEqual(lpDebugDenseRows(t, c1), [][]float64{{0, 1}, {1, 0}, {3, 0}}) {
		t.Fatal(c1.values)
	}
	p2 := lpDebugDense(t, [][]float64{{0, 1}, {1, 0}})
	c2, err := MultiplyFloatDebugMatrices(d, p2)
	lpStaticMust(t, err)
	if !reflect.DeepEqual(lpDebugDenseRows(t, c2), [][]float64{{0, 1}, {1, 0}, {0, 3}}) {
		t.Fatal(c2.values)
	}
}
func TestLPDebugDenseIndependentProductsAndApplications(t *testing.T) {
	r := rand.New(rand.NewSource(773))
	for world := 0; world < 100; world++ {
		rows, inner, cols := 1+r.Intn(7), 1+r.Intn(7), 1+r.Intn(7)
		a := make([][]float64, rows)
		b := make([][]float64, inner)
		for i := range a {
			a[i] = make([]float64, inner)
			for j := range a[i] {
				a[i][j] = float64(r.Intn(19) - 9)
			}
		}
		for i := range b {
			b[i] = make([]float64, cols)
			for j := range b[i] {
				b[i][j] = float64(r.Intn(19) - 9)
			}
		}
		got, err := MultiplyFloatDebugMatrices(lpDebugDense(t, a), lpDebugDense(t, b))
		lpStaticMust(t, err)
		want := make([][]float64, rows)
		for i := range want {
			want[i] = make([]float64, cols)
		}
		// Independent integer outer-product accumulation; all magnitudes
		// are exactly representable, so no tolerance hides product errors.
		for k := 0; k < inner; k++ {
			for i := 0; i < rows; i++ {
				for j := 0; j < cols; j++ {
					want[i][j] += float64(int64(a[i][k]) * int64(b[k][j]))
				}
			}
		}
		if !reflect.DeepEqual(lpDebugDenseRows(t, got), want) {
			t.Fatal(world, got.values, want)
		}
		square := make([][]float64, rows)
		w := make([]float64, rows+2)
		for i := range square {
			square[i] = make([]float64, rows)
			for j := range square[i] {
				square[i][j] = float64(r.Intn(19) - 9)
			}
			w[i] = float64(i - 2)
		}
		w[rows], w[rows+1] = 777, 888
		expected := append(lpFloatMatrixVector(square, w[:rows], false), 777, 888)
		m := lpDebugDense(t, square)
		lpStaticMust(t, m.ApplyLeft(w))
		if !reflect.DeepEqual(w, expected) {
			t.Fatal(world, w, expected)
		}
	}
}
func TestLPDebugDenseCopyAssignmentAndFlattenedDomains(t *testing.T) {
	a := lpDebugDense(t, [][]float64{{1, 2}, {3, 4}})
	b, err := CopyFloatDenseMatrix(a)
	lpStaticMust(t, err)
	lpStaticMust(t, a.SetElement(0, 0, 7))
	if b.values[0] != 1 {
		t.Fatal("copy aliases")
	}
	lpStaticMust(t, b.Assign(b))
	lpStaticMust(t, b.Assign(a))
	a.values[0] = 9
	if b.values[0] != 7 {
		t.Fatal("assignment aliases")
	}
	small := lpDebugDense(t, [][]float64{{11}})
	lpStaticMust(t, b.Assign(small))
	lpStaticMust(t, b.Assign(a))
	if b.rows != 2 || b.columns != 2 {
		t.Fatal("dimensions")
	}
	lpStaticMust(t, a.SetElement(0, 2, 17))
	if a.values[2] != 17 {
		t.Fatal("flattened spill")
	}
	if _, err := a.Element(2, 0); err != ErrBounds {
		t.Fatal(err)
	}
	if a.SetElement(2, 0, 1) != ErrBounds {
		t.Fatal("bounds")
	}
	wrapped, err := NewFloatDenseMatrix(65536, 65536)
	lpStaticMust(t, err)
	if len(wrapped.values) != 0 {
		t.Fatal("unsigned product")
	}
	if _, err = wrapped.Element(65536, 0); err != ErrBounds {
		t.Fatal(err)
	}
	broken := lpDebugDense(t, [][]float64{{1, 2}, {3, 4}})
	broken.values = broken.values[:3]
	partial, err := CopyFloatDenseMatrix(broken)
	if err != ErrBounds || !reflect.DeepEqual(partial.values, []float64{1, 2, 3, 0}) {
		t.Fatal(err, partial.values)
	}
	if err = b.Assign(broken); err != ErrBounds || b.values[0] != 1 || b.values[2] != 3 {
		t.Fatal(err, b.values)
	}
	if b.Assign(nil) != ErrBounds {
		t.Fatal("nil assignment")
	}
	if _, err = CopyFloatDenseMatrix(nil); err != ErrBounds {
		t.Fatal(err)
	}
}
func TestLPDebugDenseRectangularAndAliasedApplicationPrefixes(t *testing.T) {
	wide := lpDebugDense(t, [][]float64{{1, 2, 99}, {3, 4, 88}})
	w := []float64{5, 6, 7}
	lpStaticMust(t, wide.ApplyLeft(w))
	if !reflect.DeepEqual(w, []float64{17, 39, 7}) {
		t.Fatal(w)
	}
	tall := lpDebugDense(t, [][]float64{{1, 2}, {3, 4}, {5, 6}})
	w = []float64{1, 2, 3}
	if tall.ApplyLeft(w) != ErrBounds || !reflect.DeepEqual(w, []float64{1, 2, 3}) {
		t.Fatal(w)
	}
	square := lpDebugDense(t, [][]float64{{1, 2}, {3, 4}})
	if square.ApplyLeft([]float64{1}) != ErrBounds {
		t.Fatal("short vector")
	}
	lpStaticMust(t, square.ApplyLeft(square.values[:2]))
	if !reflect.DeepEqual(square.values, []float64{5, 11, 3, 4}) {
		t.Fatal(square.values)
	}
	empty, err := NewFloatDenseMatrix(0, 3)
	lpStaticMust(t, err)
	lpStaticMust(t, empty.ApplyLeft(nil))
}
func TestLPDebugDenseMatrixEqualityAndFailurePrefixes(t *testing.T) {
	for _, world := range []struct {
		x, y  float64
		equal bool
	}{{0, 1e-6, true}, {0, math.Nextafter(1e-6, math.Inf(1)), false}, {math.NaN(), 7, true}, {math.Inf(1), math.Inf(1), true}, {math.Inf(1), 0, false}, {math.Copysign(0, -1), 0, true}} {
		a, b := lpDebugDense(t, [][]float64{{world.x}}), lpDebugDense(t, [][]float64{{world.y}})
		equal, err := FloatDebugMatricesEqual(a, b)
		lpStaticMust(t, err)
		if equal != world.equal {
			t.Fatal(world, equal)
		}
	}
	a := lpDebugDense(t, [][]float64{{1, 2}, {3, 4}})
	b := lpDebugDense(t, [][]float64{{1}})
	if equal, err := FloatDebugMatricesEqual(a, b); err != nil || equal {
		t.Fatal(equal, err)
	}
	if _, err := MultiplyFloatDebugMatrices(a, b); err != ErrAssertion {
		t.Fatal(err)
	}
	b = lpDebugDense(t, [][]float64{{1, 2}, {3, 4}})
	b.values = b.values[:3]
	if equal, err := FloatDebugMatricesEqual(a, b); err != ErrBounds || equal {
		t.Fatal(equal, err)
	}
	if equal, err := FloatDebugMatricesEqual(b, a); err != ErrBounds || equal {
		t.Fatal(equal, err)
	}
	c, err := MultiplyFloatDebugMatrices(a, b)
	if err != ErrBounds || c.values[0] != 7 || c.values[1] != 0 {
		t.Fatal(err, c.values)
	}
	if _, err := MultiplyFloatDebugMatrices(b, a); err != ErrBounds {
		t.Fatal(err)
	}
	if _, err := MultiplyFloatDebugMatrices(nil, a); err != ErrBounds {
		t.Fatal(err)
	}
	if _, err := FloatDebugMatricesEqual(nil, a); err != ErrBounds {
		t.Fatal(err)
	}

}
func TestLPDebugDenseConcretePermutationAndStaticConsumers(t *testing.T) {
	a := NewFloatStaticMatrix(3, 2)
	for i := uint32(0); i < 3; i++ {
		for j := uint32(0); j < 2; j++ {
			lpStaticMust(t, a.Set(i, j, float64(2*i+j+1)))
		}
	}
	dense, err := CopyFloatDenseMatrix(a)
	lpStaticMust(t, err)
	p := NewFloatPermutation(3)
	lpStaticMust(t, p.Set(0, 1))
	lpStaticMust(t, p.Set(1, 0))
	got, err := MultiplyFloatDebugMatrices(p, dense)
	lpStaticMust(t, err)
	if !reflect.DeepEqual(lpDebugDenseRows(t, got), [][]float64{{3, 4}, {1, 2}, {5, 6}}) {
		t.Fatal(got.values)
	}
}
func TestLPDebugDenseActualLUIdentities(t *testing.T) {
	for world := 0; world < 40; world++ {
		n := 3 + world%5
		basis := make([]uint32, n)
		b := make([][]*big.Rat, n)
		for i := range basis {
			basis[i] = uint32((i + 1) % n)
			b[i] = make([]*big.Rat, n)
			for j := range b[i] {
				v := int64((i*3+j+world)%5 - 2)
				if i == j {
					v += int64(8 * n)
				}
				b[i][j] = new(big.Rat).SetInt64(v)
			}
		}
		for _, density := range []float64{math.Inf(1), 0} {
			a := lpFloatLUSource(t, b, basis)
			m, err := NewFloatLU(a, basis, lpFloatLUSettings(density))
			lpStaticMust(t, err)
			correct, err := m.DebugIsCorrect(basis)
			lpStaticMust(t, err)
			if !correct {
				t.Fatal(world, density, "source is_correct")
			}
			left, err := m.LeftDebugMatrix(basis)
			lpStaticMust(t, err)
			right, err := m.RightDebugMatrix()
			lpStaticMust(t, err)
			equal, err := FloatDebugMatricesEqual(left, right)
			lpStaticMust(t, err)
			if !equal {
				t.Fatal(world, density, "LU identity")
			}
			original, err := m.BasisDebugMatrix(basis)
			lpStaticMust(t, err)
			for i := range b {
				for j := range b[i] {
					value, err := original.Element(uint32(i), uint32(j))
					lpStaticMust(t, err)
					want, _ := b[i][j].Float64()
					if value != want {
						t.Fatal(world, i, j, value, want)
					}
				}
			}
			m.SetStatus(LUDegenerated)
			if correct, err = m.DebugIsCorrect(basis); err != nil || correct {
				t.Fatal(correct, err)
			}
			lpStaticMust(t, m.Close())
			if _, err = m.DebugIsCorrect(basis); err != ErrLUClosed {
				t.Fatal(err)
			}
			if _, err = m.LeftDebugMatrix(basis); err != ErrLUClosed {
				t.Fatal(err)
			}
		}
	}
}

type lpUnknownFloatDebugTail struct{ FloatTailMatrix }

func TestLPDebugDenseLUTailBindingsAndFailurePrefixes(t *testing.T) {
	basis := []uint32{0, 1, 2}
	b := [][]*big.Rat{{big.NewRat(5, 1), big.NewRat(1, 1), big.NewRat(0, 1)}, {big.NewRat(0, 1), big.NewRat(7, 1), big.NewRat(2, 1)}, {big.NewRat(1, 1), big.NewRat(0, 1), big.NewRat(9, 1)}}
	newLU := func() *FloatLU {
		m, err := NewFloatLU(lpFloatLUSource(t, b, basis), basis, lpFloatLUSettings(math.Inf(1)))
		lpStaticMust(t, err)
		return m
	}
	m := newLU()
	eta := NewFloatEtaMatrix(1, 3)
	eta.SetDiagonal(2)
	lpStaticMust(t, eta.Append(0, 3))
	row := NewFloatRowEtaMatrix(0, 1, 3)
	lpStaticMust(t, row.Append(2, 4))
	diag := NewFloatDiagonalMatrix(2, 5)
	permutation := NewFloatPermutation(3)
	lpStaticMust(t, permutation.Set(0, 1))
	lpStaticMust(t, permutation.Set(1, 0))
	for _, tail := range []FloatTailMatrix{eta, row, diag, permutation} {
		view, err := m.debugTail(tail)
		lpStaticMust(t, err)
		w1, w2 := []float64{2, 3, 4}, []float64{2, 3, 4}
		lpStaticMust(t, tail.ApplyLeft(w1))
		values := make([][]float64, 3)
		for i := range values {
			values[i] = make([]float64, 3)
			for j := range values[i] {
				value, err := view.Element(uint32(i), uint32(j))
				lpStaticMust(t, err)
				values[i][j] = value
			}
		}
		w2 = lpFloatMatrixVector(values, w2, false)
		for i := range w1 {
			if math.Abs(w1[i]-w2[i]) > 1e-12 {
				t.Fatal(tail, w1, w2)
			}
		}
	}
	if _, err := m.debugTail(lpUnknownFloatDebugTail{}); err != ErrBounds {
		t.Fatal(err)
	}
	if _, err := m.BasisDebugMatrix([]uint32{0, 1}); err != ErrAssertion {
		t.Fatal(err)
	}
	// Source sparse lookup of an absent external column returns zero.
	absent, err := m.BasisDebugMatrix([]uint32{0, math.MaxUint32, 2})
	lpStaticMust(t, err)
	if absent.values[1] != 0 {
		t.Fatal(absent.values)
	}
	savedRows := m.A.Rows
	m.A.Rows = m.A.Rows[:1]
	partial, err := m.BasisDebugMatrix(basis)
	if err != ErrBounds || partial.values[0] != 5 || partial.values[3] != 0 {
		t.Fatal(err, partial.values)
	}
	if _, err = m.DebugIsCorrect(basis); err != ErrBounds {
		t.Fatal(err)
	}
	m.A.Rows = savedRows
	savedR := m.R
	m.R = nil
	if _, err = m.RightDebugMatrix(); err != ErrBounds {
		t.Fatal(err)
	}
	if _, err = m.DebugIsCorrect(basis); err != ErrBounds {
		t.Fatal(err)
	}
	m.R = savedR
	savedTail := m.Tail
	m.Tail = []FloatTailMatrix{lpUnknownFloatDebugTail{}}
	if _, err = m.LeftDebugMatrix(basis); err != ErrBounds {
		t.Fatal(err)
	}
	m.Tail = []FloatTailMatrix{NewFloatPermutation(2)}
	if _, err = m.LeftDebugMatrix(basis); err != ErrAssertion {
		t.Fatal(err)
	}
	m.Tail = savedTail
	savedRow := m.U.RowPermutation.values[0]
	m.U.RowPermutation.values[0] = math.MaxUint32
	if _, err = m.RightDebugMatrix(); err != ErrBounds {
		t.Fatal(err)
	}
	m.U.RowPermutation.values[0] = savedRow
	savedU := m.U
	m.U = nil
	if _, err = m.BasisDebugMatrix(basis); err != ErrAssertion {
		t.Fatal(err)
	}
	m.U = savedU
	lpStaticMust(t, m.Close())
	if _, err = m.BasisDebugMatrix(basis); err != ErrLUClosed {
		t.Fatal(err)
	}
	if _, err = m.RightDebugMatrix(); err != ErrLUClosed {
		t.Fatal(err)
	}
}
