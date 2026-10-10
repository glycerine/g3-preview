package lp

import (
	"math"
	"testing"
)

// Source: src/test/lp.cpp:1972-1995, test_stacked_unsigned (--smap).
func TestLPStackedUnsignedSource(t *testing.T) {
	v := NewStackedUint32(0)
	v.Value = 1
	v.Value = 2
	v.Push()
	v.Value = 3
	v.Value = 4
	v.Pop(1)
	if v.Value != 2 {
		t.Fatal("first source assertion")
	}
	v.Value++
	v.Value++
	v.Push()
	v.Value++
	v.Push()
	v.Value++
	v.Pop(2)
	if v.Value != 4 || v.StackSize() != 0 {
		t.Fatal("second source assertion")
	}
	v.Pop(math.MaxUint32)
	v.Value = math.MaxUint32
	v.Value++
	if v.Value != 0 {
		t.Fatal("native unsigned wrap")
	}
	v.Push()
	v.Value = 11
	v.Pop(0)
	if v.Value != 11 || v.StackSize() != 1 {
		t.Fatal("pop zero")
	}
	v.Pop(math.MaxUint32)
	if v.Value != 0 || v.StackSize() != 0 {
		t.Fatal("excess pop restores oldest snapshot")
	}
}

func TestLPStackedSignedStatusAndStrategy(t *testing.T) {
	i := NewStackedInt32(-1)
	i.Push()
	i.Value = 3
	i.Push()
	i.Value = -7
	i.Pop(0)
	if i.Value != -7 || i.StackSize() != 2 {
		t.Fatal(i)
	}
	i.Pop(math.MaxUint32)
	if i.Value != -1 || i.StackSize() != 0 {
		t.Fatal(i)
	}
	s := NewStackedStatus(StatusOptimal)
	s.Push()
	s.Value = StatusInfeasible
	s.Push()
	s.Value = StatusCancelled
	s.Pop(0)
	if s.Value != StatusCancelled || s.StackSize() != 2 {
		t.Fatal(s)
	}
	s.Pop(1)
	if s.Value != StatusInfeasible || s.StackSize() != 1 {
		t.Fatal(s)
	}
	s.Pop(math.MaxUint32)
	if s.Value != StatusOptimal || s.StackSize() != 0 {
		t.Fatal(s)
	}
	p := NewStackedStrategy(StrategyUndecided)
	p.Push()
	p.Value = StrategyLU
	p.Push()
	p.Value = StrategyTableauRows
	p.Pop(0)
	if p.Value != StrategyTableauRows || p.StackSize() != 2 {
		t.Fatal(p)
	}
	p.Pop(1)
	if p.Value != StrategyLU {
		t.Fatal(p)
	}
	p.Pop(math.MaxUint32)
	if p.Value != StrategyUndecided || p.StackSize() != 0 {
		t.Fatal(p)
	}
}
