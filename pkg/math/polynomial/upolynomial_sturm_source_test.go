package polynomial

// Tests-first source routes from src/test/upolynomial.cpp.

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"math/big"
	"testing"
)

func upolySturmSequence(t *testing.T, m *UPolynomialManager) *UPolynomialSequence {
	t.Helper()
	seq, e := m.NewSequence()
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := seq.Close(); e != nil {
			t.Error(e)
		}
	})
	return seq
}
func upolySturmRow(t *testing.T, m *UPolynomialManager, s *UPolynomialSequence, i uint32) []int64 {
	t.Helper()
	n, e := s.RowSize(i)
	if e != nil {
		t.Fatal(e)
	}
	v, e := s.RowView(i)
	if e != nil {
		t.Fatal(e)
	}
	out := make([]int64, n)
	for j := uint32(0); j < n; j++ {
		var a unumeral.Int
		if e = m.ReadView(v, j, &a); e != nil {
			t.Fatal(e)
		}
		out[j], e = m.IntegerManager().Int64(&a)
		if e != nil {
			t.Fatal(e)
		}
		if e = m.IntegerManager().Del(&a); e != nil {
			t.Fatal(e)
		}
	}
	return out
}
func upolySturmExpectRow(t *testing.T, m *UPolynomialManager, s *UPolynomialSequence, i uint32, want []int64) {
	t.Helper()
	got := upolySturmRow(t, m, s, i)
	if len(got) != len(want) {
		t.Fatal(i, got, want)
	}
	for j := range want {
		if got[j] != want[j] {
			t.Fatal(i, got, want)
		}
	}
}

func TestUPolynomialSourceSixteenTransformENSURE(t *testing.T) {
	m, q, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{24, -50, 35, -10, 1}) // independent factor expansion
	out := upolySourceNew(t, m, nil)
	c, r1, r2 := unumeral.NewRational(), unumeral.NewRational(), unumeral.NewRational()
	if e := q.SetFraction(&c, 1, 3); e != nil {
		t.Fatal(e)
	}
	if e := q.SetInt64(&r1, 1); e != nil {
		t.Fatal(e)
	}
	if e := q.Sub(&r1, &c, &r1); e != nil {
		t.Fatal(e)
	}
	if e := q.SetInt64(&r2, 3); e != nil {
		t.Fatal(e)
	}
	if e := q.Sub(&r2, &c, &r2); e != nil {
		t.Fatal(e)
	}
	defer func() {
		for _, r := range []*unumeral.Rational{&r2, &r1, &c} {
			if e := q.Del(r); e != nil {
				t.Error(e)
			}
		}
	}()
	if e := m.TranslateRational(mustUPolyView(t, m, p), &c, out); e != nil {
		t.Fatal(e)
	}
	assertions := 0
	for _, n := range []int64{1, 2, 3, 4, -1, 5, -2} {
		point := unumeral.NewRational()
		if e := q.SetInt64(&point, n); e != nil {
			t.Fatal(e)
		}
		sign, e := m.EvalSignRational(mustUPolyView(t, m, out), &point)
		if e != nil || sign == 0 {
			t.Fatal(n, sign, e)
		}
		assertions++
		if e = q.Del(&point); e != nil {
			t.Fatal(e)
		}
	}
	for _, point := range []*unumeral.Rational{&r1, &r2} {
		sign, e := m.EvalSignRational(mustUPolyView(t, m, out), point)
		if e != nil || sign != 0 {
			t.Fatal(sign, e)
		}
		assertions++
	}
	if e := m.ReciprocalViewInPlace(mustUPolyView(t, m, p)); e != nil {
		t.Fatal(e)
	}
	for _, f := range []struct {
		n    int64
		d    uint64
		zero bool
	}{{1, 1, true}, {1, 2, true}, {1, 3, true}, {1, 4, true}, {2, 1, false}, {3, 1, false}, {4, 1, false}} {
		point := unumeral.NewRational()
		if e := q.SetFraction(&point, f.n, f.d); e != nil {
			t.Fatal(e)
		}
		sign, e := m.EvalSignRational(mustUPolyView(t, m, p), &point)
		if e != nil || (sign == 0) != f.zero {
			t.Fatal(f, sign, e)
		}
		assertions++
		if e = q.Del(&point); e != nil {
			t.Fatal(e)
		}
	}
	if assertions != 16 {
		t.Fatal(assertions)
	}
}

