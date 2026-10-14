package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestFastFloatComplementPredicatesRejectEndedOwner(t *testing.T) {
	m := fastManager(t, 2)
	var a, zero FastFloat
	serviceMust(t, m.SetInt32(&a, 1))
	ended := a
	serviceMust(t, m.Del(&a))
	check := func(name string, got bool, e error) {
		t.Helper()
		if got || !errors.Is(e, ErrFastFloatOwner) {
			t.Fatalf("%s: got %t, %v", name, got, e)
		}
	}
	for name, fn := range map[string]func(*FastFloat) (bool, error){
		"nonpositive": m.IsNonPos,
		"nonnegative": m.IsNonNeg,
	} {
		got, e := fn(&ended)
		check(name, got, e)
	}
	for name, fn := range map[string]func(*FastFloat, *FastFloat) (bool, error){
		"not equal":     m.NotEqual,
		"less equal":    m.LessEqual,
		"greater equal": m.GreaterEqual,
	} {
		got, e := fn(&ended, &zero)
		check(name, got, e)
	}
}

func TestArbitraryFloatComplementPredicatesStopAtSignificandFailure(t *testing.T) {
	core, _, _ := mpfNumeralHelpersFixture(t)
	manager, e := NewArbitraryFloatNumeralManager(core, 8, 24)
	serviceMust(t, e)
	defer func() { serviceMust(t, manager.Close()) }()
	top, e := core.MkTopExp(8)
	serviceMust(t, e)
	bottom, e := core.MkBotExp(8)
	serviceMust(t, e)
	check := func(name string, got bool, e error) {
		t.Helper()
		if got || !errors.Is(e, arena.ErrInvalidSlot) {
			t.Fatalf("%s: got %t, %v", name, got, e)
		}
	}
	bad := Int{Val: 1, Cell: core.integer.FailedRef}
	nan := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, false), exponent: top, significand: bad}
	got, e := core.IsNaN(&nan)
	check("NaN", got, e)
	got, e = core.IsPositive(&nan)
	check("positive", got, e)
	nan.format = arbitraryFloatFormat(8, 24, true)
	got, e = core.IsNegative(&nan)
	check("negative", got, e)

	zero := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, false), exponent: bottom, significand: bad}
	got, e = core.IsNormal(&zero)
	check("normal", got, e)
	for name, fn := range map[string]func(*ArbitraryFloat) (bool, error){
		"f2n positive":    manager.IsPos,
		"f2n nonpositive": manager.IsNonPos,
	} {
		got, e := fn(&zero)
		check(name, got, e)
	}
	zero.format = arbitraryFloatFormat(8, 24, true)
	for name, fn := range map[string]func(*ArbitraryFloat) (bool, error){
		"f2n negative":    manager.IsNeg,
		"f2n nonnegative": manager.IsNonNeg,
	} {
		got, e := fn(&zero)
		check(name, got, e)
	}
}
