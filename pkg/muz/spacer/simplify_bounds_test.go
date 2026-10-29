package spacer

// Source: src/muz/spacer/spacer_util.cpp simplify_bounds_new composition of
// propagate-values and propagate-inequalities.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func simplifyBoundsMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSimplifyBoundsRunsValuesThenInequalities(t *testing.T) {
	manager, globals := newSpacerTestServices(t)
	var err error
	boolean, err := manager.MkBoolSort()
	simplifyBoundsMust(t, err)
	pName, err := manager.Symbols.Intern("simplify-bounds-p")
	simplifyBoundsMust(t, err)
	qName, err := manager.Symbols.Intern("simplify-bounds-q")
	simplifyBoundsMust(t, err)
	p, err := manager.MkConst(pName, boolean)
	simplifyBoundsMust(t, err)
	q, err := manager.MkConst(qName, boolean)
	simplifyBoundsMust(t, err)
	notP, err := manager.MkNot(p)
	simplifyBoundsMust(t, err)
	implication, err := manager.MkOr(notP, q)
	simplifyBoundsMust(t, err)
	inputs := make([]ast.ASTRef, 2)
	inputs[0], err = manager.NewRef(p)
	simplifyBoundsMust(t, err)
	inputs[1], err = manager.NewRef(implication)
	simplifyBoundsMust(t, err)

	result, err := simplifyBounds(manager, globals.Context, globals, inputs)
	simplifyBoundsMust(t, err)
	if len(result) != 2 {
		t.Fatal("propagate-values output size", len(result))
	}
	foundP, foundQ := false, false
	for i := range result {
		handle, err := manager.RefGet(result[i])
		simplifyBoundsMust(t, err)
		foundP = foundP || handle == p
		foundQ = foundQ || handle == q
		simplifyBoundsMust(t, manager.DelRef(&result[i]))
	}
	if !foundP || !foundQ {
		t.Fatal("propagate-values did not simplify implication", foundP, foundQ)
	}
}
