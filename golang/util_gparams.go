// Source: src/util/gparams.cpp:81-123,633-644; src/util/gparams.h.
// Author: Leonardo de Moura (2012-11-29).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
// Explicit runtime-owned gparams::imp; source catalog dispatch is borrowed.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"sync"
)

var ErrGlobalParamsOwner = errors.New("g3: global parameters owner or lifetime mismatch")
var ErrGlobalParamsRegistrationMissing = errors.New("g3: actual parameter registration dispatcher missing")

type GlobalParamsConfig struct {
	Catalog                                     *GlobalParamsCatalog
	Budget                                      *arena.Budget
	DescriptorPageSize, DescriptorWordsPageSize uint32
	CLongBits                                   uint8
	Register                                    func(GlobalParamsRegistrar) error
}
type GlobalParamCString struct {
	Origin  uint64
	Ref     arena.BytesRegionRef
	Offset  uint32
	Present bool
}
type gparamsRefCell struct{ value ParamsRef }
type gparamsRegionBinding struct {
	origin uint64
	region *arena.BytesRegion
}
type GlobalParams struct {
	catalog                                *GlobalParamsCatalog
	mu                                     sync.Mutex
	self                                   *GlobalParams
	symbols                                *SymbolManager
	context                                *ParamsContext
	budget                                 *arena.Budget
	register                               func(GlobalParamsRegistrar) error
	modulesRegistered, registering, closed bool
	sourceStarted, sourceCompleted         bool
	sourceStage                            uint8
	sourceIndex                            uint32
	sourceCause                            error
	ownerBytes                             uint64
	cLongBits                              uint8
	moduleDescriptors                      gparamsDescriptorDictionary
	moduleDescriptions                     gparamsDescriptionDictionary
	moduleParams                           gparamsRefDictionary
	descriptors                            []*ParamDescrs
	cells                                  []*gparamsRefCell
	bindings                               []gparamsRegionBinding
	globalDescriptors                      *ParamDescrs
	globalParams                           ParamsRef
	workspace                              []byte
}

// Registrar is a cold by-value capability used only inside the registration
// critical section. Its methods use private unlocked owner operations.
type GlobalParamsRegistrar struct{ owner *GlobalParams }

