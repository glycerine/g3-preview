package g3

import (
	"errors"
	"math"
	"math/rand"
	"reflect"
	"sync"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func symbolIntTableTest(t *testing.T, m *SymbolManager, b *arena.Budget) *SymbolInt32Table {
	t.Helper()
	h, e := NewSymbolInt32Table(m, SymbolInt32TableConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := h.Close(); e != nil {
			t.Fatal(e)
		}
	})
	return h
}

func TestSymbolInt32TableColdMutex(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	h := symbolIntTableTest(t, m, nil)
	var workers sync.WaitGroup
	for worker := uint32(0); worker < 4; worker++ {
		workers.Add(1)
		go func(base uint32) {
			defer workers.Done()
			for i := uint32(0); i < 128; i++ {
				k := NumericalSymbol(base*128 + i)
				if e := h.Insert(k, int32(i)); e != nil {
					t.Error(e)
					return
				}
				if v, ok, e := h.Find(k); e != nil || !ok || v != int32(i) {
					t.Error(v, ok, e)
					return
				}
			}
		}(worker)
	}
	workers.Wait()
	if h.Stats().Size != 512 {
		t.Fatal(h.Stats())
	}
}
func TestSymbolInt32TableSourceAssertions(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	h := symbolIntTableTest(t, m, nil)
	foo := internSymbol(t, m, "foo")
	boo := internSymbol(t, m, "boo")
	insert := func(k Symbol, v int32) {
		t.Helper()
		if e := h.Insert(k, v); e != nil {
			t.Fatal(e)
		}
	}
	contains := func(k Symbol, w bool) {
		t.Helper()
		v, e := h.Contains(k)
		if e != nil || v != w {
			t.Fatal(v, e)
		}
	}
	find := func(k Symbol, w int32) {
		t.Helper()
		v, ok, e := h.Find(k)
		if e != nil || !ok || v != w {
			t.Fatal(v, ok, e)
		}
	}
	insert(foo, 35)
	contains(foo, true)
	contains(boo, false)
	if e := h.BeginScope(); e != nil {
		t.Fatal(e)
	}
	insert(boo, 20)
	contains(boo, true)
	find(boo, 20)
	find(foo, 35)
	insert(foo, 100)
	find(foo, 100)
	if e := h.EndScope(); e != nil {
		t.Fatal(e)
	}
	find(foo, 35)
	contains(boo, false)
	if e := h.Reset(); e != nil {
		t.Fatal(e)
	}
	contains(boo, false)
	contains(foo, false)
}
func TestSymbolInt32TableIndependentScopes(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	h := symbolIntTableTest(t, m, nil)
	rng := rand.New(rand.NewSource(9321))
	model := map[Symbol]int32{}
	var scopes []map[Symbol]int32
	keys := []Symbol{internSymbol(t, m, "foo"), internSymbol(t, m, "boo")}
	for i := uint32(0); i < 40; i++ {
		keys = append(keys, NumericalSymbol(i*8))
	}
	for step := 0; step < 2000; step++ {
		switch rng.Intn(7) {
		case 0:
			if len(scopes) < 8 {
				copy := map[Symbol]int32{}
				for k, v := range model {
					copy[k] = v
				}
				scopes = append(scopes, copy)
				if e := h.BeginScope(); e != nil {
					t.Fatal(e)
				}
			}
		case 1:
			if len(scopes) > 0 {
				model = scopes[len(scopes)-1]
				scopes = scopes[:len(scopes)-1]
				if e := h.EndScope(); e != nil {
					t.Fatal(e)
				}
			}
		case 2:
			if e := h.Reset(); e != nil {
				t.Fatal(e)
			}
			model = map[Symbol]int32{}
			scopes = nil
		default:
			k := keys[rng.Intn(len(keys))]
			v := int32(rng.Uint32())
			if e := h.Insert(k, v); e != nil {
				t.Fatal(e)
			}
			model[k] = v
		}
		if h.ScopeLevel() != uint32(len(scopes)) {
			t.Fatal("scope depth")
		}
		for _, k := range keys {
			v, ok, e := h.Find(k)
			w, present := model[k]
			if e != nil || ok != present || (ok && v != w) {
				t.Fatal(step, k, v, ok, w, present, e)
			}
		}
	}
	out := int32(71)
	if ok, e := h.FindInto(NumericalSymbol(9999), &out); ok || e != nil || out != 71 {
		t.Fatal("miss output")
	}
}
func TestSymbolInt32TableHashPolicyAndTrail(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	h := symbolIntTableTest(t, m, nil)
	for i := uint32(0); i < 7; i++ {
		if e := h.Insert(NumericalSymbol(i*8), int32(i)); e != nil {
			t.Fatal(e)
		}
	}
	if h.Stats().Capacity != 8 {
		t.Fatal("strict source threshold")
	}
	if e := h.BeginScope(); e != nil {
		t.Fatal(e)
	}
	if e := h.Insert(NumericalSymbol(0), 99); e != nil {
		t.Fatal(e)
	}
	if h.Stats().Capacity != 8 || h.trail[0].key.IsMarked() {
		t.Fatal("scoped existing bypasses growth")
	}
	if e := h.Insert(NumericalSymbol(56), 77); e != nil {
		t.Fatal(e)
	}
	if h.Stats().Capacity != 16 || !h.trail[1].key.IsMarked() {
		t.Fatal("marked new key")
	}
	if e := h.EndScope(); e != nil {
		t.Fatal(e)
	}
	if h.ScopeLevel() != 0 || len(h.trail) != 0 {
		t.Fatal("reverse undo limits")
	}
	v, ok, e := h.Find(NumericalSymbol(0))
	if e != nil || !ok || v != 0 {
		t.Fatal(v, ok, e)
	}
	// An unscoped duplicate grows before lookup at the source threshold.
	q := symbolIntTableTest(t, m, nil)
	for i := uint32(0); i < 7; i++ {
		q.Insert(NumericalSymbol(i), 1)
	}
	q.Insert(NumericalSymbol(0), 2)
	if q.Stats().Capacity != 16 {
		t.Fatal("duplicate prepare")
	}
	if typ := reflect.TypeOf(symbolInt32Entry{}); arena.ValidateRecordType(typ) != nil || typ.NumField() != 2 {
		t.Fatal("scalar key/data only, no hash field")
	}
}
func TestSymbolInt32TableAppendAndSentinels(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	a := symbolIntTableTest(t, m, nil)
	b := symbolIntTableTest(t, m, nil)
	a.Insert(NumericalSymbol(8), 10)
	b.Insert(NumericalSymbol(8), 20)
	b.Insert(NumericalSymbol(0), 30)
	a.BeginScope()
	if e := a.Append(b); e != nil {
		t.Fatal(e)
	}
	dom, e := a.AppendDom([]Symbol{NumericalSymbol(999)})
	if e != nil {
		t.Fatal(e)
	}
	ran, e := a.AppendRange([]int32{-1})
	if e != nil || len(dom) != 3 || len(ran) != 3 || ran[0] != -1 || dom[0] != NumericalSymbol(999) {
		t.Fatal(dom, ran, e)
	}
	for i := 1; i < len(dom); i++ {
		v, ok, e := a.Find(dom[i])
		if e != nil || !ok || v != ran[i] {
			t.Fatal("zipped bucket order")
		}
	}
	if e := a.Append(a); e != nil {
		t.Fatal(e)
	}
	if e := a.EndScope(); e != nil {
		t.Fatal(e)
	}
	v, _, _ := a.Find(NumericalSymbol(8))
	if v != 10 {
		t.Fatal(v)
	}
	foreign := symbolManagerTest(t, nil, false)
	s := internSymbol(t, foreign, "foreign")
	marked, _ := NumericalSymbol(1).mark()
	for _, k := range []Symbol{{}, DummySymbol(), marked, s} {
		if e := a.Insert(k, 1); e == nil {
			t.Fatal("invalid key accepted")
		}
	}
	if e := a.EndScope(); !errors.Is(e, ErrSymbolTableScope) {
		t.Fatal(e)
	}
	q := symbolIntTableTest(t, m, nil)
	for i := uint32(0); i < 7; i++ {
		q.Insert(NumericalSymbol(i), 1)
	}
	if e := q.Append(q); !errors.Is(e, ErrSymbolTablePrecondition) || q.Stats().Capacity != 8 {
		t.Fatal("invalidating self append", e)
	}
	if e := m.Close(); e != nil {
		t.Fatal(e)
	}
	if _, _, e := a.Find(NumericalSymbol(8)); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
func TestSymbolInt32TableBudgetFailureAndWarm(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	for n := int64(0); n < 8; n++ {
		b := arena.NewBudget(math.MaxUint64)
		b.FailAfter(n)
		h, e := NewSymbolInt32Table(m, SymbolInt32TableConfig{Budget: b})
		if e == nil {
			h.Close()
		}
		if b.Used() != 0 {
			t.Fatal("constructor unwind", n, b.Used())
		}
	}
	b := arena.NewBudget(math.MaxUint64)
	h := symbolIntTableTest(t, m, b)
	h.BeginScope()
	h.Insert(NumericalSymbol(1), 2)
	h.EndScope()
	allocs := testing.AllocsPerRun(100, func() {
		if e := h.BeginScope(); e != nil {
			panic(e)
		}
		if e := h.Insert(NumericalSymbol(1), 3); e != nil {
			panic(e)
		}
		if e := h.EndScope(); e != nil {
			panic(e)
		}
		h.Find(NumericalSymbol(1))
	})
	if allocs != 0 {
		t.Fatal("warm allocations", allocs)
	}
	for i := uint32(0); i < 7; i++ {
		h.Insert(NumericalSymbol(i), int32(i))
	}
	h.BeginScope()
	before := h.Stats()
	used := b.Used()
	b.FailAfter(0)
	if e := h.Insert(NumericalSymbol(99), 9); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if h.Stats() != before || b.Used() != used {
		t.Fatal("insert failure mutation")
	}
	b.FailAfter(-1)
	if e := h.Close(); e != nil || b.Used() != 0 {
		t.Fatal("close", e, b.Used())
	}
}

// Independent finite source bucket model, with integer sentinel states. Hashes
// are numerical key values; it does not use any table production helper.
type symbolBucketOracle struct {
	keys          []int64
	values        []int32
	used, deleted int
}

func newSymbolBucketOracle(n int) *symbolBucketOracle {
	o := &symbolBucketOracle{keys: make([]int64, n), values: make([]int32, n)}
	for i := range o.keys {
		o.keys[i] = -1
	}
	return o
}
func (o *symbolBucketOracle) insert(k int64, v int32) {
	if (o.used+o.deleted)*4 > len(o.keys)*3 {
		o.rehash(len(o.keys) * 2)
	}
	deleted := -1
	for step := 0; step < len(o.keys); step++ {
		i := (int(k) + step) % len(o.keys)
		switch o.keys[i] {
		case -2:
			deleted = i
		case -1:
			if deleted >= 0 {
				i = deleted
				o.deleted--
			}
			o.keys[i] = k
			o.values[i] = v
			o.used++
			return
		default:
			if o.keys[i] == k {
				o.values[i] = v
				return
			}
		}
	}
	panic("oracle full")
}
func (o *symbolBucketOracle) rehash(n int) {
	p := newSymbolBucketOracle(n)
	for i, k := range o.keys {
		if k >= 0 {
			p.insert(k, o.values[i])
		}
	}
	*o = *p
}
func (o *symbolBucketOracle) erase(k int64) {
	for step := 0; step < len(o.keys); step++ {
		i := (int(k) + step) % len(o.keys)
		if o.keys[i] == -1 {
			return
		}
		if o.keys[i] == k {
			o.used--
			if o.keys[(i+1)%len(o.keys)] == -1 {
				o.keys[i] = -1
			} else {
				o.keys[i] = -2
				o.deleted++
				if o.deleted > o.used && o.deleted > 64 {
					o.rehash(len(o.keys))
				}
			}
			return
		}
	}
}
func checkSymbolBuckets(t *testing.T, h *SymbolInt32Table, o *symbolBucketOracle) {
	t.Helper()
	if len(h.table) != len(o.keys) || int(h.size) != o.used || int(h.deleted) != o.deleted {
		t.Fatal("bucket totals", h.Stats(), o)
	}
	for i, e := range h.table {
		k := int64(-1)
		if e.key == DummySymbol() {
			k = -2
		} else if e.key != (Symbol{}) {
			k = int64(e.key.value)
		}
		if k != o.keys[i] || e.data != o.values[i] {
			t.Fatalf("bucket %d: %v/%d want %d/%d", i, e.key, e.data, o.keys[i], o.values[i])
		}
	}
}
func TestSymbolInt32TablePhysicalBucketsAndReset(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	h := symbolIntTableTest(t, m, nil)
	o := newSymbolBucketOracle(8)
	rng := rand.New(rand.NewSource(8724))
	for step := 0; step < 2500; step++ {
		k := uint32(rng.Intn(100)) * 8
		if rng.Intn(3) == 0 {
			if e := h.erase(&h.symbolInt32State, NumericalSymbol(k)); e != nil {
				t.Fatal(e)
			}
			o.erase(int64(k))
		} else {
			v := int32(rng.Uint32())
			if e := h.Insert(NumericalSymbol(k), v); e != nil {
				t.Fatal(e)
			}
			o.insert(int64(k), v)
		}
		checkSymbolBuckets(t, h, o)
	}
	// Last tombstone in the probe path wins, and inactive scalar data survives.
	q := symbolIntTableTest(t, m, nil)
	for _, k := range []uint32{0, 8, 16, 24} {
		q.Insert(NumericalSymbol(k), int32(k+1))
	}
	q.erase(&q.symbolInt32State, NumericalSymbol(0))
	q.erase(&q.symbolInt32State, NumericalSymbol(16))
	q.Insert(NumericalSymbol(32), 99)
	if q.table[2].key != NumericalSymbol(32) || q.table[0].key != DummySymbol() || q.table[0].data != 1 {
		t.Fatal("last tombstone/source retained data", q.table)
	}
	// Source reset uses FREE slots before clearing, not resulting empty size.
	r, e := NewSymbolInt32Table(m, SymbolInt32TableConfig{InitialCapacity: 128})
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	r.Insert(NumericalSymbol(0), 42)
	r.BeginScope()
	r.Insert(NumericalSymbol(0), 43)
	if e = r.Reset(); e != nil {
		t.Fatal(e)
	}
	if r.Stats().Capacity != 64 || r.Stats().TrailCapacity != 2 || r.Stats().ScopeCapacity != 2 {
		t.Fatal(r.Stats())
	}
	if e = r.Reset(); e != nil || r.Stats().Capacity != 64 {
		t.Fatal("empty reset no shrink", e)
	}
	for i := uint32(0); i < 32; i++ {
		r.Insert(NumericalSymbol(i), 1)
	}
	r.Reset()
	if r.Stats().Capacity != 64 {
		t.Fatal("strict overhead equality must not shrink")
	}
	// A blocker after the collision cluster prevents tail frees, forcing the
	// >64 AND >size tombstone cleanup during reverse undo.
	c, e := NewSymbolInt32Table(m, SymbolInt32TableConfig{InitialCapacity: 256})
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	c.Insert(NumericalSymbol(70), 700)
	c.BeginScope()
	for i := uint32(0); i < 70; i++ {
		c.Insert(NumericalSymbol(i*256), int32(i))
	}
	if e := c.EndScope(); e != nil {
		t.Fatal(e)
	}
	if c.size != 1 || c.deleted != 0 {
		t.Fatal("cluster cleanup rollback", c.Stats())
	}
}

func TestSymbolInt32TableStagedFailureAtomicity(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	for prefix := int64(0); prefix < 4; prefix++ {
		b := arena.NewBudget(math.MaxUint64)
		h := symbolIntTableTest(t, m, b)
		for i := uint32(0); i < 7; i++ {
			h.Insert(NumericalSymbol(i*8), int32(i))
		}
		h.BeginScope()
		h.Insert(NumericalSymbol(0), 90)
		oldTable := append([]symbolInt32Entry(nil), h.table...)
		oldTrail := append([]symbolInt32Entry(nil), h.trail[:cap(h.trail)]...)
		oldLimits := append([]uint32(nil), h.limits[:cap(h.limits)]...)
		before := h.Stats()
		used := b.Used()
		b.FailAfter(prefix)
		e := h.EndScope()
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) || h.Stats() != before || b.Used() != used || !reflect.DeepEqual(h.table, oldTable) || !reflect.DeepEqual(h.trail[:cap(h.trail)], oldTrail) || !reflect.DeepEqual(h.limits[:cap(h.limits)], oldLimits) {
				t.Fatal("staged failure changed physical graph", prefix, e)
			}
		} else {
			v, ok, _ := h.Find(NumericalSymbol(0))
			if !ok || v != 0 || h.ScopeLevel() != 0 || h.Stats().Capacity != 16 {
				t.Fatal("source undo grew before duplicate", prefix)
			}
		}
		b.FailAfter(-1)
		h.Close()
		if b.Used() != 0 {
			t.Fatal("staged refund", prefix, b.Used())
		}
	}
	// A no-rehash staged Append must preserve exact tombstones/inactive data.
	b := arena.NewBudget(math.MaxUint64)
	a := symbolIntTableTest(t, m, b)
	other := symbolIntTableTest(t, m, nil)
	a.Insert(NumericalSymbol(0), 11)
	a.Insert(NumericalSymbol(8), 22)
	a.erase(&a.symbolInt32State, NumericalSymbol(0))
	other.Insert(NumericalSymbol(7), 33)
	a.BeginScope()
	a.Insert(NumericalSymbol(8), 44)
	a.Insert(NumericalSymbol(8), 44) // fill trail so Append requires preparation
	table := append([]symbolInt32Entry(nil), a.table...)
	b.FailAfter(0)
	before := a.Stats()
	used := b.Used()
	if e := a.Append(other); !errors.Is(e, arena.ErrInjectedFailure) || a.Stats() != before || b.Used() != used {
		t.Fatal("append prepare failure", e)
	}
	b.FailAfter(-1)
	if e := a.Append(other); e != nil {
		t.Fatal(e)
	}
	if a.table[0] != table[0] {
		t.Fatal("transaction clone discarded tombstone/retained payload")
	}
	if e := a.EndScope(); e != nil {
		t.Fatal(e)
	}
	v, _, _ := a.Find(NumericalSymbol(8))
	if v != 22 {
		t.Fatal(v)
	}
	a.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestSymbolInt32TableVectorCapacitiesAndBudgetPeaks(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	b := arena.NewBudget(math.MaxUint64)
	h := symbolIntTableTest(t, m, b)
	h.Insert(NumericalSymbol(1), 7)
	for i, w := range []uint32{2, 2, 3, 5, 5, 8} {
		h.BeginScope()
		h.Insert(NumericalSymbol(1), int32(i))
		s := h.Stats()
		if s.ScopeCapacity != w || s.TrailCapacity != w {
			t.Fatal("source vector growth", i, s)
		}
		if b.Used() != s.OwnerBytes+s.TableBytes+s.TrailBytes+s.ScopeBytes {
			t.Fatal("graph charge", s, b.Used())
		}
	}
	for h.ScopeLevel() > 0 {
		if e := h.EndScope(); e != nil {
			t.Fatal(e)
		}
	}
	h.Reset()
	if h.Stats().TrailCapacity != 8 || h.Stats().ScopeCapacity != 8 {
		t.Fatal("retain scalar vectors")
	}
	h.Close()
	if b.Used() != 0 {
		t.Fatal("owner release")
	}
	// Constructor budget includes the actual cold header plus backing entries.
	limit := uint64(reflect.TypeOf(SymbolInt32Table{}).Size()) + symbolInt32EntryBytes(8) - 1
	limited := arena.NewBudget(limit)
	if _, e := NewSymbolInt32Table(m, SymbolInt32TableConfig{Budget: limited}); !errors.Is(e, arena.ErrBudget) || limited.Used() != 0 {
		t.Fatal(e, limited.Used())
	}
}

