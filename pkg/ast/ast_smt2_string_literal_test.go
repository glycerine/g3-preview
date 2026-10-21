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
	if kind != ASTSequenceOpEmpty || !value || unique || distinctEmptyString {
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
	if kind != ASTSequenceOpToRegex || emptyKind != ASTSequenceOpEmpty {
		t.Fatalf("regular-expression some value: kind=%d child=%d", kind, emptyKind)
	}
}
