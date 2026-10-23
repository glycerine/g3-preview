// Source: src/smt/smt_setup.cpp setup_QF_IDL overloads and is_dense.
package smtcore

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
)

func smtDenseArithmetic(features *ast.StaticFeatures) bool {
	return features.NumUninterpretedConstants < 1000 &&
		features.NumArithEqualities+features.NumArithInequalities > features.NumUninterpretedConstants*9
}

func (k *SMTKernel) setupIntegerDifferencePolicy(useStaticFeatures bool) (result error) {
	if k.setupLogic != "QF_IDL" {
		return nil
	}
	features, err := k.setupStaticFeatures(useStaticFeatures)
	if err != nil {
		return err
	}
	if features != nil {
		defer func() { result = errors.Join(result, features.Close()) }()
		if features.NumArithEqualities != features.NumDiffEqualities || features.NumArithTerms != features.NumDiffTerms || features.NumArithInequalities != features.NumDiffInequalities {
			return fmt.Errorf("Benchmark is not in QF_IDL (integer difference logic).")
		}
		if features.HasReal {
			return fmt.Errorf("Benchmark has real variables but it is marked as QF_IDL (integer difference logic).")
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
	arithmetic.SmallLemmaSize = 30
	preprocess.NNFCNF = false
	if features != nil {
		dense := smtDenseArithmetic(features)
		if features.NumUninterpretedConstants > 5000 {
			smt.RelevancyLvl = 2
		} else if features.CNF && !dense {
			smt.PhaseSelection = 4 // PS_CACHING_CONSERVATIVE2
		} else {
			smt.PhaseSelection = 2 // PS_CACHING
		}
		if dense && features.NumBinClauses+features.NumUnits == features.NumClauses {
			smt.RestartAdaptive = false
			smt.RestartStrategy = 0 // RS_GEOMETRIC
		}
		if features.CNF && features.NumUnits == features.NumClauses {
			smt.RandomInitialActivity = 2 // IA_RANDOM
		}
		proofs, err := k.manager.ProofsEnabled()
		if err != nil {
			return err
		}
		if proofs {
			family, err := k.manager.ArithmeticFamilyID()
			if err != nil {
				return err
			}
			if theory, ok := k.internalizer.theories[family].(*SMTLinearArithmeticTheory); ok {
				// Both concrete specializations share this implementation.
				// Select per-column integrality before any asserted term is internalized.
				theory.integer, theory.mixed = false, true
			}
		}
		if !proofs && !arithmetic.AutoConfigSimplex && dense {
			smt.PhaseSelection = 3 // PS_CACHING_CONSERVATIVE
			// The dense theory backend remains a separate incomplete port;
			// the registered integer simplex still handles these formulas.
		}
	}
	return k.applyArithmeticSetup(smt, arithmetic, preprocess)
}
