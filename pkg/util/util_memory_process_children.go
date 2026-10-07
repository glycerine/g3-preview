// Source: src/util/memory_manager.cpp:103,156;
// scripts/mk_genfile_common.py:742-802; src/util/gparams.cpp:633-644.
// Source author: Nikolaj Bjorner (2007-07-24); Leonardo de Moura (2012-11-29).
// Licensed under the MIT license; see LICENSE.txt.
package util

func (p *MemoryProcess) initializeChildren() error {
	// Stable-priority generated initialization: rational, symbol, gparams.
	if e := p.config.Rationals.InitializeSource(); e != nil {
		return e
	}
	p.initializationStage = 1
	if e := p.config.Symbols.InitializeSource(); e != nil {
		return e
	}
	p.initializationStage = 2
	s, e := p.config.Symbols.Manager()
	if e != nil {
		return e
	}
	// Real Go ownership contexts are prepared only after their actual source
	// dependencies exist. Each completed acquisition is published immediately;
	// failure does not adopt or destroy the caller's earlier children.
	p.context, e = NewParamsContext(s, p.config.Rationals.service, p.config.Params)
	if e != nil {
		return e
	}
	p.preparationStage = 1
	p.catalog, e = NewGlobalParamsCatalog(s, p.config.Catalog)
	if e != nil {
		return e
	}
	p.preparationStage = 2
	c := p.config.Global
	c.Catalog = p.catalog
	p.parameters, e = NewGlobalParamsService(s, p.context, c)
	if e != nil {
		return e
	}
	p.preparationStage = 3
	// The generated globals share this actual assertion owner. Binding stores
	// the borrow; only a reached namespace assertion reads its state.
	if e = p.parameters.BindNamespaceAssertions(p.config.Debug); e != nil {
		return e
	}
	if e = p.parameters.InitializeSource(); e != nil {
		return e
	}
	p.initializationStage = 3
	return nil
}
func (p *MemoryProcess) finalizeChildren() error {
	// Header order emitted by the generator, not reverse initializer order.
	if e := p.config.Debug.FinalizeDebug(); e != nil {
		return e
	}
	p.finalizationStage = 1
	// No prepared parameters means the actual g_imp pointer is still null:
	// source gparams::finalize does no deallocation on that branch.
	if p.parameters != nil {
		if e := p.parameters.FinalizeSource(); e != nil {
			return e
		}
	}
	p.finalizationStage = 2
	if e := p.config.Primes.Finalize(); e != nil {
		return e
	}
	p.finalizationStage = 3
	if e := p.config.Rationals.FinalizeSource(); e != nil {
		return e
	}
	p.finalizationStage = 4
	if e := p.config.Symbols.FinalizeSource(); e != nil {
		return e
	}
	p.finalizationStage = 5
	if e := p.config.Trace.FinalizeTrace(); e != nil {
		return e
	}
	p.finalizationStage = 6
	return nil
}
