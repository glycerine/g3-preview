package ast

import (
	"testing"

	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestASTSMT2LetSourceLevelGrouping(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, out.Close()) }()
	first, err := m.FormatString("left")
	paramsMust(t, err)
	second, err := m.FormatString("right")
	paramsMust(t, err)
	alias, err := m.FormatString("a!1")
	paramsMust(t, err)
	body, err := m.FormatString("a!2")
	paramsMust(t, err)
	for _, tc := range []struct {
		bindings []smt2LetBinding
		want     string
	}{
		{nil, "a!2"},
		{[]smt2LetBinding{{"a!1", first, 0}}, "(let ((a!1 left))\n  a!2)"},
		{[]smt2LetBinding{{"a!1", first, 0}, {"a!2", second, 0}}, "(let ((a!1 left) (a!2 right))\n  a!2)"},
		{[]smt2LetBinding{{"a!1", first, 0}, {"a!2", alias, 1}}, "(let ((a!1 left))\n(let ((a!2 a!1))\n  a!2))"},
	} {
		format, e := m.smt2PPLet(body, tc.bindings)
		paramsMust(t, e)
		paramsMust(t, m.PrintFormat(out, format, c, p, g))
		got, e := out.String()
		paramsMust(t, e)
		if got != tc.want {
			t.Fatalf("let format %q, want %q", got, tc.want)
		}
		paramsMust(t, out.ResetText())
	}
}
