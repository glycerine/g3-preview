// Source: src/api/dll/mem_initializer.cpp, src/shell/mem_initializer.cpp,
// src/test/mem_initializer.cpp, generated mem_initialize body.
package process

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

// InitializationConfig supplies the two source static services and the Go
// storage configurations needed to construct gparams after symbols exist.
// The runtime and services are borrowed by the generated binding. A nil Trace
// uses the process trace tags; pass the same explicit owner to finalization.
type InitializationConfig struct {
	Runtime  *memory.Runtime
	Rational *numeral.RationalGlobalService
	Symbols  *symbol.Service
	Debug    *diagnostic.DebugContext
	Trace    *diagnostic.TraceTags
	Params   params.ParamsConfig
	Catalog  params.GlobalParamsCatalogConfig
	Globals  params.GlobalParamsConfig
}

// Initialized owns only the Go parameter context/catalog/service prepared
// during generated initialization. Rational, symbol, memory and trace owners
// remain with the caller. A partial result is returned with any stage error so
// its completed Go owners can still be closed.
type Initialized struct {
	self           *Initialized
	Context        *params.ParamsContext
	Catalog        *params.GlobalParamsCatalog
	Globals        *params.GlobalParamsService
	Trace          *diagnostic.TraceTags
	debugPrevious  *diagnostic.DebugContext
	paramsPrevious *params.GlobalParamsService
	paramsBound    bool
	closed         bool
}

// InitializeGenerated executes the first-call source order. A repeated call
// updates the shared maximum through memory::initialize and returns nil,nil;
// the caller retains the first result for later finalization and owning close.
func InitializeGenerated(c InitializationConfig, maxSize uint64) (*Initialized, error) {
	if c.Runtime == nil {
		return nil, arena.ErrConfig
	}
	p := c.Runtime.ProcessOwner()
	if p == nil {
		return nil, arena.ErrConfig
	}
	// Source repeated calls update the maximum and return without reading any
	// generated child. This also preserves the pending-stage guard in Runtime.
	if p.InitializationStarted() {
		_, err := c.Runtime.BeginSourceInitialize(maxSize)
		return nil, err
	}
	if c.Rational == nil || c.Symbols == nil || c.Debug == nil || c.Params.Budget == nil ||
		c.Catalog.Budget != c.Params.Budget || c.Globals.Budget != c.Params.Budget || c.Globals.Catalog != nil {
		return nil, arena.ErrConfig
	}
	if _, err := c.Debug.AssertionsEnabled(); err != nil {
		return nil, err
	}
	rationalProcess, err := c.Rational.ProcessOwner()
	if err != nil {
		return nil, err
	}
	if rationalProcess == nil || rationalProcess != c.Runtime.ProcessOwner() {
		return nil, arena.ErrConfig
	}
	first, err := c.Runtime.BeginSourceInitialize(maxSize)
	if err != nil || !first {
		return nil, err
	}
	result := &Initialized{}
	result.self = result
	if err = c.Rational.InitializeSource(); err != nil {
		return result, err
	}
	if err = c.Symbols.InitializeSource(); err != nil {
		return result, err
	}
	s, err := c.Symbols.Manager()
	if err != nil {
		return result, err
	}
	if err = s.BindDebug(c.Debug); err != nil {
		return result, err
	}
	q, err := c.Rational.Service()
	if err != nil {
		return result, err
	}
	result.Context, err = params.NewParamsContext(s, q, c.Params)
	if err != nil {
		return result, err
	}
	result.Catalog, err = params.NewGlobalParamsCatalog(s, c.Catalog)
	if err != nil {
		return result, err
	}
	c.Globals.Catalog = result.Catalog
	result.Globals, err = params.NewGlobalParamsService(s, result.Context, c.Globals)
	if err != nil {
		return result, err
	}
	if err = result.Globals.BindNamespaceAssertions(c.Debug); err != nil {
		return result, err
	}
	result.Trace = c.Trace
	if result.Trace == nil {
		result.Trace = diagnostic.BorrowProcessTraceTags()
	}
	if err = result.Globals.BindNamespaceTrace(result.Trace); err != nil {
		return result, err
	}
	if err = result.Globals.InitializeSource(); err != nil {
		return result, err
	}
	result.paramsPrevious, err = params.BindProcessGlobalParamsService(result.Globals)
	if err != nil {
		return result, err
	}
	result.paramsBound = true
	if err = c.Runtime.CompleteSourceInitialize(); err != nil {
		return result, err
	}
	result.debugPrevious, err = diagnostic.BindProcessDebugContext(c.Debug)
	if err != nil {
		return result, err
	}
	return result, nil
}

// Close releases only the Go owners prepared by InitializeGenerated, after
// the caller has finished using them and any generated finalization has run.
func (v *Initialized) Close() error {
	if v == nil || v.self != v {
		return arena.ErrConfig
	}
	if v.closed {
		return nil
	}
	if v.Globals != nil {
		if err := v.Globals.Close(); err != nil {
			return err
		}
		v.Globals = nil
	}
	if v.paramsBound {
		if _, err := params.BindProcessGlobalParamsService(v.paramsPrevious); err != nil {
			return err
		}
		v.paramsPrevious = nil
		v.paramsBound = false
	}
	if v.Catalog != nil {
		if err := v.Catalog.Close(); err != nil {
			return err
		}
		v.Catalog = nil
	}
	if v.Context != nil {
		if err := v.Context.Close(); err != nil {
			return err
		}
		v.Context = nil
	}
	v.Trace = nil
	if v.debugPrevious != nil {
		if _, err := diagnostic.BindProcessDebugContext(v.debugPrevious); err != nil {
			return err
		}
		v.debugPrevious = nil
	}
	v.closed = true
	return nil
}
