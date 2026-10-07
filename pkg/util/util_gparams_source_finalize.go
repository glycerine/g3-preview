// Source: src/util/gparams.cpp:81-123,633-644; src/util/params.cpp:59-86.
package util

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
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
			if g.globalParams.Owner != 0 {
				if e := g.sourceResetValue(owning); e != nil {
					return e
				}
			}
			g.sourceStage = 1
		case 1:
			for g.sourceIndex < uint32(len(g.ModuleParams.Entries)) {
				x := g.ModuleParams.Entries[g.sourceIndex]
				if x.state == 2 {
					if x.cell == 0 || int(x.cell) > len(g.Cells) || g.Cells[x.cell-1] == nil {
						return ErrGlobalParamsOwner
					}
					cell := g.Cells[x.cell-1]
					if e := g.sourceDecrementValue(&cell.value, owning); e != nil {
						return e
					}
					g.Budget.Release(uint64(reflect.TypeOf(*cell).Size()))
					g.Cells[x.cell-1] = nil
				}
				g.sourceIndex++
			}
			g.sourceStage = 2
			g.sourceIndex = 0
		case 2:
			if e := g.ModuleParams.reset(g.Budget); e != nil {
				return e
			}
			g.sourceStage = 3
		case 3:
			for g.sourceIndex < uint32(len(g.ModuleDescriptors.Entries)) {
				x := g.ModuleDescriptors.Entries[g.sourceIndex]
				if x.state == 2 {
					if x.descriptor == 0 || int(x.descriptor) > len(g.Descriptors) || g.Descriptors[x.descriptor-1] == nil {
						return ErrGlobalParamsOwner
					}
					d := g.Descriptors[x.descriptor-1]
					if e := d.sourceFinalizeMembers(); e != nil {
						return e
					}
					g.Descriptors[x.descriptor-1] = nil
				}
				g.sourceIndex++
			}
			g.sourceStage = 4
			g.sourceIndex = 0
		case 4:
			if g.globalParams.Owner != 0 {
				if e := g.sourceDecrementValue(&g.globalParams, owning); e != nil {
					return e
				}
			}
			g.sourceStage = 5
		case 5:
			g.ModuleParams.close(g.Budget)
			g.sourceStage = 6
		case 6:
			if g.GlobalDescriptors != nil {
				if e := g.GlobalDescriptors.sourceFinalizeMembers(); e != nil {
					return e
				}
				g.GlobalDescriptors = nil
			}
			g.sourceStage = 7
		case 7:
			g.ModuleDescriptions.close(g.Budget)
			g.sourceStage = 8
		case 8:
			g.ModuleDescriptors.close(g.Budget)
			g.sourceStage = 9
		case 9:
			// Cold directories/text workspace are additional actual Go ownership.
			g.Budget.Release(uint64(cap(g.Cells))*uint64(reflect.TypeOf((*GparamsRefCell)(nil)).Size()) + uint64(cap(g.Descriptors))*uint64(reflect.TypeOf((*ParamDescrs)(nil)).Size()) + uint64(cap(g.Bindings))*uint64(reflect.TypeOf(GparamsRegionBinding{}).Size()) + uint64(cap(g.Workspace)) + g.OwnerBytes)
			g.Cells = nil
			g.Descriptors = nil
			g.Bindings = nil
			g.Workspace = nil
			g.OwnerBytes = 0
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
	if owning && g.sourceDecrement.InvocationField != 0 {
		if e := g.ContextField.CompleteDecrement(v, g.sourceDecrement); e != nil {
			return e
		}
		g.sourceDecrement = ParamsDecrementReceipt{}
		return nil
	}
	if e := g.ContextField.DelTracked(v, &g.sourceDecrement); e != nil {
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
			if g.sourceDecrement.InvocationField == 0 {
				return g.sourceCause
			}
		case 2:
			g.ModuleParams.close(g.Budget)
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
		if e := g.ContextField.CompleteReset(&g.globalParams, g.sourceReset); e != nil {
			return e
		}
		g.sourceReset = ParamsResetReceipt{}
		return nil
	}
	if e := g.ContextField.ResetTracked(&g.globalParams, &g.sourceReset); e != nil {
		return e
	}
	g.sourceReset = ParamsResetReceipt{}
	return nil
}
