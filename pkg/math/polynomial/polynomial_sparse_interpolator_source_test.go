package polynomial

// Source: src/math/polynomial/polynomial.cpp:3072-3182.

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialSparseInterpolatorIndependentThreeSamples(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	serviceMust(t, m.modular.SetZPUint64(97))
	p := polynomialQueryTerms(t, m, []int32{1, 3, 2}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
	sk, e := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, e)
	s, e := NewPolynomialSparseInterpolator(full, sk)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.CloseChecked()); serviceMust(t, sk.CloseChecked()) })
	for _, x := range []int32{0, 1, 2} {
		in := unumeral.NewInt(x)
		q := polynomialQueryTerms(t, m, []int32{x*x + 3*x + 2}, [][]MonomialPower{nil})
		ok, e := s.Add(&in, q)
		serviceMust(t, e)
		if !ok {
			t.Fatal("independent sample support", x)
		}
	}
	ready, e := s.Ready()
	serviceMust(t, e)
	if !ready {
		t.Fatal("three actual skeleton powers")
	}
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	ok, e := s.Interpolate(out)
	serviceMust(t, e)
	if !ok {
		t.Fatal("actual Vandermonde solution")
	}
	equal, e := m.Equal(out.root.value, p)
	serviceMust(t, e)
	if !equal {
		t.Fatal("independent x^2+3x+2 interpolation")
	}
	polynomialSparsePhysical(t, s, out, []Polynomial{p})
	serviceMust(t, out.Close())
	serviceMust(t, s.CloseChecked())
	serviceMust(t, sk.CloseChecked())
}

func TestPolynomialSparseNativeMonomialGrowthAcross129Groups(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	serviceMust(t, m.modular.SetZPUint64(97))
	coefficients := make([]int32, 129)
	powers := make([][]MonomialPower, 129)
	for i := range coefficients {
		coefficients[i] = int32(i%47 + 1)
		powers[i] = []MonomialPower{{1, uint32(i + 1)}}
	}
	p := polynomialQueryTerms(t, m, coefficients, powers)
	sk, e := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, sk.CloseChecked()) })
	n, e := sk.NumEntries()
	serviceMust(t, e)
	if n != 129 {
		t.Fatal("independent one-power groups", n)
	}
	s, e := NewPolynomialSparseInterpolator(full, sk)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.CloseChecked()) })
	input := unumeral.NewInt(3)
	ok, e := s.Add(&input, p)
	serviceMust(t, e)
	if !ok {
		t.Fatal("constant-in-x0 sample rejected")
	}
	ready, e := s.Ready()
	serviceMust(t, e)
	if !ready {
		t.Fatal("one sample should fill each single-power group")
	}
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	for i := 0; i < 2; i++ {
		ok, e = s.Interpolate(out)
		serviceMust(t, e)
		if !ok {
			t.Fatal("single-variable coefficients were not reconstructed")
		}
		equal, e := m.Equal(out.root.value, p)
		serviceMust(t, e)
		if !equal {
			t.Fatal("native monomial/coefficient association", i)
		}
		if s.interpolation.active || s.interpolation.mons != nil {
			t.Fatal("native local span survived completed cleanup")
		}
		polynomialSparsePhysical(t, s, out, []Polynomial{p})
	}
}

