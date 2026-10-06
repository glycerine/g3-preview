// Licensed under the MIT license.
// Source: src/util/mpn.cpp:206-217 and actual Algorithm M/A bodies.
package mpnum

import "github.com/glycerine/g3/golang/arena"

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
	if m.verificationActive {
		return arena.ErrInvalidSlot
	}
	pn := context.pn
	var frame *naturalPNVerificationScope
	if q := context.rootUpperBound; context.pn == nil && context.raw == nil && q != nil {
		z := q.owner.config.Integer
		if z.natural != m || context.rootUpperBoundFrame(z) == nil {
			return ErrPolynomialOwner
		}
		frame = &m.verificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, rootUpperBound: q, rootUpperInvocation: q.invocation}
		frame.buffer = naturalPNBufferReceipt{owner: m, rootUpperBound: q, rootUpperInvocation: q.invocation, invocation: q.invocation, buffer: &m.verification, private: true}
		q.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}

	if r := context.boolDivides; r != nil && r.manager != nil && r.manager.natural == m && r.manager.dividesNumeralContextRoots(context) != nil {
		frame = &m.verificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, boolDivides: *r}
		frame.buffer = naturalPNBufferReceipt{owner: m, boolDivides: *r, invocation: r.invocation, buffer: &m.verification, private: true}
		r.frame.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}
	if context.raw == nil && pn != nil && pn.owner != nil && pn.owner.manager.natural == m && pn == &pn.owner.normalization && pn.active && pn.executing {
		frame = &m.verificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, parent: pn}
		frame.buffer = naturalPNBufferReceipt{owner: m, context: pn, invocation: pn.invocation, buffer: &m.verification, stage: pn.stage, private: true}
		pn.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}

	if r := context.divides; context.raw == nil && r != nil && r.core != nil && r.core.integer.natural == m && r.core.integer.dividesContextRoots(context) != nil {
		frame = &m.verificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, divides: *r}
		frame.buffer = naturalPNBufferReceipt{owner: m, divides: *r, invocation: r.invocation, buffer: &m.verification, private: true}
		r.frame.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}
	if r := context.scalarExact; context.raw == nil && r != nil && r.core != nil && r.core.integer.natural == m && r.core.integer.scalarExactContextRoots(context) != nil {
		frame = &m.verificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, scalarExact: *r}
		frame.buffer = naturalPNBufferReceipt{owner: m, scalarExact: *r, invocation: r.invocation, buffer: &m.verification, private: true}
		r.frame.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}
	if r := context.exactDiv; context.raw == nil && r != nil && r.core != nil && r.core.integer.natural == m && r.core.integer.exactDivContextRoots(context) != nil {
		frame = &m.verificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, exactDiv: *r}
		frame.buffer = naturalPNBufferReceipt{owner: m, exactDiv: *r, invocation: r.invocation, buffer: &m.verification, private: true}
		r.frame.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}
	if context.factorFrontend != nil && context.raw == nil && context.boolDivides == nil && context.pn == nil {
		q := context.factorFrontend
		if q.service == nil || q.service.manager == nil || q.service.manager.config.Integer.natural != m || context.factorFrontendFrame(q.service.manager.config.Integer) == nil {
			return ErrPolynomialOwner
		}
		frame = &m.verificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, factorFrontendContext: context}
		frame.buffer = naturalPNBufferReceipt{owner: m, factorFrontendContext: context, invocation: q.invocation, buffer: &m.verification, private: true}
		q.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}
	if q := context.primePreparation; context.raw == nil && q != nil && q.owner.manager.config.Integer.natural == m && context.primePreparationFrame(q.owner.manager.config.Integer) != nil {
		frame = &m.verificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, primePreparationContext: context}
		frame.buffer = naturalPNBufferReceipt{owner: m, primePreparationContext: context, invocation: q.invocation, buffer: &m.verification, private: true}
		q.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}
	if q := context.recombination; context.raw == nil && q != nil && q.owner.manager.config.Integer.natural == m && context.recombinationFrame(q.owner.manager.config.Integer) != nil {
		frame = &m.verificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, recombinationContext: context}
		frame.buffer = naturalPNBufferReceipt{owner: m, recombinationContext: context, invocation: q.invocation, buffer: &m.verification, private: true}
		q.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}
	if q := context.henselCaller; context.raw == nil && q != nil && q.owner.manager.config.Integer.natural == m && context.henselCallerFrame(q.owner.manager.config.Integer) != nil {
		frame = &m.verificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, henselCallerContext: context}
		frame.buffer = naturalPNBufferReceipt{owner: m, henselCallerContext: context, invocation: q.invocation, buffer: &m.verification, private: true}
		q.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}
	if r := context.henselPair; context.raw == nil && r != nil && r.service != nil && r.service.manager != nil && r.service.manager.config.Integer.natural == m && context.henselPairReceipt(r.service.manager.config.Integer) != nil {
		frame = &m.verificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, henselPair: *r}
		frame.buffer = naturalPNBufferReceipt{owner: m, henselPair: *r, henselManager: context.henselManager, invocation: r.invocation, buffer: &m.verification, private: true}
		r.frame.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}
	if r := context.somNormalize; context.raw == nil && r != nil && r.manager != nil && r.manager.hasSOMNormalizeInvocationHistory(r) && r.manager.integer != nil && r.manager.integer.natural == m && context.somNormalizeReceipt(r.manager.integer) != nil {
		frame = &m.verificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, somNormalize: *r}
		frame.buffer = naturalPNBufferReceipt{owner: m, somNormalize: *r, invocation: r.invocation, buffer: &m.verification, private: true}
		r.frame.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}
	if r := context.normalization; context.raw == nil && r != nil && r.manager != nil && r.manager.hasNormalizeInvocationHistory(r) && r.manager.core.integer != nil && r.manager.core.integer.natural == m && context.normalizationReceipt(r.manager.core.integer) != nil {
		frame = &m.verificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, normalization: *r}
		frame.buffer = naturalPNBufferReceipt{owner: m, normalization: *r, invocation: r.invocation, buffer: &m.verification, private: true}
		r.frame.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}
	if r := context.content; context.raw == nil && r != nil && r.manager != nil && r.manager.integer != nil && r.manager.integer.natural == m && context.contentReceipt(r.manager.integer) != nil {
		frame = &m.verificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, content: *r}
		frame.buffer = naturalPNBufferReceipt{owner: m, content: *r, invocation: r.invocation, buffer: &m.verification, private: true}
		r.frame.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}
	if r := context.scalarDiv; context.raw == nil && r != nil && r.manager != nil && r.manager.integer.natural == m && r.manager.integer.scalarDivAllocationRoots(r) != nil {
		frame = &m.verificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, scalar: *r}
		frame.buffer = naturalPNBufferReceipt{owner: m, scalar: *r, invocation: r.invocation, buffer: &m.verification, private: true}
		r.frame.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}
	if r := context.raw; r != nil && r.manager != nil && r.manager.natural == m && context.rawReceipt(r.manager) != nil {
		frame = &m.verificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, raw: *r}
		frame.buffer = naturalPNBufferReceipt{owner: m, raw: *r, invocation: r.invocation, buffer: &m.verification, private: true}
		r.frame.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}

	if context.raw == nil && context.mpfFormat != nil {
		q := context.formatFrame(context.mpfFormat.manager.integer, true)
		if q == nil || q.manager.integer.natural != m {
			return ErrPolynomialOwner
		}
		frame = &m.verificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, formatContext: context}
		frame.buffer = naturalPNBufferReceipt{owner: m, formatContext: context, invocation: q.invocation, buffer: &m.verification, private: true}
		receipt := naturalPNVerificationReceipt{m, frame, frame.invocation}
		if context.formatDecimal != nil {
			context.formatDecimal.verification = receipt
		} else {
			q.verification = receipt
		}
		defer func() { frame.executing = false }()
	}

	if context.mpfRound != nil {
		s := context.roundFrame(context.mpfRound.manager.integer, true)
		if s == nil || s.manager.integer.natural != m {
			return ErrPolynomialOwner
		}
		frame = &m.verificationInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*frame = naturalPNVerificationScope{active: true, executing: true, invocation: frame.invocation + 1, roundContext: context}
		frame.buffer = naturalPNBufferReceipt{owner: m, roundContext: context, invocation: s.invocation, buffer: &m.verification, private: true}
		s.verification = naturalPNVerificationReceipt{m, frame, frame.invocation}
		defer func() { frame.executing = false }()
	}
	m.verificationActive = true
	temp := &m.verification
	var buffer *naturalPNBufferReceipt
	if frame != nil {
		buffer = &frame.buffer
	}
	if e := temp.resizeWithPNReceipt(m.runtime, a.size+1, buffer); e != nil {
		return e
	}
	if frame != nil {
		frame.constructed = true
		defer func() { err = polynomialNewtonJoin(err, m.finishPNVerificationSource(frame)) }()
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
	if e := temp.finalize(m.runtime); e != nil {
		return e
	}
	m.verificationActive = false
	return nil
}
