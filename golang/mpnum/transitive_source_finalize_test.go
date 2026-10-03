// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// mpq.cpp28-34, rational.cpp75-83 and infinitesimal reverse member destruction.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func transitiveSourceOwner(t *testing.T, service bool, profile memory.SmallAllocatorProfile) (*RationalManager, *RationalService, *memory.Runtime, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	cfg := IntegerConfig{Memory: rt, Profile: profile, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}
	var m *RationalManager
	var s *RationalService
	if service {
		s, e = NewRationalService(RationalServiceConfig{Integers: cfg})
		if e == nil {
			m = s.manager
		}
	} else {
		m, e = NewRationalManager(RationalConfig{Integers: cfg})
	}
	if e != nil {
		t.Fatal(e)
	}
	return m, s, rt, b
}
func transitiveSourceRun(m *RationalManager, s *RationalService) error {
	if s != nil {
		return s.SourceFinalize()
	}
	return m.SourceFinalize()
}
func transitiveSourceClose(t *testing.T, m *RationalManager, s *RationalService, rt *memory.Runtime, b *arena.Budget) {
	t.Helper()
	var e error
	if s != nil {
		e = s.CloseChecked()
	} else {
		e = m.CloseChecked()
	}
	if e != nil {
		t.Fatal(e)
	}
	if rt.Stats().LiveBlocks != 0 || b.Used() != rt.Stats().TotalBytes {
		t.Fatal("owning cleanup physical/cold graph", rt.Stats(), b.Used())
	}
	if e = rt.CheckOpen(); e != nil {
		t.Fatal("borrowed runtime closed", e)
	}
	if e = rt.Close(); e != nil || b.Used() != 0 {
		t.Fatal("final borrowed runtime cleanup", e, b.Used())
	}
}
func transitiveSourceDigits(n int) []uint32 { d := make([]uint32, n); d[0], d[n-1] = 7, 1; return d }

func TestTransitiveSourceFinalizeCallerPhysicalLifetimes(t *testing.T) {
	for _, service := range []bool{false, true} {
		for _, direct := range []bool{false, true} {
			m, s, rt, b := transitiveSourceOwner(t, service, memory.SmallAllocatorProfile{PointerBits: 64})
			caller := NewRational()
			n := 6
			if direct {
				n = 60
			}
			if e := m.z.SetLimbs(&caller.num, transitiveSourceDigits(n)); e != nil {
				t.Fatal(e)
			}
			raw, offset := m.z.cell(caller.num.cell).backing.raw, m.z.cell(caller.num.cell).backing.offset
			if e := transitiveSourceRun(m, s); e != nil {
				t.Fatal(e)
			}
			if !m.sourceFinalized || !m.z.sourceFinalized || (s != nil && !s.sourceFinalized) {
				t.Fatal("child source finalizer omitted")
			}
			if _, e := m.String(&caller); !errors.Is(e, arena.ErrClosed) {
				t.Fatal("ordinary rational read after source destruction", e)
			}
			if s != nil {
				if _, e := s.NewValue(); !errors.Is(e, arena.ErrClosed) {
					t.Fatal("ordinary service use after source destruction", e)
				}
			}
			if direct {
				if rt.Stats().LiveBlocks != 1 || rt.Stats().PhysicalBytes != 256 {
					t.Fatal("caller direct source lifetime", rt.Stats())
				}
				if v, e := rt.ReadUint32(raw, offset+8); e != nil || v != 7 {
					t.Fatal(v, e)
				}
			} else {
				if rt.Stats().LiveBlocks != 0 {
					t.Fatal("pool source lifetime", rt.Stats())
				}
				if _, e := rt.ReadUint32(raw, offset+8); !errors.Is(e, memory.ErrRuntimeOwner) {
					t.Fatal("destroyed pooled bytes", e)
				}
			}
			transitiveSourceAudit(t, m, s, b, &caller.num, &caller.den)
			state, used := rt.State(), b.Used()
			if e := transitiveSourceRun(m, s); e != nil || rt.State() != state || b.Used() != used || m.headerBytes == 0 || (s != nil && s.headerBytes == 0) {
				t.Fatal("source retry/cold ownership", e)
			}
			transitiveSourceClose(t, m, s, rt, b)
		}
	}
}

