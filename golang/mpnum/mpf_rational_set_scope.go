// Source: src/util/mpf.cpp:233-300; src/util/mpq.cpp:312-315.
// Licensed under the MIT license. Source authors: Christoph Wintersteiger; Leonardo de Moura.
package mpnum

import "github.com/glycerine/g3/golang/memory"

// These are the actual reached lexical holders. Branch p2 and the nested
// PrevPowerOfTwo NUM finish before the later p/t/sq declarations.
type arbitraryFloatRationalSetScope struct {
	receipt                                                  *ArbitraryFloatRationalSetReceipt
	history                                                  *arbitraryFloatRationalSetHistory
	buffers                                                  [4]naturalPNBufferReceipt
	verification                                             naturalPNVerificationReceipt
	roundChild                                               ArbitraryFloatRoundReceipt
	roundFrame                                               *arbitraryFloatRoundScope
	roundInvocation                                          uint64
	gcd, division, extended                                  integerRawInvocationReceipt
	manager                                                  *ArbitraryFloatManager
	invocation                                               uint64
	sig, t, sq                                               Rational
	exp, p2, p, prev                                         Int
	sigLive, expLive, p2Live, pLive, tLive, sqLive, prevLive bool
	active, executing, finishing                             bool
	prevSourceCause                                          error
	bodyCause, sourceCause, ownCause                         error
}

func (m *ArbitraryFloatManager) finishRationalSetSource() error {
	s := &m.rationalSet
	if s.sourceCause != nil {
		return s.sourceCause
	}
	if !s.active {
		return nil
	}
	if s.executing || s.finishing {
		return ErrPolynomialPending
	}
	s.finishing = true
	defer func() { s.finishing = false; s.recordHistory() }()
	if e := m.rationalSetChildrenSource(s); e != nil {
		s.sourceCause = e
		return e
	}
	for _, a := range []struct {
		v    *Int
		live *bool
	}{{&s.prev, &s.prevLive}, {&s.p2, &s.p2Live}} {
		if *a.live {
			if _, e := sourceNumeralDel(m.integer, a.v); e != nil {
				s.sourceCause = e
				return e
			}
			*a.live = false
		}
	}
	for _, a := range []struct {
		q    *Rational
		v    *Int
		live *bool
	}{{q: &s.sq, live: &s.sqLive}, {q: &s.t, live: &s.tLive}, {v: &s.p, live: &s.pLive}, {v: &s.exp, live: &s.expLive}, {q: &s.sig, live: &s.sigLive}} {
		if !*a.live {
			continue
		}
		if a.q != nil { // mpq::del: numerator, then denominator.
			if _, e := sourceNumeralDel(m.integer, &a.q.num); e != nil {
				s.sourceCause = e
				return e
			}
			if _, e := sourceNumeralDel(m.integer, &a.q.den); e != nil {
				s.sourceCause = e
				return e
			}
		} else if _, e := sourceNumeralDel(m.integer, a.v); e != nil {
			s.sourceCause = e
			return e
		}
		*a.live = false
	}

	s.active = false
	if m.integer.natural.matchesPNVerification(&s.verification) {
		s.active = true
	}
	for i := range s.buffers {
		if s.buffers[i].pending != (memory.Block{}) {
			s.active = true
		}
	}
	if m.integer.matchesRawInvocation(&s.extended) || m.integer.matchesRawInvocation(&s.division) || m.integer.matchesRawInvocation(&s.gcd) {
		s.active = true
	}
	if s.roundFrame == &m.roundScope && s.roundInvocation == m.roundScope.invocation && (m.roundScope.sigmaConstructed || m.roundScope.capConstructed || m.roundScope.stickyConstructed) {
		s.active = true
	}
	return nil
}
func (m *ArbitraryFloatManager) CleanupExponentRationalSet() error {
	if e := m.CleanupExponentRationalSetWithReceipt(nil, false); e != nil {
		return e
	}
	return m.CleanupExponentRationalSetWithReceipt(nil, true)
}
func (m *ArbitraryFloatManager) CleanupExponentRationalSetWithReceipt(r *ArbitraryFloatRationalSetReceipt, abort bool) error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	s := &m.rationalSet

	if s.executing || s.finishing || (m.integer != nil && m.integer.memory.FailureTransportActive()) {
		return ErrPolynomialPending
	}
	var selected *arbitraryFloatRationalSetHistory
	if r != nil {
		if *r == (ArbitraryFloatRationalSetReceipt{}) {
			return nil
		}
		var e error
		selected, e = m.rationalSetHistory(r)
		if e != nil {
			return e
		}
		if selected.owned {
			return nil
		}
		if selected != s.history {
			return m.cleanupRationalSetHistory(selected, abort)
		}
	}
	// Before this literal setter has entered, its source scope and child
	// tuples do not exist. MPF construction may still lack Integer/Natural.
	// Issued receipt validation above remains independent of this erasure.
	if s.manager == nil && s.invocation == 0 {
		return nil
	}
	if s.manager != m || s.invocation == 0 {
		return memory.ErrRuntimeOwner
	}
	if s.executing || s.finishing || (m.integer != nil && m.integer.memory.FailureTransportActive()) {
		return ErrPolynomialPending
	}
	s.finishing = true
	defer func() { s.finishing = false; s.recordHistory() }()
	stop := func(e error) error {
		if e != nil && s.ownCause == nil {
			s.ownCause = e
		}
		return e
	}
	if e := m.cleanupRationalSetChildren(s, false); e != nil {
		return stop(e)
	}
	for _, a := range []struct {
		q    *Rational
		v    *Int
		live *bool
	}{{v: &s.prev, live: &s.prevLive}, {v: &s.p2, live: &s.p2Live}, {q: &s.sq, live: &s.sqLive}, {q: &s.t, live: &s.tLive}, {v: &s.p, live: &s.pLive}, {v: &s.exp, live: &s.expLive}, {q: &s.sig, live: &s.sigLive}} {
		if !*a.live {
			continue
		}
		var e error
		if a.q != nil {
			e = m.rational.Del(a.q)
		} else {
			e = m.integer.Del(a.v)
		}
		if e != nil {
			return stop(e)
		}
		*a.live = false
	}
	if !abort {
		return nil
	}
	if e := m.cleanupRationalSetChildren(s, true); e != nil {
		return stop(e)
	}

	for h := m.rationalSetHistories; h != nil; h = h.next {
		if selected != nil && h != selected {
			continue
		}
		if !h.owned {
			if e := m.cleanupRationalSetHistory(h, true); e != nil {
				return stop(e)
			}
		}
	}

	s.active = false
	return nil
}
