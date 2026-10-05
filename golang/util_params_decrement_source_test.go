// Source: src/util/params.cpp:325-339,510-513; src/util/memory_manager.cpp.
// Author: Leonardo de Moura (2011-05-09).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
package g3

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/mpnum"
)

// Public reallocs reach the source LP64 signed-overflow guarded domain. They
// do not simulate OOM, mutate private counters, or inject a deallocation error.
func paramsDecTLSNearMinimum(t *testing.T, r *memory.Runtime) {
	t.Helper()
	a, e := r.Allocate(8)
	paramsMust(t, e)
	b, e := r.Allocate(8)
	paramsMust(t, e)
	delta := int64(-1000000)
	if e = r.Reallocate(&a, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	paramsMust(t, r.Deallocate(&a))
	s := r.State()
	if s.LocalSize != 0 || s.GlobalSize >= 0 {
		t.Fatal("first actual synchronization", s)
	}
	delta = math.MinInt64 + 1000 - s.GlobalSize + 16
	if e = r.Reallocate(&b, uint64(delta)+16-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	paramsMust(t, r.Deallocate(&b))
	if s = r.State(); s.LocalSize != 0 || s.GlobalSize != math.MinInt64+1000 {
		t.Fatal("actual source TLS lane", s)
	}
}

func paramsDecFinish(t *testing.T, fn func() error) {
	t.Helper()
	for i := 0; i < 32; i++ {
		e := fn()
		if e == nil {
			return
		}
		if !errors.Is(e, arena.ErrBounds) {
			t.Fatal("actual owning cleanup", e)
		}
	}
	t.Fatal("actual owning cleanup failed to advance")
}

func TestParamsReferenceRepeatedDECSourceAssertionAfterActualSetLimbsLeafStop(t *testing.T) {
	bn := arena.NewBudget(1 << 26)
	bc := arena.NewBudget(1 << 24)
	bs := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: bn, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	q, e := mpnum.NewRationalService(mpnum.RationalServiceConfig{Integers: mpnum.IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := NewSymbolManager(SymbolConfig{Budget: bs, PageSize: 64, DescriptorPageSize: 2})
	paramsMust(t, e)
	c, e := NewParamsContext(sy, q, ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
	paramsMust(t, e)
	m, e := q.NewUPolynomialManager(mpnum.UPolynomialConfig{Checkpoint: func() bool { panic("pure params cell route checkpoint") }, Cooperation: func() error { panic("pure params cell route cooperation") }})
	paramsMust(t, e)
	t.Cleanup(func() {
		paramsDecFinish(t, c.Close)
		paramsDecFinish(t, m.Close)
		paramsDecFinish(t, q.Close)
		paramsDecFinish(t, r.Close)
		paramsMust(t, sy.Close())
		if bn.Used() != 0 || bc.Used() != 0 || bs.Used() != 0 {
			t.Error("remaining named owners", bn.Used(), bc.Used(), bs.Used())
		}
	})
	v, e := c.NewValue()
	paramsMust(t, e)
	k, e := sy.Intern("actual-large-numeral")
	paramsMust(t, e)
	// Separate efficient fixture: public source digit setter and scoped copy.
	// Historical BitwiseNot fixture and its 252-second RED remain unchanged.
	scoped, e := mpnum.NewScopedInt(m.IntegerManager())
	paramsMust(t, e)
	num, e := scoped.Get()
	paramsMust(t, e)
	digits := make([]uint32, 32768)
	digits[0], digits[len(digits)-1] = 17, 1
	paramsMust(t, m.IntegerManager().SetLimbs(num, digits))
	big, e := q.NewFromScopedInt(scoped)
	paramsMust(t, e)
	paramsMust(t, scoped.Close())
	paramsMust(t, c.SetRat(&v, k, &big))
	paramsMust(t, q.Del(&big))
	block := &c.blocks[v.slot-1]
	if block.refs != 1 || block.size != 1 {
		t.Fatal("actual source reference", block.refs, block.size)
	}
	extras := upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}}}
	before := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	paramsDecTLSNearMinimum(t, r)
	e = c.Del(&v)
	if !errors.Is(e, arena.ErrBounds) || block.refs != 0 || !block.live || v.slot == 0 {
		t.Fatal("real decrement before stopped numeral destruction", e, block.refs, block.live, v)
	}
	after := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if after.Named != before.Named || after.Objects+1 != before.Objects {
		t.Fatal("retained descriptor and actually freed raw object", before, after)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	if !reflect.ValueOf(&c.entry(block, 0).rat).Elem().FieldByName("raw").FieldByName("num").FieldByName("cell").IsValid() {
		t.Fatal("named retained entry")
	}
	// Invalid repeat is source SASSERT(m_ref_count>0) before decrement in debug.
	// Release erases the assertion and literally wraps the unsigned count.
	again := c.Del(&v)
	if globalParamsSourceDebug {
		if !errors.Is(again, ErrParamsPrecondition) || block.refs != 0 {
			t.Fatal("source DEBUG assertion before repeated decrement", again, block.refs)
		}
	} else if again != nil || block.refs != math.MaxUint32 {
		t.Fatal("literal release unsigned decrement", again, block.refs)
	}
}

func TestParamsReferenceCleanupDecrementActualLeafReceipt(t *testing.T) {
	bn := arena.NewBudget(1 << 26)
	bc := arena.NewBudget(1 << 24)
	bs := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: bn, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	q, e := mpnum.NewRationalService(mpnum.RationalServiceConfig{Integers: mpnum.IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := NewSymbolManager(SymbolConfig{Budget: bs, PageSize: 64, DescriptorPageSize: 2})
	paramsMust(t, e)
	c, e := NewParamsContext(sy, q, ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
	paramsMust(t, e)
	m, e := q.NewUPolynomialManager(mpnum.UPolynomialConfig{Checkpoint: func() bool { panic("pure params cell route checkpoint") }, Cooperation: func() error { panic("pure params cell route cooperation") }})
	paramsMust(t, e)
	t.Cleanup(func() {
		bn.FailAfter(-1)
		bc.FailAfter(-1)
		paramsDecFinish(t, c.Close)
		paramsDecFinish(t, m.Close)
		paramsDecFinish(t, q.Close)
		paramsDecFinish(t, r.Close)
		paramsMust(t, sy.Close())
		if bn.Used() != 0 || bc.Used() != 0 || bs.Used() != 0 {
			t.Error("remaining named owners", bn.Used(), bc.Used(), bs.Used())
		}
	})
	v, e := c.NewValue()
	paramsMust(t, e)
	k, e := sy.Intern("actual-large-numeral")
	paramsMust(t, e)
	// Separate efficient fixture: public source digit setter and scoped copy.
	// Historical BitwiseNot fixture and its 252-second RED remain unchanged.
	scoped, e := mpnum.NewScopedInt(m.IntegerManager())
	paramsMust(t, e)
	num, e := scoped.Get()
	paramsMust(t, e)
	digits := make([]uint32, 32768)
	digits[0], digits[len(digits)-1] = 17, 1
	paramsMust(t, m.IntegerManager().SetLimbs(num, digits))
	big, e := q.NewFromScopedInt(scoped)
	paramsMust(t, e)
	paramsMust(t, scoped.Close())
	prefix, e := sy.Intern("completed-prefix")
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&v, prefix, 42))
	paramsMust(t, c.SetRat(&v, k, &big))
	paramsMust(t, q.Del(&big))
	block := &c.blocks[v.slot-1]
	if block.refs != 1 || block.size != 2 {
		t.Fatal("actual source reference", block.refs, block.size)
	}
	extras := upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}}}
	before := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	other, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&other, k, 7))
	block = &c.blocks[v.slot-1] // actual directory may have relocated
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	bc.FailAfter(0)
	e = c.Del(&v)
	if !errors.Is(e, arena.ErrBounds) || block.refs != 0 || !block.live || v.slot == 0 {
		t.Fatal("real decrement before stopped numeral destruction", e, block.refs, block.live, v)
	}
	after := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if after.Named != before.Named || after.Objects+1 != before.Objects {
		t.Fatal("retained descriptor and actually freed raw object", before, after)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	if !reflect.ValueOf(&c.entry(block, 1).rat).Elem().FieldByName("raw").FieldByName("num").FieldByName("cell").IsValid() {
		t.Fatal("named retained entry")
	}
	if block.resetCursor != 1 || block.resetStage != 0 || *c.entry(block, 0) != (paramEntry{}) || !errors.Is(c.decrementReceipts[v.slot-1].cause, e) {
		t.Fatal("completed source entry prefix", *block)
	}
	savedEntry := *c.entry(block, 1)
	savedReceipt := c.decrementReceipts[v.slot-1]
	for _, operation := range []func() error{
		func() error { return c.SetUInt(&v, k, 9) },
		func() error { return c.Reset(&v) },
		func() error { return c.ResetKey(&v, k) },
		func() error { return c.Assign(&v, &other) },
		func() error { return c.Assign(&other, &v) },
		func() error { return c.Copy(&v, &other) },
		func() error { return c.Copy(&other, &v) },
	} {
		if e := operation(); !errors.Is(e, ErrParamsPrecondition) {
			t.Fatal("stopped owner replacement guard", e)
		}
		if *c.entry(block, 1) != savedEntry || c.decrementReceipts[v.slot-1] != savedReceipt || block.refs != 0 || block.resetCursor != 1 {
			t.Fatal("retained exact leaf changed")
		}
		upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
		paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	}
	// Wrong concrete owning value cannot acknowledge this stopped block.
	if e := c.CleanupDecrement(&other); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("unrelated owning receipt", e)
	}
	paramsMust(t, c.Del(&other))
	paramsMust(t, c.CleanupDecrement(&v))
	if v != (ParamsRef{}) || block.live || block.refs != 0 || block.size != 0 {
		t.Fatal("exact owning continuation tombstone", v, *block)
	}
	upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}})
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
}

