// Source: src/smt/smt_setup.h and src/smt/smt_setup.cpp retained explicit
// logic routing for theories whose concrete Go solvers are not yet available.
package smtcore

import "github.com/glycerine/g3/pkg/ast"

func smtLogicUsesArithmetic(logic string) bool {
	switch logic {
	case "QF_RDL", "QF_IDL", "QF_UFIDL", "QF_LRA", "QF_LIA", "QF_UFLIA", "QF_UFLRA",
		"QF_AUFLIA", "AUFLIA", "AUFLIRA", "AUFNIRA", "AUFLIA+", "AUFLIA-", "AUFLIRA+",
		"AUFLIRA-", "AUFNIRA+", "AUFNIRA-", "UFNIA", "UFLRA", "LRA", "QF_BVRE":
		return true
	default:
		return false
	}
}

func smtLogicUsesArrays(logic string) bool {
	switch logic {
	case "QF_AX", "QF_AUFBV", "QF_ABV", "QF_AUFLIA", "AUFLIA", "AUFLIRA", "AUFNIRA",
		"AUFLIA+", "AUFLIA-", "AUFLIRA+", "AUFLIRA-", "AUFNIRA+", "AUFNIRA-":
		return true
	default:
		return false
	}
}

func smtInstallRetainedSetup(kernel *SMTKernel, manager *ast.ASTManager, logic string) error {
	if kernel == nil || manager == nil {
		return ast.ErrASTHandle
	}
	install := func(family ast.FamilyID, name string) error {
		dummy, err := NewSMTDummyTheory(family, name)
		if err != nil {
			return err
		}
		return kernel.RegisterTheory(dummy)
	}
	if smtLogicUsesArithmetic(logic) {
		family, err := manager.ArithmeticFamilyID()
		if err != nil {
			return err
		}
		if logic == "QF_RDL" {
			theory, theoryErr := NewSMTRealDiffTheory(family)
			if theoryErr != nil {
				return theoryErr
			}
			err = kernel.RegisterTheory(theory)
		} else if logic == "QF_IDL" || logic == "QF_UFIDL" {
			theory, theoryErr := NewSMTIntegerDiffTheory(family)
			if theoryErr != nil {
				return theoryErr
			}
			err = kernel.RegisterTheory(theory)
		} else if logic == "QF_LRA" || logic == "QF_UFLRA" {
			theory, theoryErr := NewSMTLinearArithmeticTheory(family)
			if theoryErr != nil {
				return theoryErr
			}
			err = kernel.RegisterTheory(theory)
		} else if logic == "QF_LIA" || logic == "QF_UFLIA" {
			theory, theoryErr := NewSMTLinearIntegerArithmeticTheory(family)
			if theoryErr != nil {
				return theoryErr
			}
			err = kernel.RegisterTheory(theory)
		} else {
			err = install(family, "arithmetic")
		}
		if err != nil {
			return err
		}
	}
	if smtLogicUsesArrays(logic) {
		family, err := manager.ArrayFamilyID()
		if err != nil {
			return err
		}
		theory, theoryErr := NewSMTArrayTheory(family)
		if theoryErr != nil {
			return theoryErr
		}
		if err = kernel.RegisterTheory(theory); err != nil {
			return err
		}
	}
	return nil
}
