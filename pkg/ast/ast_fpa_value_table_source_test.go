// Source: src/util/chashtable.h:47-89,207-244; src/ast/fpa_decl_plugin.h:109-116.
// Source author: Leonardo de Moura (2011-04-14).
// Licensed under the MIT license; see LICENSE.txt.
package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTFloatingPointActualChainedValueTableConstructor(t *testing.T) {
	for ordinal := int64(0); ordinal <= 1; ordinal++ {
		b := arena.NewBudget(1 << 20)
		var table astFloatValueTable // actual inline member, no separate wrapper
		b.FailAfter(ordinal)
		e := table.construct(b)
		if ordinal == 0 {
			if !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != 0 || table.cells != nil || table.capacity != 10 {
				t.Fatal("actual allocation-refused source prefix", e, b.Used(), table)
			}
		} else {
			if e != nil || table.capacity != 10 || table.slots != 8 || table.nextCell != 8 || table.freeCell != 0 || table.size != 0 || table.usedSlots != 0 || table.collisions != 0 {
				t.Fatal("source chained geometry", e, table)
			}
			if b.Used() != 80 || len(table.cells) != 10 {
				t.Fatal("actual pointer-free cell backing", b.Used(), len(table.cells))
			}
			for i, c := range table.cells {
				if c.next != 1 {
					t.Fatal("ascending default cell next sentinel", i, c)
				}
			}
		}
		b.FailAfter(-1)
		table.destroy(b)
		if b.Used() != 0 {
			t.Fatal("actual table member destructor", b.Used())
		}
	}
}
