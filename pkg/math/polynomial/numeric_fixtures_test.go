package polynomial

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func binaryManager(t *testing.T) (*unumeral.BinaryRationalManager, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 26)
	s, e := unumeral.NewRationalService(serviceConfig(b))
	serviceMust(t, e)
	z, e := unumeral.NewIntegerManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	m, e := unumeral.NewBinaryRationalManager(z, unumeral.BinaryRationalConfig{Diagnostics: s})
	serviceMust(t, e)
	t.Cleanup(func() {
		m.Close()
		z.Close()
		serviceMust(t, s.Close())
		if b.Used() != 0 {
			t.Errorf("retained %d", b.Used())
		}
	})
	return m, b
}
func binaryText(t *testing.T, m *unumeral.BinaryRationalManager, a *unumeral.BinaryRational, want string) {
	t.Helper()
	got, e := m.String(a)
	if e != nil || got != want {
		t.Fatalf("got %q %v want %q", got, e, want)
	}
}

func intervalSet(t *testing.T, m *unumeral.BinaryIntervalManager, a *unumeral.BinaryInterval, l, u int32) {
	t.Helper()
	lo, hi := unumeral.NewBinaryInt(l), unumeral.NewBinaryInt(u)
	serviceMust(t, m.SetBounds(a, &lo, &hi))
}
func intervalText(t *testing.T, m *unumeral.BinaryIntervalManager, a *unumeral.BinaryInterval, w string) {
	t.Helper()
	s, e := m.String(a)
	if e != nil || s != w {
		t.Fatalf("got %q %v want %q", s, e, w)
	}
}
func rationalManager(t *testing.T, synch bool) (*unumeral.RationalManager, *arena.Budget) {
	t.Helper()
	return rationalManagerWithLimit(t, synch, 1<<20)
}

func rationalManagerWithLimit(t *testing.T, synch bool, limit uint64) (*unumeral.RationalManager, *arena.Budget) {
	t.Helper()
	budget := arena.NewBudget(limit)
	m, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: budget}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}}, Synchronized: synch})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		m.Close()
		if budget.Used() != 0 {
			t.Errorf("rational close retained %d", budget.Used())
		}
	})
	return m, budget
}
func rationalString(t *testing.T, m *unumeral.RationalManager, a *unumeral.Rational) string {
	t.Helper()
	s, e := m.String(a)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func numericPhysicalReservationRefusal(e error) bool {
	if errors.Is(e, arena.ErrInjectedFailure) {
		return true
	}
	var f umemory.Failure
	return errors.As(e, &f) && f.Kind == umemory.FailureMemout && f.ErrorCode() == 101
}
func integerManager(t *testing.T) (*unumeral.IntegerManager, *arena.Budget) {
	t.Helper()
	budget := arena.NewBudget(1 << 20)
	m, err := unumeral.NewIntegerManager(unumeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: budget}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		m.Close()
		if budget.Used() != 0 {
			t.Errorf("close retained %d bytes", budget.Used())
		}
	})
	return m, budget
}
func integerString(t *testing.T, m *unumeral.IntegerManager, n *unumeral.Int) string {
	t.Helper()
	s, e := m.String(n)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func modularManager(t *testing.T, p uint64) (*unumeral.IntegerModularManager, *unumeral.IntegerManager, *arena.Budget) {
	t.Helper()
	z, b := integerManager(t)
	m, e := unumeral.NewIntegerModularManager(z)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := m.Close(); e != nil {
			t.Error(e)
		}
	})
	if p != 0 {
		if e = m.SetZPUint64(p); e != nil {
			t.Fatal(e)
		}
	}
	return m, z, b
}

// Cold Go metadata reservations retain their arena category. An actual source
// malloc refusal goes through memory_manager.cpp's out_of_memory_error(101).
func numericStorageReservationRefusal(err error) bool {
	if errors.Is(err, arena.ErrInjectedFailure) {
		return true
	}
	var physical umemory.Failure
	return errors.As(err, &physical) && physical.Kind == umemory.FailureMemout && physical.ErrorCode() == 101
}
func serviceConfig(b *arena.Budget) unumeral.RationalServiceConfig {
	return unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}}
}
func serviceNew(t *testing.T) (*unumeral.RationalService, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	s, e := unumeral.NewRationalService(serviceConfig(b))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := s.Close(); e != nil {
			t.Error(e)
		}
		if b.Used() != 0 {
			t.Errorf("close retained %d", b.Used())
		}
	})
	return s, b
}
func serviceMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func sxMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}