func TestSymbolInt32TableCleanupFailureAndDynamicHash(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	for prefix := int64(0); prefix < 3; prefix++ {
		b := arena.NewBudget(math.MaxUint64)
		h, e := NewSymbolInt32Table(m, SymbolInt32TableConfig{InitialCapacity: 256, Budget: b})
		if e != nil {
			t.Fatal(e)
		}
		h.Insert(NumericalSymbol(70), 700)
		h.BeginScope()
		for i := uint32(0); i < 70; i++ {
			h.Insert(NumericalSymbol(i*256), int32(i))
		}
		before := h.Stats()
		original := append([]symbolInt32Entry(nil), h.table...)
		used := b.Used()
		b.FailAfter(prefix)
		e = h.EndScope()
		if prefix < 2 {
			if !errors.Is(e, arena.ErrInjectedFailure) || h.Stats() != before || b.Used() != used || !reflect.DeepEqual(h.table, original) {
				t.Fatal("cleanup failure atomicity", prefix, e)
			}
		} else if e != nil || h.size != 1 || h.deleted != 0 {
			t.Fatal("cleanup success", e, h.Stats())
		}
		b.FailAfter(-1)
		h.Close()
		if b.Used() != 0 {
			t.Fatal("cleanup graph release")
		}
	}
	h := symbolIntTableTest(t, m, nil)
	s := internSymbol(t, m, "dynamic hash")
	h.Insert(s, 45)
	old, e := m.Hash(s)
	if e != nil {
		t.Fatal(e)
	}
	// Controlled source-prefix fixture: change hash while preserving the initial
	// bucket. A separately cached entry hash would incorrectly reject the key.
	ref := arena.BytesRegionRef{Slot: arena.Slot(s.value), Generation: s.generation}
	changed := old + 8
	for i := uint32(0); i < 4; i++ {
		if e := m.region.Write(ref, i, byte(changed>>(8*i))); e != nil {
			t.Fatal(e)
		}
	}
	v, ok, e := h.Find(s)
	if e != nil || !ok || v != 45 {
		t.Fatal("hash read through", v, ok, e)
	}
	for i := uint32(0); i < 4; i++ {
		m.region.Write(ref, i, byte(old>>(8*i)))
	}
	one, e := NewSymbolInt32Table(m, SymbolInt32TableConfig{InitialCapacity: 1})
	if e != nil {
		t.Fatal("source power-of-two capacity1", e)
	}
	one.Insert(NumericalSymbol(0), 1)
	one.Insert(NumericalSymbol(1), 2)
	if one.Stats().Capacity != 2 {
		t.Fatal(one.Stats())
	}
	one.Close()
}

