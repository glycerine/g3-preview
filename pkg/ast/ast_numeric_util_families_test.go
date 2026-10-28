package ast

// Source: src/ast/arith_decl_plugin.cpp:641-645;
// src/ast/bv_decl_plugin.cpp:850-855.

import "testing"

func TestASTNumericUtilityFamilyConstruction(t *testing.T) {
	m := astTestManager(t)
	arith, e := m.ArithmeticFamilyID()
	astTestMust(t, e)
	// arith_util constructs a family identity without installing a plugin.
	if m.pluginAt(arith).present {
		t.Fatal("arith_util registered a declaration plugin")
	}
	same, e := m.ArithmeticFamilyID()
	astTestMust(t, e)
	if same != arith {
		t.Fatal("family identity changed")
	}
	registered, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	if registered != arith {
		t.Fatal("arithmetic plugin changed family", registered, arith)
	}
	bv, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	got, e := m.BitVectorUtilityFamilyID()
	astTestMust(t, e)
	if got != bv {
		t.Fatal(got, bv)
	}
	m.Budget().FailAfter(0)
	same, e = m.ArithmeticFamilyID()
	astTestMust(t, e)
	got, e = m.BitVectorUtilityFamilyID()
	astTestMust(t, e)
	m.Budget().FailAfter(-1)
	if same != arith || got != bv {
		t.Fatal("warm construction changed identities")
	}
}
