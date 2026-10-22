package smtcore

import (
	"math/big"
	"testing"
)

func TestSMTOldIntervalAddSubtractContainsAndDependencies(t *testing.T) {
	var manager smtArithmeticBoundDependencyManager
	a, b := &smtArithmeticBound{variable: 1}, &smtArithmeticBound{variable: 2}
	left, err := newSMTOldIntervalBounds(&manager, big.NewRat(-1, 1), false, manager.Leaf(a), big.NewRat(2, 1), true, manager.Leaf(b))
	if err != nil {
		t.Fatal(err)
	}
	right, err := newSMTOldIntervalPoint(&manager, big.NewRat(3, 1), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = left.add(right); err != nil || left.String() != "[2, 5)" {
		t.Fatalf("addition = %s, %v", left, err)
	}
	if !left.contains(big.NewRat(2, 1)) || left.contains(big.NewRat(5, 1)) || left.containsZero() {
		t.Fatal("containment after addition")
	}
	if !manager.Contains(left.lowerDep, a) || !manager.Contains(left.upperDep, b) {
		t.Fatal("bound dependencies lost")
	}
	if err = left.sub(right); err != nil || left.String() != "[-1, 2)" {
		t.Fatalf("subtraction = %s, %v", left, err)
	}
}

func TestSMTOldIntervalInverseAndPower(t *testing.T) {
	var manager smtArithmeticBoundDependencyManager
	positive, err := newSMTOldIntervalBounds(&manager, big.NewRat(0, 1), true, nil, big.NewRat(2, 1), false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = positive.inverse(); err != nil || positive.String() != "[1/2, oo)" {
		t.Fatalf("inverse = %s, %v", positive, err)
	}
	crossing, err := newSMTOldIntervalBounds(&manager, big.NewRat(-3, 1), false, nil, big.NewRat(2, 1), true, nil)
	if err != nil {
		t.Fatal(err)
	}
	crossing.power(2)
	if crossing.String() != "[0, 9]" {
		t.Fatalf("even power = %s", crossing)
	}
	negative, err := newSMTOldIntervalBounds(&manager, big.NewRat(-2, 1), false, nil, big.NewRat(-1, 1), false, nil)
	if err != nil {
		t.Fatal(err)
	}
	negative.power(3)
	if negative.String() != "[-8, -1]" {
		t.Fatalf("odd power = %s", negative)
	}
	zero, err := newSMTOldIntervalPoint(&manager, new(big.Rat), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = zero.inverse(); err == nil {
		t.Fatal("inverse accepted interval containing zero")
	}
}
