package ast

// Source: src/ast/array_decl_plugin.cpp:120-137,235-306,446-465.

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

func arrayDeclFixture(t *testing.T, m *ASTManager, n int) (FamilyID, ASTHandle, []ASTHandle) {
	fid, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	sorts := make([]ASTHandle, n+1)
	parameters := make([]ASTParameter, n+1)
	for i := range sorts {
		sorts[i] = b
		parameters[i], e = m.NewParameterAST(b)
		astTestMust(t, e)
	}
	s, e := m.MkFamilySort(fid, 0, parameters)
	astTestMust(t, e)
	astTestPin(t, m, s)
	for i := range parameters {
		astTestMust(t, m.DelParameter(&parameters[i]))
	}
	return fid, s, sorts
}
func TestASTArrayOriginalSelectStoreConstDeclarationsAndApplications(t *testing.T) {
	m := astTestManager(t)
	for _, n := range []int{1, 2, 18} {
		fid, s, sorts := arrayDeclFixture(t, m, n)
		domain := append([]ASTHandle{s}, sorts[:n]...)
		selectDecl, e := m.MkFamilyFuncDecl(fid, ASTArrayOpSelect, nil, domain, ASTHandle{})
		astTestMust(t, e)
		astTestPin(t, m, selectDecl)
		got, e := m.GetRange(selectDecl)
		astTestMust(t, e)
		if got != sorts[n] {
			t.Fatal("select range", got)
		}
		storeDecl, e := m.MkFamilyFuncDecl(fid, ASTArrayOpStore, nil, append(domain, sorts[n]), ASTHandle{})
		astTestMust(t, e)
		astTestPin(t, m, storeDecl)
		got, e = m.GetRange(storeDecl)
		astTestMust(t, e)
		if got != s {
			t.Fatal("store range", got)
		}
		param, e := m.NewParameterAST(s)
		astTestMust(t, e)
		constDecl, e := m.MkFamilyFuncDecl(fid, ASTArrayOpConst, []ASTParameter{param}, sorts[n:], ASTHandle{})
		astTestMust(t, e)
		astTestPin(t, m, constDecl)
		astTestMust(t, m.DelParameter(&param))
		rangeDecl, e := m.MkFamilyFuncDecl(fid, ASTArrayOpConst, nil, sorts[n:], s)
		astTestMust(t, e)
		astTestPin(t, m, rangeDecl)
		if rangeDecl != constDecl {
			t.Fatal("const explicit range fallback", rangeDecl, constDecl)
		}
		record := m.declAt(mustInternal(t, m, constDecl).slot)
		info, e := m.info(record.info.slot, record.info.generation)
		astTestMust(t, e)
		if !info.private || info.count != 1 {
			t.Fatal("private const-array sort parameter")
		}
		yes, e := m.MkTrue()
		astTestMust(t, e)
		array, e := m.MkApp(constDecl, []ASTHandle{yes})
		astTestMust(t, e)
		astTestPin(t, m, array)
		args := make([]ASTHandle, n+1)
		args[0] = array
		for i := 1; i < len(args); i++ {
			args[i] = yes
		}
		selected, e := m.MkFamilyApp(fid, ASTArrayOpSelect, nil, args)
		astTestMust(t, e)
		astTestPin(t, m, selected)
		stored, e := m.MkFamilyApp(fid, ASTArrayOpStore, nil, append(args, yes))
		astTestMust(t, e)
		astTestPin(t, m, stored)
		sd, e := m.GetDecl(stored)
		astTestMust(t, e)
		if sd != storeDecl {
			t.Fatal("store expression overload declaration identity")
		}
		arrayParameter, e := m.NewParameterAST(s)
		astTestMust(t, e)
		constantArray, e := m.MkFamilyApp(fid, ASTArrayOpConst, []ASTParameter{arrayParameter}, []ASTHandle{yes})
		astTestMust(t, e)
		astTestPin(t, m, constantArray)
		astTestMust(t, m.DelParameter(&arrayParameter))
		if constantArray != array {
			t.Fatal("const expression overload identity")
		}
		decl, e := m.GetDecl(selected)
		astTestMust(t, e)
		if decl != selectDecl {
			t.Fatal("expression overload declaration identity")
		}
	}
}
func mustInternal(t *testing.T, m *ASTManager, h ASTHandle) astNodeRef {
	r, e := m.internal(h)
	astTestMust(t, e)
	return r
}
func TestASTArrayDeclarationRejectsInvalidAritiesAndSorts(t *testing.T) {
	m := astTestManager(t)
	fid, s, sorts := arrayDeclFixture(t, m, 1)
	b := sorts[0]
	cases := []struct {
		op        int32
		domain    []ASTHandle
		rangeSort ASTHandle
	}{
		{ASTArrayOpSelect, []ASTHandle{s}, ASTHandle{}},
		{ASTArrayOpSelect, []ASTHandle{s, b, b}, ASTHandle{}},
		{ASTArrayOpStore, []ASTHandle{s, b}, ASTHandle{}},
		{ASTArrayOpStore, []ASTHandle{b, b, b}, ASTHandle{}},
		{ASTArrayOpConst, []ASTHandle{b}, b},
		{ASTArrayOpConst, nil, s},
	}
	for _, c := range cases {
		_, e := m.MkFamilyFuncDecl(fid, c.op, nil, c.domain, c.rangeSort)
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal("invalid array declaration accepted", c.op, e)
		}
	}
}

