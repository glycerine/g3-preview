package ast

import (
	"testing"

	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestASTSMT2StringLiteralSourceEncoding(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, out.Close()) }()
	for _, tc := range []struct {
		value string
		want  string
	}{
		{"", `""`},
		{`a"b`, `"a""b"`},
		{"\n", `"\n"`},
		{`\`, `"\\"`},
		{string([]byte{0x80}), `"\x80"`},
	} {
		literal, e := m.MkString(tc.value)
		paramsMust(t, e)
		decoded, matched, e := m.SequenceStringValue(literal)
		paramsMust(t, e)
		if !matched || decoded != tc.value {
			t.Fatalf("string round trip: matched=%v got=%q want=%q", matched, decoded, tc.value)
		}
		format, e := m.SMT2PPExprTree(literal, nil)
		paramsMust(t, e)
		paramsMust(t, m.PrintFormat(out, format, c, p, g))
		got, e := out.String()
		paramsMust(t, e)
		if got != tc.want {
			t.Fatalf("SMT2 string %q: %q, want %q", tc.value, got, tc.want)
		}
		paramsMust(t, out.ResetText())
	}
}

func TestASTSequenceStringValueClassification(t *testing.T) {
	m := astTestManager(t)
	first, err := m.MkString("alpha")
	astTestMust(t, err)
	again, err := m.MkString("alpha")
	astTestMust(t, err)
	second, err := m.MkString("beta")
	astTestMust(t, err)
	for _, h := range []ASTHandle{first, again, second} {
		value, e := m.IsValue(h)
		astTestMust(t, e)
		unique, e := m.IsUniqueValue(h)
		astTestMust(t, e)
		if !value || unique {
			t.Fatalf("string constant classification: value=%v unique=%v", value, unique)
		}
	}
	equal, err := m.AreEqual(first, again)
	astTestMust(t, err)
	distinctSame, err := m.AreDistinct(first, again)
	astTestMust(t, err)
	distinctDifferent, err := m.AreDistinct(first, second)
	astTestMust(t, err)
	if !equal || distinctSame || !distinctDifferent {
		t.Fatalf("string equality: equal=%v same-distinct=%v different-distinct=%v", equal, distinctSame, distinctDifferent)
	}
	empty, err := m.GetSomeValue(m.external(m.sequencePlugin.stringSort))
	astTestMust(t, err)
	kind, err := m.GetDeclKind(empty)
	astTestMust(t, err)
	value, err := m.IsValue(empty)
	astTestMust(t, err)
	unique, err := m.IsUniqueValue(empty)
	astTestMust(t, err)
	distinctEmptyString, err := m.AreDistinct(empty, first)
	astTestMust(t, err)
	decoded, matched, err := m.SequenceStringValue(empty)
	astTestMust(t, err)
	if kind != ASTSequenceOpStringConst || !matched || decoded != "" || !value || unique || !distinctEmptyString {
		t.Fatalf("empty sequence value: kind=%d value=%v unique=%v distinct-string=%v", kind, value, unique, distinctEmptyString)
	}
	regex, err := m.GetSomeValue(m.external(m.sequencePlugin.regex))
	astTestMust(t, err)
	kind, err = m.GetDeclKind(regex)
	astTestMust(t, err)
	regexEmpty, err := m.GetArg(regex, 0)
	astTestMust(t, err)
	emptyKind, err := m.GetDeclKind(regexEmpty)
	astTestMust(t, err)
	if kind != ASTSequenceOpToRegex || emptyKind != ASTSequenceOpStringConst {
		t.Fatalf("regular-expression some value: kind=%d child=%d", kind, emptyKind)
	}
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	element, err := m.NewParameterAST(boolSort)
	astTestMust(t, err)
	sequenceSort, err := m.MkFamilySort(m.sequencePlugin.family, 0, []ASTParameter{element})
	astTestMust(t, err)
	astTestMust(t, m.DelParameter(&element))
	genericEmpty, err := m.GetSomeValue(sequenceSort)
	astTestMust(t, err)
	genericKind, err := m.GetDeclKind(genericEmpty)
	astTestMust(t, err)
	if genericKind != ASTSequenceOpEmpty {
		t.Fatalf("generic sequence some value kind: %d", genericKind)
	}
	truth, err := m.MkTrue()
	astTestMust(t, err)
	falsity, err := m.MkFalse()
	astTestMust(t, err)
	truthUnit, err := m.MkSequenceUnit(truth)
	astTestMust(t, err)
	falseUnit, err := m.MkSequenceUnit(falsity)
	astTestMust(t, err)
	unitsDistinct, err := m.AreDistinct(truthUnit, falseUnit)
	astTestMust(t, err)
	emptyUnitDistinct, err := m.AreDistinct(genericEmpty, truthUnit)
	astTestMust(t, err)
	concat, err := m.MkSequenceConcat(truthUnit, genericEmpty)
	astTestMust(t, err)
	concatValue, err := m.IsValue(concat)
	astTestMust(t, err)
	concatUnique, err := m.IsUniqueValue(concat)
	astTestMust(t, err)
	if !unitsDistinct || !emptyUnitDistinct || !concatValue || concatUnique {
		t.Fatalf("sequence values: units-distinct=%v empty-unit=%v concat-value=%v concat-unique=%v", unitsDistinct, emptyUnitDistinct, concatValue, concatUnique)
	}
	stringConcat, err := m.MkSequenceConcat(first, second)
	astTestMust(t, err)
	stringKind, err := m.GetDeclKind(stringConcat)
	astTestMust(t, err)
	stringValue, err := m.IsValue(stringConcat)
	astTestMust(t, err)
	if stringKind != ASTSequenceOpConcat || !stringValue {
		t.Fatalf("string concat: kind=%d value=%v", stringKind, stringValue)
	}
}