func TestTransitiveSourceFinalizeActualDiagnosticPrefix(t *testing.T) {
	for _, service := range []bool{false, true} {
		for _, retry := range []bool{false, true} {
			m, s, rt, b := transitiveSourceOwner(t, service, memory.SmallAllocatorProfile{PointerBits: 64, Debug: true, Windows: true})
			caller := NewRational()
			if e := m.z.SetLimbs(&caller.num, transitiveSourceDigits(60)); e != nil {
				t.Fatal(e)
			}
			if e := transitiveSourceRun(m, s); !errors.Is(e, memory.ErrAllocatorIO) {
				t.Fatal("real post-pool-free diagnostic refusal", e)
			}
			if m.cleanupStage != 9 || m.sourceFinalized || m.z.sourceFinalizeStage != 7 || rt.Stats().LiveBlocks != 1 || m.z.natural != nil || m.headerBytes == 0 {
				t.Fatal("source diagnostic prefix lost")
			}
			if s != nil && (s.cleanupStage != 25 || s.sourceFinalized || s.cacheBytes != 0) {
				t.Fatal("service outer prefix lost")
			}
			transitiveSourceAudit(t, m, s, b, &caller.num, &caller.den)
			if retry {
				state, used := rt.State(), b.Used()
				if e := transitiveSourceRun(m, s); e != nil || rt.State() != state || b.Used() != used {
					t.Fatal("diagnostic replay", e)
				}
			}
			transitiveSourceClose(t, m, s, rt, b)
		}
	}
}

func TestRationalSourceFinalizeActualPrivateGuardOrder(t *testing.T) {
	for stop := 0; stop < 9; stop++ {
		m, s, rt, b := transitiveSourceOwner(t, false, memory.SmallAllocatorProfile{PointerBits: 64, Debug: true})
		roots := [9]*Int{&m.nTmp, &m.addTmp1, &m.addTmp2, &m.ltTmp1.num, &m.ltTmp1.den, &m.ltTmp2.num, &m.ltTmp2.den, &m.addMulTmp.num, &m.addMulTmp.den}
		// Reverse physical allocation order prevents descriptor scanning from
		// accidentally satisfying source destructor-order assertions.
		var raws [9]memory.Block
		for i := 8; i >= 0; i-- {
			if e := m.z.SetLimbs(roots[i], transitiveSourceDigits(60)); e != nil {
				t.Fatal(e)
			}
			raws[i] = m.z.cell(roots[i].cell).backing.raw
		}
		var input Int
		if e := m.z.SetLimbs(&input, transitiveSourceDigits(120)); e != nil {
			t.Fatal(e)
		}
		b.FailAfter(0)
		e := m.z.Copy(roots[stop], &input)
		if !numericPhysicalReservationRefusal(e) || roots[stop].cell != m.z.failedRef {
			t.Fatal("real private setter failure prefix", e)
		}
		b.FailAfter(-1)
		if e = m.SourceFinalize(); !errors.Is(e, arena.ErrInvalidSlot) {
			t.Fatal("invalid private capacity access", e)
		}
		if !m.sourceGuard || m.cleanupStage != uint8(stop) || m.sourceFinalized || m.z.sourceFinalizing {
			t.Fatal("undefined leaf silently completed", stop)
		}
		for i := range raws {
			_, e = rt.ReadUint32(raws[i], 8)
			if i <= stop {
				if !errors.Is(e, memory.ErrRuntimeOwner) {
					t.Fatal("prior source release missing", stop, i, e)
				}
			} else if e != nil {
				t.Fatal("later source action executed", stop, i, e)
			}
		}
		transitiveSourceAudit(t, m, s, b, &input)
		q := NewRational()
		if e = m.check(&q); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("ordinary read at outer source guard", e)
		}
		state, used := rt.State(), b.Used()
		if e = m.SourceFinalize(); !errors.Is(e, arena.ErrInvalidSlot) || rt.State() != state || b.Used() != used {
			t.Fatal("guard repeated source actions", e)
		}
		transitiveSourceClose(t, m, s, rt, b)
		if !m.sourceGuard || m.cleanupStage != uint8(stop) || m.sourceFinalized {
			t.Fatal("owning abort rewrote stopped source history")
		}
	}
}

