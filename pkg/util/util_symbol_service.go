// Source: src/util/symbol.cpp:70-83; internal_symbol_table members:33-35.
package util

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrSymbolServiceOwner = errors.New("g3: invalid symbol service owner")
var ErrSymbolServiceUninitialized = errors.New("g3: source symbol table is null")

// SymbolService holds the source symbol-table pointer in an explicit single-
// lane owner. The actual manager constructs region then table, and destroys
// table then region. The cold service header is charged once; no synthetic
// reservation represents initialize_symbols or finalize_symbols.
// Process singleton and global Runtime accounting remain separate bindings.
type SymbolService struct {
	self       *SymbolService
	config     SymbolConfig
	manager    *SymbolManager
	ownerBytes uint64
	closed     bool
}

func NewSymbolService(c SymbolConfig) (*SymbolService, error) {
	n := uint64(reflect.TypeOf(SymbolService{}).Size())
	if e := c.Budget.Reserve(n); e != nil {
		return nil, e
	}
	s := &SymbolService{config: c, ownerBytes: n}
	s.self = s
	return s, nil
}
func (s *SymbolService) open() error {
	if s == nil || s.self != s {
		return ErrSymbolServiceOwner
	}
	if s.closed {
		return arena.ErrClosed
	}
	return nil
}
func (s *SymbolService) InitializeSource() error {
	if e := s.open(); e != nil {
		return e
	}
	if s.manager != nil {
		return nil
	}
	m, e := NewSymbolManager(s.config)
	if e != nil {
		return e
	}
	s.manager = m
	return nil
}
func (s *SymbolService) FinalizeSource() error {
	if e := s.open(); e != nil {
		return e
	}
	if s.manager != nil {
		if e := s.manager.Close(); e != nil {
			return e
		}
		s.manager = nil
	}
	return nil
}

// Manager borrows the currently published table. Finalization invalidates all
// symbols of that owner; a later initialization acquires a fresh owner identity.
func (s *SymbolService) Manager() (*SymbolManager, error) {
	if e := s.open(); e != nil {
		return nil, e
	}
	if s.manager == nil {
		return nil, ErrSymbolServiceUninitialized
	}
	return s.manager, nil
}
func (s *SymbolService) Close() error {
	if s == nil || s.self != s {
		return ErrSymbolServiceOwner
	}
	if s.closed {
		return nil
	}
	if e := s.FinalizeSource(); e != nil {
		return e
	}
	s.config.Budget.Release(s.ownerBytes)
	s.ownerBytes = 0
	s.closed = true
	return nil
}
