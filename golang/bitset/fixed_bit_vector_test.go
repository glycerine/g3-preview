// Licensed under the MIT license.
// Tests translated from src/test/fixed_bit_vector.cpp, Nikolaj Bjorner.
package bitset

import (
	_ "embed"
	"encoding/json"
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math/rand/v2"
	"testing"
)

//go:embed test_vectors/fixed_hash.json
var fixedHashJSON []byte

func fm(t *testing.T, n uint32) *FixedManager {
	t.Helper()
	m, e := NewFixedManager(n, arena.VectorConfig{PageSize: 8, Descriptors: arena.Config{PageSize: 4}})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(m.Close)
	return m
}
func alloc0(t *testing.T, m *FixedManager) FixedRef {
	t.Helper()
	r, e := m.Allocate0()
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func must(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func fget(t *testing.T, m *FixedManager, r FixedRef, i uint32) bool {
	t.Helper()
	v, e := m.Get(r, i)
	must(t, e)
	return v
}
func feq(t *testing.T, m *FixedManager, a, b FixedRef) bool {
	t.Helper()
	v, e := m.Equals(a, b)
	must(t, e)
	return v
}
func TestFixedSourceTst1(t *testing.T) {
	m := fm(t, 30)
	b := alloc0(t, m)
	must(t, m.Set(b, 0, true))
	must(t, m.Set(b, 1, false))
	must(t, m.Set(b, 2, true))
	for _, i := range []uint32{0, 1, 2, 3, 29} {
		if fget(t, m, b, i) != (i == 0 || i == 2) {
			t.Fatal(i)
		}
	}
	must(t, m.Deallocate(b))
}
func TestFixedSourceOr(t *testing.T) {
	m := fm(t, 10)
	a, b := alloc0(t, m), alloc0(t, m)
	must(t, m.Set(a, 4, true))
	for _, i := range []uint32{8, 3, 2, 1} {
		must(t, m.Set(b, i, true))
	}
	must(t, m.Or(a, b))
	if feq(t, m, a, b) {
		t.Fatal("or equality")
	}
	must(t, m.Unset(a, 4))
	if !feq(t, m, a, b) {
		t.Fatal("unset equal")
	}
	must(t, m.Unset(a, 3))
	if feq(t, m, a, b) {
		t.Fatal("unset unequal")
	}
	must(t, m.Deallocate(a))
	must(t, m.Deallocate(b))
}
func TestFixedSourceEq(t *testing.T) {
	for _, n := range []uint32{15, 16, 17, 31, 32, 33, 63, 64, 65} {
		m := fm(t, n)
		a, b, c := alloc0(t, m), alloc0(t, m), alloc0(t, m)
		must(t, m.Set(a, 3, true))
		if feq(t, m, a, b) || !feq(t, m, b, c) {
			t.Fatal(n, "initial equality")
		}
		must(t, m.Set(c, 3, true))
		if !feq(t, m, a, c) {
			t.Fatal(n, "equal after set")
		}
		must(t, m.Set(b, n-1, true))
		must(t, m.Set(c, n-1, true))
		must(t, m.Unset(c, 3))
		if !feq(t, m, b, c) {
			t.Fatal(n, "last bit")
		}
		must(t, m.Fill0(a))
		must(t, m.Neg(a))
		must(t, m.Fill1(b))
		if !feq(t, m, a, b) {
			t.Fatal(n, "neg vs fill")
		}
		must(t, m.Fill0(a))
		for i := uint32(0); i < n; i++ {
			must(t, m.Set(a, i, true))
		}
		if !feq(t, m, a, b) {
			t.Fatal(n, "unused bits equality")
		}
		for _, r := range []FixedRef{a, b, c} {
			must(t, m.Deallocate(r))
		}
	}
}
func TestFixedMaskedWordsAndZero(t *testing.T) {
	for _, n := range []uint32{0, 1, 31, 32, 33, 64, 65} {
		m := fm(t, n)
		a, e := m.Allocate1()
		must(t, e)
		word, e := m.LastWord(a)
		must(t, e)
		mask := uint32(0xffffffff)
		if n%32 != 0 {
			mask = (uint32(1) << (n % 32)) - 1
		}
		want := mask
		if n == 0 {
			want = 0
		}
		if m.Mask() != mask || word != want || m.NumWords() != uint32((uint64(n)+31)/32) || m.NumBytes() != m.NumWords()*4 || m.NumBits() != n {
			t.Fatal(n, "word/mask/size", word, mask)
		}
		if n == 0 {
			b := alloc0(t, m)
			if a != b {
				t.Fatal("source m_0 shared")
			}
		}
		must(t, m.Deallocate(a))
	}
}
func TestFixedRangeAliasTraversal(t *testing.T) {
	m := fm(t, 100)
	a, b := alloc0(t, m), alloc0(t, m)
	for _, i := range []uint32{0, 3, 31, 32, 63} {
		must(t, m.Set(b, i, true))
	}
	must(t, m.SetRange(a, b, 95, 32))
	for i := uint32(0); i < 100; i++ {
		want := i == 32 || i == 35 || i == 63 || i == 64 || i == 95
		if fget(t, m, a, i) != want {
			t.Fatal(i)
		}
	}
	must(t, m.Fill0(a))
	must(t, m.SetRange(a, b, 67, 4))
	for i := uint32(0); i < 100; i++ {
		want := i == 4 || i == 7 || i == 35 || i == 36 || i == 67
		if fget(t, m, a, i) != want {
			t.Fatal(i)
		}
	}
	// Source deliberately copies forward, so overlapping self assignment cascades.
	must(t, m.Fill0(a))
	must(t, m.Set(a, 0, true))
	must(t, m.SetRange(a, a, 10, 1))
	for i := uint32(0); i < 11; i++ {
		if !fget(t, m, a, i) {
			t.Fatal("self assignment must cascade", i)
		}
	}
}
func TestFixedOwnershipAndLifetime(t *testing.T) {
	m := fm(t, 65)
	a := alloc0(t, m)
	must(t, m.Set(a, 64, true))
	b, e := m.Clone(a)
	must(t, e)
	must(t, m.Unset(b, 64))
	if !fget(t, m, a, 64) {
		t.Fatal("clone aliases")
	}
	must(t, m.Assign(b, a))
	if !feq(t, m, a, b) {
		t.Fatal("assign")
	}
	must(t, m.Deallocate(a))
	if _, e := m.Get(a, 0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("stale ref", e)
	}
	c := alloc0(t, m)
	if c == a {
		t.Fatal("recycled generation")
	}
	m.Reset()
	if _, e := m.Get(c, 0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("reset invalidates", e)
	}
	d := alloc0(t, m)
	must(t, m.ResetVector(d))
	m.Close()
	if _, e := m.Allocate(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed", e)
	}
}
func TestFixedSourceHash(t *testing.T) {
	var vectors []struct{ Bits, Want uint32 }
	must(t, json.Unmarshal(fixedHashJSON, &vectors))
	for _, v := range vectors {
		m := fm(t, v.Bits)
		a := alloc0(t, m)
		for i, w := range []uint32{0x80000001, 0xf0000003, 7} {
			if uint32(i) < m.NumWords() {
				must(t, m.WriteWord(a, uint32(i), w))
			}
		}
		h, e := m.Hash(a)
		must(t, e)
		if h != v.Want {
			t.Fatal(v.Bits, h, v.Want)
		}
	}
}
func TestFixedBoundedOracle(t *testing.T) {
	r := rand.New(rand.NewPCG(29, 35))
	for _, n := range []uint32{0, 1, 17, 32, 33, 65, 101} {
		m := fm(t, n)
		a, b := alloc0(t, m), alloc0(t, m)
		aa, bb := make([]bool, n), make([]bool, n)
		for step := 0; step < 600; step++ {
			switch r.IntN(8) {
			case 0:
				if n > 0 {
					i := uint32(r.IntN(int(n)))
					val := r.IntN(2) == 0
					must(t, m.Set(a, i, val))
					aa[i] = val
				}
			case 1:
				must(t, m.Neg(a))
				for i := range aa {
					aa[i] = !aa[i]
				}
			case 2:
				must(t, m.Fill0(b))
				clear(bb)
			case 3:
				must(t, m.Fill1(b))
				for i := range bb {
					bb[i] = true
				}
			case 4:
				must(t, m.And(a, b))
				for i := range aa {
					aa[i] = aa[i] && bb[i]
				}
			case 5:
				must(t, m.Or(a, b))
				for i := range aa {
					aa[i] = aa[i] || bb[i]
				}
			case 6:
				must(t, m.Copy(b, a))
				copy(bb, aa)
			case 7:
				if n > 0 {
					lo := uint32(r.IntN(int(n)))
					hi := lo + uint32(r.IntN(int(n-lo)))
					must(t, m.SetRange(a, b, hi, lo))
					for i := lo; i <= hi; i++ {
						aa[i] = bb[i-lo]
					}
				}
			}
			equal, contains := true, true
			for i := range aa {
				if fget(t, m, a, uint32(i)) != aa[i] {
					t.Fatal(n, step, i)
				}
				equal = equal && aa[i] == bb[i]
				contains = contains && (!bb[i] || aa[i])
			}
			eq, e := m.Equals(a, b)
			must(t, e)
			ct, e := m.Contains(a, b)
			must(t, e)
			if eq != equal || ct != contains {
				t.Fatal(n, step, "relation")
			}
		}
	}
}
func TestFixedBudgetsAndReuse(t *testing.T) {
	budget := arena.NewBudget(1 << 20)
	cfg := arena.VectorConfig{PageSize: 8, Descriptors: arena.Config{PageSize: 4, Budget: budget}}
	m, e := NewFixedManager(65, cfg)
	must(t, e)
	t.Cleanup(m.Close)
	root := budget.Used()
	if root == 0 {
		t.Fatal("constructor/root unbudgeted")
	}
	a := alloc0(t, m)
	must(t, m.Set(a, 2, true))
	b := alloc0(t, m) // Both payload blocks are live, so the next allocate must reserve storage.
	budget.FailAfter(0)
	if _, e := m.Allocate(); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("failure", e)
	}
	if !fget(t, m, a, 2) {
		t.Fatal("failure changed live source")
	}
	budget.FailAfter(-1)
	must(t, m.Deallocate(a))
	must(t, m.Deallocate(b))
	slots := m.Stats().Descriptors.Slots
	var operationErr error
	if n := testing.AllocsPerRun(100, func() {
		r, e := m.Allocate1()
		if e != nil {
			operationErr = e
			return
		}
		if _, e = m.Hash(r); e != nil {
			operationErr = e
		}
		if e = m.Neg(r); e != nil {
			operationErr = e
		}
		if e = m.Deallocate(r); e != nil {
			operationErr = e
		}
	}); n != 0 {
		t.Fatal("warmed fixed allocator/ops allocate", n)
	}
	must(t, operationErr)
	if m.Stats().Live != 0 || m.Stats().Descriptors.Slots != slots {
		t.Fatal("pool failed to reuse bounded capacity", m.Stats())
	}
	m.Reset()
	if budget.Used() < root {
		t.Fatal("reset released live manager root")
	}
	m.Close()
	if budget.Used() != 0 {
		t.Fatal("close budget", budget.Used())
	}
	m.Close()
	limited := arena.NewBudget(0)
	if _, e := NewFixedManager(0, arena.VectorConfig{Descriptors: arena.Config{Budget: limited}}); !errors.Is(e, arena.ErrBudget) {
		t.Fatal("zero root must be budgeted", e)
	}
	invalid := arena.NewBudget(1 << 20)
	if _, e := NewFixedManager(30, arena.VectorConfig{PageSize: 3, Descriptors: arena.Config{Budget: invalid}}); !errors.Is(e, arena.ErrConfig) || invalid.Used() != 0 {
		t.Fatal("failed constructor budget", e, invalid.Used())
	}
}
func TestFixedSourceWordSelfAliasAndDisplay(t *testing.T) {
	m := fm(t, 100)
	a := alloc0(t, m)
	must(t, m.Set(a, 0, true))
	must(t, m.SetRange(a, a, 95, 32))
	for _, i := range []uint32{0, 32, 64} {
		if !fget(t, m, a, i) {
			t.Fatal("aligned forward self copy", i)
		}
	}
	text, e := m.String(a)
	must(t, e)
	if len(text) != 100 || text[99] != '1' || text[67] != '1' || text[35] != '1' {
		t.Fatal("MSB-first display", text)
	}
	if _, e := m.Word(a, m.NumWords()); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("bounds", e)
	}
	if e := m.SetRange(a, a, 100, 0); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("range", e)
	}
	if _, e := m.Get(a, 100); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("get bounds", e)
	}
}
func TestFixedSourceWidthOverflowBoundary(t *testing.T) {
	budget := arena.NewBudget(1 << 20)
	cfg := arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}
	for _, n := range []uint32{^uint32(0), ^uint32(0) - 30} {
		if _, e := NewFixedManager(n, cfg); !errors.Is(e, arena.ErrBounds) || budget.Used() != 0 {
			t.Fatal(n, e, budget.Used())
		}
	}
	m, e := NewFixedManager(^uint32(0)-31, cfg)
	must(t, e)
	if m.NumWords() != 134217727 || m.NumBytes() != 536870908 || m.NumBits() != 4294967264 {
		t.Fatal("highest source width metadata")
	}
	m.Close()
	if budget.Used() != 0 {
		t.Fatal("constructor boundary budget")
	}
}
