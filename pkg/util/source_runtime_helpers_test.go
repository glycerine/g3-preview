// Source: src/util/memory_manager.cpp; src/util/vector.h (failure audit adaptation).
package util

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func contextTLSNearMinimum(t *testing.T, r *Runtime) {
	t.Helper()
	first, e := r.Allocate(8)
	serviceMust(t, e)
	second, e := r.Allocate(8)
	serviceMust(t, e)
	delta := int64(-1000000)
	request := uint64(16) + uint64(int64(delta)) - 8
	if e = r.Reallocate(&first, request); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	serviceMust(t, r.Deallocate(&first))
	state := r.State()
	if state.LocalSize != 0 || state.GlobalSize >= 0 {
		t.Fatal("first real TLS synchronization", state)
	}
	target := int64(math.MinInt64) + 1000 - state.GlobalSize + 16
	request = uint64(target) + 16 - 8
	if e = r.Reallocate(&second, request); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	serviceMust(t, r.Deallocate(&second))
	if state = r.State(); state.GlobalSize != math.MinInt64+1000 || state.LocalSize != 0 {
		t.Fatal("public TLS fixture", state)
	}
}

func contextTLSFinish(t *testing.T, r *Runtime, close func() error) {
	t.Helper()
	for {
		before := r.Stats().LiveBlocks
		e := close()
		if e == nil {
			return
		}
		if !errors.Is(e, arena.ErrBounds) || r.Stats().LiveBlocks >= before {
			t.Fatal("teardown did not consume actual block", e, r.State())
		}
	}
}

func runtimeVectorAuditBlocks(t *testing.T, c *RuntimeVectorColumn, external ...Block) {
	t.Helper()
	roots := append([]Block{}, external...)
	if !c.PendingReplaces && c.Block != (Block{}) {
		roots = append(roots, c.Block)
	}
	if c.Pending != (Block{}) {
		roots = append(roots, c.Pending)
	}
	type identity struct{ slot, generation uint32 }
	expected := map[identity]bool{}
	rv := reflect.ValueOf(c.Runtime).Elem()
	origin := testFieldBySourceName(rv, "origin").Uint()
	for _, h := range roots {
		hv := reflect.ValueOf(h)
		if testFieldBySourceName(hv, "origin").Uint() != origin {
			t.Fatal("foreign declared root")
		}
		id := identity{uint32(testFieldBySourceName(hv, "slot").Uint()), uint32(testFieldBySourceName(hv, "generation").Uint())}
		if expected[id] {
			t.Fatal("duplicate owning root", id)
		}
		expected[id] = true
		_, e := c.Runtime.Size(h)
		serviceMust(t, e)
	}
	records := testFieldBySourceName(rv, "blocks")
	var physical, init uint64
	live := 0
	for i := 0; i < records.Len(); i++ {
		b := records.Index(i)
		if !testFieldBySourceName(b, "live").Bool() {
			continue
		}
		id := identity{uint32(i + 1), uint32(testFieldBySourceName(b, "generation").Uint())}
		if !expected[id] {
			t.Fatal("anonymous raw allocation", id)
		}
		delete(expected, id)
		live++
		physical += uint64(testFieldBySourceName(b, "data").Len())
		init += uint64(testFieldBySourceName(b, "initialized").Len()) * 8
	}
	st := c.Runtime.Stats()
	if len(expected) != 0 || uint32(live) != st.LiveBlocks || physical != st.PhysicalBytes || init != st.InitializationBytes {
		t.Fatal("actual raw graph", expected, st, physical, init)
	}
}
