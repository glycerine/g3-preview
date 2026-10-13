package ast

import "testing"

func TestASTSkolemDeclarationFlagForPatternInference(t *testing.T) {
	m := astTestManager(t)
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	info, err := m.NewFuncDeclInfo(NullFamilyID, -1, nil, ASTFuncFlags{Skolem: true}, false)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelFuncDeclInfo(&info)) }()
	name := astTestSymbol(t, m, "pattern-skolem")
	skolem, err := m.MkFuncDecl(name, []ASTHandle{b}, b, info)
	astTestMust(t, err)
	plain, err := m.MkFuncDecl(astTestSymbol(t, m, "pattern-plain"), []ASTHandle{b}, b, ASTFuncDeclInfo{})
	astTestMust(t, err)
	if got, e := m.IsSkolemDecl(skolem); e != nil || !got {
		t.Fatal("skolem declaration flag", got, e)
	}
	if got, e := m.IsSkolemDecl(plain); e != nil || got {
		t.Fatal("ordinary declaration flag", got, e)
	}
	if _, e := m.IsSkolemDecl(b); e != ErrASTDomain {
		t.Fatal("non-declaration must be rejected", e)
	}
}
