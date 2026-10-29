package pdr

// Source: core_bool_inductive_generalizer in
// src/muz/pdr/pdr_generalizers.cpp.

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
)

func closeASTRefs(manager *ast.ASTManager, references []ast.ASTRef) (result error) {
	for index := len(references) - 1; index >= 0; index-- {
		result = errors.Join(result, manager.DelRef(&references[index]))
	}
	return result
}

func pinASTHandles(manager *ast.ASTManager, handles []ast.ASTHandle) ([]ast.ASTRef, error) {
	references := make([]ast.ASTRef, 0, len(handles))
	for _, handle := range handles {
		reference, err := manager.NewRef(handle)
		if err != nil {
			return nil, errors.Join(err, closeASTRefs(manager, references))
		}
		references = append(references, reference)
	}
	return references, nil
}

// generalizeBoolInductiveCore drops literals one at a time, retaining a drop
// whenever the remaining cube is still inductive. The returned references own
// the result across later solver checks and must be released by the caller.
func generalizeBoolInductiveCore(transformer *PredicateTransformer, level uint32, input []ast.ASTHandle, usesLevel bool, failureLimit uint32) (_ []ast.ASTHandle, resultReferences []ast.ASTRef, _ bool, resultErr error) {
	if transformer == nil {
		return nil, nil, usesLevel, ErrPredicateTransformerOwner
	}
	if err := transformer.open(); err != nil {
		return nil, nil, usesLevel, err
	}
	core := append([]ast.ASTHandle(nil), input...)
	workspace, err := pinASTHandles(transformer.manager, core)
	if err != nil {
		return nil, nil, usesLevel, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, closeASTRefs(transformer.manager, workspace))
		if resultErr != nil {
			resultErr = errors.Join(resultErr, closeASTRefs(transformer.manager, resultReferences))
			resultReferences = nil
		}
	}()
	if len(core) > 1 {
		truth, err := transformer.manager.MkTrue()
		if err != nil {
			return nil, nil, usesLevel, err
		}
		processed := make(map[ast.ASTHandle]bool, len(core))
		var failures uint32
		for index := 0; index < len(core) && len(core) > 1 && (failureLimit == 0 || failures <= failureLimit); {
			literal := core[index]
			core[index] = truth
			inductive, assumesLevel, reduced, err := transformer.CheckInductive(level, core)
			if err != nil {
				return nil, nil, usesLevel, err
			}
			if inductive {
				pinned, err := pinASTHandles(transformer.manager, reduced)
				if err != nil {
					return nil, nil, usesLevel, err
				}
				workspace = append(workspace, pinned...)
				core = append(core[:0], reduced...)
				usesLevel = assumesLevel
				failures = 0
				index = 0
				for index < len(core) && processed[core[index]] {
					index++
				}
				continue
			}
			core[index] = literal
			processed[literal] = true
			failures++
			index++
		}
	}
	resultReferences, err = pinASTHandles(transformer.manager, core)
	if err != nil {
		return nil, nil, usesLevel, err
	}
	return core, resultReferences, usesLevel, nil
}
