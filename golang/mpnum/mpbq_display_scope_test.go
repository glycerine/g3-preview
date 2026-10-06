// Licensed under the MIT license.
// Source mpbq.cpp:426–515 raw mpz locals.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
	"testing"
)

func binaryDisplayTestOwner(t *testing.T, z *IntegerManager) (*BinaryRationalManager, error) {
	t.Helper()
	s, e := NewRationalService(serviceConfig(z.budget))
	if e != nil {
		return nil, e
	}
	t.Cleanup(func() { serviceMust(t, s.Close()) })
	return NewBinaryRationalManager(z, BinaryRationalConfig{Diagnostics: s})
}
func binaryDisplayDiagnosticBytes(m *BinaryRationalManager) uint64 {
	s := m.diagnostics
	if s == nil || s.closed {
		return 0
	}
	return integerActualOwnedBytes(s.manager.z) + s.manager.headerBytes + s.headerBytes + s.cacheBytes
}

func TestBinaryDecimalErrorDoesNotExecuteSourceEndLabel(t *testing.T) {
	z, b := mpzPublicationDirectOwner(t)
	m, e := binaryDisplayTestOwner(t, z)
	serviceMust(t, e)
	defer m.Close()
	var a BinaryRational
	serviceMust(t, z.SetUint64(&a.num, 1<<63))
	a.k = 128
	defer z.Del(&a.num)
	// Prime genuine retained cold locator/descriptor capacity before observing
	// Copy's physical allocation followed by Power's allocation refusal.
	var prime Int
	serviceMust(t, z.Copy(&prime, &a.num))
	serviceMust(t, z.Del(&prime))
	baseline := z.cells.Stats().Live
	b.FailAfter(1)
	var dst [32]byte
	_, e = m.FormatDecimalInto(&a, 8, dst[:])
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal("actual refusal", e)
	}
	if z.cells.Stats().Live != baseline+1 {
		t.Fatal("raw v1 must survive exception before end label", baseline, z.cells.Stats().Live)
	}
	// The completed Copy's live root is retained before fault disable/cleanup.
	state := m.DisplayState()
	if !state.Pending || state.SourceEnd || state.OriginalCause != e || state.CleanupCause != nil || !validNumericRef(z.cells, m.display.values[1].cell) {
		t.Fatal(state)
	}
	binaryDisplayAudit(t, m, &a)
	b.FailAfter(-1)
	serviceMust(t, m.CleanupDisplay())
	if m.DisplayState().Pending || !m.DisplayState().OwningAbort || m.DisplayState().SourceEnd {
		t.Fatal("owning abort is not source end", m.DisplayState())
	}
	if z.cells.Stats().Live != baseline {
		t.Fatal("real local cleanup", baseline, z.cells.Stats().Live)
	}
}

func TestBinaryDisplayScopeAPIAndGuards(t *testing.T) {
	m, _ := binaryManager(t)
	s := m.DisplayState()
	if s.Pending || s.SourceEnd || s.SourceStage != 0 {
		t.Fatal(s)
	}
	serviceMust(t, m.CleanupDisplay())
	copied := *m
	if copied.CleanupDisplay() != ErrBinaryOwner || copied.CloseChecked() != ErrBinaryOwner {
		t.Fatal("copied cold owner")
	}
}

func binaryDisplayAudit(t *testing.T, m *BinaryRationalManager, public ...*BinaryRational) {
	t.Helper()
	roots := []*Int{&m.tmp, &m.tmp2, &m.addMulTmp.num, &m.selectIntTmp1, &m.selectIntTmp2, &m.selectSmallTmp, &m.selectSmallTmp1.num, &m.selectSmallTmp2.num, &m.divTmp1, &m.divTmp2, &m.divTmp3}
	for i := range m.display.values {
		roots = append(roots, &m.display.values[i])
	}
	for _, a := range public {
		roots = append(roots, &a.num)
	}
	integerPhysicalAudit(t, m.z, roots...)
	if m.z.budget.Used() != integerActualOwnedBytes(m.z)+m.headerBytes+binaryDisplayDiagnosticBytes(m) {
		t.Fatal("actual scope header and complete physical graph", m.z.budget.Used(), integerActualOwnedBytes(m.z), m.headerBytes)
	}
	if m.headerBytes != uint64(reflect.TypeOf(BinaryRationalManager{}).Size()) {
		t.Fatal("scope header charge")
	}
}

