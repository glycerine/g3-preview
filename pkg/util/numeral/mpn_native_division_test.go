package numeral

// Source: src/util/mpn.cpp:158-217,219-350; src/util/mpn.h.

import (
	"math/big"
	"math/rand/v2"
	"testing"
)

func TestNaturalNativeDivisionColdAndWarm(t *testing.T) {
	m := naturalManager(t)
	state, stats, used := m.runtime.State(), m.runtime.Stats(), m.budget.Used()
	m.budget.FailAfter(0)
	defer m.budget.FailAfter(-1)
	rng := rand.New(rand.NewChaCha8([32]byte{9, 23}))
	// Exercise normalization, one- and many-limb division, changing operand sizes,
	// and scratch reuse after shrinking. math/big is an independent test oracle.
	for _, size := range []int{2, 3, 17, 65, 129, 4, 33} {
		for sample := 0; sample < 24; sample++ {
			denominatorSize := 1 + sample%size
			a, b := make([]uint32, size), make([]uint32, denominatorSize)
			for i := range a {
				a[i] = rng.Uint32()
			}
			for i := range b {
				b[i] = rng.Uint32()
			}
			a[len(a)-1] |= 1
			b[len(b)-1] |= 1
			wantA, wantB := oracle(a), oracle(b)
			wq, wr := new(big.Int), new(big.Int)
			wq.QuoRem(wantA, wantB, wr)
			q, r := make([]uint32, size-denominatorSize+1), make([]uint32, denominatorSize)
			if _, e := m.Div(a, b, q, r); e != nil {
				t.Fatal(size, sample, e)
			}
			if oracle(q).Cmp(wq) != 0 || oracle(r).Cmp(wr) != 0 {
				t.Fatal("division mismatch", size, sample)
			}
			var err error
			allocations := testing.AllocsPerRun(5, func() { _, err = m.Div(a, b, q, r) })
			if err != nil || allocations != 0 || oracle(a).Cmp(wantA) != 0 || oracle(b).Cmp(wantB) != 0 {
				t.Fatal("warm division", size, sample, allocations, err)
			}
		}
	}
	if m.runtime.State() != state || m.runtime.Stats() != stats || m.budget.Used() != used {
		t.Fatal("division entered Runtime")
	}
	if e := m.SourceFinalize(); e != nil {
		t.Fatal(e)
	}
	for i := range m.Division {
		if m.Division[i] != nil {
			t.Fatal("source close retained scratch", i)
		}
	}
}

func TestNaturalNativeDivisionResizeClearsRegrowth(t *testing.T) {
	m := naturalManager(t)
	if e := m.ResizeDiv(0, 65); e != nil {
		t.Fatal(e)
	}
	for i := range m.Division[0] {
		m.Division[0][i] = uint32(i + 1)
	}
	backing := &m.Division[0][0]
	if e := m.ResizeDiv(0, 3); e != nil {
		t.Fatal(e)
	}
	if e := m.ResizeDiv(0, 65); e != nil {
		t.Fatal(e)
	}
	if &m.Division[0][0] != backing {
		t.Fatal("discarded reusable workspace")
	}
	for i, v := range m.Division[0] {
		want := uint32(0)
		if i < 3 {
			want = uint32(i + 1)
		}
		if v != want {
			t.Fatal("stale scratch after regrowth", i, v, want)
		}
	}
	if e := m.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	for i := range m.Division {
		if m.Division[i] != nil {
			t.Fatal("owning close retained scratch", i)
		}
	}
}
