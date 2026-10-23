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
	if !k.preprocess || !k.preprocessState.MacroFinder || k.assertedHead == len(k.assertions) {
		return nil
	}
	hasQuantifiers := k.assertedMacros.HasMacros()
	inputs := make([]macros.Formula, 0, len(k.assertions)-k.assertedHead)
	for _, assertion := range k.assertions[k.assertedHead:] {
		nested, err := k.manager.HasQuantifiers(assertion.Formula)
		if err != nil {
			return err
		}
		hasQuantifiers = hasQuantifiers || nested
		inputs = append(inputs, macros.Formula{Expression: assertion.Formula, Proof: assertion.Proof})
	}
	if !hasQuantifiers {
		return nil
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
	original := k.assertions
	// Keep the old references alive until all replacement assertions have
	// acquired theirs. On failure, retain the original pending formula list.
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
		formula, proof, err := k.preprocessAssertion(row.Expression, row.Proof)
		if err != nil {
			return err
		}
		err = k.pushAssertion(formula, proof)
		err = errors.Join(err, k.manager.DecRef(proof), k.manager.DecRef(formula))
		if err != nil {
			return err
		}
	}
	return k.propagatePendingAssertions()
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
