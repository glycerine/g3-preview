// Source: src/util/double_manager.h:34-100.
package util

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"math"
	"math/big"
	"math/rand/v2"
	"strconv"
	"testing"
)

func TestDoubleManagerSourceToleranceQuirk(t *testing.T) {
	m, e := NewDoubleManager(nil, nil)
	serviceMust(t, e)
	if DoubleManagerPrecise() || m.ZeroTolerance() != 1e-8 {
		t.Fatal("defaults")
	}
	for _, v := range []float64{-2e-8, -1e-8, math.Nextafter(-1e-8, 0), -0.0, 0, math.Nextafter(1e-8, 0), 1e-8, 2e-8, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if m.IsPos(v) != (v > 1e-8) || m.IsNeg(v) != (v < 1e-8) || m.IsZero(v) != (-1e-8 <= v && v <= 1e-8) || m.IsNonPos(v) == m.IsPos(v) || m.IsNonNeg(v) == m.IsNeg(v) {
			t.Fatal("source tolerance", v)
		}
	}
	if !m.IsNeg(0.5e-8) || !m.Eq(0, 1e-8) || m.Eq(math.Inf(1), math.Inf(1)) {
		t.Fatal("original predicates")
	}
}

func TestDoubleManagerParameterBorrowAndFailedUpdate(t *testing.T) {
	c, _, _ := paramsTestOwner(t)
	value, e := c.NewValue()
	serviceMust(t, e)
	key := "zero_tolerance"
	m, e := NewDoubleManager(c, &value)
	serviceMust(t, e)
	if m.ZeroTolerance() != 1e-8 {
		t.Fatal("missing tolerance")
	}
	for _, tol := range []float64{0, 1e-3, -1e-3, math.Inf(1), math.Inf(-1), math.NaN()} {
		serviceMust(t, c.SetDoubleName(&value, &key, tol))
		serviceMust(t, m.UpdtParams(c, &value))
		if math.Float64bits(m.ZeroTolerance()) != math.Float64bits(tol) {
			t.Fatal("parameter filtering")
		}
		for _, a := range []float64{-2, 0, 2, math.NaN()} {
			if m.IsZero(a) != (-tol <= a && a <= tol) || m.IsNeg(a) != (a < tol) {
				t.Fatal("custom tolerance", tol, a)
			}
		}
	}
	before := math.Float64bits(m.ZeroTolerance())
	if !errors.Is(m.UpdtParams(nil, &value), ErrParamsOwner) || math.Float64bits(m.ZeroTolerance()) != before {
		t.Fatal("failed update publication")
	}
	serviceMust(t, c.Del(&value))
	// The manager owns only the copied tolerance; provider destruction cannot
	// invalidate the source scalar. Default update erases a missing context.
	serviceMust(t, m.UpdtParams(nil, nil))
	if m.ZeroTolerance() != 1e-8 {
		t.Fatal("default update")
	}
	copied := m
	if copied.IsPos(1e-7) != m.IsPos(1e-7) {
		t.Fatal("copyable scalar")
	}
}

