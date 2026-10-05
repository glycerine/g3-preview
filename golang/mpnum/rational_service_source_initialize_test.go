// Source: src/util/rational.cpp:64-83; src/util/rational.h.
// Leonardo de Moura, 2006-09-18. Licensed under the MIT license; see LICENSE.txt.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

func TestRationalServicePreparedSourcePublicationAndConstants(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	s, e := PrepareRationalService(serviceConfig(b))
	if e != nil {
		t.Fatal(e)
	}
	if s.manager != nil || b.Used() != uint64(reflect.TypeOf(RationalService{}).Size()) {
		t.Fatal("cold context must not publish source manager", b.Used())
	}
	for i, q := range []*Rational{&s.zero, &s.one, &s.minusOne, &s.infZero.first, &s.infZero.second, &s.infOne.first, &s.infOne.second, &s.infMinusOne.first, &s.infMinusOne.second, &s.infIntZero.first, &s.infIntOne.first, &s.infIntMinusOne.first} {
		if q.num != NewInt(0) || q.den != NewInt(1) {
			t.Fatal("source static default", i, q)
		}
	}
	if _, e = s.One(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("ordinary cold access", e)
	}
	serviceMust(t, s.InitializeSource())
	m := s.manager
	for i, a := range []*Rational{&s.zero, &s.one, &s.minusOne, &s.infZero.first, &s.infOne.first, &s.infMinusOne.first, &s.infIntZero.first, &s.infIntOne.first, &s.infIntMinusOne.first} {
		want := []string{"0", "1", "-1"}[i%3]
		got, e := m.String(a)
		if e != nil || got != want {
			t.Fatal(i, got, e, want)
		}
	}
	b.FailAfter(0)
	serviceMust(t, s.InitializeSource())
	if s.manager != m {
		t.Fatal("pointer-present source init recreated manager")
	}
	b.FailAfter(-1)
	serviceMust(t, s.SourceFinalize())
	serviceMust(t, s.CloseChecked())
	if b.Used() != 0 {
		t.Fatal("remaining roots", b.Used())
	}
}

func TestRationalServicePreparedEveryActualManagerConstructorRefusal(t *testing.T) {
	successes, retainedCount := 0, 0
	successOrdinal := int64(-1)
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		b := arena.NewBudget(1 << 24)
		r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
		if e != nil {
			t.Fatal(e)
		}
		cfg := serviceConfig(b)
		cfg.Integers.Memory = r
		s, e := PrepareRationalService(cfg)
		if e != nil {
			t.Fatal(e)
		}
		b.FailAfter(ordinal)
		e = s.InitializeSource()
		if e == nil {
			successes++
			if s.manager == nil {
				t.Fatal("success unpublished manager")
			}
		} else {
			var mf memory.Failure
			if !errors.Is(e, arena.ErrInjectedFailure) && !(errors.As(e, &mf) && mf.Kind == memory.FailureMemout && mf.ErrorCode() == 101 && r.State().OutOfMemory) {
				t.Fatal(ordinal, e)
			}
			if s.manager != nil {
				t.Fatal("incomplete manager ctor falsely published", ordinal)
			}
			if b.Used() < uint64(reflect.TypeOf(RationalService{}).Size()) {
				t.Fatal("lost cold owner", ordinal)
			}
		}
		sourcePreparedGraph(t, s, r)
		if s.initializationFailure != nil {
			retainedCount++
			if s.RuntimeOwner() != r {
				t.Fatal("failed child receipt provider", ordinal)
			}
			before := r.State()
			used := b.Used()
			if again := s.InitializeSource(); again != e || r.State() != before || b.Used() != used {
				t.Fatal("pending-own retry guard", ordinal, again)
			}
		}
		if e != nil {
			cause := s.initializationCause
			if g := s.SourceFinalize(); !errors.Is(g, ErrRationalServiceNoManager) {
				t.Fatal("failed ctor source finalizer", ordinal, g)
			}
			if s.initializationCause != cause || s.cleanupStage != 0 || !s.sourceGuard {
				t.Fatal("source guard altered initialization prefix", ordinal)
			}
			sourcePreparedGraph(t, s, r)
		}
		// Observe the invocation before lifting its real allocation fault.
		b.FailAfter(-1)
		serviceMust(t, s.CloseChecked())
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(ordinal, b.Used())
		}
		if successes != 0 {
			successOrdinal = ordinal
			break
		}
	}
	if successes != 1 {
		t.Fatal("no actual first success")
	}
	t.Logf("actual constructor refusals=%d retained-child-prefixes=%d firstsuccess=%d", successOrdinal, retainedCount, successOrdinal)
}

