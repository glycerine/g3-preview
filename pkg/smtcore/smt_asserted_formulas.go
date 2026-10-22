// Source: src/smt/asserted_formulas.cpp push_assertion().
package smtcore

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
)

func (k *SMTKernel) appendAssertion(formula, proof ast.ASTHandle) error {
	if err := k.manager.IncRef(formula); err != nil {
		return err
	}
	if !proof.IsNull() {
		if err := k.manager.IncRef(proof); err != nil {
			return errors.Join(err, k.manager.DecRef(formula))
		}
	}
	k.assertions = append(k.assertions, SMTKernelAssertion{Formula: formula, Proof: proof})
	return nil
}

func (k *SMTKernel) pushAssertion(formula, proof ast.ASTHandle) error {
	if k.manager.IsTrue(formula) {
		return nil
	}
	op, basic, err := k.internalizer.basicOperation(formula)
	if err != nil {
		return err
	}
	if basic && op == ast.ASTOpAnd {
		count, err := k.manager.GetNumArgs(formula)
		if err != nil {
			return err
		}
		for index := uint32(0); index < count; index++ {
			child, err := k.manager.GetArg(formula, index)
			if err != nil {
				return err
			}
			childProof := ast.ASTHandle{}
			if !proof.IsNull() {
				childProof, err = k.manager.MkProofAndElim(proof, index)
				if err != nil {
					return err
				}
			}
			if err = k.pushAssertion(child, childProof); err != nil {
				return fmt.Errorf("push conjunction child %d: %w", index, err)
			}
		}
		return nil
	}
	if basic && op == ast.ASTOpNot {
		child, err := k.manager.GetArg(formula, 0)
		if err != nil {
			return err
		}
		childOp, childBasic, err := k.internalizer.basicOperation(child)
		if err != nil {
			return err
		}
		if childBasic && childOp == ast.ASTOpOr {
			count, err := k.manager.GetNumArgs(child)
			if err != nil {
				return err
			}
			for index := uint32(0); index < count; index++ {
				argument, err := k.manager.GetArg(child, index)
				if err != nil {
					return err
				}
				childProof := ast.ASTHandle{}
				if !proof.IsNull() {
					childProof, err = k.manager.MkProofNotOrElim(proof, index)
					if err != nil {
						return err
					}
				}
				negated, err := k.manager.MkNot(argument)
				if err != nil {
					return err
				}
				if err = k.pushAssertion(negated, childProof); err != nil {
					return fmt.Errorf("push negated disjunction child %d: %w", index, err)
				}
			}
			return nil
		}
	}
	return k.appendAssertion(formula, proof)
}
