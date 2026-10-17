// Source: src/ast/rewriter/fpa_rewriter.cpp:41-88,547-638.
package rewriter

import "github.com/glycerine/g3/pkg/ast"

// FPARewriter currently owns the retained classification dispatch. It borrows
// the AST manager and its floating-point numeral service.
type FPARewriter struct {
	self    *FPARewriter
	manager *ast.ASTManager
	family  ast.FamilyID
	closed  bool
}

func NewFPARewriter(m *ast.ASTManager) (*FPARewriter, error) {
	if m == nil {
		return nil, ast.ErrASTHandle
	}
	family, err := m.RegisterFloatingPointPlugin()
	if err != nil {
		return nil, err
	}
	r := &FPARewriter{manager: m, family: family}
	r.self = r
	return r, nil
}

func (r *FPARewriter) Close() error {
	if r == nil || r.self != r {
		return ast.ErrASTHandle
	}
	r.closed = true
	return nil
}

func (r *FPARewriter) MkAppCore(d ast.ASTHandle, args ast.AppArguments, out ast.ASTRef) (RewriteStatus, error) {
	if r == nil || r.self != r || r.closed {
		return RewriteFailed, ast.ErrASTHandle
	}
	family, err := r.manager.GetFamilyID(d)
	if err != nil {
		return RewriteFailed, err
	}
	if family != r.family {
		return RewriteFailed, ast.ErrASTDomain
	}
	kind, err := r.manager.GetDeclKind(d)
	if err != nil {
		return RewriteFailed, err
	}
	if kind < int32(ast.FPAIsNaN) || kind > int32(ast.FPAIsPositive) {
		return RewriteFailed, nil
	}
	if args.Count() != 1 {
		return RewriteFailed, ast.ErrASTDomain
	}
	arg, err := args.At(0)
	if err != nil {
		return RewriteFailed, err
	}
	found, value, err := r.manager.ClassifyFPANumeral(arg, ast.FPAClassificationKind(kind))
	if err != nil || !found {
		return RewriteFailed, err
	}
	var answer ast.ASTHandle
	if value {
		answer, err = r.manager.MkTrue()
	} else {
		answer, err = r.manager.MkFalse()
	}
	if err != nil {
		return RewriteFailed, err
	}
	return RewriteDone, r.manager.SetRef(out, answer)
}

// MkEqCore is SMT equality on floating-point expressions. Unlike fp.eq,
// paired NaNs compare equal and signed zeros remain distinct.
func (r *FPARewriter) MkEqCore(left, right ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if r == nil || r.self != r || r.closed {
		return RewriteFailed, ast.ErrASTHandle
	}
	found, equal, err := r.manager.EqualFPANumerals(left, right)
	if err != nil || !found {
		return RewriteFailed, err
	}
	var answer ast.ASTHandle
	if equal {
		answer, err = r.manager.MkTrue()
	} else {
		answer, err = r.manager.MkFalse()
	}
	if err != nil {
		return RewriteFailed, err
	}
	return RewriteDone, r.manager.SetRef(out, answer)
}
