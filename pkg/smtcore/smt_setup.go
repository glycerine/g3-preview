// Source: src/smt/smt_setup.h and src/smt/smt_setup.cpp retained explicit
// logic routing for theories whose concrete Go solvers are not yet available.
package smtcore

import "github.com/glycerine/g3/pkg/ast"

// smtLogicUsesUnknownSetup mirrors setup::setup_default's fallthrough to
// setup_unknown. QF_FD is a retained Go specialization whose concrete finite
// domain theories are selected below.
func smtLogicUsesUnknownSetup(logic string) bool {
	switch logic {
	case "QF_UF", "QF_RDL", "QF_IDL", "QF_UFIDL", "QF_LRA", "QF_LIA", "QF_UFLIA", "QF_UFLRA",
		"QF_AX", "QF_AUFLIA", "QF_BV", "QF_AUFBV", "QF_ABV", "QF_UFBV", "QF_BVRE",
		"AUFLIA", "AUFLIRA", "AUFNIRA", "AUFLIA+", "AUFLIA-", "AUFLIRA+", "AUFLIRA-",
		"AUFNIRA+", "AUFNIRA-", "UFNIA", "UFLRA", "LRA", "QF_FP", "QF_FPBV", "QF_BVFP",
		"QF_S", "QF_DT", "QF_FD":
		return false
	default:
		return true
	}
}

func smtLogicUsesArithmetic(logic string) bool {
	if smtLogicUsesUnknownSetup(logic) {
		return true
	}
	switch logic {
	case "QF_RDL", "QF_IDL", "QF_UFIDL", "QF_LRA", "QF_LIA", "QF_UFLIA", "QF_UFLRA",
		"QF_AUFLIA", "AUFLIA", "AUFLIRA", "AUFNIRA", "AUFLIA+", "AUFLIA-", "AUFLIRA+",
		"AUFLIRA-", "AUFNIRA+", "AUFNIRA-", "UFNIA", "UFLRA", "LRA", "QF_BVRE", "QF_S", "ALL":
		return true
	default:
		return false
	}
}

func smtLogicUsesArrays(logic string) bool {
	if smtLogicUsesUnknownSetup(logic) {
		return true
	}
	switch logic {
	case "QF_AX", "QF_AUFBV", "QF_ABV", "QF_UFBV", "QF_AUFLIA", "AUFLIA", "AUFLIRA", "AUFNIRA", "UFLRA", "UFNIA",
		"AUFLIA+", "AUFLIA-", "AUFLIRA+", "AUFLIRA-", "AUFNIRA+", "AUFNIRA-":
		return true
	default:
		return false
	}
}

func smtLogicUsesBitVectors(logic string) bool {
	if smtLogicUsesUnknownSetup(logic) {
		return true
	}
	switch logic {
	case "QF_BV", "QF_UFBV", "QF_AUFBV", "QF_ABV", "QF_BVRE", "QF_FP", "QF_FPBV", "QF_BVFP", "QF_FD", "ALL":
		return true
	default:
		return false
	}
}

func smtLogicUsesDatatypes(logic string) bool {
	if smtLogicUsesUnknownSetup(logic) {
		return true
	}
	switch logic {
	case "QF_DT", "QF_FD", "ALL":
		return true
	default:
		return false
	}
}

func smtLogicUsesPseudoBooleans(logic string) bool {
	return logic == "QF_FD" || smtLogicUsesUnknownSetup(logic)
}

func smtLogicUsesDatalog(logic string) bool {
	return logic == "QF_FD" || smtLogicUsesUnknownSetup(logic)
}

func smtLogicUsesFloatingPoint(logic string) bool {
	return logic == "QF_FP" || logic == "QF_FPBV" || logic == "QF_BVFP" || smtLogicUsesUnknownSetup(logic)
}

func smtLogicUsesSequences(logic string) bool {
	return logic == "QF_BVRE" || logic == "QF_S" || smtLogicUsesUnknownSetup(logic)
}

