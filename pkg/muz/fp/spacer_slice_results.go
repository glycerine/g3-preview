package fp

// Source: src/muz/spacer/spacer_dl_interface.cpp sliced-predicate forwarding
// and src/muz/transforms/dl_mk_slice.cpp result-variable expansion.

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	astrewriter "github.com/glycerine/g3/pkg/ast/rewriter"
)

func (e *PropositionalEngine) spacerPredicate(declaration ast.ASTHandle) ast.ASTHandle {
	if derived, found := e.spacerPredicates[declaration]; found {
		return derived
	}
	return declaration
}

func (e *PropositionalEngine) spacerHasRemovedColumns(declaration ast.ASTHandle) bool {
	for _, removed := range e.spacerSlices[declaration] {
		if removed {
			return true
		}
	}
	return false
}

func (e *PropositionalEngine) expandSpacerFormula(declaration ast.ASTHandle, source ast.ASTRef) (_ ast.ASTRef, resultErr error) {
	removed, found := e.spacerSlices[declaration]
	if !found {
		return source, nil
	}
	formula, err := e.manager.RefGet(source)
	if err != nil {
		return ast.ASTRef{}, errors.Join(err, e.manager.DelRef(&source))
	}
	bindings := make([]ast.ASTHandle, 0, len(removed))
	for index, sliced := range removed {
		if sliced {
			continue
		}
		sort, getErr := e.manager.GetDomain(declaration, uint32(index))
		if getErr != nil {
			return ast.ASTRef{}, errors.Join(getErr, e.manager.DelRef(&source))
		}
		variable, getErr := e.manager.MkVar(uint32(index), sort)
		if getErr != nil {
			return ast.ASTRef{}, errors.Join(getErr, e.manager.DelRef(&source))
		}
		bindings = append(bindings, variable)
	}
	substitute, err := astrewriter.NewVarSubstitution(e.manager, false)
	if substitute != nil {
		defer func() { resultErr = errors.Join(resultErr, substitute.Close()) }()
	}
	if err != nil {
		return ast.ASTRef{}, errors.Join(err, e.manager.DelRef(&source))
	}
	result, err := e.manager.NewRef(ast.ASTHandle{})
	if err != nil {
		return ast.ASTRef{}, errors.Join(err, e.manager.DelRef(&source))
	}
	if err = substitute.Substitute(formula, bindings, result); err != nil {
		return ast.ASTRef{}, errors.Join(err, e.manager.DelRef(&result), e.manager.DelRef(&source))
	}
	if err = e.manager.DelRef(&source); err != nil {
		return ast.ASTRef{}, errors.Join(err, e.manager.DelRef(&result))
	}
	return result, nil
}

func (e *PropositionalEngine) rejectSlicedProperty(declaration ast.ASTHandle) error {
	if e.spacerHasRemovedColumns(declaration) {
		return fmt.Errorf("muz/fp: covers and invariants are incompatible with sliced predicate columns")
	}
	return nil
}
