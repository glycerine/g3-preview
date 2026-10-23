// Source: src/smt/smt_setup.cpp setup_QF_LIA() and setup_QF_LIA(static_features).
package smtcore

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
)

func (k *SMTKernel) setupIntegerLogicPolicy(useStaticFeatures bool) (result error) {
	if k.setupLogic != "QF_LIA" {
		return nil
	}
	var features *ast.StaticFeatures
	if useStaticFeatures {
		var err error
		features, err = ast.NewStaticFeatures(k.manager)
		if err != nil {
			return err
		}
		defer func() { result = errors.Join(result, features.Close()) }()
		formulas := make([]ast.ASTHandle, len(k.assertions))
		for index, assertion := range k.assertions {
			formulas[index] = assertion.Formula
		}
		if err = features.Collect(formulas...); err != nil {
			return err
		}
		if features.NumUninterpretedFunctions != 0 {
			return fmt.Errorf("Benchmark contains uninterpreted function symbols, but specified logic does not support them.")
		}
	}
	smt := k.smtParams.state
	arithmetic := k.smtParams.arith.state
	preprocess := k.smtParams.preprocess.state
	smt.RelevancyLvl = 0
	arithmetic.Eq2Ineq = true
	arithmetic.Reflect = false
	arithmetic.PropagateEqs = false
	preprocess.NNFCNF = false
	if features != nil {
		switch {
		case features.MaxITETreeDepth > 50:
			arithmetic.Eq2Ineq = false
			preprocess.PullCheapITE = true
			arithmetic.PropagateEqs = true
			smt.RelevancyLvl = 2
			smt.RelevancyLemma = false
		case features.NumClauses == features.NumUnits:
			arithmetic.GCDTest = false
			arithmetic.BranchCutRatio = 4
			smt.RelevancyLvl = 2
			arithmetic.Eq2Ineq = true
			preprocess.EliminateTermITE = true
		default:
			preprocess.EliminateTermITE = true
			smt.PhaseSelection = 2 // PS_CACHING
			smt.RestartAdaptive = false
			smt.RestartStrategy = 0 // RS_GEOMETRIC
			smt.RestartFactor = 1.5
		}
		if features.NumBinClauses+features.NumUnits == features.NumClauses && features.CNF {
			large, err := k.manager.NumeralService().GreaterInt(&features.ArithKSum, 100000)
			if err != nil {
				return err
			}
			if large {
				arithmetic.BoundProp = 0 // BP_NONE
				arithmetic.StrongerLemmas = false
			}
		}
	}
	k.smtParams.state, k.smtState = smt, smt
	k.smtParams.arith.state, k.arithState = arithmetic, arithmetic
	k.smtParams.preprocess.state, k.preprocessState = preprocess, preprocess
	k.relevancy.enabled = smt.RelevancyLvl > 0
	k.internalizer.relevancyLevel = smt.RelevancyLvl
	for _, theory := range k.internalizer.theoryOrder {
		if configurable, ok := theory.(interface{ ConfigureArithmetic(SMTArithState) error }); ok {
			if err := configurable.ConfigureArithmetic(arithmetic); err != nil {
				return err
			}
		}
	}
	k.assertedPhaseReady = false
	return k.setAssertedEliminateAnd(false)
}
