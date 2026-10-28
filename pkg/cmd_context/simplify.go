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
	result, _, resultErr = c.SimplifyWithProof(expression, params, false)
	return result, resultErr
}

// SimplifyWithProof optionally returns the rewrite proof in a second caller-
// owned reference. Proof production requires a proof-enabled AST manager.
func (c *Context) SimplifyWithProof(expression ast.ASTHandle, params *uparams.ParamsRef, produceProof bool) (result, proof ast.ASTRef, resultErr error) {
	if err := c.valid(); err != nil {
		return result, proof, err
	}
	m, err := c.Manager()
	if err != nil {
		return result, proof, err
	}
	value, err := rewriter.NewTheoryRewriter(m, c.config.Params, c.config.Globals, params)
	if err != nil {
		if value != nil {
			err = errors.Join(err, value.Close())
		}
		return result, proof, err
	}
	defer func() { resultErr = errors.Join(resultErr, value.Close()) }()
	result, err = m.NewRef(ast.ASTHandle{})
	if err != nil {
		return result, proof, err
	}
	published := false
	defer func() {
		if !published {
			if proof != (ast.ASTRef{}) {
				resultErr = errors.Join(resultErr, m.DelRef(&proof))
			}
			resultErr = errors.Join(resultErr, m.DelRef(&result))
		}
	}()
	if produceProof {
		proof, err = m.NewRef(ast.ASTHandle{})
		if err != nil {
			return result, proof, err
		}
		err = value.RewriteWithProof(expression, result, proof)
	} else {
		err = value.Rewrite(expression, result)
	}
	if err != nil {
		return result, proof, err
	}
	published = true
	return result, proof, nil
}
