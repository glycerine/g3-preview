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

func (k *SMTKernel) preprocessAssertion(formula, proof ast.ASTHandle) (result, resultProof ast.ASTHandle, err error) {
	if !k.preprocess || k.preprocessor == nil {
		return formula, proof, nil
	}
	resultRef, err := k.manager.NewRef(ast.ASTHandle{})
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	defer func() { err = errors.Join(err, k.manager.DelRef(&resultRef)) }()
	proofRef, err := k.manager.NewRef(ast.ASTHandle{})
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	defer func() { err = errors.Join(err, k.manager.DelRef(&proofRef)) }()
	proofs, err := k.manager.ProofsEnabled()
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	if proofs {
		err = k.preprocessor.RewriteWithProof(formula, resultRef, proofRef)
	} else {
		err = k.preprocessor.Rewrite(formula, resultRef)
	}
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	result, err = k.manager.RefGet(resultRef)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	resultProof = proof
	if proofs && result != formula {
		rewriteProof, getErr := k.manager.RefGet(proofRef)
		if getErr != nil {
			return ast.ASTHandle{}, ast.ASTHandle{}, getErr
		}
		resultProof, err = k.manager.MkProofModusPonens(proof, rewriteProof)
	}
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	if err = k.manager.IncRef(result); err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	if !resultProof.IsNull() {
		if err = k.manager.IncRef(resultProof); err != nil {
			return ast.ASTHandle{}, ast.ASTHandle{}, errors.Join(err, k.manager.DecRef(result))
		}
	}
	return result, resultProof, nil
}
