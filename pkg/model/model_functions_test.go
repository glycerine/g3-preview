// Source: src/model/model_core.cpp:21-42,68-109; src/model/model_core.h:45-64.
package model_test

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"testing"
)

func modelTestFunctionDecl(t *testing.T, m *ast.ASTManager, name string) ast.ASTHandle {
	t.Helper()
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, name), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, d)
	return d
}
func TestModelCoreFunctionRegistrationReplacementAndSamePointer(t *testing.T) {
	m := astTestManager(t)
	md := modelTestCore(t, m)
	d := modelTestFunctionDecl(t, m, "function-register")
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	astTestPin(t, m, yes)
	astTestPin(t, m, no)
	old := modelTestFunction(t, m, 1)
	astTestMust(t, old.SetElse(yes))
	before, e := m.GetRefCount(d)
	astTestMust(t, e)
	astTestMust(t, md.RegisterFunction(d, old))
	astTestMust(t, md.RegisterFunction(d, old))
	if md.NumFunctionDeclarations() != 1 {
		t.Fatal("same-pointer registration appended")
	}
	after, e := m.GetRefCount(d)
	astTestMust(t, e)
	if after != before+1 {
		t.Fatal("declaration owned once", before, after)
	}
	replacement := modelTestFunction(t, m, 1)
	astTestMust(t, replacement.SetElse(no))
	astTestMust(t, md.RegisterFunction(d, replacement))
	if _, e = old.Else(); e == nil {
		t.Fatal("replaced function graph still live")
	}
	got, e := md.FunctionInterpretation(d)
	astTestMust(t, e)
	if got != replacement {
		t.Fatal("replacement not published")
	}
	astTestMust(t, md.Close())
	if _, e = replacement.Else(); e == nil {
		t.Fatal("model did not destroy owned graph")
	}
	after, e = m.GetRefCount(d)
	astTestMust(t, e)
	if after != before {
		t.Fatal("declaration leaked", before, after)
	}
}
func TestModelCoreFunctionEvalMissingPartialAndUnregistration(t *testing.T) {
	m := astTestManager(t)
	md := modelTestCore(t, m)
	d := modelTestFunctionDecl(t, m, "function-eval")
	yes, e := m.MkTrue()
	astTestMust(t, e)
	out := astTestPin(t, m, yes)
	ok, e := md.EvalDeclaration(d, out)
	astTestMust(t, e)
	if ok {
		t.Fatal("absent function")
	}
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if got != yes {
		t.Fatal("absent function must preserve output")
	}
	fi := modelTestFunction(t, m, 1)
	astTestMust(t, md.RegisterFunction(d, fi))
	ok, e = md.EvalDeclaration(d, out)
	astTestMust(t, e)
	if ok {
		t.Fatal("partial function")
	}
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if !got.IsNull() {
		t.Fatal("present partial function must assign null")
	}
	astTestMust(t, fi.SetElse(yes))
	ok, e = md.EvalDeclaration(d, out)
	astTestMust(t, e)
	if !ok {
		t.Fatal("total function")
	}
	fn, e := md.Function(0)
	astTestMust(t, e)
	if fn != d {
		t.Fatal(fn)
	}
	astTestMust(t, md.UnregisterDeclaration(d))
	astTestMust(t, md.UnregisterDeclaration(d))
	if md.NumFunctionDeclarations() != 0 {
		t.Fatal("function declaration remains")
	}
	n, e := md.NumDeclarations()
	astTestMust(t, e)
	if n != 1 {
		t.Fatal("source historical m_decls must remain", n)
	}
	has, e := md.HasInterpretation(d)
	astTestMust(t, e)
	if has {
		t.Fatal("removed interpretation")
	}
	if _, e = fi.Else(); e == nil {
		t.Fatal("unregister did not destroy graph")
	}
}

func TestModelCoreFunctionTableGrowthRemovalReuseAndDerivedClose(t *testing.T) {
	m := astTestManager(t)
	derived, e := model.NewModel(m)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, derived.Close()) })
	md := derived.Core()
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	decls := make([]ast.ASTHandle, 32)
	graphs := make([]*model.FunctionInterpretation, 32)
	for i := range decls {
		decls[i] = modelTestFunctionDecl(t, m, fmt.Sprint("table-", i))
		graphs[i] = modelTestFunction(t, m, 1)
		fallback := yes
		if i&1 != 0 {
			fallback = no
		}
		astTestMust(t, graphs[i].SetElse(fallback))
		astTestMust(t, md.RegisterFunction(decls[i], graphs[i]))
	}
	if md.Functions.Len() != len(decls) {
		t.Fatal("native dictionary lost functions")
	}
	for i, d := range decls {
		got, e := md.FunctionInterpretation(d)
		astTestMust(t, e)
		if got != graphs[i] {
			t.Fatal("growth lost graph", i)
		}
		active, e := md.Function(uint32(i))
		astTestMust(t, e)
		if active != d {
			t.Fatal("source function declaration order", i)
		}
	}
	for i, d := range decls {
		if i&1 == 0 {
			astTestMust(t, md.UnregisterDeclaration(d))
		}
	}
	for i := 0; i < 16; i++ {
		d, e := md.Function(uint32(i))
		astTestMust(t, e)
		if d != decls[2*i+1] {
			t.Fatal("erase must preserve order", i)
		}
	}
	for i, d := range decls {
		if i&1 == 0 {
			replacement := modelTestFunction(t, m, 1)
			astTestMust(t, replacement.SetElse(no))
			astTestMust(t, md.RegisterFunction(d, replacement))
			graphs[i] = replacement
		}
	}
	if md.NumFunctionDeclarations() != 32 {
		t.Fatal("active function domain")
	}
	n, e := md.NumDeclarations()
	astTestMust(t, e)
	if n != 48 {
		t.Fatal("historical declarations must append on reregistration", n)
	}
	astTestMust(t, derived.Close())
	for i, fi := range graphs {
		if _, e = fi.Else(); e == nil {
			t.Fatal("derived model failed to close graph", i)
		}
	}
}

func TestModelCoreFunctionGuardsAndConstantDispatch(t *testing.T) {
	m := astTestManager(t)
	md := modelTestCore(t, m)
	d := modelTestFunctionDecl(t, m, "guard-function")
	foreign := modelTestFunction(t, astTestManager(t), 1)
	if e := md.RegisterFunction(d, foreign); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("foreign manager", e)
	}
	wrongArity := modelTestFunction(t, m, 2)
	if e := md.RegisterFunction(d, wrongArity); !errors.Is(e, ast.ErrASTDomain) {
		t.Fatal("wrong arity", e)
	}
	n, e := md.NumDeclarations()
	astTestMust(t, e)
	if n != 0 {
		t.Fatal("guard published declaration")
	}
	c := modelTestDecl(t, m, "constant-dispatch")
	yes, e := m.MkTrue()
	astTestMust(t, e)
	out := astTestPin(t, m, yes)
	ok, e := md.EvalDeclaration(c, out)
	astTestMust(t, e)
	if ok {
		t.Fatal("missing constant")
	}
	value, e := m.RefGet(out)
	astTestMust(t, e)
	if !value.IsNull() {
		t.Fatal("missing constant must clear output")
	}
	astTestMust(t, md.RegisterConstant(c, yes))
	ok, e = md.EvalDeclaration(c, out)
	astTestMust(t, e)
	if !ok {
		t.Fatal("registered constant")
	}
	astTestMust(t, md.UnregisterDeclaration(c))
	has, e := md.HasInterpretation(c)
	astTestMust(t, e)
	if has {
		t.Fatal("constant unregister dispatch")
	}
}
