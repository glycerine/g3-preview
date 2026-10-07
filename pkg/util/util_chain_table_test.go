package util

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func chainDummy(v int32) uint32 { return uint32(v % 2) }
func chainNew(t *testing.T, dummy bool) *Int32ChainTable {
	t.Helper()
	c := Int32ChainConfig{}
	if dummy {
		c.Hash = chainDummy
	}
	h, e := NewInt32ChainTable(c)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(h.Close)
	return h
}
func chainInsert(t *testing.T, h *Int32ChainTable, v int32) {
	t.Helper()
	if e := h.Insert(v); e != nil {
		t.Fatal(e)
	}
}
func chainHas(t *testing.T, h *Int32ChainTable, v int32, want bool) {
	t.Helper()
	got, e := h.Contains(v)
	if e != nil || got != want {
		t.Fatalf("contains %d: %v %v want %v", v, got, e, want)
	}
}
func chainSize(t *testing.T, h *Int32ChainTable, size, used uint32) {
	t.Helper()
	s := h.Stats()
	if s.Size != size || s.UsedSlots != used {
		t.Fatalf("stats %+v want size %d used %d", s, size, used)
	}
}
func chainValues(t *testing.T, h *Int32ChainTable) []int32 {
	t.Helper()
	var out []int32
	for it := h.Begin(); !it.AtEnd(); {
		v, e := it.Entry()
		if e != nil {
			t.Fatal(e)
		}
		out = append(out, v)
		if e = it.Next(); e != nil {
			t.Fatal(e)
		}
	}
	return out
}

func TestChainSourceTst1(t *testing.T) {
	h := chainNew(t, false)
	for _, v := range []int32{10, 20, 30} {
		chainInsert(t, h, v)
		chainHas(t, h, v, true)
	}
	if h.Stats().Size != 3 {
		t.Fatal(h.Stats())
	}
	if e := h.Erase(20); e != nil {
		t.Fatal(e)
	}
	chainHas(t, h, 20, false)
	if h.Stats().Size != 2 {
		t.Fatal(h.Stats())
	}
}
func TestChainSourceTst2(t *testing.T) {
	h := chainNew(t, true)
	chainInsert(t, h, 10)
	chainInsert(t, h, 12)
	if h.Stats().UsedSlots != 1 {
		t.Fatal(h.Stats())
	}
	chainInsert(t, h, 13)
	if h.Stats().UsedSlots != 2 {
		t.Fatal(h.Stats())
	}
	chainInsert(t, h, 14)
	chainSize(t, h, 4, 2)
	if e := h.Erase(12); e != nil {
		t.Fatal(e)
	}
	chainHas(t, h, 12, false)
	chainSize(t, h, 3, 2)
	for _, v := range []int32{10, 14, 13} {
		chainHas(t, h, v, true)
	}
	chainHas(t, h, 12, false)
	chainInsert(t, h, 16)
	if h.Stats().Size != 4 {
		t.Fatal(h.Stats())
	}
	chainInsert(t, h, 18)
	chainSize(t, h, 5, 2)
	if e := h.Erase(10); e != nil {
		t.Fatal(e)
	}
	chainHas(t, h, 10, false)
	chainHas(t, h, 12, false)
	for _, v := range []int32{14, 13, 16, 18} {
		chainHas(t, h, v, true)
	}
}
func TestChainSourceTst3(t *testing.T) {
	h := chainNew(t, true)
	chainInsert(t, h, 10)
	chainInsert(t, h, 12)
	chainSize(t, h, 2, 1)
	chainHas(t, h, 10, true)
	chainHas(t, h, 12, true)
	if e := h.Erase(12); e != nil {
		t.Fatal(e)
	}
	if e := h.Erase(10); e != nil {
		t.Fatal(e)
	}
	chainSize(t, h, 0, 0)
	if !h.Empty() {
		t.Fatal("not empty")
	}
	chainInsert(t, h, 10)
	chainSize(t, h, 1, 1)
	chainHas(t, h, 10, true)
}
func TestChainSourceTst4(t *testing.T) {
	// Source libc rand sequence is not specified. These finite operation counts
	// exactly match its registrations; the independent oracle is a bool array.
	for _, c := range []struct {
		n, limit uint32
		dummy    bool
	}{{1000, 10, true}, {10000, 10, true}, {50000, 1000, false}} {
		h := chainNew(t, c.dummy)
		r := NewRandom(719)
		oracle := make([]bool, c.limit)
		var size uint32
		for i := uint32(0); i < c.n; i++ {
			v := r.Next() % c.limit
			if r.Next()%3 == 2 {
				if e := h.Erase(int32(v)); e != nil {
					t.Fatal(e)
				}
				if oracle[v] {
					size--
				}
				oracle[v] = false
				chainHas(t, h, int32(v), false)
			} else {
				chainInsert(t, h, int32(v))
				if !oracle[v] {
					size++
				}
				oracle[v] = true
				chainHas(t, h, int32(v), true)
			}
			if h.Stats().Size != size || h.Empty() != (size == 0) {
				t.Fatalf("operation %d size", i)
			}
		}
		for v, exists := range oracle {
			if exists {
				chainHas(t, h, int32(v), true)
			}
		}
		seen := make([]bool, c.limit)
		for _, v := range chainValues(t, h) {
			if v < 0 || uint32(v) >= c.limit || !oracle[v] || seen[v] {
				t.Fatal("iterator", v)
			}
			seen[v] = true
		}
		if !reflect.DeepEqual(seen, oracle) {
			t.Fatal("iteration differs")
		}
	}
}
func TestChainSourceTst5(t *testing.T) {
	h := chainNew(t, true)
	for _, v := range []int32{4, 9, 8, 1} {
		chainInsert(t, h, v)
	}
	if e := h.Erase(1); e != nil {
		t.Fatal(e)
	}
	for _, v := range []int32{7, 1, 2} {
		chainInsert(t, h, v)
	}
	for _, v := range []int32{4, 9, 8, 7, 1, 2} {
		chainHas(t, h, v, true)
	}
}
func TestChainSourceTst6(t *testing.T) {
	h, e := NewInt32ChainMap(Int32ChainConfig{})
	if e != nil {
		t.Fatal(e)
	}
	defer h.Close()
	if e = h.Insert(10, 4); e != nil {
		t.Fatal(e)
	}
	ok, e := h.Contains(10)
	if e != nil || !ok {
		t.Fatal(ok, e)
	}
	v, ok, e := h.Find(10)
	if e != nil || !ok || v != 4 {
		t.Fatal(v, ok, e)
	}
}

