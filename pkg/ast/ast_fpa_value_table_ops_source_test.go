// Source: src/test/chashtable.cpp:37-172; src/util/chashtable.h:91-584.
package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func floatTableTestProcs(collide bool) astFloatValueTableProcs {
	return astFloatValueTableProcs{
		hash: func(v uint32) (uint32, error) {
			if collide {
				return v % 2, nil
			}
			return v, nil
		},
		equal: func(a, b uint32) (bool, error) { return a == b, nil },
	}
}

func TestASTFloatTableSourceChaining(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	var table astFloatValueTable
	if e := table.construct(b); e != nil {
		t.Fatal(e)
	}
	defer func() {
		table.destroy(b)
		if b.Used() != 0 {
			t.Fatal("backing retained", b.Used())
		}
	}()
	p := floatTableTestProcs(true)
	insert := func(v uint32) {
		t.Helper()
		if e := table.insert(b, p, v); e != nil {
			t.Fatal(e)
		}
	}
	has := func(v uint32, want bool) {
		t.Helper()
		_, ok, e := table.find(p, v)
		if e != nil || ok != want {
			t.Fatal(v, ok, want, e)
		}
	}
	insert(10)
	insert(12)
	if table.usedSlots != 1 {
		t.Fatal(table.usedSlots)
	}
	insert(13)
	if table.usedSlots != 2 {
		t.Fatal(table.usedSlots)
	}
	insert(14)
	if table.usedSlots != 2 || table.size != 4 {
		t.Fatal(table.usedSlots, table.size)
	}
	if e := table.erase(p, 12); e != nil {
		t.Fatal(e)
	}
	has(12, false)
	if table.size != 3 {
		t.Fatal(table.size)
	}
	has(10, true)
	has(12, false)
	has(14, true)
	has(13, true)
	insert(16)
	if table.size != 4 {
		t.Fatal(table.size)
	}
	insert(18)
	if table.size != 5 || table.usedSlots != 2 {
		t.Fatal(table.size, table.usedSlots)
	}
	if e := table.erase(p, 10); e != nil {
		t.Fatal(e)
	}
	has(10, false)
	has(12, false)
	for _, v := range []uint32{14, 13, 16, 18} {
		has(v, true)
	}
	seen := map[uint32]bool{}
	for it := table.begin(); it.valid(); it.advance() {
		seen[it.value()] = true
	}
	if len(seen) != 4 || !seen[14] || !seen[13] || !seen[16] || !seen[18] {
		t.Fatal(seen)
	}
}

