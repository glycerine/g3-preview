package ast

import "testing"

func TestASTSequenceUtilitySortAndExpressionRecognizers(t *testing.T) {
	m := astTestManager(t)
	str, err := m.MkString("")
	astTestMust(t, err)
	stringSort, err := m.GetSort(str)
	astTestMust(t, err)
	element, matched, err := m.SequenceSortElement(stringSort)
	astTestMust(t, err)
	isString, err := m.IsStringSort(stringSort)
	astTestMust(t, err)
	isChar, err := m.IsCharacterSort(element)
	astTestMust(t, err)
	if !matched || !isString || !isChar {
		t.Fatalf("String sort recognition: matched=%v string=%v char=%v", matched, isString, isChar)
	}
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	parameter, err := m.NewParameterAST(boolSort)
	astTestMust(t, err)
	sequenceSort, err := m.MkFamilySort(m.sequencePlugin.family, 0, []ASTParameter{parameter})
	astTestMust(t, err)
	astTestMust(t, m.DelParameter(&parameter))
	element, matched, err = m.SequenceSortElement(sequenceSort)
	astTestMust(t, err)
	isString, err = m.IsStringSort(sequenceSort)
	astTestMust(t, err)
	if !matched || element != boolSort || isString {
		t.Fatalf("generic sequence sort: matched=%v element=%v string=%v", matched, element, isString)
	}

	regex, err := m.MkSequenceToRegex(str)
	astTestMust(t, err)
	regexSort, err := m.GetSort(regex)
	astTestMust(t, err)
	regexSequence, matched, err := m.RegexSequenceSort(regexSort)
	astTestMust(t, err)
	if !matched || regexSequence != stringSort {
		t.Fatalf("regex sequence parameter: matched=%v sort=%v", matched, regexSequence)
	}

	genericEmpty, err := m.MkSequenceEmpty(sequenceSort)
	astTestMust(t, err)
	truth, err := m.MkTrue()
	astTestMust(t, err)
	unit, err := m.MkSequenceUnit(truth)
	astTestMust(t, err)
	concat, err := m.MkSequenceConcat(unit, genericEmpty)
	astTestMust(t, err)
	skolem, err := m.MkSequenceSkolem(astTestSymbol(t, m, "recognizer-skolem"), []ASTHandle{truth}, sequenceSort)
	astTestMust(t, err)
	for _, tc := range []struct {
		h    ASTHandle
		call func(ASTHandle) (bool, error)
	}{
		{str, m.IsSequenceEmpty},
		{genericEmpty, m.IsSequenceEmpty},
		{unit, m.IsSequenceUnit},
		{concat, m.IsSequenceConcat},
		{skolem, m.IsSequenceSkolem},
	} {
		yes, e := tc.call(tc.h)
		astTestMust(t, e)
		if !yes {
			t.Fatalf("sequence recognizer rejected %v", tc.h)
		}
	}
	if yes, err := m.IsSequenceConcat(truth); err != nil || yes {
		t.Fatalf("foreign expression recognized as concat: %v %v", yes, err)
	}
}
