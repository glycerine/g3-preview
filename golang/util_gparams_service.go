// Source: src/util/gparams.cpp:633-644; source imp:81-105.
// Author: Leonardo de Moura (2012-11-29).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

var ErrGlobalParamsUninitialized = errors.New("g3: source global parameter pointer is null")

// An explicit single-lane source global pointer. The actual catalog, Symbol
// and Params owners are borrowed and outlive every named current/old/failed
// node. Process singleton, TRACE stream, native ABI and global Runtime table
// accounting remain separate bindings. The header and all nodes are charged.
type GlobalParamsService struct {
	self                        *GlobalParamsService
	symbols                     *SymbolManager
	context                     *ParamsContext
	config                      GlobalParamsConfig
	current, superseded, failed *GlobalParams
	ownerBytes                  uint64
	closed                      bool
}

func NewGlobalParamsService(s *SymbolManager, p *ParamsContext, c GlobalParamsConfig) (*GlobalParamsService, error) {
	if s == nil || p == nil || p.symbols != s {
		return nil, ErrGlobalParamsOwner
	}
	if e := p.open(); e != nil {
		return nil, e
	}
	if c.Register != nil {
		return nil, arena.ErrConfig
	} // source uses the concrete catalog
	if c.Catalog == nil {
		return nil, ErrGlobalParamsRegistrationMissing
	}
	c.Catalog.mu.Lock()
	e := c.Catalog.open()
	same := c.Catalog.symbols == s
	c.Catalog.mu.Unlock()
	if e != nil {
		return nil, e
	}
	if !same {
		return nil, ErrGlobalParamsOwner
	}
	n := uint64(reflect.TypeOf(GlobalParamsService{}).Size())
	if e := c.Budget.Reserve(n); e != nil {
		return nil, e
	}
	result := &GlobalParamsService{symbols: s, context: p, config: c, ownerBytes: n}
	result.self = result
	return result, nil
}
func (s *GlobalParamsService) open() error {
	if s == nil || s.self != s {
		return ErrGlobalParamsOwner
	}
	if s.closed {
		return arena.ErrClosed
	}
	return nil
}
func (s *GlobalParamsService) InitializeSource() error {
	if e := s.open(); e != nil {
		return e
	}
	// Source init is unconditional; a previous live pointer does not skip alloc.
	g, e := newSourceGlobalParams(s.symbols, s.context, s.config)
	if e != nil {
		if g != nil {
			g.retainedNext = s.failed
			s.failed = g
		}
		return e
	}
	old := s.current
	s.current = g // actual source g_imp=alloc(imp) publication
	if old != nil {
		old.retainedNext = s.superseded
		s.superseded = old
	}
	return nil
}
func (s *GlobalParamsService) Manager() (*GlobalParams, error) {
	if e := s.open(); e != nil {
		return nil, e
	}
	if s.current == nil {
		return nil, ErrGlobalParamsUninitialized
	}
	return s.current, nil
}
func (s *GlobalParamsService) FinalizeSource() error {
	if e := s.open(); e != nil {
		return e
	}
	if s.current != nil {
		if e := s.current.SourceFinalize(); e != nil {
			return e
		}
		s.current = nil // only after completed actual source dealloc
	}
	return nil
}
func (s *GlobalParamsService) Close() error {
	if s == nil || s.self != s {
		return ErrGlobalParamsOwner
	}
	if s.closed {
		return nil
	}
	// Owning close can acknowledge only the actual stopped child owned here.
	if s.current != nil {
		if e := s.current.SourceFinalize(); e != nil {
			if cleanup := s.current.CleanupStopped(); cleanup != nil {
				return errors.Join(e, cleanup)
			}
		}
		s.current = nil
	}
	for s.superseded != nil {
		node := s.superseded
		if e := node.SourceFinalize(); e != nil {
			if cleanup := node.CleanupStopped(); cleanup != nil {
				return errors.Join(e, cleanup)
			}
		}
		s.superseded = node.retainedNext
		node.retainedNext = nil
	}
	for s.failed != nil {
		node := s.failed
		if e := node.abortConstructor(); e != nil {
			return e
		}
		s.failed = node.retainedNext
		node.retainedNext = nil
	}
	s.config.Budget.Release(s.ownerBytes)
	s.ownerBytes = 0
	s.closed = true
	return nil
}
