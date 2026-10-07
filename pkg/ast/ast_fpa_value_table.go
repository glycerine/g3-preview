// Source: src/util/chashtable.h:47-89,207-244; src/ast/fpa_decl_plugin.h:109-116.
// Source author: Leonardo de Moura (2011-04-14).
// Licensed under the MIT license; see LICENSE.txt.
package ast

import (
	"github.com/glycerine/g3/pkg/arena"
)

// This is the inline unsigned MPF-id specialization. Source cell pointers are
// encoded as indices plus two; zero is null and one is the source free sentinel.
// Payload is unread until an insertion initializes it. The eight-byte Go cell
// geometry is distinct from the native pointer ABI. The FPA owner charges this
// inline header once; only the actual cell allocation is charged here.
type astFloatValueCell struct{ next, data uint32 }
type astFloatValueTable struct {
	cells                                           []astFloatValueCell
	capacity, initSlots, initCellar, slots          uint32
	usedSlots, size, collisions, nextCell, freeCell uint32
}

func (t *astFloatValueTable) construct(b *arena.Budget) error {
	// Both literal constructor assertions are true for these source defaults.
	t.initSlots, t.initCellar = 8, 2
	t.capacity = t.initSlots + t.initCellar
	if e := b.Reserve(uint64(t.capacity) * 8); e != nil {
		return e
	}
	a := make([]astFloatValueCell, t.capacity)
	for i := range a {
		a[i].next = 1
	}
	t.cells = a
	t.slots, t.usedSlots, t.size = t.initSlots, 0, 0
	t.nextCell, t.freeCell = t.slots, 0
	t.collisions = 0
	return nil
}

func (t *astFloatValueTable) destroy(b *arena.Budget) {
	if t.cells != nil {
		b.Release(uint64(t.capacity) * 8)
		t.cells = nil
	}
}