func TestParamsReferenceIndependentStoppedBlocksAndReplacementGeneration(t *testing.T) {
	bn := arena.NewBudget(1 << 26)
	bc := arena.NewBudget(1 << 24)
	bs := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: bn, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	q, e := mpnum.NewRationalService(mpnum.RationalServiceConfig{Integers: mpnum.IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := NewSymbolManager(SymbolConfig{Budget: bs, PageSize: 64, DescriptorPageSize: 2})
	paramsMust(t, e)
	c, e := NewParamsContext(sy, q, ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
	paramsMust(t, e)
	m, e := q.NewUPolynomialManager(mpnum.UPolynomialConfig{Checkpoint: func() bool { panic("pure params cell route checkpoint") }, Cooperation: func() error { panic("pure params cell route cooperation") }})
	paramsMust(t, e)
	t.Cleanup(func() {
		bn.FailAfter(-1)
		bc.FailAfter(-1)
		paramsDecFinish(t, c.Close)
		paramsDecFinish(t, m.Close)
		paramsDecFinish(t, q.Close)
		paramsDecFinish(t, r.Close)
		paramsMust(t, sy.Close())
		if bn.Used() != 0 || bc.Used() != 0 || bs.Used() != 0 {
			t.Error("remaining named owners", bn.Used(), bc.Used(), bs.Used())
		}
	})
	v, e := c.NewValue()
	paramsMust(t, e)
	k, e := sy.Intern("actual-large-numeral")
	paramsMust(t, e)
	// Separate efficient fixture: public source digit setter and scoped copy.
	// Historical BitwiseNot fixture and its 252-second RED remain unchanged.
	scoped, e := mpnum.NewScopedInt(m.IntegerManager())
	paramsMust(t, e)
	num, e := scoped.Get()
	paramsMust(t, e)
	digits := make([]uint32, 32768)
	digits[0], digits[len(digits)-1] = 17, 1
	paramsMust(t, m.IntegerManager().SetLimbs(num, digits))
	big, e := q.NewFromScopedInt(scoped)
	paramsMust(t, e)
	paramsMust(t, scoped.Close())
	paramsMust(t, c.SetRat(&v, k, &big))
	v2, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetRat(&v2, k, &big))
	paramsMust(t, q.Del(&big))
	block := &c.blocks[v.slot-1]
	if block.refs != 1 || block.size != 1 {
		t.Fatal("actual source reference", block.refs, block.size)
	}
	extras := upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v}, {c, &v2}}}
	before := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	other, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&other, k, 7))
	block = &c.blocks[v.slot-1] // actual directory may have relocated
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	bc.FailAfter(0)
	e = c.Del(&v)
	if !errors.Is(e, arena.ErrBounds) || block.refs != 0 || !block.live || v.slot == 0 {
		t.Fatal("real decrement before stopped numeral destruction", e, block.refs, block.live, v)
	}
	after := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if after.Named != before.Named || after.Objects+1 != before.Objects {
		t.Fatal("retained descriptor and actually freed raw object", before, after)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	if !reflect.ValueOf(&c.entry(block, 0).rat).Elem().FieldByName("raw").FieldByName("num").FieldByName("cell").IsValid() {
		t.Fatal("named retained entry")
	}
	// A second real physical release reaches the same synchronization guard.
	e = c.Del(&v2)
	b2 := &c.blocks[v2.slot-1]
	if !errors.Is(e, arena.ErrBounds) || b2.refs != 0 || !b2.resetActive || !block.resetActive {
		t.Fatal("two distinct stopped source owners", e, *block, *b2)
	}
	if block.resetCursor != 0 || block.resetStage != 0 || b2.resetCursor != 0 || b2.resetStage != 0 || !errors.Is(c.decrementReceipts[v.slot-1].cause, arena.ErrBounds) || !errors.Is(c.decrementReceipts[v2.slot-1].cause, arena.ErrBounds) {
		t.Fatal("independent exact leaf receipts")
	}
	second := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if second.Named != before.Named || second.Objects+2 != before.Objects {
		t.Fatal(before, second)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	paramsMust(t, c.Del(&other))
	oldSlot, oldGeneration := v.slot, v.generation
	paramsMust(t, c.CleanupDecrement(&v))
	remaining := upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &v2}}}
	upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, remaining)
	if !b2.resetActive || b2.refs != 0 {
		t.Fatal("unrelated stopped receipt consumed")
	}
	replacement, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(&replacement, k, 9))
	if replacement.slot != oldSlot || replacement.generation != oldGeneration+1 || c.decrementReceipts[oldSlot-1].cause != nil || c.decrementReceipts[oldSlot-1].generation != 0 {
		t.Fatal("fresh replacement generation", replacement)
	}
	paramsMust(t, c.Del(&replacement))
	paramsMust(t, c.CleanupDecrement(&v2))
	upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}})
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
}

