package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestElimUnusedVarsUFBVGroundAllUsedAndRemappedQuantifiers(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	satOK(t, e)
	v := make([]ast.ASTHandle, 3)
	for i := range v {
		v[i], e = m.MkVar(uint32(i), b)
		satOK(t, e)
		astTestPin(t, m, v[i])
	}
	truth, e := m.MkTrue()
	satOK(t, e)
	astTestPin(t, m, truth)
	makeQ := func(sorts []ast.ASTHandle, names []symbol.Symbol, body ast.ASTHandle, noPatterns []ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		h, e := m.MkQuantifier(ast.ASTQuantifierSpec{
			Forall: true, Sorts: sorts, Names: names, Body: body,
			NoPatterns: noPatterns,
		})
		satOK(t, e)
		astTestPin(t, m, h)
		return h
	}
	twoSorts := []ast.ASTHandle{b, b}
	twoNames := []symbol.Symbol{symbol.Numerical(10), symbol.Numerical(11)}
	ground := makeQ(twoSorts, twoNames, truth, []ast.ASTHandle{v[0]})
	out, e := ElimUnusedVarsUFBV(m, ground)
	satOK(t, e)
	got, e := m.RefGet(out)
	satOK(t, e)
	if got != truth {
		t.Fatal("ground body must bypass annotation variables", got, truth)
	}
	satOK(t, m.DelRef(&out))

	both, e := m.MkAnd(v[0], v[1])
	satOK(t, e)
	astTestPin(t, m, both)
	allUsed := makeQ(twoSorts, twoNames, both, nil)
	out, e = ElimUnusedVarsUFBV(m, allUsed)
	satOK(t, e)
	got, e = m.RefGet(out)
	satOK(t, e)
	if got != allUsed {
		t.Fatal("all-used quantifier identity", got, allUsed)
	}
	satOK(t, m.DelRef(&out))

	oneUsed := makeQ(twoSorts, twoNames, v[0], nil)
	wantOne := makeQ([]ast.ASTHandle{b}, []symbol.Symbol{twoNames[1]}, v[0], nil)
	beforeMapping, e := m.GetRefCount(v[0])
	satOK(t, e)
	out, e = ElimUnusedVarsUFBV(m, oneUsed)
	satOK(t, e)
	got, e = m.RefGet(out)
	satOK(t, e)
	if got != wantOne {
		t.Fatal("unused outer declaration must be removed with source reverse declaration order", got, wantOne)
	}
	satOK(t, m.DelRef(&out))
	afterMapping, e := m.GetRefCount(v[0])
	satOK(t, e)
	if afterMapping != beforeMapping {
		t.Fatal("temporary variable mapping retained an AST reference", beforeMapping, afterMapping)
	}
	d, e := m.MkFuncDecl(symbol.Numerical(12), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	satOK(t, e)
	astTestPin(t, m, d)
	annotation, e := m.MkApp(d, []ast.ASTHandle{v[0]})
	satOK(t, e)
	astTestPin(t, m, annotation)
	annotated := makeQ(twoSorts, twoNames, v[0], []ast.ASTHandle{annotation})
	wantAnnotated := makeQ([]ast.ASTHandle{b}, []symbol.Symbol{twoNames[1]}, v[0], []ast.ASTHandle{annotation})
	out, e = ElimUnusedVarsUFBV(m, annotated)
	satOK(t, e)
	got, e = m.RefGet(out)
	satOK(t, e)
	if got != wantAnnotated {
		t.Fatal("remaining no-pattern must be substituted with the body", got, wantAnnotated)
	}
	satOK(t, m.DelRef(&out))
	annotation2, e := m.MkApp(d, []ast.ASTHandle{v[1]})
	satOK(t, e)
	astTestPin(t, m, annotation2)
	threeSorts := []ast.ASTHandle{b, b, b}
	threeNames := []symbol.Symbol{symbol.Numerical(20), symbol.Numerical(21), symbol.Numerical(22)}
	withTwoNoPatterns := makeQ(threeSorts, threeNames, v[0], []ast.ASTHandle{annotation, annotation2})
	wantTwoNoPatterns := makeQ(twoSorts, threeNames[1:], v[0], []ast.ASTHandle{annotation, annotation2})
	firstBefore, e := m.GetRefCount(annotation)
	satOK(t, e)
	secondBefore, e := m.GetRefCount(annotation2)
	satOK(t, e)
	out, e = ElimUnusedVarsUFBV(m, withTwoNoPatterns)
	satOK(t, e)
	got, e = m.RefGet(out)
	satOK(t, e)
	if got != wantTwoNoPatterns {
		t.Fatal("two no-patterns must survive shared tmp substitution", got, wantTwoNoPatterns)
	}
	satOK(t, m.DelRef(&out))
	firstAfter, e := m.GetRefCount(annotation)
	satOK(t, e)
	secondAfter, e := m.GetRefCount(annotation2)
	satOK(t, e)
	if firstAfter != firstBefore || secondAfter != secondBefore {
		t.Fatal("no-pattern ref_buffer retained a temporary reference", firstBefore, firstAfter, secondBefore, secondAfter)
	}

	outerOnly := makeQ(twoSorts, twoNames, v[2], nil)
	out, e = ElimUnusedVarsUFBV(m, outerOnly)
	satOK(t, e)
	got, e = m.RefGet(out)
	satOK(t, e)
	if got != v[0] {
		t.Fatal("removing every binder must shift external variable index two to zero", got, v[0])
	}
	satOK(t, m.DelRef(&out))
}
