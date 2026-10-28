package cmd_context

// Source: src/cmd_context/cmd_context.cpp result ownership and
// src/cmd_context/basic_cmds.cpp get-proof/get-unsat-core availability.

import (
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

// UnsatCore returns a borrowed copy of the most recent unsatisfiable result's
// tracked labels. The handles remain owned by the context result.
func (c *Context) UnsatCore() ([]ast.ASTHandle, error) {
	if err := c.valid(); err != nil {
		return nil, err
	}
	state, err := c.parameters.State()
	if err != nil {
		return nil, err
	}
	if !c.checked || c.status != lbool.False || !state.UnsatCore {
		return nil, fmt.Errorf("unsat core is not available")
	}
	return c.checkSatResult().UnsatCore()
}

// Proof returns the borrowed proof from the most recent unsatisfiable result.
// It remains owned by the context result and is invalidated by the next
// mutating solver operation.
func (c *Context) Proof() (ast.ASTHandle, error) {
	if err := c.valid(); err != nil {
		return ast.ASTHandle{}, err
	}
	state, err := c.parameters.State()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if !state.Proof || !c.checked || c.status != lbool.False {
		return ast.ASTHandle{}, fmt.Errorf("proof is not available")
	}
	proof, err := c.checkSatResult().Proof()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if proof.IsNull() {
		return ast.ASTHandle{}, fmt.Errorf("proof is not available")
	}
	return proof, nil
}
