// Source: src/util/hash.cpp:26-102.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	uhash "github.com/glycerine/g3/pkg/util/hash"
)

func TestRuntimeBytesSourceLengthAndSignedTail(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	raw, e := r.Allocate(64)
	if e != nil {
		t.Fatal(e)
	}
	for n := uint32(0); n <= 64; n++ {
		s := make([]byte, n)
		for i := range s {
			s[i] = byte(i*73 + 129)
			if e = r.Write(raw, uint64(i), s[i]); e != nil {
				t.Fatal(e)
			}
		}
		got, e := RuntimeBytes(r, raw, 0, n, 17)
		if e != nil {
			t.Fatal(e)
		}
		if got != uhash.String(string(s), 17) {
			t.Fatal(n, got, uhash.String(string(s), 17))
		}
	}
	if e = r.Deallocate(&raw); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestRuntimeBytesReachedDescendingTailRead(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	raw, e := r.Allocate(1)
	if e != nil {
		t.Fatal(e)
	}
	// Source tail starts str[10]. A forward reader instead touches uninitialized str[0].
	_, e = RuntimeBytes(r, raw, 0, 11, 0)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if e = r.Deallocate(&raw); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
}