func smtInstallRetainedSetup(kernel *SMTKernel, manager *ast.ASTManager, logic string) error {
	if kernel == nil || manager == nil {
		return ast.ErrASTHandle
	}
	kernel.setupLogic = logic
	if logic == "QF_UFLIA" && kernel.smtParams != nil && kernel.smtState.AutoConfig {
		// Source setup_QF_UFLIA mutates the owning parameter object before
		// constructing integer arithmetic. Its threshold is an internal
		// setup field, so later updt_params calls must retain it.
		kernel.smtParams.state.RelevancyLvl = 0
		kernel.smtParams.arith.state.Reflect = false
		kernel.smtParams.arith.state.PropagationThreshold = 1000
		kernel.smtParams.preprocess.state.NNFCNF = false
		kernel.smtState.RelevancyLvl = 0
		kernel.arithState.Reflect = false
		kernel.arithState.PropagationThreshold = 1000
		kernel.preprocessState.NNFCNF = false
		kernel.relevancy.enabled = false
		kernel.internalizer.relevancyLevel = 0
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
		} else if logic == "QF_IDL" && kernel.smtState.AutoConfig {
			theory, theoryErr := NewSMTLinearIntegerArithmeticTheory(family)
			if theoryErr != nil {
				return theoryErr
			}
			err = kernel.RegisterTheory(theory)
		} else if logic == "QF_UFIDL" {
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
		} else if logic == "QF_LIA" || logic == "QF_UFLIA" || logic == "QF_AUFLIA" || logic == "AUFLIA" ||
			logic == "AUFLIA+" || logic == "AUFLIA-" || logic == "UFNIA" || logic == "QF_BVRE" || logic == "QF_S" {
			theory, theoryErr := NewSMTLinearIntegerArithmeticTheory(family)
			if theoryErr != nil {
				return theoryErr
			}
			err = kernel.RegisterTheory(theory)
		} else {
			if kernel.arithState.Mode == 0 {
				err = install(family, "no arithmetic")
			} else {
				theory, theoryErr := NewSMTMixedLinearArithmeticTheory(family)
				if theoryErr != nil {
					return theoryErr
				}
				err = kernel.RegisterTheory(theory)
			}
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
	if smtLogicUsesBitVectors(logic) {
		name, err := manager.Symbols.Intern("bv")
		if err != nil {
			return err
		}
		family, err := manager.FamilyIDForSymbol(name)
		if err != nil {
			return err
		}
		theory, theoryErr := NewSMTBitVectorTheory(family)
		if theoryErr != nil {
			return theoryErr
		}
		if err = kernel.RegisterTheory(theory); err != nil {
			return err
		}
	}
	if smtLogicUsesFloatingPoint(logic) {
		if _, err := manager.RegisterArithmeticPlugin(); err != nil {
			return err
		}
		if _, err := manager.RegisterBitVectorPlugin(); err != nil {
			return err
		}
		family, err := manager.RegisterFloatingPointPlugin()
		if err != nil {
			return err
		}
		theory, theoryErr := NewSMTFloatingPointTheory(family)
		if theoryErr != nil {
			return theoryErr
		}
		if err = kernel.RegisterTheory(theory); err != nil {
			return err
		}
	}
	if smtLogicUsesSequences(logic) {
		if _, err := manager.RegisterArithmeticPlugin(); err != nil {
			return err
		}
		if _, err := manager.RegisterBitVectorPlugin(); err != nil {
			return err
		}
		if _, err := manager.RegisterArrayPlugin(); err != nil {
			return err
		}
		family, err := manager.RegisterSequencePlugin()
		if err != nil {
			return err
		}
		theory, theoryErr := NewSMTSequenceTheory(family)
		if theoryErr != nil {
			return theoryErr
		}
		if err = kernel.RegisterTheory(theory); err != nil {
			return err
		}
	}
	if smtLogicUsesDatatypes(logic) {
		family, err := manager.RegisterDatatypePlugin()
		if err != nil {
			return err
		}
		theory, theoryErr := NewSMTDatatypeTheory(family)
		if theoryErr != nil {
			return theoryErr
		}
		if err = kernel.RegisterTheory(theory); err != nil {
			return err
		}
	}
	if smtLogicUsesPseudoBooleans(logic) {
		family, err := manager.RegisterPseudoBooleanPlugin()
		if err != nil {
			return err
		}
		theory, theoryErr := NewSMTPseudoBooleanTheory(family)
		if theoryErr != nil {
			return theoryErr
		}
		if err = kernel.RegisterTheory(theory); err != nil {
			return err
		}
	}
	if smtLogicUsesDatalog(logic) {
		family, err := manager.RegisterDatalogPlugin()
		if err != nil {
			return err
		}
		theory, theoryErr := NewSMTDatalogTheory(family)
		if theoryErr != nil {
			return theoryErr
		}
		if err = kernel.RegisterTheory(theory); err != nil {
			return err
		}
	}
	return nil
}
