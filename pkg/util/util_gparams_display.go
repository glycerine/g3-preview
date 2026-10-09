// Source: src/util/gparams.cpp:150-160,446-550,623-627; src/util/gparams.h.
package util

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// Region bindings are cold borrowed owners. The source stores char* without
// reading it at registration; closed/stale referents fail only at output.
func (g *GlobalParams) bindDescriptionRegion(r *arena.BytesRegion) error {
	origin := r.OwnerID()
	if origin == 0 {
		return ErrGlobalParamsOwner
	}
	for _, x := range g.Bindings {
		if x.origin == origin {
			if x.region != r {
				return ErrGlobalParamsOwner
			}
			return nil
		}
	}
	if len(g.Bindings) == cap(g.Bindings) {
		n, e := pdGrowth(cap(g.Bindings))
		if e != nil {
			return e
		}
		stride := uint64(reflect.TypeOf(GparamsRegionBinding{}).Size())
		if uint64(n) > uint64(^uint(0)>>1)/stride {
			return arena.ErrBounds
		}
		if e = g.Budget.Reserve(uint64(n) * stride); e != nil {
			return e
		}
		next := make([]GparamsRegionBinding, len(g.Bindings), n)
		copy(next, g.Bindings)
		g.Budget.Release(uint64(cap(g.Bindings)) * stride)
		g.Bindings = next
	}
	g.Bindings = append(g.Bindings, GparamsRegionBinding{origin, r})
	return nil
}
func (g *GlobalParams) registerModuleDescription(name string, r *arena.BytesRegion, ref arena.BytesRegionRef, offset uint32) error {
	if e := g.open(); e != nil {
		return e
	}
	key, e := g.SymbolsField.Intern(name)
	if e != nil {
		return e
	}
	if _, e := g.SymbolsField.Hash(key); e != nil {
		return e
	}
	var text GlobalParamCString
	if r != nil {
		if e = g.bindDescriptionRegion(r); e != nil {
			return e
		}
		text = GlobalParamCString{Origin: r.OwnerID(), Ref: ref, Offset: offset, Present: true}
	}
	g.ModuleDescriptions.set(key, text)
	return nil
}
func (r GlobalParamsRegistrar) RegisterModuleDescription(name string, region *arena.BytesRegion, ref arena.BytesRegionRef, offset uint32) error {
	if r.owner == nil || !r.owner.registering {
		return ErrGlobalParamsOwner
	}
	return r.owner.registerModuleDescription(name, region, ref, offset)
}
func (g *GlobalParams) RegisterModuleDescription(name string, region *arena.BytesRegion, ref arena.BytesRegionRef, offset uint32) error {
	if g == nil {
		return ErrGlobalParamsOwner
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.registerModuleDescription(name, region, ref, offset)
}
func (g *GlobalParams) descriptionInto(out *StringBuffer64, text GlobalParamCString) error {
	if !text.Present {
		return ErrParamsStreamFailure
	}
	var r *arena.BytesRegion
	for _, x := range g.Bindings {
		if x.origin == text.Origin {
			r = x.region
			break
		}
	}
	if r == nil {
		return ErrGlobalParamsOwner
	}
	for i := text.Offset; ; i++ {
		ch, e := r.Read(text.Ref, i)
		if e != nil {
			return e
		}
		if ch == 0 {
			return nil
		}
		if e = out.AppendByte(ch); e != nil {
			return e
		}
		if i == ^uint32(0) {
			return arena.ErrBounds
		}
	}
}
func (g *GlobalParams) moduleHeaderInto(out *StringBuffer64, key Symbol) error {
	if out == nil {
		return ErrStringBufferOwner
	}
	if e := out.AppendCString("[module] "); e != nil {
		return e
	}
	if e := out.AppendSymbol(g.SymbolsField, key); e != nil {
		return e
	}
	if _, e := g.SymbolsField.Hash(key); e != nil {
		return e
	}
	if text, ok := g.ModuleDescriptions.get(key); ok {
		if e := out.AppendCString(", description: "); e != nil {
			return e
		}
		if e := g.descriptionInto(out, text); e != nil {
			return e
		}
	}
	return out.AppendByte('\n')
}
func (g *GlobalParams) descriptorEntry(n uint32) (*ParamDescrs, error) {
	if n == 0 || uint64(n) > uint64(len(g.Descriptors)) || g.Descriptors[n-1] == nil {
		return nil, ErrGlobalParamsOwner
	}
	return g.Descriptors[n-1], nil
}
func (g *GlobalParams) DisplayInto(out *StringBuffer64, indent uint32, smt2, includeDescr bool) error {
	if g == nil {
		return ErrGlobalParamsOwner
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if e := g.open(); e != nil {
		return e
	}
	if out == nil {
		return ErrStringBufferOwner
	}
	if e := out.AppendCString("Global parameters\n"); e != nil {
		return e
	}
	if e := g.ensureRegistered(); e != nil {
		return e
	}
	if e := g.GlobalDescriptors.DisplayInto(out, indent+4, smt2, includeDescr); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if !smt2 {
		if e := out.AppendCString("To set a module parameter, use <module-name>.<parameter-name>=value\nExample:  pp.decimal=true\n\n"); e != nil {
			return e
		}
	}
	for key, descriptor := range g.ModuleDescriptors.values.All() {
		if e := g.moduleHeaderInto(out, key); e != nil {
			return e
		}
		d, e := g.descriptorEntry(descriptor)
		if e != nil {
			return e
		}
		if e = d.DisplayInto(out, indent+4, smt2, includeDescr); e != nil {
			return e
		}
	}
	return nil
}
func (g *GlobalParams) DisplayModulesInto(out *StringBuffer64) error {
	if g == nil {
		return ErrGlobalParamsOwner
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if e := g.ensureRegistered(); e != nil {
		return e
	}
	for key := range g.ModuleDescriptors.values.All() {
		if e := g.moduleHeaderInto(out, key); e != nil {
			return e
		}
	}
	return nil
}
func (g *GlobalParams) DisplayModuleInto(out *StringBuffer64, name string) error {
	if g == nil {
		return ErrGlobalParamsOwner
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if e := g.open(); e != nil {
		return e
	}
	key, e := g.SymbolsField.Intern(name)
	if e != nil {
		return e
	}
	return g.displayModuleKey(out, key)
}

// Caller owns the existing imp lock and has reached its ordinary open guard.
func (g *GlobalParams) displayModuleKey(out *StringBuffer64, key Symbol) error {
	d, found, e := g.findModuleDescriptor(key)
	if e != nil {
		return e
	}
	if !found {
		return g.exception(gpText("unknown module '"), gpSymbol(key), gpText("'"))
	}
	if e = g.moduleHeaderInto(out, key); e != nil {
		return e
	}
	return d.DisplayInto(out, 4, false, true)
}

// DisplayParameterInto preserves the source's unsolicited stdout line in the
// explicit debugOut stream before descriptor lookup. Binding that stream to
// process stdout remains the runtime caller's job; it is not a global CLI.
func (g *GlobalParams) DisplayParameterInto(out, debugOut *StringBuffer64, name string) error {
	if g == nil {
		return ErrGlobalParamsOwner
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if e := g.open(); e != nil {
		return e
	}
	module, param, e := g.normalizeName(name)
	if e != nil {
		return e
	}
	if debugOut == nil {
		return ErrStringBufferOwner
	}
	if e = debugOut.AppendCString(name); e != nil {
		return e
	}
	if e = debugOut.AppendByte(' '); e != nil {
		return e
	}
	if e = debugOut.AppendSymbol(g.SymbolsField, module); e != nil {
		return e
	}
	if e = debugOut.AppendByte(' '); e != nil {
		return e
	}
	if e = debugOut.AppendSymbol(g.SymbolsField, param); e != nil {
		return e
	}
	if e = debugOut.AppendByte('\n'); e != nil {
		return e
	}
	var d *ParamDescrs
	if module == (Symbol{}) {
		if e = g.ensureRegistered(); e != nil {
			return e
		}
		d = g.GlobalDescriptors
	} else {
		var found bool
		d, found, e = g.findModuleDescriptor(module)
		if e != nil {
			return e
		}
		if !found {
			return g.exception(gpText("unknown module '"), gpSymbol(module), gpText("'"))
		}
	}
	found, e := d.Contains(param)
	if e != nil {
		return e
	}
	if !found {
		return g.unknownParameter(param, d, module)
	}
	if out == nil {
		return ErrStringBufferOwner
	}
	if e = out.AppendCString("  name:           "); e != nil {
		return e
	}
	if e = out.AppendSymbol(g.SymbolsField, param); e != nil {
		return e
	}
	if e = out.AppendByte('\n'); e != nil {
		return e
	}
	if module != (Symbol{}) {
		if e = out.AppendCString("  module:         "); e != nil {
			return e
		}
		if e = out.AppendSymbol(g.SymbolsField, module); e != nil {
			return e
		}
		if e = out.AppendCString("\n  qualified name: "); e != nil {
			return e
		}
		if e = out.AppendSymbol(g.SymbolsField, module); e != nil {
			return e
		}
		if e = out.AppendByte('.'); e != nil {
			return e
		}
		if e = out.AppendSymbol(g.SymbolsField, param); e != nil {
			return e
		}
		if e = out.AppendByte('\n'); e != nil {
			return e
		}
	}
	// Lock order matches DisplayInto: output→descriptor→symbols. Direct text
	// reads use the charged descriptor words and allocate no intermediate string.
	out.State.Mu.Lock()
	defer out.State.Mu.Unlock()
	if e = out.State.Open(out.Self == out); e != nil {
		return e
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if e = d.open(); e != nil {
		return e
	}
	i, e := d.find(param)
	if e != nil {
		return e
	}
	if i < 0 {
		return ErrParamDescrsPrecondition
	}
	info := d.Entries[i].info
	if e = out.State.AppendCString(out.Inline[:], "  type:           "); e != nil {
		return e
	}
	if e = out.State.AppendCString(out.Inline[:], info.kind.String()); e != nil {
		return e
	}
	if e = out.State.AppendCString(out.Inline[:], "\n  description:    "); e != nil {
		return e
	}
	if info.descr.ref == (arena.VectorRef{}) {
		return ErrParamsStreamFailure
	}
	if e = d.writeTextInto(out, info.descr); e != nil {
		return e
	}
	if e = out.State.AppendCString(out.Inline[:], "\n  default value:  "); e != nil {
		return e
	}
	if info.def.ref == (arena.VectorRef{}) {
		return ErrParamsStreamFailure
	}
	if e = d.writeTextInto(out, info.def); e != nil {
		return e
	}
	return out.State.AppendByte(out.Inline[:], '\n')
}
