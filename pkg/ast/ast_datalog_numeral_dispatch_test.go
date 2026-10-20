package ast

import (
	"errors"
	"testing"
)

func TestASTDatalogOriginalNumeralSortDispatchAndValueClassification(t *testing.T) {
	m := astTestManager(t)
	finite, e := m.MkDatalogFiniteSort(astTestSymbol(t, m, "dispatch-finite"), 8)
	astTestMust(t, e)
	finiteValue, e := m.MkDatalogNumeral(7, finite)
	astTestMust(t, e)
	if yes, e := m.IsDatalogNumeral(finiteValue); e != nil || !yes {
		t.Fatal("finite numeral recognizer", yes, e)
	}
	if yes, e := m.IsValue(finiteValue); e != nil || !yes {
		t.Fatal("finite numeral value classification", yes, e)
	}
	if yes, e := m.IsUniqueValue(finiteValue); e != nil || !yes {
		t.Fatal("finite numeral unique-value classification", yes, e)
	}
	boolSort, e := m.MkBoolSort()
	astTestMust(t, e)
	zero, e := m.MkDatalogNumeral(0, boolSort)
	astTestMust(t, e)
	one, e := m.MkDatalogNumeral(1, boolSort)
	astTestMust(t, e)
	if !m.IsFalse(zero) || !m.IsTrue(one) {
		t.Fatal("Boolean numeral dispatch", zero, one)
	}
	if _, e = m.MkDatalogNumeral(2, boolSort); !errors.Is(e, ErrASTDomain) {
		t.Fatal("Boolean numeral range", e)
	}
	arithmetic, e := m.ArithmeticFamilyID()
	astTestMust(t, e)
	for kind := int32(0); kind <= 1; kind++ {
		sort, e := m.MkFamilySort(arithmetic, kind, nil)
		astTestMust(t, e)
		astTestPin(t, m, sort)
		if integer, matched, e := m.datalogArithmeticSort(sort); e != nil || !matched || integer != (kind == 1) {
			t.Fatal("arithmetic sort recognizer", kind, integer, matched, e)
		}
		if yes, e := m.IsDatalogFiniteSort(sort); e != nil || yes {
			t.Fatal("foreign finite-sort recognizer", kind, yes, e)
		}
		numeral, e := m.MkDatalogNumeral(6, sort)
		astTestMust(t, e)
		text, integer, matched, e := m.ArithmeticNumeralText(numeral)
		astTestMust(t, e)
		if !matched || text != "6" || integer != (kind == 1) {
			t.Fatal("arithmetic numeral dispatch", kind, text, integer, matched)
		}
	}
	bitVector, e := m.BitVectorUtilityFamilyID()
	astTestMust(t, e)
	width, e := m.NewParameterInt(8)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelParameter(&width)) }()
	bvSort, e := m.MkFamilySort(bitVector, 0, []ASTParameter{width})
	astTestMust(t, e)
	astTestPin(t, m, bvSort)
	bv, e := m.MkDatalogNumeral(257, bvSort)
	astTestMust(t, e)
	text, matched, e := m.BitVectorNumeralText(bv)
	astTestMust(t, e)
	if !matched || text != "1" {
		t.Fatal("bit-vector numeral dispatch", text, matched)
	}
	unsupported, e := m.MkUninterpretedSort(astTestSymbol(t, m, "dispatch-unsupported"), nil)
	astTestMust(t, e)
	astTestPin(t, m, unsupported)
	if _, e = m.MkDatalogNumeral(0, unsupported); !errors.Is(e, ErrASTDomain) {
		t.Fatal("unsupported numeral sort", e)
	}
}

func TestASTDatalogOriginalRecognizersAndEmptyBuiltinNames(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterDatalogPlugin()
	astTestMust(t, e)
	sort, e := m.MkDatalogFiniteSort(astTestSymbol(t, m, "recognizer"), 2)
	astTestMust(t, e)
	zero, e := m.MkDatalogNumeral(0, sort)
	astTestMust(t, e)
	one, e := m.MkDatalogNumeral(1, sort)
	astTestMust(t, e)
	lt, e := m.MkDatalogLT(zero, one)
	astTestMust(t, e)
	if yes, e := m.IsDatalogLT(lt); e != nil || !yes {
		t.Fatal("less-than recognizer", yes, e)
	}
	var names ASTBuiltinNames
	astTestMust(t, m.GetDatalogOpNames(&names))
	astTestMust(t, m.GetDatalogSortNames(&names))
	if len(names.Items()) != 0 {
		t.Fatal("Datalog builtin names", names.Items())
	}
}
