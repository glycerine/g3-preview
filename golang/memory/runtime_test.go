// Licensed under the MIT license.
package memory

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"testing"
)

func memoryRuntimeFixture(t *testing.T, tls bool, bits uint8) (*Runtime, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	m, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: tls, SizeTBits: bits})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := m.Close(); e != nil {
			t.Error(e)
		}
		if b.Used() != 0 {
			t.Error("retained budget", b.Used())
		}
	})
	return m, b
}
func memoryRuntimeRequire(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func memoryRuntimeCode(t *testing.T, e error, want uint32) {
	t.Helper()
	var z Failure
	if !errors.As(e, &z) || z.ErrorCode() != want {
		t.Fatalf("error %v want code %d", e, want)
	}
}
func TestMemoryRuntimeActualBlocksHeadersAndReallocate(t *testing.T) {
	for _, tls := range []bool{false, true} {
		for _, bits := range []uint8{32, 64} {
			m, _ := memoryRuntimeFixture(t, tls, bits)
			h := uint64(bits / 8)
			a, e := m.Allocate(19)
			memoryRuntimeRequire(t, e)
			for i := uint64(0); i < 19; i++ {
				memoryRuntimeRequire(t, m.Write(a, i, byte(i+1)))
			}
			n, e := m.Size(a)
			memoryRuntimeRequire(t, e)
			if n != 19 {
				t.Fatal(n)
			}
			if s := m.Stats(); s.LiveBlocks != 1 || s.PhysicalBytes != 19+h {
				t.Fatal(s)
			}
			memoryRuntimeRequire(t, m.Reallocate(&a, 37))
			memoryRuntimeRequire(t, m.Reallocate(&a, 7))
			for i := uint64(0); i < 7; i++ {
				v, e := m.Read(a, i)
				memoryRuntimeRequire(t, e)
				if v != byte(i+1) {
					t.Fatal(i, v)
				}
			}
			if s := m.Stats(); s.PhysicalBytes != 7+h || s.LiveBlocks != 1 {
				t.Fatal(s)
			}
			memoryRuntimeRequire(t, m.Deallocate(&a))
			if a != (Block{}) || m.Stats().PhysicalBytes != 0 {
				t.Fatal(a, m.Stats())
			}
		}
	}
}
func TestMemoryRuntimeTLSStrictThresholdAndRepeatedCounts(t *testing.T) {
	m, _ := memoryRuntimeFixture(t, true, 64)
	a, e := m.Allocate(99992)
	memoryRuntimeRequire(t, e)
	s := m.State()
	if s.GlobalSize != 0 || s.LocalSize != 100000 || s.GlobalCount != 0 || s.LocalCount != 1 {
		t.Fatal(s)
	}
	b, e := m.Allocate(0)
	memoryRuntimeRequire(t, e)
	s = m.State()
	if s.GlobalSize != 100008 || s.LocalSize != 0 || s.GlobalCount != 2 || s.LocalCount != 2 || s.Peak != 100008 {
		t.Fatal(s)
	}
	memoryRuntimeRequire(t, m.Deallocate(&a))
	s = m.State()
	if s.GlobalSize != 100008 || s.LocalSize != -100000 {
		t.Fatal(s)
	}
	memoryRuntimeRequire(t, m.Deallocate(&b))
	s = m.State()
	if s.GlobalSize != 0 || s.LocalSize != 0 || s.GlobalCount != 4 || s.LocalCount != 2 {
		t.Fatal(s)
	}
}
func TestMemoryRuntimeLimitBeforePhysicalAndTLSOwnedFailure(t *testing.T) {
	for _, tls := range []bool{false, true} {
		m, _ := memoryRuntimeFixture(t, tls, 64)
		memoryRuntimeRequire(t, m.SetMaxSize(1))
		n := uint64(2)
		if tls {
			n = 100000
		}
		a, e := m.Allocate(n)
		memoryRuntimeCode(t, e, uint32(101))
		s := m.State()
		if !s.OutOfMemory || s.GlobalSize != int64(n+8) || s.GlobalCount != 1 {
			t.Fatal(s)
		}
		if tls {
			if a == (Block{}) || m.Stats().PhysicalBytes != n+8 {
				t.Fatal(a, m.Stats())
			}
			memoryRuntimeRequire(t, m.Deallocate(&a))
		} else {
			if a != (Block{}) || m.Stats().PhysicalBytes != 0 {
				t.Fatal(a, m.Stats())
			}
		}
	}
	m, _ := memoryRuntimeFixture(t, false, 64)
	memoryRuntimeRequire(t, m.SetMaxCount(1))
	a, e := m.Allocate(0)
	memoryRuntimeRequire(t, e)
	b, e := m.Allocate(0)
	memoryRuntimeCode(t, e, uint32(113))
	if b != (Block{}) || m.State().OutOfMemory {
		t.Fatal(b, m.State())
	}
	memoryRuntimeRequire(t, m.Deallocate(&a))
}
func TestMemoryRuntimeFailedReallocateKeepsActualOldBlock(t *testing.T) {
	for _, tls := range []bool{false, true} {
		m, b := memoryRuntimeFixture(t, tls, 64)
		a, e := m.Allocate(20)
		memoryRuntimeRequire(t, e)
		memoryRuntimeRequire(t, m.Write(a, 0, 71))
		old := a
		b.FailAfter(0)
		e = m.Reallocate(&a, 44)
		b.FailAfter(-1)
		memoryRuntimeCode(t, e, uint32(101))
		if a != old {
			t.Fatal(a, old)
		}
		v, e := m.Read(a, 0)
		memoryRuntimeRequire(t, e)
		if v != 71 {
			t.Fatal(v)
		}
		s := m.State()
		if !s.OutOfMemory || s.LocalCount+s.GlobalCount != 2 {
			t.Fatal(s)
		}
		if tls {
			if s.LocalSize != 52 || s.GlobalSize != 0 {
				t.Fatal(s)
			}
		} else {
			if s.GlobalSize != 52 {
				t.Fatal(s)
			}
		}
		if m.Stats().PhysicalBytes != 28 {
			t.Fatal(m.Stats())
		}
		memoryRuntimeRequire(t, m.Deallocate(&a))
		s = m.State()
		if tls {
			if s.LocalSize != 24 {
				t.Fatal(s)
			}
		} else {
			if s.GlobalSize != 24 {
				t.Fatal(s)
			}
		}
	}
}
func TestMemoryRuntimeSourceUnsignedDeltaWatermarkAndGuards(t *testing.T) {
	m, b := memoryRuntimeFixture(t, true, 32)
	a, e := m.Allocate(20)
	memoryRuntimeRequire(t, e)
	memoryRuntimeRequire(t, m.Reallocate(&a, 12))
	s := m.State()
	if s.GlobalSize != 4294967312 || s.LocalSize != 0 || s.GlobalCount != 2 {
		t.Fatal(s)
	}
	w, _ := memoryRuntimeFixture(t, false, 64)
	wa, e := w.Allocate(20)
	memoryRuntimeRequire(t, e)
	memoryRuntimeRequire(t, w.SetWatermark(28))
	above, e := w.AboveWatermark()
	memoryRuntimeRequire(t, e)
	if above {
		t.Fatal("equality")
	}
	memoryRuntimeRequire(t, w.SetWatermark(27))
	above, e = w.AboveWatermark()
	memoryRuntimeRequire(t, e)
	if !above {
		t.Fatal("strict inequality")
	}
	memoryRuntimeRequire(t, w.Deallocate(&wa))
	if e = m.Close(); !errors.Is(e, ErrRuntimeLive) {
		t.Fatal(e)
	}
	memoryRuntimeRequire(t, m.Deallocate(&a))
	if e = m.Deallocate(&a); !errors.Is(e, ErrRuntimeOwner) {
		t.Fatal(e)
	}
	b.FailAfter(0)
	_, e = m.Allocate(1)
	b.FailAfter(-1)
	memoryRuntimeCode(t, e, uint32(101))
	if m.State().GlobalCount != 2 || m.State().LocalCount != 2 {
		t.Fatal("TLS malloc refusal changed counters", m.State())
	}
}

func TestMemoryRuntimeReallocateGenerationRefusalPrefix(t *testing.T) {
	m, _ := memoryRuntimeFixture(t, false, 64)
	a, e := m.Allocate(7)
	memoryRuntimeRequire(t, e)
	m.blocks[a.slot-1].generation = ^uint32(0)
	a.generation = ^uint32(0)
	before := m.State()
	physical := m.Stats()
	if e = m.Reallocate(&a, 11); !errors.Is(e, arena.ErrSlotLimit) {
		t.Fatal(e)
	}
	if m.State() != before || m.Stats() != physical {
		t.Fatal("token exhaustion changed source/physical prefix", m.State(), m.Stats())
	}
	memoryRuntimeRequire(t, m.Deallocate(&a))
}
func TestMemoryRuntimeActualGettersAndUninitializedBytes(t *testing.T) {
	m, _ := memoryRuntimeFixture(t, false, 64)
	a, e := m.Allocate(1)
	memoryRuntimeRequire(t, e)
	if _, e = m.Read(a, 0); !errors.Is(e, ErrRuntimeUninitialized) {
		t.Fatal(e)
	}
	memoryRuntimeRequire(t, m.Write(a, 0, 91))
	old := a
	memoryRuntimeRequire(t, m.Reallocate(&a, 73))
	if _, e = m.Read(old, 0); !errors.Is(e, ErrRuntimeOwner) {
		t.Fatal(e)
	}
	if _, e = m.Read(a, 72); !errors.Is(e, ErrRuntimeUninitialized) {
		t.Fatal(e)
	}
	v, e := m.Read(a, 0)
	memoryRuntimeRequire(t, e)
	if v != 91 {
		t.Fatal(v)
	}
	size, e := m.AllocationSize()
	memoryRuntimeRequire(t, e)
	peak, e := m.MaximumUsed()
	memoryRuntimeRequire(t, e)
	count, e := m.AllocationCount()
	memoryRuntimeRequire(t, e)
	if size != 81 || peak != 81 || count != 2 {
		t.Fatal(size, peak, count)
	}
	memoryRuntimeRequire(t, m.Deallocate(&a))
	size, e = m.AllocationSize()
	memoryRuntimeRequire(t, e)
	peak, e = m.MaximumUsed()
	memoryRuntimeRequire(t, e)
	count, e = m.AllocationCount()
	memoryRuntimeRequire(t, e)
	if size != 0 || peak != 81 || count != 2 {
		t.Fatal(size, peak, count)
	}
}

func TestMemoryRuntimeIndependentPhysicalAndCounterModel(t *testing.T) {
	for _, tls := range []bool{false, true} {
		m, b := memoryRuntimeFixture(t, tls, 64)
		var handles [8]Block
		var sizes [8]uint64
		var gs, ls, gc, lc, peak int64
		account := func(delta int64, allocating bool) {
			if allocating {
				if tls {
					ls += delta
					lc++
				} else {
					gs += delta
					gc++
					if gs > peak {
						peak = gs
					}
				}
			} else {
				if tls {
					ls -= delta
				} else {
					gs -= delta
				}
			}
			if tls && ((allocating && ls > 100000) || (!allocating && ls < -100000)) {
				gs += ls
				gc += lc
				if gs > peak {
					peak = gs
				}
				ls = 0
			}
		}
		seed := uint32(991)
		for step := 0; step < 1200; step++ {
			seed = 1664525*seed + 1013904223
			i := (seed >> 8) % 8
			op := (seed >> 20) % 3
			n := uint64(seed % 110003)
			if handles[i] == (Block{}) {
				h, e := m.Allocate(n)
				memoryRuntimeRequire(t, e)
				handles[i] = h
				sizes[i] = n
				account(int64(n+8), true)
			} else if op == 0 {
				memoryRuntimeRequire(t, m.Deallocate(&handles[i]))
				account(int64(sizes[i]+8), false)
				sizes[i] = 0
			} else {
				old := sizes[i]
				memoryRuntimeRequire(t, m.Reallocate(&handles[i], n))
				sizes[i] = n
				account(int64(n)-int64(old), true)
			}
			var physical uint64
			for j, h := range handles {
				if h != (Block{}) {
					size, e := m.Size(h)
					memoryRuntimeRequire(t, e)
					if size != sizes[j] {
						t.Fatal(step, j, size, sizes[j])
					}
					physical += sizes[j] + 8
				}
			}
			state := m.State()
			if state.GlobalSize != gs || state.LocalSize != ls || state.GlobalCount != gc || state.LocalCount != lc || state.Peak != peak {
				t.Fatal(step, state, gs, ls, gc, lc, peak)
			}
			actual := uint64(reflect.TypeOf(*m).Size()) + uint64(cap(m.blocks))*uint64(reflect.TypeOf(memoryRuntimeBlock{}).Size())
			for j := range m.blocks {
				x := &m.blocks[j]
				if x.live {
					actual += uint64(len(x.data)) + uint64(len(x.initialized))*8
				}
			}
			if m.Stats().PhysicalBytes != physical || b.Used() != actual {
				t.Fatal(step, m.Stats(), physical, b.Used(), actual)
			}
		}
		for i, h := range handles {
			if h != (Block{}) {
				memoryRuntimeRequire(t, m.Deallocate(&handles[i]))
				account(int64(sizes[i]+8), false)
			}
		}
		actual, e := m.AllocationSize()
		memoryRuntimeRequire(t, e)
		expected := gs
		if expected < 0 {
			expected = 0
		}
		if actual != uint64(expected) {
			t.Fatal(actual, expected)
		}
	}
}
func TestMemoryRuntimeImmediateFaultGraphAndOwnerDomains(t *testing.T) {
	for _, tls := range []bool{false, true} {
		for prefix := int64(0); prefix < 4; prefix++ {
			m, b := memoryRuntimeFixture(t, tls, 64)
			b.FailAfter(prefix)
			h, e := m.Allocate(73)
			b.FailAfter(-1)
			if e != nil {
				memoryRuntimeCode(t, e, uint32(101))
				if h != (Block{}) {
					t.Fatal(h)
				}
			} else {
				if h == (Block{}) {
					t.Fatal(h)
				}
			}
			if b.Used() != m.Stats().TotalBytes {
				t.Fatal("immediate physical roots", prefix, b.Used(), m.Stats())
			}
			state := m.State()
			if e != nil && tls {
				if state.LocalCount != 0 || state.GlobalCount != 0 {
					t.Fatal(state)
				}
			} else if state.LocalCount+state.GlobalCount != 1 {
				t.Fatal(state)
			}
			if h != (Block{}) {
				memoryRuntimeRequire(t, m.Deallocate(&h))
			}
		}
	}
	b := arena.NewBudget(1024)
	b.FailAfter(0)
	m, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	b.FailAfter(-1)
	if m != nil || !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != 0 {
		t.Fatal(m, e, b.Used())
	}
	m, b = memoryRuntimeFixture(t, false, 64)
	shallow := *m
	if _, e = shallow.Allocate(1); !errors.Is(e, ErrRuntimeOwner) {
		t.Fatal(e)
	}
	memoryRuntimeRequire(t, m.Close())
	if _, e = m.Allocate(1); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestMemoryRuntimeWrappedHeaderActualPhysicalPrefix(t *testing.T) {
	for _, tls := range []bool{false, true} {
		m, b := memoryRuntimeFixture(t, tls, 32)
		h, e := m.Allocate(uint64(^uint32(0)) - 2)
		if !errors.Is(e, arena.ErrBounds) || h == (Block{}) {
			t.Fatal("missing completed malloc before bad header", h, e)
		}
		if m.Stats().PhysicalBytes != 1 || m.Stats().LiveBlocks != 1 || b.Used() != m.Stats().TotalBytes {
			t.Fatal(m.Stats(), b.Used())
		}
		s := m.State()
		if tls {
			if s.LocalCount != 0 || s.GlobalCount != 0 {
				t.Fatal(s)
			}
		} else {
			if s.GlobalCount != 1 || s.GlobalSize != 1 {
				t.Fatal(s)
			}
		}
		if e = m.Deallocate(&h); !errors.Is(e, arena.ErrBounds) {
			t.Fatal(e)
		}
		memoryRuntimeRequire(t, m.CleanupPartial(&h))
		h, e = m.Allocate(3)
		memoryRuntimeRequire(t, e)
		old := h
		e = m.Reallocate(&h, uint64(^uint32(0))-2)
		if !errors.Is(e, arena.ErrBounds) || h == old {
			t.Fatal("old backing was not replaced before unsafe header store", h, old, e)
		}
		if m.Stats().PhysicalBytes != 1 || m.Stats().LiveBlocks != 1 || b.Used() != m.Stats().TotalBytes {
			t.Fatal(m.Stats(), b.Used())
		}
		if _, e = m.Read(old, 0); !errors.Is(e, ErrRuntimeOwner) {
			t.Fatal(e)
		}
		memoryRuntimeRequire(t, m.CleanupPartial(&h))
	}
}

func TestMemoryRuntimeShrinkRegrowByteLifetime(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		for _, keep := range []uint64{0, 1, 7, 55, 56, 57, 59, 60, 61, 63, 64, 65} {
			m, _ := memoryRuntimeFixture(t, false, bits)
			h, e := m.Allocate(99)
			memoryRuntimeRequire(t, e)
			for i := uint64(0); i < 99; i++ {
				memoryRuntimeRequire(t, m.Write(h, i, byte(i+17)))
			}
			memoryRuntimeRequire(t, m.Reallocate(&h, keep))
			memoryRuntimeRequire(t, m.Reallocate(&h, 99))
			for i := uint64(0); i < 99; i++ {
				v, e := m.Read(h, i)
				if i < keep {
					memoryRuntimeRequire(t, e)
					if v != byte(i+17) {
						t.Fatal(bits, keep, i, v)
					}
				} else if !errors.Is(e, ErrRuntimeUninitialized) {
					t.Fatal(bits, keep, i, v, e)
				}
			}
			memoryRuntimeRequire(t, m.Deallocate(&h))
		}
	}
}