func TestASTArrayDeclarationsNormalizeCompatibleArithmeticDomains(t *testing.T) {
	m := astTestManager(t)
	fid, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	arithmetic, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	integer, e := m.MkFamilySort(arithmetic, 0, nil)
	astTestMust(t, e)
	real, e := m.MkFamilySort(arithmetic, 1, nil)
	astTestMust(t, e)
	params := make([]ASTParameter, 2)
	for i := range params {
		params[i], e = m.NewParameterAST(real)
		astTestMust(t, e)
	}
	s, e := m.MkFamilySort(fid, 0, params)
	astTestMust(t, e)
	astTestPin(t, m, s)
	for i := range params {
		astTestMust(t, m.DelParameter(&params[i]))
	}
	selectDecl, e := m.MkFamilyFuncDecl(fid, ASTArrayOpSelect, nil, []ASTHandle{s, integer}, ASTHandle{})
	astTestMust(t, e)
	astTestPin(t, m, selectDecl)
	domain, e := m.GetDomain(selectDecl, 1)
	astTestMust(t, e)
	if domain != real {
		t.Fatal("select must normalize the compatible index sort", domain)
	}
	storeDecl, e := m.MkFamilyFuncDecl(fid, ASTArrayOpStore, nil, []ASTHandle{s, integer, integer}, ASTHandle{})
	astTestMust(t, e)
	astTestPin(t, m, storeDecl)
	for i := uint32(1); i < 3; i++ {
		domain, e = m.GetDomain(storeDecl, i)
		astTestMust(t, e)
		if domain != real {
			t.Fatal("store normalized domain", i, domain)
		}
	}
	constant, e := m.MkFamilyFuncDecl(fid, ASTArrayOpConst, nil, []ASTHandle{integer}, s)
	astTestMust(t, e)
	astTestPin(t, m, constant)
	domain, e = m.GetDomain(constant, 0)
	astTestMust(t, e)
	if domain != integer {
		t.Fatal("const source preserves supplied domain", domain)
	}
	astTestMust(t, m.EnableIntRealCoercions(false))
	if _, e = m.MkFamilyFuncDecl(fid, ASTArrayOpSelect, nil, []ASTHandle{s, integer}, ASTHandle{}); !errors.Is(e, ErrASTDomain) {
		t.Fatal("disabled coercions accepted incompatible sorts", e)
	}
}
func TestASTArrayDeclarationLexicalSpillRefusalAndReuse(t *testing.T) {
	m := astTestManager(t)
	fid, s, sorts := arrayDeclFixture(t, m, 18)
	domain := append([]ASTHandle{s}, sorts[:18]...)
	baseline := m.Budget().Used()
	m.Budget().FailAfter(0)
	_, e := m.MkFamilyFuncDecl(fid, ASTArrayOpSelect, nil, domain, ASTHandle{})
	m.Budget().FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || m.Budget().Used() != baseline {
		t.Fatal("normalized domain spill refusal", e, baseline, m.Budget().Used())
	}
	decl, e := m.MkFamilyFuncDecl(fid, ASTArrayOpSelect, nil, domain, ASTHandle{})
	astTestMust(t, e)
	astTestPin(t, m, decl)
}
