// Licensed under the MIT license.
package g3

import "testing"

func TestASTFamilyDeclarationDomainOverloadUsesActualPlugin(t *testing.T) {
	m := astTestManager(t)
	s, e := m.MkBoolSort()
	astTestMust(t, e)
	d, e := m.MkFamilyFuncDecl(0, ASTOpEq, nil, []ASTHandle{s, s}, ASTHandle{})
	astTestMust(t, e)
	if d.IsNull() {
		t.Fatal("basic equality declaration")
	}
	pattern, e := m.MkFamilyFuncDecl(2, 99, nil, []ASTHandle{s}, ASTHandle{})
	astTestMust(t, e)
	if pattern.IsNull() || m.nodeAt(pattern.slot).kind != ASTFuncDecl {
		t.Fatal("actual pattern domain plugin")
	}
	missing, e := m.MkFamilyFuncDecl(5, 0, nil, nil, ASTHandle{})
	astTestMust(t, e)
	if !missing.IsNull() {
		t.Fatal("source missing plugin returns null")
	}
}

func TestASTFuncDeclInfoPointerOverloadKeepsNullMetadata(t *testing.T) {
	m := astTestManager(t)
	s, e := m.MkBoolSort()
	astTestMust(t, e)
	info, e := m.NewFuncDeclInfo(NullFamilyID, -1, nil, ASTFuncFlags{}, false)
	astTestMust(t, e)
	defer m.DelFuncDeclInfo(&info)
	name := astTestSymbol(t, m, "pointer-info")
	plain, e := m.MkFuncDecl(name, nil, s, ASTFuncDeclInfo{})
	astTestMust(t, e)
	byRef, e := m.MkFuncDecl(name, nil, s, info)
	astTestMust(t, e)
	byPtr, e := m.MkFuncDeclWithInfo(name, nil, s, info)
	astTestMust(t, e)
	if plain != byRef || byPtr == plain {
		t.Fatal("source reference collapses is_null, pointer preserves it")
	}
	if m.declAt(byPtr.slot).info == (ASTFuncDeclInfo{}) {
		t.Fatal("pointer metadata not retained")
	}
}
