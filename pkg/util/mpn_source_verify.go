// Licensed under the MIT license.
// Source: src/util/mpn.cpp:206-217 and actual Algorithm M/A bodies.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

// sourceVerifyDivision is the source Z3DEBUG lexical temp, after both outputs.
// The embedded charged holder retains a failed lexical construction/destruction
// only for explicit Go owning cleanup. Successful calls destroy it immediately;
// its allocation is never reused as source manager scratch.

func (m *NaturalManager) sourceVerifyDivision(a, b, q, r limbView) error {
	return m.sourceVerifyDivisionWithPNReceipt(a, b, q, r, nil)
}
func (m *NaturalManager) sourceVerifyDivisionWithPNReceipt(a, b, q, r limbView, pn *integerPNormalizeScope) (err error) {
	return m.sourceVerifyDivisionWithAllocationContext(a, b, q, r, pnAllocationContext(pn))
}
func (m *NaturalManager) sourceVerifyDivisionWithAllocationContext(a, b, q, r limbView, context integerAllocationInvocationContext) (err error) {

	if !modularSourceDebug {
		return nil
	}
	// Guard reuse of this retained lexical holder only when the source reaches
	// its constructor, after the quotient and remainder have been published.
	if m.VerificationActive {
		return arena.ErrInvalidSlot
	}
	pn := context.pn
	var frame *naturalPNVerificationScope
	if !context.hasLocalInvocation() && context.Caller != nil && context.Caller.hasVerification() {
		var e error
		frame, e = context.Caller.callVerification(m)
		if e != nil {
			return e
		}
		if frame == nil || frame != &m.VerificationInvocation {
			return ErrNumericOwner
		}
		defer func() { frame.executing = false }()
	}

	if r := context.BoolDivides; r != nil && r.Manager != nil && r.Manager.Natural == m && r.Manager.dividesNumeralContextRoots(context) != nil {
		frame = &m.VerificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrNumericPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, boolDivides: *r}
		frame.Buffer = naturalPNBufferReceipt{Owner: m, BoolDivides: *r, Invocation: r.Invocation, Buffer: &m.Verification, private: true}
		r.Frame.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}
	if context.raw == nil && pn != nil && pn.owner != nil && pn.owner.manager.Natural == m && pn == &pn.owner.Normalization && pn.active && pn.executing {
		frame = &m.VerificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrNumericPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, parent: pn}
		frame.Buffer = naturalPNBufferReceipt{Owner: m, context: pn, Invocation: pn.invocation, Buffer: &m.Verification, stage: pn.Stage, private: true}
		pn.Verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}

	if r := context.raw; r != nil && r.Manager != nil && r.Manager.Natural == m && context.rawReceipt(r.Manager) != nil {
		frame = &m.VerificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrNumericPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, raw: *r}
		frame.Buffer = naturalPNBufferReceipt{Owner: m, raw: *r, Invocation: r.Invocation, Buffer: &m.Verification, private: true}
		r.Frame.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}

	if context.raw == nil && context.mpfFormat != nil {
		q := context.formatFrame(context.mpfFormat.manager.integer, true)
		if q == nil || q.manager.integer.Natural != m {
			return ErrNumericOwner
		}
		frame = &m.VerificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrNumericPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, formatContext: context}
		frame.Buffer = naturalPNBufferReceipt{Owner: m, formatContext: context, Invocation: q.invocation, Buffer: &m.Verification, private: true}
		receipt := naturalPNVerificationReceipt{m, frame, frame.invocation}
		if context.formatDecimal != nil {
			context.formatDecimal.verification = receipt
		} else {
			q.verification = receipt
		}
		defer func() { frame.executing = false }()
	}

	if context.raw == nil && context.mpfRationalSet != nil {
		s := context.rationalSetFrame(context.mpfRationalSet.manager.integer, true)
		if s == nil || s.manager.integer.Natural != m {
			return ErrNumericOwner
		}
		frame = &m.VerificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrNumericPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, rationalSetContext: context}
		frame.Buffer = naturalPNBufferReceipt{Owner: m, rationalSetContext: context, Invocation: s.invocation, Buffer: &m.Verification, private: true}
		s.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}
	if context.mpfRound != nil {
		s := context.roundFrame(context.mpfRound.Manager.integer, true)
		if s == nil || s.Manager.integer.Natural != m {
			return ErrNumericOwner
		}
		frame = &m.VerificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrNumericPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, roundContext: context}
		frame.Buffer = naturalPNBufferReceipt{Owner: m, roundContext: context, Invocation: s.Invocation, Buffer: &m.Verification, private: true}
		s.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}
	m.VerificationActive = true
	temp := &m.Verification
	var buffer *naturalPNBufferReceipt
	if frame != nil {
		buffer = &frame.Buffer
	}
	if e := temp.resizeWithPNReceipt(m.runtime, a.size+1, buffer); e != nil {
		return e
	}
	if frame != nil {
		frame.Constructed = true
		defer func() { err = JoinNumericErrors(err, m.FinishPNVerificationSource(frame)) }()
	}
	out := limbView{runtime: m.runtime, source: temp, size: a.size + 1}
	// mpn_manager::mul, Knuth M: initialize lnga cells, then every column.
	for i := uint32(0); i < q.size; i++ {
		out.write(i, 0)
	}
	for j := uint32(0); j < b.size; j++ {
		digit := b.read(j)
		if digit == 0 {
			out.write(j+q.size, 0)
			continue
		}
		var carry uint32
		for i := uint32(0); i < q.size; i++ {
			value := uint64(q.read(i))*uint64(digit) + uint64(out.read(i+j)) + uint64(carry)
			out.write(i+j, uint32(value))
			carry = uint32(value >> 32)
		}
		out.write(j+q.size, carry)
	}
	// Source aliasing add reads temp at each position before writing that cell.
	var carry uint32
	for i := uint32(0); i < a.size; i++ {
		u := out.read(i)
		var v uint32
		if i < r.size {
			v = r.read(i)
		}
		sum := u + v
		c1 := sum < u
		value := sum + carry
		c2 := value < sum
		out.write(i, value)
		carry = 0
		if c1 || c2 {
			carry = 1
		}
	}
	out.write(a.size, carry)
	// The source add also walks the normalized size even though the caller does
	// not consume it. Preserve these reads rather than replacing the algorithm.
	realSize := a.size + 1
	for realSize > 1 && out.read(realSize-1) == 0 {
		realSize--
	}
	ok := true
	for i := uint32(0); i < a.size && ok; i++ {
		if out.read(i) != a.read(i) {
			ok = false
		}
	}
	if out.read(a.size) != 0 {
		ok = false
	}
	if !ok {
		return ErrIntegerAssertion
	}
	if frame != nil {
		return nil
	}
	if e := temp.Finalize(m.runtime); e != nil {
		return e
	}
	m.VerificationActive = false
	return nil
}
