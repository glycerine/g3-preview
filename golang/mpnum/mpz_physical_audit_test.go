// Licensed under the MIT license.
// Independent test-only physical roots for the source-backed Integer owner.
// Source: src/util/mpz.h, src/util/mpz.cpp, src/util/mpq.h.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func numericPhysicalReservationRefusal(e error) bool {
	if errors.Is(e, arena.ErrInjectedFailure) {
		return true
	}
	var f memory.Failure
	return errors.As(e, &f) && f.Kind == memory.FailureMemout && f.ErrorCode() == 101
}

func integerPhysicalAudit(t *testing.T, m *IntegerManager, public ...*Int) {
	integerPhysicalAuditWithRaw(t, m, nil, public...)
}

func integerPhysicalAuditWithRaw(t *testing.T, m *IntegerManager, external []memory.Block, public ...*Int) {
	t.Helper()
	cells := map[numericCellRef]bool{}
	add := func(r numericCellRef) {
		if r == 0 || m.isFailureRef(r) {
			return
		}
		if !validNumericRef(m.cells, r) {
			t.Fatal("invalid ordinary integer root", r)
		}
		cells[r] = true
	}
	for _, r := range m.tmp {
		add(r)
	}
	for _, r := range m.arg {
		add(r)
	}
	for i := range m.rawDisplay.roots {
		add(m.rawDisplay.roots[i].cell)
	}
	if m.matchesDividesNumeralInvocation(&m.dividesNumeral.receipt) {
		add(m.dividesNumeral.numeral.cell)
	}
	for _, s := range []*integerRawScope{&m.divisionRaw, &m.extendedRaw, &m.gcdRaw} {
		for i := uint8(0); i < s.count; i++ {
			add(s.values[i].cell)
		}
	}
	if m.matchesAddMulInvocation(&m.gcdRaw.addMul) || m.matchesAddMulInvocation(&m.addMulInvocation.receipt) {
		add(m.addMulInvocation.tmp.cell)
	}
	add(m.powerRaw.value.cell)
	for i := range m.rootRaw.values {
		add(m.rootRaw.values[i].cell)
	}
	for i := range m.perfectSquare.values {
		add(m.perfectSquare.values[i].cell)
	}
	add(m.smt2Display.value.cell)
	add(m.intMin.cell)
	add(m.two64.cell)
	if parent := m.mpfParent; parent != nil {
		for s := parent.formatScopes; s != nil; s = s.next {
			if !parent.matchesFormatFrame(s) {
				t.Fatal("invalid actual format frame")
			}
			if s.constructed >= 1 {
				add(s.num.cell)
			}
			if s.constructed >= 2 {
				add(s.denom.cell)
			}
			if s.constructed >= 3 {
				add(s.q.num.cell)
				add(s.q.den.cell)
			}
			if s.decimal.parent != nil {
				for i := 1; i < 4; i++ {
					add(s.decimal.values[i].cell)
				}
				if s.decimal.count >= 4 {
					add(s.decimal.values[0].cell)
				}
			}
		}
		for w := parent.powerWrappers; w != nil; w = w.next {
			if !w.valid() {
				t.Fatal("invalid actual Power owner")
			}
			if w.initialized && !w.sourceDelDone && w.block != (memory.Block{}) {
				a, e := w.capture()
				if e != nil {
					t.Fatal("invalid actual Power NUM", e)
				}
				add(a.cell)
			}
		}
	}
	for _, v := range public {
		if v != nil {
			add(v.cell)
		}
	}
	// These flags name genuinely retained preparations/cleanup continuations;
	// ordinary unnamed published descriptors are deliberately not accepted.
	for slot := uint64(1); slot <= uint64(m.cells.Stats().Slots); slot++ {
		c, e := m.cells.Read(arena.Slot(slot))
		if e != nil {
			continue
		}
		if c.unpublished || c.cleanupPending {
			add(makeNumericRef(arena.Slot(slot), m.cells.state(arena.Slot(slot)).generation))
		}
	}
	if uint32(len(cells)) != m.cells.Stats().Live {
		t.Fatal("unnamed actual integer descriptor", len(cells), m.cells.Stats().Live)
	}
	objects := map[memory.ObjectRef]bool{}
	for ref := range cells {
		c := m.cell(ref)
		if c.payload != (arena.VectorRef{}) {
			t.Fatal("phantom cell Words payload")
		}
		if c.released {
			continue
		}
		object, e := m.cellObject(ref)
		if e != nil {
			t.Fatal("invalid actual object root", ref, e)
		}
		if objects[object] {
			t.Fatal("duplicate physical numeric object", object)
		}
		objects[object] = true
	}
	if m.pendingCell.object != (memory.ObjectRef{}) {
		if objects[m.pendingCell.object] {
			t.Fatal("duplicated pending physical object")
		}
		if _, e := m.allocator.ObjectExtent(m.pendingCell.object); e != nil {
			t.Fatal("invalid pending object", e)
		}
	}
	words := map[arena.VectorRef]bool{}
	for _, r := range m.natural.formatScratch {
		if r == (arena.VectorRef{}) {
			continue
		}
		if !m.words.IsLive(r) {
			t.Fatal("invalid format Words root")
		}
		words[r] = true
	}
	if uint32(len(words)) != m.words.Stats().Live {
		t.Fatal("unnamed real formatting Words root", len(words), m.words.Stats().Live)
	}
	raw := map[memory.Block]bool{}
	if parent := m.mpfParent; parent != nil {
		for s := parent.formatScopes; s != nil; s = s.next {
			if !parent.matchesFormatFrame(s) {
				t.Fatal("invalid actual format frame")
			}
			for _, b := range []memory.Block{s.transferred.Block(), s.decimal.transferred.Block()} {
				if b != (memory.Block{}) {
					if raw[b] {
						t.Fatal("duplicate format transferred raw")
					}
					if _, e := m.memory.Size(b); e != nil {
						t.Fatal(e)
					}
					raw[b] = true
				}
			}
			for _, c := range []sourceCellBacking{s.numeric.cell, s.decimal.numeric.cell} {
				if c.object == (memory.ObjectRef{}) {
					continue
				}
				if objects[c.object] {
					t.Fatal("duplicate format object")
				}
				ended, e := m.allocator.SourceObjectOwnerEnded()
				if e != nil {
					t.Fatal(e)
				}
				if ended && c.offset == 0 {
					if raw[c.raw] {
						t.Fatal("duplicate format direct raw")
					}
					if _, e := m.memory.Size(c.raw); e != nil {
						t.Fatal(e)
					}
					raw[c.raw] = true
				} else {
					if _, e := m.allocator.ObjectExtent(c.object); e != nil {
						t.Fatal(e)
					}
					objects[c.object] = true
				}
			}
		}
	}

	if parent := m.mpfParent; parent != nil {
		for w := parent.powerWrappers; w != nil; w = w.next {
			if !w.valid() {
				t.Fatal("invalid actual Power owner")
			}
			for _, block := range []memory.Block{w.block, w.transferred.Block()} {
				if block == (memory.Block{}) {
					continue
				}
				if raw[block] {
					t.Fatal("duplicate actual Power raw owner")
				}
				if _, e := m.memory.Size(block); e != nil {
					t.Fatal("invalid actual Power raw owner", e)
				}
				raw[block] = true
			}
			if c := w.roots.cell; c.object != (memory.ObjectRef{}) {
				if objects[c.object] {
					t.Fatal("duplicate actual Power object")
				}
				if _, e := m.allocator.ObjectExtent(c.object); e != nil {
					t.Fatal("invalid actual Power object", e)
				}
				objects[c.object] = true
			}
		}
		if s := &parent.roundScope; s.manager == parent && s.invocation != 0 {
			if block := s.transferred.Block(); block != (memory.Block{}) {
				if raw[block] {
					t.Fatal("duplicate round transferred raw")
				}
				if _, e := m.memory.Size(block); e != nil {
					t.Fatal(e)
				}
				raw[block] = true
			}
			if c := s.numeric.cell; c.object != (memory.ObjectRef{}) {
				if objects[c.object] {
					t.Fatal("duplicate round cell")
				}
				ended, e := m.allocator.SourceObjectOwnerEnded()
				if e != nil {
					t.Fatal(e)
				}
				if ended && c.offset == 0 {
					if raw[c.raw] {
						t.Fatal("duplicate round direct raw")
					}
					if _, e := m.memory.Size(c.raw); e != nil {
						t.Fatal(e)
					}
					raw[c.raw] = true
				} else {
					if _, e := m.allocator.ObjectExtent(c.object); e != nil {
						t.Fatal(e)
					}
					objects[c.object] = true
				}
			}
		}
		for q := parent.inputCalls; q != nil; q = q.next {
			if q.self != q || q.manager != parent || q.issued.manager != parent || q.issued.frame != q || q.issued.invocation == 0 {
				t.Fatal("invalid actual input call owner")
			}
			if block := q.transferred.Block(); block != (memory.Block{}) {
				if raw[block] {
					t.Fatal("duplicate input transferred raw")
				}
				if _, e := m.memory.Size(block); e != nil {
					t.Fatal(e)
				}
				raw[block] = true
			}
			if c := q.numeric.cell; c.object != (memory.ObjectRef{}) {
				if objects[c.object] {
					t.Fatal("duplicate input cell")
				}
				ended, e := m.allocator.SourceObjectOwnerEnded()
				if e != nil {
					t.Fatal(e)
				}
				if ended && c.offset == 0 {
					if raw[c.raw] {
						t.Fatal("duplicate input direct raw")
					}
					if _, e := m.memory.Size(c.raw); e != nil {
						t.Fatal(e)
					}
					raw[c.raw] = true
				} else {
					if _, e := m.allocator.ObjectExtent(c.object); e != nil {
						t.Fatal(e)
					}
					objects[c.object] = true
				}
			}
		}
		for q := parent.valueCalls; q != nil; q = q.next {
			if q.retired {
				continue
			}
			child := &q.childFrame
			if !m.matchesCopyAllocationInvocation(&q.issued.child) {
				t.Fatal("invalid retained MPF child")
			}
			block := child.transferred.Block()
			if block != (memory.Block{}) {
				if raw[block] {
					t.Fatal("duplicate transferred MPF raw root")
				}
				if _, e := m.memory.Size(block); e != nil {
					t.Fatal("invalid transferred MPF raw root", e)
				}
				raw[block] = true
			}
			if c := child.roots.cell; c.object != (memory.ObjectRef{}) {
				if objects[c.object] {
					t.Fatal("duplicate retained MPF cell")
				}
				ended, e := m.allocator.SourceObjectOwnerEnded()
				if e != nil {
					t.Fatal(e)
				}
				if ended && c.offset == 0 {
					if raw[c.raw] {
						t.Fatal("duplicate retained direct MPF raw")
					}
					if _, e := m.memory.Size(c.raw); e != nil {
						t.Fatal(e)
					}
					raw[c.raw] = true
				} else {
					if _, e := m.allocator.ObjectExtent(c.object); e != nil {
						t.Fatal("invalid retained MPF cell", e)
					}
					objects[c.object] = true
				}
			}
		}
	}
	if m.sourceFinalized || m.constructorUnwound {
		for ref := range cells {
			c := m.cell(ref)
			if !c.released && c.backing.offset == 0 {
				if _, e := m.memory.Size(c.backing.raw); e != nil {
					t.Fatal("invalid named surviving direct source cell", e)
				}
				raw[c.backing.raw] = true
			}
		}
	}

	buffers := []*sourceLimbBuffer{&m.natural.division[0], &m.natural.division[1], &m.natural.division[2], &m.natural.division[3]}
	if m.natural.verificationActive {
		buffers = append(buffers, &m.natural.verification)
	}
	if m.natural.formatting.active || m.natural.formatting.pending {
		for i := range m.natural.formatting.locals {
			buffers = append(buffers, &m.natural.formatting.locals[i])
		}
	}
	for _, s := range buffers {
		for _, r := range []memory.Block{s.raw, s.pending} {
			if r == (memory.Block{}) {
				continue
			}
			if raw[r] {
				t.Fatal("duplicate actual Natural raw root")
			}
			raw[r] = true
			if _, e := m.memory.ReadUint32(r, 0); e != nil && !errors.Is(e, memory.ErrRuntimeUninitialized) {
				t.Fatal("invalid Natural raw root", e)
			}
		}
	}
	for _, r := range []memory.Block{m.natural.charFormatting.buffer.raw, m.natural.charFormatting.buffer.pending} {
		if r == (memory.Block{}) {
			continue
		}
		if raw[r] {
			t.Fatal("duplicate character raw root")
		}
		raw[r] = true
		if _, e := m.memory.Read(r, 0); e != nil && !errors.Is(e, memory.ErrRuntimeUninitialized) {
			t.Fatal("invalid character raw root", e)
		}
	}
	for _, block := range external {
		if block == (memory.Block{}) || raw[block] {
			t.Fatal("duplicate or empty named external block", block)
		}
		if _, e := m.memory.Size(block); e != nil {
			t.Fatal("invalid actual external raw root", block, e)
		}
		raw[block] = true
	}
	allocator := m.allocator.Stats()
	if m.memory.Stats().LiveBlocks != allocator.Chunks+allocator.DirectObjects+allocator.PendingBlocks+uint32(len(raw)) {
		t.Fatal("unnamed physical runtime allocation", m.memory.Stats(), allocator, len(raw))
	}
}

