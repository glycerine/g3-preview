package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestHardwareFloatNumeralDirectedIntervalOperations(t *testing.T) {
	m, err := NewHardwareFloatNumeralManager(NewHardwareFloatManager(), 11, 53)
	if err != nil || m.RoundingMode() != ArbitraryFloatTowardPositive || m.ExponentBits() != 11 || m.SignificandBits() != 53 {
		t.Fatal(m, err)
	}
	x, y, out := HardwareFloat{1}, HardwareFloat{3}, HardwareFloat{}
	if err := m.Div(&x, &y, &out); err != nil || out.value != math.Nextafter(1.0/3.0, math.Inf(1)) {
		t.Fatal("upper division bound", out, err)
	}
	m.RoundToMinusInf()
	if err := m.Div(&x, &y, &out); err != nil || out.value != 1.0/3.0 {
		t.Fatal("lower division bound", out, err)
	}
	if err := m.SetFloat64(&out, math.Inf(1)); !errors.Is(err, ErrHardwareFloatNonRegular) || !math.IsInf(out.value, 1) {
		t.Fatal("source check follows output mutation", out, err)
	}
	if err := m.SetFloat64(&out, math.Copysign(0, -1)); err != nil || m.IsNeg(&out) || m.IsPos(&out) || !m.IsNonNeg(&out) || !m.IsNonPos(&out) {
		t.Fatal("signed zero wrapper predicates", out, err)
	}
	if err := m.Reset(&out); err != nil || out.RawBits() != 0 {
		t.Fatal("reset", out, err)
	}
}

func TestHardwareFloatNumeralFloorCeilPowerAndGuards(t *testing.T) {
	m, err := NewHardwareFloatNumeralManager(NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	x, out := HardwareFloat{-1.25}, HardwareFloat{}
	if err := m.Floor(&x, &out); err != nil || out.value != -2 {
		t.Fatal("floor", out, err)
	}
	if err := m.Ceil(&x, &out); err != nil || out.value != -1 {
		t.Fatal("ceil", out, err)
	}
	x.value = math.Copysign(0, -1)
	if err := m.Floor(&x, &x); err != nil || x.RawBits() != 0x8000000000000000 {
		t.Fatal("signed-zero floor", x, err)
	}
	x.value = 2
	if err := m.Power(&x, 5, &out); err != nil || out.value != 32 {
		t.Fatal("power", out, err)
	}
	if err := m.Power(&x, 0, &out); err != nil || out.value != 1 {
		t.Fatal("zero exponent", out, err)
	}
	if err := m.Power(&x, 1<<31, &out); !errors.Is(err, arena.ErrBounds) {
		t.Fatal("wrapped source exponent", err)
	}
	x.value = math.MaxFloat64
	if err := m.Inc(&x); !errors.Is(err, ErrHardwareFloatNonRegular) || !math.IsInf(x.value, 1) {
		t.Fatal("overflow check", x, err)
	}
}

func TestHardwareFloatNumeralSourceThinPredicatesAndMoves(t *testing.T) {
	m, e := NewHardwareFloatNumeralManager(NewHardwareFloatManager(), 11, 53)
	if e != nil {
		t.Fatal(e)
	}
	a, b, out := HardwareFloat{-1}, HardwareFloat{2}, HardwareFloat{}
	before := a.RawBits()
	if !m.IsMinusOne(&a) || m.IsOne(&a) || !m.IsInteger(&a) || a.RawBits() != before {
		t.Fatal("source minus-one test must restore borrowed value", a)
	}
	if !m.Less(&a, &b) || !m.LessEqual(&a, &b) || m.Equal(&a, &b) || m.Greater(&a, &b) || m.GreaterEqual(&a, &b) {
		t.Fatal("wrapper comparison order")
	}
	if e := m.Abs(&a, &out); e != nil || out.value != 1 {
		t.Fatal("abs", out, e)
	}
	if e := m.Neg(&out, &out); e != nil || out.value != -1 {
		t.Fatal("in-place neg", out, e)
	}
	if e := m.Swap(&a, &b); e != nil || a.value != 2 || b.value != -1 {
		t.Fatal("swap", a, b, e)
	}
	if m.ToFloat64(&a) != 2 || !m.IsMinusOne(&out) || m.IsOne(&out) {
		t.Fatal("scalar forwarding")
	}
	if e := m.SetUint32(&out, ^uint32(0)); e != nil || out.value != float64(^uint32(0)) {
		t.Fatal("unsigned source setter", out, e)
	}
}
