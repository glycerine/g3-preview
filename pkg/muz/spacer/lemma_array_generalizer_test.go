package spacer

// Source: src/muz/spacer/spacer_generalizers.cpp
// lemma_array_eq_generalizer pairwise contradiction strengthening.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/smtcore"
)

func TestArrayEqualityGeneralizerStrengthensContradictoryLiteral(t *testing.T) {
	manager, globals := newSpacerTestServices(t)
	owner, err := NewManager(manager, nil, 1, smtcore.SMTSolverConfig{Params: globals.Context, Globals: globals})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := owner.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	boolean, err := manager.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	predicateName, err := manager.Symbols.Intern("array-generalizer-predicate")
	if err != nil {
		t.Fatal(err)
	}
	predicate, err := manager.MkFuncDecl(predicateName, nil, boolean, ast.ASTFuncDeclInfo{})
	if err != nil {
		t.Fatal(err)
	}
	transformer, err := NewRulePredicateTransformer(owner, predicate, PropSolverConfig{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := transformer.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	arraySort, err := manager.MkArraySort(ast.SliceArguments([]ast.ASTHandle{boolean}), boolean)
	if err != nil {
		t.Fatal(err)
	}
	values := make([]ast.ASTHandle, 2)
	for index, text := range []string{"array-generalizer-a", "array-generalizer-b"} {
		name, nameErr := manager.Symbols.Intern(text)
		if nameErr != nil {
			t.Fatal(nameErr)
		}
		values[index], err = manager.MkConst(name, arraySort)
		if err != nil {
			t.Fatal(err)
		}
	}
	equality, err := manager.MkEq(values[0], values[1])
	if err != nil {
		t.Fatal(err)
	}
	disequality, err := manager.MkNot(equality)
	if err != nil {
		t.Fatal(err)
	}
	constants, err := transformer.collectArrayConstants(disequality)
	if err != nil {
		t.Fatal(err)
	}
	if len(constants) != 2 {
		t.Fatal("array constant collection", constants)
	}
	strengthened, applicable, err := transformer.strengthenArrayEqualities([]ast.ASTHandle{disequality})
	if err != nil {
		t.Fatal(err)
	}
	if !applicable {
		t.Fatal("array equality strengthening was not applicable")
	}
	if len(strengthened) != 1 || strengthened[0] != disequality {
		t.Fatal("array equality strengthening", strengthened, disequality)
	}
	truth, err := manager.MkTrue()
	if err != nil {
		t.Fatal(err)
	}
	unchanged, applicable, err := transformer.strengthenArrayEqualities([]ast.ASTHandle{truth})
	if err != nil {
		t.Fatal(err)
	}
	if applicable || len(unchanged) != 1 || unchanged[0] != truth {
		t.Fatal("array equality generalizer applied without array constants", applicable, unchanged)
	}
}
