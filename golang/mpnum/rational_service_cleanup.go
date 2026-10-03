// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package mpnum

import "github.com/glycerine/g3/golang/memory"

func (s *RationalService) Error() string { return "mpnum: retained rational service" }
func (s *RationalService) Unwrap() error {
	if s.originalCause != nil {
		return s.originalCause
	}
	return s.cleanupCause
}
func (s *RationalService) OriginalCause() error { return s.originalCause }
func (s *RationalService) CleanupCause() error  { return s.cleanupCause }
func (s *RationalService) RuntimeOwner() *memory.Runtime {
	if s == nil || s.self != s || s.manager == nil {
		return nil
	}
	return s.manager.RuntimeOwner()
}
func (s *RationalService) Cleanup() error { return s.CloseChecked() }
func (s *RationalService) CloseChecked() error {
	if s == nil || s.self != s {
		return ErrRationalServiceCopy
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		s.cleanupCause = nil
		return nil
	}
	if e := s.lexical.cleanup(s.manager.z); e != nil {
		s.cleanupCause = e
		return s
	}
	s.closing = true
	fail := func(e error) error { s.cleanupCause = e; return s }
	if s.sourceFinalizing && !s.sourceFinalized && !s.sourceGuard {
		if e := s.sourceFinalizeLocked(); e != nil {
			return fail(e)
		}
	}
	if !s.sourceGuard {
		if e := s.cleanupOuter(false); e != nil {
			return fail(e)
		}
	}
	if e := s.manager.CloseChecked(); e != nil {
		return fail(e)
	}
	if s.sourceGuard {
		s.cache = nil
		s.manager.budget.Release(s.cacheBytes)
		s.cacheBytes = 0
		s.infZero = OwnedInfRational{}
		s.infOne = OwnedInfRational{}
		s.infMinusOne = OwnedInfRational{}
		s.infIntZero = OwnedInfIntRational{}
		s.infIntOne = OwnedInfIntRational{}
		s.infIntMinusOne = OwnedInfIntRational{}
		s.zero = Rational{}
		s.one = Rational{}
		s.minusOne = Rational{}
	}
	s.manager.budget.Release(s.headerBytes)
	s.headerBytes = 0
	s.closed = true
	s.cleanupCause = nil
	return nil
}

func (s *RationalService) cleanupOuter(source bool) error {
	// rational.cpp finalization hooks, then ascending cache and constants.
	// inf_rational members destroy SECOND before FIRST, each NUM before DEN.
	infRoots := [18]*Int{
		&s.infZero.second.num, &s.infZero.second.den, &s.infZero.first.num, &s.infZero.first.den,
		&s.infOne.second.num, &s.infOne.second.den, &s.infOne.first.num, &s.infOne.first.den,
		&s.infMinusOne.second.num, &s.infMinusOne.second.den, &s.infMinusOne.first.num, &s.infMinusOne.first.den,
		&s.infIntZero.first.num, &s.infIntZero.first.den, &s.infIntOne.first.num, &s.infIntOne.first.den, &s.infIntMinusOne.first.num, &s.infIntMinusOne.first.den}
	for s.cleanupStage < 18 {
		if e := s.releaseOuterLeaf(infRoots[s.cleanupStage], source); e != nil {
			return e
		}
		s.cleanupStage++
		switch s.cleanupStage {
		case 4:
			s.infZero = OwnedInfRational{}
		case 8:
			s.infOne = OwnedInfRational{}
		case 12:
			s.infMinusOne = OwnedInfRational{}
		case 14:
			s.infIntZero = OwnedInfIntRational{}
		case 16:
			s.infIntOne = OwnedInfIntRational{}
		case 18:
			s.infIntMinusOne = OwnedInfIntRational{}
		}
	}
	if s.cleanupStage == 18 {
		for s.cleanupCacheIndex < uint32(len(s.cache)) {
			r := &s.cache[s.cleanupCacheIndex]
			p := &r.num
			if s.cleanupComponent == 1 {
				p = &r.den
			}
			if e := s.releaseOuterLeaf(p, source); e != nil {
				return e
			}
			s.cleanupComponent++
			if s.cleanupComponent == 2 {
				s.cleanupComponent = 0
				s.cleanupCacheIndex++
			}
		}
		s.cache = nil
		s.manager.budget.Release(s.cacheBytes)
		s.cacheBytes = 0
		s.cleanupStage++
	}
	constants := [6]*Int{&s.zero.num, &s.zero.den, &s.one.num, &s.one.den, &s.minusOne.num, &s.minusOne.den}
	for s.cleanupStage < 25 {
		if e := s.releaseOuterLeaf(constants[s.cleanupStage-19], source); e != nil {
			return e
		}
		s.cleanupStage++
	}

	return nil
}
