package cmd_context

// Source: src/cmd_context/simplify_cmd.cpp default term-rewrite execution.

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/rewriter"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

// Simplify rewrites one term with command-local theory-rewriter parameters.
// The returned reference is owned by the caller.
func (c *Context) Simplify(expression ast.ASTHandle, params *uparams.ParamsRef) (result ast.ASTRef, resultErr error) {
	if err := c.valid(); err != nil {
		return result, err
	}
	m, err := c.Manager()
	if err != nil {
		return result, err
	}
	value, err := rewriter.NewTheoryRewriter(m, c.config.Params, c.config.Globals, params)
	if err != nil {
		if value != nil {
			err = errors.Join(err, value.Close())
		}
		return result, err
	}
	defer func() { resultErr = errors.Join(resultErr, value.Close()) }()
	result, err = m.NewRef(ast.ASTHandle{})
	if err != nil {
		return result, err
	}
	if err = value.Rewrite(expression, result); err != nil {
		return ast.ASTRef{}, errors.Join(err, m.DelRef(&result))
	}
	return result, nil
}
