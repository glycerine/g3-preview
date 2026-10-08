// Source: src/util/checked_int64.h:30-219.
package util

import (
	"errors"
	"math"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"
)

func TestCheckedInt64OriginalParityAndSubtractionHole(t *testing.T) {
	for _, v := range []int64{math.MinInt64, -2, -1, 0, 1, 2, 3, math.MaxInt64} {
		n := NewCheckedInt64(v)
		if n.Int64() != v || n.IsEven() != (v == 1) || n.Hash() != uint32(v) {
			t.Fatal("source scalar", v)
		}
	}
	n := NewCheckedInt64(0)
	serviceMust(t, n.SubAssign(NewCheckedInt64(math.MinInt64)))
	if n.Int64() != math.MinInt64 {
		t.Fatal("source zero-minus-min hole")
	}
	n = NewCheckedInt64(math.MaxInt64)
	if !errors.Is(n.AddAssign(NewCheckedInt64(1)), ErrCheckedInt64Overflow) || n.Int64() != math.MaxInt64 {
		t.Fatal("overflow publication")
	}
}

func TestCheckedInt64IndependentAddSubOracle(t *testing.T) {
	values := []int64{math.MinInt64, math.MinInt64 + 1, -1 << 32, math.MinInt32, -1, 0, 1, math.MaxInt32, 1 << 32, math.MaxInt64 - 1, math.MaxInt64}
	random := rand.New(rand.NewChaCha8([32]byte{51, 43, 89}))
	pairs := make([][2]int64, 0, 10200)
	for _, a := range values {
		for _, b := range values {
			pairs = append(pairs, [2]int64{a, b})
		}
	}
	for i := 0; i < 10000; i++ {
		pairs = append(pairs, [2]int64{int64(random.Uint64()), int64(random.Uint64())})
	}
	for _, pair := range pairs {
		a, b := pair[0], pair[1]
		for _, subtract := range []bool{false, true} {
			exact := new(big.Int).Add(big.NewInt(a), big.NewInt(b))
			if subtract {
				exact.Sub(big.NewInt(a), big.NewInt(b))
			}
			n := NewCheckedInt64(a)
			var e error
			if subtract {
				e = n.SubAssign(NewCheckedInt64(b))
			} else {
				e = n.AddAssign(NewCheckedInt64(b))
			}
			hole := subtract && a == 0 && b == math.MinInt64
			if exact.IsInt64() || hole {
				serviceMust(t, e)
				want := exact.Int64()
				if hole {
					want = math.MinInt64
				}
				if n.Int64() != want {
					t.Fatal(a, b, subtract, n.Int64(), want)
				}
			} else if !errors.Is(e, ErrCheckedInt64Overflow) || n.Int64() != a {
				t.Fatal("failed mutation", a, b, subtract, e, n)
			}
			u := NewUncheckedInt64(a)
			if subtract {
				serviceMust(t, u.SubAssign(NewUncheckedInt64(b)))
			} else {
				serviceMust(t, u.AddAssign(NewUncheckedInt64(b)))
			}
			wrapped := new(big.Int).Mod(exact, new(big.Int).Lsh(big.NewInt(1), 64)).Uint64()
			if uint64(u.Int64()) != wrapped {
				t.Fatal("unchecked wrap", a, b, subtract)
			}
		}
	}
	n := NewCheckedInt64(3)
	if !errors.Is(n.AddAssign(NewUncheckedInt64(2)), ErrCheckedInt64Profile) || n.Int64() != 3 {
		t.Fatal("profile crossing")
	}
}

