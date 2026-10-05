// Source: src/util/gparams.cpp:81-123,633-644; src/util/params.cpp:59-86.
// Author: Leonardo de Moura (2012-11-29).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

// SourceFinalize executes imp's destructor body and reverse members. A reached
// error is sticky; only the separate owning adapter may acknowledge its prefix.
// This owner-local operation does not implement the process g_imp pointer.
func (g *GlobalParams) SourceFinalize() error {
	if g == nil || g.self != g {
		return ErrGlobalParamsOwner
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.constructorReady {
		return ErrGlobalParamsUnconstructed
	}
	if g.sourceCause != nil {
		return g.sourceCause
	}
	if g.sourceCompleted {
		return nil
	}
	if g.closed {
		return arena.ErrClosed
	}
	g.sourceStarted = true
	e := g.sourceFinalizeBody(false)
	if e != nil {
		g.sourceCause = e
	} else {
		g.sourceCompleted = true
	}
	return e
}

func (g *GlobalParams) sourceFinalizeBody(owning bool) error {
	for {
		switch g.sourceStage {
		case 0:
			if g.globalParams.owner != 0 {
				if e := g.sourceResetValue(owning); e != nil {
					return e
				}
			}
			g.sourceStage = 1
		case 1:
			for g.sourceIndex < uint32(len(g.moduleParams.entries)) {
				x := g.moduleParams.entries[g.sourceIndex]
				if x.state == 2 {
					if x.cell == 0 || int(x.cell) > len(g.cells) || g.cells[x.cell-1] == nil {
						return ErrGlobalParamsOwner
					}
					cell := g.cells[x.cell-1]
					if e := g.sourceDecrementValue(&cell.value, owning); e != nil {
						return e
					}
					g.budget.Release(uint64(reflect.TypeOf(*cell).Size()))
					g.cells[x.cell-1] = nil
				}
				g.sourceIndex++
			}
			g.sourceStage = 2
			g.sourceIndex = 0
		case 2:
			if e := g.moduleParams.reset(g.budget); e != nil {
				return e
			}
			g.sourceStage = 3
		case 3:
			for g.sourceIndex < uint32(len(g.moduleDescriptors.entries)) {
				x := g.moduleDescriptors.entries[g.sourceIndex]
				if x.state == 2 {
					if x.descriptor == 0 || int(x.descriptor) > len(g.descriptors) || g.descriptors[x.descriptor-1] == nil {
						return ErrGlobalParamsOwner
					}
					d := g.descriptors[x.descriptor-1]
					if e := d.sourceFinalizeMembers(); e != nil {
						return e
					}
					g.descriptors[x.descriptor-1] = nil
				}
				g.sourceIndex++
			}
			g.sourceStage = 4
			g.sourceIndex = 0
		case 4:
			if g.globalParams.owner != 0 {
				if e := g.sourceDecrementValue(&g.globalParams, owning); e != nil {
					return e
				}
			}
			g.sourceStage = 5
		case 5:
			g.moduleParams.close(g.budget)
			g.sourceStage = 6
		case 6:
			if g.globalDescriptors != nil {
				if e := g.globalDescriptors.sourceFinalizeMembers(); e != nil {
					return e
				}
				g.globalDescriptors = nil
			}
			g.sourceStage = 7
		case 7:
			g.moduleDescriptions.close(g.budget)
			g.sourceStage = 8
		case 8:
			g.moduleDescriptors.close(g.budget)
			g.sourceStage = 9
		case 9:
			// Cold directories/text workspace are additional actual Go ownership.
			g.budget.Release(uint64(cap(g.cells))*uint64(reflect.TypeOf((*gparamsRefCell)(nil)).Size()) + uint64(cap(g.descriptors))*uint64(reflect.TypeOf((*ParamDescrs)(nil)).Size()) + uint64(cap(g.bindings))*uint64(reflect.TypeOf(gparamsRegionBinding{}).Size()) + uint64(cap(g.workspace)) + g.ownerBytes)
			g.cells = nil
			g.descriptors = nil
			g.bindings = nil
			g.workspace = nil
			g.ownerBytes = 0
			g.sourceStage = 10
			g.closed = true
		case 10:
			return nil
		default:
			return ErrGlobalParamsOwner
		}
	}
}

// sourceDecrementValue captures a receipt only at the child's actual guarded
// zero-reference destructor entry. The owning path finishes that exact child;
// it cannot adopt a stopped block reached by another caller's decrement.
func (g *GlobalParams) sourceDecrementValue(v *ParamsRef, owning bool) error {
	if owning && g.sourceDecrement.invocation != 0 {
		if e := g.context.CompleteDecrement(v, g.sourceDecrement); e != nil {
			return e
		}
		g.sourceDecrement = ParamsDecrementReceipt{}
		return nil
	}
	if e := g.context.DelTracked(v, &g.sourceDecrement); e != nil {
		return e
	}
	g.sourceDecrement = ParamsDecrementReceipt{}
	return nil
}

// CleanupStopped finishes the matched child or acknowledges a completed table
// release. The original source cause and source completion flag remain sticky.
func (g *GlobalParams) CleanupStopped() error {
	if g == nil || g.self != g {
		return ErrGlobalParamsOwner
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return nil
	}
	if !g.sourceStarted {
		return ErrGlobalParamsOwner
	}
	if g.sourceCause != nil {
		switch g.sourceStage {
		case 0:
			if g.sourceReset.invocation == 0 {
				return g.sourceCause
			}
		case 1, 4:
			if g.sourceDecrement.invocation == 0 {
				return g.sourceCause
			}
		case 2:
			g.moduleParams.close(g.budget)
			g.sourceStage = 3
		default:
			return g.sourceCause
		}
	}
	return g.sourceFinalizeBody(true)
}

// The reset receipt is stamped at the live child reset entry, independently
// from zero-reference destruction. Owning completion preserves child identity.
func (g *GlobalParams) sourceResetValue(owning bool) error {
	if owning && g.sourceReset.invocation != 0 {
		if e := g.context.CompleteReset(&g.globalParams, g.sourceReset); e != nil {
			return e
		}
		g.sourceReset = ParamsResetReceipt{}
		return nil
	}
	if e := g.context.ResetTracked(&g.globalParams, &g.sourceReset); e != nil {
		return e
	}
	g.sourceReset = ParamsResetReceipt{}
	return nil
}