func TestUPolynomialSourceFiveHalfRootENSURE(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{3, -1, -12, 4})
	out := upolySourceNew(t, m, nil)
	expected := upolySourceNew(t, m, []int32{-24, -40, 16})
	yes, e := m.HasOneHalfRoot(mustUPolyView(t, m, p))
	if e != nil || !yes {
		t.Fatal(yes, e)
	}
	if e = m.RemoveOneHalfRoot(mustUPolyView(t, m, p), out); e != nil {
		t.Fatal(e)
	}
	yes, e = m.HasOneHalfRoot(mustUPolyView(t, m, out))
	if e != nil || yes {
		t.Fatal(yes, e)
	}
	equal, e := m.Equal(out, expected)
	if e != nil || !equal {
		t.Fatal(equal, e)
	}
	// Same source factors, expanded by actual dense arithmetic as explicit
	// multivariate conversion input adapter, not a fake is_univariate assertion.
	if e = m.SetInts(p, []unumeral.Int{unumeral.NewInt(1)}); e != nil {
		t.Fatal(e)
	}
	f := upolySourceNew(t, m, nil)
	for _, factor := range []struct {
		coeff []int32
		power int
	}{{[]int32{-1000000000, 0, 0, 0, 0, 1}, 3}, {[]int32{-10000000, 3}, 2}, {[]int32{-632, 10}, 2}} {
		nums := make([]unumeral.Int, len(factor.coeff))
		for i, v := range factor.coeff {
			nums[i] = unumeral.NewInt(v)
		}
		if e = m.SetInts(f, nums); e != nil {
			t.Fatal(e)
		}
		for i := 0; i < factor.power; i++ {
			if e = m.Mul(p, f, p); e != nil {
				t.Fatal(e)
			}
		}
	}
	yes, e = m.HasOneHalfRoot(mustUPolyView(t, m, p))
	if e != nil || yes {
		t.Fatal(yes, e)
	}
	if e = m.SetInts(p, []unumeral.Int{unumeral.NewInt(1)}); e != nil {
		t.Fatal(e)
	}
	for _, factor := range [][2]int32{{-2, 1}, {-4, 1}, {-8, 1}, {-16, 1}, {-32, 1}, {-64, 1}, {-1, 2}, {-1, 4}, {-1, 8}, {-1, 16}, {-1, 32}} {
		if e = m.SetInts(f, []unumeral.Int{unumeral.NewInt(factor[0]), unumeral.NewInt(factor[1])}); e != nil {
			t.Fatal(e)
		}
		if e = m.Mul(p, f, p); e != nil {
			t.Fatal(e)
		}
	}
	yes, e = m.HasOneHalfRoot(mustUPolyView(t, m, p))
	if e != nil || !yes {
		t.Fatal(yes, e)
	}
}

