// Source: src/math/polynomial/upolynomial_factorization.cpp:1030-1050,1153-1175,1238-1292;
// src/math/polynomial/upolynomial_factorization.h:85-89.
// Actual explicit Go service binding and reflected enclosing cold storage.
// Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

type UPolynomialFactorParams struct{ MaxPrime, PrimeTrials, MaxSearchSize uint32 }

func DefaultUPolynomialFactorParams() UPolynomialFactorParams {
	return UPolynomialFactorParams{MaxPrime: 4294967295, PrimeTrials: 1, MaxSearchSize: 0xffffffff}
}

type UPolynomialFactorizationConfig struct {
	Primes                *PrimeGenerator
	FactorizationDebugTag bool
}
type UPolynomialFactorizationService struct {
	squareFreeHistory           *upolyFactorSquareFreeScope
	squareFreeDiagnosticHistory *upolySquareFreeDiagnosticHistory
	squareFreeInvocation        uint64
	diagnostic, preparation     upolyFactorPreparationScope
	self                        *UPolynomialFactorizationService
	manager                     *UPolynomialManager
	config                      UPolynomialFactorizationConfig
	scope                       *upolyFactorSquareFreeScope
	henselPairs                 *upolyHenselPairScope
	header                      uint64
	closed                      bool
}
type UPolynomialFactorizationStats struct {
	OwnerBytes, RetainedOwnerBytes uint64
	CachedOwnerBytes               uint64
	CachedScopes                   uint32
	RetainedScopes                 uint32
	MignottePending                bool
	PrimeTrialPending              bool
	RecombinationPending           bool
	HenselPending                  bool
}

func NewUPolynomialFactorizationService(m *UPolynomialManager, c UPolynomialFactorizationConfig) (*UPolynomialFactorizationService, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	if c.Primes == nil {
		return nil, ErrUPolynomialOwner
	}
	if e := c.Primes.check(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(UPolynomialFactorizationService{}).Size())
	if e := m.config.Integer.budget.Reserve(n); e != nil {
		return nil, e
	}
	s := &UPolynomialFactorizationService{manager: m, config: c, header: n}
	s.self = s
	return s, nil
}
func (s *UPolynomialFactorizationService) checkOwner() error {
	if s == nil || s.self != s || s.manager == nil {
		return ErrUPolynomialOwner
	}
	if s.closed {
		return arena.ErrClosed
	}
	if e := s.manager.check(); e != nil {
		return e
	}
	return s.config.Primes.check()
}
func (s *UPolynomialFactorizationService) check() error {
	if e := s.checkOwner(); e != nil {
		return e
	}
	if s.scope != nil {
		return ErrUPolynomialCleanupPending
	}
	return nil
}

func (s *UPolynomialFactorizationService) Close() error {
	if s == nil || s.self != s || s.manager == nil {
		return ErrUPolynomialOwner
	}
	if s.closed {
		return arena.ErrClosed
	}
	if e := s.CleanupRetained(); e != nil {
		return e
	}
	for h := s.henselPairs; h != nil; h = h.next {
		s.manager.config.Integer.budget.Release(h.header)
	}
	s.henselPairs = nil
	for c := s.squareFreeHistory; c != nil; c = c.historyNext {
		s.manager.config.Integer.budget.Release(c.header)
		c.header = 0
	}
	s.squareFreeHistory = nil
	for h := s.squareFreeDiagnosticHistory; h != nil; h = h.next {
		s.manager.config.Integer.budget.Release(h.header)
		h.header = 0
	}
	s.squareFreeDiagnosticHistory = nil
	s.manager.config.Integer.budget.Release(s.header)
	s.closed = true
	return nil
}
func (s *UPolynomialFactorizationService) Stats() UPolynomialFactorizationStats {
	if s == nil || s.self != s || s.closed {
		return UPolynomialFactorizationStats{}
	}
	r := UPolynomialFactorizationStats{OwnerBytes: s.header}
	for c := s.squareFreeHistory; c != nil; c = c.historyNext {
		r.CachedOwnerBytes += c.header
		r.CachedScopes++
	}
	for h := s.squareFreeDiagnosticHistory; h != nil; h = h.next {
		r.CachedOwnerBytes += h.header
		r.CachedScopes++
	}
	if s.scope != nil {
		r.RetainedScopes = 1
		r.RetainedOwnerBytes = s.scope.header
		r.MignottePending = s.scope.mignotte.active
		r.PrimeTrialPending = s.scope.primeTrial.active
		r.RecombinationPending = s.scope.recombinationStep.active
		r.HenselPending = s.scope.hensel.active
	}
	for h := s.henselPairs; h != nil; h = h.next {
		r.CachedOwnerBytes += h.header
		r.CachedScopes++
		if h.active {
			r.RetainedOwnerBytes += h.header
			r.RetainedScopes++
			r.HenselPending = true
		}
	}
	return r
}
func (s *UPolynomialFactorizationService) PrimeSquareFreeFactors(m *UPolynomialManager, f UPolynomialView, out *UPolynomialFactors) error {
	if e := s.check(); e != nil {
		return e
	}
	return m.SquareFreeFactorZP(f, out)
}
func (s *UPolynomialFactorizationService) FactorPrime(m *UPolynomialManager, f UPolynomialView, out *UPolynomialFactors) (bool, error) {
	if e := s.check(); e != nil {
		return false, e
	}
	return m.FactorZP(f, out)
}
func (s *UPolynomialFactorizationService) FactorPrimeSquareFree(m *UPolynomialManager, f UPolynomialView, out *UPolynomialFactors) (bool, error) {
	if e := s.check(); e != nil {
		return false, e
	}
	return m.FactorZPSquareFree(f, out)
}
func (s *UPolynomialFactorizationService) BerlekampPrimeSquareFree(m *UPolynomialManager, f UPolynomialView, out *UPolynomialFactors, randomized bool) (bool, error) {
	if e := s.check(); e != nil {
		return false, e
	}
	return m.FactorZPSquareFreeBerlekamp(f, out, randomized)
}
