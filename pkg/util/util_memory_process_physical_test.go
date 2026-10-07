// Source: src/util/memory_manager.cpp:89-113,153-160; src/util/gparams.cpp:81-105.
package util

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func memoryProcessNumericalBytes(t *testing.T, g *RationalGlobalService, r *Runtime) uint64 {
	t.Helper()
	n := r.Stats().TotalBytes + g.header
	sv := reflect.ValueOf(g.service).Elem()
	n += testFieldBySourceName(sv, "headerBytes").Uint() + testFieldBySourceName(sv, "cacheBytes").Uint()
	seen := map[uintptr]bool{}
	slab := func(v reflect.Value) uint64 {
		if v.IsNil() {
			return 0
		}
		s := v.Elem()
		out := testFieldBySourceName(s, "directoryBytes").Uint()
		if !testFieldBySourceName(s, "closed").Bool() {
			out += uint64(s.Type().Size())
		}
		pages := testFieldBySourceName(s, "pages")
		for i := 0; i < pages.Len(); i++ {
			p := pages.Index(i)
			for _, f := range []string{"records", "meta"} {
				a := testFieldBySourceName(p, f)
				out += uint64(a.Len()) * uint64(a.Type().Elem().Size())
			}
		}
		return out
	}
	addZ := func(v reflect.Value) {
		if v.IsNil() {
			return
		}
		if seen[v.Pointer()] {
			t.Fatal("duplicate owning Integer manager")
		}
		seen[v.Pointer()] = true
		z := v.Elem()
		n += testFieldBySourceName(z, "headerBytes").Uint()
		a := testFieldBySourceName(z, "allocator")
		if !a.IsNil() {
			a = a.Elem()
			loc := testFieldBySourceName(a, "locations")
			n += testFieldBySourceName(a, "header").Uint() + uint64(loc.Cap())*uint64(loc.Type().Elem().Size())
		}
		n += slab(testFieldBySourceName(z, "cells"))
		w := testFieldBySourceName(z, "words")
		if !w.IsNil() {
			w = w.Elem()
			if !testFieldBySourceName(w, "closed").Bool() {
				n += uint64(w.Type().Size())
			}
			n += testFieldBySourceName(w, "directoryBytes").Uint() + slab(testFieldBySourceName(w, "descriptors"))
			pages := testFieldBySourceName(w, "pages")
			for i := 0; i < pages.Len(); i++ {
				p := pages.Index(i)
				n += uint64(testFieldBySourceName(p, "data").Len())*4 + uint64(testFieldBySourceName(p, "next").Len())*4
			}
		}
		nat := testFieldBySourceName(z, "natural")
		if !nat.IsNil() {
			nat = nat.Elem()
			n += testFieldBySourceName(nat, "headerBytes").Uint()
			if !testFieldBySourceName(nat, "text").IsNil() {
				t.Fatal("undeclared Natural text owner")
			}
		}
	}
	addM := func(v reflect.Value) {
		if v.IsNil() {
			return
		}
		m := v.Elem()
		n += testFieldBySourceName(m, "headerBytes").Uint()
		addZ(testFieldBySourceName(m, "z"))
	}
	addM(testFieldBySourceName(sv, "manager"))
	addZ(testFieldBySourceName(sv, "initializationFailure"))
	for node := g.acquisitions; node != nil; node = node.next {
		if node.owner != g || node.invocation == 0 || node.bytes != uint64(reflect.TypeOf(*node).Size()) {
			t.Fatal("invalid acquisition receipt")
		}
		n += node.bytes
		switch node.acquired {
		case 0:
			if node.manager != nil || node.failed != nil {
				t.Fatal("unacquired node claims owner")
			}
		case 1:
			addM(reflect.ValueOf(node.manager))
		case 2:
			addZ(reflect.ValueOf(node.failed))
		default:
			t.Fatal("invalid constructor acquisition")
		}
	}

	return n
}

func memoryProcessColdAudit(t *testing.T, p *MemoryProcess) {
	t.Helper()
	c := p.config
	n := p.header + memoryProcessNumericalBytes(t, c.Rationals, c.Runtime)
	n += c.Symbols.ownerBytes
	if s := c.Symbols.manager; s != nil {
		v := s.Stats()
		d := v.Region.Descriptors
		n += v.OwnerBytes + v.Table.OwnerBytes + v.Table.TableBytes + v.Region.OwnerBytes + v.Region.PayloadBytes + v.Region.PageHeaderBytes + v.Region.DirectoryBytes + v.Region.LogBytes + v.Region.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
	}
	n += c.Debug.ownerBytes
	n += debugTagColdBytes(c.Debug)
	n += c.Trace.ownerBytes
	if c.Trace.tags != nil {
		s := c.Trace.tags.Stats()
		n += s.OwnerBytes + s.TableBytes + s.DirectoryBytes
	}
	regions := map[uintptr]bool{}
	for _, region := range []*arena.BytesRegion{c.Debug.region, c.Trace.region} {
		id := reflect.ValueOf(region).Pointer()
		if regions[id] {
			continue
		}
		regions[id] = true
		s := region.Stats()
		d := s.Descriptors
		n += s.OwnerBytes + s.PayloadBytes + s.PageHeaderBytes + s.DirectoryBytes + s.LogBytes + s.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
	}
	n += c.Primes.Stats().TotalBytes
	if v := p.context; v != nil {
		s := v.Stats()
		n += s.HeaderBytes + s.BlockBytes + s.ReceiptBytes + s.SourceFrameBytes + s.SourceSetBytes + s.ReplacementBytes + s.PageBytes + s.DirectoryBytes + s.FormatBytes + gparamsVectorGraph(s.Text)
	}
	if p.catalog != nil {
		n += catalogGraph(p.catalog)
	}
	if v := p.parameters; v != nil {
		n += v.OwnerBytes
		seen := map[*GlobalParams]bool{}
		add := func(g *GlobalParams) {
			if seen[g] {
				t.Fatal("duplicate gparams child owner")
			}
			seen[g] = true
			n += gparamsGraph(g)
			if d := g.constructorPendingDescriptor; d != nil {
				n += d.OwnerBytes + ParamDescriptorEntryBytes(len(d.Entries)) + ParamDescriptorNameBytes(cap(d.Names))
				if d.Texts != nil {
					n += gparamsVectorGraph(d.Texts.Stats())
				}
			}
		}
		if v.Current != nil {
			add(v.Current)
		}
		for g := v.Superseded; g != nil; g = g.RetainedNext {
			add(g)
		}
		for g := v.Failed; g != nil; g = g.RetainedNext {
			add(g)
		}
	}
	if n != c.Budget.Used() {
		t.Fatal("once-only explicit process/borrowed/retained cold graph", n, c.Budget.Used())
	}
}
