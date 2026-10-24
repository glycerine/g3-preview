package cmd_context

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestCommandContextQFAUFLIACombinedSolverProofResult(t *testing.T) {
	c := contextFixture(t)
	contextMust(t, c.SetLogic("QF_AUFLIA"))
	contextMust(t, c.SetBoolOption("produce-proofs", true))
	integer, err := c.LookupSort("Int")
	contextMust(t, err)
	_, err = c.DeclareFunction("x", nil, integer)
	contextMust(t, err)
	x, err := c.MakeApp("x", nil, nil, ast.ASTHandle{})
	contextMust(t, err)
	zero, err := c.manager.ArithmeticIntegerNumeralText("0")
	contextMust(t, err)
	one, err := c.manager.ArithmeticIntegerNumeralText("1")
	contextMust(t, err)
	equality, err := c.manager.MkEq(x, zero)
	contextMust(t, err)
	contextMust(t, c.Assert(equality, ""))
	lower, err := c.MakeApp(">=", []ast.ASTHandle{x, one}, nil, ast.ASTHandle{})
	contextMust(t, err)
	contextMust(t, c.Assert(lower, ""))
	if _, ok := c.solver.(*solver.CombinedSolver); !ok {
		t.Fatal("QF_AUFLIA did not select the combined solver")
	}
	status, err := c.CheckSat(nil)
	contextMust(t, err)
	if status != lbool.False {
		t.Fatal("combined strategy failed to derive the contradiction", status)
	}
	proof, err := c.solver.Proof()
	contextMust(t, err)
	fact, err := c.manager.GetProofFact(proof)
	contextMust(t, err)
	if !c.manager.IsFalse(fact) {
		t.Fatal("combined solver did not retain its contradiction proof")
	}
}

func TestCommandContextQFAUFLIATypedArrayFallbackAndAssumptionRouting(t *testing.T) {
	c := contextFixture(t)
	contextMust(t, c.SetLogic("QF_AUFLIA"))
	contextMust(t, c.SetBoolOption("auto_config", false))
	u, err := c.DeclareSort("U")
	contextMust(t, err)
	boolean, err := c.LookupSort("Bool")
	contextMust(t, err)
	array, err := c.manager.MkArraySort(ast.SliceArguments([]ast.ASTHandle{u}), boolean)
	contextMust(t, err)
	_, err = c.DeclareFunction("a", nil, array)
	contextMust(t, err)
	f, err := c.DeclareFunction("f", []ast.ASTHandle{u}, boolean)
	contextMust(t, err)
	_, err = c.DeclareFunction("p", nil, boolean)
	contextMust(t, err)
	a, err := c.MakeApp("a", nil, nil, ast.ASTHandle{})
	contextMust(t, err)
	p, err := c.MakeApp("p", nil, nil, ast.ASTHandle{})
	contextMust(t, err)
	asArray, err := c.manager.MkAsArray(f)
	contextMust(t, err)
	equality, err := c.manager.MkEq(a, asArray)
	contextMust(t, err)
	contextMust(t, c.Assert(equality, ""))
	status, err := c.CheckSat(nil)
	contextMust(t, err)
	if status != lbool.True {
		t.Fatal("nonincremental strategy failed", status)
	}
	contextMust(t, c.Push(1))
	status, err = c.CheckSat(nil)
	contextMust(t, err)
	if status != lbool.True {
		t.Fatal("incremental unknown did not fall back to strategy", status)
	}
	status, err = c.CheckSat([]ast.ASTHandle{p})
	contextMust(t, err)
	if status != lbool.Undef {
		t.Fatal("assumptions must select the direct incremental backend", status)
	}
	status, err = c.CheckSat(nil)
	contextMust(t, err)
	if status != lbool.True {
		t.Fatal("fallback result selection did not recover after assumptions", status)
	}
	contextMust(t, c.Pop(1))
	status, err = c.CheckSat(nil)
	contextMust(t, err)
	if status != lbool.True {
		t.Fatal("fallback did not survive pop", status)
	}
}
