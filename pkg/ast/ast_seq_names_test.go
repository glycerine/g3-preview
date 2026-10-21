package ast

import "testing"

func TestASTSequenceSourceBuiltinNames(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	_, err = m.RegisterArrayPlugin()
	astTestMust(t, err)
	_, err = m.RegisterSequencePlugin()
	astTestMust(t, err)
	var operations, sorts ASTBuiltinNames
	astTestMust(t, m.GetSequenceOpNames(&operations, astTestSymbol(t, m, "QF_S")))
	astTestMust(t, m.GetSequenceSortNames(&sorts, astTestSymbol(t, m, "QF_S")))
	if len(operations.Items()) != 42 {
		t.Fatalf("sequence operation count: %d", len(operations.Items()))
	}
	for _, expected := range []struct {
		index int
		kind  int32
		name  string
	}{{0, 0, "seq.unit"}, {1, 1, "seq.empty"}, {38, 39, "str.indexof"}, {41, 43, "_re.unroll"}} {
		item := operations.Items()[expected.index]
		text, e := m.Symbols.String(item.Name)
		astTestMust(t, e)
		if item.Kind != expected.kind || text != expected.name {
			t.Fatalf("operation %d: kind=%d name=%q", expected.index, item.Kind, text)
		}
	}
	if len(sorts.Items()) != 4 {
		t.Fatalf("sequence sort count: %d", len(sorts.Items()))
	}
	for i, expected := range []struct {
		kind int32
		name string
	}{{0, "Seq"}, {1, "RegEx"}, {2, "String"}, {2, "StringSequence"}} {
		item := sorts.Items()[i]
		text, e := m.Symbols.String(item.Name)
		astTestMust(t, e)
		if item.Kind != expected.kind || text != expected.name {
			t.Fatalf("sort %d: kind=%d name=%q", i, item.Kind, text)
		}
	}
}
