// Source: src/ast/arith_decl_plugin.cpp:511-559.
package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestASTArithmeticOriginalSortNamesAndPersistentConversion(t *testing.T) {
	for _, text := range []string{"NRA", "QF_NRA", "QF_UFNRA", "QF_LIA", "ALL", ""} {
		m := astTestManager(t)
		_, e := m.RegisterArithmeticPlugin()
		astTestMust(t, e)
		logic := util.Symbol{}
		if text != "" {
			logic = astTestSymbol(t, m, text)
		}
		var names ASTBuiltinNames
		defer names.Close()
		astTestMust(t, m.GetArithmeticSortNames(&names, logic))
		converted := text == "NRA" || text == "QF_NRA" || text == "QF_UFNRA"
		want := []string{"Int", "Real"}
		if converted {
			want = []string{"Real"}
		}
		if len(names.Items()) != len(want) {
			t.Fatal("original sort name count", text, names.Items())
		}
		for j, item := range names.Items() {
			got, e := m.Symbols.String(item.Name)
			astTestMust(t, e)
			kind := int32(0)
			if want[j] == "Int" {
				kind = 1
			}
			if got != want[j] || item.Kind != kind {
				t.Fatal("sort name/kind/order", text, j, got, item.Kind)
			}
		}
		if m.arithmeticPlugin.convertIntNumerals != converted {
			t.Fatal("original conversion mode", text)
		}
		// A later non-NRA query does not clear the source one-way flag.
		names.Reset()
		astTestMust(t, m.GetArithmeticSortNames(&names, astTestSymbol(t, m, "QF_LIA")))
		if m.arithmeticPlugin.convertIntNumerals != converted {
			t.Fatal("conversion mode must persist", text)
		}
		n, e := m.numerals.NewValue()
		astTestMust(t, e)
		h, e := m.ArithmeticNumeral(&n, true)
		astTestMust(t, e)
		astTestMust(t, m.numerals.Del(&n))
		s, e := m.GetSort(h)
		astTestMust(t, e)
		index := 1
		if converted {
			index = 0
		}
		if s != m.external(m.arithmeticPlugin.fields[index]) {
			t.Fatal("logic-selected numeral sort", text)
		}
	}
}

func TestASTArithmeticOriginalOperatorNamesOrderAndLogic(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	want := []string{"<=", ">=", "<", ">", "+", "-", "~", "*", "/", "div", "rem", "mod", "to_real", "to_int", "is_int", "abs", "^", "sin", "cos", "tan", "asin", "acos", "atan", "sinh", "cosh", "tanh", "asinh", "acosh", "atanh", "pi", "euler"}
	for _, text := range []string{"", "ALL", "QF_NRA", "QF_LIA", "NRA"} {
		logic := util.Symbol{}
		if text != "" {
			logic = astTestSymbol(t, m, text)
		}
		var names ASTBuiltinNames
		astTestMust(t, m.GetArithmeticOpNames(&names, logic))
		count := 16
		if text == "" || text == "ALL" {
			count = 31
		}
		if len(names.Items()) != count {
			t.Fatal("logic-dependent operator names", text, len(names.Items()))
		}
		for j, item := range names.Items() {
			got, e := m.Symbols.String(item.Name)
			astTestMust(t, e)
			kind := int32(j + 2)
			if got != want[j] || item.Kind != kind {
				t.Fatal("original alias/name/order", text, j, got, item.Kind, kind)
			}
		}
		names.Close()
	}
}

func TestASTArithmeticNameCallbacksAppendToExistingVector(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	var names ASTBuiltinNames
	defer names.Close()
	astTestMust(t, m.GetArithmeticSortNames(&names, util.Symbol{}))
	first, second := names.Items()[0], names.Items()[1]
	astTestMust(t, m.GetArithmeticSortNames(&names, astTestSymbol(t, m, "NRA")))
	if len(names.Items()) != 3 || names.Items()[0] != first || names.Items()[1] != second || names.Items()[2] != second {
		t.Fatal("original sort callback append prefix")
	}
	astTestMust(t, m.GetArithmeticOpNames(&names, astTestSymbol(t, m, "QF_LIA")))
	if len(names.Items()) != 19 || names.Items()[0] != first || names.Items()[2] != second {
		t.Fatal("operator callback erased caller prefix")
	}
}

func TestASTArithmeticNameVectorRefusalPreservesFlagAndPrefix(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	logic := astTestSymbol(t, m, "NRA")
	var names ASTBuiltinNames
	m.Budget().FailAfter(0)
	e = m.GetArithmeticSortNames(&names, logic)
	m.Budget().FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || !m.arithmeticPlugin.convertIntNumerals || len(names.Items()) != 0 {
		t.Fatal("source conversion flag precedes name vector growth", e)
	}
	defer names.Close()
	astTestMust(t, m.GetArithmeticOpNames(&names, util.Symbol{}))
	names.Close()
	for fault := int64(0); fault < 32; fault++ {
		m.Budget().FailAfter(fault)
		e = m.GetArithmeticOpNames(&names, util.Symbol{})
		m.Budget().FailAfter(-1)
		if e == nil {
			if len(names.Items()) != 31 {
				t.Fatal("complete name vector")
			}
			return
		}
		if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal("name vector growth refusal", fault, e)
		}
		for j, item := range names.Items() {
			if item.Kind != int32(j+2) {
				t.Fatal("original append publication prefix", fault, j, item.Kind)
			}
		}
		names.Close()
	}
	t.Fatal("name growth never succeeded")
}