func TestCheckedInt64UnaryCopiesComparisonsAndFunctors(t *testing.T) {
	for _, check := range []bool{false, true} {
		for _, v := range []int64{math.MinInt64, -3, -1, 0, 1, 3, math.MaxInt64} {
			n := CheckedInt64{v, check}
			if n.IsZero() != (v == 0) || n.IsPos() != (v > 0) || n.IsNeg() != (v < 0) || n.IsNonNeg() != (v >= 0) || n.IsNonPos() != (v <= 0) || n.IsOne() != (v == 1) || n.IsMinusOne() != (v == -1) {
				t.Fatal("predicates", v)
			}
			for _, other := range []int64{math.MinInt64, -1, 0, 1, math.MaxInt64} {
				m := CheckedInt64{other, check}
				if n.Eq(m) != (v == other) || n.Neq(m) != (v != other) || n.Lt(m) != (v < other) || n.Le(m) != (v <= other) || n.Gt(m) != (v > other) || n.Ge(m) != (v >= other) || (CheckedInt64EqProc{}).Equal(n, m) != n.Eq(m) {
					t.Fatal("comparison", v, other)
				}
			}
			if (CheckedInt64HashProc{}).Hash(n) != uint32(v) {
				t.Fatal("hash")
			}
			a, e := n.Abs()
			if check && v == math.MinInt64 {
				if !errors.Is(e, ErrCheckedInt64Overflow) {
					t.Fatal("abs guard")
				}
			} else {
				serviceMust(t, e)
				want := v
				if v < 0 {
					want = -v
				}
				if a.Int64() != want || a.Checks() != check {
					t.Fatal("abs", v, a)
				}
			}
			copy := n
			e = copy.Neg()
			if check && v == math.MinInt64 {
				if !errors.Is(e, ErrCheckedInt64Overflow) || copy != n {
					t.Fatal("neg guard")
				}
			} else {
				serviceMust(t, e)
				if copy.Int64() != -v {
					t.Fatal("neg")
				}
			}
			if n.Int64() != v {
				t.Fatal("copy mutated original")
			}
			copy = n
			old, e := copy.PostInc()
			if old != n {
				t.Fatal("post increment old")
			}
			if check && v == math.MaxInt64 {
				if !errors.Is(e, ErrCheckedInt64Overflow) || copy != n {
					t.Fatal("increment guard")
				}
			} else {
				serviceMust(t, e)
				if copy.Int64() != v+1 {
					t.Fatal("increment")
				}
			}
			copy = n
			old, e = copy.PostDec()
			if old != n {
				t.Fatal("post decrement old")
			}
			if check && v == math.MinInt64 {
				if !errors.Is(e, ErrCheckedInt64Overflow) || copy != n {
					t.Fatal("decrement guard")
				}
			} else {
				serviceMust(t, e)
				if copy.Int64() != v-1 {
					t.Fatal("decrement")
				}
			}
		}
		if CheckedInt64Zero(check).Int64() != 0 || CheckedInt64One(check).Int64() != 1 || CheckedInt64MinusOne(check).Int64() != -1 {
			t.Fatal("constants")
		}
	}
	a, b := NewCheckedInt64(7), NewCheckedInt64(3)
	sum, e := CheckedInt64Add(a, b)
	serviceMust(t, e)
	difference, e := CheckedInt64Sub(a, b)
	serviceMust(t, e)
	neg, e := CheckedInt64Neg(a)
	serviceMust(t, e)
	abs, e := CheckedInt64Abs(neg)
	serviceMust(t, e)
	if sum.Int64() != 10 || difference.Int64() != 4 || neg.Int64() != -7 || abs != a {
		t.Fatal("value operators")
	}
}

func TestCheckedInt64MultiplyOriginalRationalFallback(t *testing.T) {
	s, _ := serviceNew(t)
	values := []int64{math.MinInt64, math.MinInt64 + 1, -1 << 32, math.MinInt32, math.MinInt32 + 1, -1, 0, 1, math.MaxInt32, 1 << 32, math.MaxInt64}
	for _, a := range values {
		for _, b := range values {
			exact := new(big.Int).Mul(big.NewInt(a), big.NewInt(b))
			n := NewCheckedInt64(a)
			var local CheckedInt64MulScope
			e := n.MulAssign(NewCheckedInt64(b), s, &local)
			if exact.IsInt64() {
				serviceMust(t, e)
				if n.Int64() != exact.Int64() {
					t.Fatal("multiply", a, b, n, exact)
				}
			} else {
				if !errors.Is(e, ErrCheckedInt64Overflow) || n.Int64() != a {
					t.Fatal("multiply guard", a, b, e, n)
				}
			}
			if local.self != nil {
				serviceMust(t, local.Cleanup())
			}
			fast := math.MinInt32 < a && a <= math.MaxInt32 && math.MinInt32 < b && b <= math.MaxInt32
			if (local.self == nil) != fast {
				t.Fatal("strict original fast guard", a, b)
			}
			u := NewUncheckedInt64(a)
			serviceMust(t, u.MulAssign(NewUncheckedInt64(b), nil, nil))
			wrapped := new(big.Int).Mod(exact, new(big.Int).Lsh(big.NewInt(1), 64)).Uint64()
			if uint64(u.Int64()) != wrapped {
				t.Fatal("unchecked product", a, b)
			}
		}
	}
	// Exact INT_MIN is excluded from the source fast path, even times zero.
	n := NewCheckedInt64(math.MinInt32)
	if !errors.Is(n.MulAssign(NewCheckedInt64(0), nil, nil), ErrParamsPrecondition) {
		t.Fatal("INT_MIN fast shortcut")
	}
	n = NewCheckedInt64(math.MinInt32 + 1)
	serviceMust(t, n.MulAssign(NewCheckedInt64(math.MaxInt32), nil, nil))
	square := NewCheckedInt64(7)
	serviceMust(t, square.MulAssign(square, nil, nil))
	if square.Int64() != 49 {
		t.Fatal("alias")
	}
	a, b := NewCheckedInt64(1<<40), NewCheckedInt64(1<<10)
	var local CheckedInt64MulScope
	result, e := CheckedInt64Mul(a, b, s, &local)
	serviceMust(t, e)
	if result.Int64() != 1<<50 || a.Int64() != 1<<40 {
		t.Fatal("value multiply")
	}
	serviceMust(t, local.Cleanup())
}

