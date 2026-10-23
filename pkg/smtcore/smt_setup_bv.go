// Source: src/smt/smt_setup.cpp setup_QF_BV, setup_QF_AUFBV,
// setup_QF_BVRE, setup_QF_FP and setup_QF_FPBV.
package smtcore

func (k *SMTKernel) setupBitVectorLogicPolicy() error {
	arrays := false
	switch k.setupLogic {
	case "QF_BV", "QF_BVRE", "QF_FP", "QF_FPBV", "QF_BVFP":
		k.smtParams.arith.state.Reflect = false
		k.arithState.Reflect = false
	case "QF_AUFBV", "QF_ABV", "QF_UFBV":
		arrays = true
	default:
		return nil
	}
	k.smtParams.state.RelevancyLvl = 0
	k.smtParams.bv.state.CC = false
	k.bvState = k.smtParams.bv.state
	k.smtParams.preprocess.bit.ExtGates = true
	k.smtParams.preprocess.state.NNFCNF = false
	if arrays {
		k.smtParams.array.state.Mode = 1 // AR_SIMPLE
		k.smtParams.preprocess.state.PropagateBooleans = true
	}
	for _, theory := range k.internalizer.theoryOrder {
		if bv, ok := theory.(*SMTBitVectorTheory); ok {
			if err := bv.ConfigureBitVector(k.bvState); err != nil {
				return err
			}
			bv.bitBlasterParams = k.smtParams.preprocess.bit
		}
	}
	return k.applySimpleSetupPolicy()
}
