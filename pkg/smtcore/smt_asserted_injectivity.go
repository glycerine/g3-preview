// Source: src/smt/asserted_formulas.cpp refine_inj_axiom_fn::simplify and
// simplify_fmls::operator().
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/macros"
	"github.com/glycerine/g3/pkg/ast/rewriter"
)

func (k *SMTKernel) refinePendingInjectivityAxioms() (result error) {
	if !k.preprocess || !k.preprocessState.RefineInjectivityAxiom || !k.assertedHasQuantifiers || k.assertedHead == len(k.assertions) {
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
	proofs, err := k.manager.ProofsEnabled()
	if err != nil {
		return err
	}
	changed := false
	for _, assertion := range k.assertions[k.assertedHead:] {
		row := macros.Formula{Expression: assertion.Formula, Proof: assertion.Proof}
		kind, err := k.manager.GetKind(assertion.Formula)
		if err != nil {
			return err
		}
		if kind == ast.ASTQuantifier {
			matched, err := rewriter.SimplifyInjectivityAxiom(k.manager, assertion.Formula, out)
			if err != nil {
				return err
			}
			if matched {
				row.Expression, err = k.manager.RefGet(out)
				if err != nil {
					return err
				}
				if proofs {
					rewrite, err := k.manager.MkProofRewrite(assertion.Formula, row.Expression)
					if err != nil {
						return err
					}
					row.Proof, err = k.manager.MkProofModusPonens(assertion.Proof, rewrite)
					if err != nil {
						return err
					}
				}
				changed = true
			}
		}
		if err := batch.Insert(row); err != nil {
			return err
		}
	}
	if !changed {
		return nil
	}
	return k.replacePendingFormulaBatch(batch, false)
}
