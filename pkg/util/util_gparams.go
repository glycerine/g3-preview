// Source: src/util/gparams.cpp:81-123,633-644; src/util/gparams.h.
// Author: Leonardo de Moura (2012-11-29).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
// Explicit runtime-owned gparams::imp; source catalog dispatch is borrowed.
package util

import (
	"errors"
	"reflect"
	"sync"

	"github.com/glycerine/g3/pkg/arena"
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
type GparamsRefCell struct{ value ParamsRef }
type GparamsRegionBinding struct {
	origin uint64
	region *arena.BytesRegion
}
type GlobalParams struct {
	catalog                                 *GlobalParamsCatalog
	mu                                      sync.Mutex
	self                                    *GlobalParams
	SymbolsField                            *SymbolManager
	ContextField                            *ParamsContext
	Budget                                  *arena.Budget
	Register                                func(GlobalParamsRegistrar) error
	modulesRegistered, registering, closed  bool
	sourceStarted, sourceCompleted          bool
	sourceStage                             uint8
	sourceIndex                             uint32
	sourceCause                             error
	sourceDecrement                         ParamsDecrementReceipt
	sourceReset                             ParamsResetReceipt
	OwnerBytes                              uint64
	constructorReady                        bool
	constructorCompleted, constructorUnwind uint8
	constructorCause                        error
	constructorPendingDescriptor            *ParamDescrs
	RetainedNext                            *GlobalParams
	cLongBits                               uint8
	ModuleDescriptors                       gparamsDescriptorDictionary
	ModuleDescriptions                      gparamsDescriptionDictionary
	ModuleParams                            gparamsRefDictionary
	Descriptors                             []*ParamDescrs
	Cells                                   []*GparamsRefCell
	Bindings                                []GparamsRegionBinding
	GlobalDescriptors                       *ParamDescrs
	globalParams                            ParamsRef
	Workspace                               []byte
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
	return r.owner.GlobalDescriptors.Copy(d)
}
func newSourceGlobalParams(s *SymbolManager, p *ParamsContext, c GlobalParamsConfig) (*GlobalParams, error) {
	if s == nil || p == nil || p.SymbolsField != s {
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
	g := &GlobalParams{SymbolsField: s, ContextField: p, Budget: c.Budget, Register: c.Register, OwnerBytes: n, catalog: c.Catalog}
	g.self = g
	g.cLongBits = longBits
	fail := func(e error) (*GlobalParams, error) {
		g.constructorCause = e
		g.constructorUnwind = g.constructorCompleted
		if cleanup := g.unwindConstructor(); cleanup != nil {
			return g, errors.Join(e, cleanup)
		}
		return g, e
	}
	if e := g.ModuleDescriptors.allocate(c.Budget, 8); e != nil {
		return fail(e)
	}
	g.constructorCompleted = 1
	if e := g.ModuleDescriptions.allocate(c.Budget, 8); e != nil {
		return fail(e)
	}
	g.constructorCompleted = 2
	d, e := newSourceParamDescrs(s, ParamDescrsConfig{Budget: c.Budget, Text: arena.VectorConfig{PageSize: c.DescriptorWordsPageSize, Descriptors: arena.Config{PageSize: c.DescriptorPageSize, Budget: c.Budget}}})
	if e != nil {
		g.constructorPendingDescriptor = d
		return fail(e)
	}
	g.GlobalDescriptors = d
	g.constructorCompleted = 3
	if e = g.ModuleParams.allocate(c.Budget, 8); e != nil {
		return fail(e)
	}
	g.constructorCompleted = 4
	g.globalParams, e = p.NewValue()
	if e != nil {
		return fail(e)
	}
	g.constructorCompleted = 5
	g.constructorReady = true
	return g, nil
}
func (g *GlobalParams) open() error {
	if g == nil || g.self != g {
		return ErrGlobalParamsOwner
	}
	if !g.constructorReady {
		return ErrGlobalParamsUnconstructed
	}
	if g.closed || g.sourceStarted {
		return arena.ErrClosed
	}
	return g.ContextField.open()
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
	if g.Register == nil {
		return ErrGlobalParamsRegistrationMissing
	}
	g.registering = true
	defer func() { g.registering = false }()
	return g.Register(GlobalParamsRegistrar{g})
}
func (g *GlobalParams) moduleDescriptor(k Symbol) (*ParamDescrs, error) {
	if e := g.ensureRegistered(); e != nil {
		return nil, e
	}
	h, e := g.SymbolsField.Hash(k)
	if e != nil {
		return nil, e
	}
	i := g.ModuleDescriptors.find(k, h)
	if i < 0 {
		return nil, ErrGlobalParamsOwner
	}
	n := g.ModuleDescriptors.Entries[i].descriptor
	if n == 0 || int(n) > len(g.Descriptors) || g.Descriptors[n-1] == nil {
		return nil, ErrGlobalParamsOwner
	}
	return g.Descriptors[n-1], nil
}
func (g *GlobalParams) growDescriptors() error {
	if len(g.Descriptors) < cap(g.Descriptors) {
		return nil
	}
	n, e := pdGrowth(cap(g.Descriptors))
	if e != nil {
		return e
	}
	stride := uint64(reflect.TypeOf((*ParamDescrs)(nil)).Size())
	if uint64(n) > uint64(^uint(0)>>1)/stride {
		return arena.ErrBounds
	}
	if e = g.Budget.Reserve(uint64(n) * stride); e != nil {
		return e
	}
	next := make([]*ParamDescrs, len(g.Descriptors), n)
	copy(next, g.Descriptors)
	g.Budget.Release(uint64(cap(g.Descriptors)) * stride)
	g.Descriptors = next
	return nil
}
func (g *GlobalParams) registerModule(name string, d *ParamDescrs) error {
	if e := g.open(); e != nil {
		return e
	}
	if d == nil || d.manager != g.SymbolsField {
		return ErrGlobalParamsOwner
	}
	if e := d.open(); e != nil {
		return e
	}
	k, e := g.SymbolsField.Intern(name)
	if e != nil {
		return e
	}
	h, e := g.SymbolsField.Hash(k)
	if e != nil {
		return e
	}
	i := g.ModuleDescriptors.find(k, h)
	if i >= 0 {
		old := g.Descriptors[g.ModuleDescriptors.Entries[i].descriptor-1]
		if old == d {
			return ErrGlobalParamsOwner
		}
		if e = old.Copy(d); e != nil {
			return e
		}
		return d.Close()
	}
	if uint64(len(g.Descriptors)) >= uint64(^uint32(0)) {
		return arena.ErrBounds
	}
	if e = g.growDescriptors(); e != nil {
		return e
	}
	slot := uint32(len(g.Descriptors) + 1)
	if e = g.ModuleDescriptors.insert(g.Budget, GparamsDescriptorEntry{key: k, hash: h, descriptor: slot}); e != nil {
		return e
	}
	g.Descriptors = append(g.Descriptors, d)
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
	return g.GlobalDescriptors.Copy(d)
}
func (g *GlobalParams) growCells() error {
	if len(g.Cells) < cap(g.Cells) {
		return nil
	}
	n, e := pdGrowth(cap(g.Cells))
	if e != nil {
		return e
	}
	stride := uint64(reflect.TypeOf((*GparamsRefCell)(nil)).Size())
	if uint64(n) > uint64(^uint(0)>>1)/stride {
		return arena.ErrBounds
	}
	if e = g.Budget.Reserve(uint64(n) * stride); e != nil {
		return e
	}
	next := make([]*GparamsRefCell, len(g.Cells), n)
	copy(next, g.Cells)
	g.Budget.Release(uint64(cap(g.Cells)) * stride)
	g.Cells = next
	return nil
}
func (g *GlobalParams) GetParams(k Symbol) (*ParamsRef, error) {
	if e := g.open(); e != nil {
		return nil, e
	}
	h, e := g.SymbolsField.Hash(k)
	if e != nil {
		return nil, e
	}
	if k == (Symbol{}) {
		return &g.globalParams, nil
	}
	if i := g.ModuleParams.find(k, h); i >= 0 {
		n := g.ModuleParams.Entries[i].cell
		if n == 0 || int(n) > len(g.Cells) || g.Cells[n-1] == nil {
			return nil, ErrGlobalParamsOwner
		}
		return &g.Cells[n-1].value, nil
	}
	bytes := uint64(reflect.TypeOf(GparamsRefCell{}).Size())
	if e = g.Budget.Reserve(bytes); e != nil {
		return nil, e
	}
	v, e := g.ContextField.NewValue()
	if e != nil {
		g.Budget.Release(bytes)
		return nil, e
	}
	cell := &GparamsRefCell{v}
	// Scalar directory slots are private metadata, not externally issued handles.
	// Reuse released roots so repeated source reset/new-cell cycles plateau.
	slot := len(g.Cells)
	for i, old := range g.Cells {
		if old == nil {
			slot = i
			break
		}
	}
	if uint64(slot) >= uint64(^uint32(0)) {
		g.Budget.Release(bytes)
		return nil, arena.ErrBounds
	}
	if slot == len(g.Cells) {
		if e = g.growCells(); e != nil {
			g.Budget.Release(bytes)
			return nil, e
		}
	}
	n := uint32(slot + 1)
	if e = g.ModuleParams.insert(g.Budget, GparamsRefEntry{key: k, hash: h, cell: n}); e != nil {
		g.Budget.Release(bytes)
		return nil, e
	}
	if slot == len(g.Cells) {
		g.Cells = append(g.Cells, cell)
	} else {
		g.Cells[slot] = cell
	}
	return &cell.value, nil
}
func (g *GlobalParams) getModule(k Symbol) (ParamsRef, error) {
	if e := g.open(); e != nil {
		return ParamsRef{}, e
	}
	h, e := g.SymbolsField.Hash(k)
	if e != nil {
		return ParamsRef{}, e
	}
	v, e := g.ContextField.NewValue()
	if e != nil {
		return ParamsRef{}, e
	}
	if i := g.ModuleParams.find(k, h); i >= 0 {
		n := g.ModuleParams.Entries[i].cell
		if n == 0 || int(n) > len(g.Cells) || g.Cells[n-1] == nil {
			return ParamsRef{}, ErrGlobalParamsOwner
		}
		if e = g.ContextField.Assign(&v, &g.Cells[n-1].value); e != nil {
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
	k, e := g.SymbolsField.Intern(name)
	if e != nil {
		return ParamsRef{}, e
	}
	return g.getModule(k)
}
func (g *GlobalParams) reset() error {
	if g.globalParams.Owner != 0 {
		if e := g.ContextField.Reset(&g.globalParams); e != nil {
			return e
		}
	}
	for _, x := range g.ModuleParams.Entries {
		if x.state != 2 {
			continue
		}
		i := x.cell
		if i == 0 || int(i) > len(g.Cells) || g.Cells[i-1] == nil {
			return ErrGlobalParamsOwner
		}
		cell := g.Cells[i-1]
		if e := g.ContextField.Del(&cell.value); e != nil {
			return e
		}
		g.Budget.Release(uint64(reflect.TypeOf(*cell).Size()))
		g.Cells[i-1] = nil
	}
	return g.ModuleParams.reset(g.Budget)
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
	for _, x := range g.ModuleDescriptors.Entries {
		if x.state == 2 {
			d := g.Descriptors[x.descriptor-1]
			if d != nil {
				if e := d.Close(); e != nil {
					return e
				}
				g.Descriptors[x.descriptor-1] = nil
			}
		}
	}
	if g.GlobalDescriptors != nil {
		if e := g.GlobalDescriptors.Close(); e != nil {
			return e
		}
	}
	// Global null Ref is a borrowed owner-local scalar after reset; source
	// member destruction decrements it if still initialized, without mutation.
	if g.globalParams.Owner != 0 {
		if e := g.ContextField.Del(&g.globalParams); e != nil {
			return e
		}
	}
	g.ModuleParams.close(g.Budget)
	g.ModuleDescriptions.close(g.Budget)
	g.ModuleDescriptors.close(g.Budget)
	g.Budget.Release(uint64(cap(g.Cells))*uint64(reflect.TypeOf((*GparamsRefCell)(nil)).Size()) + uint64(cap(g.Descriptors))*uint64(reflect.TypeOf((*ParamDescrs)(nil)).Size()) + uint64(cap(g.Bindings))*uint64(reflect.TypeOf(GparamsRegionBinding{}).Size()) + uint64(cap(g.Workspace)) + g.OwnerBytes)
	g.Cells = nil
	g.Descriptors = nil
	g.Bindings = nil
	g.Workspace = nil
	g.OwnerBytes = 0
	g.closed = true
	return nil
}