func TestPolynomialSparseNativeNumeralGrowthAndReuse(t *testing.T) {
	full := polynomialPSCFullFixtureWithLimit(t, 1<<28)
	m := full.core
	serviceMust(t, m.modular.SetZPUint64(97))
	const size = 4097
	coefficients := make([]int32, size)
	powers := make([][]MonomialPower, size)
	for i := range coefficients {
		coefficients[i] = 1
		if i != 0 {
			powers[i] = []MonomialPower{{0, uint32(i)}}
		}
	}
	p := polynomialQueryTerms(t, m, coefficients, powers)
	sk, e := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, sk.CloseChecked()) })
	before, used := m.memory.Stats(), m.integer.Budget().Used()
	m.integer.Budget().FailAfter(1) // Only the actual class header may reserve.
	s, e := NewPolynomialSparseInterpolator(full, sk)
	serviceMust(t, e)
	t.Cleanup(func() { m.integer.Budget().FailAfter(-1); serviceMust(t, s.CloseChecked()) })
	if len(s.outputs) != size || m.memory.Stats() != before || m.integer.Budget().Used() != used+s.header {
		t.Fatal("native output construction charged container storage")
	}
	m.integer.Budget().FailAfter(0)
	first, last := unumeral.NewInt(13), unumeral.NewInt(29)
	serviceMust(t, s.copyField(s.outputs, 0, &first))
	serviceMust(t, s.copyField(s.outputs, size-1, &last))
	for pass := 0; pass < 2; pass++ {
		for i := int32(0); i < size; i++ {
			input := unumeral.NewInt(i)
			ok, e := s.Add(&input, m.zero)
			serviceMust(t, e)
			if !ok {
				t.Fatal("zero sample support", i)
			}
		}
		ready, e := s.Ready()
		serviceMust(t, e)
		if !ready || len(s.inputs) != size {
			t.Fatal("native sample count")
		}
		for i, input := range s.inputs {
			value, e := m.integer.Int64(&input)
			serviceMust(t, e)
			want := int64(i % 97)
			if want > 48 {
				want -= 97
			}
			if value != want {
				t.Fatal("native input association", i, value, want)
			}
		}
		capacity := cap(s.inputs)
		serviceMust(t, s.Reset())
		if len(s.inputs) != 0 || cap(s.inputs) != capacity || s.outputs[0] != first || s.outputs[size-1] != last {
			t.Fatal("source reset must retain outputs and reusable input capacity")
		}
		if m.memory.Stats() != before || m.integer.Budget().Used() != used+s.header {
			t.Fatal("native sample growth/reset charged Runtime")
		}
	}
	polynomialSparsePhysical(t, s, nil, []Polynomial{p})
	serviceMust(t, s.SourceFinalize())
	if s.inputs != nil || s.outputs != nil {
		t.Fatal("source destruction retained native slices")
	}
	serviceMust(t, s.CloseChecked())
	if m.memory.Stats() != before || m.integer.Budget().Used() != used {
		t.Fatal("native sparse owning cleanup changed actual preexisting graph")
	}
}

func TestPolynomialSparseNativeLocalsIndependentSeventeenSamples(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	serviceMust(t, m.modular.SetZPUint64(97))
	const size = 17
	coefficients := make([]int32, size)
	powers := make([][]MonomialPower, size)
	for i := range coefficients {
		coefficients[i] = int32(i + 1)
		if i != 0 {
			powers[i] = []MonomialPower{{0, uint32(i)}}
		}
	}
	p := polynomialQueryTerms(t, m, coefficients, powers)
	sk, e := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, sk.CloseChecked()) })
	s, e := NewPolynomialSparseInterpolator(full, sk)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.CloseChecked()) })
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	for pass := int32(0); pass < 2; pass++ {
		for x := int32(1); x <= size; x++ {
			point := x + pass*size
			value := int32(0)
			for i := size - 1; i >= 0; i-- {
				value = (value*point + coefficients[i]) % 97
			}
			if value > 48 {
				value -= 97
			}
			q := polynomialQueryTerms(t, m, []int32{value}, [][]MonomialPower{nil})
			input := unumeral.NewInt(point)
			ok, e := s.Add(&input, q)
			serviceMust(t, e)
			if !ok {
				t.Fatal("independent sample support", point)
			}
		}
		ok, e := s.Interpolate(out)
		serviceMust(t, e)
		if !ok {
			t.Fatal("distinct prime-field Vandermonde samples")
		}
		equal, e := m.Equal(out.root.value, p)
		serviceMust(t, e)
		if !equal {
			t.Fatal("independent seventeen coefficients", pass)
		}
		if s.interpolation.active {
			t.Fatal("native local values survived completed destruction")
		}
		polynomialSparsePhysical(t, s, out, []Polynomial{p})
		serviceMust(t, s.Reset())
	}
}
