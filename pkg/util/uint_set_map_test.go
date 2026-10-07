// Source tst5 in src/test/uint_set.cpp and finite container/lifetime coverage.
package util

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func newSetMap(t *testing.T) *UintSetMap {
	t.Helper()
	m, e := NewUintSetMap(UintSetMapConfig{})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(m.Close)
	return m
}
func setKey(values ...uint32) *UintSet {
	s := new(UintSet)
	for _, v := range values {
		s.Insert(v)
	}
	return s
}
func TestUintSetMapSourceTst5(t *testing.T) {
	m := newSetMap(t)
	s := new(UintSet)
	var out strings.Builder
	fmt.Fprintln(&out, s.Hash())
	s.Insert(1)
	fmt.Fprintln(&out, s.Hash())
	s.Insert(2)
	fmt.Fprintln(&out, s.Hash())
	s.Insert(44)
	fmt.Fprintln(&out, s.Hash())
	if e := m.Insert(s, 1); e != nil {
		t.Fatal(e)
	}
	s.Insert(4)
	if e := m.Insert(s, 3); e != nil {
		t.Fatal(e)
	}
	for it := m.Begin(); !it.AtEnd(); {
		r, e := it.Ref()
		if e != nil {
			t.Fatal(e)
		}
		key, e := r.Key()
		if e != nil {
			t.Fatal(e)
		}
		text, e := key.String()
		if e != nil {
			t.Fatal(e)
		}
		v, e := r.Value()
		if e != nil {
			t.Fatal(e)
		}
		fmt.Fprintf(&out, "%s : %d\n", text, v)
		if e := it.Next(); e != nil {
			t.Fatal(e)
		}
	}
	if want := "0\n2\n6\n8198\n{1, 2, 44} : 1\n{1, 2, 4, 44} : 3\n"; out.String() != want {
		t.Fatalf("tst5 output %q want %q", out.String(), want)
	}
}
func TestUintSetMapIndependentOracle(t *testing.T) {
	m := newSetMap(t)
	oracle := make(map[uint32]uint32)
	// Each 8-bit mask independently describes a finite mathematical set.
	for round := uint32(0); round < 700; round++ {
		mask := (round*73 + 19) & 255
		key := new(UintSet)
		for bit := uint32(0); bit < 8; bit++ {
			if mask&(1<<bit) != 0 {
				key.Insert(bit)
			}
		}
		if round%3 == 0 {
			if e := m.Remove(key); e != nil {
				t.Fatal(e)
			}
			delete(oracle, mask)
		} else {
			if e := m.Insert(key, round); e != nil {
				t.Fatal(e)
			}
			oracle[mask] = round
		}
		if m.Size() != uint32(len(oracle)) || !m.CheckInvariant() {
			t.Fatalf("round %d size/invariant", round)
		}
		for candidate := uint32(0); candidate < 256; candidate += 17 {
			s := new(UintSet)
			for bit := uint32(0); bit < 8; bit++ {
				if candidate&(1<<bit) != 0 {
					s.Insert(bit)
				}
			}
			got, ok, e := m.Find(s)
			if e != nil {
				t.Fatal(e)
			}
			want, present := oracle[candidate]
			if ok != present || ok && got != want {
				t.Fatalf("round %d mask %d got %d,%v want %d,%v", round, candidate, got, ok, want, present)
			}
		}
	}
	seen := make(map[uint32]bool)
	for it := m.Begin(); !it.AtEnd(); {
		r, e := it.Ref()
		if e != nil {
			t.Fatal(e)
		}
		k, e := r.Key()
		if e != nil {
			t.Fatal(e)
		}
		var mask uint32
		for bit := uint32(0); bit < 8; bit++ {
			b, e := k.Contains(bit)
			if e != nil {
				t.Fatal(e)
			}
			if b {
				mask |= 1 << bit
			}
		}
		v, e := r.Value()
		if e != nil {
			t.Fatal(e)
		}
		if want, ok := oracle[mask]; !ok || v != want || seen[mask] {
			t.Fatal("iteration mapping")
		}
		seen[mask] = true
		if e := it.Next(); e != nil {
			t.Fatal(e)
		}
	}
	if len(seen) != len(oracle) {
		t.Fatal("iteration incomplete")
	}
}
func TestUintSetMapRetainedKeysAndCopies(t *testing.T) {
	m := newSetMap(t)
	a := setKey(1)
	b := setKey(9) // hashes 2,512; distinct buckets.
	if e := m.Insert(a, 4); e != nil {
		t.Fatal(e)
	}
	r, ok, e := m.FindCore(a)
	if e != nil || !ok {
		t.Fatal(e)
	}
	old := m.table[r.slot].key.ref
	if e := m.Remove(a); e != nil {
		t.Fatal(e)
	}
	if !m.words.IsLive(old) || m.WordStats().Live != 1 {
		t.Fatal("FREE retained key destroyed")
	}
	if e := m.Reset(); e != nil {
		t.Fatal(e)
	}
	if !m.words.IsLive(old) {
		t.Fatal("empty-clean reset destroyed key")
	}
	if e := m.Insert(b, 5); e != nil {
		t.Fatal(e)
	}
	if e := m.Reset(); e != nil {
		t.Fatal(e)
	}
	if m.WordStats().Live != 2 {
		t.Fatal("nonshrinking reset lost key ownership")
	}
	// Equal keys with a trailing zero have distinct source logical extents.
	padded := setKey(1, 100)
	padded.Remove(100)
	if e := m.Insert(padded, 7); e != nil {
		t.Fatal(e)
	}
	r, ok, e = m.FindCore(a)
	if e != nil || !ok {
		t.Fatal(e)
	}
	view, e := r.Key()
	if e != nil {
		t.Fatal(e)
	}
	n, e := view.NumWords()
	if e != nil || n != 4 {
		t.Fatalf("logical length %d %v", n, e)
	}
	c, e := view.CapacityWords()
	if e != nil || c != 5 {
		t.Fatalf("copy-ctor geometric capacity %d %v", c, e)
	}
	if _, e := m.InsertIfNotThere(a, 8); e != nil {
		t.Fatal(e)
	}
	n, _ = view.NumWords()
	if n != 4 {
		t.Fatal("absent insertion replaced existing key")
	}
	if e := m.Insert(a, 9); e != nil {
		t.Fatal(e)
	}
	n, _ = view.NumWords()
	if n != 1 {
		t.Fatal("equal overwrite did not assign complete key")
	}
	if m.words.IsLive(old) {
		t.Fatal("overwritten retained key not freed")
	}
	clone, e := m.Clone()
	if e != nil {
		t.Fatal(e)
	}
	defer clone.Close()
	if clone.WordStats().Live != 1 {
		t.Fatal("clone copied FREE key")
	}
	cr, _, e := clone.FindCore(a)
	if e != nil {
		t.Fatal(e)
	}
	if clone.words == m.words || clone.table[cr.slot].key.ref == (arena.VectorRef{}) {
		t.Fatal("clone lacks separate ownership")
	}
}
func TestUintSetMapOperationsAndBorrows(t *testing.T) {
	m := newSetMap(t)
	other := newSetMap(t)
	a := setKey(2)
	b := setKey(5)
	c := setKey(7)
	if e := m.Insert(a, 1); e != nil {
		t.Fatal(e)
	}
	r, ok, e := m.FindCore(a)
	if e != nil || !ok {
		t.Fatal(e)
	}
	if e := r.SetValue(9); e != nil {
		t.Fatal(e)
	}
	if v, e := m.MustFind(a); e != nil || v != 9 {
		t.Fatal(v, e)
	}
	var out uint32 = 77
	if ok, e := m.FindInto(c, &out); e != nil || ok || out != 77 {
		t.Fatal("findinto default")
	}
	if v, e := m.Get(c, 88); e != nil || v != 88 {
		t.Fatal(v, e)
	}
	if e := other.Insert(a, 3); e != nil {
		t.Fatal(e)
	}
	if e := other.Insert(b, 4); e != nil {
		t.Fatal(e)
	}
	if e := m.Union(other); e != nil {
		t.Fatal(e)
	}
	if e := r.SetValue(2); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("structural borrow stayed valid", e)
	}
	if e := m.Intersect(other); e != nil {
		t.Fatal(e)
	}
	if e := m.Insert(c, 5); e != nil {
		t.Fatal(e)
	}
	if e := m.Intersect(other); e != nil {
		t.Fatal(e)
	}
	if m.Size() != 2 {
		t.Fatal("intersection")
	}
	if e := other.Assign(m); e != nil {
		t.Fatal(e)
	}
	if e := other.Assign(other); e != nil {
		t.Fatal(e)
	}
	if e := m.Union(m); e != nil {
		t.Fatal(e)
	}
	if e := m.Intersect(m); e != nil {
		t.Fatal(e)
	}
	if e := other.Reset(); e != nil {
		t.Fatal(e)
	}
	if e := m.Swap(other); e != nil {
		t.Fatal(e)
	}
	if m.Size() != 0 || other.Size() != 2 {
		t.Fatal("swap")
	}
	if e := other.Erase(a); e != nil {
		t.Fatal(e)
	}
	if e := other.Finalize(); e != nil {
		t.Fatal(e)
	}
	if !other.Empty() || other.CollisionCount() != 0 {
		t.Fatal("finalize/stats")
	}
	other.Close()
	if _, _, e := other.Find(a); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
