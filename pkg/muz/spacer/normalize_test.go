package spacer

// Source: src/muz/spacer/spacer_util.cpp normalize.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
)

func normalizeMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func normalizePin(t *testing.T, manager *ast.ASTManager, value ast.ASTHandle) ast.ASTRef {
	t.Helper()
	ref, err := manager.NewRef(value)
	normalizeMust(t, err)
	t.Cleanup(func() { normalizeMust(t, manager.DelRef(&ref)) })
	return ref
}

func normalizeNumeral(t *testing.T, manager *ast.ASTManager, text string) ast.ASTHandle {
	t.Helper()
	service := manager.NumeralService()
	value, err := service.NewValue()
	normalizeMust(t, err)
	defer func() { normalizeMust(t, service.Del(&value)) }()
	normalizeMust(t, service.SetString(&value, text))
	result, err := manager.ArithmeticNumeral(&value, true)
	normalizeMust(t, err)
	normalizePin(t, manager, result)
	return result
}

func TestNormalizeFormulaCanonicalOrderAndBounds(t *testing.T) {
	command, err := cmd_context.NewStandaloneContext()
	normalizeMust(t, err)
	t.Cleanup(func() { normalizeMust(t, command.Close()) })
	manager, err := command.Manager()
	normalizeMust(t, err)
	globals, err := command.GlobalParameters()
	normalizeMust(t, err)
	boolean, err := manager.MkBoolSort()
	normalizeMust(t, err)
	pName, err := manager.Symbols.Intern("normalize-p")
	normalizeMust(t, err)
	qName, err := manager.Symbols.Intern("normalize-q")
	normalizeMust(t, err)
	p, err := manager.MkConst(pName, boolean)
	normalizeMust(t, err)
	q, err := manager.MkConst(qName, boolean)
	normalizeMust(t, err)
	notP, err := manager.MkNot(p)
	normalizeMust(t, err)
	implication, err := manager.MkOr(notP, q)
	normalizeMust(t, err)
	formula, err := manager.MkAndArray([]ast.ASTHandle{implication, p})
	normalizeMust(t, err)
	result, err := normalizeFormula(manager, globals.Context, globals, formula, true, false)
	normalizeMust(t, err)
	defer func() { normalizeMust(t, manager.DelRef(&result)) }()
	handle, err := manager.RefGet(result)
	normalizeMust(t, err)
	parts, err := manager.FlattenAnd([]ast.ASTHandle{handle})
	normalizeMust(t, err)
	if len(parts) != 2 {
		t.Fatal("normalized conjunction size", len(parts))
	}
	foundP, foundQ := false, false
	for _, part := range parts {
		foundP = foundP || part == p
		foundQ = foundQ || part == q
	}
	if !foundP || !foundQ {
		t.Fatal("normalized bounds", foundP, foundQ)
	}
}

func TestNormalizeFormulaIntegerThresholds(t *testing.T) {
	command, err := cmd_context.NewStandaloneContext()
	normalizeMust(t, err)
	t.Cleanup(func() { normalizeMust(t, command.Close()) })
	manager, err := command.Manager()
	normalizeMust(t, err)
	globals, err := command.GlobalParameters()
	normalizeMust(t, err)
	minusOne := normalizeNumeral(t, manager, "-1")
	zero := normalizeNumeral(t, manager, "0")
	integer, err := manager.GetSort(zero)
	normalizeMust(t, err)
	xName, err := manager.Symbols.Intern("normalize-x")
	normalizeMust(t, err)
	x, err := manager.MkConst(xName, integer)
	normalizeMust(t, err)
	family, err := manager.ArithmeticFamilyID()
	normalizeMust(t, err)
	leMinusOne, err := manager.MkFamilyApp(family, 2, nil, []ast.ASTHandle{x, minusOne})
	normalizeMust(t, err)
	result, err := normalizeFormula(manager, globals.Context, globals, leMinusOne, true, false)
	normalizeMust(t, err)
	defer func() { normalizeMust(t, manager.DelRef(&result)) }()
	geZero, err := manager.MkFamilyApp(family, 3, nil, []ast.ASTHandle{x, zero})
	normalizeMust(t, err)
	want, err := manager.MkNot(geZero)
	normalizeMust(t, err)
	got, err := manager.RefGet(result)
	normalizeMust(t, err)
	if got != want {
		t.Fatal("integer threshold normalization", got, want)
	}
}
