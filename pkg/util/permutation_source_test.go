// Source: src/util/permutation.h; src/test/permutation.cpp.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package util

import (
	"errors"
	"reflect"
	"testing"
)

func TestPermutationCyclesExhaustiveIndependentCopy(t *testing.T) {
	for n := 0; n <= 8; n++ {
		p := make([]uint32, n)
		for i := range p {
			p[i] = uint32(i)
		}
		var visit func(int)
		visit = func(k int) {
			if k != n {
				for i := k; i < n; i++ {
					p[k], p[i] = p[i], p[k]
					visit(k + 1)
					p[k], p[i] = p[i], p[k]
				}
				return
			}
			data := make([]uint32, n)
			want := make([]uint32, n)
			for i := range data {
				data[i] = uint32(i)*7919 + 31
			}
			for i := range want {
				want[i] = data[p[i]]
			}
			mapping := append([]uint32{}, p...)
			if e := ApplyPermutationUint32(uint32(n), data, mapping); e != nil {
				t.Fatal(n, p, e)
			}
			if !reflect.DeepEqual(data, want) || !reflect.DeepEqual(mapping, p) {
				t.Fatal(n, p, data, want, mapping)
			}
			for i := range data {
				data[i] = uint32(i)*7919 + 31
			}
			if e := ApplyPermutationCoreUint32(uint32(n), data, mapping); e != nil {
				t.Fatal(e)
			}
			if !reflect.DeepEqual(data, want) {
				t.Fatal(data, want)
			}
			for i := range p {
				if mapping[i] != ^p[i] {
					t.Fatal("actual destructive marker", p, mapping)
				}
			}
		}
		visit(0)
	}
}

func TestPermutationCyclesActualMarkBeforeReachedDataFailure(t *testing.T) {
	p := []uint32{1, 0, 2}
	data := []uint32{11}
	if e := ApplyPermutationUint32(2, data, p); !errors.Is(e, ErrBitBounds) {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(p, []uint32{^uint32(1), 0, 2}) || data[0] != 11 {
		t.Fatal(p, data)
	}
	// The preserving wrapper cannot restore after its child stops.
	p = []uint32{0}
	if e := ApplyPermutationUint32(2, nil, p); !errors.Is(e, ErrBitBounds) {
		t.Fatal(e)
	}
	if p[0] != ^uint32(0) {
		t.Fatal(p)
	}
}

func TestPermutationCyclesUnreadAndSignedCountBranches(t *testing.T) {
	if e := ApplyPermutationCoreUint32(0, nil, nil); e != nil {
		t.Fatal(e)
	}
	if e := ApplyPermutationUint32(0, nil, nil); e != nil {
		t.Fatal(e)
	}
	if e := ApplyPermutationCoreUint32(0x80000000, nil, nil); e != nil {
		t.Fatal(e)
	}
	if e := ApplyPermutationUint32(0x80000000, nil, nil); !errors.Is(e, ErrBitBounds) {
		t.Fatal(e)
	}
	// Pre-marked entries skip without reading data. Restoration is literal,
	// even when the caller supplied those markers before the invocation.
	p := []uint32{0xffffffff, 0xfffffffe}
	if e := ApplyPermutationUint32(2, nil, p); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(p, []uint32{0, 1}) {
		t.Fatal(p)
	}
	// Identity cycles do not read data, and counts leave actual trailing words unread.
	p = []uint32{0, 1, 99}
	if e := ApplyPermutationUint32(2, nil, p); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(p, []uint32{0, 1, 99}) {
		t.Fatal(p)
	}
}

func TestPermutationCyclesOverlappingActualBorrow(t *testing.T) {
	p := []uint32{0, 1, 2, 99}
	if e := ApplyPermutationUint32(3, p, p); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(p, []uint32{0, 1, 2, 99}) {
		t.Fatal(p)
	}
	p = []uint32{0, 1, 2, 99}
	if e := ApplyPermutationCoreUint32(3, p, p); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(p, []uint32{^uint32(0), ^uint32(1), ^uint32(2), 99}) {
		t.Fatal(p)
	}
}

func TestPermutationCyclesReachedSourceAssertionAndDuplicateCycle(t *testing.T) {
	p := []uint32{3, 0}
	e := ApplyPermutationCoreUint32(2, nil, p)
	if sourceDebug {
		if !errors.Is(e, ErrBitAssertion) || !reflect.DeepEqual(p, []uint32{3, 0}) {
			t.Fatal(e, p)
		}
	} else {
		if !errors.Is(e, ErrBitBounds) || !reflect.DeepEqual(p, []uint32{^uint32(3), 0}) {
			t.Fatal(e, p)
		}
	}
	p = []uint32{1, 1}
	data := []uint32{11, 22}
	e = ApplyPermutationUint32(2, data, p)
	if !reflect.DeepEqual(data, []uint32{22, 11}) {
		t.Fatal(data)
	}
	if sourceDebug {
		if !errors.Is(e, ErrBitAssertion) || !reflect.DeepEqual(p, []uint32{^uint32(1), ^uint32(1)}) {
			t.Fatal(e, p)
		}
	} else {
		if !errors.Is(e, ErrBitBounds) || !reflect.DeepEqual(p, []uint32{^uint32(1), 1}) {
			t.Fatal(e, p)
		}
	}
}