func numericWarmProfileSnapshot(t *testing.T, m *unumeral.IntegerManager, b *arena.Budget) numericWarmSnapshot {
	t.Helper()
	size, e := m.Memory.AllocationSize()
	if e != nil {
		t.Fatal(e)
	}
	count, e := m.Memory.AllocationCount()
	if e != nil {
		t.Fatal(e)
	}
	return numericWarmSnapshot{b.Used(), integerActualOwnedBytes(m), size, count, m.Memory.Stats(), m.Allocator.Stats(), m.Cells.Stats().Live}
}
func numericWarmProfileAudit(t *testing.T, m *unumeral.IntegerManager, b *arena.Budget, before numericWarmSnapshot, goAllocs float64, sourceEvents bool) {
	t.Helper()
	numericWarmRuntimeAudit(t, m, b, before, sourceEvents)
	if !sourceEvents && goAllocs != 0 {
		t.Fatal("warmed numeric path allocates Go storage", goAllocs)
	}
}

// Native polynomial containers may allocate Go storage while the real numeric
// and Runtime allocator graph remains warm. Audit that graph independently.
func numericWarmRuntimeAudit(t *testing.T, m *unumeral.IntegerManager, b *arena.Budget, before numericWarmSnapshot, sourceEvents bool) {
	t.Helper()
	after := numericWarmProfileSnapshot(t, m, b)
	if after.used != before.used || after.owned != before.owned || after.size != before.size || after.runtime != before.runtime || after.allocator != before.allocator || after.cells != before.cells {
		t.Fatal("warmed actual physical graph changed", before, after)
	}
	if sourceEvents {
		if after.count <= before.count {
			t.Fatal("missing real warmed source allocation events", before.count, after.count)
		}
	} else if after.count != before.count {
		t.Fatal("warmed path allocates Runtime storage", before.count, after.count)
	}
}

func roundPublicationDecode(t *testing.T, m *unumeral.IntegerManager, n *unumeral.Int) *big.Int {
	t.Helper()
	qopsMust(t, m.Check(n))
	if n.IsSmall() {
		return big.NewInt(int64(n.Val))
	}
	value := new(big.Int)
	cell := m.Cell(n.Cell)
	for i := cell.Size; i > 0; i-- {
		value.Lsh(value, 32)
		value.Add(value, new(big.Int).SetUint64(uint64(m.Digit(n.Cell, i-1))))
	}
	if n.Val < 0 {
		value.Neg(value)
	}
	return value
}