func TestUPolynomialSturmFourierAndRootIDIndependent(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{-1, 0, 1})
	s := upolySturmSequence(t, m)
	if e := m.SturmSequence(mustUPolyView(t, m, p), s); e != nil {
		t.Fatal(e)
	}
	n, e := s.Size()
	if e != nil || n != 3 {
		t.Fatal(n, e)
	}
	for i, row := range [][]int64{{-1, 0, 1}, {0, 2}, {1}} {
		upolySturmExpectRow(t, m, s, uint32(i), row)
	}
	minus, e := m.SignVariationsMinusInfinity(s)
	if e != nil || minus != 2 {
		t.Fatal(minus, e)
	}
	plus, e := m.SignVariationsPlusInfinity(s)
	if e != nil || plus != 0 {
		t.Fatal(plus, e)
	}
	zero, e := m.SignVariationsZero(s)
	if e != nil || zero != 1 {
		t.Fatal(zero, e)
	}
	for _, f := range []struct {
		point         int32
		variation, id uint32
	}{{-2, 2, 0}, {-1, 1, 1}, {0, 1, 1}, {1, 0, 2}, {2, 0, 2}} {
		b := unumeral.NewBinaryInt(f.point)
		got, e := m.SignVariationsBinary(s, &b)
		if e != nil || got != f.variation {
			t.Fatal(f, got, e)
		}
		id, e := m.GetRootID(mustUPolyView(t, m, p), &b)
		if e != nil || id != f.id {
			t.Fatal(f, id, e)
		}
	}
	derivative := upolySourceNew(t, m, []int32{0, 2})
	if e = m.SturmSequencePair(mustUPolyView(t, m, p), mustUPolyView(t, m, derivative), s); e != nil {
		t.Fatal(e)
	}
	for i, row := range [][]int64{{-1, 0, 1}, {0, 2}, {1}} {
		upolySturmExpectRow(t, m, s, uint32(i), row)
	}
	multiplier := upolySourceNew(t, m, []int32{0, 1})
	if e = m.SturmTarskiSequence(mustUPolyView(t, m, p), mustUPolyView(t, m, multiplier), s); e != nil {
		t.Fatal(e)
	}
	for i, row := range [][]int64{{-1, 0, 1}, {0, 0, 2}, {1}} {
		upolySturmExpectRow(t, m, s, uint32(i), row)
	}
	minus, e = m.SignVariationsMinusInfinity(s)
	if e != nil || minus != 0 {
		t.Fatal("opposite signs of x at the two roots cancel in Tarski query", minus, e)
	}
	plus, e = m.SignVariationsPlusInfinity(s)
	if e != nil || plus != 0 {
		t.Fatal(plus, e)
	}

	if e = m.FourierSequence(mustUPolyView(t, m, p), s); e != nil {
		t.Fatal(e)
	}
	for i, row := range [][]int64{{-1, 0, 1}, {0, 1}, {1}} {
		upolySturmExpectRow(t, m, s, uint32(i), row)
	}
	empty := upolySourceNew(t, m, nil)
	if e = m.FourierSequence(mustUPolyView(t, m, empty), s); e != nil {
		t.Fatal(e)
	}
	n, e = s.Size()
	if e != nil || n != 1 {
		t.Fatal(n, e)
	}
	upolySturmExpectRow(t, m, s, 0, nil)
	zero, e = m.SignVariationsZero(s)
	if e != nil || zero != 0 {
		t.Fatal(zero, e)
	}
}

func TestUPolynomialRegisteredSturmAndTarskiInputs(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	s := upolySturmSequence(t, m)
	p := upolySourceNew(t, m, []int32{8, 2, 8, 10, 10, 0, 1, 0, 1, 3, 7})
	if e := m.SturmSequence(mustUPolyView(t, m, p), s); e != nil {
		t.Fatal(e)
	}
	// Source tst_sturm and tst_sturm2 print only: zero source ENSURE/VERIFY.
	// Input rows must remain exact before the independent oracle suite added
	// for the full algorithm packet; no printed-data equality claim here.
	upolySturmExpectRow(t, m, s, 0, []int64{8, 2, 8, 10, 10, 0, 1, 0, 1, 3, 7})
	p2 := upolySourceNew(t, m, []int32{46225, 0, -5596840, 0, 13950764, 0, -7453176, 0, 1513334, 0, -141912, 0, 6476, 0, -136, 0, 1})
	q := upolySourceNew(t, m, []int32{576, 0, -960, 0, 352, 0, -40, 0, 1})
	if e := m.Mul(q, q, q); e != nil {
		t.Fatal(e)
	}
	if e := m.SturmTarskiSequence(mustUPolyView(t, m, p2), mustUPolyView(t, m, q), s); e != nil {
		t.Fatal(e)
	}
	upolySturmExpectRow(t, m, s, 0, []int64{46225, 0, -5596840, 0, 13950764, 0, -7453176, 0, 1513334, 0, -141912, 0, 6476, 0, -136, 0, 1})
}

