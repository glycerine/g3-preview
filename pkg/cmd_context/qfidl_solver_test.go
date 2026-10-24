package cmd_context

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestCommandContextQFIDLCombinedSolverProofResult(t *testing.T) {
	c := contextFixture(t)
	contextMust(t, c.SetLogic("QF_IDL"))
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
	eq, err := c.manager.MkEq(x, zero)
	contextMust(t, err)
	contextMust(t, c.Assert(eq, ""))
	lower, err := c.MakeApp(">=", []ast.ASTHandle{x, one}, nil, ast.ASTHandle{})
	contextMust(t, err)
	contextMust(t, c.Assert(lower, ""))
	if _, ok := c.solver.(*solver.CombinedSolver); !ok {
		t.Fatal("QF_IDL did not select the combined solver")
	}
	status, err := c.CheckSat(nil)
	contextMust(t, err)
	if status != lbool.False {
		t.Fatal("combined strategy failed", status)
	}
	proof, err := c.solver.Proof()
	contextMust(t, err)
	fact, err := c.manager.GetProofFact(proof)
	contextMust(t, err)
	if !c.manager.IsFalse(fact) {
		t.Fatal("missing contradiction proof")
	}
}
