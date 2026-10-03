// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
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
	self    *UPolynomialFactorizationService
	manager *UPolynomialManager
	config  UPolynomialFactorizationConfig
	header  uint64
	closed  bool
}
type UPolynomialFactorizationStats struct{ OwnerBytes uint64 }

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
func (s *UPolynomialFactorizationService) check() error {
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
func (s *UPolynomialFactorizationService) Close() error {
	if s == nil || s.self != s || s.manager == nil {
		return ErrUPolynomialOwner
	}
	if s.closed {
		return arena.ErrClosed
	}
	s.manager.config.Integer.budget.Release(s.header)
	s.closed = true
	return nil
}
func (s *UPolynomialFactorizationService) Stats() UPolynomialFactorizationStats {
	if s == nil || s.self != s || s.closed {
		return UPolynomialFactorizationStats{}
	}
	return UPolynomialFactorizationStats{OwnerBytes: s.header}
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