func TestBinaryRawDisplayAllFaultPrefixesAndOwningAbort(t *testing.T) {
	for _, interval := range []bool{false, true} {
		success, retained := false, false
		for fault := int64(0); fault < 160; fault++ {
			z, b := mpzPublicationDirectOwner(t)
			m, e := binaryDisplayTestOwner(t, z)
			serviceMust(t, e)
			var a, c BinaryRational
			serviceMust(t, z.SetUint64(&a.num, 1<<63))
			a.k = 128
			serviceMust(t, z.Copy(&c.num, &a.num))
			c.k = 129
			aRef, cRef := a.num.cell, c.num.cell
			b.FailAfter(fault)
			dst := [32]byte{}
			for i := range dst {
				dst[i] = '!'
			}
			var n int
			if interval {
				n, e = m.FormatIntervalDecimalInto(&a, &c, 3, dst[:])
			} else {
				n, e = m.FormatDecimalInto(&a, 3, dst[:])
			}
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal(interval, fault, e)
				}
				for _, v := range dst {
					if v != '!' {
						t.Fatal("premature caller bytes", interval, fault)
					}
				}
				state := m.DisplayState()
				if !state.Pending || state.SourceEnd || state.OriginalCause != e {
					t.Fatal("exception source prefix", interval, fault, state)
				}
				for _, v := range m.display.values {
					retained = retained || validNumericRef(z.cells, v.cell)
				}
			} else {
				success = true
				if string(dst[:n]) != "0.000?" || m.DisplayState().Pending || !m.DisplayState().SourceEnd || m.DisplayState().SourceStage != m.display.count() {
					t.Fatal(interval, fault, n, string(dst[:n]), m.DisplayState())
				}
			}
			if a.num.cell != aRef || c.num.cell != cRef || a.k != 128 || c.k != 129 || z.digit(aRef, 1) != 0x80000000 || z.digit(cRef, 1) != 0x80000000 {
				t.Fatal("actual input roots/limbs changed", interval, fault)
			}
			binaryDisplayAudit(t, m, &a, &c) // before fault disable or any cleanup
			b.FailAfter(-1)
			serviceMust(t, m.CleanupDisplay())
			serviceMust(t, z.Del(&a.num))
			serviceMust(t, z.Del(&c.num))
			serviceMust(t, m.CloseChecked())
			serviceMust(t, z.CloseChecked())
			serviceMust(t, m.diagnostics.Close())
			if b.Used() != 0 {
				t.Fatal("owning cleanup whole graph", interval, fault, b.Used())
			}
			if success {
				t.Logf("interval=%v first complete source invocation at reservation ordinal %d", interval, fault)
				break
			}
		}
		if !success || !retained {
			t.Fatal("complete real prefix campaign", interval, success, retained)
		}
	}
}

func TestBinarySourceEndLabelsAndInlineWarm(t *testing.T) {
	z, b := mpzPublicationOwner(t)
	m, e := binaryDisplayTestOwner(t, z)
	serviceMust(t, e)
	a, c := NewBinaryIntPower(3, 3), NewBinaryIntPower(3, 3)
	var dst [32]byte
	n, e := m.FormatDecimalInto(&a, 8, dst[:])
	serviceMust(t, e)
	if string(dst[:n]) != "0.375" || m.DisplayState().SourceStage != 3 {
		t.Fatal(n, string(dst[:n]), m.DisplayState())
	}
	n, e = m.FormatIntervalDecimalInto(&a, &c, 8, dst[:])
	serviceMust(t, e)
	if string(dst[:n]) != "0.375" || m.DisplayState().SourceStage != 6 {
		t.Fatal(n, string(dst[:n]), m.DisplayState())
	}
	before := numericWarmProfileSnapshot(t, z, b)
	b.FailAfter(0)
	var problem error
	wrong := false
	goalloc := testing.AllocsPerRun(100, func() {
		n, e := m.FormatIntervalDecimalInto(&a, &c, 8, dst[:])
		if e != nil {
			problem = e
		}
		if n != 5 || string(dst[:n]) != "0.375" {
			wrong = true
		}
	})
	if problem != nil || wrong {
		t.Fatal(problem, wrong)
	}
	numericWarmProfileAudit(t, z, b, before, goalloc, false)
	binaryDisplayAudit(t, m, &a, &c)
	b.FailAfter(-1)
	serviceMust(t, m.CloseChecked())
}

