package pdr

// Source: context::get_answer and context::mk_sat_answer in
// src/muz/pdr/pdr_context.cpp. The retained SMT-LIB query surface asks whether
// a relation has any tuple, so a successful declaration query has the closed
// existential answer true. General rule-trace reconstruction remains separate.

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

// Answer returns one owning reference for the latest successful relation-
// declaration query. The caller releases it through ASTManager.DelRef.
func (c *Context) Answer(queryDeclaration ast.ASTHandle) (ast.ASTRef, error) {
	if err := c.open(); err != nil {
		return ast.ASTRef{}, err
	}
	if c.lastResult != lbool.True || c.query == nil || c.query.head != queryDeclaration {
		return ast.ASTRef{}, ast.ErrASTDomain
	}
	truth, err := c.manager.MkTrue()
	if err != nil {
		return ast.ASTRef{}, err
	}
	return c.manager.NewRef(truth)
}
