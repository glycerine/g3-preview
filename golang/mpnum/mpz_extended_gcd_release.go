//go:build !g3debug

// Copyright (c) 2006 Microsoft Corporation
package mpnum

const sourceExtendedGCDDebug = false

// Z3DEBUG is disabled independently of checked storage handles.
func (m *IntegerManager) extendedGCDDiagnostic(r1, r2, a, b, expected, nexta, nextb *Int) error {
	return nil
}
