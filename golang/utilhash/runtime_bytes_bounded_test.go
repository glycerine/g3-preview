// Source: src/util/hash.cpp:26-102; src/util/fixed_bit_vector.cpp:138-140.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2006-09-11.
package utilhash

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestRuntimeBytesBoundedActualReachedLoads(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	h, e := r.Allocate(16)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = RuntimeBytesBounded(r, h, 0, 6, 12, 0); !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal("first word before later bound", e)
	}
	if e = r.WriteUint32(h, 0, 1); e != nil {
		t.Fatal(e)
	}
	if _, e = RuntimeBytesBounded(r, h, 0, 6, 12, 0); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("second whole word requested bound", e)
	}
	if _, e = RuntimeBytesBounded(r, h, 0, 1, 11, 0); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("descending byte10 before initialized byte0", e)
	}
	got, e := RuntimeBytesBounded(nil, memory.Block{}, ^uint64(0), 0, 0, 9)
	if e != nil || got != String("", 9) {
		t.Fatal("zero no provider access", got, e)
	}
	if e = r.Deallocate(&h); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestInlineWordBytesActualColdAndReadOrder(t *testing.T) {
	var word uint32
	initialized := uint8(0)
	if _, e := InlineWordBytes(&word, &initialized, 4, 12, 0); !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal(e)
	}
	if _, e := InlineWordBytes(&word, &initialized, 4, 11, 0); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("tail10 before word0", e)
	}
	word = ^uint32(0)
	initialized = 0xF
	got, e := InlineWordBytes(&word, &initialized, 4, 1, 8)
	if e != nil || got != String(string([]byte{255}), 8) {
		t.Fatal(got, e)
	}
	got, e = InlineWordBytes(nil, nil, 0, 0, 3)
	if e != nil || got != String("", 3) {
		t.Fatal(got, e)
	}
}