func TestRationalServicePreparedColdAbortAndOwnerGuards(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	s, e := PrepareRationalService(serviceConfig(b))
	if e != nil {
		t.Fatal(e)
	}
	// A literal test-only shallow copy. The provenance guard runs before any
	// access to the copied mutex; reflection avoids vet treating this diagnostic
	// construction as an ordinary lock-bearing value transfer.
	var copy RationalService
	reflect.ValueOf(&copy).Elem().Set(reflect.ValueOf(s).Elem())
	if e = copy.InitializeSource(); !errors.Is(e, ErrRationalServiceCopy) {
		t.Fatal(e)
	}
	if e = s.SourceFinalize(); !errors.Is(e, ErrRationalServiceNoManager) {
		t.Fatal("reached unconditional rational destructor", e)
	}
	if !s.sourceGuard || s.cleanupStage != 0 {
		t.Fatal("nil manager source cursor", s.cleanupStage)
	}
	serviceMust(t, s.CloseChecked())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	if e = s.InitializeSource(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	serviceMust(t, s.CloseChecked())
}

// Every byte below belongs to named concrete owners. Runtime provider storage
// is added once; failed constructor metadata is not a published source manager.
func sourcePreparedGraph(t *testing.T, s *RationalService, r *memory.Runtime) {
	t.Helper()
	want := s.headerBytes + r.Stats().TotalBytes
	var z *IntegerManager
	if s.manager != nil {
		want += s.manager.headerBytes
		z = s.manager.z
	} else {
		z = s.initializationFailure
	}
	if z != nil {
		want += z.headerBytes + z.allocator.Stats().TotalBytes
		if z.natural != nil {
			want += z.natural.headerBytes
		}
		if z.cells != nil {
			q := z.cells.Stats()
			want += q.OwnerBytes + q.RecordBytes + q.MetadataBytes + q.DirectoryBytes
		}
		if z.words != nil {
			q := z.words.Stats()
			want += q.OwnerBytes + q.PayloadBytes + q.BlockMetadataBytes + q.DirectoryBytes + q.Descriptors.OwnerBytes + q.Descriptors.RecordBytes + q.Descriptors.MetadataBytes + q.Descriptors.DirectoryBytes
		}
		if z.natural != nil && z.natural.text != nil {
			q := z.natural.text.Stats()
			want += q.OwnerBytes + q.PayloadBytes + q.MetadataBytes + q.DirectoryBytes
		}
		if z.natural != nil {
			integerPhysicalAudit(t, z)
		} else {
			sourcePreparedEarlyIntegerPhysical(t, z)
		}
	}
	if s.initializationBudget.Used() != want {
		t.Fatal("named source-init graph", s.initializationBudget.Used(), want)
	}
}

// An interrupted Integer constructor may precede Natural construction entirely.
// Its real cell preparations and allocator objects are still named and checked.
func sourcePreparedEarlyIntegerPhysical(t *testing.T, m *IntegerManager) {
	t.Helper()
	cells := map[numericCellRef]bool{}
	add := func(r numericCellRef) {
		if r == 0 || m.isFailureRef(r) {
			return
		}
		if !validNumericRef(m.cells, r) {
			t.Fatal("invalid constructor numeral", r)
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
	for i := range m.perfectSquare.values {
		add(m.perfectSquare.values[i].cell)
	}
	add(m.smt2Display.value.cell)
	add(m.intMin.cell)
	add(m.two64.cell)
	if m.cells != nil {
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
			t.Fatal("unnamed constructor descriptors", len(cells), m.cells.Stats().Live)
		}
	}
	objects := map[memory.ObjectRef]bool{}
	closedRaw := map[memory.Block]bool{}
	checkRetainedRaw := func(c sourceCellBacking) {
		n, e := m.memory.Size(c.raw)
		if e != nil || c.raw == (memory.Block{}) || c.offset > n || c.extent > n-c.offset {
			t.Fatal("invalid retained constructor raw", c, e)
		}
		if closedRaw[c.raw] {
			t.Fatal("duplicate retained constructor raw owner", c.raw)
		}
		closedRaw[c.raw] = true
	}
	for ref := range cells {
		c := m.cell(ref)
		if c.payload != (arena.VectorRef{}) {
			t.Fatal("shadow cell payload")
		}
		if c.released {
			continue
		}
		o, e := m.cellObject(ref)
		if e == nil {
			_, e = m.allocator.ObjectExtent(o)
		}
		if objects[o] {
			t.Fatal("duplicate constructor object", o)
		}
		objects[o] = true
		if errors.Is(e, arena.ErrClosed) {
			checkRetainedRaw(c.backing)
			continue
		}
		if e != nil {
			t.Fatal("actual constructor object", ref, o, e)
		}
	}
	if m.pendingCell.object != (memory.ObjectRef{}) {
		if objects[m.pendingCell.object] {
			t.Fatal("duplicate pending constructor object")
		}
		if _, e := m.allocator.ObjectExtent(m.pendingCell.object); errors.Is(e, arena.ErrClosed) {
			checkRetainedRaw(m.pendingCell)
		} else if e != nil {
			t.Fatal("pending constructor extent", e)
		}
	}
	if m.words != nil && m.words.Stats().Live != 0 {
		t.Fatal("unnamed early Words root")
	}
	a := m.allocator.Stats()
	if m.memory.Stats().LiveBlocks != a.Chunks+a.DirectObjects+a.PendingBlocks+uint32(len(closedRaw)) {
		t.Fatal("unnamed early raw block", m.memory.Stats(), a)
	}
}

func TestRationalServicePreparedOrdinaryReadiness(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	cold, e := PrepareRationalService(serviceConfig(b))
	serviceMust(t, e)
	ready, e := NewRationalService(serviceConfig(b))
	serviceMust(t, e)
	if _, e = cold.Stats(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("cold Stats", e)
	}
	if e = cold.CopyFrom(nil, ready, nil); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("cold destination copy", e)
	}
	if e = ready.CopyFrom(nil, cold, nil); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("cold source copy", e)
	}
	if cold.OwnerID() == 0 || cold.RuntimeOwner() != nil {
		t.Fatal("cold scalar inspection")
	}
	serviceMust(t, cold.CloseChecked())
	serviceMust(t, ready.CloseChecked())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
