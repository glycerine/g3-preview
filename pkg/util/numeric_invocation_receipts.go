// Source: src/util/mpz.cpp, mpzzp.h and mpn.cpp.
package util

// These opaque receipts identify actual primitive invocations. Their ownership
// and generation remain private; caller packages can retain and finish them.
type IntegerNormalizeInvocationReceipt = integerPNormalizeInvocationReceipt
type IntegerAddMulInvocationReceipt = integerAddMulInvocationReceipt
type IntegerCopyAllocationReceipt = integerCopyAllocationReceipt
type IntegerPowerInvocationReceipt = integerPowerInvocationReceipt
type IntegerRootInvocationReceipt = integerRootInvocationReceipt
type NaturalVerificationReceipt = naturalPNVerificationReceipt

func (r *integerPNormalizeInvocationReceipt) Matches(m *IntegerModularManager) bool {
	return m != nil && m.MatchesPNormalizeInvocation(r)
}
func (r *integerPNormalizeInvocationReceipt) SourceCause() error {
	if r == nil || r.Manager == nil {
		return nil
	}
	return r.Manager.PNormalizeInvocationSourceCause(r)
}
func (r *integerPNormalizeInvocationReceipt) Cleanup(abort bool) error {
	if r == nil || r.Manager == nil {
		return ErrNumericOwner
	}
	return r.Manager.CleanupPNormalizeInvocation(r, abort)
}
func (r *integerAddMulInvocationReceipt) Matches(m *IntegerManager) bool {
	return m != nil && m.MatchesAddMulInvocation(r)
}
func (r *integerAddMulInvocationReceipt) SourceCause() error {
	if r == nil || r.manager == nil {
		return nil
	}
	return r.manager.AddMulInvocationSourceCause(r)
}
func (r *integerAddMulInvocationReceipt) Cleanup(abort bool) error {
	if r == nil || r.manager == nil {
		return ErrNumericOwner
	}
	return r.manager.CleanupAddMulInvocation(r, abort)
}
func (r *integerRawInvocationReceipt) Matches(m *IntegerManager) bool {
	return m != nil && m.MatchesRawInvocation(r)
}
func (r *naturalPNVerificationReceipt) Matches(m *NaturalManager) bool {
	return m != nil && m.MatchesPNVerification(r)
}
func (r *naturalPNVerificationReceipt) Cleanup(abort bool) error {
	if r == nil || r.owner == nil {
		return ErrNumericOwner
	}
	return r.owner.CleanupPNVerification(r, abort)
}

// IntegerRawKind names the concrete source RAW declaration, independently of
// which polynomial operation borrows its receipt.
type IntegerRawKind uint8

const (
	IntegerRawUnknown IntegerRawKind = iota
	IntegerRawDivision
	IntegerRawGCD
	IntegerRawExtendedGCD
	IntegerRawBitwiseNot
)

func (r *integerRawInvocationReceipt) Kind() IntegerRawKind {
	if r == nil || r.Manager == nil {
		return IntegerRawUnknown
	}
	switch r.Frame {
	case &r.Manager.DivisionRaw:
		return IntegerRawDivision
	case &r.Manager.GcdRaw:
		return IntegerRawGCD
	case &r.Manager.BitwiseNotRaw:
		return IntegerRawBitwiseNot
	case &r.Manager.ExtendedRaw:
		return IntegerRawExtendedGCD
	}
	return IntegerRawUnknown
}

// NaturalBufferIdentity is a comparable diagnostic tuple. It conveys no
// authority to mutate or release the utility manager's persistent buffer.
type NaturalBufferIdentity struct {
	owner      *NaturalManager
	buffer     *SourceLimbBuffer
	invocation uint64
	private    bool
}

func (r *naturalPNBufferReceipt) Identity() NaturalBufferIdentity {
	if r == nil {
		return NaturalBufferIdentity{}
	}
	return NaturalBufferIdentity{r.Owner, r.Buffer, r.Invocation, r.private}
}
func (r *naturalPNBufferReceipt) Pending() Block {
	if r == nil {
		return Block{}
	}
	return r.PendingField
}

// Receipt borrows the identity of the real lexical verification holder.
func (s *naturalPNVerificationScope) Receipt(m *NaturalManager) (NaturalVerificationReceipt, error) {
	if m == nil || !m.ValidOwner() || s != &m.VerificationInvocation || !s.active || s.invocation == 0 {
		return NaturalVerificationReceipt{}, ErrNumericOwner
	}
	return naturalPNVerificationReceipt{m, s, s.invocation}, nil
}
func (r *integerDividesNumeralInvocationReceipt) Pending() bool {
	if r == nil || r.Manager == nil || !r.Manager.MatchesDividesNumeralInvocation(r) {
		return false
	}
	b := r.Frame
	return b.Pending(r.Manager) || b.Constructed && !b.Ended && !b.owned
}
func (m *IntegerManager) MatchesRootInvocation(r *IntegerRootInvocationReceipt) bool {
	return m.matchesRootInvocation(r)
}
func (r *integerPerfectSquareInvocationReceipt) Causes() (body, source, own error) {
	if r == nil || r.manager == nil || !r.manager.MatchesPerfectSquareHistory(r) {
		return nil, nil, nil
	}
	return r.Frame.BodyCause, r.Frame.sourceCause, r.Frame.cleanupCause
}
func (r *integerPerfectSquareInvocationReceipt) SourceCause() error {
	if r == nil || r.manager == nil {
		return nil
	}
	return r.manager.perfectSquareInvocationSourceCause(r)
}

// State reports only the invocation identified by this receipt.
func (r *integerPerfectSquareInvocationReceipt) State(m *IntegerManager) (active, endEntered, stopped bool) {
	if m == nil || !m.MatchesPerfectSquareHistory(r) {
		return false, false, false
	}
	return r.Frame.Active, r.Frame.EndEntered, r.Frame.stopped
}
