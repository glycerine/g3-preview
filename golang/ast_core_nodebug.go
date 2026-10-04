//go:build !g3debug

// Licensed under the MIT license.
package g3

const astSourceDebug = false

func (m *ASTManager) registrationBefore(r astNodeRef) (bool, error) { return false, nil }
func (m *ASTManager) registrationAfter(candidate, result astNodeRef, contains, unique bool) error {
	return nil
}
func (m *ASTManager) deletionBefore(r astNodeRef) error { return nil }
func (m *ASTManager) deletionAfter(r astNodeRef) error  { return nil }
