// Source: src/util/rational.cpp:75-83; src/util/rational.h:67.
// Leonardo de Moura (2006-09-18). Licensed under the MIT license; see LICENSE.txt.
package mpnum

import "github.com/glycerine/g3/golang/arena"

func (s *RationalService) releaseOuterLeaf(a *Int, source bool) error {
	if source && s.manager == nil {
		s.sourceGuard = true
		s.sourceCause = ErrRationalServiceNoManager
		return s.sourceCause
	}
	if !source {
		return s.manager.z.Del(a)
	}
	guard, e := sourceNumeralDel(s.manager.z, a)
	if guard {
		s.sourceGuard = true
		s.sourceCause = e
	}
	return e
}
func (s *RationalService) SourceFinalize() error {
	if s == nil || s.self != s {
		return ErrRationalServiceCopy
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sourceFinalizeLocked()
}

// Caller owns the existing service mutex; owning cleanup can resume without
// recursively acquiring a public lock or returning through a destroyed child.
func (s *RationalService) sourceFinalizeLocked() error {
	if s.closed || s.sourceFinalized {
		return nil
	}
	if s.closing && !s.sourceFinalizing {
		return arena.ErrClosed
	}
	if s.lexical.pending() {
		return ErrRationalLexicalPending
	}
	s.sourceFinalizing = true
	if s.sourceGuard {
		return s.sourceCause
	}
	fail := func(e error) error { s.sourceCause = e; s.cleanupCause = e; return e }
	if e := s.cleanupOuter(true); e != nil {
		return fail(e)
	}
	if e := s.manager.SourceFinalize(); e != nil {
		if s.manager.sourceGuard {
			s.sourceGuard = true
		}
		return fail(e)
	}
	s.sourceFinalized = true
	s.sourceCause = nil
	s.cleanupCause = nil
	return nil
}
