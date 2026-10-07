// Source: src/util/permutation.cpp:21-76; src/util/permutation.h:25-44.
package util

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func utilityRuntimePermutationFixture(t *testing.T, n uint32) (*RuntimePermutation, *Runtime, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	paramsMust(t, e)
	p, e := NewRuntimePermutation(r, b, n)
	paramsMust(t, e)
	t.Cleanup(func() {
		b.FailAfter(-1)
		paramsMust(t, p.Close())
		paramsMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("unnamed owner", b.Used())
		}
	})
	return p, r, b
}
func permutationExpect(t *testing.T, p *RuntimePermutation, want []uint32) {
	t.Helper()
	n, e := p.Size()
	paramsMust(t, e)
	if n != uint32(len(want)) {
		t.Fatal(n, want)
	}
	for i, v := range want {
		got, e := p.At(uint32(i))
		paramsMust(t, e)
		if got != v {
			t.Fatal(i, got, v)
		}
		inv, e := p.Inv(v)
		paramsMust(t, e)
		if inv != uint32(i) {
			t.Fatal("inverse", v, inv, i)
		}
	}
	yes, e := p.CheckInvariant()
	paramsMust(t, e)
	if !yes {
		t.Fatal("source invariant")
	}
}
func TestRuntimePermutationIndependentResetSwapMove(t *testing.T) {
	for n := uint32(0); n < 9; n++ {
		t.Run(string(rune('a'+n)), func(t *testing.T) {
			p, _, _ := utilityRuntimePermutationFixture(t, n)
			want := make([]uint32, n)
			for i := range want {
				want[i] = uint32(i)
			}
			permutationExpect(t, p, want)
			for i := uint32(0); i < n; i++ {
				for j := uint32(0); j < n; j++ {
					paramsMust(t, p.Swap(i, j))
					want[i], want[j] = want[j], want[i]
					permutationExpect(t, p, want)
					paramsMust(t, p.MoveAfter(i, j))
					if i < j {
						v := want[i]
						copy(want[i:j], want[i+1:j+1])
						want[j] = v
					}
					permutationExpect(t, p, want)
				}
			}
			paramsMust(t, p.Reset(n+3))
			want = make([]uint32, n+3)
			for i := range want {
				want[i] = uint32(i)
			}
			permutationExpect(t, p, want)
			paramsMust(t, p.Reset(0))
			permutationExpect(t, p, nil)
		})
	}
}
func TestRuntimePermutationMoveNoopLeavesInvalidIndicesUnread(t *testing.T) {
	p, _, b := utilityRuntimePermutationFixture(t, 0)
	used := b.Used()
	b.FailAfter(0)
	paramsMust(t, p.MoveAfter(^uint32(0), 0))
	if b.Used() != used {
		t.Fatal("noop allocated")
	}
}
func TestRuntimePermutationSourceResetRetainsActualAlternatingFailurePrefix(t *testing.T) {
	p, r, b := utilityRuntimePermutationFixture(t, 2)
	// Both source capacities are two. First grow succeeds; inverse grow refuses.
	b.FailAfter(1)
	e := p.Reset(3)
	if e == nil {
		t.Fatal("missing actual growth refusal")
	}
	n, e := p.Size()
	paramsMust(t, e)
	if n != 3 {
		t.Fatal("p completed prefix", n)
	}
	n, e = p.inverse.Size()
	paramsMust(t, e)
	if n != 2 {
		t.Fatal("inverse unread next push", n)
	}
	a, e := p.At(2)
	paramsMust(t, e)
	if a != 2 {
		t.Fatal("p prefix", a)
	}
	permutationPhysical(t, p, r, b)
	b.FailAfter(-1)
	paramsMust(t, p.Reset(3))
	permutationExpect(t, p, []uint32{0, 1, 2})
}

func TestRuntimePermutationDisplayExactInsertionAndRefusalPrefixes(t *testing.T) {
	p, r, b := utilityRuntimePermutationFixture(t, 3)
	paramsMust(t, p.Swap(0, 2))
	want := "0:2 1:1 2:0"
	var out strings.Builder
	paramsMust(t, p.Display(&out))
	if out.String() != want {
		t.Fatal(out.String())
	}
	for n := 0; n < len(want); n++ {
		w := &uintSetPrefixWriter{remaining: n}
		e := p.Display(w)
		if !errors.Is(e, uintSetStreamRefusal) || string(w.bytes) != want[:n] {
			t.Fatal(n, e, string(w.bytes))
		}
		permutationPhysical(t, p, r, b)
	}
}
func TestRuntimePermutationResetEveryReservationPrefixAndRecovery(t *testing.T) {
	success := -1
	for ordinal := int64(0); ordinal < 100; ordinal++ {
		p, r, b := utilityRuntimePermutationFixture(t, 2)
		b.FailAfter(ordinal)
		e := p.Reset(9)
		permutationPhysical(t, p, r, b)
		n, x := p.forward.Size()
		paramsMust(t, x)
		m, x := p.inverse.Size()
		paramsMust(t, x)
		if n < m || n > m+1 {
			t.Fatal("alternating source pushes", ordinal, n, m)
		}
		for i := uint32(0); i < n; i++ {
			v, x := p.forward.At(i)
			paramsMust(t, x)
			if v != i {
				t.Fatal("p initialized prefix", ordinal, i, v)
			}
		}
		for i := uint32(0); i < m; i++ {
			v, x := p.inverse.At(i)
			paramsMust(t, x)
			if v != i {
				t.Fatal("inv initialized prefix", ordinal, i, v)
			}
		}
		if e == nil {
			success = int(ordinal)
			permutationExpect(t, p, []uint32{0, 1, 2, 3, 4, 5, 6, 7, 8})
		}
		b.FailAfter(-1)
		paramsMust(t, p.Reset(9))
		permutationExpect(t, p, []uint32{0, 1, 2, 3, 4, 5, 6, 7, 8})
		paramsMust(t, p.Close())
		if success >= 0 {
			break
		}
	}
	if success < 0 {
		t.Fatal("no first success")
	}
	t.Log("reset first success reservation", success)
}

func TestRuntimePermutationDestructorStoppedPhysicalReceiptKeepsOlderMember(t *testing.T) {
	// Public TLS signed-overflow-domain recipe, not ordinary source OOM.
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	p, e := NewRuntimePermutation(r, b, 30000)
	paramsMust(t, e)
	contextTLSNearMinimum(t, r)
	before := p.forward.Column.Raw.Block
	e = p.SourceFinalize()
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual inverse physical free/TLS stop", e)
	}
	if p.inverse.Column.Raw.Block != (Block{}) || p.forward.Column.Raw.Block != before || p.destructorStage != 0 || p.destructorCause != e {
		t.Fatal("inverse before older forward/sticky source cursor")
	}
	permutationPhysical(t, p, r, b)
	if p.Close() != e {
		t.Fatal("ordinary Close must retain source stop")
	}
	first := p.CleanupStopped()
	if !errors.Is(first, arena.ErrBounds) {
		t.Fatal("actual older forward physical free/TLS stop", first)
	}
	if p.forward.Column.Raw.Block != (Block{}) || p.closed || r.Stats().LiveBlocks != 0 {
		t.Fatal("owning consumed physical receipt")
	}
	permutationPhysical(t, p, r, b)
	paramsMust(t, p.CleanupStopped())
	if p.destructorCause != e || p.destructorStage != 0 {
		t.Fatal("owning abort repaired source history")
	}
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("actual once-only release", b.Used())
	}
}