func TestUintSetMapFailures(t *testing.T) {
	for fail := int64(0); fail < 10; fail++ {
		budget := arena.NewBudget(1 << 22)
		budget.FailAfter(fail)
		m, e := NewUintSetMap(UintSetMapConfig{Budget: budget})
		if e == nil {
			m.Close()
		} else if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(e)
		}
		if budget.Used() != 0 {
			t.Fatal("constructor leaked budget", fail, budget.Used())
		}
	}
	for fail := int64(0); fail < 14; fail++ {
		budget := arena.NewBudget(1 << 22)
		m, e := NewUintSetMap(UintSetMapConfig{Budget: budget, Words: arena.VectorConfig{PageSize: 2, Descriptors: arena.Config{PageSize: 2}}})
		if e != nil {
			t.Fatal(e)
		}
		a := setKey(1)
		budget.FailAfter(fail)
		e = m.Insert(a, 7)
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(e)
			}
			if m.Size() != 0 || m.WordStats().Live != 0 {
				t.Fatal("failed insert retained logical key", fail)
			}
		}
		budget.FailAfter(-1)
		if e == nil {
			budget.FailAfter(0)
			// A different, larger source key forces a fresh capacity class;
			// warmed small temporaries may reuse already-accounted pool pages.
			_, _, e = m.Find(setKey(1000))
			budget.FailAfter(-1)
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal("lookup did not make key temporary", e)
			}
			if m.WordStats().Live != 1 {
				t.Fatal("lookup temporary retained")
			}
		}
		m.Close()
		if budget.Used() != 0 {
			t.Fatal("close leaked", fail, budget.Used())
		}
	}
}