func (r GlobalParamsRegistrar) RegisterModule(name string, d *ParamDescrs) error {
	if r.owner == nil || !r.owner.registering {
		return ErrGlobalParamsOwner
	}
	return r.owner.registerModule(name, d)
}
func (r GlobalParamsRegistrar) RegisterGlobal(d *ParamDescrs) error {
	if r.owner == nil || !r.owner.registering {
		return ErrGlobalParamsOwner
	}
	return r.owner.globalDescriptors.Copy(d)
}
func NewGlobalParams(s *SymbolManager, p *ParamsContext, c GlobalParamsConfig) (*GlobalParams, error) {
	if s == nil || p == nil || p.symbols != s {
		return nil, ErrGlobalParamsOwner
	}
	if e := p.open(); e != nil {
		return nil, e
	}
	if c.Catalog != nil {
		if c.Register != nil {
			return nil, arena.ErrConfig
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
	}
	longBits := c.CLongBits
	if longBits == 0 {
		longBits = 64
	}
	if longBits != 32 && longBits != 64 {
		return nil, arena.ErrConfig
	}

	n := uint64(reflect.TypeOf(GlobalParams{}).Size())
	if e := c.Budget.Reserve(n); e != nil {
		return nil, e
	}
	g := &GlobalParams{symbols: s, context: p, budget: c.Budget, register: c.Register, ownerBytes: n, catalog: c.Catalog}
	g.self = g
	g.cLongBits = longBits
	fail := func(e error) (*GlobalParams, error) { _ = g.Close(); return nil, e }
	if e := g.moduleDescriptors.allocate(c.Budget, 8); e != nil {
		return fail(e)
	}
	if e := g.moduleDescriptions.allocate(c.Budget, 8); e != nil {
		return fail(e)
	}
	d, e := NewParamDescrs(s, ParamDescrsConfig{Budget: c.Budget, Text: arena.VectorConfig{PageSize: c.DescriptorWordsPageSize, Descriptors: arena.Config{PageSize: c.DescriptorPageSize, Budget: c.Budget}}})
	if e != nil {
		return fail(e)
	}
	g.globalDescriptors = d
	if e = g.moduleParams.allocate(c.Budget, 8); e != nil {
		return fail(e)
	}
	g.globalParams, e = p.NewValue()
	if e != nil {
		return fail(e)
	}
	return g, nil
}
func (g *GlobalParams) open() error {
	if g == nil || g.self != g {
		return ErrGlobalParamsOwner
	}
	if g.closed || g.sourceStarted {
		return arena.ErrClosed
	}
	return g.context.open()
}
func (g *GlobalParams) ensureRegistered() error {
	if e := g.open(); e != nil {
		return e
	}
	if g.modulesRegistered {
		return nil
	}
	g.modulesRegistered = true
	if g.catalog != nil {
		g.registering = true
		defer func() { g.registering = false }()
		return g.catalog.Register(GlobalParamsRegistrar{g})
	}
	if g.register == nil {
		return ErrGlobalParamsRegistrationMissing
	}
	g.registering = true
	defer func() { g.registering = false }()
	return g.register(GlobalParamsRegistrar{g})
}
func (g *GlobalParams) moduleDescriptor(k Symbol) (*ParamDescrs, error) {
	if e := g.ensureRegistered(); e != nil {
		return nil, e
	}
	h, e := g.symbols.Hash(k)
	if e != nil {
		return nil, e
	}
	i := g.moduleDescriptors.find(k, h)
	if i < 0 {
		return nil, ErrGlobalParamsOwner
	}
	n := g.moduleDescriptors.entries[i].descriptor
	if n == 0 || int(n) > len(g.descriptors) || g.descriptors[n-1] == nil {
		return nil, ErrGlobalParamsOwner
	}
	return g.descriptors[n-1], nil
}
func (g *GlobalParams) growDescriptors() error {
	if len(g.descriptors) < cap(g.descriptors) {
		return nil
	}
	n, e := pdGrowth(cap(g.descriptors))
	if e != nil {
		return e
	}
	stride := uint64(reflect.TypeOf((*ParamDescrs)(nil)).Size())
	if uint64(n) > uint64(^uint(0)>>1)/stride {
		return arena.ErrBounds
	}
	if e = g.budget.Reserve(uint64(n) * stride); e != nil {
		return e
	}
	next := make([]*ParamDescrs, len(g.descriptors), n)
	copy(next, g.descriptors)
	g.budget.Release(uint64(cap(g.descriptors)) * stride)
	g.descriptors = next
	return nil
}
func (g *GlobalParams) registerModule(name string, d *ParamDescrs) error {
	if e := g.open(); e != nil {
		return e
	}
	if d == nil || d.manager != g.symbols {
		return ErrGlobalParamsOwner
	}
	if e := d.open(); e != nil {
		return e
	}
	k, e := g.symbols.Intern(name)
	if e != nil {
		return e
	}
	h, e := g.symbols.Hash(k)
	if e != nil {
		return e
	}
	i := g.moduleDescriptors.find(k, h)
	if i >= 0 {
		old := g.descriptors[g.moduleDescriptors.entries[i].descriptor-1]
		if old == d {
			return ErrGlobalParamsOwner
		}
		if e = old.Copy(d); e != nil {
			return e
		}
		return d.Close()
	}
	if uint64(len(g.descriptors)) >= uint64(^uint32(0)) {
		return arena.ErrBounds
	}
	if e = g.growDescriptors(); e != nil {
		return e
	}
	slot := uint32(len(g.descriptors) + 1)
	if e = g.moduleDescriptors.insert(g.budget, gparamsDescriptorEntry{key: k, hash: h, descriptor: slot}); e != nil {
		return e
	}
	g.descriptors = append(g.descriptors, d)
	return nil
}
func (g *GlobalParams) RegisterModule(name string, d *ParamDescrs) error {
	if g == nil {
		return ErrGlobalParamsOwner
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.registerModule(name, d)
}
func (g *GlobalParams) RegisterGlobal(d *ParamDescrs) error {
	if g == nil {
		return ErrGlobalParamsOwner
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if e := g.open(); e != nil {
		return e
	}
	return g.globalDescriptors.Copy(d)
}
func (g *GlobalParams) growCells() error {
	if len(g.cells) < cap(g.cells) {
		return nil
	}
	n, e := pdGrowth(cap(g.cells))
	if e != nil {
		return e
	}
	stride := uint64(reflect.TypeOf((*gparamsRefCell)(nil)).Size())
	if uint64(n) > uint64(^uint(0)>>1)/stride {
		return arena.ErrBounds
	}
	if e = g.budget.Reserve(uint64(n) * stride); e != nil {
		return e
	}
	next := make([]*gparamsRefCell, len(g.cells), n)
	copy(next, g.cells)
	g.budget.Release(uint64(cap(g.cells)) * stride)
	g.cells = next
	return nil
}
func (g *GlobalParams) getParams(k Symbol) (*ParamsRef, error) {
	if e := g.open(); e != nil {
		return nil, e
	}
	h, e := g.symbols.Hash(k)
	if e != nil {
		return nil, e
	}
	if k == (Symbol{}) {
		return &g.globalParams, nil
	}
	if i := g.moduleParams.find(k, h); i >= 0 {
		n := g.moduleParams.entries[i].cell
		if n == 0 || int(n) > len(g.cells) || g.cells[n-1] == nil {
			return nil, ErrGlobalParamsOwner
		}
		return &g.cells[n-1].value, nil
	}
	bytes := uint64(reflect.TypeOf(gparamsRefCell{}).Size())
	if e = g.budget.Reserve(bytes); e != nil {
		return nil, e
	}
	v, e := g.context.NewValue()
	if e != nil {
		g.budget.Release(bytes)
		return nil, e
	}
	cell := &gparamsRefCell{v}
	// Scalar directory slots are private metadata, not externally issued handles.
	// Reuse released roots so repeated source reset/new-cell cycles plateau.
	slot := len(g.cells)
	for i, old := range g.cells {
		if old == nil {
			slot = i
			break
		}
	}
	if uint64(slot) >= uint64(^uint32(0)) {
		g.budget.Release(bytes)
		return nil, arena.ErrBounds
	}
	if slot == len(g.cells) {
		if e = g.growCells(); e != nil {
			g.budget.Release(bytes)
			return nil, e
		}
	}
	n := uint32(slot + 1)
	if e = g.moduleParams.insert(g.budget, gparamsRefEntry{key: k, hash: h, cell: n}); e != nil {
		g.budget.Release(bytes)
		return nil, e
	}
	if slot == len(g.cells) {
		g.cells = append(g.cells, cell)
	} else {
		g.cells[slot] = cell
	}
	return &cell.value, nil
}
func (g *GlobalParams) getModule(k Symbol) (ParamsRef, error) {
	if e := g.open(); e != nil {
		return ParamsRef{}, e
	}
	h, e := g.symbols.Hash(k)
	if e != nil {
		return ParamsRef{}, e
	}
	v, e := g.context.NewValue()
	if e != nil {
		return ParamsRef{}, e
	}
	if i := g.moduleParams.find(k, h); i >= 0 {
		n := g.moduleParams.entries[i].cell
		if n == 0 || int(n) > len(g.cells) || g.cells[n-1] == nil {
			return ParamsRef{}, ErrGlobalParamsOwner
		}
		if e = g.context.Assign(&v, &g.cells[n-1].value); e != nil {
			return ParamsRef{}, e
		}
	}
	return v, nil
}
func (g *GlobalParams) GetModule(k Symbol) (ParamsRef, error) {
	if g == nil {
		return ParamsRef{}, ErrGlobalParamsOwner
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.getModule(k)
}
func (g *GlobalParams) GetModuleName(name string) (ParamsRef, error) {
	if g == nil {
		return ParamsRef{}, ErrGlobalParamsOwner
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if e := g.open(); e != nil {
		return ParamsRef{}, e
	}
	k, e := g.symbols.Intern(name)
	if e != nil {
		return ParamsRef{}, e
	}
	return g.getModule(k)
}
func (g *GlobalParams) reset() error {
	if g.globalParams.owner != 0 {
		if e := g.context.Reset(&g.globalParams); e != nil {
			return e
		}
	}
	for _, x := range g.moduleParams.entries {
		if x.state != 2 {
			continue
		}
		i := x.cell
		if i == 0 || int(i) > len(g.cells) || g.cells[i-1] == nil {
			return ErrGlobalParamsOwner
		}
		cell := g.cells[i-1]
		if e := g.context.Del(&cell.value); e != nil {
			return e
		}
		g.budget.Release(uint64(reflect.TypeOf(*cell).Size()))
		g.cells[i-1] = nil
	}
	return g.moduleParams.reset(g.budget)
}
func (g *GlobalParams) Reset() error {
	if g == nil {
		return ErrGlobalParamsOwner
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if e := g.open(); e != nil {
		return e
	}
	return g.reset()
}
func (g *GlobalParams) Close() error {
	if g != nil && g.self == g && g.sourceStarted {
		return g.CleanupStopped()
	}
	return g.closeLegacy()
}
func (g *GlobalParams) closeLegacy() error {
	if g == nil || g.self != g {
		return ErrGlobalParamsOwner
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return nil
	}
	if e := g.reset(); e != nil {
		return e
	}
	for _, x := range g.moduleDescriptors.entries {
		if x.state == 2 {
			d := g.descriptors[x.descriptor-1]
			if d != nil {
				if e := d.Close(); e != nil {
					return e
				}
				g.descriptors[x.descriptor-1] = nil
			}
		}
	}
	if g.globalDescriptors != nil {
		if e := g.globalDescriptors.Close(); e != nil {
			return e
		}
	}
	// Global null Ref is a borrowed owner-local scalar after reset; source
	// member destruction decrements it if still initialized, without mutation.
	if g.globalParams.owner != 0 {
		if e := g.context.Del(&g.globalParams); e != nil {
			return e
		}
	}
	g.moduleParams.close(g.budget)
	g.moduleDescriptions.close(g.budget)
	g.moduleDescriptors.close(g.budget)
	g.budget.Release(uint64(cap(g.cells))*uint64(reflect.TypeOf((*gparamsRefCell)(nil)).Size()) + uint64(cap(g.descriptors))*uint64(reflect.TypeOf((*ParamDescrs)(nil)).Size()) + uint64(cap(g.bindings))*uint64(reflect.TypeOf(gparamsRegionBinding{}).Size()) + uint64(cap(g.workspace)) + g.ownerBytes)
	g.cells = nil
	g.descriptors = nil
	g.bindings = nil
	g.workspace = nil
	g.ownerBytes = 0
	g.closed = true
	return nil
}
