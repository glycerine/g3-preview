package polynomial

// CleanupBerlekampMatrix reclaims the exact matrix retained by interrupted
// source destruction, without retiring caller polynomials or the manager.
func (m *UPolynomialManager) CleanupBerlekampMatrix() error {
	if m == nil || m.self != m {
		return ErrUPolynomialOwner
	}
	if m.factorTrace != nil && m.factorTrace.executing || m.displayExecuting() || m.config.Integer.RuntimeOwner().FailureTransportActive() {
		return ErrUPolynomialCleanupPending
	}
	return m.cleanupBerlekamp()
}

// cleanupBerlekamp is the owning abort of the exact matrix retained by a
// stopped lexical destructor. It cannot adopt a different display invocation.
func (m *UPolynomialManager) cleanupBerlekamp() error {
	q := m.pendingBerlekamp
	if q == nil {
		return nil
	}
	if q.m != m {
		return ErrUPolynomialOwner
	}
	if q.displaying || q.destroying {
		return ErrUPolynomialCleanupPending
	}
	if e := m.cleanupDisplayReceipt(&q.entryTrace); e != nil {
		return e
	}
	q.destroying = true
	defer func() { q.destroying = false }()
	return q.closeOwnedMembers()
}
