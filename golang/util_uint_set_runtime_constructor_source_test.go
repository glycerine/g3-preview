// Source: src/util/uint_set.h:35-40; src/util/vector.h:125-129.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2006-12-07.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestUintSetRuntimeCopyConstructorUnwindsCompletedBase(t *testing.T) {
	observed := false
	for ordinal := int64(0); ordinal < 24; ordinal++ {
		b := arena.NewBudget(1 << 24)
		r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
		heapMust(t, e)
		src, e := NewRuntimeUintSet(r, b)
		heapMust(t, e)
		heapMust(t, src.Insert(511))
		b.FailAfter(ordinal)
		out, e := src.Clone(r, b)
		if e != nil && out != nil {
			intHeapActualRefusal(t, r, e)
			if out.stopped == nil || (!out.sourceDone && out.destructorStop == nil) {
				t.Fatal("missing retained source constructor cause/unwind phase", ordinal, e)
			}
			if _, probe := out.Contains(0); probe == nil {
				t.Fatal("failed ctor abort context is ordinary usable owner")
			}
			// A later allocation refusal proves a completed base already owned a
			// nonempty vector. Its source destructor must have run before return.
			if out.words.raw.pendingReplaces || out.words.raw.pending != (memory.Block{}) || ordinal >= 4 {
				observed = true
				if out.words.raw.block != (memory.Block{}) && out.destructorStop == nil {
					t.Fatal("copy ctor omitted base source destructor", ordinal, out.words.raw.block)
				}
			}
		}
		uintSetRuntimeAudit(t, r, src, out)
		b.FailAfter(-1)
		if out != nil {
			if stop := out.Close(); stop != nil {
				heapMust(t, out.CleanupStopped())
			}
		}
		heapMust(t, src.Close())
		heapMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
		if observed {
			break
		}
	}
	if !observed {
		t.Fatal("no actual later copy refusal")
	}
}
