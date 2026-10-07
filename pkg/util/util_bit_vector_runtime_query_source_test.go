// Source: src/util/bit_vector.h:122-132,191-203; src/util/bit_vector.cpp:225-227.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2006-10-03.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestBitVectorRuntimeScalarAndZeroHashAfterActualTLSPushRefusal(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true})
	heapMust(t, e)
	v, e := NewRuntimeBitVector(r, b)
	heapMust(t, e)
	heapMust(t, r.SetMaxSize(1))
	// Actual caller allocation stays below the TLS synchronization threshold.
	// The source two-word Push allocation crosses it only AFTER physical malloc.
	external, e := r.Allocate(99980)
	heapMust(t, e)
	e = v.Push(true)
	if e == nil || v.pending == (Block{}) || v.block != (Block{}) || v.numBits != 1 {
		t.Fatal("reached pending Push", e, v.numBits, v.pending)
	}
	n, e := v.Size()
	heapMust(t, e)
	if n != 1 {
		t.Fatal(n)
	}
	empty, e := v.Empty()
	heapMust(t, e)
	if empty {
		t.Fatal(empty)
	}
	words, e := v.NumWords()
	heapMust(t, e)
	if words != 1 {
		t.Fatal(words)
	}
	hash, e := v.Hash()
	heapMust(t, e)
	if hash != String("", 0) {
		t.Fatal(hash)
	}
	// Both physical tokens are independent named owners, never Runtime-directory
	// ownership inference. Header/bitmap bytes are counted once.
	ex, e := r.Size(external)
	heapMust(t, e)
	pn, e := r.Size(v.pending)
	heapMust(t, e)
	st := r.Stats()
	physical := ex + pn + 16
	bitmap := ((ex+8+63)/64)*8 + ((pn+8+63)/64)*8
	if st.LiveBlocks != 2 || st.PhysicalBytes != physical || st.InitializationBytes != bitmap || b.Used() != st.TotalBytes+v.ownerBytes {
		t.Fatal("named exact prefix", st, physical, bitmap, b.Used())
	}
	heapMust(t, v.CleanupStopped())
	heapMust(t, r.Deallocate(&external))
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestBitVectorRuntimeScalarAndReachedHashAfterDestructiveAssignRefusal(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	heapMust(t, e)
	v, e := NewRuntimeBitVector(r, b)
	heapMust(t, e)
	src, e := NewRuntimeBitVector(r, b)
	heapMust(t, e)
	heapMust(t, v.Resize(1, true))
	heapMust(t, src.Resize(513, true))
	b.FailAfter(0)
	body := v.Assign(src)
	intHeapActualRefusal(t, r, body)
	n, e := v.Size()
	heapMust(t, e)
	if n != 513 {
		t.Fatal(n)
	}
	empty, e := v.Empty()
	heapMust(t, e)
	if empty {
		t.Fatal(empty)
	}
	words, e := v.NumWords()
	heapMust(t, e)
	if words != 17 {
		t.Fatal(words)
	}
	_, e = v.Hash()
	if !errors.Is(e, ErrRuntimeOwner) {
		t.Fatal("reached dangling data guard instead of sticky body", e, body)
	}
	bitVectorRuntimeAudit(t, r, v, src)
	b.FailAfter(-1)
	heapMust(t, v.CleanupStopped())
	heapMust(t, src.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
