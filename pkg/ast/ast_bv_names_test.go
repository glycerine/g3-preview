// Source: src/ast/bv_decl_plugin.cpp:672-743; src/ast/bv_decl_plugin.h:28-105.
package ast

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTBitVectorOriginalNameCallbacks(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	want := []string{"bit1", "bit0", "bvneg", "bvadd", "bvsub", "bvmul", "bvsdiv", "bvudiv", "bvsrem", "bvurem", "bvsmod", "bvule", "bvsle", "bvuge", "bvsge", "bvult", "bvslt", "bvugt", "bvsgt", "bvand", "bvor", "bvnot", "bvxor", "bvnand", "bvnor", "bvxnor", "concat", "sign_extend", "zero_extend", "extract", "repeat", "bvredor", "bvredand", "bvcomp", "bvshl", "bvlshr", "bvashr", "rotate_left", "rotate_right", "bvumul_noovfl", "bvsmul_noovfl", "bvsmul_noudfl", "bvsdiv0", "bvudiv0", "bvsrem0", "bvurem0", "bvsmod0", "bvsdiv_i", "bvudiv_i", "bvsrem_i", "bvurem_i", "bvsmod_i", "ext_rotate_left", "ext_rotate_right", "int2bv", "bv2int", "bv2nat", "mkbv"}
	extendedKinds := []int32{52, 53, 54, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 50, 51, 57, 58, 58, 56}
	for _, logicText := range []string{"", "ALL", "QF_BV", "QF_UFBV", "NRA"} {
		logic := usymbol.Symbol{}
		if logicText != "" {
			logic = astTestSymbol(t, m, logicText)
		}
		var names ASTBuiltinNames
		extended := logicText == "" || logicText == "ALL"
		astTestMust(t, m.GetBitVectorSortNames(&names, logic))
		count := 1
		if extended {
			count = 2
		}
		if len(names.Items()) != count {
			t.Fatal("sort aliases", logicText, names.Items())
		}
		for i, item := range names.Items() {
			text, e := m.Symbols.String(item.Name)
			astTestMust(t, e)
			want := "BitVec"
			if extended && i == 0 {
				want = "bv"
			}
			if text != want || item.Kind != 0 {
				t.Fatal("source sort name/kind/order", logicText, text, item.Kind)
			}
		}
		names.Reset()
		astTestMust(t, m.GetBitVectorOpNames(&names, logic))
		count = 39
		if extended {
			count = 58
		}
		if len(names.Items()) != count {
			t.Fatal("operator count", logicText, len(names.Items()))
		}
		for i, item := range names.Items() {
			text, e := m.Symbols.String(item.Name)
			astTestMust(t, e)
			kind := int32(i + 1)
			if i >= 11 && i < 39 {
				kind += 10
			}
			if i >= 39 {
				kind = extendedKinds[i-39]
			}
			if text != want[i] || item.Kind != kind {
				t.Fatal("original operator alias/kind/order", logicText, i, text, item.Kind, kind)
			}
		}
		names.Close()
	}
}

func TestASTBitVectorNameNativeGrowthAndAppend(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	var names ASTBuiltinNames
	defer names.Close()
	astTestMust(t, m.GetBitVectorOpNames(&names, usymbol.Symbol{})) // warm literals
	names.Close()
	astTestMust(t, m.GetBitVectorSortNames(&names, usymbol.Symbol{})) // warm sort literals
	names.Close()
	before := m.Budget().Used()
	m.Budget().FailAfter(0)
	e = m.GetBitVectorOpNames(&names, usymbol.Symbol{})
	var first ASTBuiltinName
	if e == nil {
		first = names.Items()[0]
		e = m.GetBitVectorSortNames(&names, usymbol.Symbol{})
	}
	m.Budget().FailAfter(-1)
	astTestMust(t, e)
	if len(names.Items()) != 60 || names.Items()[0] != first || m.Budget().Used() != before {
		t.Fatal("native name growth and append", len(names.Items()), m.Budget().Used(), before)
	}
}
