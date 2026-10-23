// Source: theory_utvpi<rdl_ext>::mk_fresh and context::copy_plugins.
package smtcore

func (k *SMTKernel) copyUTVPIPlugin() error {
	family, err := k.manager.ArithmeticFamilyID()
	if err != nil {
		return err
	}
	theory, err := NewSMTUTVPITheory(family)
	if err != nil {
		return err
	}
	if err = k.replaceUnusedSetupArithmetic(theory); err != nil {
		return err
	}
	k.copiedArithmeticPlugin = true
	return nil
}