func TestChainOrderGrowthAndCounters(t *testing.T) {
	h := chainNew(t, true)
	for _, v := range []int32{10, 12, 14} {
		chainInsert(t, h, v)
	}
	if got := chainValues(t, h); !reflect.DeepEqual(got, []int32{14, 12, 10}) {
		t.Fatal(got)
	}
	if s := h.Stats(); s.Capacity != 10 || s.Collisions != 3 {
		t.Fatal(s)
	}
	// All cellar cells are occupied: duplicate insertion grows BEFORE lookup.
	chainInsert(t, h, 14)
	if s := h.Stats(); s.Slots != 16 || s.Capacity != 20 || s.Collisions != 5 {
		t.Fatal(s)
	}
	if got := chainValues(t, h); !reflect.DeepEqual(got, []int32{10, 12, 14}) {
		t.Fatal(got)
	}
	chainHas(t, h, 14, true)
	chainHas(t, h, 16, false)
	if h.Stats().Collisions != 10 {
		t.Fatal(h.Stats())
	}
	if e := h.Erase(10); e != nil {
		t.Fatal(e)
	}
	if got := chainValues(t, h); !reflect.DeepEqual(got, []int32{12, 14}) {
		t.Fatal(got)
	}
	chainInsert(t, h, 16)
	if got := chainValues(t, h); !reflect.DeepEqual(got, []int32{16, 12, 14}) {
		t.Fatal(got)
	}
}
func TestChainSignedAndMapEntryRoutes(t *testing.T) {
	h := chainNew(t, true)
	for _, v := range []int32{-3, -2, -1, 0, 1, 2, math.MinInt32, math.MaxInt32} {
		chainInsert(t, h, v)
		chainHas(t, h, v, true)
	}
	m, e := NewInt32ChainMap(Int32ChainConfig{Hash: chainDummy})
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	if e = m.Insert(2, 7); e != nil {
		t.Fatal(e)
	}
	r, e := m.InsertIfNotThere(2, 99)
	if e != nil {
		t.Fatal(e)
	}
	p, e := r.Entry()
	if e != nil || p != (Int32ChainPair{2, 7}) {
		t.Fatal(p, e)
	}
	if e = r.SetValue(8); e != nil {
		t.Fatal(e)
	}
	if e = m.Insert(2, 9); e != nil {
		t.Fatal(e)
	}
	v, ok, e := m.Find(2)
	if e != nil || !ok || v != 9 {
		t.Fatal(v, ok, e)
	}
	r, ok, e = m.FindCore(2)
	if e != nil || !ok {
		t.Fatal(e)
	}
	if e = m.Erase(2); e != nil {
		t.Fatal(e)
	}
	if _, e = r.Entry(); !errors.Is(e, ErrChainView) {
		t.Fatal(e)
	}
	h2 := chainNew(t, false)
	added, e := h2.InsertIfNotThere2(4)
	if e != nil || !added {
		t.Fatal(e)
	}
	added, e = h2.InsertIfNotThere2(4)
	if e != nil || added {
		t.Fatal(e)
	}
	v, ok, e = h2.Find(4)
	if e != nil || !ok || v != 4 {
		t.Fatal(v, ok, e)
	}
	v = 19
	if ok, e = h2.FindInto(5, &v); e != nil || ok || v != 19 {
		t.Fatal(v, ok, e)
	}
}
func TestChainLifecycleAndViews(t *testing.T) {
	h := chainNew(t, true)
	chainInsert(t, h, 2)
	r, ok, e := h.FindCore(2)
	if e != nil || !ok {
		t.Fatal(e)
	}
	it := h.Begin()
	old, e := it.PostNext()
	if e != nil || !it.Equal(h.End()) {
		t.Fatal(e)
	}
	v, e := old.Entry()
	if e != nil || v != 2 {
		t.Fatal(v, e)
	}
	if e = h.Reset(); e != nil {
		t.Fatal(e)
	}
	if _, e = r.Entry(); !errors.Is(e, ErrChainView) {
		t.Fatal(e)
	}
	if _, e = old.Entry(); !errors.Is(e, ErrChainView) {
		t.Fatal(e)
	}
	if h.Stats().Capacity != 10 {
		t.Fatal(h.Stats())
	}
	before := h.Stats().Collisions
	if e = h.Finalize(); e != nil {
		t.Fatal(e)
	}
	if h.Stats().Collisions != before {
		t.Fatal("counter reset")
	}
	h.Close()
	h.Close()
	if e = h.Insert(1); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
func TestChainBudgetsFailureAndSwap(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	h, e := NewInt32ChainTable(Int32ChainConfig{Budget: b, Hash: chainDummy})
	if e != nil {
		t.Fatal(e)
	}
	defer h.Close()
	s := h.Stats()
	if b.Used() != s.OwnerBytes+s.TableBytes {
		t.Fatal("accounting")
	}
	for _, v := range []int32{2, 4, 6} {
		chainInsert(t, h, v)
	}
	baseline := b.Used()
	ref, _, e := h.FindCore(2)
	if e != nil {
		t.Fatal(e)
	}
	b.FailAfter(0)
	if e = h.Insert(6); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if b.Used() != baseline || h.Stats().Capacity != 10 {
		t.Fatal("failed growth published")
	}
	if _, e = ref.Entry(); e != nil {
		t.Fatal(e)
	}
	if e = h.Finalize(); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if b.Used() != baseline || h.Stats().Size != 3 {
		t.Fatal("failed finalize")
	}
	b.FailAfter(-1)
	b2 := arena.NewBudget(1 << 20)
	other, e := NewInt32ChainTable(Int32ChainConfig{Budget: b2, Hash: chainDummy, InitialSlots: 16, InitialCellar: 4})
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	chainInsert(t, other, 9)
	u1, u2 := b.Used(), b2.Used()
	b.FailAfter(0)
	if e = h.Swap(other); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if b.Used() != u1 || b2.Used() != u2 {
		t.Fatal("swap refund")
	}
	b.FailAfter(-1)
	if e = h.Swap(other); e != nil {
		t.Fatal(e)
	}
	chainHas(t, h, 9, true)
	chainSize(t, other, 3, 1)
	if b.Used() != h.Stats().OwnerBytes+h.Stats().TableBytes || b2.Used() != other.Stats().OwnerBytes+other.Stats().TableBytes {
		t.Fatal("swap charge")
	}
	if _, e = ref.Entry(); !errors.Is(e, ErrChainView) {
		t.Fatal(e)
	}
	h.Close()
	other.Close()
	if b.Used() != 0 || b2.Used() != 0 {
		t.Fatal("close leak")
	}
}
func TestChainSwapTransfersOnlyDeltaAndKeepsHash(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	a, e := NewInt32ChainTable(Int32ChainConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	d, e := NewInt32ChainTable(Int32ChainConfig{Budget: b, Hash: chainDummy})
	if e != nil {
		t.Fatal(e)
	}
	defer d.Close()
	chainInsert(t, a, 7)
	chainInsert(t, d, 4)
	used := b.Used()
	b.FailAfter(0)
	if e = a.Swap(d); e != nil {
		t.Fatal(e)
	}
	if b.Used() != used {
		t.Fatal("same-budget reserve")
	}
	chainHas(t, a, 4, false)
	chainHas(t, d, 7, false)
	if !reflect.DeepEqual(chainValues(t, a), []int32{4}) || !reflect.DeepEqual(chainValues(t, d), []int32{7}) {
		t.Fatal("payload")
	}
	b1 := arena.NewBudget(1 << 20)
	x, e := NewInt32ChainTable(Int32ChainConfig{Budget: b1})
	if e != nil {
		t.Fatal(e)
	}
	defer x.Close()
	exact := arena.NewBudget(x.Stats().OwnerBytes + x.Stats().TableBytes)
	y, e := NewInt32ChainTable(Int32ChainConfig{Budget: exact})
	if e != nil {
		t.Fatal(e)
	}
	defer y.Close()
	if e = x.Swap(y); e != nil {
		t.Fatal("equal-size reserves", e)
	}
	z, e := NewInt32ChainTable(Int32ChainConfig{Budget: b1, InitialSlots: 16, InitialCellar: 4})
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	before := exact.Used()
	if e = y.Swap(z); !errors.Is(e, arena.ErrBudget) {
		t.Fatal(e)
	}
	if exact.Used() != before || y.Stats().Capacity != 10 || z.Stats().Capacity != 20 {
		t.Fatal("quota failure publication")
	}
}
func TestChainEpochAndAllocationPlateau(t *testing.T) {
	h := chainNew(t, false)
	chainInsert(t, h, 1)
	h.core.epoch = math.MaxUint64
	if e := h.Insert(2); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	chainHas(t, h, 2, false)
	if e := h.Erase(1); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	chainHas(t, h, 1, true)
	if e := h.Insert(1); e != nil {
		t.Fatal(e)
	}
	if e := h.Reset(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	h.Close()
	h = chainNew(t, false)
	chainInsert(t, h, 1)
	if got := testing.AllocsPerRun(100, func() {
		if e := h.Insert(1); e != nil {
			panic(e)
		}
		if _, e := h.Contains(1); e != nil {
			panic(e)
		}
		it := h.Begin()
		if _, e := it.Entry(); e != nil {
			panic(e)
		}
	}); got != 0 {
		t.Fatal(got)
	}
}
func TestChainHashActionOrderAndEndEquality(t *testing.T) {
	var calls []int32
	h, e := NewInt32ChainTable(Int32ChainConfig{Hash: func(v int32) uint32 { calls = append(calls, v); return chainDummy(v) }})
	if e != nil {
		t.Fatal(e)
	}
	defer h.Close()
	chainInsert(t, h, 2)
	if !reflect.DeepEqual(calls, []int32{2}) {
		t.Fatal("insert hash", calls)
	}
	calls = nil
	chainInsert(t, h, 4)
	chainInsert(t, h, 6)
	calls = nil
	chainInsert(t, h, 6)
	if !reflect.DeepEqual(calls, []int32{6, 4, 2, 6}) {
		t.Fatal("rehash before incoming hash", calls)
	}
	calls = nil
	if e = h.Erase(4); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(calls, []int32{4}) {
		t.Fatal("erase hash", calls)
	}
	other := chainNew(t, false)
	var zero Int32ChainIterator
	if !h.End().Equal(other.End()) || !zero.Equal(h.End()) || !zero.AtEnd() {
		t.Fatal("source null end equality")
	}
}
func TestChainRepeatedCellarGrowthAndFailurePrefixes(t *testing.T) {
	// Mutable source entry references can move keys to a different bucket.
	// Expansion copies actual values and repairs this finite configuration.
	for fail := int64(0); fail < 4; fail++ {
		b := arena.NewBudget(1 << 20)
		h, e := NewInt32ChainTable(Int32ChainConfig{Budget: b})
		if e != nil {
			t.Fatal(e)
		}
		for v := int32(0); v < 10; v++ {
			chainInsert(t, h, v)
		}
		refs := make([]Int32ChainEntryRef, 10)
		for v := int32(0); v < 10; v++ {
			r, ok, e := h.FindCore(v)
			if e != nil || !ok {
				t.Fatal(e)
			}
			refs[v] = r
		}
		for v, r := range refs {
			if e := r.Set(int32(v) * 16); e != nil {
				t.Fatal(e)
			}
		}
		old := chainValues(t, h)
		baseline := b.Used()
		b.FailAfter(fail)
		e = h.Insert(0)
		if fail < 3 {
			if !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != baseline || h.Stats().Capacity != 10 || !reflect.DeepEqual(chainValues(t, h), old) {
				t.Fatal("retry failure", fail, e, h.Stats())
			}
		} else {
			if e != nil || h.Stats().Slots != 16 || h.Stats().Capacity != 32 {
				t.Fatal("retry success", e, h.Stats())
			}
			want := []int32{112, 96, 80, 64, 48, 32, 16, 144, 0, 128}
			if !reflect.DeepEqual(chainValues(t, h), want) {
				t.Fatal(chainValues(t, h))
			}
		}
		h.Close()
		if b.Used() != 0 {
			t.Fatal("retry leaked", fail, b.Used())
		}
	}
}
func TestChainMapIterationAndLifecycleSurface(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	m, e := NewInt32ChainMap(Int32ChainConfig{Budget: b, Hash: chainDummy})
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	for _, p := range []Int32ChainPair{{2, 20}, {4, 40}, {1, 10}} {
		if e = m.Insert(p.Key, p.Value); e != nil {
			t.Fatal(e)
		}
	}
	if m.Size() != 3 || m.Capacity() != 10 || m.UsedSlots() != 2 || m.Collisions() != 1 || m.Empty() || !m.CheckInvariant() {
		t.Fatal(m.Stats())
	}
	var got []Int32ChainPair
	for it := m.Begin(); !it.Equal(m.End()); {
		p, e := it.Entry()
		if e != nil {
			t.Fatal(e)
		}
		got = append(got, p)
		r, e := it.Ref()
		if e != nil {
			t.Fatal(e)
		}
		if e = r.SetValue(p.Value + 1); e != nil {
			t.Fatal(e)
		}
		if _, e = it.PostNext(); e != nil {
			t.Fatal(e)
		}
	}
	if !reflect.DeepEqual(got, []Int32ChainPair{{4, 40}, {2, 20}, {1, 10}}) {
		t.Fatal(got)
	}
	v := int32(77)
	ok, e := m.FindInto(99, &v)
	if e != nil || ok || v != 77 {
		t.Fatal(v, ok, e)
	}
	ok, e = m.FindInto(2, &v)
	if e != nil || !ok || v != 21 {
		t.Fatal(v, ok, e)
	}
	if e = m.Reset(); e != nil {
		t.Fatal(e)
	}
	if !m.Empty() || !m.CheckInvariant() {
		t.Fatal(m.Stats())
	}
	if e = m.Finalize(); e != nil {
		t.Fatal(e)
	}
	other, e := NewInt32ChainMap(Int32ChainConfig{Budget: b, Hash: chainDummy})
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	if e = other.Insert(-1, -2); e != nil {
		t.Fatal(e)
	}
	b.FailAfter(0)
	if e = m.Swap(other); e != nil {
		t.Fatal(e)
	}
	p, ok, e := m.Find(-1)
	if e != nil || !ok || p != -2 {
		t.Fatal(p, ok, e)
	}
	h := chainNew(t, false)
	if h.Size() != 0 || h.Capacity() != 10 || h.UsedSlots() != 0 || h.Collisions() != 0 || !h.CheckInvariant() {
		t.Fatal(h.Stats())
	}
}
func TestChainConstructorAndEmptyResetFailureActions(t *testing.T) {
	for fail := int64(0); fail < 2; fail++ {
		b := arena.NewBudget(1 << 20)
		b.FailAfter(fail)
		h, e := NewInt32ChainTable(Int32ChainConfig{Budget: b})
		if !errors.Is(e, arena.ErrInjectedFailure) || h != nil || b.Used() != 0 {
			t.Fatal(fail, e, b.Used())
		}
		b.FailAfter(fail)
		m, e := NewInt32ChainMap(Int32ChainConfig{Budget: b})
		if !errors.Is(e, arena.ErrInjectedFailure) || m != nil || b.Used() != 0 {
			t.Fatal(fail, e, b.Used())
		}
	}
	h := chainNew(t, false)
	h.core.epoch = math.MaxUint64
	if e := h.Reset(); e != nil {
		t.Fatal("empty reset", e)
	}
	if e := h.Finalize(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	other := chainNew(t, false)
	if e := h.Swap(other); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	for _, c := range []Int32ChainConfig{{InitialSlots: 3}, {InitialSlots: 1 << 31, InitialCellar: 1 << 31}} {
		b := arena.NewBudget(1 << 20)
		c.Budget = b
		if h, e := NewInt32ChainTable(c); e == nil || h != nil || b.Used() != 0 {
			t.Fatal(e, b.Used())
		}
	}
}
func TestChainFailedCopyPublishesUsedSlotsPrefix(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	h, e := NewInt32ChainTable(Int32ChainConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	defer h.Close()
	for v := int32(0); v < 10; v++ {
		chainInsert(t, h, v)
	}
	// Old bucket0 has [8,0], bucket1 [9,1], then bucket2 [2],
	// bucket3 [3], followed by four other occupied buckets. The first six
	// values now hash to target bucket0: target cellar4 overflows on bucket3,
	// before those later buckets are visited. Source m_used_slots becomes1.
	for _, p := range []Int32ChainPair{{8, 16}, {9, 32}, {1, 48}, {2, 64}, {3, 80}} {
		r, ok, e := h.FindCore(p.Key)
		if e != nil || !ok {
			t.Fatal(e)
		}
		if e = r.Set(p.Value); e != nil {
			t.Fatal(e)
		}
	}
	graph := append([]chainCell(nil), h.core.table...)
	values := chainValues(t, h)
	before := b.Used()
	epoch := h.core.epoch
	oldNext, oldFree := h.core.next, h.core.free
	if h.UsedSlots() != 8 {
		t.Fatal(h.UsedSlots())
	}
	b.FailAfter(1)
	if e = h.Insert(0); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if h.UsedSlots() != 1 {
		t.Fatalf("source failed-copy used-slots prefix: got %d want1", h.UsedSlots())
	}
	if h.Size() != 10 || h.Capacity() != 10 || h.core.slots != 8 || h.core.epoch != epoch || h.core.next != oldNext || h.core.free != oldFree || b.Used() != before || !reflect.DeepEqual(h.core.table, graph) || !reflect.DeepEqual(chainValues(t, h), values) {
		t.Fatal("failed expansion changed physical graph or entries", h.Stats())
	}
	h.Close()
	if b.Used() != 0 {
		t.Fatal("close leak", b.Used())
	}
}
