// Source: src/ast/ast.h:1480; src/util/map.h:285;
// src/util/hashtable.h:244-256,272-274.
package util

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestUintMapInlineMemberActualConstructorReservationPrefixes(t *testing.T) {
	for ordinal := int64(0); ordinal <= 1; ordinal++ {
		b := arena.NewBudget(^uint64(0))
		b.FailAfter(ordinal)
		var m UintMap
		e := InitializeInlineUintMap(&m, UintMapConfig{Budget: b})
		if m.rootBytes != 0 {
			t.Fatal("embedded member acquired a standalone header")
		}
		if ordinal == 0 {
			if !errors.Is(e, arena.ErrInjectedFailure) || m.table != nil || m.tableBytes != 0 || b.Used() != 0 {
				t.Fatalf("failed source table prefix: %v %#v used=%d", e, m, b.Used())
			}
		} else {
			if e != nil || m.Capacity() != 8 || m.Size() != 0 || m.Deleted() != 0 || m.tableBytes != 128 || b.Used() != 128 {
				t.Fatalf("completed source member: %v %#v used=%d", e, m, b.Used())
			}
			for _, cell := range m.table {
				if cell.state != uintMapFree {
					t.Fatal("ascending default table entry not free")
				}
			}
		}
		// Immediate named entry backing before fault lift, then only this member's DT.
		if b.Used() != m.tableBytes {
			t.Fatal("unaccounted embedded backing")
		}
		m.Close()
		if b.Used() != 0 {
			t.Fatal("member DT failed to refund actual entry array")
		}
		b.FailAfter(-1)
	}
}

func TestUintMapInlineMemberSameEngineAndExistingOwnerRefusal(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	standalone, e := NewUintMap(UintMapConfig{Budget: b})
	mapMust(t, e)
	var embedded UintMap
	mapMust(t, InitializeInlineUintMap(&embedded, UintMapConfig{Budget: b}))
	header := uint64(reflect.TypeOf(UintMap{}).Size())
	if b.Used() != header+256 {
		t.Fatalf("actual wrapper plus two entry arrays: %d", b.Used())
	}
	for _, k := range []uint32{0, 8, 16, 24, 32, 40, 48, 56, 64, 72, 80} {
		mapMust(t, embedded.Insert(k, k+3))
		mapMust(t, standalone.Insert(k, k+3))
	}
	if embedded.Capacity() != standalone.Capacity() || embedded.Size() != standalone.Size() {
		t.Fatal("different embedded algorithm")
	}
	for _, k := range []uint32{0, 8, 16, 24, 32, 40, 48, 56, 64, 72, 80} {
		a, ok := embedded.Find(k)
		z, ok2 := standalone.Find(k)
		if !ok || !ok2 || a != z || a != k+3 {
			t.Fatal("independent payload oracle", k, a, z)
		}
	}
	used := b.Used()
	old := embedded.table
	if e = InitializeInlineUintMap(&embedded, UintMapConfig{Budget: b}); !errors.Is(e, arena.ErrConfig) || b.Used() != used || &old[0] != &embedded.table[0] {
		t.Fatal("existing member was reconstructed", e)
	}
	embedded.Close()
	standalone.Close()
	if b.Used() != 0 {
		t.Fatal("actual member/header refunds", b.Used())
	}
	if e = InitializeInlineUintMap(&embedded, UintMapConfig{Budget: b}); !errors.Is(e, arena.ErrConfig) || b.Used() != 0 {
		t.Fatal("ended member lifetime silently restarted", e)
	}
}
