// Source checkpoint: src/interp/iz3proof_itp.cpp local arithmetic rule cases.
package interp

import "github.com/glycerine/g3/pkg/ast"

func (p *DirectProofInterpolator) MakeFarkas(conclusion ast.ASTHandle, premises, premiseConclusions, coefficients []ast.ASTHandle) (ast.ASTHandle, error) {
	if err := p.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	if len(premises) != len(premiseConclusions) || len(coefficients) != len(premiseConclusions) {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	var aCoefficients, aInequalities []ast.ASTHandle
	for i, literal := range premiseConclusions {
		side, err := p.TermSide(literal)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		if side == TermMixed {
			return ast.ASTHandle{}, ProofError{}
		}
		if side == TermA {
			aCoefficients = append(aCoefficients, coefficients[i])
			aInequalities = append(aInequalities, literal)
		}
	}
	result, err := p.base.manager.SumInequalities(aCoefficients, aInequalities, true)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	resolutionConclusion := make([]ast.ASTHandle, len(premiseConclusions))
	for i := range premiseConclusions {
		resolutionConclusion[len(premiseConclusions)-i-1] = premiseConclusions[i]
	}
	for i, literal := range premiseConclusions {
		result, err = p.MakeResolution(literal, resolutionConclusion, result, premises[i])
		if err != nil {
			return ast.ASTHandle{}, err
		}
		resolutionConclusion = resolutionConclusion[:len(resolutionConclusion)-1]
	}
	return p.keep(result, nil)
}

func (p *DirectProofInterpolator) MakeLeqToEq(x, y, xLeqY, yLeqX ast.ASTHandle) (ast.ASTHandle, error) {
	if err := p.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	conclusion, err := p.base.manager.Make(OperatorEqual, x, y)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	side, err := p.TermSide(conclusion)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if side == TermA {
		return p.keep(p.base.manager.Make(OperatorFalse))
	}
	if side == TermB {
		return p.keep(p.base.manager.Make(OperatorTrue))
	}
	return ast.ASTHandle{}, ProofError{}
}

func (p *DirectProofInterpolator) MakeEqToLeq(x, y, xLeqY ast.ASTHandle) (ast.ASTHandle, error) {
	if err := p.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	side, err := p.TermSide(xLeqY)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if side == TermA {
		return p.keep(p.base.manager.Make(OperatorFalse))
	}
	if side == TermB {
		return p.keep(p.base.manager.Make(OperatorTrue))
	}
	return ast.ASTHandle{}, ProofError{}
}

func (p *DirectProofInterpolator) MakeCutRule(tLeqC, divisor, conclusion, premise ast.ASTHandle) (ast.ASTHandle, error) {
	if err := p.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	side, err := p.TermSide(conclusion)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	var interpolation ast.ASTHandle
	if side == TermA {
		interpolation, err = p.base.manager.Make(OperatorFalse)
	} else if side == TermB {
		interpolation, err = p.base.manager.Make(OperatorTrue)
	} else {
		return ast.ASTHandle{}, ProofError{}
	}
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return p.MakeResolution(tLeqC, []ast.ASTHandle{conclusion}, interpolation, premise)
}