func TestUintSetMapCrossBudgetSwap(t *testing.T) {
	for failureOwner := 0; failureOwner < 2; failureOwner++ {
		for fail := int64(0); fail < 3; fail++ {
			ba, bb := arena.NewBudget(1<<24), arena.NewBudget(1<<24)
			a, e := NewUintSetMap(UintSetMapConfig{Budget: ba})
			if e != nil {
				t.Fatal(e)
			}
			b, e := NewUintSetMap(UintSetMapConfig{Budget: bb})
			if e != nil {
				t.Fatal(e)
			}
			ka, kb := setKey(1), setKey(1000)
			if e := a.Insert(ka, 1); e != nil {
				t.Fatal(e)
			}
			if e := b.Insert(kb, 2); e != nil {
				t.Fatal(e)
			}
			ra, _, e := a.FindCore(ka)
			if e != nil {
				t.Fatal(e)
			}
			rb, _, e := b.FindCore(kb)
			if e != nil {
				t.Fatal(e)
			}
			wa, wb := a.words, b.words
			pa, pb := a.table[ra.slot].key.ref, b.table[rb.slot].key.ref
			ua, ub := ba.Used(), bb.Used()
			if failureOwner == 0 {
				ba.FailAfter(fail)
			} else {
				bb.FailAfter(fail)
			}
			e = a.Swap(b)
			ba.FailAfter(-1)
			bb.FailAfter(-1)
			if e != nil {
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal("swap typed failure", e)
				}
				if a.words != wa || b.words != wb || ba.Used() != ua || bb.Used() != ub {
					t.Fatal("partial graph swap", fail)
				}
				if v, e := ra.Value(); e != nil || v != 1 {
					t.Fatal("failed swap invalidated borrow")
				}
			} else {
				if a.words != wb || b.words != wa || !a.words.IsLive(pb) || !b.words.IsLive(pa) {
					t.Fatal("swap copied or destroyed payload")
				}
				if _, e := ra.Value(); !errors.Is(e, arena.ErrInvalidSlot) {
					t.Fatal("swap preserved old borrow")
				}
				if v, ok, e := a.Find(kb); e != nil || !ok || v != 2 {
					t.Fatal("swapped mapping")
				}
			}
			a.Close()
			b.Close()
			if ba.Used() != 0 || bb.Used() != 0 {
				t.Fatal("swap close leaked graph charges")
			}
		}
	}
	// A nil destination participates in the same graph exchange, with no
	// accounting of its own. Capacities and retained dead owners differ.
	a := newSetMap(t)
	budget := arena.NewBudget(1 << 24)
	b, e := newUintSetMapCapacity(UintSetMapConfig{Budget: budget}, 16)
	if e != nil {
		t.Fatal(e)
	}
	if e := a.Insert(setKey(1), 1); e != nil {
		t.Fatal(e)
	}
	if e := b.Insert(setKey(100), 2); e != nil {
		t.Fatal(e)
	}
	if e := b.Reset(); e != nil {
		t.Fatal(e)
	}
	wa, wb := a.words, b.words
	if e := a.Swap(b); e != nil {
		t.Fatal(e)
	}
	if a.words != wb || b.words != wa || a.Capacity() != 16 || b.Capacity() != 8 || a.WordStats().Live != 1 {
		t.Fatal("nil swap retained graph/capacities")
	}
	a.Close()
	b.Close()
	if budget.Used() != 0 {
		t.Fatal("nil swap budget leak")
	}
}

