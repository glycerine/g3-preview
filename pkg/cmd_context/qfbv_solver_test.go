package cmd_context

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestCommandContextQFBVCombinedSolverProofResult(t *testing.T) {
	c := contextFixture(t)
	contextMust(t, c.SetLogic("QF_BV"))
	contextMust(t, c.SetBoolOption("produce-proofs", true))
	m, err := c.Manager()
	contextMust(t, err)
	sort, err := m.BitVectorSort(4)
	contextMust(t, err)
	_, err = c.DeclareFunction("x", nil, sort)
	contextMust(t, err)
	x, err := c.MakeApp("x", nil, nil, ast.ASTHandle{})
	contextMust(t, err)
	square, err := c.MakeApp("bvmul", []ast.ASTHandle{x, x}, nil, ast.ASTHandle{})
	contextMust(t, err)
	two, err := m.BitVectorNumeralUnsigned(2, 4)
	contextMust(t, err)
	constraint, err := m.MkEq(square, two)
	contextMust(t, err)
	contextMust(t, c.Assert(constraint, ""))
	if _, ok := c.solver.(*solver.CombinedSolver); !ok {
		t.Fatal("QF_BV did not select the combined solver")
	}
	status, err := c.CheckSat(nil)
	contextMust(t, err)
	if status != lbool.False {
		t.Fatal("modular-square contradiction", status)
	}
	proof, err := c.solver.Proof()
	contextMust(t, err)
	fact, err := m.GetProofFact(proof)
	contextMust(t, err)
	if !m.IsFalse(fact) {
		t.Fatal("missing contradiction proof")
	}
}
