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
