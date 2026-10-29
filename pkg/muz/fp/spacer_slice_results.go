package fp

// Source: src/muz/spacer/spacer_dl_interface.cpp sliced-predicate forwarding
// and src/muz/transforms/dl_mk_slice.cpp result-variable expansion.

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	astrewriter "github.com/glycerine/g3/pkg/ast/rewriter"
	"github.com/glycerine/g3/pkg/model"
)

func (e *PropositionalEngine) spacerPredicate(declaration ast.ASTHandle) ast.ASTHandle {
	if derived, found := e.spacerQuantified[declaration]; found {
		declaration = derived
	}
	if derived, found := e.spacerScaled[declaration]; found {
		declaration = derived
	}
	if derived, found := e.spacerMagic[declaration]; found {
		declaration = derived
	}
	if derived, found := e.spacerBitBlasted[declaration]; found {
		declaration = derived
	}
	if derived, found := e.spacerInstantiated[declaration]; found {
		declaration = derived
	}
	if derived, found := e.spacerPredicates[declaration]; found {
		declaration = derived
	}
	return declaration
}

func slicedPredicateHasRemovedColumns(slices map[ast.ASTHandle][]bool, declaration ast.ASTHandle) bool {
	for _, removed := range slices[declaration] {
		if removed {
			return true
		}
	}
	return false
}

func expandSlicedFormula(manager *ast.ASTManager, slices map[ast.ASTHandle][]bool, declaration ast.ASTHandle, source ast.ASTRef) (_ ast.ASTRef, resultErr error) {
	removed, found := slices[declaration]
	if !found {
		return source, nil
	}
	formula, err := manager.RefGet(source)
	if err != nil {
		return ast.ASTRef{}, errors.Join(err, manager.DelRef(&source))
	}
	bindings := make([]ast.ASTHandle, 0, len(removed))
	for index, sliced := range removed {
		if sliced {
			continue
		}
		sort, getErr := manager.GetDomain(declaration, uint32(index))
		if getErr != nil {
			return ast.ASTRef{}, errors.Join(getErr, manager.DelRef(&source))
		}
		variable, getErr := manager.MkVar(uint32(index), sort)
		if getErr != nil {
			return ast.ASTRef{}, errors.Join(getErr, manager.DelRef(&source))
		}
		bindings = append(bindings, variable)
	}
	substitute, err := astrewriter.NewVarSubstitution(manager, false)
	if substitute != nil {
		defer func() { resultErr = errors.Join(resultErr, substitute.Close()) }()
	}
	if err != nil {
		return ast.ASTRef{}, errors.Join(err, manager.DelRef(&source))
	}
	result, err := manager.NewRef(ast.ASTHandle{})
	if err != nil {
		return ast.ASTRef{}, errors.Join(err, manager.DelRef(&source))
	}
	if err = substitute.Substitute(formula, bindings, result); err != nil {
		return ast.ASTRef{}, errors.Join(err, manager.DelRef(&result), manager.DelRef(&source))
	}
	if err = manager.DelRef(&source); err != nil {
		return ast.ASTRef{}, errors.Join(err, manager.DelRef(&result))
	}
	return result, nil
}

func rejectSlicedPredicateProperty(slices map[ast.ASTHandle][]bool, declaration ast.ASTHandle) error {
	if slicedPredicateHasRemovedColumns(slices, declaration) {
		return fmt.Errorf("muz/fp: covers and invariants are incompatible with sliced predicate columns")
	}
	return nil
}

func expandSlicedHandle(manager *ast.ASTManager, slices map[ast.ASTHandle][]bool, declaration, source ast.ASTHandle) (ast.ASTRef, error) {
	reference, err := manager.NewRef(source)
	if err != nil {
		return ast.ASTRef{}, err
	}
	return expandSlicedFormula(manager, slices, declaration, reference)
}

