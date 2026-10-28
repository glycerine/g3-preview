package checkedint

// Source: src/util/checked_int64.h:30-219.

import (
	"errors"
	"math"
	"math/big"
	rand "math/rand/v2"
	"testing"
)

func TestSourcePredicatesHashStringAndSubtractionHole(t *testing.T) {
	for _, value := range []int64{math.MinInt64, -2, -1, 0, 1, 2, math.MaxInt64} {
		n := New(value)
		if n.Value() != value || n.IsZero() != (value == 0) || n.IsPositive() != (value > 0) ||
			n.IsNegative() != (value < 0) || n.IsOne() != (value == 1) ||
			n.IsMinusOne() != (value == -1) || n.Hash() != uint32(value) ||
			n.String() != big.NewInt(value).String() || n.SourceIsEven() != (value == 1) {
			t.Fatal(value, n)
		}
	}
	n := New(0)
	if err := n.Subtract(New(math.MinInt64)); err != nil || n.Value() != math.MinInt64 {
		t.Fatal("source subtraction condition", n, err)
	}
}

func TestCheckedArithmeticBoundaries(t *testing.T) {
	for _, test := range []struct {
		value int64
		op    func(*Int64) error
	}{
		{math.MaxInt64, func(n *Int64) error { return n.Increment() }},
		{math.MinInt64, func(n *Int64) error { return n.Decrement() }},
		{math.MinInt64, func(n *Int64) error { return n.Negate() }},
	} {
		n := New(test.value)
		if err := test.op(&n); !errors.Is(err, ErrOverflow) || n.Value() != test.value {
			t.Fatal(test.value, n, err)
		}
	}
	if _, err := New(math.MinInt64).Abs(); !errors.Is(err, ErrOverflow) {
		t.Fatal(err)
	}
}

func TestAddSubtractAgainstSourceConditions(t *testing.T) {
	rng := rand.New(rand.NewChaCha8([32]byte{2, 7, 1, 8, 2, 8}))
	for i := 0; i < 10000; i++ {
		a, b := int64(rng.Uint64()), int64(rng.Uint64())
		wrappedAdd := int64(uint64(a) + uint64(b))
		addOverflow := a > 0 && b > 0 && wrappedAdd <= 0 || a < 0 && b < 0 && wrappedAdd >= 0
		n := New(a)
		err := n.Add(New(b))
		if addOverflow {
			if !errors.Is(err, ErrOverflow) || n.Value() != a {
				t.Fatal("add overflow", i, a, b, n, err)
			}
		} else if err != nil || n.Value() != wrappedAdd {
			t.Fatal("add", i, a, b, n, wrappedAdd, err)
		}

		wrappedSub := int64(uint64(a) - uint64(b))
		subOverflow := a > 0 && b < 0 && wrappedSub <= 0 || a < 0 && b > 0 && wrappedSub >= 0
		n = New(a)
		err = n.Subtract(New(b))
		if subOverflow {
			if !errors.Is(err, ErrOverflow) || n.Value() != a {
				t.Fatal("subtract overflow", i, a, b, n, err)
			}
		} else if err != nil || n.Value() != wrappedSub {
			t.Fatal("subtract", i, a, b, n, wrappedSub, err)
		}
	}
}

func TestMultiplyAgainstBigInt(t *testing.T) {
	rng := rand.New(rand.NewChaCha8([32]byte{3, 1, 4, 1, 5, 9}))
	min := big.NewInt(math.MinInt64)
	max := big.NewInt(math.MaxInt64)
	for i := 0; i < 10000; i++ {
		a, b := int64(rng.Uint64()), int64(rng.Uint64())
		n := New(a)
		err := n.Multiply(New(b))
		want := new(big.Int).Mul(big.NewInt(a), big.NewInt(b))
		fits := want.Cmp(min) >= 0 && want.Cmp(max) <= 0
		if fits {
			if err != nil || n.Value() != want.Int64() {
				t.Fatal(i, a, b, n, want, err)
			}
		} else if !errors.Is(err, ErrOverflow) || n.Value() != a {
			t.Fatal(i, a, b, n, want, err)
		}
	}
}

func TestWarmOperationsDoNotAllocate(t *testing.T) {
	if allocations := testing.AllocsPerRun(100, func() {
		n := New(7)
		_ = n.Add(New(5))
		_ = n.Subtract(New(3))
		_ = n.Multiply(New(-11))
		_, _ = n.Abs()
	}); allocations != 0 {
		t.Fatal(allocations)
	}
}