func TestUintSetMapGrowthTombstonesAndReset(t *testing.T) {
	m := newSetMap(t)
	// Bits 3..9 all hash to bucket zero at capacity 8. The strict >3/4
	// condition permits seven entries, and even an equal insertion then grows.
	for i := uint32(3); i < 10; i++ {
		if e := m.Insert(setKey(i), i); e != nil {
			t.Fatal(e)
		}
	}
	if m.Capacity() != 8 {
		t.Fatal("growth threshold must be strict")
	}
	retained := m.table[0].key.ref
	if e := m.Insert(setKey(3), 99); e != nil {
		t.Fatal(e)
	}
	if m.Capacity() != 16 || m.words.IsLive(retained) {
		t.Fatal("prelookup grow or copied old ownership")
	}
	if e := m.Remove(setKey(4)); e != nil {
		t.Fatal(e)
	}
	if e := m.Remove(setKey(5)); e != nil {
		t.Fatal(e)
	}
	if m.Deleted() != 2 {
		t.Fatal("collision deletion")
	}
	lastDeleted := uint32(0)
	for i, e := range m.table {
		if e.state == setMapDeleted {
			lastDeleted = uint32(i)
		}
	}
	dead := m.table[lastDeleted].key.ref
	if !m.words.IsLive(dead) {
		t.Fatal("tombstone payload destroyed")
	}
	if e := m.Insert(setKey(10), 10); e != nil {
		t.Fatal(e)
	}
	r, ok, e := m.FindCore(setKey(10))
	if e != nil || !ok || r.slot != lastDeleted {
		t.Fatal("last tombstone reuse", r.slot, lastDeleted, e)
	}
	if m.words.IsLive(dead) || m.Deleted() != 1 {
		t.Fatal("reused dead payload retained")
	}
	clone, e := m.Clone()
	if e != nil {
		t.Fatal(e)
	}
	defer clone.Close()
	if clone.Deleted() != 0 || clone.Size() != m.Size() || clone.WordStats().Live != clone.Size() {
		t.Fatal("clone state cleanup")
	}
	if e := m.Reset(); e != nil {
		t.Fatal(e)
	}
	if m.Capacity() != 16 || m.WordStats().Live == 0 {
		t.Fatal("small reset destroys retained payload")
	}
	// Reset counts only preexisting FREE slots, and shrinks at most once.
	large, e := newUintSetMapCapacity(UintSetMapConfig{}, 128)
	if e != nil {
		t.Fatal(e)
	}
	defer large.Close()
	if e := large.Insert(setKey(1), 1); e != nil {
		t.Fatal(e)
	}
	if e := large.Reset(); e != nil {
		t.Fatal(e)
	}
	if large.Capacity() != 64 || large.WordStats().Live != 0 {
		t.Fatal("reset shrink/destruction")
	}
	if e := large.Reset(); e != nil {
		t.Fatal(e)
	}
	if large.Capacity() != 64 {
		t.Fatal("empty-clean reset shrank")
	}
	if e := large.rehash(256); e != nil {
		t.Fatal(e)
	}
	if e := large.Insert(setKey(2), 2); e != nil {
		t.Fatal(e)
	}
	if e := large.Finalize(); e != nil {
		t.Fatal(e)
	}
	if large.Capacity() != 64 || large.WordStats().Live != 0 {
		t.Fatal("large finalize")
	}
}

