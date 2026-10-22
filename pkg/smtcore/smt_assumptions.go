// Source: src/smt/smt_context.cpp validate_assumptions(),
// init_assumptions(), and reset_assumptions().
package smtcore

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func (k *SMTKernel) validAssumption(assumption ast.ASTHandle) (bool, error) {
	sort, err := k.manager.GetSort(assumption)
	if err != nil {
		return false, err
	}
	boolean, err := k.manager.MkBoolSort()
	if err != nil || sort != boolean {
		return false, err
	}
	kind, err := k.manager.GetKind(assumption)
	if err != nil || kind != ast.ASTApp {
		return false, err
	}
	count, err := k.manager.GetNumArgs(assumption)
	if err != nil || count == 0 {
		return count == 0, err
	}
	op, basic, err := k.internalizer.basicOperation(assumption)
	if err != nil || !basic || op != ast.ASTOpNot || count != 1 {
		return false, err
	}
	argument, err := k.manager.GetArg(assumption, 0)
	if err != nil {
		return false, err
	}
	argumentKind, err := k.manager.GetKind(argument)
	if err != nil || argumentKind != ast.ASTApp {
		return false, err
	}
	argumentCount, err := k.manager.GetNumArgs(argument)
	return argumentCount == 0, err
}

// CheckAssumptions checks under temporary propositional literals. The
// assumption scope and every variable introduced in it are removed before the
// method returns.
func (k *SMTKernel) CheckAssumptions(assumptions []ast.ASTHandle) (result lbool.LBool, resultErr error) {
	if err := k.open(); err != nil {
		return lbool.Undef, err
	}
	for index, assumption := range assumptions {
		valid, err := k.validAssumption(assumption)
		if err != nil {
			return lbool.Undef, err
		}
		if !valid {
			return lbool.Undef, fmt.Errorf("assumption %d: %w", index, ast.ErrASTDomain)
		}
	}
	queue, err := k.prepareCheck()
	if err != nil {
		return lbool.Undef, err
	}
	consistent, err := k.internalizer.PropagateCore()
	if err != nil {
		return lbool.Undef, err
	}
	if !consistent || len(assumptions) == 0 {
		if !consistent {
			return lbool.False, nil
		}
		return k.search(queue)
	}
	if err = k.internalizer.PushScope(); err != nil {
		return lbool.Undef, err
	}
	assumptionVariables := make([]int32, 0, len(assumptions))
	defer func() {
		for _, variable := range assumptionVariables {
			if variable >= SMTFirstBoolVar && int(variable) < len(k.internalizer.booleanData) {
				k.internalizer.booleanData[variable].Assumption = false
			}
		}
		resultErr = errors.Join(resultErr, k.internalizer.PopScopes(1))
		if resultErr != nil {
			result = lbool.Undef
		}
	}()
	for index, assumption := range assumptions {
		if k.manager.IsTrue(assumption) {
			continue
		}
		proof := ast.ASTHandle{}
		proofs, err := k.manager.ProofsEnabled()
		if err != nil {
			return lbool.Undef, err
		}
		if proofs {
			proof, err = k.manager.MkProofAsserted(assumption)
			if err != nil {
				return lbool.Undef, err
			}
		}
		if err = k.internalizer.Assert(assumption, proof, 0); err != nil {
			return lbool.Undef, fmt.Errorf("internalize assumption %d: %w", index, err)
		}
		literal, err := k.internalizer.Literal(assumption)
		if err != nil {
			return lbool.Undef, err
		}
		if literal.Var() >= SMTFirstBoolVar {
			k.internalizer.booleanData[literal.Var()].Assumption = true
			assumptionVariables = append(assumptionVariables, literal.Var())
		}
	}
	return k.search(queue)
}
