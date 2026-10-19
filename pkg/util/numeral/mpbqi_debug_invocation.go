// Source: basic_interval arithmetic's actual MPBQ_DEBUG children, retained by
// algebraic_numbers.cpp:1034 and its saved-interval destructors.
package numeral

type BinaryIntervalDebugInvocationReceipt struct {
	owner      *BinaryIntervalManager
	invocation uint64
}
type binaryIntervalDebugChildren struct {
	owned, integer, arithmetic uint64
	snapshots                  *OwnedRational
}

func (s *binaryIntervalDebugChildren) capture(b *BinaryRationalManager) {
	*s = binaryIntervalDebugChildren{}
	if b.toOwnedRationalScope.pending {
		s.owned = b.toOwnedInvocation
	}
	if b.debugIntegerPending {
		s.integer = b.debugIntegerInvocation
	}
	if b.debugArithmeticPending {
		s.arithmetic = b.debugArithmeticInvocation
	}
	if n := len(b.debugSnapshotQueue); n != 0 {
		s.snapshots = &b.debugSnapshotQueue[n-1]
	}
}
func (s *binaryIntervalDebugChildren) ownsSnapshots(b *BinaryRationalManager) bool {
	if s.snapshots != nil {
		for i := range b.debugSnapshotQueue {
			if &b.debugSnapshotQueue[i] == s.snapshots {
				return true
			}
		}
	}
	return false
}
func (s *binaryIntervalDebugChildren) pending(b *BinaryRationalManager) bool {
	return s.owned != 0 && s.owned == b.toOwnedInvocation && b.toOwnedRationalScope.pending || s.integer != 0 && s.integer == b.debugIntegerInvocation && b.debugIntegerPending || s.arithmetic != 0 && s.arithmetic == b.debugArithmeticInvocation && b.debugArithmeticPending || s.ownsSnapshots(b)
}

func (m *BinaryIntervalManager) AddWithDebugReceipt(a, b, out *BinaryInterval, r *BinaryIntervalDebugInvocationReceipt) error {
	return m.arithmeticWithDebugReceipt(a, b, out, 0, r)
}
func (m *BinaryIntervalManager) SubWithDebugReceipt(a, b, out *BinaryInterval, r *BinaryIntervalDebugInvocationReceipt) error {
	return m.arithmeticWithDebugReceipt(a, b, out, 1, r)
}
func (m *BinaryIntervalManager) MulWithDebugReceipt(a, b, out *BinaryInterval, r *BinaryIntervalDebugInvocationReceipt) error {
	return m.arithmeticWithDebugReceipt(a, b, out, 2, r)
}
func (m *BinaryIntervalManager) arithmeticWithDebugReceipt(a, b, out *BinaryInterval, kind uint8, r *BinaryIntervalDebugInvocationReceipt) (cause error) {
	if m != nil && m.self == m && (m.debugExecuting || m.debugPending) {
		return ErrBinaryDebugSnapshotPending
	}
	if e := m.triple(a, b, out); e != nil {
		return e
	}
	if m.debugInvocation == ^uint64(0) {
		return ErrBinaryDebugSnapshotPending
	}
	m.debugInvocation++
	m.debugSourceCause = nil
	if r != nil {
		*r = BinaryIntervalDebugInvocationReceipt{m, m.debugInvocation}
	}
	m.debugExecuting = true
	defer func() {
		b := m.bounds
		m.debugChildren.capture(b)
		m.debugPending = b.toOwnedRationalScope.pending || b.debugIntegerPending || b.debugArithmeticPending || len(b.debugSnapshotQueue) != 0
		m.debugSourceCause = b.toOwnedRationalScope.sourceCause
		if m.debugSourceCause == nil && b.debugArithmeticPending {
			m.debugSourceCause = b.debugArithmeticCall.SourceReady()
		}
		if m.debugSourceCause == nil && (b.debugIntegerPending || len(b.debugSnapshotQueue) != 0) {
			m.debugSourceCause = cause
		}
		m.debugExecuting = false
	}()
	switch kind {
	case 0:
		return m.Add(a, b, out)
	case 1:
		return m.Sub(a, b, out)
	default:
		return m.Mul(a, b, out)
	}
}
func (m *BinaryIntervalManager) matchesDebugInvocation(r *BinaryIntervalDebugInvocationReceipt) bool {
	return m != nil && m.self == m && r != nil && r.owner == m && r.invocation != 0 && r.invocation == m.debugInvocation
}
func (m *BinaryIntervalManager) DebugInvocationPending(r *BinaryIntervalDebugInvocationReceipt) bool {
	return m.matchesDebugInvocation(r) && m.debugPending && m.debugChildren.pending(m.bounds)
}
func (m *BinaryIntervalManager) DebugInvocationSourceCause(r *BinaryIntervalDebugInvocationReceipt) error {
	if m.matchesDebugInvocation(r) {
		return m.debugSourceCause
	}
	return nil
}
func (m *BinaryIntervalManager) CleanupDebugInvocation(r *BinaryIntervalDebugInvocationReceipt) error {
	if !m.matchesDebugInvocation(r) {
		return nil
	}
	if m.debugExecuting {
		return ErrBinaryDebugSnapshotPending
	}
	if !m.debugPending {
		return nil
	}
	b, s := m.bounds, &m.debugChildren
	if s.owned != 0 && s.owned == b.toOwnedInvocation && b.toOwnedRationalScope.pending {
		if e := b.CleanupToOwnedRational(); e != nil {
			return e
		}
	}
	if s.integer != 0 && s.integer == b.debugIntegerInvocation && b.debugIntegerPending {
		if e := b.CleanupDebugInteger(); e != nil {
			return e
		}
	}
	if s.arithmetic != 0 && s.arithmetic == b.debugArithmeticInvocation && b.debugArithmeticPending || s.ownsSnapshots(b) {
		if e := b.CleanupDebugSnapshots(); e != nil {
			return e
		}
	}
	m.debugPending = false
	return nil
}
