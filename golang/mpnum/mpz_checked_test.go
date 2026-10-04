// Licensed under the MIT license.
//go:build g3checked

package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestIntegerCheckedStaleCellAfterReuse(t *testing.T) {
	m, _ := integerManager(t)
	var original Int
	if e := m.SetInt64(&original, 1<<40); e != nil {
		t.Fatal(e)
	}
	// Deliberately violate copying rules solely to test the checked generation
	// guard. The stale copy is never a valid owner and must not be deleted.
	stale := original
	slot := numericSlot(original.cell)
	if e := m.Del(&original); e != nil {
		t.Fatal(e)
	}
	if e := m.SetInt64(&original, 1<<41); e != nil {
		t.Fatal(e)
	}
	if numericSlot(original.cell) != slot {
		t.Fatal("test did not reuse slot")
	}
	if _, e := m.String(&stale); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("stale generation accepted", e)
	}
}
