// Source: src/util/chashtable.h:91-584; src/ast/fpa_decl_plugin.h:96-108.
// Source author: Leonardo de Moura (2011-04-14).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import "github.com/glycerine/g3/golang/arena"

// Procs borrow the actual MPF manager and value vector through the calling FPA
// owner. They are call-local: cells store only unsigned IDs, never closures.
type astFloatValueTableProcs struct {
	hash  func(uint32) (uint32, error)
	equal func(uint32, uint32) (bool, error)
}

func (t *astFloatValueTable) hasFreeCells() bool {
	return t.freeCell != 0 || t.nextCell < t.capacity
}

func (t *astFloatValueTable) takeFreeCell() uint32 {
	if t.freeCell != 0 {
		i := t.freeCell - 2
		t.freeCell = t.cells[i].next
		return i
	}
	i := t.nextCell
	t.nextCell++
	return i
}

func (t *astFloatValueTable) recycleCell(i uint32) {
	t.cells[i].next = t.freeCell
	t.freeCell = i + 2
}

// copyInto visits ascending source slots, then their literal chain order. It
// prepends collisions by copying the target head into the next cellar cell.
func (t *astFloatValueTable) copyInto(p astFloatValueTableProcs, cells []astFloatValueCell, slots uint32) (uint32, bool, error) {
	next := slots
	t.usedSlots = 0 // source passes this member by reference, including retries
	for i := uint32(0); i < t.slots; i++ {
		if t.cells[i].next == 1 {
			continue
		}
		for j := i; ; {
			h, e := p.hash(t.cells[j].data)
			if e != nil {
				return next, false, e
			}
			k := h & (slots - 1)
			if cells[k].next == 1 {
				cells[k].data, cells[k].next = t.cells[j].data, 0
				t.usedSlots++
			} else {
				if next == uint32(len(cells)) {
					return next, false, nil
				}
				cells[next] = cells[k]
				cells[k].data, cells[k].next = t.cells[j].data, next+2
				next++
			}
			if t.cells[j].next == 0 {
				break
			}
			j = t.cells[j].next - 2
		}
	}
	return next, true, nil
}

func (t *astFloatValueTable) expand(b *arena.Budget, p astFloatValueTableProcs) error {
	slots, cellar := t.slots*2, (t.capacity-t.slots)*2
	for {
		// Source unsigned overflow is outside the valid allocation domain.
		capacity := slots + cellar
		if slots <= t.slots || cellar == 0 || capacity < slots || capacity > ^uint32(0)-2 {
			return arena.ErrBudget
		}
		bytes := uint64(capacity) * 8
		if e := b.Reserve(bytes); e != nil {
			return e
		}
		cells := make([]astFloatValueCell, capacity)
		for i := range cells {
			cells[i].next = 1
		}
		next, ok, e := t.copyInto(p, cells, slots)
		if e != nil {
			b.Release(bytes)
			return e
		}
		if ok {
			t.destroy(b)
			t.cells, t.capacity, t.slots = cells, capacity, slots
			t.nextCell, t.freeCell = next, 0
			return nil
		}
		b.Release(bytes)
		if cellar > ^uint32(0)/2 {
			return arena.ErrBudget
		}
		cellar *= 2
	}
}

// put shares the three source insertion bodies. replace distinguishes insert
// from insert_if_not_there; the latter returns the original canonical MPF ID.
func (t *astFloatValueTable) put(b *arena.Budget, p astFloatValueTableProcs, d uint32, replace bool) (uint32, bool, error) {
	if !t.hasFreeCells() {
		if e := t.expand(b, p); e != nil {
			return 0, false, e
		}
	}
	h, e := p.hash(d)
	if e != nil {
		return 0, false, e
	}
	i := h & (t.slots - 1)
	if t.cells[i].next == 1 {
		t.size++
		t.usedSlots++
		t.cells[i].data, t.cells[i].next = d, 0
		return d, true, nil
	}
	for j := i; ; {
		eq, e := p.equal(t.cells[j].data, d)
		if e != nil {
			return 0, false, e
		}
		if eq {
			if replace {
				t.cells[j].data = d
			}
			return t.cells[j].data, false, nil
		}
		if t.cells[j].next == 0 {
			break
		}
		j = t.cells[j].next - 2
	}
	t.size++
	j := t.takeFreeCell()
	t.cells[j] = t.cells[i]
	t.cells[i].data, t.cells[i].next = d, j+2
	return d, true, nil
}