// Cold graph uses the actual named owners and their typed backing capacities.
// Source allocation counters remain separate from this physical Go charge.
func paramsDecAuditColdGraph(t *testing.T, q *mpnum.RationalService, m *mpnum.UPolynomialManager, r *memory.Runtime, bn *arena.Budget, c *ParamsContext, bc *arena.Budget, sy *SymbolManager, bs *arena.Budget) {
	t.Helper()
	qv := reflect.ValueOf(q).Elem()
	qm := qv.FieldByName("manager").Elem()
	z := reflect.ValueOf(m.IntegerManager()).Elem()
	n := r.Stats().TotalBytes + m.Stats().TotalBytes + qv.FieldByName("headerBytes").Uint() + qv.FieldByName("cacheBytes").Uint() + qm.FieldByName("headerBytes").Uint() + z.FieldByName("headerBytes").Uint()
	a := z.FieldByName("allocator").Elem()
	locations := a.FieldByName("locations")
	n += uint64(a.Type().Size()) + uint64(locations.Cap())*uint64(locations.Type().Elem().Size())
	slabBytes := func(s reflect.Value) uint64 {
		pages := s.FieldByName("pages")
		bytes := uint64(s.Type().Size()) + uint64(pages.Cap())*uint64(pages.Type().Elem().Size())
		for i := 0; i < pages.Len(); i++ {
			page := pages.Index(i)
			for _, f := range []string{"records", "meta"} {
				values := page.FieldByName(f)
				bytes += uint64(values.Len()) * uint64(values.Type().Elem().Size())
			}
		}
		return bytes
	}
	n += slabBytes(z.FieldByName("cells").Elem())
	words := z.FieldByName("words").Elem()
	n += uint64(words.Type().Size()) + slabBytes(words.FieldByName("descriptors").Elem()) + words.FieldByName("directoryBytes").Uint()
	pages := words.FieldByName("pages")
	for i := 0; i < pages.Len(); i++ {
		page := pages.Index(i)
		n += uint64(page.FieldByName("data").Len())*4 + uint64(page.FieldByName("next").Len())*4
	}
	nat := z.FieldByName("natural").Elem()
	n += nat.FieldByName("headerBytes").Uint()
	if !nat.FieldByName("text").IsNil() {
		t.Fatal("fixture reached undeclared Natural text owner")
	}
	if n != bn.Used() {
		t.Fatal("once-only named numeric/provider graph", n, bn.Used())
	}
	cs := c.Stats()
	cn := cs.HeaderBytes + cs.BlockBytes + cs.ReceiptBytes + cs.PageBytes + cs.DirectoryBytes + cs.FormatBytes + gparamsVectorGraph(cs.Text)
	if cn != bc.Used() {
		t.Fatal("actual ParamsContext owner graph", cn, bc.Used())
	}
	s := sy.Stats()
	d := s.Region.Descriptors
	sn := s.OwnerBytes + s.Table.OwnerBytes + s.Table.TableBytes + s.Region.OwnerBytes + s.Region.PayloadBytes + s.Region.PageHeaderBytes + s.Region.DirectoryBytes + s.Region.LogBytes + s.Region.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
	if sn != bs.Used() {
		t.Fatal("actual symbol provider graph", sn, bs.Used())
	}
}
