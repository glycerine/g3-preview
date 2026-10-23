// Source: theory_array::instantiate_extensionality and theory_array_params.
package smtcore

import "github.com/glycerine/g3/pkg/ast"

func (t *SMTArrayTheory) ConfigureArray(config SMTArrayState) error {
	if t == nil {
		return ast.ErrASTHandle
	}
	t.extensional = config.Extensional
	return nil
}

func (k *SMTKernel) configureArrayTheories(config SMTArrayState) error {
	for _, theory := range k.internalizer.theoryOrder {
		if array := smtArrayCore(theory); array != nil {
			if err := array.ConfigureArray(config); err != nil {
				return err
			}
		}
	}
	return nil
}