func TestDoubleManagerArithmeticIndependentOracleAndBoundaries(t *testing.T) {
	random := rand.New(rand.NewChaCha8([32]byte{39, 52, 7}))
	for i := 0; i < 5000; i++ {
		a := int64(random.IntN(200001) - 100000)
		b := int64(random.IntN(200001) - 100000)
		c := int64(random.IntN(2001) - 1000)
		// These integers/products lie inside the exact IEEE64 range, giving an
		// independent arbitrary-precision oracle without matching float code.
		sum := new(big.Int).Add(big.NewInt(a), big.NewInt(b)).Int64()
		product := new(big.Int).Mul(big.NewInt(b), big.NewInt(c))
		addmul := new(big.Int).Add(big.NewInt(a), product).Int64()
		submul := new(big.Int).Sub(big.NewInt(a), product).Int64()
		if DoubleManagerAdd(float64(a), float64(b)) != float64(sum) || DoubleManagerSub(float64(a), float64(b)) != float64(a-b) || DoubleManagerMul(float64(b), float64(c)) != float64(product.Int64()) || DoubleManagerAddmul(float64(a), float64(b), float64(c)) != float64(addmul) || DoubleManagerSubmul(float64(a), float64(b), float64(c)) != float64(submul) {
			t.Fatal("arithmetic", i)
		}
		if b != 0 {
			want, _ := new(big.Rat).SetFrac(big.NewInt(a), big.NewInt(b)).Float64()
			if DoubleManagerDiv(float64(a), float64(b)) != want {
				t.Fatal("division", i)
			}
		}
	}
	if DoubleManagerAddmul(-1, 1+0x1p-27, 1-0x1p-27) != 0 {
		t.Fatal("unexpected FMA contraction")
	}
	for _, a := range []float64{math.Copysign(0, -1), 0, -1.5, 1.5, math.Inf(1), math.Inf(-1), math.Float64frombits(0xfff8000000000042)} {
		original := math.Float64bits(a)
		copy := a
		DoubleManagerAbs(&copy)
		want := a
		if a < 0 {
			want = -a
		}
		if math.Float64bits(copy) != math.Float64bits(want) {
			t.Fatal("source abs", a)
		}
		DoubleManagerDel(&copy)
		if math.Float64bits(copy) != math.Float64bits(want) {
			t.Fatal("del changed scalar")
		}
		copy = a
		DoubleManagerNeg(&copy)
		if math.Float64bits(copy) != original^(uint64(1)<<63) {
			t.Fatal("neg bits")
		}
		DoubleManagerReset(&copy)
		if math.Float64bits(copy) != 0 {
			t.Fatal("reset positive zero")
		}
		if DoubleManagerIsInt(a) != (a == math.Floor(a)) || DoubleManagerFloor(a) != math.Floor(a) && !math.IsNaN(a) || DoubleManagerCeil(a) != math.Ceil(a) && !math.IsNaN(a) {
			t.Fatal("projection")
		}
	}
	DoubleManagerDel(nil)
	for _, p := range []uint32{0, 1, 2, 31, math.MaxInt32, math.MaxInt32 + 1, math.MaxUint32} {
		v, e := DoubleManagerPower(2, p)
		if sourceDebug && p > math.MaxInt32 {
			if !errors.Is(e, ErrDoubleManagerPower) {
				t.Fatal("debug power", e)
			}
		} else {
			serviceMust(t, e)
			if v != math.Pow(2, float64(int32(p))) {
				t.Fatal("source power", p, v)
			}
		}
	}
	if !math.IsInf(DoubleManagerInv(0), 1) || !math.IsInf(DoubleManagerInv(math.Copysign(0, -1)), -1) || !math.IsNaN(DoubleManagerDiv(0, 0)) {
		t.Fatal("IEEE zero division")
	}
	if !DoubleManagerIsOne(1) || DoubleManagerIsOne(math.Nextafter(1, 2)) || !DoubleManagerIsMinusOne(-1) {
		t.Fatal("exact units")
	}
}

