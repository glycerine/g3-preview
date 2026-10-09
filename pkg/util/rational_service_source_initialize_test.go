// Source: src/util/rational.cpp:64-83; src/util/rational.h.
// Leonardo de Moura, 2006-09-18.
package util

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalServicePreparedSourcePublicationAndConstants(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	s, e := PrepareRationalService(serviceConfig(b))
	if e != nil {
		t.Fatal(e)
	}
	if s.Manager != nil || b.Used() != uint64(reflect.TypeOf(RationalService{}).Size()) {
		t.Fatal("cold context must not publish source manager", b.Used())
	}
	for i, q := range []*Rational{&s.ZeroField, &s.OneField, &s.MinusOneField, &s.InfZeroField.First, &s.InfZeroField.Second, &s.InfOneField.First, &s.InfOneField.Second, &s.InfMinusOneField.First, &s.InfMinusOneField.Second, &s.InfIntZeroField.First, &s.InfIntOneField.First, &s.InfIntMinusOneField.First} {
		if q.Num != NewInt(0) || q.Den != NewInt(1) {
			t.Fatal("source static default", i, q)
		}
	}
	if _, e = s.One(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("ordinary cold access", e)
	}
	serviceMust(t, s.InitializeSource())
	m := s.Manager
	for i, a := range []*Rational{&s.ZeroField, &s.OneField, &s.MinusOneField, &s.InfZeroField.First, &s.InfOneField.First, &s.InfMinusOneField.First, &s.InfIntZeroField.First, &s.InfIntOneField.First, &s.InfIntMinusOneField.First} {
		want := []string{"0", "1", "-1"}[i%3]
		got, e := m.String(a)
		if e != nil || got != want {
			t.Fatal(i, got, e, want)
		}
	}
	b.FailAfter(0)
	serviceMust(t, s.InitializeSource())
	if s.Manager != m {
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
		r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
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
			if s.Manager == nil {
				t.Fatal("success unpublished manager")
			}
		} else {
			var mf Failure
			if !errors.Is(e, arena.ErrInjectedFailure) && !(errors.As(e, &mf) && mf.Kind == FailureMemout && mf.ErrorCode() == 101 && r.State().OutOfMemory) {
				t.Fatal(ordinal, e)
			}
			if s.Manager != nil {
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
func sourcePreparedGraph(t *testing.T, s *RationalService, r *Runtime) {
	t.Helper()
	want := s.HeaderBytes + r.Stats().TotalBytes
	var z *IntegerManager
	if s.Manager != nil {
		want += s.Manager.HeaderBytes
		z = s.Manager.Z
	} else {
		z = s.initializationFailure
	}
	if z != nil {
		want += z.HeaderBytes + z.Allocator.Stats().TotalBytes
		if z.Natural != nil {
			want += z.Natural.HeaderBytes
		}
		if z.Cells != nil {
			q := z.Cells.Stats()
			want += q.OwnerBytes + q.RecordBytes + q.MetadataBytes + q.DirectoryBytes
		}
		if z.Natural != nil {
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
	cells := map[NumericCellRef]bool{}
	add := func(r NumericCellRef) {
		if r == 0 || m.IsFailureRef(r) {
			return
		}
		if !ValidNumericRef(m.Cells, r) {
			t.Fatal("invalid constructor numeral", r)
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
	for i := range m.PerfectSquare.Values {
		add(m.PerfectSquare.Values[i].Cell)
	}
	add(m.Smt2Display.Value.Cell)
	add(m.IntMin.Cell)
	add(m.Two64.Cell)
	if m.Cells != nil {
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
			t.Fatal("unnamed constructor descriptors", len(cells), m.Cells.Stats().Live)
		}
	}
	objects := map[ObjectRef]bool{}
	closedRaw := map[Block]bool{}
	checkRetainedRaw := func(c SourceCellBacking) {
		n, e := m.Memory.Size(c.Raw)
		if e != nil || c.Raw == (Block{}) || c.Offset > n || c.extent > n-c.Offset {
			t.Fatal("invalid retained constructor raw", c, e)
		}
		if closedRaw[c.Raw] {
			t.Fatal("duplicate retained constructor raw owner", c.Raw)
		}
		closedRaw[c.Raw] = true
	}
	for ref := range cells {
		c := m.Cell(ref)
		if c.Payload != (arena.VectorRef{}) {
			t.Fatal("shadow cell payload")
		}
		if c.Released {
			continue
		}
		o, e := m.CellObject(ref)
		if e == nil {
			_, e = m.Allocator.ObjectExtent(o)
		}
		if objects[o] {
			t.Fatal("duplicate constructor object", o)
		}
		objects[o] = true
		if errors.Is(e, arena.ErrClosed) {
			checkRetainedRaw(c.Backing)
			continue
		}
		if e != nil {
			t.Fatal("actual constructor object", ref, o, e)
		}
	}
	if m.PendingCell.Object != (ObjectRef{}) {
		if objects[m.PendingCell.Object] {
			t.Fatal("duplicate pending constructor object")
		}
		if _, e := m.Allocator.ObjectExtent(m.PendingCell.Object); errors.Is(e, arena.ErrClosed) {
			checkRetainedRaw(m.PendingCell)
		} else if e != nil {
			t.Fatal("pending constructor extent", e)
		}
	}
	a := m.Allocator.Stats()
	if m.Memory.Stats().LiveBlocks != a.Chunks+a.DirectObjects+a.PendingBlocks+uint32(len(closedRaw)) {
		t.Fatal("unnamed early raw block", m.Memory.Stats(), a)
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