func TestServiceSourceFinalizeActualCacheDenominatorGuard(t *testing.T) {
	m, s, rt, b := transitiveSourceOwner(t, true, memory.SmallAllocatorProfile{PointerBits: 64, Debug: true})
	if _, e := s.PowerOfTwo(1); e != nil {
		t.Fatal(e)
	}
	prior := [3]*Int{&s.cache[0].num, &s.cache[0].den, &s.cache[1].num}
	var raws [3]memory.Block
	for i, p := range prior {
		if e := m.z.SetLimbs(p, transitiveSourceDigits(60)); e != nil {
			t.Fatal(e)
		}
		raws[i] = m.z.cell(p.cell).backing.raw
	}
	b.FailAfter(0)
	e := m.z.Copy(&s.cache[1].den, &m.z.two64)
	if !numericPhysicalReservationRefusal(e) || s.cache[1].den.cell != m.z.failedRef {
		t.Fatal(e)
	}
	b.FailAfter(-1)
	if e = s.SourceFinalize(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	if !s.sourceGuard || s.cleanupStage != 18 || s.cleanupCacheIndex != 1 || s.cleanupComponent != 1 || s.cacheBytes == 0 || m.sourceFinalizing {
		t.Fatal("cache NUM/DEN source prefix")
	}
	for _, raw := range raws {
		if _, e = rt.ReadUint32(raw, 8); !errors.Is(e, memory.ErrRuntimeOwner) {
			t.Fatal("prior cache source release", e)
		}
	}
	transitiveSourceAudit(t, m, s, b)
	if _, e = s.NewValue(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("service read at cache guard", e)
	}
	transitiveSourceClose(t, m, s, rt, b)
	if !s.sourceGuard || s.cleanupStage != 18 || s.cleanupCacheIndex != 1 || s.cleanupComponent != 1 || s.sourceFinalized {
		t.Fatal("owning cache abort rewrote source history")
	}
}

func TestTransitiveSourceFinalizeActualChildScratchHole(t *testing.T) {
	for _, service := range []bool{false, true} {
		m, s, rt, b := transitiveSourceOwner(t, service, memory.SmallAllocatorProfile{PointerBits: 64, Debug: true})
		old := m.z.tmp[0]
		b.FailAfter(0)
		e := m.z.ensureTmp(0, 33)
		if !numericPhysicalReservationRefusal(e) || m.z.tmp[0] != 0 || validNumericRef(m.z.cells, old) {
			t.Fatal("real free-before-replacement scratch hole", e)
		}
		b.FailAfter(-1)
		if e = transitiveSourceRun(m, s); !errors.Is(e, arena.ErrInvalidSlot) {
			t.Fatal(e)
		}
		if !m.sourceGuard || m.cleanupStage != 9 || m.sourceFinalized || !m.z.sourceGuard || m.z.sourceFinalizeStage != 2 {
			t.Fatal("child source guard propagation")
		}
		if s != nil && (!s.sourceGuard || s.cleanupStage != 25 || s.sourceFinalized) {
			t.Fatal("service child source guard")
		}
		transitiveSourceAudit(t, m, s, b)
		transitiveSourceClose(t, m, s, rt, b)
		if m.z.tmp[0] != 0 || m.z.sourceFinalizeStage != 2 || m.z.sourceFinalized || m.sourceFinalized || (s != nil && s.sourceFinalized) {
			t.Fatal("owning cleanup repaired or falsely finalized child")
		}
	}
}

// Independent retained graph audit works both before child destruction and
// after pooled payloads have died while charged logical descriptors survive.
func transitiveSourceAudit(t *testing.T, m *RationalManager, s *RationalService, b *arena.Budget, public ...*Int) {
	t.Helper()
	z := m.z
	refs := map[numericCellRef]bool{}
	add := func(p *Int) {
		if p == nil || p.cell == 0 || p.cell == z.failedRef {
			return
		}
		if !validNumericRef(z.cells, p.cell) {
			t.Fatal("invalid named retained descriptor", p.cell)
		}
		refs[p.cell] = true
	}
	for i := range z.tmp {
		r := Int{cell: z.tmp[i]}
		add(&r)
		r.cell = z.arg[i]
		add(&r)
	}
	add(&z.two64)
	add(&z.intMin)
	for _, p := range []*Int{&m.nTmp, &m.addTmp1, &m.addTmp2, &m.ltTmp1.num, &m.ltTmp1.den, &m.ltTmp2.num, &m.ltTmp2.den, &m.addMulTmp.num, &m.addMulTmp.den} {
		add(p)
	}
	if s != nil {
		for _, r := range []*Rational{&s.infZero.first, &s.infZero.second, &s.infOne.first, &s.infOne.second, &s.infMinusOne.first, &s.infMinusOne.second, &s.infIntZero.first, &s.infIntOne.first, &s.infIntMinusOne.first, &s.zero, &s.one, &s.minusOne} {
			add(&r.num)
			add(&r.den)
		}
		for i := range s.cache {
			add(&s.cache[i].num)
			add(&s.cache[i].den)
		}
	}
	for _, p := range public {
		add(p)
	}
	for i := uint64(1); i <= uint64(z.cells.Stats().Slots); i++ {
		c, e := z.cells.Read(arena.Slot(i))
		if e == nil && (c.unpublished || c.cleanupPending) {
			p := Int{cell: makeNumericRef(arena.Slot(i), z.cells.state(arena.Slot(i)).generation)}
			add(&p)
		}
	}
	cs := z.cells.Stats()
	if uint32(len(refs)) != cs.Live {
		t.Fatal("unnamed source lifetime descriptor", len(refs), cs.Live)
	}
	direct := map[memory.Block]bool{}
	for ref := range refs {
		c := z.cell(ref)
		v, e := z.memory.ReadUint32(c.backing.raw, c.backing.offset+4)
		if c.released {
			if !errors.Is(e, memory.ErrRuntimeOwner) {
				t.Fatal("released pooled raw remains accessible", e)
			}
			continue
		}
		if e != nil || v != c.capacity {
			t.Fatal("retained physical header", v, c.capacity, e)
		}
		if c.backing.offset == 0 {
			if direct[c.backing.raw] {
				t.Fatal("duplicate direct physical root")
			}
			direct[c.backing.raw] = true
		}
	}
	if z.pendingCell.object != (memory.ObjectRef{}) {
		if z.pendingCell.offset == 0 {
			direct[z.pendingCell.raw] = true
		}
	}
	a := z.allocator.Stats()
	rawBuffers := map[memory.Block]bool{}
	owned := m.headerBytes + z.headerBytes + z.memory.Stats().TotalBytes + a.TotalBytes + cs.OwnerBytes + cs.RecordBytes + cs.MetadataBytes + cs.DirectoryBytes
	ws := z.words.Stats()
	owned += ws.OwnerBytes + ws.PayloadBytes + ws.BlockMetadataBytes + ws.DirectoryBytes + ws.Descriptors.OwnerBytes + ws.Descriptors.RecordBytes + ws.Descriptors.MetadataBytes + ws.Descriptors.DirectoryBytes
	if z.natural != nil {
		n := z.natural
		owned += n.headerBytes
		for _, buffer := range []*sourceLimbBuffer{&n.division[0], &n.division[1], &n.division[2], &n.division[3], &n.verification} {
			for _, raw := range []memory.Block{buffer.raw, buffer.pending} {
				if raw != (memory.Block{}) {
					if rawBuffers[raw] {
						t.Fatal("duplicate Natural raw root")
					}
					rawBuffers[raw] = true
				}
			}
		}
		if n.text != nil {
			ts := n.text.Stats()
			owned += ts.OwnerBytes + ts.PayloadBytes + ts.MetadataBytes + ts.DirectoryBytes
		}
	}
	if s != nil {
		owned += s.headerBytes + s.cacheBytes
	}
	if b.Used() != owned {
		t.Fatal("retained complete physical/cold graph", b.Used(), owned)
	}
	if z.memory.Stats().LiveBlocks != uint32(len(direct)+len(rawBuffers))+a.Chunks+a.PendingBlocks {
		t.Fatal("unnamed actual Runtime physical root", z.memory.Stats(), a, len(direct), len(rawBuffers))
	}
}

func TestTransitiveSourceFinalizeColdOwnerAndReadinessGuards(t *testing.T) {
	var zeroM RationalManager
	var zeroS RationalService
	if e := zeroM.SourceFinalize(); !errors.Is(e, memory.ErrRuntimeOwner) {
		t.Fatal(e)
	}
	if e := zeroS.SourceFinalize(); !errors.Is(e, ErrRationalServiceCopy) {
		t.Fatal(e)
	}
	m, s, rt, b := transitiveSourceOwner(t, true, memory.SmallAllocatorProfile{PointerBits: 64})
	copiedM := *m
	copiedS := RationalService{self: s, manager: m, ownerID: s.ownerID}
	state, used := rt.State(), b.Used()
	if e := copiedM.SourceFinalize(); !errors.Is(e, memory.ErrRuntimeOwner) {
		t.Fatal(e)
	}
	if e := copiedS.SourceFinalize(); !errors.Is(e, ErrRationalServiceCopy) {
		t.Fatal(e)
	}
	// Cold readiness unit test, not a fabricated physical free refusal. The
	// malformed numeral must remain unread at the earlier owner-state guard.
	bad := Rational{num: Int{cell: numericCellRef(^uint32(0))}, den: NewInt(1)}
	m.closing = true
	if e := m.checkOwnerReady(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if e := m.check(&bad); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closing guard read numeral", e)
	}
	m.closing = false
	s.closing = true
	if _, e := s.NewValue(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	s.closing = false
	if rt.State() != state || b.Used() != used {
		t.Fatal("guard performed source actions")
	}
	transitiveSourceAudit(t, m, s, b)
	if e := s.SourceFinalize(); e != nil {
		t.Fatal(e)
	}
	transitiveSourceClose(t, m, s, rt, b)
	if e := m.SourceFinalize(); e != nil {
		t.Fatal(e)
	}
	if e := s.SourceFinalize(); e != nil {
		t.Fatal(e)
	}
	if e := m.checkOwnerReady(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

// The selected service constants are private scalar fields in Go. Give each
// distinct actual backing, in reverse allocation order, and provoke the real
// source setter failure at every hook/member boundary before child destruction.
func TestServiceSourceFinalizeActualHookAndConstantOrder(t *testing.T) {
	for stop := 0; stop < 24; stop++ {
		m, s, rt, b := transitiveSourceOwner(t, true, memory.SmallAllocatorProfile{PointerBits: 64, Debug: true})
		if _, e := s.PowerOfTwo(1); e != nil {
			t.Fatal(e)
		}
		roots := [24]*Int{
			&s.infZero.second.num, &s.infZero.second.den, &s.infZero.first.num, &s.infZero.first.den,
			&s.infOne.second.num, &s.infOne.second.den, &s.infOne.first.num, &s.infOne.first.den,
			&s.infMinusOne.second.num, &s.infMinusOne.second.den, &s.infMinusOne.first.num, &s.infMinusOne.first.den,
			&s.infIntZero.first.num, &s.infIntZero.first.den, &s.infIntOne.first.num, &s.infIntOne.first.den, &s.infIntMinusOne.first.num, &s.infIntMinusOne.first.den,
			&s.zero.num, &s.zero.den, &s.one.num, &s.one.den, &s.minusOne.num, &s.minusOne.den}
		var raws [24]memory.Block
		for i := 23; i >= 0; i-- {
			if e := m.z.SetLimbs(roots[i], transitiveSourceDigits(60)); e != nil {
				t.Fatal(e)
			}
			raws[i] = m.z.cell(roots[i].cell).backing.raw
		}
		var input Int
		if e := m.z.SetLimbs(&input, transitiveSourceDigits(120)); e != nil {
			t.Fatal(e)
		}
		b.FailAfter(0)
		e := m.z.Copy(roots[stop], &input)
		if !numericPhysicalReservationRefusal(e) || roots[stop].cell != m.z.failedRef {
			t.Fatal("actual hook setter refusal", stop, e)
		}
		b.FailAfter(-1)
		if e = s.SourceFinalize(); !errors.Is(e, arena.ErrInvalidSlot) {
			t.Fatal(stop, e)
		}
		stage := uint8(stop)
		if stop >= 18 {
			stage++
		} // cache is source stage18.
		if !s.sourceGuard || s.cleanupStage != stage || s.sourceFinalized || m.sourceFinalizing {
			t.Fatal("hook/constant source boundary", stop, s.cleanupStage)
		}
		for i, raw := range raws {
			_, e = rt.ReadUint32(raw, 8)
			if i <= stop {
				if !errors.Is(e, memory.ErrRuntimeOwner) {
					t.Fatal("prior hook/member release missing", stop, i, e)
				}
			} else if e != nil {
				t.Fatal("later hook/member release occurred", stop, i, e)
			}
		}
		transitiveSourceAudit(t, m, s, b, &input)
		transitiveSourceClose(t, m, s, rt, b)
		if !s.sourceGuard || s.cleanupStage != stage || s.sourceFinalized {
			t.Fatal("owning abort changed source hook history", stop)
		}
	}
}
