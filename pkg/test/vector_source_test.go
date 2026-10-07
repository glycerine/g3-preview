// Source: src/test/vector.cpp. Licensed under the MIT license.
package test

import (
	"github.com/glycerine/g3/pkg/arena"

	"errors"

	"testing"
)

func TestSourceVectorTst1(t *testing.T) {
	v, _ := arena.NewVectors32Signed(arena.VectorConfig{})
	defer v.Close()
	ref := arena.VectorRef{}
	if n, _ := v.Len(ref); n != 0 {
		t.Fatal(n)
	}
	for i := uint32(0); i < 1000; i++ {
		var e error
		ref, e = v.Append(ref, int32(i+3))
		if e != nil {
			t.Fatal(e)
		}
		x, e := v.Read(ref, i)
		if e != nil || x != int32(i+3) {
			t.Fatal(i, x, e)
		}
		n, _ := v.Len(ref)
		c, _ := v.Cap(ref)
		if n == 0 || c < n {
			t.Fatal(n, c)
		}
	}
	for i := uint32(0); i < 1000; i++ {
		x, _ := v.Read(ref, i)
		if x != int32(i+3) {
			t.Fatal(i, x)
		}
	}
	for i := uint32(0); i < 1000; i++ {
		n, _ := v.Len(ref)
		x, _ := v.Read(ref, n-1)
		if n != 1000-i || x != int32(1000-i-1+3) {
			t.Fatal(n, x)
		}
		v.Resize(ref, n-1)
	}
	if n, _ := v.Len(ref); n != 0 {
		t.Fatal(n)
	}
	// The source's billion-element unbounded allocation loop is replaced with
	// a deterministic finite budget failure, preserving the allocation error.
	b := arena.NewBudget(4096)
	bounded, _ := arena.NewWords(arena.VectorConfig{PageSize: 2, Descriptors: arena.Config{PageSize: 1, Budget: b}})
	defer bounded.Close()
	if _, e := bounded.Allocate(1000000000, 1000000000); !errors.Is(e, arena.ErrBudget) {
		t.Fatal(e)
	}
}
