// Licensed under the MIT license.
//go:build !g3debug

package mpnum

const sourceExtendedGCDDebug = false

// Z3DEBUG is disabled independently of checked storage handles.
func (m *IntegerManager) extendedGCDDiagnostic(r1, r2, a, b, expected, nexta, nextb *Int) error {
	return m.extendedGCDDiagnosticWithAllocationContext(r1, r2, a, b, expected, nexta, nextb, integerAllocationInvocationContext{})
}
func (m *IntegerManager) extendedGCDDiagnosticWithAllocationContext(r1, r2, a, b, expected, nexta, nextb *Int, context integerAllocationInvocationContext) error {
	return nil
}
