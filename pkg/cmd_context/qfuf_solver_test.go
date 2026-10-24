package cmd_context

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestCommandContextQFUFCombinedSolverProofResult(t *testing.T) {
	c := contextFixture(t)
	contextMust(t, c.SetLogic("QF_UF"))
	contextMust(t, c.SetBoolOption("produce-proofs", true))
	u, err := c.DeclareSort("U")
	contextMust(t, err)
	for _, name := range []string{"a", "b"} {
		_, err = c.DeclareFunction(name, nil, u)
		contextMust(t, err)
	}
	_, err = c.DeclareFunction("f", []ast.ASTHandle{u}, u)
	contextMust(t, err)
	a, err := c.MakeApp("a", nil, nil, ast.ASTHandle{})
	contextMust(t, err)
	b, err := c.MakeApp("b", nil, nil, ast.ASTHandle{})
	contextMust(t, err)
	eq, err := c.MakeApp("=", []ast.ASTHandle{a, b}, nil, ast.ASTHandle{})
	contextMust(t, err)
	contextMust(t, c.Assert(eq, ""))
	fa, err := c.MakeApp("f", []ast.ASTHandle{a}, nil, ast.ASTHandle{})
	contextMust(t, err)
	fb, err := c.MakeApp("f", []ast.ASTHandle{b}, nil, ast.ASTHandle{})
	contextMust(t, err)
	diff, err := c.MakeApp("distinct", []ast.ASTHandle{fa, fb}, nil, ast.ASTHandle{})
	contextMust(t, err)
	contextMust(t, c.Assert(diff, ""))
	if _, ok := c.solver.(*solver.CombinedSolver); !ok {
		t.Fatal("QF_UF did not select combined solver")
	}
	status, err := c.CheckSat(nil)
	contextMust(t, err)
	if status != lbool.False {
		t.Fatal("congruence contradiction", status)
	}
	proof, err := c.solver.Proof()
	contextMust(t, err)
	fact, err := c.manager.GetProofFact(proof)
	contextMust(t, err)
	if !c.manager.IsFalse(fact) {
		t.Fatal("missing contradiction proof")
	}
}
