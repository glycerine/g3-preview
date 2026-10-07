package ast

import (
	"testing"
)

func TestASTProofQuantInstantiationAndTheoryParameters(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	a, b := proofTestBool(t, m, "a"), proofTestBool(t, m, "b")
	f := proofTestApp(t, m, ASTOpOr, a, b)
	p, e := m.MkProofQuantInstantiation(f, []ASTHandle{a, b})
	astTestMust(t, e)
	proofTestFact(t, m, p, f)
	d, e := m.GetDecl(p)
	astTestMust(t, e)
	info := m.infoAt(m.declAt(d.slot).info.slot)
	if info.count != 2 {
		t.Fatal("ordered AST bindings", info.count)
	}
	for i, want := range []ASTHandle{a, b} {
		param, e := m.infoParameter(info, uint32(i))
		astTestMust(t, e)
		record, e := m.parameter(param)
		astTestMust(t, e)
		if record.Kind != ASTParameterAST || m.external(record.ast) != want {
			t.Fatal("source binding", i)
		}
	}
	custom, e := m.NewParameterInt(-7)
	astTestMust(t, e)
	defer m.DelParameter(&custom)
	out, e := m.MkProofTheoryLemma(2, f, []ASTHandle{p}, []ASTParameter{custom})
	astTestMust(t, e)
	proofTestFact(t, m, out, f)
	d, e = m.GetDecl(out)
	astTestMust(t, e)
	info = m.infoAt(m.declAt(d.slot).info.slot)
	if info.count != 2 {
		t.Fatal("family prefix", info.count)
	}
	param, e := m.infoParameter(info, 0)
	astTestMust(t, e)
	record, e := m.parameter(param)
	astTestMust(t, e)
	name, e := m.families.GetName(2)
	astTestMust(t, e)
	if record.Kind != ASTParameterSymbol || record.symbol != name {
		t.Fatal("actual family name")
	}
	param, e = m.infoParameter(info, 1)
	astTestMust(t, e)
	record, e = m.parameter(param)
	astTestMust(t, e)
	if record.Kind != ASTParameterInt || record.integer != -7 {
		t.Fatal("source parameter copy")
	}
	astTestMust(t, m.ToggleProofMode(ASTProofDisabled))
	out, e = m.MkProofQuantInstantiation(ASTHandle{}, []ASTHandle{{}})
	astTestMust(t, e)
	if !out.IsNull() {
		t.Fatal("disabled before parameter inspection")
	}
	out, e = m.MkProofTheoryLemma(-1, ASTHandle{}, nil, []ASTParameter{{}})
	astTestMust(t, e)
	if !out.IsNull() {
		t.Fatal("disabled before family/parameter inspection")
	}
}
