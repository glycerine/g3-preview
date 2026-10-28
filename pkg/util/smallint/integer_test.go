package smallint

// Source: src/util/s_integer.h:25-145; src/util/s_integer.cpp:22-70.

import (
	"errors"
	"math"
	"math/big"
	"math/rand/v2"
	"testing"
)

func TestIntegerConversionsPredicatesAndText(t *testing.T) {
	for _, v := range []int32{math.MinInt32, -17, -1, 0, 1, 2, math.MaxInt32} {
		n := New(v)
		if n.Int32() != v || n.Int64() != int64(v) || n.Uint32() != uint32(v) || n.Uint64() != uint64(int64(v)) || n.Hash() != uint32(v) {
			t.Fatal("conversion", v)
		}
		if n.IsZero() != (v == 0) || n.IsOne() != (v == 1) || n.IsMinusOne() != (v == -1) || n.IsNegative() != (v < 0) || n.IsPositive() != (v > 0) || n.IsNonnegative() != (v >= 0) || n.IsNonpositive() != (v <= 0) || n.IsEven() != (v&1 == 0) {
			t.Fatal("predicate", v)
		}
		if n.String() != big.NewInt(int64(v)).String() {
			t.Fatal("text", v, n.String())
		}
		if FromInt64(int64(v)+(1<<32)) != n || FromUint64(uint64(int64(v))) != n {
			t.Fatal("source narrowing", v)
		}
	}
}

func TestIntegerArithmeticAgainstWideOracle(t *testing.T) {
	random := rand.New(rand.NewChaCha8([32]byte{91, 4, 17}))
	for i := 0; i < 5000; i++ {
		a := int32(random.IntN(2000001) - 1000000)
		b := int32(random.IntN(2000001) - 1000000)
		c := int32(random.IntN(2001) - 1000)
		x, y, z := New(a), New(b), New(c)
		if x.Add(y).Int64() != int64(a)+int64(b) || x.Sub(y).Int64() != int64(a)-int64(b) || y.Mul(z).Int64() != int64(b)*int64(c) {
			t.Fatal("arithmetic", i)
		}
		if x.AddMul(y, z).Int64() != int64(a)+int64(b)*int64(c) || x.SubMul(y, z).Int64() != int64(a)-int64(b)*int64(c) {
			t.Fatal("fused arithmetic", i)
		}
		if b != 0 {
			if x.Quo(y) != New(a/b) || x.Rem(y) != New(a%b) {
				t.Fatal("division", i)
			}
			if b > 0 {
				want := a % b
				if want < 0 {
					want += b
				}
				if x.Mod(y) != New(want) {
					t.Fatal("mod", i)
				}
			}
		}
		if x.Compare(y) != compareInt32(a, b) {
			t.Fatal("compare", i)
		}
	}
}

func TestIntegerPowerAndNumberTheory(t *testing.T) {
	for _, tc := range []struct {
		base Integer
		exp  uint32
		want Integer
	}{{2, 0, 1}, {2, 10, 1024}, {-2, 5, -32}, {3, 7, 2187}} {
		got, err := Power(tc.base, tc.exp)
		if err != nil || got != tc.want {
			t.Fatal(tc, got, err)
		}
		if tc.exp <= math.MaxInt32 && tc.base.Expt(int32(tc.exp)) != tc.want {
			t.Fatal("expt", tc)
		}
	}
	if _, err := Power(2, 1<<31); !errors.Is(err, ErrPowerNonterminating) {
		t.Fatal(err)
	}
	for _, tc := range []struct{ a, b, gcd, lcm Integer }{
		{12, 18, 6, 36}, {-12, 18, 6, -36}, {17, 19, 1, 323},
	} {
		if GCD(tc.a, tc.b) != tc.gcd || LCM(tc.a, tc.b) != tc.lcm {
			t.Fatal(tc)
		}
	}
}

func TestIntegerWarmValueOperationsDoNotAllocate(t *testing.T) {
	a, b := New(17), New(5)
	if got := testing.AllocsPerRun(1000, func() {
		a = a.Add(b).Sub(b).AddMul(b, 2).SubMul(b, 2)
		if a.Compare(b) == 0 {
			t.Fatal("unexpected equality")
		}
	}); got != 0 {
		t.Fatal("allocations", got)
	}
}

func compareInt32(a, b int32) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