func TestCheckedInt64EveryColdRationalFallbackReservation(t *testing.T) {
	for _, pair := range [][2]int64{{math.MinInt32, 0}, {1 << 40, 1 << 10}, {math.MinInt64, -1}, {math.MaxInt64, 2}} {
		completed := false
		for ordinal := int64(0); ordinal < 100; ordinal++ {
			s, b := serviceNew(t)
			n := NewCheckedInt64(pair[0])
			other := NewCheckedInt64(pair[1])
			var local CheckedInt64MulScope
			b.FailAfter(ordinal)
			e := n.MulAssign(other, s, &local)
			facadeAudit(t, s, &local.left, &local.right, &local.product)
			actual := s.HeaderBytes + s.CacheBytes + s.Manager.HeaderBytes + integerActualOwnedBytes(s.Manager.Z)
			if b.Used() != actual {
				t.Fatal("immediate actual graph", pair, ordinal, b.Used(), actual)
			}
			overflow := errors.Is(e, ErrCheckedInt64Overflow)
			if e != nil && !overflow {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal(pair, ordinal, e)
				}
				if n.Int64() != pair[0] {
					t.Fatal("failed source assignment")
				}
				if n.MulAssign(other, s, &local) != e {
					t.Fatal("stopped scope replay")
				}
			} else {
				exact := new(big.Int).Mul(big.NewInt(pair[0]), big.NewInt(pair[1]))
				if overflow != !exact.IsInt64() {
					t.Fatal("overflow result", pair, e)
				}
				if exact.IsInt64() && n.Int64() != exact.Int64() {
					t.Fatal("successful scalar")
				}
				completed = true
			}
			b.FailAfter(-1)
			serviceMust(t, local.Cleanup())
			serviceMust(t, local.Cleanup())
			serviceMust(t, s.Close())
			if b.Used() != 0 {
				t.Fatal("owning fallback refund", pair, ordinal, b.Used())
			}
			if completed {
				break
			}
		}
		if !completed {
			t.Fatal("fallback completion horizon", pair)
		}
	}
}

func TestCheckedInt64DisplayIntegerFlagsAndEveryReservation(t *testing.T) {
	n := NewCheckedInt64(math.MinInt64)
	completed := false
	for ordinal := int64(0); ordinal < 10; ordinal++ {
		s, b := serviceNew(t) // convenient common budget/cleanup owner.
		out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
		serviceMust(t, e)
		serviceMust(t, out.SetWidth(150))
		serviceMust(t, out.SetFill('_'))
		b.FailAfter(ordinal)
		e = n.Display(out)
		st := out.Stats()
		actual := s.HeaderBytes + s.CacheBytes + s.Manager.HeaderBytes + integerActualOwnedBytes(s.Manager.Z) + st.OwnerBytes + st.ScratchBytes + st.OutputOwnerBytes + st.OutputOverflowBytes
		if b.Used() != actual {
			t.Fatal("stream graph")
		}
		if e != nil {
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal(e)
			}
		} else {
			got, err := out.String()
			serviceMust(t, err)
			if got != strings.Repeat("_", 130)+"-9223372036854775808" {
				t.Fatal(got)
			}
			completed = true
		}
		b.FailAfter(-1)
		serviceMust(t, out.Close())
		serviceMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatal("display refund")
		}
		if completed {
			break
		}
	}
	if !completed {
		t.Fatal("display completion horizon")
	}
	out, e := NewNumericTextStream(NumericTextStreamConfig{})
	serviceMust(t, e)
	serviceMust(t, out.SetIntegerBase(16))
	serviceMust(t, out.SetShowBase(true))
	serviceMust(t, n.Display(out))
	got, e := out.String()
	serviceMust(t, e)
	if got != "0x8000000000000000" {
		t.Fatal("signed64 hex", got)
	}
	serviceMust(t, out.Close())
}

