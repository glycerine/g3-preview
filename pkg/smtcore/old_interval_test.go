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

func TestSMTOldIntervalMultiplicationSignTable(t *testing.T) {
	var manager smtArithmeticBoundDependencyManager
	makeInterval := func(lower, upper int64) *smtOldInterval {
		result, err := newSMTOldIntervalBounds(&manager, big.NewRat(lower, 1), false, nil, big.NewRat(upper, 1), false, nil)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	classes := []*smtOldInterval{makeInterval(-3, -2), makeInterval(-2, 3), makeInterval(2, 4)}
	wants := [3][3]string{{"[4, 9]", "[-9, 6]", "[-12, -4]"}, {"[-9, 6]", "[-6, 9]", "[-8, 12]"}, {"[-12, -4]", "[-8, 12]", "[4, 16]"}}
	for left := range classes {
		for right := range classes {
			result := classes[left].clone()
			if err := result.mul(classes[right]); err != nil {
				t.Fatal(left, right, err)
			}
			if result.String() != wants[left][right] {
				t.Fatalf("case %d,%d = %s, want %s", left, right, result, wants[left][right])
			}
		}
	}
}

func TestSMTOldIntervalMultiplyOpenZeroAndDivide(t *testing.T) {
	var manager smtArithmeticBoundDependencyManager
	left, err := newSMTOldIntervalBounds(&manager, big.NewRat(0, 1), true, nil, big.NewRat(2, 1), false, nil)
	if err != nil {
		t.Fatal(err)
	}
	right, err := newSMTOldIntervalBounds(&manager, big.NewRat(-1, 1), false, nil, big.NewRat(0, 1), false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = left.mul(right); err != nil || left.String() != "[-2, 0]" {
		t.Fatalf("open-zero product = %s, %v", left, err)
	}
	numerator, err := newSMTOldIntervalBounds(&manager, big.NewRat(2, 1), false, nil, big.NewRat(4, 1), false, nil)
	if err != nil {
		t.Fatal(err)
	}
	denominator, err := newSMTOldIntervalBounds(&manager, big.NewRat(1, 1), false, nil, big.NewRat(2, 1), false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = numerator.div(denominator); err != nil || numerator.String() != "[1, 4]" {
		t.Fatalf("division = %s, %v", numerator, err)
	}
	zero, err := newSMTOldIntervalPoint(&manager, new(big.Rat), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = zero.div(right); err == nil {
		t.Fatal("division by interval containing zero accepted")
	}
}

func TestSMTOldIntervalMultiplicationClosedOracle(t *testing.T) {
	var manager smtArithmeticBoundDependencyManager
	for a := int64(-5); a <= 5; a++ {
		for b := a; b <= 5; b++ {
			for c := int64(-5); c <= 5; c++ {
				for d := c; d <= 5; d++ {
					left, err := newSMTOldIntervalBounds(&manager, big.NewRat(a, 1), false, nil, big.NewRat(b, 1), false, nil)
					if err != nil {
						t.Fatal(err)
					}
					right, err := newSMTOldIntervalBounds(&manager, big.NewRat(c, 1), false, nil, big.NewRat(d, 1), false, nil)
					if err != nil {
						t.Fatal(err)
					}
					products := []int64{a * c, a * d, b * c, b * d}
					minimum, maximum := products[0], products[0]
					for _, value := range products[1:] {
						if value < minimum {
							minimum = value
						}
						if value > maximum {
							maximum = value
						}
					}
					if err = left.mul(right); err != nil {
						t.Fatal(a, b, c, d, err)
					}
					if left.lower.infinite() || left.upper.infinite() || left.lower.value.Cmp(big.NewRat(minimum, 1)) != 0 || left.upper.value.Cmp(big.NewRat(maximum, 1)) != 0 || left.lowerOpen || left.upperOpen {
						t.Fatalf("[%d,%d]*[%d,%d] = %s, want [%d, %d]", a, b, c, d, left, minimum, maximum)
					}
				}
			}
		}
	}
}
