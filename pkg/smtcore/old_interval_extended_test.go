package smtcore

import (
	"math/big"
	"testing"
)

func TestSMTExtendedRationalArithmeticAndUndefinedInfinity(t *testing.T) {
	minusInfinity := smtInfiniteExtended(false)
	plusInfinity := smtInfiniteExtended(true)
	if minusInfinity.cmp(&plusInfinity) >= 0 || minusInfinity.String() != "-oo" || plusInfinity.String() != "oo" {
		t.Fatal("extended infinity ordering or formatting")
	}
	if err := plusInfinity.add(&minusInfinity); err == nil {
		t.Fatal("opposite infinity addition accepted")
	}
	plusInfinity = smtInfiniteExtended(true)
	if err := plusInfinity.sub(&plusInfinity); err == nil {
		t.Fatal("same infinity subtraction accepted")
	}
	value := smtFiniteExtended(big.NewRat(-3, 2))
	other := smtFiniteExtended(big.NewRat(5, 3))
	if err := value.add(&other); err != nil || value.value.Cmp(big.NewRat(1, 6)) != 0 {
		t.Fatalf("finite addition = %s, err %v", &value.value, err)
	}
	value.mul(&other)
	if value.value.Cmp(big.NewRat(5, 18)) != 0 {
		t.Fatalf("finite multiplication = %s", &value.value)
	}
	if err := value.inverse(); err != nil || value.value.Cmp(big.NewRat(18, 5)) != 0 {
		t.Fatalf("finite inverse = %s, err %v", &value.value, err)
	}
}

func TestSMTExtendedRationalZeroInfinityPowerAndNegation(t *testing.T) {
	zero := smtFiniteExtended(nil)
	infinity := smtInfiniteExtended(false)
	infinity.mul(&zero)
	if !infinity.zero() {
		t.Fatal("zero times infinity must be zero")
	}
	if err := infinity.inverse(); err == nil {
		t.Fatal("zero inverse accepted")
	}
	infinity = smtInfiniteExtended(false)
	infinity.power(2)
	if infinity.kind != smtExtendedPlusInfinity {
		t.Fatal("even power of minus infinity")
	}
	infinity = smtInfiniteExtended(false)
	infinity.power(3)
	if infinity.kind != smtExtendedMinusInfinity {
		t.Fatal("odd power of minus infinity")
	}
	value := smtFiniteExtended(big.NewRat(-2, 3))
	value.power(3)
	if value.value.Cmp(big.NewRat(-8, 27)) != 0 {
		t.Fatalf("finite power = %s", &value.value)
	}
	value.neg()
	if value.value.Cmp(big.NewRat(8, 27)) != 0 {
		t.Fatalf("finite negation = %s", &value.value)
	}
}
