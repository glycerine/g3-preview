package util

import (
	"errors"
	"math"
	"math/big"
	"math/rand"
	"testing"
)

func TestIntegerScalarStrictConversions(t *testing.T) {
	m, _ := integerManager(t)
	for _, v := range []int64{math.MinInt64, math.MinInt32, math.MinInt32 + 1, -1, 0, 1, math.MaxInt32 - 1, math.MaxInt32, math.MaxUint32 - 1, math.MaxUint32, math.MaxInt64} {
		var a Int
		if e := m.SetInt64(&a, v); e != nil {
			t.Fatal(e)
		}
		wi := v > math.MinInt32 && v < math.MaxInt32
		wu := v >= 0 && v < math.MaxUint32
		if got, e := m.IsInt32(&a); e != nil || got != wi {
			t.Fatal(v, got, e)
		}
		if got, e := m.IsUint32(&a); e != nil || got != wu {
			t.Fatal(v, got, e)
		}
		iv, ie := m.Int32(&a)
		uv, ue := m.Uint32(&a)
		if wi {
			if ie != nil || iv != int32(v) {
				t.Fatal(v, iv, ie)
			}
		} else if modularSourceDebug {
			if !errors.Is(ie, ErrIntegerAssertion) {
				t.Fatal(v, ie)
			}
		} else if ie != nil || iv != int32(v) {
			t.Fatal("release signed narrowing", v, iv, ie)
		}
		if wu {
			if ue != nil || uv != uint32(v) {
				t.Fatal(v, uv, ue)
			}
		} else if modularSourceDebug {
			if !errors.Is(ue, ErrIntegerAssertion) {
				t.Fatal(v, ue)
			}
		} else {
			want := uint32(v)
			if !a.IsSmall() && v < 0 {
				want = uint32(-v)
			}
			if ue != nil || uv != want {
				t.Fatal("release unsigned source magnitude/inline cast", v, uv, want, ue)
			}
		}
		m.Del(&a)
	}
	if !m.Precise() || m.Field() {
		t.Fatal("integer manager traits")
	}
	var huge Int
	if e := m.SetLimbs(&huge, []uint32{0, 0, 1}); e != nil {
		t.Fatal(e)
	}
	if ok, e := m.IsInt32(&huge); e != nil || ok {
		t.Fatal(ok, e)
	}
	if ok, e := m.IsUint32(&huge); e != nil || ok {
		t.Fatal(ok, e)
	}
	m.Del(&huge)
}

func TestIntegerScalarDecomposeParityAndMultiplicity(t *testing.T) {
	m, _ := integerManager(t)
	rng := rand.New(rand.NewSource(917))
	values := []*big.Int{big.NewInt(0), big.NewInt(-1), big.NewInt(math.MinInt32), big.NewInt(math.MaxInt32)}
	for i := 0; i < 100; i++ {
		z := new(big.Int).SetUint64(rng.Uint64())
		z.Lsh(z, uint(i%97))
		if i%2 != 0 {
			z.Neg(z)
		}
		values = append(values, z)
	}
	for _, z := range values {
		var a Int
		setIntegerOracle(t, m, &a, z)
		dst := make([]uint32, 4, 16)
		neg, digits, e := m.Decompose(&a, dst)
		if e != nil || neg != (z.Sign() < 0) {
			t.Fatal(z, neg, e)
		}
		decoded := new(big.Int)
		for i := len(digits) - 1; i >= 0; i-- {
			decoded.Lsh(decoded, 32)
			decoded.Add(decoded, new(big.Int).SetUint64(uint64(digits[i])))
		}
		if decoded.Cmp(new(big.Int).Abs(z)) != 0 || len(digits) == 0 {
			t.Fatal(z, digits)
		}
		if z.Sign() == 0 && (len(digits) != 1 || digits[0] != 0) {
			t.Fatal("inline zero has one digit", digits)
		}
		even, e := m.IsEven(&a)
		if e != nil || even != (z.Bit(0) == 0) {
			t.Fatal(z, even, e)
		}
		odd, e := m.IsOdd(&a)
		if e != nil || odd == even {
			t.Fatal(z, odd, e)
		}
		k, e := m.PowerOfTwoMultiple(&a)
		want := uint32(0)
		if z.Sign() != 0 {
			want = uint32(new(big.Int).Abs(z).TrailingZeroBits())
		}
		if e != nil || k != want {
			t.Fatal(z, k, want, e)
		}
		size, e := m.SizeInfo(&a)
		ws := uint32(1)
		if !a.IsSmall() {
			ws = uint32(len(digits)) + 1
		}
		if e != nil || size != ws {
			t.Fatal(z, size, ws, e)
		}
		m.Del(&a)
	}
}

