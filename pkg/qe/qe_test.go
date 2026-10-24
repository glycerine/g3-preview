package qe_test

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/qe"
	"github.com/glycerine/g3/pkg/util/cooperate"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

type fixture struct {
	m                *ast.ASTManager
	config           qe.Config
	integer, boolean ast.ASTHandle
	arrayFamily      ast.FamilyID
	budget           *arena.Budget
}

func setup(t *testing.T) *fixture {
	return setupProofs(t, false)
}

func setupProofs(t *testing.T, proofs bool) *fixture {
	t.Helper()
	c, err := cmd_context.NewStandaloneContext()
	must(t, err)
	if proofs {
		must(t, c.SetBoolOption("produce-proofs", true))
	}
	f := &fixture{}
	t.Cleanup(func() {
		must(t, c.Close())
		if f.budget != nil && f.budget.Used() != 0 {
			t.Errorf("QE fixture retained %d bytes", f.budget.Used())
		}
	})
	f.m, err = c.Manager()
	must(t, err)
	f.budget = f.m.Budget()
	g, err := c.GlobalParameters()
	must(t, err)
	coop, err := cooperate.NewCooperation(f.budget)
	must(t, err)
	t.Cleanup(func() { must(t, coop.Close()) })
	f.config = qe.Config{Context: g.Context, Globals: g, Cooperation: coop}
	f.integer, err = c.LookupSort("Int")
	must(t, err)
	f.boolean, err = f.m.MkBoolSort()
	must(t, err)
	f.arrayFamily, err = f.m.ArrayFamilyID()
	must(t, err)
	return f
}

func (f *fixture) pin(t *testing.T, h ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	r, err := f.m.NewRef(h)
	must(t, err)
	t.Cleanup(func() { must(t, f.m.DelRef(&r)) })
	return h
}
func (f *fixture) variable(t *testing.T, index uint32, sort ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	h, err := f.m.MkVar(index, sort)
	must(t, err)
	return f.pin(t, h)
}
func (f *fixture) constant(t *testing.T, name string, sort ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	s, err := f.m.Symbols.Intern(name)
	must(t, err)
	h, err := f.m.MkConst(s, sort)
	must(t, err)
	return f.pin(t, h)
}
func (f *fixture) numeral(t *testing.T, text string) ast.ASTHandle {
	t.Helper()
	h, err := f.m.ArithmeticIntegerNumeralText(text)
	must(t, err)
	return f.pin(t, h)
}
func (f *fixture) arraySort(t *testing.T, domains []ast.ASTHandle, rangeSort ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	h, err := f.m.MkArraySort(ast.SliceArguments(domains), rangeSort)
	must(t, err)
	return f.pin(t, h)
}
func (f *fixture) selectArray(t *testing.T, args ...ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	h, err := f.m.MkArrayAppArguments(f.arrayFamily, ast.ASTArrayOpSelect, ast.SliceArguments(args))
	must(t, err)
	return f.pin(t, h)
}
func (f *fixture) equality(t *testing.T, a, b ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	h, err := f.m.MkEq(a, b)
	must(t, err)
	return f.pin(t, h)
}
func (f *fixture) not(t *testing.T, h ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	n, err := f.m.MkNot(h)
	must(t, err)
	return f.pin(t, n)
}
func (f *fixture) refs(t *testing.T, hs ...ast.ASTHandle) []ast.ASTRef {
	t.Helper()
	result := make([]ast.ASTRef, 0, len(hs))
	for _, h := range hs {
		r, err := f.m.NewRef(h)
		must(t, err)
		result = append(result, r)
	}
	t.Cleanup(func() {
		for i := len(result) - 1; i >= 0; i-- {
			must(t, f.m.DelRef(&result[i]))
		}
	})
	return result
}
func (f *fixture) get(t *testing.T, r ast.ASTRef) ast.ASTHandle {
	t.Helper()
	h, err := f.m.RefGet(r)
	must(t, err)
	return h
}
func (f *fixture) arg(t *testing.T, h ast.ASTHandle, i uint32) ast.ASTHandle {
	t.Helper()
	a, err := f.m.GetArg(h, i)
	must(t, err)
	return a
}

func TestVariableTestCopiesSetComplementAndDeclarationBoundary(t *testing.T) {
	f := setup(t)
	indices := []uint32{1, 7, 7, ^uint32(0)}
	bound, err := qe.NewVariableSetTest(f.m, indices, true)
	must(t, err)
	t.Cleanup(func() { must(t, bound.Close()) })
	free, err := qe.NewVariableSetTest(f.m, indices, false)
	must(t, err)
	t.Cleanup(func() { must(t, free.Close()) })
	decls, err := qe.NewVariableDeclarationTest(f.m, 7)
	must(t, err)
	t.Cleanup(func() { must(t, decls.Close()) })
	indices[0] = 2
	for _, index := range []uint32{0, 1, 2, 6, 7, 8, ^uint32(0)} {
		h := f.variable(t, index, f.integer)
		selected, err := bound.IsVariable(h)
		must(t, err)
		complement, err := free.IsVariable(h)
		must(t, err)
		declaration, err := decls.IsVariable(h)
		must(t, err)
		want := index == 1 || index == 7 || index == ^uint32(0)
		if selected != want || complement == want || declaration != (index < 7) {
			t.Fatal(index, selected, complement, declaration)
		}
	}
	constant := f.constant(t, "selector-constant", f.integer)
	for _, selection := range []*qe.VariableTest{bound, free, decls} {
		selected, err := selection.IsVariable(constant)
		must(t, err)
		if selected {
			t.Fatal("constant selected as de Bruijn variable")
		}
	}
	copy := *bound
	if _, err := copy.IsVariable(constant); err == nil {
		t.Fatal("copied selector accepted")
	}
	must(t, bound.Close())
	if _, err := bound.IsVariable(constant); err == nil {
		t.Fatal("closed selector accepted")
	}
}