func TestASTFloatTableIndependentTransitions(t *testing.T) {
	for _, collide := range []bool{false, true} {
		b := arena.NewBudget(1 << 20)
		var table astFloatValueTable
		if e := table.construct(b); e != nil {
			t.Fatal(e)
		}
		p := floatTableTestProcs(collide)
		model := map[uint32]bool{}
		// Bounded deterministic input replaces the source's random input, not its table algorithm.
		for i := uint32(0); i < 12000; i++ {
			v := (i*i + 17*i + i/11) % 67
			if i%3 == 2 {
				if e := table.erase(p, v); e != nil {
					t.Fatal(e)
				}
				delete(model, v)
			} else {
				if e := table.insert(b, p, v); e != nil {
					t.Fatal(e)
				}
				model[v] = true
			}
			if table.size != uint32(len(model)) {
				t.Fatal(i, table.size, len(model))
			}
			for k := uint32(0); k < 67; k++ {
				_, ok, e := table.find(p, k)
				if e != nil || ok != model[k] {
					t.Fatal(i, k, ok, e)
				}
			}
		}
		if e := table.reset(b); e != nil {
			t.Fatal(e)
		}
		if table.size != 0 || table.usedSlots != 0 || table.capacity != 10 || b.Used() != 80 {
			t.Fatal(table, b.Used())
		}
		table.destroy(b)
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
}

func TestASTFloatTableExpandBeforeDuplicateAndCanonicalID(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	var table astFloatValueTable
	if e := table.construct(b); e != nil {
		t.Fatal(e)
	}
	defer table.destroy(b)
	p := floatTableTestProcs(true)
	for _, v := range []uint32{10, 12, 14} {
		if e := table.insert(b, p, v); e != nil {
			t.Fatal(e)
		}
	}
	// Even a duplicate triggers growth before the source probes for equality.
	b.FailAfter(0)
	if _, _, e := table.insertIfAbsent(b, p, 10); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if table.capacity != 10 || table.size != 3 || b.Used() != 80 {
		t.Fatal(table, b.Used())
	}
	b.FailAfter(-1)
	if v, added, e := table.insertIfAbsent(b, p, 10); e != nil || added || v != 10 || table.capacity != 20 {
		t.Fatal(v, added, e, table.capacity)
	}
	// Distinct MPF IDs can denote equal values. The existing ID must win insert-if-absent.
	q := astFloatValueTableProcs{
		hash:  func(v uint32) (uint32, error) { return v % 100, nil },
		equal: func(a, c uint32) (bool, error) { return a%100 == c%100, nil },
	}
	var canonical astFloatValueTable
	if e := canonical.construct(b); e != nil {
		t.Fatal(e)
	}
	defer canonical.destroy(b)
	if e := canonical.insert(b, q, 10); e != nil {
		t.Fatal(e)
	}
	if v, added, e := canonical.insertIfAbsent(b, q, 110); e != nil || added || v != 10 {
		t.Fatal(v, added, e)
	}
	if e := canonical.insert(b, q, 110); e != nil {
		t.Fatal(e)
	}
	if v, ok, e := canonical.find(q, 10); e != nil || !ok || v != 110 {
		t.Fatal(v, ok, e)
	}
}

func TestASTFloatTableEveryGrowthReservation(t *testing.T) {
	p := floatTableTestProcs(true)
	success := false
	for ordinal := int64(0); ordinal < 12; ordinal++ {
		b := arena.NewBudget(1 << 20)
		var table, foreign astFloatValueTable
		if e := table.construct(b); e != nil {
			t.Fatal(e)
		}
		if e := foreign.construct(b); e != nil {
			t.Fatal(e)
		}
		if e := foreign.insert(b, p, 123); e != nil {
			t.Fatal(e)
		}
		b.FailAfter(ordinal)
		inserted := uint32(0)
		var failure error
		for inserted < 64 {
			failure = table.insert(b, p, inserted*2)
			if failure != nil {
				break
			}
			inserted++
		}
		// Check the actual two backing roots before disabling injection.
		if b.Used() != uint64(table.capacity+foreign.capacity)*8 || table.size != inserted {
			t.Fatal(ordinal, b.Used(), table.capacity, table.size, inserted)
		}
		for i := uint32(0); i < inserted; i++ {
			if v, ok, e := table.find(p, i*2); e != nil || !ok || v != i*2 {
				t.Fatal(ordinal, i, v, ok, e)
			}
		}
		if failure != nil && !errors.Is(failure, arena.ErrInjectedFailure) {
			t.Fatal(failure)
		}
		b.FailAfter(-1)
		table.destroy(b)
		if b.Used() != 80 {
			t.Fatal("foreign backing was retired", ordinal, b.Used())
		}
		if v, ok, e := foreign.find(p, 123); e != nil || !ok || v != 123 {
			t.Fatal(v, ok, e)
		}
		foreign.destroy(b)
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
		if failure == nil {
			success = true
			if ordinal != 5 {
				t.Fatal("literal growth horizon", ordinal)
			}
			break
		}
	}
	if !success {
		t.Fatal("no successful growth frontier")
	}
}

func TestASTFloatTableRecyclingOrderAndWarmBacking(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	var table astFloatValueTable
	if e := table.construct(b); e != nil {
		t.Fatal(e)
	}
	defer table.destroy(b)
	p := floatTableTestProcs(true)
	for _, v := range []uint32{10, 12, 14} {
		if e := table.insert(b, p, v); e != nil {
			t.Fatal(e)
		}
	}
	it := table.begin()
	for _, want := range []uint32{14, 12, 10} {
		if !it.valid() || it.value() != want {
			t.Fatal("head-prepend iteration", it, want)
		}
		it.advance()
	}
	if it.valid() {
		t.Fatal("extra chain member")
	}
	if e := table.erase(p, 12); e != nil {
		t.Fatal(e)
	}
	freed, next := table.freeCell, table.nextCell
	b.FailAfter(0)
	if e := table.insert(b, p, 16); e != nil {
		t.Fatal("recycled cellar must precede expansion", e)
	}
	if table.freeCell != 0 || table.nextCell != next || table.cells[0].next != freed {
		t.Fatal(table)
	}
	if n := testing.AllocsPerRun(50, func() {
		if e := table.erase(p, 16); e != nil {
			panic(e)
		}
		if e := table.insert(b, p, 16); e != nil {
			panic(e)
		}
	}); n != 0 {
		t.Fatal("warm cell operations allocate", n)
	}
	b.FailAfter(-1)
}

func TestASTFloatTableResetAllocationFailure(t *testing.T) {
	b := arena.NewBudget(1024)
	var table astFloatValueTable
	if e := table.construct(b); e != nil {
		t.Fatal(e)
	}
	p := floatTableTestProcs(false)
	if e := table.insert(b, p, 42); e != nil {
		t.Fatal(e)
	}
	b.FailAfter(0)
	if e := table.reset(b); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	// finalize deletes before it allocates, unlike expansion's retained old table.
	if table.cells != nil || b.Used() != 0 {
		t.Fatal("source delete-before-init", table, b.Used())
	}
	table.destroy(b)
	b.FailAfter(-1)
	if e := table.finalize(b); e != nil {
		t.Fatal(e)
	}
	if table.size != 0 || table.capacity != 10 || b.Used() != 80 {
		t.Fatal(table, b.Used())
	}
	table.destroy(b)
}

func TestASTFloatTableIteratorKeepsCapturedBacking(t *testing.T) {
	b := arena.NewBudget(1024)
	var table astFloatValueTable
	if e := table.construct(b); e != nil {
		t.Fatal(e)
	}
	defer table.destroy(b)
	p := floatTableTestProcs(true)
	for _, v := range []uint32{10, 12, 14} {
		if e := table.insert(b, p, v); e != nil {
			t.Fatal(e)
		}
	}
	it := table.begin()
	if !it.valid() || it.value() != 14 {
		t.Fatal(it)
	}
	if e := table.insert(b, p, 16); e != nil {
		t.Fatal(e)
	}
	// Native iterators capture old table pointers. A guarded Go view must reject
	// that retired backing rather than silently follow its replacement.
	if it.valid() {
		t.Fatal("old iterator retargeted a replacement table")
	}
}
