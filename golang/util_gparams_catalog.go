// Copyright (c) 2011, 2012 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"sync"
)

var ErrGlobalParamsCatalogPending = errors.New("g3: parameter catalog retains unpublished descriptor owner")

type GlobalParamsCatalogConfig struct {
	Budget       *arena.Budget
	Descriptions arena.BytesRegionConfig
	Text         arena.VectorConfig
}

// Catalog owns actual storage for the source's borrowed static descriptions.
// Registries borrow this region and must be closed before the catalog. Every
// created descriptor owner is transferred or retained in pending on failure.
type GlobalParamsCatalog struct {
	mu           sync.Mutex
	self         *GlobalParamsCatalog
	symbols      *SymbolManager
	budget       *arena.Budget
	region       *arena.BytesRegion
	descriptions [16]arena.BytesRegionRef
	text         arena.VectorConfig
	pending      *ParamDescrs
	ownerBytes   uint64
	closed       bool
}

func NewGlobalParamsCatalog(s *SymbolManager, c GlobalParamsCatalogConfig) (*GlobalParamsCatalog, error) {
	if s == nil {
		return nil, ErrGlobalParamsOwner
	}
	if _, e := s.Hash(Symbol{}); e != nil {
		return nil, e
	}
	if c.Descriptions.Descriptors.Budget != nil && c.Descriptions.Descriptors.Budget != c.Budget || c.Text.Descriptors.Budget != nil && c.Text.Descriptors.Budget != c.Budget {
		return nil, arena.ErrConfig
	}
	c.Descriptions.Descriptors.Budget = c.Budget
	c.Text.Descriptors.Budget = c.Budget
	bytes := uint64(reflect.TypeOf(GlobalParamsCatalog{}).Size())
	if e := c.Budget.Reserve(bytes); e != nil {
		return nil, e
	}
	owner := &GlobalParamsCatalog{symbols: s, budget: c.Budget, text: c.Text, ownerBytes: bytes}
	owner.self = owner
	fail := func(e error) (*GlobalParamsCatalog, error) { _ = owner.Close(); return nil, e }
	var e error
	owner.region, e = arena.NewBytesRegion(c.Descriptions)
	if e != nil {
		return fail(e)
	}
	for i, row := range catalogDescriptionLiterals {
		ref, e := owner.region.Allocate(uint32(len(row.text) + 1))
		if e != nil {
			return fail(e)
		}
		for j := range row.text {
			if e = owner.region.Write(ref, uint32(j), row.text[j]); e != nil {
				return fail(e)
			}
		}
		if e = owner.region.Write(ref, uint32(len(row.text)), 0); e != nil {
			return fail(e)
		}
		owner.descriptions[i] = ref
	}
	return owner, nil
}
func (c *GlobalParamsCatalog) open() error {
	if c == nil || c.self != c {
		return ErrGlobalParamsOwner
	}
	if c.closed {
		return arena.ErrClosed
	}
	return nil
}
func (c *GlobalParamsCatalog) newDescriptors() (*ParamDescrs, error) {
	return NewParamDescrs(c.symbols, ParamDescrsConfig{Budget: c.budget, Text: c.text})
}
func (c *GlobalParamsCatalog) global(r GlobalParamsRegistrar, collector func(*ParamDescrs) error) error {
	d, e := c.newDescriptors()
	if e != nil {
		return e
	}
	defer d.Close()
	if e = collector(d); e != nil {
		return e
	}
	return r.RegisterGlobal(d)
}
func (c *GlobalParamsCatalog) module(r GlobalParamsRegistrar, name string, collector func(*ParamDescrs) error) error {
	d, e := c.newDescriptors()
	if e != nil {
		return e
	}
	c.pending = d
	if e = collector(d); e != nil {
		return e
	}
	if e = r.RegisterModule(name, d); e != nil {
		return e
	}
	c.pending = nil
	return nil
}
func (c *GlobalParamsCatalog) Register(r GlobalParamsRegistrar) error {
	if c == nil || c.self != c {
		return ErrGlobalParamsOwner
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.open(); e != nil {
		return e
	}
	if r.owner == nil || !r.owner.registering || r.owner.symbols != c.symbols {
		return ErrGlobalParamsOwner
	}
	if c.pending != nil {
		return ErrGlobalParamsCatalogPending
	}
	return c.dispatch(r)
}
func (c *GlobalParamsCatalog) CleanupPending() error {
	if c == nil || c.self != c {
		return ErrGlobalParamsOwner
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.open(); e != nil {
		return e
	}
	if c.pending != nil {
		if e := c.pending.Close(); e != nil {
			return e
		}
		c.pending = nil
	}
	return nil
}
func (c *GlobalParamsCatalog) Close() error {
	if c == nil || c.self != c {
		return ErrGlobalParamsOwner
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	if c.pending != nil {
		if e := c.pending.Close(); e != nil {
			return e
		}
		c.pending = nil
	}
	if c.region != nil {
		c.region.Close()
		c.region = nil
	}
	c.budget.Release(c.ownerBytes)
	c.ownerBytes = 0
	c.closed = true
	return nil
}