func TestSymbolInt32TableAppendFaultSweepAndPeak(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	source := symbolIntTableTest(t, m, nil)
	for i := uint32(0); i < 15; i++ {
		source.Insert(NumericalSymbol(i*8), int32(i+90))
	}
	failures, successes := 0, 0
	for prefix := int64(0); prefix < 16; prefix++ {
		b := arena.NewBudget(math.MaxUint64)
		h := symbolIntTableTest(t, m, b)
		for i := uint32(0); i < 7; i++ {
			h.Insert(NumericalSymbol(i*8), int32(i))
		}
		h.BeginScope()
		before := h.Stats()
		table := append([]symbolInt32Entry(nil), h.table...)
		used := b.Used()
		b.FailAfter(prefix)
		e := h.Append(source)
		if e != nil {
			failures++
			if !errors.Is(e, arena.ErrInjectedFailure) || h.Stats() != before || !reflect.DeepEqual(h.table, table) || b.Used() != used {
				t.Fatal("append failure", prefix, e)
			}
		} else {
			successes++
			for i := uint32(0); i < 15; i++ {
				v, ok, _ := h.Find(NumericalSymbol(i * 8))
				if !ok || v != int32(i+90) {
					t.Fatal("append model")
				}
			}
			b.FailAfter(-1)
			if e := h.EndScope(); e != nil {
				t.Fatal(e)
			}
			for i := uint32(0); i < 15; i++ {
				v, ok, _ := h.Find(NumericalSymbol(i * 8))
				if ok != (i < 7) || (ok && v != int32(i)) {
					t.Fatal("append reverse oracle", i, v, ok)
				}
			}
		}
		b.FailAfter(-1)
		h.Close()
		if b.Used() != 0 {
			t.Fatal("append release", prefix, b.Used())
		}
	}
	if failures == 0 || successes == 0 {
		t.Fatal("finite sweep did not cover both", failures, successes)
	}
	// The old graph and physical staging clone remain charged while growth is
	// prepared. Set a finite limit one byte below this simultaneous peak.
	header := uint64(reflect.TypeOf(SymbolInt32Table{}).Size())
	peak := header + symbolInt32EntryBytes(8)*2 + 8 + symbolInt32EntryBytes(16)
	b := arena.NewBudget(peak - 1)
	h := symbolIntTableTest(t, m, b)
	for i := uint32(0); i < 7; i++ {
		h.Insert(NumericalSymbol(i), int32(i))
	}
	h.BeginScope()
	// Nested trail growth also contributes to the simultaneous prepared peak.
	before := h.Stats()
	used := b.Used()
	if e := h.Append(source); !errors.Is(e, arena.ErrBudget) || h.Stats() != before || b.Used() != used {
		t.Fatal("staged peak not bounded", e, h.Stats(), before)
	}
	h.Close()
	if b.Used() != 0 {
		t.Fatal("peak unwind")
	}
	// Exact output order, including caller prefix, is ascending physical buckets.
	d, e := source.AppendDom([]Symbol{NumericalSymbol(999)})
	if e != nil {
		t.Fatal(e)
	}
	r, e := source.AppendRange([]int32{-9})
	if e != nil {
		t.Fatal(e)
	}
	at := 1
	for _, entry := range source.table {
		if usedKey := entry.key != (Symbol{}) && entry.key != DummySymbol(); usedKey {
			if d[at] != entry.key || r[at] != entry.data {
				t.Fatal("source bucket order")
			}
			at++
		}
	}
	if n, e := nextSymbolVectorCapacity(math.MaxUint32); !errors.Is(e, arena.ErrBounds) || n != 0 {
		t.Fatal("vector overflow", n, e)
	}
}

