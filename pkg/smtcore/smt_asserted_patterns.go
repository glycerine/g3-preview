// Source: src/smt/asserted_formulas.h pattern_inference_fn and
// src/smt/asserted_formulas.cpp simplify_fmls::operator(), reduce.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/macros"
)

func (k *SMTKernel) inferPendingPatterns() (result error) {
	if !k.preprocess || !k.smtState.Ematching || k.patternInference == nil || k.assertedHead == len(k.assertions) {
		return nil
	}
	quantified := false
	for _, row := range k.assertions[k.assertedHead:] {
		has, err := k.manager.HasQuantifiers(row.Formula)
		if err != nil {
			return err
		}
		quantified = quantified || has
	}
	if !quantified {
		return nil
	}
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
	proofs, err := k.manager.ProofsEnabled()
	if err != nil {
		return err
	}
	for _, row := range k.assertions[k.assertedHead:] {
		if proofs {
			err = k.patternInference.RewriteWithProof(row.Formula, out, proof)
		} else {
			err = k.patternInference.Rewrite(row.Formula, out)
		}
		if err != nil {
			return err
		}
		if err = k.appendSimplifiedAssertion(batch, row, out, proof); err != nil {
			return err
		}
	}
	// Source inference has no reduce_and_solve post-operation.
	return k.replacePendingFormulaBatch(batch, false)
}