// This public-runtime recipe deliberately reaches the guarded signed-overflow
// boundary through two real oversized realloc counter-before-physical prefixes.
// It is not normal exhaustion or an allocation-limit deallocation refusal.
func TestBinarySourceEndLabelGuardedPhysicalFreeContinuation(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = r
	cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	z, e := NewIntegerManager(cfg)
	serviceMust(t, e)
	m, e := binaryDisplayTestOwner(t, z)
	serviceMust(t, e)
	serviceMust(t, m.beginBinaryDisplay(true))
	two := NewInt(2)
	serviceMust(t, z.Power(&two, 1048576, &m.display.values[0]))
	for i := 1; i < 6; i++ {
		serviceMust(t, z.SetUint64(&m.display.values[i], uint64(1)<<63|uint64(i)))
	}
	refs := [6]numericCellRef{}
	for i := range refs {
		refs[i] = m.display.values[i].cell
	}
	raw := z.cell(refs[0]).backing.raw
	size, e := r.Size(raw)
	serviceMust(t, e)
	if size <= 100000 {
		t.Fatal("real source local extent", size)
	}
	first, e := r.Allocate(8)
	serviceMust(t, e)
	second, e := r.Allocate(8)
	serviceMust(t, e)
	before := r.State()
	delta := int64(-1000000)
	request := uint64(16) + uint64(delta) - 8
	old := first
	if e = r.Reallocate(&first, request); e != arena.ErrBounds || first != old {
		t.Fatal("first old caller retained", e)
	}
	prefix := r.State()
	if prefix.GlobalSize != before.GlobalSize || prefix.LocalSize != before.LocalSize+delta || prefix.LocalCount != before.LocalCount+1 {
		t.Fatal(before, prefix)
	}
	serviceMust(t, r.Deallocate(&first))
	negative := r.State()
	if negative.GlobalSize >= 0 || negative.LocalSize != 0 {
		t.Fatal(negative)
	}
	target := int64(math.MinInt64) + 1000 - negative.GlobalSize + 16
	request = uint64(target) + 16 - 8
	old = second
	if e = r.Reallocate(&second, request); e != arena.ErrBounds || second != old {
		t.Fatal("second old caller retained", e)
	}
	serviceMust(t, r.Deallocate(&second))
	near := r.State()
	if near.GlobalSize != math.MinInt64+1000 || near.LocalSize != 0 {
		t.Fatal(near)
	}
	binaryDisplayAudit(t, m)
	length := uint32(7)
	var result error
	m.finishBinaryDisplay(&length, &result)
	if !errors.Is(result, arena.ErrBounds) || length != 0 {
		t.Fatal(result, length)
	}
	state := m.DisplayState()
	stopped := r.State()
	if !state.Pending || !state.SourceEnd || state.SourceStage != 0 || state.Guard || state.CleanupCause != arena.ErrBounds || state.OriginalCause != nil {
		t.Fatal(state)
	}
	if stopped.GlobalSize != near.GlobalSize || stopped.LocalSize != -int64(size+8) || stopped.GlobalCount != near.GlobalCount {
		t.Fatal("actual physical free before sync guard", near, stopped)
	}
	if _, e = r.Size(raw); e != memory.ErrRuntimeOwner {
		t.Fatal("source physical release", e)
	}
	if !z.cell(refs[0]).released || !z.cell(refs[0]).cleanupPending {
		t.Fatal("real descriptor release authority")
	}
	for i := 1; i < 6; i++ {
		if !validNumericRef(z.cells, refs[i]) || m.display.values[i].cell != refs[i] {
			t.Fatal("older source local lost", i)
		}
	}
	binaryDisplayAudit(t, m)

	// Each remaining actual free can independently reach the same guarded sync;
	// the retained cursor acknowledges its previous consumed free before advancing.
	for stage := uint8(1); stage < 6; stage++ {
		raw := z.cell(refs[stage]).backing.raw
		extent, e := r.Size(raw)
		serviceMust(t, e)
		before := r.State()
		charge := m.headerBytes
		if e = m.CloseChecked(); e != arena.ErrBounds {
			t.Fatal("real next source free", stage, e)
		}
		state := m.DisplayState()
		after := r.State()
		if state.SourceStage != stage || !state.Pending || state.Guard || state.OwningAbort || !m.closing || m.headerBytes != charge {
			t.Fatal("retained source and owner cursor", stage, state, m.headerBytes, charge)
		}
		if after.GlobalSize != before.GlobalSize || after.LocalSize != before.LocalSize-int64(extent+8) || after.GlobalCount != before.GlobalCount || after.LocalCount != before.LocalCount {
			t.Fatal("single actual physical free prefix", stage, before, after)
		}
		if _, e = r.Size(raw); e != memory.ErrRuntimeOwner {
			t.Fatal("real freed token", stage, e)
		}
		for i := uint8(0); i < stage; i++ {
			if m.display.values[i].cell != 0 {
				t.Fatal("acknowledged source del", stage, i)
			}
		}
		for i := stage + 1; i < 6; i++ {
			if m.display.values[i].cell != refs[i] || z.cell(refs[i]).released {
				t.Fatal("older remaining local", stage, i)
			}
		}
		binaryDisplayAudit(t, m)
		zero := NewBinaryInt(0)
		if _, e = m.DecimalString(&zero, 1); e != arena.ErrClosed {
			t.Fatal("ordinary operation while owning close", e)
		}
	}
	before = r.State()
	serviceMust(t, m.CloseChecked())
	if r.State() != before || m.DisplayState().SourceStage != 6 || m.DisplayState().Pending || m.headerBytes != 0 {
		t.Fatal("final free acknowledgement", m.DisplayState(), r.State(), before)
	}
	// Integer owning cleanup also preserves each real postphysical refusal. It
	// frees only its own remaining source roots, with no replay/repair of counters.
	known := [6]numericCellRef{z.tmp[0], z.tmp[1], z.arg[0], z.arg[1], z.intMin.cell, z.two64.cell}
	done := false
	for attempt := 0; attempt < 32; attempt++ {
		e = z.CloseChecked()
		if e == nil {
			done = true
			break
		}
		if e != z || z.CleanupCause() != arena.ErrBounds || !errors.Is(e, arena.ErrBounds) {
			t.Fatal("actual retained child owner/cause", e, z.CleanupCause())
		}
		// The owning scanner acknowledges descriptors before clearing cached private
		// pointer fields at final closure. Audit the actual surviving subset of the
		// six already named constructor roots, not their now-dangling cache fields.
		alive, physical := uint32(0), uint32(0)
		for _, ref := range known {
			if !validNumericRef(z.cells, ref) {
				continue
			}
			alive++
			c := z.cell(ref)
			if !c.released {
				physical++
				if _, e := z.allocator.ObjectExtent(c.backing.object); e != nil {
					t.Fatal("real surviving private root", e)
				}
			}
		}
		if z.cells.Stats().Live != alive || r.Stats().LiveBlocks != physical || z.allocator.Stats().DirectObjects != physical {
			t.Fatal("exact owning remaining physical subset", alive, physical, z.cells.Stats(), r.Stats(), z.allocator.Stats())
		}
		if b.Used() != integerActualOwnedBytes(z)+binaryDisplayDiagnosticBytes(m) {
			t.Fatal("retained actual child graph")
		}
	}
	if !done {
		t.Fatal("bounded owning cleanup")
	}
	serviceMust(t, r.Close())
	serviceMust(t, m.diagnostics.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
