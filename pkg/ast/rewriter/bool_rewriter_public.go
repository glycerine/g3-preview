// Source: src/ast/rewriter/bool_rewriter.h:86-105;
// src/ast/rewriter/bool_rewriter.cpp:23-87; src/model/model_evaluator.cpp:66-93.
package rewriter

import (
	"github.com/glycerine/g3/pkg/ast"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

// BoolRewriter is the original configured Boolean reducer. Traversal invokes its
// core through a Configuration; it does not traverse child expressions itself.
type BoolRewriter = boolRewriter

// NewBoolRewriter borrows the AST and parameter services. A nonnil partial owner
// returned with an error must be closed.
func NewBoolRewriter(m *ast.ASTManager, c *uparams.ParamsContext, g *uparams.GlobalParamsService, p *uparams.ParamsRef) (*BoolRewriter, error) {
	return newBoolRewriter(m, c, g, p)
}
func (rw *boolRewriter) Flat() (bool, error) {
	if e := rw.configuredOpen(); e != nil {
		return false, e
	}
	return rw.flat, nil
}
func (rw *boolRewriter) SetFlat(flat bool) error {
	if e := rw.configuredOpen(); e != nil {
		return e
	}
	rw.flat = flat
	return nil
}

// MkAppCore retains the source status and owning result-reference contract.
// RewriteFailed leaves result unchanged, as the original core does.
func (rw *boolRewriter) MkAppCore(d ast.ASTHandle, args ast.AppArguments, result ast.ASTRef) (RewriteStatus, error) {
	if e := rw.configuredOpen(); e != nil {
		return RewriteFailed, e
	}
	return rw.mkAppCoreArguments(d, args, result)
}

// SimplifyBooleanApplicationDefault is the concrete default-parameter
// bool_rewriter used by proof elimination. It owns no persistent parameter
// snapshot because the source constructs and destroys this reducer within one
// mk_app call.
func SimplifyBooleanApplicationDefault(m *ast.ASTManager, d ast.ASTHandle, args []ast.ASTHandle, result ast.ASTRef) error {
	if m == nil {
		return ast.ErrASTHandle
	}
	if _, err := m.ProofsEnabled(); err != nil {
		return err
	}
	rw := boolRewriter{manager: m, flat: true, localContextLimit: ^uint32(0)}
	return rw.mkApp(d, args, result)
}