func (t *astFloatValueTable) insert(b *arena.Budget, p astFloatValueTableProcs, d uint32) error {
	_, _, e := t.put(b, p, d, true)
	return e
}

func (t *astFloatValueTable) insertIfAbsent(b *arena.Budget, p astFloatValueTableProcs, d uint32) (uint32, bool, error) {
	return t.put(b, p, d, false)
}

func (t *astFloatValueTable) find(p astFloatValueTableProcs, d uint32) (uint32, bool, error) {
	h, e := p.hash(d)
	if e != nil {
		return 0, false, e
	}
	i := h & (t.slots - 1)
	if t.cells[i].next == 1 {
		return 0, false, nil
	}
	for {
		eq, e := p.equal(t.cells[i].data, d)
		if e != nil {
			return 0, false, e
		}
		if eq {
			return t.cells[i].data, true, nil
		}
		if t.cells[i].next == 0 {
			return 0, false, nil
		}
		i = t.cells[i].next - 2
	}
}

func (t *astFloatValueTable) contains(p astFloatValueTableProcs, d uint32) (bool, error) {
	_, ok, e := t.find(p, d)
	return ok, e
}

func (t *astFloatValueTable) erase(p astFloatValueTableProcs, d uint32) error {
	h, e := p.hash(d)
	if e != nil {
		return e
	}
	i := h & (t.slots - 1)
	if t.cells[i].next == 1 {
		return nil
	}
	prev := uint32(0)
	for {
		eq, e := p.equal(t.cells[i].data, d)
		if e != nil {
			return e
		}
		if eq {
			t.size--
			if prev == 0 {
				if t.cells[i].next == 0 {
					t.usedSlots--
					t.cells[i].next = 1
				} else {
					j := t.cells[i].next - 2
					t.cells[i] = t.cells[j]
					t.recycleCell(j)
				}
			} else {
				t.cells[prev-2].next = t.cells[i].next
				t.recycleCell(i)
			}
			return nil
		}
		if t.cells[i].next == 0 {
			return nil
		}
		prev, i = i+2, t.cells[i].next-2
	}
}

func (t *astFloatValueTable) reset(b *arena.Budget) error {
	if t.size == 0 {
		return nil
	}
	return t.finalize(b)
}

func (t *astFloatValueTable) finalize(b *arena.Budget) error {
	t.destroy(b)
	capacity := t.initSlots + t.initCellar
	if e := b.Reserve(uint64(capacity) * 8); e != nil {
		return e
	}
	cells := make([]astFloatValueCell, capacity)
	for i := range cells {
		cells[i].next = 1
	}
	t.cells, t.capacity, t.slots = cells, capacity, t.initSlots
	t.usedSlots, t.size, t.nextCell, t.freeCell = 0, 0, t.slots, 0
	return nil
}

// Iterator holds a borrowed table view, never an owning root. The enclosing
// FPA owner must not mutate its table while that source iterator is in use.
type astFloatValueTableIterator struct {
	table      *astFloatValueTable
	cells      []astFloatValueCell
	slots      uint32
	slot, cell uint32
}

func (t *astFloatValueTable) begin() astFloatValueTableIterator {
	it := astFloatValueTableIterator{table: t, cells: t.cells, slots: t.slots}
	it.moveToUsed()
	return it
}

func (it *astFloatValueTableIterator) moveToUsed() {
	for it.slot < it.slots {
		if it.cells[it.slot].next != 1 {
			it.cell = it.slot + 2
			return
		}
		it.slot++
	}
	it.cell = 0
}

func (it astFloatValueTableIterator) valid() bool {
	return it.cell != 0 && it.table != nil && len(it.cells) != 0 &&
		len(it.table.cells) == len(it.cells) && &it.table.cells[0] == &it.cells[0]
}
func (it astFloatValueTableIterator) value() uint32 {
	if !it.valid() {
		panic(arena.ErrClosed)
	}
	return it.cells[it.cell-2].data
}
func (it *astFloatValueTableIterator) advance() {
	if !it.valid() {
		panic(arena.ErrClosed)
	}
	it.cell = it.cells[it.cell-2].next
	if it.cell == 0 {
		it.slot++
		it.moveToUsed()
	}
}
