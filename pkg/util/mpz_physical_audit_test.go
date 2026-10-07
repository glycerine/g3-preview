// Independent test-only physical roots for the source-backed Integer owner.
// Source: src/util/mpz.h, src/util/mpz.cpp, src/util/mpq.h.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func numericPhysicalReservationRefusal(e error) bool {
	if errors.Is(e, arena.ErrInjectedFailure) {
		return true
	}
	var f Failure
	return errors.As(e, &f) && f.Kind == FailureMemout && f.ErrorCode() == 101
}

func integerPhysicalAudit(t *testing.T, m *IntegerManager, public ...*Int) {
	integerPhysicalAuditWithRaw(t, m, nil, public...)
}

func integerPhysicalAuditWithRaw(t *testing.T, m *IntegerManager, external []Block, public ...*Int) {
	t.Helper()
	cells := map[NumericCellRef]bool{}
	add := func(r NumericCellRef) {
		if r == 0 || m.IsFailureRef(r) {
			return
		}
		if !ValidNumericRef(m.Cells, r) {
			t.Fatal("invalid ordinary integer root", r)
		}
		cells[r] = true
	}
	for _, r := range m.Tmp {
		add(r)
	}
	for _, r := range m.Arg {
		add(r)
	}
	for i := range m.RawDisplay.Roots {
		add(m.RawDisplay.Roots[i].Cell)
	}
	if m.MatchesDividesNumeralInvocation(&m.DividesNumeral.Receipt) {
		add(m.DividesNumeral.Numeral.Cell)
	}
	for _, s := range []*IntegerRawScope{&m.DivisionRaw, &m.ExtendedRaw, &m.GcdRaw} {
		for i := uint8(0); i < s.Count; i++ {
			add(s.Values[i].Cell)
		}
	}
	if m.MatchesAddMulInvocation(&m.GcdRaw.AddMul) || m.MatchesAddMulInvocation(&m.AddMulInvocation.Receipt) {
		add(m.AddMulInvocation.Tmp.Cell)
	}
	add(m.PowerRaw.Value.Cell)
	for i := range m.RootRaw.Values {
		add(m.RootRaw.Values[i].Cell)
	}
	for i := range m.PerfectSquare.Values {
		add(m.PerfectSquare.Values[i].Cell)
	}
	add(m.Smt2Display.Value.Cell)
	add(m.IntMin.Cell)
	add(m.Two64.Cell)
	if parent := m.MpfParent; parent != nil {
		for s := parent.FormatScopes; s != nil; s = s.Next {
			if !parent.MatchesFormatFrame(s) {
				t.Fatal("invalid actual format frame")
			}
			if s.Constructed >= 1 {
				add(s.Num.Cell)
			}
			if s.Constructed >= 2 {
				add(s.Denom.Cell)
			}
			if s.Constructed >= 3 {
				add(s.Q.Num.Cell)
				add(s.Q.Den.Cell)
			}
			if s.Decimal.Parent != nil {
				for i := 1; i < 4; i++ {
					add(s.Decimal.Values[i].Cell)
				}
				if s.Decimal.Count >= 4 {
					add(s.Decimal.Values[0].Cell)
				}
			}
		}
		for w := parent.PowerWrappers; w != nil; w = w.Next {
			if !w.Valid() {
				t.Fatal("invalid actual Power owner")
			}
			if w.Initialized && !w.SourceDelDone && w.Block != (Block{}) {
				a, e := w.Capture()
				if e != nil {
					t.Fatal("invalid actual Power NUM", e)
				}
				add(a.Cell)
			}
		}
	}
	for _, v := range public {
		if v != nil {
			add(v.Cell)
		}
	}
	// These flags name genuinely retained preparations/cleanup continuations;
	// ordinary unnamed published descriptors are deliberately not accepted.
	for slot := uint64(1); slot <= uint64(m.Cells.Stats().Slots); slot++ {
		c, e := m.Cells.Read(arena.Slot(slot))
		if e != nil {
			continue
		}
		if c.Unpublished || c.CleanupPending {
			add(MakeNumericRef(arena.Slot(slot), m.Cells.State(arena.Slot(slot)).Generation))
		}
	}
	if uint32(len(cells)) != m.Cells.Stats().Live {
		t.Fatal("unnamed actual integer descriptor", len(cells), m.Cells.Stats().Live)
	}
	objects := map[ObjectRef]bool{}
	for ref := range cells {
		c := m.Cell(ref)
		if c.Payload != (arena.VectorRef{}) {
			t.Fatal("phantom cell Words payload")
		}
		if c.Released {
			continue
		}
		object, e := m.CellObject(ref)
		if e != nil {
			t.Fatal("invalid actual object root", ref, e)
		}
		if objects[object] {
			t.Fatal("duplicate physical numeric object", object)
		}
		objects[object] = true
	}
	if m.PendingCell.Object != (ObjectRef{}) {
		if objects[m.PendingCell.Object] {
			t.Fatal("duplicated pending physical object")
		}
		if _, e := m.Allocator.ObjectExtent(m.PendingCell.Object); e != nil {
			t.Fatal("invalid pending object", e)
		}
	}
	words := map[arena.VectorRef]bool{}
	for _, r := range m.Natural.FormatScratch {
		if r == (arena.VectorRef{}) {
			continue
		}
		if !m.Words.IsLive(r) {
			t.Fatal("invalid format Words root")
		}
		words[r] = true
	}
	if uint32(len(words)) != m.Words.Stats().Live {
		t.Fatal("unnamed real formatting Words root", len(words), m.Words.Stats().Live)
	}
	raw := map[Block]bool{}
	if parent := m.MpfParent; parent != nil {
		for s := parent.FormatScopes; s != nil; s = s.Next {
			if !parent.MatchesFormatFrame(s) {
				t.Fatal("invalid actual format frame")
			}
			for _, b := range []Block{s.Transferred.Block(), s.Decimal.Transferred.Block()} {
				if b != (Block{}) {
					if raw[b] {
						t.Fatal("duplicate format transferred raw")
					}
					if _, e := m.Memory.Size(b); e != nil {
						t.Fatal(e)
					}
					raw[b] = true
				}
			}
			for _, c := range []SourceCellBacking{s.Numeric.Cell, s.Decimal.Numeric.Cell} {
				if c.Object == (ObjectRef{}) {
					continue
				}
				if objects[c.Object] {
					t.Fatal("duplicate format object")
				}
				ended, e := m.Allocator.SourceObjectOwnerEnded()
				if e != nil {
					t.Fatal(e)
				}
				if ended && c.Offset == 0 {
					if raw[c.Raw] {
						t.Fatal("duplicate format direct raw")
					}
					if _, e := m.Memory.Size(c.Raw); e != nil {
						t.Fatal(e)
					}
					raw[c.Raw] = true
				} else {
					if _, e := m.Allocator.ObjectExtent(c.Object); e != nil {
						t.Fatal(e)
					}
					objects[c.Object] = true
				}
			}
		}
	}

	if parent := m.MpfParent; parent != nil {
		for w := parent.PowerWrappers; w != nil; w = w.Next {
			if !w.Valid() {
				t.Fatal("invalid actual Power owner")
			}
			for _, block := range []Block{w.Block, w.Transferred.Block()} {
				if block == (Block{}) {
					continue
				}
				if raw[block] {
					t.Fatal("duplicate actual Power raw owner")
				}
				if _, e := m.Memory.Size(block); e != nil {
					t.Fatal("invalid actual Power raw owner", e)
				}
				raw[block] = true
			}
			if c := w.Roots.Cell; c.Object != (ObjectRef{}) {
				if objects[c.Object] {
					t.Fatal("duplicate actual Power object")
				}
				if _, e := m.Allocator.ObjectExtent(c.Object); e != nil {
					t.Fatal("invalid actual Power object", e)
				}
				objects[c.Object] = true
			}
		}

		for h := parent.RationalSetHistories; h != nil; h = h.Next {
			if h.Self != h || h.Manager != parent || h.Invocation == 0 {
				t.Fatal("invalid actual rational SET record")
			}
			for _, block := range []Block{h.Transferred.Block(), h.Numeric.Allocator[0], h.Numeric.Allocator[1], h.Numeric.Allocator[2]} {
				if block == (Block{}) {
					continue
				}
				if raw[block] {
					t.Fatal("duplicate rational SET raw")
				}
				if _, e := m.Memory.Size(block); e != nil {
					t.Fatal(e)
				}
				raw[block] = true
			}
			if c := h.Numeric.Cell; c.Object != (ObjectRef{}) {
				if objects[c.Object] {
					t.Fatal("duplicate rational SET object")
				}
				if _, e := m.Allocator.ObjectExtent(c.Object); e != nil {
					t.Fatal(e)
				}
				objects[c.Object] = true
			}
		}
		for h := parent.FormattedSetHistories; h != nil; h = h.Next {
			if h.Self != h || h.Manager != parent || h.Issued.Frame != h || h.Issued.Manager != parent || h.Issued.Invocation != h.Invocation || h.Invocation == 0 {
				t.Fatal("invalid actual formatted SET record")
			}
			for _, block := range []Block{h.Transferred.Block(), h.Numeric.Allocator[0], h.Numeric.Allocator[1], h.Numeric.Allocator[2]} {
				if block == (Block{}) {
					continue
				}
				if raw[block] {
					t.Fatal("duplicate formatted SET raw")
				}
				if _, e := m.Memory.Size(block); e != nil {
					t.Fatal(e)
				}
				raw[block] = true
			}
			if c := h.Numeric.Cell; c.Object != (ObjectRef{}) {
				if objects[c.Object] {
					t.Fatal("duplicate formatted SET cell")
				}
				ended, e := m.Allocator.SourceObjectOwnerEnded()
				if e != nil {
					t.Fatal(e)
				}
				if ended && c.Offset == 0 {
					if raw[c.Raw] {
						t.Fatal("duplicate formatted SET direct raw")
					}
					if _, e := m.Memory.Size(c.Raw); e != nil {
						t.Fatal(e)
					}
					raw[c.Raw] = true
				} else {
					if _, e := m.Allocator.ObjectExtent(c.Object); e != nil {
						t.Fatal(e)
					}
					objects[c.Object] = true
				}
			}
		}
		for _, s := range []*ArbitraryFloatRoundScope{&parent.RoundScope, &parent.RoundSqrtScope} {
			if s.Manager == parent && s.Invocation != 0 {
				if block := s.Transferred.Block(); block != (Block{}) {
					if raw[block] {
						t.Fatal("duplicate round transferred raw")
					}
					if _, e := m.Memory.Size(block); e != nil {
						t.Fatal(e)
					}
					raw[block] = true
				}
				if c := s.Numeric.Cell; c.Object != (ObjectRef{}) {
					if objects[c.Object] {
						t.Fatal("duplicate round cell")
					}
					ended, e := m.Allocator.SourceObjectOwnerEnded()
					if e != nil {
						t.Fatal(e)
					}
					if ended && c.Offset == 0 {
						if raw[c.Raw] {
							t.Fatal("duplicate round direct raw")
						}
						if _, e := m.Memory.Size(c.Raw); e != nil {
							t.Fatal(e)
						}
						raw[c.Raw] = true
					} else {
						if _, e := m.Allocator.ObjectExtent(c.Object); e != nil {
							t.Fatal(e)
						}
						objects[c.Object] = true
					}
				}
			}
		}
		for q := parent.InputCalls; q != nil; q = q.Next {
			if q.Self != q || q.Manager != parent || q.Issued.Manager != parent || q.Issued.Frame != q || q.Issued.Invocation == 0 {
				t.Fatal("invalid actual input call owner")
			}
			if block := q.Transferred.Block(); block != (Block{}) {
				if raw[block] {
					t.Fatal("duplicate input transferred raw")
				}
				if _, e := m.Memory.Size(block); e != nil {
					t.Fatal(e)
				}
				raw[block] = true
			}
			if c := q.Numeric.Cell; c.Object != (ObjectRef{}) {
				if objects[c.Object] {
					t.Fatal("duplicate input cell")
				}
				ended, e := m.Allocator.SourceObjectOwnerEnded()
				if e != nil {
					t.Fatal(e)
				}
				if ended && c.Offset == 0 {
					if raw[c.Raw] {
						t.Fatal("duplicate input direct raw")
					}
					if _, e := m.Memory.Size(c.Raw); e != nil {
						t.Fatal(e)
					}
					raw[c.Raw] = true
				} else {
					if _, e := m.Allocator.ObjectExtent(c.Object); e != nil {
						t.Fatal(e)
					}
					objects[c.Object] = true
				}
			}
		}
		for q := parent.ValueCalls; q != nil; q = q.Next {
			if q.Retired {
				continue
			}
			child := &q.ChildFrame
			if !m.MatchesCopyAllocationInvocation(&q.Issued.Child) {
				t.Fatal("invalid retained MPF child")
			}
			block := child.Transferred.Block()
			if block != (Block{}) {
				if raw[block] {
					t.Fatal("duplicate transferred MPF raw root")
				}
				if _, e := m.Memory.Size(block); e != nil {
					t.Fatal("invalid transferred MPF raw root", e)
				}
				raw[block] = true
			}
			if c := child.Roots.Cell; c.Object != (ObjectRef{}) {
				if objects[c.Object] {
					t.Fatal("duplicate retained MPF cell")
				}
				ended, e := m.Allocator.SourceObjectOwnerEnded()
				if e != nil {
					t.Fatal(e)
				}
				if ended && c.Offset == 0 {
					if raw[c.Raw] {
						t.Fatal("duplicate retained direct MPF raw")
					}
					if _, e := m.Memory.Size(c.Raw); e != nil {
						t.Fatal(e)
					}
					raw[c.Raw] = true
				} else {
					if _, e := m.Allocator.ObjectExtent(c.Object); e != nil {
						t.Fatal("invalid retained MPF cell", e)
					}
					objects[c.Object] = true
				}
			}
		}
	}
	if m.SourceFinalized || m.ConstructorUnwound {
		for ref := range cells {
			c := m.Cell(ref)
			if !c.Released && c.Backing.Offset == 0 {
				if _, e := m.Memory.Size(c.Backing.Raw); e != nil {
					t.Fatal("invalid named surviving direct source cell", e)
				}
				raw[c.Backing.Raw] = true
			}
		}
	}

	buffers := []*SourceLimbBuffer{&m.Natural.Division[0], &m.Natural.Division[1], &m.Natural.Division[2], &m.Natural.Division[3]}
	if m.Natural.VerificationActive {
		buffers = append(buffers, &m.Natural.Verification)
	}
	if m.Natural.Formatting.Active || m.Natural.Formatting.PendingField {
		for i := range m.Natural.Formatting.Locals {
			buffers = append(buffers, &m.Natural.Formatting.Locals[i])
		}
	}
	for _, s := range buffers {
		for _, r := range []Block{s.Raw, s.Pending} {
			if r == (Block{}) {
				continue
			}
			if raw[r] {
				t.Fatal("duplicate actual Natural raw root")
			}
			raw[r] = true
			if _, e := m.Memory.ReadUint32(r, 0); e != nil && !errors.Is(e, ErrRuntimeUninitialized) {
				t.Fatal("invalid Natural raw root", e)
			}
		}
	}
	for _, r := range []Block{m.Natural.CharFormatting.Buffer.Raw, m.Natural.CharFormatting.Buffer.PendingField} {
		if r == (Block{}) {
			continue
		}
		if raw[r] {
			t.Fatal("duplicate character raw root")
		}
		raw[r] = true
		if _, e := m.Memory.Read(r, 0); e != nil && !errors.Is(e, ErrRuntimeUninitialized) {
			t.Fatal("invalid character raw root", e)
		}
	}
	for _, block := range external {
		if block == (Block{}) || raw[block] {
			t.Fatal("duplicate or empty named external block", block)
		}
		if _, e := m.Memory.Size(block); e != nil {
			t.Fatal("invalid actual external raw root", block, e)
		}
		raw[block] = true
	}
	allocator := m.Allocator.Stats()
	if m.Memory.Stats().LiveBlocks != allocator.Chunks+allocator.DirectObjects+allocator.PendingBlocks+uint32(len(raw)) {
		t.Fatal("unnamed physical runtime allocation", m.Memory.Stats(), allocator, len(raw))
	}
}

