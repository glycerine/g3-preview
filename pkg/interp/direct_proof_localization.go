// Source checkpoint: src/interp/iz3proof_itp.cpp localization-variable
// creation and add_quants active branch.
package interp

import (
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
)

type localizationVariable struct {
	variable ast.ASTHandle
	term     ast.ASTHandle
	frame    int
}

func (p *DirectProofInterpolator) freshLocalizationVariable(term ast.ASTHandle, frame int) (ast.ASTHandle, error) {
	sort, err := p.base.manager.Type(term)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	variable, err := p.base.manager.MakeVariable(fmt.Sprintf("%%%d", len(p.localizationVariables)), sort)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if err = p.hold(variable); err != nil {
		return ast.ASTHandle{}, err
	}
	declaration, err := p.base.manager.Symbol(variable)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	p.base.symbolRange[declaration] = FullScopeRange()
	p.localizationVariables = append(p.localizationVariables, localizationVariable{variable: variable, term: term, frame: frame})
	return variable, nil
}

func (p *DirectProofInterpolator) addLocalizationQuantifiers(expression ast.ASTHandle) (ast.ASTHandle, error) {
	result := expression
	for i := len(p.localizationVariables) - 1; i >= 0; i-- {
		variable := p.localizationVariables[i]
		inside, err := p.base.scopes.InRange(variable.frame, p.scope)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		quantifier := OperatorForall
		if inside {
			quantifier = OperatorExists
		}
		result, err = p.base.manager.ApplyQuantifier(quantifier, variable.variable, result)
		if err != nil {
			return ast.ASTHandle{}, err
		}
	}
	return result, nil
}