func TestUintSetMapCleanupAndOutOfMemory(t *testing.T) {
	for _, oom := range []bool{false, true} {
		callbacks := 0
		m, e := NewUintSetMap(UintSetMapConfig{OutOfMemory: func() bool { callbacks++; return oom }})
		if e != nil {
			t.Fatal(e)
		}
		// Singleton positions >=8 have hash low byte zero: one 130-entry chain.
		for i := uint32(0); i < 130; i++ {
			if e := m.Insert(setKey(8+i*32), i); e != nil {
				t.Fatal(e)
			}
		}
		if m.Capacity() != 256 {
			t.Fatal("chain growth")
		}
		for i := uint32(0); i < 66; i++ {
			if e := m.Remove(setKey(8 + i*32)); e != nil {
				t.Fatal(e)
			}
		}
		if callbacks != 1 {
			t.Fatal("cleanup strict threshold callback", callbacks)
		}
		if oom {
			if m.Deleted() != 66 || m.WordStats().Live != 130 {
				t.Fatal("oom cleanup changed owners")
			}
		} else {
			if m.Deleted() != 0 || m.WordStats().Live != 64 {
				t.Fatal("cleanup retained dead keys")
			}
		}
		if !m.CheckInvariant() {
			t.Fatal("cleanup invariant")
		}
		m.Close()
	}
}