func integerActualOwnedBytes(m *IntegerManager) uint64 {
	cs, ws := m.cells.Stats(), m.words.Stats()
	n := m.headerBytes + m.natural.headerBytes + m.memory.Stats().TotalBytes + m.allocator.Stats().TotalBytes
	n += cs.OwnerBytes + cs.RecordBytes + cs.MetadataBytes + cs.DirectoryBytes
	n += ws.OwnerBytes + ws.PayloadBytes + ws.BlockMetadataBytes + ws.DirectoryBytes + ws.Descriptors.OwnerBytes + ws.Descriptors.RecordBytes + ws.Descriptors.MetadataBytes + ws.Descriptors.DirectoryBytes
	if m.natural.text != nil {
		s := m.natural.text.Stats()
		n += s.OwnerBytes + s.PayloadBytes + s.MetadataBytes + s.DirectoryBytes
	}
	return n
}

// Snapshot actual retained physical storage independently of source event count.
// Callers prime every branch first and select sourceEvents only when the actual
// source profile/body allocates lexical locals during the measured operation.
type numericWarmSnapshot struct {
	used, owned, size, count uint64
	runtime                  memory.RuntimeStats
	allocator                memory.AllocatorStats
	cells, words             uint32
}

func numericWarmProfileSnapshot(t *testing.T, m *IntegerManager, b *arena.Budget) numericWarmSnapshot {
	t.Helper()
	size, e := m.memory.AllocationSize()
	if e != nil {
		t.Fatal(e)
	}
	count, e := m.memory.AllocationCount()
	if e != nil {
		t.Fatal(e)
	}
	return numericWarmSnapshot{b.Used(), integerActualOwnedBytes(m), size, count, m.memory.Stats(), m.allocator.Stats(), m.cells.Stats().Live, m.words.Stats().Live}
}
func numericWarmProfileAudit(t *testing.T, m *IntegerManager, b *arena.Budget, before numericWarmSnapshot, goAllocs float64, sourceEvents bool) {
	t.Helper()
	after := numericWarmProfileSnapshot(t, m, b)
	if after.used != before.used || after.owned != before.owned || after.size != before.size || after.runtime != before.runtime || after.allocator != before.allocator || after.cells != before.cells || after.words != before.words {
		t.Fatal("warmed actual physical graph changed", before, after)
	}
	if sourceEvents {
		if after.count <= before.count {
			t.Fatal("missing real warmed source allocation events", before.count, after.count)
		}
	} else if after.count != before.count || goAllocs != 0 {
		t.Fatal("warmed source-free path allocates", before.count, after.count, goAllocs)
	}
}
