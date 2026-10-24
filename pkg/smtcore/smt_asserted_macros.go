// Source: src/smt/asserted_formulas.cpp find_macros_core and commit;
// src/smt/smt_model_generator.cpp register_macros.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/macros"
	"github.com/glycerine/g3/pkg/model"
)

func (k *SMTKernel) findPendingMacros() (result error) {
	if !k.preprocess || !k.preprocessState.MacroFinder || !k.assertedHasQuantifiers || k.assertedHead == len(k.assertions) {
		return nil
	}
	inputs := make([]macros.Formula, 0, len(k.assertions)-k.assertedHead)
	for _, assertion := range k.assertions[k.assertedHead:] {
		inputs = append(inputs, macros.Formula{Expression: assertion.Formula, Proof: assertion.Proof})
	}
	finder, err := macros.NewFinder(k.manager, k.assertedMacros)
	if err != nil {
		return err
	}
	batch, err := finder.Apply(inputs)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, batch.Close()) }()
	return k.replacePendingMacroBatch(batch)
}

func (k *SMTKernel) replacePendingMacroBatch(batch *macros.FormulaBatch) error {
	if err := k.swapPendingFormulaBatch(batch); err != nil {
		return err
	}
	return k.reduceAndSolveAssertions()
}

// swapPendingFormulaBatch copies already justified, flattened rows. Source
// swap_asserted_formulas does not push, rewrite or poll cancellation again.
func (k *SMTKernel) swapPendingFormulaBatch(batch *macros.FormulaBatch) (result error) {
	original := k.assertions
	k.assertions = append([]SMTKernelAssertion(nil), original[:k.assertedHead]...)
	defer func() {
		if result != nil {
			for _, assertion := range k.assertions[k.assertedHead:] {
				result = errors.Join(result, k.manager.DecRef(assertion.Proof), k.manager.DecRef(assertion.Formula))
			}
			k.assertions = original
			return
		}
		for _, assertion := range original[k.assertedHead:] {
			result = errors.Join(result, k.manager.DecRef(assertion.Proof), k.manager.DecRef(assertion.Formula))
		}
	}()
	for i := 0; i < batch.Len(); i++ {
		row, err := batch.At(i)
		if err != nil {
			return err
		}
		if err := k.appendAssertion(row.Expression, row.Proof); err != nil {
			return err
		}
	}
	return nil
}

// Discovery flags do not change definitions already removed from the formula
// list. Expand those definitions before committing any later assertions.
func (k *SMTKernel) expandPendingMacros() (result error) {
	if !k.assertedMacros.HasMacros() || k.assertedHead == len(k.assertions) {
		return nil
	}
	batch, err := macros.NewFormulaBatch(k.manager)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, batch.Close()) }()
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
	for _, assertion := range k.assertions[k.assertedHead:] {
		dependency, err := k.assertedMacros.ExpandMacros(assertion.Formula, assertion.Proof, ast.ASTDependency{}, out, proof)
		if err != nil {
			return err
		}
		row := macros.Formula{Dependency: dependency}
		row.Expression, err = k.manager.RefGet(out)
		if err == nil {
			row.Proof, err = k.manager.RefGet(proof)
		}
		if err == nil {
			err = batch.Insert(row)
		}
		if err = errors.Join(err, k.manager.DependencyDec(dependency)); err != nil {
			return err
		}
	}
	return k.replacePendingMacroBatch(batch)
}

// applyPendingQuasiMacros follows the source repeated quasi-macro pass. Each
// batch owns the formulas until the next pass has acquired its references.
func (k *SMTKernel) applyPendingQuasiMacros() (result error) {
	if !k.preprocess || !k.preprocessState.QuasiMacros || !k.assertedHasQuantifiers || k.assertedHead == len(k.assertions) {
		return nil
	}
	var inputs []macros.Formula
	for _, assertion := range k.assertions[k.assertedHead:] {
		inputs = append(inputs, macros.Formula{Expression: assertion.Formula, Proof: assertion.Proof})
	}
	proc, err := macros.NewQuasiMacros(k.manager, k.assertedMacros)
	if err != nil {
		return err
	}
	var batch *macros.FormulaBatch
	defer func() {
		if batch != nil {
			result = errors.Join(result, batch.Close())
		}
	}()
	for {
		next, changed, err := proc.Apply(inputs)
		if err != nil {
			return err
		}
		if batch != nil {
			if err = batch.Close(); err != nil {
				return errors.Join(err, next.Close())
			}
		}
		batch = next
		if !changed {
			break
		}
		inputs = nil
		for i := 0; i < batch.Len(); i++ {
			row, err := batch.At(i)
			if err != nil {
				return err
			}
			inputs = append(inputs, row)
		}
	}
	return k.replacePendingMacroBatch(batch)
}

func (g *SMTModelGenerator) registerAssertedMacros(manager *macros.Manager) (result error) {
	if manager == nil || !manager.HasMacros() {
		return nil
	}
	out, err := g.manager.NewRef(ast.ASTHandle{})
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, g.manager.DelRef(&out)) }()
	for i := 0; i < manager.NumMacros(); i++ {
		declaration, err := manager.MacroInterpretation(i, out)
		if err != nil {
			return err
		}
		definition, err := g.manager.RefGet(out)
		if err != nil {
			return err
		}
		arity, err := g.manager.GetArity(declaration)
		if err != nil {
			return err
		}
		interpretation, err := model.NewFunctionInterpretation(g.manager, arity)
		if err != nil {
			return err
		}
		if err = interpretation.SetElse(definition); err == nil {
			err = g.model.Core().RegisterFunction(declaration, interpretation)
		}
		if err != nil {
			return errors.Join(err, interpretation.Close())
		}
	}
	return nil
}
