// Source: src/smt/smt_setup.cpp setup_QF_UF overloads and setup_QF_UFLRA.
package smtcore

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
)

func (k *SMTKernel) setupSimpleLogicPolicy(useStaticFeatures bool) (result error) {
	switch k.setupLogic {
	case "QF_UF":
		if useStaticFeatures {
			features, err := ast.NewStaticFeatures(k.manager)
			if err != nil {
				return err
			}
			defer func() { result = errors.Join(result, features.Close()) }()
			formulas := make([]ast.ASTHandle, len(k.assertions))
			for index, row := range k.assertions {
				formulas[index] = row.Formula
			}
			if err = features.Collect(formulas...); err != nil {
				return err
			}
			if features.NumArithInequalities > 0 || features.NumArithTerms > 0 || features.NumArithEqualities > 0 {
				// Retain the source exception's spelling.
				return fmt.Errorf("Benchmark constains arithmetic, but specified loging does not support it.")
			}
		}
		k.smtParams.state.RestartStrategy = 2       // RS_LUBY
		k.smtParams.state.PhaseSelection = 4        // PS_CACHING_CONSERVATIVE2
		k.smtParams.state.RandomInitialActivity = 2 // IA_RANDOM
	case "QF_UFLRA":
		k.smtParams.arith.state.Reflect = false
		k.arithState.Reflect = false
	default:
		return nil
	}
	k.smtParams.state.RelevancyLvl = 0
	k.smtState = k.smtParams.state
	k.smtParams.preprocess.state.NNFCNF = false
	k.preprocessState.NNFCNF = false
	k.relevancy.enabled = false
	k.internalizer.relevancyLevel = 0
	// First-push setup precedes Boolean variable creation, so the selected
	// random initial-activity policy must reach the internalizer immediately.
	if err := k.internalizer.configureBooleanHeuristics(k.smtState); err != nil {
		return err
	}
	k.assertedPhaseReady = false
	return k.setAssertedEliminateAnd(false)
}