func expandSlicedFunction(manager *ast.ASTManager, slices map[ast.ASTHandle][]bool, destination *model.ModelCore, original, derived ast.ASTHandle) (resultErr error) {
	source, err := destination.FunctionInterpretation(derived)
	if err != nil {
		return err
	}
	if source == nil {
		return destination.UnregisterDeclaration(derived)
	}
	oldArity, err := manager.GetArity(original)
	if err != nil {
		return err
	}
	interpretation, err := model.NewFunctionInterpretation(manager, oldArity)
	if err != nil {
		return err
	}
	transferred := false
	defer func() {
		if !transferred {
			resultErr = errors.Join(resultErr, interpretation.Close())
		}
	}()
	compact, err := source.Interpretation()
	if err != nil {
		return err
	}
	if !compact.IsNull() {
		expanded, expandErr := expandSlicedHandle(manager, slices, original, compact)
		if expandErr != nil {
			return expandErr
		}
		expandedHandle, expandErr := manager.RefGet(expanded)
		if expandErr == nil {
			expandErr = interpretation.SetElse(expandedHandle)
		}
		expandErr = errors.Join(expandErr, manager.DelRef(&expanded))
		if expandErr != nil {
			return expandErr
		}
	}
	if err = destination.RegisterFunction(original, interpretation); err != nil {
		return err
	}
	transferred = true
	return destination.UnregisterDeclaration(derived)
}

func expandSlicedModel(manager *ast.ASTManager, predicates map[ast.ASTHandle]ast.ASTHandle, slices map[ast.ASTHandle][]bool, source *model.Model) (_ *model.Model, resultErr error) {
	if len(predicates) == 0 {
		return source, nil
	}
	if source == nil {
		return nil, ast.ErrASTHandle
	}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, source.Close())
			source = nil
		}
	}()
	core := source.Core()
	for original, derived := range predicates {
		arity, err := manager.GetArity(derived)
		if err != nil {
			return nil, err
		}
		if arity != 0 {
			if err = expandSlicedFunction(manager, slices, core, original, derived); err != nil {
				return nil, err
			}
			continue
		}
		present, err := core.HasConstantInterpretation(derived)
		if err != nil {
			return nil, err
		}
		if !present {
			continue
		}
		value, err := core.ConstantInterpretation(derived)
		if err != nil {
			return nil, err
		}
		oldArity, err := manager.GetArity(original)
		if err != nil {
			return nil, err
		}
		interpretation, err := model.NewFunctionInterpretation(manager, oldArity)
		if err != nil {
			return nil, err
		}
		if !value.IsNull() {
			err = interpretation.SetElse(value)
		}
		if err == nil {
			err = core.RegisterFunction(original, interpretation)
		}
		if err != nil {
			return nil, errors.Join(err, interpretation.Close())
		}
		if err = core.UnregisterDeclaration(derived); err != nil {
			return nil, err
		}
	}
	return source, nil
}

func (e *PropositionalEngine) spacerHasRemovedColumns(declaration ast.ASTHandle) bool {
	return slicedPredicateHasRemovedColumns(e.spacerSlices, declaration)
}

func (e *PropositionalEngine) expandSpacerFormula(declaration ast.ASTHandle, source ast.ASTRef) (ast.ASTRef, error) {
	return expandSlicedFormula(e.manager, e.spacerSlices, declaration, source)
}

func (e *PropositionalEngine) rejectSlicedProperty(declaration ast.ASTHandle) error {
	return rejectSlicedPredicateProperty(e.spacerSlices, declaration)
}

func (e *PropositionalEngine) expandSpacerHandle(declaration, source ast.ASTHandle) (ast.ASTRef, error) {
	return expandSlicedHandle(e.manager, e.spacerSlices, declaration, source)
}

func (e *PropositionalEngine) expandSpacerFunction(destination *model.ModelCore, original, derived ast.ASTHandle) error {
	return expandSlicedFunction(e.manager, e.spacerSlices, destination, original, derived)
}

func (e *PropositionalEngine) expandSpacerModel(source *model.Model) (_ *model.Model, resultErr error) {
	if len(e.spacerPredicates) == 0 && len(e.spacerQuantified) == 0 && len(e.spacerBitBlasted) == 0 && len(e.spacerScaled) == 0 && len(e.spacerKarr) == 0 {
		return source, nil
	}
	source, err := expandSlicedModel(e.manager, e.spacerPredicates, e.spacerSlices, source)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, ast.ErrASTHandle
	}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, source.Close())
			source = nil
		}
	}()
	core := source.Core()
	for original, derived := range e.spacerBitBlasted {
		if err := e.expandSpacerBitBlastedFunction(core, original, derived); err != nil {
			return nil, err
		}
	}
	for original, derived := range e.spacerScaled {
		if err := e.expandSpacerScaledFunction(core, original, derived); err != nil {
			return nil, err
		}
	}
	for original, derived := range e.spacerQuantified {
		if err := e.expandSpacerQuantifiedFunction(core, original, derived); err != nil {
			return nil, err
		}
	}
	if err := e.applySpacerKarrInvariants(core); err != nil {
		return nil, err
	}
	return source, nil
}
