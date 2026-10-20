package ast

import "testing"

func TestASTFloatingPointOriginalValueAndUniqueValueClassification(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	family, e := m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	floatSort, e := m.MkFamilySort(family, 3, nil)
	astTestMust(t, e)
	astTestPin(t, m, floatSort)
	nan, e := m.MkFPANaNOfSort(floatSort)
	astTestMust(t, e)
	astTestPin(t, m, nan)
	if value, e := m.IsValue(nan); e != nil || !value {
		t.Fatal("NaN value", value, e)
	}
	if unique, e := m.IsUniqueValue(nan); e != nil || unique {
		t.Fatal("NaN unique value", unique, e)
	}
	rounding, e := m.MkFamilyApp(family, astFPARMTowardZero, nil, nil)
	astTestMust(t, e)
	astTestPin(t, m, rounding)
	if value, e := m.IsValue(rounding); e != nil || !value {
		t.Fatal("rounding value", value, e)
	}
	if unique, e := m.IsUniqueValue(rounding); e != nil || !unique {
		t.Fatal("rounding unique value", unique, e)
	}
	sign, e := m.BitVectorNumeralUint64(0, 1)
	astTestMust(t, e)
	exponent, e := m.BitVectorNumeralUint64(0, 8)
	astTestMust(t, e)
	significand, e := m.BitVectorNumeralUint64(0, 23)
	astTestMust(t, e)
	fp, e := m.MkFamilyApp(family, astFPAFP, nil, []ASTHandle{sign, exponent, significand})
	astTestMust(t, e)
	astTestPin(t, m, fp)
	if value, e := m.IsValue(fp); e != nil || !value {
		t.Fatal("fp value", value, e)
	}
	if unique, e := m.IsUniqueValue(fp); e != nil || !unique {
		t.Fatal("fp unique value", unique, e)
	}
	signSort, e := m.GetSort(sign)
	astTestMust(t, e)
	variable, e := m.MkVar(0, signSort)
	astTestMust(t, e)
	nonvalue, e := m.MkFamilyApp(family, astFPAFP, nil, []ASTHandle{variable, exponent, significand})
	astTestMust(t, e)
	astTestPin(t, m, nonvalue)
	if value, e := m.IsValue(nonvalue); e != nil || value {
		t.Fatal("fp nonvalue", value, e)
	}
}
