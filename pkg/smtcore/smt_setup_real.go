// Source: src/smt/smt_setup.cpp setup_QF_LRA overloads and shared setup inputs.
package smtcore

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
)

func (k *SMTKernel) setupStaticFeatures(enabled bool) (*ast.StaticFeatures, error) {
	if !enabled {
		return nil, nil
	}
	features, err := ast.NewStaticFeatures(k.manager)
	if err != nil {
		return nil, err
	}
	formulas := make([]ast.ASTHandle, len(k.assertions))
	for index, assertion := range k.assertions {
		formulas[index] = assertion.Formula
	}
	if err = features.Collect(formulas...); err != nil {
		return nil, errors.Join(err, features.Close())
	}
	return features, nil
}

func (k *SMTKernel) arithmeticSetupFeatures(enabled bool) (*ast.StaticFeatures, error) {
	features, err := k.setupStaticFeatures(enabled)
	if err != nil || features == nil {
		return features, err
	}
	if features.NumUninterpretedFunctions != 0 {
		return nil, errors.Join(fmt.Errorf("Benchmark contains uninterpreted function symbols, but specified logic does not support them."), features.Close())
	}
	return features, nil
}

func (k *SMTKernel) applyArithmeticSetup(smt SMTParamsState, arithmetic SMTArithState, preprocess SMTPreprocessorState) error {
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

func (k *SMTKernel) setupRealLogicPolicy(useStaticFeatures bool) (result error) {
	if k.setupLogic != "QF_LRA" {
		return nil
	}
	features, err := k.arithmeticSetupFeatures(useStaticFeatures)
	if err != nil {
		return err
	}
	if features != nil {
		defer func() { result = errors.Join(result, features.Close()) }()
	}
	smt := k.smtParams.state
	arithmetic := k.smtParams.arith.state
	preprocess := k.smtParams.preprocess.state
	smt.RelevancyLvl = 0
	arithmetic.Eq2Ineq = true
	arithmetic.Reflect = false
	arithmetic.PropagateEqs = false
	preprocess.EliminateTermITE = true
	preprocess.NNFCNF = false
	if features != nil {
		numerals := k.manager.NumeralService()
		numerator, err := numerals.Numerator(&features.ArithKSum)
		if err != nil {
			return err
		}
		defer func() { result = errors.Join(result, numerals.Del(&numerator)) }()
		largeNumerator, err := numerals.GreaterInt(&numerator, 2000000)
		if err != nil {
			return err
		}
		if largeNumerator {
			denominator, err := numerals.Denominator(&features.ArithKSum)
			if err != nil {
				return err
			}
			defer func() { result = errors.Join(result, numerals.Del(&denominator)) }()
			largeDenominator, err := numerals.GreaterInt(&denominator, 500)
			if err != nil {
				return err
			}
			if largeDenominator {
				smt.RelevancyLvl = 2
				smt.RelevancyLemma = false
			}
		}
		if features.CNF {
			smt.PhaseSelection = 4 // PS_CACHING_CONSERVATIVE2
		} else {
			smt.RestartStrategy = 0 // RS_GEOMETRIC
			arithmetic.StrongerLemmas = false
			smt.PhaseSelection = 0 // PS_ALWAYS_FALSE
			smt.RestartAdaptive = false
		}
		arithmetic.SmallLemmaSize = 32
	}
	return k.applyArithmeticSetup(smt, arithmetic, preprocess)
}
