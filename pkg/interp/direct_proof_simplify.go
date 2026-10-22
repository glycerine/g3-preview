// Source checkpoint: src/interp/iz3proof_itp.cpp generic simplify_rec cases.
package interp

import (
	"errors"
	"strings"

	"github.com/glycerine/g3/pkg/ast"
)

func (p *DirectProofInterpolator) isPlaceholder(term ast.ASTHandle) (bool, error) {
	op, err := p.base.manager.Operator(term)
	if err != nil || op != OperatorUninterpreted {
		return false, err
	}
	declaration, err := p.base.manager.Symbol(term)
	if err != nil {
		return false, err
	}
	name, err := p.base.manager.StringOfSymbol(declaration)
	return strings.HasPrefix(name, "@p") && len(name) > 2, err
}

func (p *DirectProofInterpolator) simplifyProofTerm(expression ast.ASTHandle) (ast.ASTHandle, error) {
	memo := make(map[ast.ASTHandle]ast.ASTHandle)
	var simplify func(ast.ASTHandle) (ast.ASTHandle, error)
	simplify = func(current ast.ASTHandle) (ast.ASTHandle, error) {
		if result, found := memo[current]; found {
			return result, nil
		}
		op, err := p.base.manager.Operator(current)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		var declaration ast.ASTHandle
		if op == OperatorUninterpreted {
			declaration, err = p.base.manager.Symbol(current)
			if err != nil {
				return ast.ASTHandle{}, err
			}
			if declaration == p.concat {
				memo[current] = current
				return current, nil
			}
		}
		args, err := p.base.manager.Args(current)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		hasPlaceholder := false
		for i := range args {
			if declaration == p.rotateSum && i == 0 {
				continue
			}
			args[i], err = simplify(args[i])
			if err != nil {
				return ast.ASTHandle{}, err
			}
			placeholder, e := p.isPlaceholder(args[i])
			if e != nil {
				return ast.ASTHandle{}, e
			}
			hasPlaceholder = hasPlaceholder || placeholder
		}
		var result ast.ASTHandle
		switch {
		case op == OperatorEqual && len(args) == 2 && args[0] == args[1]:
			result, err = p.base.manager.Make(OperatorTrue)
		case op == OperatorAnd:
			result, err = p.base.manager.MkAndSlice(args)
		case op == OperatorOr:
			result, err = p.base.manager.MkOrSlice(args)
		case op == OperatorIDiv && len(args) == 2:
			var ref ast.ASTRef
			ref, err = p.base.manager.MakeIDiv(args[0], args[1])
			if err == nil {
				result, err = p.base.manager.ast.RefGet(ref)
				if err == nil {
					p.owned = append(p.owned, ref)
				} else {
					err = errors.Join(err, p.base.manager.ast.DelRef(&ref))
				}
			}
		case op == OperatorUninterpreted && !hasPlaceholder && (declaration == p.rotateSum || declaration == p.sum || declaration == p.leqToEq || declaration == p.eqToLeq || declaration == p.excludedMid):
			return ast.ASTHandle{}, ProofError{}
		default:
			result, err = p.base.manager.Clone(current, args)
		}
		if err != nil {
			return ast.ASTHandle{}, err
		}
		memo[current] = result
		return result, nil
	}
	return simplify(expression)
}

func (p *DirectProofInterpolator) SimplifyProofTerm(expression ast.ASTHandle) (ast.ASTHandle, error) {
	if err := p.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	result, err := p.simplifyProofTerm(expression)
	return p.keep(result, err)
}