func mpzPublicationDirectOwner(t *testing.T) (*unumeral.IntegerManager, *arena.Budget) {
	return mpzPublicationOwnerProfile(t, umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true})
}
func integerPhysicalAuditWithRaw(t *testing.T, m *unumeral.IntegerManager, external []umemory.Block, public ...*unumeral.Int) {
	t.Helper()
	cells := map[unumeral.NumericCellRef]bool{}
	add := func(r unumeral.NumericCellRef) {
		if r == 0 || m.IsFailureRef(r) {
			return
		}
		if !unumeral.ValidNumericRef(m.Cells, r) {
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
	for _, s := range []*unumeral.IntegerRawScope{&m.DivisionRaw, &m.ExtendedRaw, &m.GcdRaw} {
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
			if w.Initialized && !w.SourceDelDone && w.Block != (umemory.Block{}) {
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

	for slot := uint64(1); slot <= uint64(m.Cells.Stats().Slots); slot++ {
		c, e := m.Cells.Read(arena.Slot(slot))
		if e != nil {
			continue
		}
		if c.Unpublished || c.CleanupPending {
			add(unumeral.MakeNumericRef(arena.Slot(slot), m.Cells.State(arena.Slot(slot)).Generation))
		}
	}
	if uint32(len(cells)) != m.Cells.Stats().Live {
		t.Fatal("unnamed actual integer descriptor", len(cells), m.Cells.Stats().Live)
	}
	objects := map[umemory.ObjectRef]bool{}
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
	if m.PendingCell.Object != (umemory.ObjectRef{}) {
		if objects[m.PendingCell.Object] {
			t.Fatal("duplicated pending physical object")
		}
		if _, e := m.Allocator.ObjectExtent(m.PendingCell.Object); e != nil {
			t.Fatal("invalid pending object", e)
		}
	}
	raw := map[umemory.Block]bool{}
	if parent := m.MpfParent; parent != nil {
		for s := parent.FormatScopes; s != nil; s = s.Next {
			if !parent.MatchesFormatFrame(s) {
				t.Fatal("invalid actual format frame")
			}
			for _, b := range []umemory.Block{s.Transferred.Block(), s.Decimal.Transferred.Block()} {
				if b != (umemory.Block{}) {
					if raw[b] {
						t.Fatal("duplicate format transferred raw")
					}
					if _, e := m.Memory.Size(b); e != nil {
						t.Fatal(e)
					}
					raw[b] = true
				}
			}
			for _, c := range []unumeral.SourceCellBacking{s.Numeric.Cell, s.Decimal.Numeric.Cell} {
				if c.Object == (umemory.ObjectRef{}) {
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
			for _, block := range []umemory.Block{w.Block, w.Transferred.Block()} {
				if block == (umemory.Block{}) {
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
			if c := w.Roots.Cell; c.Object != (umemory.ObjectRef{}) {
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
			for _, block := range []umemory.Block{h.Transferred.Block(), h.Numeric.Allocator[0], h.Numeric.Allocator[1], h.Numeric.Allocator[2]} {
				if block == (umemory.Block{}) {
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
			if c := h.Numeric.Cell; c.Object != (umemory.ObjectRef{}) {
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
			for _, block := range []umemory.Block{h.Transferred.Block(), h.Numeric.Allocator[0], h.Numeric.Allocator[1], h.Numeric.Allocator[2]} {
				if block == (umemory.Block{}) {
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
			if c := h.Numeric.Cell; c.Object != (umemory.ObjectRef{}) {
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
		for _, s := range []*unumeral.ArbitraryFloatRoundScope{&parent.RoundScope, &parent.RoundSqrtScope} {
			if s.Manager == parent && s.Invocation != 0 {
				if block := s.Transferred.Block(); block != (umemory.Block{}) {
					if raw[block] {
						t.Fatal("duplicate round transferred raw")
					}
					if _, e := m.Memory.Size(block); e != nil {
						t.Fatal(e)
					}
					raw[block] = true
				}
				if c := s.Numeric.Cell; c.Object != (umemory.ObjectRef{}) {
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
			if block := q.Transferred.Block(); block != (umemory.Block{}) {
				if raw[block] {
					t.Fatal("duplicate input transferred raw")
				}
				if _, e := m.Memory.Size(block); e != nil {
					t.Fatal(e)
				}
				raw[block] = true
			}
			if c := q.Numeric.Cell; c.Object != (umemory.ObjectRef{}) {
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
			if block != (umemory.Block{}) {
				if raw[block] {
					t.Fatal("duplicate transferred MPF raw root")
				}
				if _, e := m.Memory.Size(block); e != nil {
					t.Fatal("invalid transferred MPF raw root", e)
				}
				raw[block] = true
			}
			if c := child.Roots.Cell; c.Object != (umemory.ObjectRef{}) {
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

	for _, block := range external {
		if block == (umemory.Block{}) || raw[block] {
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
func binaryRat(t *testing.T, m *unumeral.BinaryRationalManager, a *unumeral.BinaryRational) *big.Rat {
	t.Helper()
	n, e := m.Z.String(&a.Num)
	serviceMust(t, e)
	v, ok := new(big.Int).SetString(n, 10)
	if !ok {
		t.Fatal(n)
	}
	return new(big.Rat).SetFrac(v, new(big.Int).Lsh(big.NewInt(1), uint(a.KField)))
}

func integerActualOwnedBytes(m *unumeral.IntegerManager) uint64 {
	cs := m.Cells.Stats()
	n := m.HeaderBytes + m.Natural.HeaderBytes + m.Memory.Stats().TotalBytes + m.Allocator.Stats().TotalBytes
	n += cs.OwnerBytes + cs.RecordBytes + cs.MetadataBytes + cs.DirectoryBytes

	return n
}
func integerPhysicalAudit(t *testing.T, m *unumeral.IntegerManager, public ...*unumeral.Int) {
	integerPhysicalAuditWithRaw(t, m, nil, public...)
}

func rationalPublicationDirectOwner(t *testing.T, synch bool) (*unumeral.RationalManager, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	m, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true, Windows: false}}, Synchronized: synch})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		m.Close()
		if b.Used() != 0 {
			t.Errorf("direct rational owner retained %d", b.Used())
		}
	})
	return m, b
}
func sxOwner(t *testing.T, b *arena.Budget) (*util.SexprManager, *usymbol.Manager, *unumeral.RationalService) {
	t.Helper()
	s, e := usymbol.NewManager(usymbol.Config{Budget: b, PageSize: 64, DescriptorPageSize: 2})
	sxMust(t, e)
	q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 2}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{Budget: b, PageSize: 2}}}})
	sxMust(t, e)
	m, e := util.NewSexprManager(util.SexprConfig{Budget: b, PageSize: 2, Symbols: s, Rationals: q})
	sxMust(t, e)
	t.Cleanup(func() {
		sxMust(t, m.Close())
		sxMust(t, q.Close())
		sxMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatal("owner bytes", b.Used())
		}
	})
	return m, s, q
}
func internSymbol(t *testing.T, m *usymbol.Manager, text string) usymbol.Symbol {
	t.Helper()
	s, e := m.Intern(text)
	if e != nil {
		t.Fatal(e)
	}
	return s
}

func sxNumber(t *testing.T, q *unumeral.RationalService, text string) unumeral.OwnedRational {
	v, e := q.NewValue()
	sxMust(t, e)
	sxMust(t, q.SetString(&v, text))
	return v
}

func serviceAuditRationalRoots(t *testing.T, s *unumeral.RationalService, values ...*unumeral.Rational) {
	t.Helper()
	roots := []*unumeral.Int{}
	add := func(q *unumeral.Rational) { roots = append(roots, &q.Num, &q.Den) }
	for _, q := range []*unumeral.Rational{&s.ZeroField, &s.OneField, &s.MinusOneField, &s.InfZeroField.First, &s.InfZeroField.Second, &s.InfOneField.First, &s.InfOneField.Second, &s.InfMinusOneField.First, &s.InfMinusOneField.Second, &s.InfIntZeroField.First, &s.InfIntOneField.First, &s.InfIntMinusOneField.First} {
		add(q)
	}
	for i := range s.Cache {
		add(&s.Cache[i])
	}
	for _, q := range values {
		if q != nil {
			add(q)
		}
	}
	qopsAuditLiveOwnership(t, s.Manager, roots)
}
func serviceValue(t *testing.T, s *unumeral.RationalService, text string) unumeral.OwnedRational {
	t.Helper()
	v, e := s.NewValue()
	serviceMust(t, e)
	serviceMust(t, s.SetString(&v, text))
	return v
}

func qopsAuditLiveOwnership(t *testing.T, m *unumeral.RationalManager, public []*unumeral.Int) {
	t.Helper()
	roots := append([]*unumeral.Int(nil), public...)
	scratch := []*unumeral.Int{&m.NTmp, &m.AddTmp1, &m.AddTmp2, &m.AddMulTmp.Num, &m.AddMulTmp.Den, &m.LtTmp1.Num, &m.LtTmp1.Den, &m.LtTmp2.Num, &m.LtTmp2.Den}
	for _, n := range scratch {
		if m.Synchronized && n.Cell != 0 && n.Cell != m.Z.FailedRef {
			t.Fatal("local mode retained rational scratch")
		}
		if !m.Synchronized {
			roots = append(roots, n)
		}
	}
	raw := &m.RawArithmetic
	if raw.PowerActive {
		roots = append(roots, &raw.Power.Num, &raw.Power.Den)
	}
	if raw.AddActive {
		roots = append(roots, &raw.AddOne, &raw.AddTwo)
	}
	if raw.NormActive {
		roots = append(roots, &raw.Norm)
	}
	if raw.DivActive {
		roots = append(roots, &raw.Div)
	}

	if raw.CompareActive {
		roots = append(roots, &raw.CompareOne.Num, &raw.CompareOne.Den, &raw.CompareTwo.Num, &raw.CompareTwo.Den)
	}
	integerPhysicalAudit(t, m.Z, roots...)
}

type numericWarmSnapshot struct {
	used, owned, size, count uint64
	runtime                  umemory.RuntimeStats
	allocator                umemory.AllocatorStats
	cells                    uint32
}

func qopsMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}

func mpzPublicationOwnerProfile(t *testing.T, profile umemory.SmallAllocatorProfile) (*unumeral.IntegerManager, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 20)
	m, err := unumeral.NewIntegerManager(unumeral.IntegerConfig{
		Cells:   arena.Config{PageSize: 2, Budget: b},
		Profile: profile,
		Words:   arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		b.FailAfter(-1)
		m.Close()
		if b.Used() != 0 {
			t.Errorf("publication fixture leaked %d bytes", b.Used())
		}
	})
	return m, b
}