func TestSymbolInt32TableExactUndoPeakBudget(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	header := uint64(reflect.TypeOf(SymbolInt32Table{}).Size())
	graph := symbolInt32EntryBytes(8) + symbolInt32EntryBytes(2) + 8
	peak := header + graph*2 + symbolInt32EntryBytes(16)
	for _, limit := range []uint64{peak - 1, peak} {
		b := arena.NewBudget(limit)
		h := symbolIntTableTest(t, m, b)
		for i := uint32(0); i < 7; i++ {
			h.Insert(NumericalSymbol(i), int32(i))
		}
		h.BeginScope()
		h.Insert(NumericalSymbol(0), 99)
		if b.Used() != header+graph {
			t.Fatal("prepared fixture graph", b.Used(), graph)
		}
		before := h.Stats()
		e := h.EndScope()
		if limit < peak {
			if !errors.Is(e, arena.ErrBudget) || h.Stats() != before || b.Used() != header+graph {
				t.Fatal("one byte below peak", e)
			}
		} else if e != nil || h.ScopeLevel() != 0 {
			t.Fatal("exact peak must fit", e)
		}
		h.Close()
		if b.Used() != 0 {
			t.Fatal("peak release")
		}
	}
}

func TestSymbolInt32TablePreparedGrowthAndWarmAppend(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	for prefix := int64(0); prefix < 3; prefix++ {
		b := arena.NewBudget(math.MaxUint64)
		h := symbolIntTableTest(t, m, b)
		for i := uint32(0); i < 7; i++ {
			h.Insert(NumericalSymbol(i), int32(i))
		}
		h.BeginScope()
		h.Insert(NumericalSymbol(0), 10)
		h.Insert(NumericalSymbol(0), 11)
		before := h.Stats()
		used := b.Used()
		tablePtr := &h.table[0]
		trailPtr := &h.trail[0]
		b.FailAfter(prefix)
		e := h.Insert(NumericalSymbol(99), 99)
		if prefix < 2 {
			if !errors.Is(e, arena.ErrInjectedFailure) || h.Stats() != before || b.Used() != used || &h.table[0] != tablePtr || &h.trail[0] != trailPtr {
				t.Fatal("prepared table/trail failure", prefix, e)
			}
		} else if e != nil || h.Stats().Capacity != 16 || h.Stats().TrailCapacity != 3 {
			t.Fatal("prepared growth success", e)
		}
		b.FailAfter(-1)
		h.Close()
		if b.Used() != 0 {
			t.Fatal("growth unwind")
		}
	}
	b := arena.NewBudget(math.MaxUint64)
	a := symbolIntTableTest(t, m, b)
	other := symbolIntTableTest(t, m, nil)
	a.Insert(NumericalSymbol(1), 1)
	other.Insert(NumericalSymbol(1), 2)
	a.BeginScope()
	a.Insert(NumericalSymbol(1), 3)
	a.Insert(NumericalSymbol(1), 4)
	a.EndScope()
	b.FailAfter(0)
	if allocs := testing.AllocsPerRun(100, func() {
		if e := a.Append(other); e != nil {
			panic(e)
		}
		if e := a.BeginScope(); e != nil {
			panic(e)
		}
		if e := a.Append(other); e != nil {
			panic(e)
		}
		if e := a.Append(a); e != nil {
			panic(e)
		}
		if e := a.EndScope(); e != nil {
			panic(e)
		}
	}); allocs != 0 {
		t.Fatal("warm append", allocs)
	}
	b.FailAfter(-1)
	s := a.Stats()
	for i := 0; i < 1000; i++ {
		a.BeginScope()
		a.Insert(NumericalSymbol(1), 9)
		a.EndScope()
	}
	if a.Stats() != s {
		t.Fatal("retained plateau")
	}
}

