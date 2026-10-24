// Source: src/smt/asserted_formulas.cpp set_eliminate_and, reduce_and_solve
// and simplify_fmls; src/smt/asserted_formulas.h distribute_forall_fn.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/macros"
	"github.com/glycerine/g3/pkg/ast/rewriter"
)

func (k *SMTKernel) setAssertedEliminateAnd(flag bool) error {
	if !k.preprocess || k.preprocessor == nil || (k.assertedPhaseReady && k.assertedElimAnd == flag) {
		return nil
	}
	settings := []struct {
		name  string
		value bool
	}{
		{"elim_and", flag}, {"arith_ineq_lhs", true}, {"sort_sums", true},
		{"rewrite_patterns", true}, {"eq2ineq", k.arithState.Eq2Ineq},
		{"gcd_rounding", true}, {"expand_select_store", true},
		{"bv_sort_ac", true}, {"som", true},
	}
	if k.preprocessState.PullCheapITE {
		settings = append(settings, struct {
			name  string
			value bool
		}{"pull_cheap_ite", true})
	}
	for _, setting := range settings {
		name := setting.name
		if err := k.paramContext.SetBoolName(&k.assertedParams, &name, setting.value); err != nil {
			return err
		}
	}
	if err := k.preprocessor.UpdateParams(&k.assertedParams); err != nil {
		return err
	}
	if err := k.preprocessor.Reset(); err != nil {
		return err
	}
	k.assertedElimAnd, k.assertedPhaseReady = flag, true
	return nil
}

func (k *SMTKernel) reducePendingAssertions() (result error) {
	if !k.preprocess || k.preprocessor == nil || k.assertedHead == len(k.assertions) {
		return nil
	}
	batch, err := macros.NewFormulaBatch(k.manager)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, batch.Close()) }()
	for _, row := range k.assertions[k.assertedHead:] {
		if err := batch.Insert(macros.Formula{Expression: row.Formula, Proof: row.Proof}); err != nil {
			return err
		}
	}
	return k.replacePendingFormulaBatch(batch, true)
}

func (k *SMTKernel) distributePendingUniversals() (result error) {
	if !k.preprocess || !k.preprocessState.DistributeForall || k.assertedHead == len(k.assertions) {
		return nil
	}
	if !k.assertedHasQuantifiers {
		return nil
	}
	proc, err := rewriter.NewDistributeForall(k.manager, k.paramContext, k.paramGlobals)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, proc.Close()) }()
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
	batch, err := macros.NewFormulaBatch(k.manager)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, batch.Close()) }()
	for _, assertion := range k.assertions[k.assertedHead:] {
		if err := proc.Apply(assertion.Formula, out); err != nil {
			return err
		}
		if err := k.appendSimplifiedAssertion(batch, assertion, out, proof); err != nil {
			return err
		}
	}
	return k.replacePendingFormulaBatch(batch, true)
}
