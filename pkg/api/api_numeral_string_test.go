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
	datalog, err := m.RegisterDatalogPlugin()
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
	domainName, err := m.Symbols.Intern("D")
	astTestMust(t, err)
	domainNameParam, err := m.NewParameterSymbol(domainName)
	astTestMust(t, err)
	domainSize, err := m.NumeralService().NewValue()
	astTestMust(t, err)
	astTestMust(t, m.NumeralService().SetUint64(&domainSize, 1<<63))
	domainSizeParam, err := m.NewParameterRational(&domainSize)
	astTestMust(t, err)
	domain, err := m.MkFamilySort(datalog, 1, []ast.ASTParameter{domainNameParam, domainSizeParam})
	astTestMust(t, err)
	astTestMust(t, m.DelParameter(&domainNameParam))
	astTestMust(t, m.DelParameter(&domainSizeParam))
	astTestMust(t, m.NumeralService().SetUint64(&domainSize, (1<<63)-1))
	domainValueParam, err := m.NewParameterRational(&domainSize)
	astTestMust(t, err)
	astTestMust(t, m.NumeralService().Del(&domainSize))
	domainSortParam, err := m.NewParameterAST(domain)
	astTestMust(t, err)
	domainNumeral, err := m.MkFamilyApp(datalog, 13, []ast.ASTParameter{domainValueParam, domainSortParam}, nil)
	astTestMust(t, err)
	astTestMust(t, m.DelParameter(&domainValueParam))
	astTestMust(t, m.DelParameter(&domainSortParam))
	for _, tc := range []struct {
		h    ast.ASTHandle
		want string
	}{
		{numeral, "7/3"},
		{byteBV, "255"},
		{wideBV, "18446744073709551617"},
		{domainNumeral, "9223372036854775807"},
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
	symbolicDomain, err := m.MkVar(2, domain)
	astTestMust(t, err)
	if got, err := c.numeralString(symbolicDomain); got != "" || !errors.Is(err, errAPINumeralArgument) {
		t.Fatal("source symbolic finite-domain refusal", got, err)
	}
	astTestMust(t, c.Close())
}
