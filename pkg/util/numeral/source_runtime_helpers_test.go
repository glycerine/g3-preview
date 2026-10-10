// Source: src/util/memory_manager.cpp; src/util/vector.h (failure audit adaptation).
package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func contextTLSNearMinimum(t *testing.T, r *umemory.Runtime) {
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

func contextTLSFinish(t *testing.T, r *umemory.Runtime, close func() error) {
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
