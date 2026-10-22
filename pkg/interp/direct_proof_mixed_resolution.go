// Source checkpoint: src/interp/iz3proof_itp.cpp mixed contradiction
// recognition and resolvent collection.
package interp

import "github.com/glycerine/g3/pkg/ast"

func (p *DirectProofInterpolator) isNegativeEquality(term ast.ASTHandle) (bool, error) {
	op, err := p.base.manager.Operator(term)
	if err != nil || op != OperatorNot {
		return false, err
	}
	body, err := p.base.manager.Arg(term, 0)
	if err != nil {
		return false, err
	}
	op, err = p.base.manager.Operator(body)
	return op == OperatorEqual || op == OperatorIff, err
}

func (p *DirectProofInterpolator) countNegativeEqualities(resolvent []ast.ASTHandle) (int, error) {
	count := 0
	for _, entry := range resolvent {
		literal, err := p.base.manager.Arg(entry, 1)
		if err != nil {
			return 0, err
		}
		negative, err := p.isNegativeEquality(literal)
		if err != nil {
			return 0, err
		}
		if negative {
			count++
		}
	}
	return count, nil
}

func (p *DirectProofInterpolator) contraPivot(target, interpolation ast.ASTHandle) (ast.ASTHandle, bool, error) {
	op, err := p.base.manager.Operator(interpolation)
	if err != nil || op != OperatorAnd {
		return ast.ASTHandle{}, false, err
	}
	count, err := p.base.manager.NumArgs(interpolation)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	for i := uint32(1); i < count; i++ {
		entry, e := p.base.manager.Arg(interpolation, i)
		if e != nil {
			return ast.ASTHandle{}, false, e
		}
		entryOp, e := p.base.manager.Operator(entry)
		if e != nil {
			return ast.ASTHandle{}, false, e
		}
		if entryOp != OperatorUninterpreted {
			break
		}
		declaration, e := p.base.manager.Symbol(entry)
		if e != nil {
			return ast.ASTHandle{}, false, e
		}
		if declaration != p.contra {
			break
		}
		literal, e := p.base.manager.Arg(entry, 1)
		if e != nil {
			return ast.ASTHandle{}, false, e
		}
		if literal == target {
			return entry, true, nil
		}
	}
	return ast.ASTHandle{}, false, nil
}

func (p *DirectProofInterpolator) collectContraResolvents(from uint32, pivot1, pivot, conjunction ast.ASTHandle, result *[]ast.ASTHandle) error {
	count, err := p.base.manager.NumArgs(conjunction)
	if err != nil {
		return err
	}
	negatedLiteral, err := p.base.manager.Arg(pivot1, 1)
	if err != nil {
		return err
	}
	negatedLiteral, err = p.base.manager.MkNot(negatedLiteral)
	if err != nil {
		return err
	}
	placeholder, err := p.placeholder(negatedLiteral)
	if err != nil {
		return err
	}
	proof, err := p.base.manager.Arg(pivot1, 0)
	if err != nil {
		return err
	}
	for i := from; i < count; i++ {
		entry, e := p.base.manager.Arg(conjunction, i)
		if e != nil {
			return e
		}
		if entry == pivot {
			continue
		}
		entryProof, e := p.base.manager.Arg(entry, 0)
		if e != nil {
			return e
		}
		if proof != p.noProof {
			entryProof, e = p.SubstituteProofTerm(placeholder, proof, entryProof)
			if e != nil {
				return e
			}
		} else {
			entryProof = p.noProof
		}
		literal, e := p.base.manager.Arg(entry, 1)
		if e != nil {
			return e
		}
		resolved, e := p.makeSymbol(p.contra, entryProof, literal)
		if e != nil {
			return e
		}
		*result = append(*result, resolved)
	}
	return nil
}