func TestSymbolInt32TableConstructorAndSourceGuards(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	for _, n := range []uint32{3, 7, math.MaxUint32} {
		b := arena.NewBudget(math.MaxUint64)
		if h, e := NewSymbolInt32Table(m, SymbolInt32TableConfig{InitialCapacity: n, Budget: b}); e == nil || h != nil || b.Used() != 0 {
			t.Fatal("bad capacity", n, e, b.Used())
		}
	}
	if _, e := NewSymbolInt32Table(nil, SymbolInt32TableConfig{}); !errors.Is(e, ErrSymbolTablePrecondition) {
		t.Fatal(e)
	}
	a := symbolIntTableTest(t, m, nil)
	otherManager := symbolManagerTest(t, nil, false)
	other := symbolIntTableTest(t, otherManager, nil)
	other.Insert(NumericalSymbol(1), 1)
	before := a.Stats()
	if e := a.Append(other); !errors.Is(e, ErrSymbolOwner) || a.Stats() != before {
		t.Fatal("foreign append", e)
	}
	for _, k := range []Symbol{{}, DummySymbol()} {
		if _, _, e := a.Find(k); !errors.Is(e, ErrInvalidSymbol) {
			t.Fatal("sentinel lookup", e)
		}
	}
	if _, e := a.FindInto(NumericalSymbol(1), nil); !errors.Is(e, ErrSymbolTablePrecondition) {
		t.Fatal(e)
	}
	a.Close()
	if e := a.BeginScope(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if e := a.Reset(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if e := a.Close(); e != nil {
		t.Fatal(e)
	}
	m.Close()
	if _, e := NewSymbolInt32Table(m, SymbolInt32TableConfig{}); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestSymbolInt32TableHighOccupancyNoGrowthScopes(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	b := arena.NewBudget(math.MaxUint64)
	h := symbolIntTableTest(t, m, b)
	for i := uint32(0); i < 7; i++ {
		h.Insert(NumericalSymbol(i), int32(i))
	}
	h.BeginScope()
	b.FailAfter(0)
	if e := h.EndScope(); e != nil {
		t.Fatal("empty source scope only pops limit", e)
	}
	if allocs := testing.AllocsPerRun(100, func() {
		if e := h.BeginScope(); e != nil {
			panic(e)
		}
		if e := h.EndScope(); e != nil {
			panic(e)
		}
	}); allocs != 0 {
		t.Fatal("empty high occupancy scope allocations", allocs)
	}
	b.FailAfter(-1)
	// Fresh high-occupancy table; marked-only undo retains scalar trail capacity.
	q := symbolIntTableTest(t, m, b)
	q.BeginScope()
	for i := uint32(0); i < 7; i++ {
		q.Insert(NumericalSymbol(i), int32(i))
	}
	// Marked-only undo erases; no restore-insert means no growth at >75%.
	b.FailAfter(0)
	if e := q.EndScope(); e != nil {
		t.Fatal("marked-only source undo cannot grow", e)
	}
	b.FailAfter(-1)
	for i := uint32(0); i < 7; i++ {
		q.Insert(NumericalSymbol(i), int32(i))
	}
	q.BeginScope()
	b.FailAfter(0)
	if e := q.Append(q); e != nil {
		t.Fatal("scoped self-overwrite cannot grow", e)
	}
	if q.Stats().Capacity != 8 || q.Stats().TrailLength != 7 {
		t.Fatal(q.Stats())
	}
	b.FailAfter(-1)
	// This rollback does restore inserts and correctly may require staging.
	if e := q.EndScope(); e != nil {
		t.Fatal(e)
	}
	r, e := NewSymbolInt32Table(m, SymbolInt32TableConfig{InitialCapacity: 256, Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	for i := uint32(0); i < 170; i++ {
		r.Insert(NumericalSymbol(i*256), int32(i))
	}
	for i := uint32(0); i < 65; i++ {
		r.erase(&r.symbolInt32State, NumericalSymbol(i*256))
	}
	if r.deleted != 65 {
		t.Fatal("retained >64 tombstone fixture", r.Stats())
	}
	r.BeginScope()
	b.FailAfter(0)
	if e := r.EndScope(); e != nil {
		t.Fatal("empty scope cannot trigger cleanup either", e)
	}
	b.FailAfter(-1)
}
