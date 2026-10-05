// Source: src/util/gparams.cpp:81-105; src/util/params.cpp:55-77,210-217;
// placement allocation: src/util/memory_manager.h:77-89.
// Author: Leonardo de Moura (gparams: 2012-11-29; params: 2011-05-09).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
package g3

import (
	"errors"
)

var ErrGlobalParamsUnconstructed = errors.New("g3: source global parameters constructor incomplete")

// Public constructors preserve the existing ergonomic all-refund adapter.
// Source member unwind and owning abortion of unfinished placement storage
// are separate actions; neither invokes the uncompleted outer destructor.
func NewParamDescrs(m *SymbolManager, c ParamDescrsConfig) (*ParamDescrs, error) {
	d, e := newSourceParamDescrs(m, c)
	if e != nil && d != nil {
		if cleanup := d.abortConstructor(); cleanup != nil {
			return nil, errors.Join(e, cleanup)
		}
	}
	if e != nil {
		return nil, e
	}
	return d, nil
}
func NewGlobalParams(s *SymbolManager, p *ParamsContext, c GlobalParamsConfig) (*GlobalParams, error) {
	g, e := newSourceGlobalParams(s, p, c)
	if e != nil && g != nil {
		if cleanup := g.abortConstructor(); cleanup != nil {
			return nil, errors.Join(e, cleanup)
		}
	}
	if e != nil {
		return nil, e
	}
	return g, nil
}

func (d *ParamDescrs) abortConstructor() error {
	if d == nil || d.self != d {
		return ErrParamDescrsPrecondition
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	if d.constructorReady || d.constructorCause == nil {
		return ErrParamDescrsPrecondition
	}
	// Completed info/names source members alone unwind in reverse order.
	// Ordinary failure reaches no source allocation between completed info and
	// the infallible default names-vector construction.
	if d.constructorCompleted >= 2 {
		d.budget.Release(pdNameBytes(cap(d.names)))
		d.names = nil
	}
	if d.constructorCompleted >= 1 {
		d.budget.Release(pdEntryBytes(len(d.entries)))
		d.entries = nil
	}
	if d.texts != nil {
		d.texts.Close()
		d.texts = nil
	}
	d.budget.Release(d.ownerBytes)
	d.ownerBytes = 0
	d.closed = true
	return nil
}

// The caller owns g. constructorCompleted is historical source publication;
// constructorUnwind advances only after each real completed child destruction.
func (g *GlobalParams) unwindConstructor() error {
	if g.self != g || g.constructorReady {
		return ErrGlobalParamsOwner
	}
	for g.constructorUnwind != 0 {
		switch g.constructorUnwind {
		case 5:
			if e := g.context.Del(&g.globalParams); e != nil {
				return e
			}
		case 4:
			g.moduleParams.close(g.budget)
		case 3:
			if g.globalDescriptors != nil {
				if e := g.globalDescriptors.sourceFinalizeMembers(); e != nil {
					return e
				}
				g.globalDescriptors = nil
			}
		case 2:
			g.moduleDescriptions.close(g.budget)
		case 1:
			g.moduleDescriptors.close(g.budget)
		default:
			return ErrGlobalParamsOwner
		}
		g.constructorUnwind--
	}
	return nil
}
func (g *GlobalParams) abortConstructor() error {
	if g == nil || g.self != g {
		return ErrGlobalParamsOwner
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return nil
	}
	if g.constructorReady || g.constructorCause == nil {
		return ErrGlobalParamsOwner
	}
	if e := g.unwindConstructor(); e != nil {
		return e
	}
	if g.constructorPendingDescriptor != nil {
		if e := g.constructorPendingDescriptor.abortConstructor(); e != nil {
			return e
		}
		g.constructorPendingDescriptor = nil
	}
	// Additional cold Go context/outer unfinished placement header is owned
	// abortion, not an imp Reset/body or completed source destructor.
	g.budget.Release(g.ownerBytes)
	g.ownerBytes = 0
	g.closed = true
	return nil
}
