// Source: src/smt/asserted_formulas.cpp asserted_formulas::nnf_cnf.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/macros"
	"github.com/glycerine/g3/pkg/ast/normal_forms"
)

func (k *SMTKernel) normalizePendingAssertions() (result error) {
	if !k.preprocess || k.assertedHead == len(k.assertions) {
		return nil
	}
	apply := k.preprocessState.NNFCNF || (k.assertedHasQuantifiers &&
		k.internalizer.quantifierManager != nil && k.internalizer.quantifierManager.mbqi)
	if !apply {
		return nil
	}
	// The C++ caller supplies an empty local parameter set, while the NNF
	// constructor still reads the global nnf module's defaults/overrides.
	empty := k.paramContext.EmptyValue()
	proc, err := normal_forms.NewNNFFromParams(k.manager, k.definedNames, k.paramContext, k.paramGlobals, &empty)
	if proc != nil {
		defer func() { result = errors.Join(result, proc.Close()) }()
	}
	if err != nil {
		return err
	}
	batch, err := macros.NewFormulaBatch(k.manager)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, batch.Close()) }()
	for _, assertion := range k.assertions[k.assertedHead:] {
		if err := k.appendNormalizedAssertion(proc, batch, assertion); err != nil {
			return err
		}
	}
	return k.replacePendingMacroBatch(batch)
}

func (k *SMTKernel) appendNormalizedAssertion(proc *normal_forms.NNF, batch *macros.FormulaBatch, assertion SMTKernelAssertion) (result error) {
	out, err := k.manager.NewRef(ast.ASTHandle{})
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, k.manager.DelRef(&out)) }()
	proof, err := k.manager.NewRef(ast.ASTHandle{})
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, k.manager.DelRef(&proof)) }()
	var definitions, definitionProofs []ast.ASTRef
	defer func() {
		for i := range definitionProofs {
			result = errors.Join(result, k.manager.DelRef(&definitionProofs[i]))
		}
		for i := range definitions {
			result = errors.Join(result, k.manager.DelRef(&definitions[i]))
		}
	}()
	if err = proc.Transform(assertion.Formula, &definitions, &definitionProofs, out, proof); err != nil {
		return err
	}
	proofs, err := k.manager.ProofsEnabled()
	if err != nil {
		return err
	}
	for i, definition := range definitions {
		value, err := k.manager.RefGet(definition)
		if err != nil {
			return err
		}
		var p ast.ASTHandle
		if proofs {
			p, err = k.manager.RefGet(definitionProofs[i])
			if err != nil {
				return err
			}
		}
		if err = batch.Insert(macros.Formula{Expression: value, Proof: p}); err != nil {
			return err
		}
	}
	value, err := k.manager.RefGet(out)
	if err != nil {
		return err
	}
	var p ast.ASTHandle
	if proofs {
		rewrite, err := k.manager.RefGet(proof)
		if err != nil {
			return err
		}
		p, err = k.manager.MkProofModusPonens(assertion.Proof, rewrite)
		if err != nil {
			return err
		}
	}
	return batch.Insert(macros.Formula{Expression: value, Proof: p})
}