func TestCheckedInt64ScopeCopyLifetimeAndErasedProvider(t *testing.T) {
	var empty CheckedInt64MulScope
	serviceMust(t, empty.Cleanup())
	s, b := serviceNew(t)
	n := NewCheckedInt64(math.MaxInt64)
	b.FailAfter(0)
	var local CheckedInt64MulScope
	cause := n.MulAssign(NewCheckedInt64(1), s, &local)
	if !numericPhysicalReservationRefusal(cause) {
		t.Fatal(cause)
	}
	copied := local
	if !errors.Is(copied.Cleanup(), ErrParamsPrecondition) {
		t.Fatal("copied owning scope")
	}
	b.FailAfter(-1)
	serviceMust(t, local.Cleanup())
	if n.Int64() != math.MaxInt64 || local.active {
		t.Fatal("cleanup executed assignment")
	}
	if n.MulAssign(NewCheckedInt64(1), s, &local) != cause {
		t.Fatal("cleanup replayed stopped call")
	}
	serviceMust(t, s.Close())
	fast := NewCheckedInt64(2)
	serviceMust(t, fast.MulAssign(NewCheckedInt64(3), s, &local))
	if fast.Int64() != 6 || b.Used() != 0 {
		t.Fatal("erased closed provider")
	}
	var untouched CheckedInt64MulScope
	slow := NewCheckedInt64(math.MinInt32)
	if e := slow.MulAssign(NewCheckedInt64(0), s, &untouched); e == nil || untouched.self != nil || slow.Int64() != math.MinInt32 {
		t.Fatal("reached closed provider")
	}
	if n := testing.AllocsPerRun(1000, func() {
		a := NewCheckedInt64(23)
		_ = a.AddAssign(NewCheckedInt64(7))
		_ = a.MulAssign(a, nil, nil)
		_ = a.Neg()
	}); n != 0 {
		t.Fatal("scalar allocation", n)
	}
}

func TestCheckedInt64OverflowExceptionSubtype(t *testing.T) {
	n := NewCheckedInt64(math.MinInt64)
	e := n.Neg()
	var overflow CheckedInt64OverflowException
	var base Z3Exception
	if !errors.As(e, &overflow) || !errors.As(e, &base) || base.Message() != "checked_int64 overflow/underflow" || base.ErrorCode() != ErrorOK || base.HasErrorCode() {
		t.Fatal("source exception subtype", e, base)
	}
}

func TestCheckedInt64WideRandomRationalProductOracle(t *testing.T) {
	s, _ := serviceNew(t)
	random := rand.New(rand.NewChaCha8([32]byte{74, 93, 6}))
	for i := 0; i < 600; i++ {
		a, b := int64(random.Uint64()), int64(random.Uint64())
		if i%2 == 0 {
			a = int64(random.Uint64() & ((1 << 38) - 1))
			b = int64(random.Uint64() & ((1 << 24) - 1))
			if i%4 == 0 {
				a = -a
			}
		}
		exact := new(big.Int).Mul(big.NewInt(a), big.NewInt(b))
		n := NewCheckedInt64(a)
		var local CheckedInt64MulScope
		e := n.MulAssign(NewCheckedInt64(b), s, &local)
		if exact.IsInt64() {
			serviceMust(t, e)
			if n.Int64() != exact.Int64() {
				t.Fatal(i, a, b, n, exact)
			}
		} else if !errors.Is(e, ErrCheckedInt64Overflow) || n.Int64() != a {
			t.Fatal("wide overflow", i, a, b, e)
		}
		if local.self != nil {
			serviceMust(t, local.Cleanup())
		}
	}
}
