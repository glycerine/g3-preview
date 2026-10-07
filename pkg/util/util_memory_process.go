// Source: src/util/memory_manager.cpp:89-113,153-160;
// scripts/mk_genfile_common.py:742-802.
// Source author: Nikolaj Bjorner (2007-07-24).
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"errors"
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrMemoryProcessOwner = errors.New("g3: invalid process memory binding owner")
var ErrMemoryProcessInitializationWait = errors.New("g3: source initialization waits for an unfinished first call on this confined lane")

type MemoryProcessConfig struct {
	Runtime   *Runtime
	Rationals *RationalGlobalService
	Symbols   *SymbolService
	Debug     *DebugContext
	Trace     *TraceContext
	Primes    *DefaultPrimeService
	Budget    *arena.Budget
	Params    ParamsConfig
	Catalog   GlobalParamsCatalogConfig
	Global    GlobalParamsConfig
}

// These children remain caller-owned. Only the parent header and the concrete
// parameter contexts prepared at the reached generated gparams call are owned
// here. Legacy child storage is charged Go storage, not global Runtime tickets.
// Native process/OpenMP binding and complete global allocation accounting remain
// open. The generated source order is implemented explicitly, without callbacks.
type MemoryProcess struct {
	self                                                     *MemoryProcess
	config                                                   MemoryProcessConfig
	context                                                  *ParamsContext
	catalog                                                  *GlobalParamsCatalog
	parameters                                               *GlobalParamsService
	header                                                   uint64
	initialized, fullyInitialized, finalizing                bool
	initializationStage, preparationStage, finalizationStage uint8
	sourceCause                                              error
	closing, closed                                          bool
}

func NewMemoryProcess(c MemoryProcessConfig) (*MemoryProcess, error) {
	if c.Runtime == nil || c.Runtime.Budget() != c.Budget || c.Rationals == nil || c.Rationals.config.Integers.Memory != c.Runtime || c.Rationals.config.Integers.Cells.Budget != c.Budget || c.Symbols == nil || c.Debug == nil || c.Trace == nil || c.Primes == nil {
		return nil, arena.ErrConfig
	}
	if e := c.Runtime.CheckOpen(); e != nil {
		return nil, e
	}
	if e := c.Rationals.open(); e != nil {
		return nil, e
	}
	if e := c.Symbols.open(); e != nil {
		return nil, e
	}
	if e := c.Debug.open(); e != nil {
		return nil, e
	}
	if e := c.Trace.open(); e != nil {
		return nil, e
	}
	if e := c.Primes.open(); e != nil {
		return nil, e
	}
	if c.Symbols.config.Budget != c.Budget || c.Debug.BudgetField != c.Budget || c.Trace.budget != c.Budget || c.Primes.budget != c.Budget {
		return nil, arena.ErrConfig
	}
	if c.Params.Budget != c.Budget || c.Params.Text.Descriptors.Budget != c.Budget || c.Catalog.Budget != c.Budget || c.Global.Budget != c.Budget || c.Global.Register != nil || c.Global.Catalog != nil {
		return nil, arena.ErrConfig
	}
	if e := c.Debug.canBindTagRuntime(c.Runtime); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(MemoryProcess{}).Size())
	if e := c.Budget.Reserve(n); e != nil {
		return nil, e
	}
	p := &MemoryProcess{config: c, header: n}
	p.self = p
	if e := c.Debug.BindTagRuntime(c.Runtime); e != nil {
		c.Budget.Release(n)
		return nil, e
	}
	return p, nil
}
func (p *MemoryProcess) open() error {
	if p == nil || p.self != p {
		return ErrMemoryProcessOwner
	}
	if p.closed || p.closing {
		return arena.ErrClosed
	}
	return nil
}
func (p *MemoryProcess) Initialize(maxSize uint64) error {
	if e := p.open(); e != nil {
		return e
	}
	// size_t conversion precedes the actual UINT_MAX comparison on both profiles.
	if p.config.Runtime.SizeTBits() == 32 {
		maxSize = uint64(uint32(maxSize))
	}
	if maxSize != math.MaxUint32 {
		if e := p.config.Runtime.SetMaxSize(maxSize); e != nil {
			return e
		}
	}
	if p.initialized {
		if !p.fullyInitialized {
			return ErrMemoryProcessInitializationWait
		}
		return nil
	}
	p.initialized = true // source first-call publication before all child actions
	if e := p.config.Runtime.ClearOutOfMemoryForInitialization(); e != nil {
		return e
	}
	if e := p.initializeChildren(); e != nil {
		return e
	}
	p.fullyInitialized = true
	return nil
}
func (p *MemoryProcess) Finalize() error {
	if e := p.open(); e != nil {
		return e
	}
	if !p.initialized {
		return nil
	}
	// A stopped source destructor is not replayed by the owning adapter. The
	// native exception/repeated-static-DT domain is explicitly guarded here.
	if p.sourceCause != nil {
		return p.sourceCause
	}
	p.finalizing = true
	if e := p.finalizeChildren(); e != nil {
		p.sourceCause = e
		return e
	}
	p.initialized = false
	p.finalizing = false
	// Source deliberately retains fully_initialized and all accounting counters.
	return nil
}