// This oracle uses rational Euclidean division, independently of the source's
// signed pseudo-remainders and primitive integer normalization.
func upolyRatSturmOracle(input []int32) [][]*big.Rat {
	p := make([]*big.Rat, len(input))
	for i, a := range input {
		p[i] = new(big.Rat).SetInt64(int64(a))
	}
	d := make([]*big.Rat, len(p)-1)
	for i := range d {
		d[i] = new(big.Rat).Mul(p[i+1], new(big.Rat).SetInt64(int64(i+1)))
	}
	out := [][]*big.Rat{p, d}
	for len(d) > 0 {
		r := make([]*big.Rat, len(p))
		for i := range p {
			r[i] = new(big.Rat).Set(p[i])
		}
		for len(r) >= len(d) {
			shift := len(r) - len(d)
			f := new(big.Rat).Quo(r[len(r)-1], d[len(d)-1])
			for j := range d {
				r[j+shift].Sub(r[j+shift], new(big.Rat).Mul(f, d[j]))
			}
			for len(r) > 0 && r[len(r)-1].Sign() == 0 {
				r = r[:len(r)-1]
			}
		}
		if len(r) == 0 {
			break
		}
		for _, a := range r {
			a.Neg(a)
		}
		out = append(out, r)
		p, d = d, r
	}
	return out
}
func upolyRatVariations(rows [][]*big.Rat, point int64, infinity int) uint32 {
	var result uint32
	previous := 0
	for _, row := range rows {
		sign := 0
		if infinity != 0 {
			sign = row[len(row)-1].Sign()
			if infinity < 0 && (len(row)-1)%2 != 0 {
				sign = -sign
			}
		} else {
			value := new(big.Rat)
			x := new(big.Rat).SetInt64(point)
			for i := len(row) - 1; i >= 0; i-- {
				value.Mul(value, x)
				value.Add(value, row[i])
			}
			sign = value.Sign()
		}
		if sign == 0 {
			continue
		}
		if previous != 0 && previous != sign {
			result++
		}
		previous = sign
	}
	return result
}
func TestUPolynomialSturmRationalEuclideanOracle(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	seq := upolySturmSequence(t, m)
	for _, coefficients := range [][]int32{{-6, 11, -6, 1}, {4, 0, -5, 0, 1}, {1, 1, 0, 1}, {-2, 3, 0, -1, 1}, {4, -4, -3, 2, 1}} {
		p := upolySourceNew(t, m, coefficients)
		view := mustUPolyView(t, m, p)
		rows := upolyRatSturmOracle(coefficients)
		if e := m.SturmSequence(view, seq); e != nil {
			t.Fatal(coefficients, e)
		}
		minus, e := m.SignVariationsMinusInfinity(seq)
		if e != nil || minus != upolyRatVariations(rows, 0, -1) {
			t.Fatal(coefficients, minus, e)
		}
		plus, e := m.SignVariationsPlusInfinity(seq)
		if e != nil || plus != upolyRatVariations(rows, 0, 1) {
			t.Fatal(coefficients, plus, e)
		}
		for point := int64(-4); point <= 4; point++ {
			b := unumeral.NewBinaryInt(int32(point))
			want := upolyRatVariations(rows, point, 0)
			got, e := m.SignVariationsBinary(seq, &b)
			if e != nil || got != want {
				t.Fatal(coefficients, point, got, want, e)
			}
			id, e := m.GetRootID(view, &b)
			if e != nil || id != minus-want {
				t.Fatal(coefficients, point, id, minus-want, e)
			}
		}
	}
}