func integerActualOwnedBytes(m *IntegerManager) uint64 {
	cs, ws := m.Cells.Stats(), m.Words.Stats()
	n := m.HeaderBytes + m.Natural.HeaderBytes + m.Memory.Stats().TotalBytes + m.Allocator.Stats().TotalBytes
	n += cs.OwnerBytes + cs.RecordBytes + cs.MetadataBytes + cs.DirectoryBytes
	n += ws.OwnerBytes + ws.PayloadBytes + ws.BlockMetadataBytes + ws.DirectoryBytes + ws.Descriptors.OwnerBytes + ws.Descriptors.RecordBytes + ws.Descriptors.MetadataBytes + ws.Descriptors.DirectoryBytes
	if m.Natural.Text != nil {
		s := m.Natural.Text.Stats()
		n += s.OwnerBytes + s.PayloadBytes + s.MetadataBytes + s.DirectoryBytes
	}
	return n
}

// Snapshot actual retained physical storage independently of source event count.
// Callers prime every branch first and select sourceEvents only when the actual
// source profile/body allocates lexical locals during the measured operation.
type numericWarmSnapshot struct {
	used, owned, size, count uint64
	runtime                  RuntimeStats
	allocator                AllocatorStats
	cells, words             uint32
}

func numericWarmProfileSnapshot(t *testing.T, m *IntegerManager, b *arena.Budget) numericWarmSnapshot {
	t.Helper()
	size, e := m.Memory.AllocationSize()
	if e != nil {
		t.Fatal(e)
	}
	count, e := m.Memory.AllocationCount()
	if e != nil {
		t.Fatal(e)
	}
	return numericWarmSnapshot{b.Used(), integerActualOwnedBytes(m), size, count, m.Memory.Stats(), m.Allocator.Stats(), m.Cells.Stats().Live, m.Words.Stats().Live}
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
