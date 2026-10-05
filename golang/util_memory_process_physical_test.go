// Source: src/util/memory_manager.cpp:89-113,153-160; src/util/gparams.cpp:81-105.
// Source author: Nikolaj Bjorner (2007-07-24); Leonardo de Moura (2012-11-29).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/mpnum"
	"reflect"
	"testing"
)

func memoryProcessNumericalBytes(t *testing.T, g *RationalGlobalService, r *memory.Runtime, anchors ...*mpnum.UPolynomialManager) uint64 {
	t.Helper()
	n := r.Stats().TotalBytes + g.header
	sv := reflect.ValueOf(g.service).Elem()
	n += sv.FieldByName("headerBytes").Uint() + sv.FieldByName("cacheBytes").Uint()
	seen := map[uintptr]bool{}
	slab := func(v reflect.Value) uint64 {
		if v.IsNil() {
			return 0
		}
		s := v.Elem()
		out := s.FieldByName("directoryBytes").Uint()
		if !s.FieldByName("closed").Bool() {
			out += uint64(s.Type().Size())
		}
		pages := s.FieldByName("pages")
		for i := 0; i < pages.Len(); i++ {
			p := pages.Index(i)
			for _, f := range []string{"records", "meta"} {
				a := p.FieldByName(f)
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
		n += z.FieldByName("headerBytes").Uint()
		a := z.FieldByName("allocator")
		if !a.IsNil() {
			a = a.Elem()
			loc := a.FieldByName("locations")
			n += a.FieldByName("header").Uint() + uint64(loc.Cap())*uint64(loc.Type().Elem().Size())
		}
		n += slab(z.FieldByName("cells"))
		w := z.FieldByName("words")
		if !w.IsNil() {
			w = w.Elem()
			if !w.FieldByName("closed").Bool() {
				n += uint64(w.Type().Size())
			}
			n += w.FieldByName("directoryBytes").Uint() + slab(w.FieldByName("descriptors"))
			pages := w.FieldByName("pages")
			for i := 0; i < pages.Len(); i++ {
				p := pages.Index(i)
				n += uint64(p.FieldByName("data").Len())*4 + uint64(p.FieldByName("next").Len())*4
			}
		}
		nat := z.FieldByName("natural")
		if !nat.IsNil() {
			nat = nat.Elem()
			n += nat.FieldByName("headerBytes").Uint()
			if !nat.FieldByName("text").IsNil() {
				t.Fatal("undeclared Natural text owner")
			}
		}
	}
	addM := func(v reflect.Value) {
		if v.IsNil() {
			return
		}
		m := v.Elem()
		n += m.FieldByName("headerBytes").Uint()
		addZ(m.FieldByName("z"))
	}
	addM(sv.FieldByName("manager"))
	addZ(sv.FieldByName("initializationFailure"))
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
	for _, a := range anchors {
		n += a.Stats().TotalBytes
	}
	return n
}

func memoryProcessColdAudit(t *testing.T, p *MemoryProcess, anchors ...*mpnum.UPolynomialManager) {
	t.Helper()
	c := p.config
	n := p.header + memoryProcessNumericalBytes(t, c.Rationals, c.Runtime, anchors...)
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
		n += v.ownerBytes
		seen := map[*GlobalParams]bool{}
		add := func(g *GlobalParams) {
			if seen[g] {
				t.Fatal("duplicate gparams child owner")
			}
			seen[g] = true
			n += gparamsGraph(g)
			if d := g.constructorPendingDescriptor; d != nil {
				n += d.ownerBytes + pdEntryBytes(len(d.entries)) + pdNameBytes(cap(d.names))
				if d.texts != nil {
					n += gparamsVectorGraph(d.texts.Stats())
				}
			}
		}
		if v.current != nil {
			add(v.current)
		}
		for g := v.superseded; g != nil; g = g.retainedNext {
			add(g)
		}
		for g := v.failed; g != nil; g = g.retainedNext {
			add(g)
		}
	}
	if n != c.Budget.Used() {
		t.Fatal("once-only explicit process/borrowed/retained cold graph", n, c.Budget.Used())
	}
}
