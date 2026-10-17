package api

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestAPINumeralStringArithmeticFPAAndRefusal(t *testing.T) {
	m := astTestManager(t)
	arith, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	fpa, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	c, err := newAPIContextResults(m, false)
	astTestMust(t, err)
	real, err := m.MkFamilySort(arith, 0, nil)
	astTestMust(t, err)
	value, err := m.NumeralService().NewValue()
	astTestMust(t, err)
	astTestMust(t, m.NumeralService().SetString(&value, "7/3"))
	numeral, err := m.ArithmeticNumeral(&value, false)
	astTestMust(t, err)
	astTestMust(t, m.NumeralService().SetString(&value, "18446744073709551617"))
	wideBV, err := m.BitVectorNumeral(&value, 65)
	astTestMust(t, err)
	astTestMust(t, m.NumeralService().Del(&value))
	byteBV, err := m.BitVectorNumeralUint64(255, 8)
	astTestMust(t, err)
	floatSort, err := m.MkFamilySort(fpa, 3, nil)
	astTestMust(t, err)
	zero, err := m.MkFPAZeroOfSort(floatSort, true)
	astTestMust(t, err)
	rmSort, err := m.MkFamilySort(fpa, 1, nil)
	astTestMust(t, err)
	rm, err := m.GetSomeValue(rmSort)
	astTestMust(t, err)
	for _, tc := range []struct {
		h    ast.ASTHandle
		want string
	}{
		{numeral, "7/3"},
		{byteBV, "255"},
		{wideBV, "18446744073709551617"},
		{zero, "-zero"},
		{rm, "roundTowardZero"},
	} {
		got, err := c.numeralString(tc.h)
		if err != nil || got != tc.want {
			t.Fatal("source API numeral string", got, tc.want, err)
		}
	}
	symbolic, err := m.MkVar(0, real)
	astTestMust(t, err)
	if got, err := c.numeralString(symbolic); got != "" || !errors.Is(err, errAPINumeralArgument) {
		t.Fatal("source invalid numeral argument", got, err)
	}
	bvSort, err := m.GetSort(byteBV)
	astTestMust(t, err)
	symbolicBV, err := m.MkVar(1, bvSort)
	astTestMust(t, err)
	if got, err := c.numeralString(symbolicBV); got != "" || !errors.Is(err, errAPINumeralArgument) {
		t.Fatal("source symbolic BitVec refusal", got, err)
	}
	astTestMust(t, c.Close())
}