func TestDoubleManagerSettersOrderingAndNumericHash(t *testing.T) {
	var a, b float64
	DoubleManagerSetFraction(&a, -3, 2)
	if a != -1.5 {
		t.Fatal("fraction")
	}
	DoubleManagerSetFraction(&a, 1, 0)
	if !math.IsInf(a, 1) {
		t.Fatal("zero denominator")
	}
	DoubleManagerSetInt32(&a, math.MinInt32)
	if a != -2147483648 {
		t.Fatal("signed32")
	}
	DoubleManagerSetUint32(&a, math.MaxUint32)
	if a != 4294967295 {
		t.Fatal("unsigned32")
	}
	DoubleManagerSetInt64(&a, math.MaxInt64)
	if a != 0x1p63 {
		t.Fatal("signed64 rounding")
	}
	DoubleManagerSetUint64(&a, math.MaxUint64)
	if a != 0x1p64 {
		t.Fatal("unsigned64 rounding")
	}
	DoubleManagerSet(&b, math.Copysign(0, -1))
	DoubleManagerSwap(&a, &b)
	if !math.Signbit(a) || b != 0x1p64 {
		t.Fatal("swap")
	}
	DoubleManagerSwap(&a, &a)
	if !math.Signbit(a) {
		t.Fatal("self swap")
	}
	values := []float64{math.NaN(), math.Inf(-1), -2, -0.5, math.Copysign(0, -1), 0, 0.5, 1, math.Inf(1)}
	for _, a := range values {
		for _, b := range values {
			if DoubleManagerLt(a, b) != (a < b) || DoubleManagerLe(a, b) != (a <= b) || DoubleManagerGt(a, b) != (a > b) || DoubleManagerGe(a, b) != (a >= b) {
				t.Fatal("ordering")
			}
		}
	}
	for _, a := range []float64{-0.9, 0, 0.9, 1, 1.9, 0x1p32 + 3, math.Nextafter(0x1p64, 0)} {
		got, e := DoubleManagerHash(a)
		serviceMust(t, e)
		if got != Uint64(uint64(math.Trunc(a))) {
			t.Fatal("numeric cast hash", a, got)
		}
	}
	for _, a := range []float64{-1, -100, 0x1p64, math.Inf(1), math.NaN()} {
		if _, e := DoubleManagerHash(a); !errors.Is(e, ErrDoubleManagerHashDomain) {
			t.Fatal("undefined conversion guard", a, e)
		}
	}
}

func TestDoubleManagerAtOfPrefixAndIEEEBoundaries(t *testing.T) {
	for _, tc := range []struct {
		text string
		want float64
	}{
		{"", 0}, {"  junk", 0}, {"+", 0}, {"-", 0}, {"-0", math.Copysign(0, -1)}, {"\t\n\v\f\r -1.5tail", -1.5}, {"1/2", 1}, {"1_2", 1}, {"1,2", 1}, {"1.25e+2foo", 125}, {"1.25e+foo", 1.25}, {"1.25e-", 1.25}, {".5", 0.5}, {".e2", 0}, {"1\x00e99", 1},
		{"0x1.8", 1.5}, {"0x1.8p2rest", 6}, {"-0XAp-2", -2.5}, {"0x1p-", 1}, {"0x", 0}, {"-0x.", math.Copysign(0, -1)},
		{"Inf", math.Inf(1)}, {"-iNfInItYtail", math.Inf(-1)}, {"1e99999", math.Inf(1)}, {"-1e-99999", math.Copysign(0, -1)},
		{"0x1p-1074", math.SmallestNonzeroFloat64}, {"0x1p-1075", 0}, {"0x1.fffffffffffffp1023", math.MaxFloat64},
	} {
		var got float64
		DoubleManagerSetString(&got, tc.text)
		if math.Float64bits(got) != math.Float64bits(tc.want) {
			t.Fatal(tc.text, got, tc.want)
		}
	}
	for _, text := range []string{"NaN", "nan(payload)", "-NAN(0x123)", "+nan(foo"} {
		var got float64
		DoubleManagerSetString(&got, text)
		if !math.IsNaN(got) || math.Signbit(got) != (text[0] == '-') {
			t.Fatal(text, got)
		}
	}
	random := rand.New(rand.NewChaCha8([32]byte{1, 7, 5}))
	for i := 0; i < 3000; i++ {
		n := int64(random.IntN(2000001) - 1000000)
		exponent := random.IntN(620) - 310
		token := fmt.Sprintf("%de%d", n, exponent)
		want, _ := strconv.ParseFloat(token, 64)
		var got float64
		DoubleManagerSetString(&got, " \t"+token+"trailing")
		if math.Float64bits(got) != math.Float64bits(want) {
			t.Fatal("decimal prefix", token, got, want)
		}
	}
}