func TestIntegerScalarFloatSourceRadix(t *testing.T) {
	m, _ := integerManager(t)
	for _, v := range []int32{math.MinInt32, -1, 0, 1, math.MaxInt32} {
		a := NewInt(v)
		if got, e := m.Float64(&a); e != nil || got != float64(v) {
			t.Fatal(v, got, e)
		}
	}
	for _, tc := range []struct {
		digits []uint32
		want   float64
	}{
		{[]uint32{0, 1}, 4294967295},
		{[]uint32{1, 1}, 4294967296},
		{[]uint32{0, 0, 1}, 4294967295.0 * 4294967295.0},
	} {
		var a Int
		if e := m.SetLimbs(&a, tc.digits); e != nil {
			t.Fatal(e)
		}
		got, e := m.Float64(&a)
		if e != nil || got != tc.want {
			t.Fatal(tc.digits, got, tc.want, e)
		}
		m.Neg(&a)
		got, e = m.Float64(&a)
		if e != nil || got != -tc.want {
			t.Fatal(tc.digits, got, e)
		}
		m.Del(&a)
	}
	// Source keeps multiplying its double radix after overflow. Zero high
	// interior limbs then contribute Inf*0 and propagate NaN.
	digits := make([]uint32, 36)
	digits[0], digits[35] = 1, 1
	var a Int
	if e := m.SetLimbs(&a, digits); e != nil {
		t.Fatal(e)
	}
	if got, e := m.Float64(&a); e != nil || !math.IsNaN(got) {
		t.Fatal(got, e)
	}
	m.Del(&a)
}

func TestIntegerScalarHashRepresentation(t *testing.T) {
	m, _ := integerManager(t)
	rng := rand.New(rand.NewSource(197))
	// Adaptation of rational.cpp::tst_hash's active assignment-history assertion:
	// check the underlying MPZ representation, not the unported rational wrapper.
	for i := 0; i <= 10000; i++ {
		v := int32(0)
		if i != 0 {
			v = int32(rng.Int31())
			if i%2 != 0 {
				v = -v
			}
		}
		a := NewInt(v)
		var b Int
		if e := m.SetString(&b, "10203939394995449949494394932929"); e != nil {
			t.Fatal(e)
		}
		if e := m.Copy(&b, &a); e != nil {
			t.Fatal(e)
		}
		ha, e := m.Hash(&a)
		if e != nil || ha != uint32(v) {
			t.Fatal(v, ha, e)
		}
		hb, e := m.Hash(&b)
		if e != nil || hb != ha {
			t.Fatal(v, ha, hb, e)
		}
		m.Del(&b)
	}
	for n := 1; n <= 17; n++ {
		digits := make([]uint32, n)
		bytes := make([]byte, 4*n)
		for i := range digits {
			digits[i] = rng.Uint32() | 0x80000000
			for j := 0; j < 4; j++ {
				bytes[4*i+j] = byte(digits[i] >> (8 * j))
			}
		}
		var a Int
		if e := m.SetLimbs(&a, digits); e != nil {
			t.Fatal(e)
		}
		want := digits[0]
		if n > 1 {
			want = String(string(bytes), 17)
		}
		for sign := 0; sign < 2; sign++ {
			got, e := m.Hash(&a)
			if e != nil || got != want {
				t.Fatal(n, got, want, e)
			}
			m.Neg(&a)
		}
		m.Del(&a)
	}
}

func TestIntegerScalarWarmAndGuards(t *testing.T) {
	m, budget := integerManager(t)
	var a Int
	if e := m.SetLimbs(&a, []uint32{0, 0x80000000, 3}); e != nil {
		t.Fatal(e)
	}
	dst := make([]uint32, 0, 4)
	before := budget.Used()
	budget.FailAfter(0)
	if allocs := testing.AllocsPerRun(100, func() {
		m.IsEven(&a)
		m.IsOdd(&a)
		m.PowerOfTwoMultiple(&a)
		m.SizeInfo(&a)
		m.Hash(&a)
		m.Float64(&a)
		m.IsInt32(&a)
		m.IsUint32(&a)
		m.Decompose(&a, dst)
	}); allocs != 0 {
		t.Fatal("scalar allocations", allocs)
	}
	if budget.Used() != before {
		t.Fatal("scalar budget mutation")
	}
	if _, e := m.Hash(nil); e == nil {
		t.Fatal("nil accepted")
	}
	old := []uint32{12, 34}
	if _, got, e := m.Decompose(nil, old); e == nil || len(got) != 2 || got[0] != 12 || got[1] != 34 {
		t.Fatal("failed decomposition changed buffer", got, e)
	}
	budget.FailAfter(-1)
	m.Del(&a)
	m.Close()
	if _, e := m.Float64(&a); e == nil {
		t.Fatal("closed accepted")
	}
}
