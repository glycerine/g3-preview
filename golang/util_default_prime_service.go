// Source: src/util/prime_generator.cpp:23-27,99-130; src/util/prime_generator.h:29-52.
// Licensed under the MIT license. See LICENSE.txt.
// Source author: Leonardo de Moura (leonardo), 2011-12-23.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"reflect"
)

var ErrDefaultPrimeServiceOwner = errors.New("g3: invalid default prime service owner")

// The actual generator is eagerly constructed with this binding, before any
// memory initialization call. Finalize clears its vector and keeps its identity.
// The enclosing header is charged cold Go binding metadata, not a source malloc
// event. The existing generator's shared-Runtime cache storage and OpenMP/global
// process wiring remain separate obligations.
type DefaultPrimeService struct {
	self      *DefaultPrimeService
	generator *mpnum.PrimeGenerator
	budget    *arena.Budget
	header    uint64
	closed    bool
}
type DefaultPrimeServiceStats struct {
	OwnerBytes, TotalBytes uint64
	Generator              mpnum.PrimeGeneratorStats
}

func NewDefaultPrimeService(b *arena.Budget) (*DefaultPrimeService, error) {
	n := uint64(reflect.TypeOf(DefaultPrimeService{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	g, e := mpnum.NewPrimeGenerator(b)
	if e != nil {
		b.Release(n)
		return nil, e
	}
	s := &DefaultPrimeService{generator: g, budget: b, header: n}
	s.self = s
	return s, nil
}
func (s *DefaultPrimeService) open() error {
	if s == nil || s.self != s {
		return ErrDefaultPrimeServiceOwner
	}
	if s.closed {
		return arena.ErrClosed
	}
	return nil
}
func (s *DefaultPrimeService) Finalize() error {
	if e := s.open(); e != nil {
		return e
	}
	return s.generator.Finalize()
}
func (s *DefaultPrimeService) Stats() DefaultPrimeServiceStats {
	if s.open() != nil {
		return DefaultPrimeServiceStats{}
	}
	g := s.generator.Stats()
	return DefaultPrimeServiceStats{s.header, s.header + g.TotalBytes, g}
}
func (s *DefaultPrimeService) Close() error {
	if s == nil || s.self != s {
		return ErrDefaultPrimeServiceOwner
	}
	if s.closed {
		return nil
	}
	if e := s.generator.Close(); e != nil {
		return e
	}
	s.budget.Release(s.header)
	s.closed = true
	return nil
}

// This is a nonowning, freely copyable source iterator value. Local construction
// does not read the unused default service or the supplied generator contents.
type PrimeServiceIterator struct {
	service   *DefaultPrimeService
	generator *mpnum.PrimeGenerator
	index     uint32
	global    bool
}

func NewPrimeServiceIterator(s *DefaultPrimeService, g *mpnum.PrimeGenerator) (PrimeServiceIterator, error) {
	if g != nil {
		return PrimeServiceIterator{generator: g}, nil
	}
	if e := s.open(); e != nil {
		return PrimeServiceIterator{}, e
	}
	return PrimeServiceIterator{service: s, generator: s.generator, global: true}, nil
}
func (it *PrimeServiceIterator) Next() (uint64, error) {
	if it == nil {
		return 0, arena.ErrConfig
	}
	index := it.index
	it.index++
	if !it.global {
		return it.generator.At(index)
	}
	if e := it.service.open(); e != nil {
		return 0, e
	}
	return it.generator.At(index)
}