func TestDoubleManagerOwnedPrecision12FormattingEveryReservation(t *testing.T) {
	for _, tc := range []struct {
		a    float64
		want string
	}{
		{1.23456789012345, "1.23456789012"}, {1e-5, "1e-05"}, {1e-4, "0.0001"}, {1e11, "100000000000"}, {1e12, "1e+12"}, {math.Copysign(0, -1), "-0"}, {math.SmallestNonzeroFloat64, "4.94065645841e-324"}, {math.MaxFloat64, "1.79769313486e+308"}, {math.Inf(1), "inf"}, {math.Inf(-1), "-inf"}, {math.NaN(), "nan"}, {math.Float64frombits(0xfff8000000000001), "-nan"},
	} {
		completed := false
		for fail := int64(0); fail < 4; fail++ {
			b := arena.NewBudget(10000)
			b.FailAfter(fail)
			var local OwnedStringStream
			var result OwnedString
			e := DoubleManagerToString(tc.a, b, &local, &result)
			actual := uint64(local.Buffer.String.StorageBytes() + result.StorageBytes())
			if b.Used() != actual {
				t.Fatal("physical graph", tc.a, fail, b.Used(), actual)
			}
			if e != nil {
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal(e)
				}
			} else {
				got, err := result.View().String()
				serviceMust(t, err)
				if got != tc.want {
					t.Fatal(tc.a, got, tc.want)
				}
				if !local.Buffer.String.Closed {
					t.Fatal("source stream not destroyed")
				}
				completed = true
			}
			b.FailAfter(-1)
			result.OwnAbort()
			local.OwnAbort()
			if b.Used() != 0 {
				t.Fatal("refund", tc.a, fail, b.Used())
			}
			if completed {
				break
			}
		}
		if !completed {
			t.Fatal("format completion horizon", tc.a)
		}
	}
}

func TestDoubleManagerSelectedIEEE64NaNPayload(t *testing.T) {
	for _, tc := range []struct {
		text    string
		payload uint64
	}{
		{"nan()", 0}, {"nan(123)", 123}, {"nan(0123)", 83}, {"nan(0x123)", 0x123}, {"nan(0Xfff)", 0xfff}, {"nan(foo)", 0}, {"nan(09)", 0}, {"nan(1_2)", 0}, {"nan(-1)", 0}, {"nan(0x)", 0}, {"nan(17", 0},
		{"nan(18446744073709551616)", math.MaxUint64}, {"nan(0x10000000000000)", 0},
	} {
		var got float64
		DoubleManagerSetString(&got, tc.text)
		want := uint64(0x7ff8000000000000) | tc.payload&((uint64(1)<<51)-1)
		if math.Float64bits(got) != want {
			t.Fatal(tc.text, math.Float64bits(got), want)
		}
		DoubleManagerSetString(&got, "-"+tc.text)
		if math.Float64bits(got) != want|(uint64(1)<<63) {
			t.Fatal("nan sign", tc.text)
		}
	}
}

func TestDoubleManagerInPlaceInverseAndScalarAllocation(t *testing.T) {
	for _, v := range []float64{2, -2, 0, math.Copysign(0, -1), math.Inf(1), math.Inf(-1)} {
		a := v
		DoubleManagerInvInPlace(&a)
		if math.Float64bits(a) != math.Float64bits(1/v) {
			t.Fatal(v, a)
		}
	}
	m, e := NewDoubleManager(nil, nil)
	serviceMust(t, e)
	if n := testing.AllocsPerRun(1000, func() {
		a := DoubleManagerAdd(2, 3)
		DoubleManagerNeg(&a)
		DoubleManagerAbs(&a)
		DoubleManagerInvInPlace(&a)
		DoubleManagerSetInt32(&a, 7)
		if m.IsPos(a) {
			DoubleManagerReset(&a)
		}
	}); n != 0 {
		t.Fatal("scalar allocated", n)
	}
}