func TestUintSetMapCopyAssignmentShapeAndIterator(t *testing.T) {
	m := newSetMap(t)
	a := setKey(1)
	if e := m.Insert(a, 1); e != nil {
		t.Fatal(e)
	}
	r, _, e := m.FindCore(a)
	if e != nil {
		t.Fatal(e)
	}
	// Model a source mutable entry key whose vector once held more words and
	// then shrank. Rehash/Clone must copy source capacity, not copy-ctor shape.
	wide, e := m.allocateKey(1, 7)
	if e != nil {
		t.Fatal(e)
	}
	if e := m.words.Write(wide.ref, 0, 2); e != nil {
		t.Fatal(e)
	}
	m.freeKey(m.table[r.slot].key)
	m.table[r.slot].key = wide
	clone, e := m.Clone()
	if e != nil {
		t.Fatal(e)
	}
	defer clone.Close()
	cr, _, e := clone.FindCore(a)
	if e != nil {
		t.Fatal(e)
	}
	k := clone.table[cr.slot].key
	if k.length != 1 || k.capacity != 7 {
		t.Fatal("copy assignment changed source capacity", k)
	}
	physicalLen, e := clone.words.Len(k.ref)
	if e != nil || physicalLen != 7 {
		t.Fatal("pool length is source allocation capacity", physicalLen, e)
	}
	if e := m.rehash(16); e != nil {
		t.Fatal(e)
	}
	r, _, e = m.FindCore(a)
	if e != nil {
		t.Fatal(e)
	}
	if m.table[r.slot].key.capacity != 7 || m.words.IsLive(wide.ref) {
		t.Fatal("rehash skipped synthesized copy/destruction")
	}
	assigned := newSetMap(t)
	if e := assigned.Assign(m); e != nil {
		t.Fatal(e)
	}
	ar, _, e := assigned.FindCore(a)
	if e != nil {
		t.Fatal(e)
	}
	if assigned.table[ar.slot].key.capacity != 2 {
		t.Fatal("assignment core const temporary did not reconstruct capacity")
	}
	view, e := r.Key()
	if e != nil {
		t.Fatal(e)
	}
	copy, e := view.CopyKey()
	if e != nil {
		t.Fatal(e)
	}
	copy.Insert(4)
	if ok, e := view.Contains(4); e != nil || ok {
		t.Fatal("caller copy aliases stored key")
	}
	if h, e := view.Hash(); e != nil || h != 2 {
		t.Fatal(h, e)
	}
	if n, e := view.MaxElem(); e != nil || n != 32 {
		t.Fatal(n, e)
	}
	it, e := m.FindIterator(a)
	if e != nil {
		t.Fatal(e)
	}
	old, e := it.PostNext()
	if e != nil || !old.Equal(m.Begin()) || !it.Equal(m.End()) {
		t.Fatal("postincrement", e)
	}
	if _, e := it.Ref(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	absent, e := m.FindIterator(setKey(7))
	if e != nil || !absent.AtEnd() {
		t.Fatal("missing iterator")
	}
	if reflect.TypeOf(setMapEntry{}).Size() != 28 {
		t.Fatal("scalar entry size")
	}
}

func TestUintSetMapRehashFailureAndTemporaryCopies(t *testing.T) {
	for fail := int64(0); fail < 18; fail++ {
		budget := arena.NewBudget(1 << 24)
		m, e := NewUintSetMap(UintSetMapConfig{Budget: budget, Words: arena.VectorConfig{PageSize: 2, Descriptors: arena.Config{PageSize: 2}}})
		if e != nil {
			t.Fatal(e)
		}
		for i := uint32(3); i < 10; i++ {
			if e := m.Insert(setKey(i), i); e != nil {
				t.Fatal(e)
			}
		}
		before := make([]arena.VectorRef, len(m.table))
		for i, v := range m.table {
			before[i] = v.key.ref
		}
		budget.FailAfter(fail)
		e = m.rehash(16)
		budget.FailAfter(-1)
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) || m.Capacity() != 8 || m.Size() != 7 || m.WordStats().Live != 7 {
				t.Fatal("rehash rollback", fail, e)
			}
			for i, v := range m.table {
				if v.key.ref != before[i] {
					t.Fatal("rehash replaced old owner on failure")
				}
			}
		} else {
			if m.Capacity() != 16 || m.WordStats().Live != 7 {
				t.Fatal("rehash success")
			}
			for _, ref := range before {
				if m.words.IsLive(ref) {
					t.Fatal("old source assignment payload survived")
				}
			}
		}
		m.Close()
		if budget.Used() != 0 {
			t.Fatal("rehash leaked budget", fail)
		}
	}
	// PageSize=2 gives one capacity-2 key per page. Prime all allocation
	// directories, then leave exactly one free block and descriptor. A one-copy
	// existing-key core lookup succeeds; the two-copy wrapper must grow a page.
	budget := arena.NewBudget(1 << 24)
	m, e := NewUintSetMap(UintSetMapConfig{Budget: budget, Words: arena.VectorConfig{PageSize: 2, Descriptors: arena.Config{PageSize: 2}}})
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	key := setKey(1)
	if e := m.Insert(key, 1); e != nil {
		t.Fatal(e)
	}
	budget.FailAfter(0)
	inserted, _, e := m.InsertIfNotThereCore(key, 8)
	budget.FailAfter(-1)
	if e != nil || inserted {
		t.Fatal("one source temporary", e)
	}
	budget.FailAfter(0)
	_, e = m.InsertIfNotThere(key, 8)
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || m.WordStats().Live != 1 {
		t.Fatal("two wrapper temporary copies", e)
	}
	if allocations := testing.AllocsPerRun(50, func() {
		if _, _, e := m.Find(key); e != nil {
			panic(e)
		}
		if e := m.Insert(key, 1); e != nil {
			panic(e)
		}
	}); allocations != 0 {
		t.Fatal("warmed operations allocate", allocations)
	}
}
