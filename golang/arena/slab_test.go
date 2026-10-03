package arena

import (
	"errors"
	"math"
	"math/rand"
	"testing"
)

type record struct {
	Value    uint64
	Children [3]uint32
}

func TestSlabLifetime(t *testing.T) {
	s, err := NewSlab[record](Config{PageSize: 2, SlotLimit: 5})
	if err != nil {
		t.Fatal(err)
	}
	a, g, err := s.Allocate()
	if err != nil || a == 0 || g == 0 {
		t.Fatal(a, g, err)
	}
	if err = s.Write(a, record{Value: 42}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if _, _, err = s.Allocate(); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err = s.Allocate(); !errors.Is(err, ErrSlotLimit) {
		t.Fatal(err)
	}
	v, err := s.Read(a)
	if err != nil || v.Value != 42 {
		t.Fatal(v, err)
	}
	if err = s.Recycle(a); err != nil {
		t.Fatal(err)
	}
	if s.IsLive(a, g) {
		t.Fatal("dead token live")
	}
	b, h, err := s.Allocate()
	if err != nil || a != b || h <= g {
		t.Fatal(b, h, err)
	}
	v, _ = s.Read(b)
	if v != (record{}) {
		t.Fatal("record not cleared")
	}
	s.Reset()
	s.TrimFreePages()
	if s.Stats().Live != 0 || s.Stats().RecordBytes != 0 {
		t.Fatal(s.Stats())
	}
	for i := 0; i < 5; i++ {
		slot, gen, e := s.Allocate()
		if e != nil {
			t.Fatal(e)
		}
		if slot == a && gen <= h {
			t.Fatal("generation reset")
		}
	}
	s.Close()
	s.Close()
	if _, _, err = s.Allocate(); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}

func TestSlabFailureAtomic(t *testing.T) {
	b := NewBudget(1 << 20)
	s, err := NewSlab[record](Config{PageSize: 2, Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	b.FailAfter(0)
	before := s.Stats()
	if _, _, err = s.Allocate(); !errors.Is(err, ErrInjectedFailure) {
		t.Fatal(err)
	}
	if s.Stats() != before || b.Used() != 0 {
		t.Fatal("failed growth changed state", s.Stats(), b.Used())
	}
	b.FailAfter(-1)
	a, g, err := s.Allocate()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Write(a, record{Value: 9}); err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.Allocate(); err != nil {
		t.Fatal(err)
	}
	b.FailAfter(0)
	before = s.Stats()
	if _, _, err = s.Allocate(); !errors.Is(err, ErrInjectedFailure) {
		t.Fatal(err)
	}
	if s.Stats() != before || !s.IsLive(a, g) {
		t.Fatal("growth corrupted live state")
	}
	s.Reset()
	before = s.Stats()
	if _, _, err = s.Allocate(); !errors.Is(err, ErrInjectedFailure) {
		t.Fatal(err)
	}
	if s.Stats() != before {
		t.Fatal("failed page recovery changed free list")
	}
	s.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestSlabPointerFreedom(t *testing.T) {
	for _, err := range []error{newSlabError[*int](), newSlabError[string](), newSlabError[[]uint32](), newSlabError[struct{ X [2]*int }](), newSlabError[uintptr]()} {
		if !errors.Is(err, ErrPointerful) {
			t.Fatal(err)
		}
	}
	if _, err := NewSlab[record](Config{PageSize: 3}); !errors.Is(err, ErrConfig) {
		t.Fatal(err)
	}
}
func newSlabError[T any]() error { _, err := NewSlab[T](Config{}); return err }

func TestSlabGenerationExhaustion(t *testing.T) {
	s, _ := NewSlab[record](Config{PageSize: 1, SlotLimit: 1})
	a, _, _ := s.Allocate()
	s.pages[0].meta[0].generation = Generation(math.MaxUint32)
	if err := s.Recycle(a); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Allocate(); !errors.Is(err, ErrSlotLimit) {
		t.Fatal(err)
	}
	s.Reset()
	s.TrimFreePages()
	if _, _, err := s.Allocate(); !errors.Is(err, ErrSlotLimit) {
		t.Fatal("retired slot revived", err)
	}
}

func TestSlabStateMachine(t *testing.T) {
	s, _ := NewSlab[record](Config{PageSize: 2, SlotLimit: 17})
	defer s.Close()
	live := map[Slot]Generation{}
	values := map[Slot]record{}
	history := map[Slot]Generation{}
	rng := rand.New(rand.NewSource(713))
	for step := 0; step < 4000; step++ {
		switch rng.Intn(5) {
		case 0, 1:
			a, g, e := s.Allocate()
			if e != nil {
				if len(live) != 17 || !errors.Is(e, ErrSlotLimit) {
					t.Fatal(step, e)
				}
				break
			}
			if _, ok := live[a]; ok || g <= history[a] {
				t.Fatal("identity reuse", a, g)
			}
			live[a] = g
			history[a] = g
			v := record{Value: uint64(step)}
			values[a] = v
			if e = s.Write(a, v); e != nil {
				t.Fatal(e)
			}
		case 2:
			for a := Slot(1); a <= 17; a++ {
				if _, ok := live[a]; ok {
					if e := s.Recycle(a); e != nil {
						t.Fatal(e)
					}
					delete(live, a)
					delete(values, a)
					break
				}
			}
		case 3:
			s.TrimFreePages()
		case 4:
			if step%23 == 0 {
				s.Reset()
				clear(live)
				clear(values)
			}
		}
		if s.Stats().Live != uint32(len(live)) {
			t.Fatal(step, s.Stats())
		}
		for a := Slot(1); a <= 17; a++ {
			g, ok := live[a]
			if s.IsLive(a, history[a]) != ok {
				t.Fatal("liveness", step, a, g)
			}
			v, e := s.Read(a)
			if ok {
				if e != nil || v != values[a] {
					t.Fatal(step, a, v, e)
				}
			} else if !errors.Is(e, ErrInvalidSlot) {
				t.Fatal(step, a, e)
			}
		}
	}
}

func TestSlabWarmedAllocations(t *testing.T) {
	s, _ := NewSlab[record](Config{PageSize: 2})
	a, _, _ := s.Allocate()
	s.Recycle(a)
	n := testing.AllocsPerRun(1000, func() {
		a, _, e := s.Allocate()
		if e != nil {
			panic(e)
		}
		if e = s.Write(a, record{Value: 4}); e != nil {
			panic(e)
		}
		if _, e = s.Read(a); e != nil {
			panic(e)
		}
		if e = s.Recycle(a); e != nil {
			panic(e)
		}
	})
	if n != 0 {
		t.Fatal("warm allocations", n)
	}
}

func TestBudget(t *testing.T) {
	b := NewBudget(9)
	if e := b.Reserve(9); e != nil {
		t.Fatal(e)
	}
	if e := b.Reserve(1); !errors.Is(e, ErrBudget) {
		t.Fatal(e)
	}
	b.Release(4)
	if b.Used() != 5 {
		t.Fatal(b.Used())
	}
	b.FailAfter(1)
	if e := b.Reserve(1); e != nil {
		t.Fatal(e)
	}
	if e := b.Reserve(1); !errors.Is(e, ErrInjectedFailure) {
		t.Fatal(e)
	}
}

func TestSlabGrowthChargesPeakDirectory(t *testing.T) {
	probe, _ := NewSlab[record](Config{PageSize: 1})
	probe.Allocate()
	one := probe.Stats()
	probe.Allocate()
	two := probe.Stats()
	// Final capacity fits, but copying the page directory also needs its old
	// allocation to remain live until publication of the replacement.
	limit := two.RecordBytes + two.MetadataBytes + two.DirectoryBytes
	b := NewBudget(limit)
	s, _ := NewSlab[record](Config{PageSize: 1, Budget: b})
	s.Allocate()
	if _, _, e := s.Allocate(); !errors.Is(e, ErrBudget) {
		t.Fatal(e)
	}
	if s.Stats() != one || b.Used() != one.RecordBytes+one.MetadataBytes+one.DirectoryBytes {
		t.Fatal("peak rejection changed state")
	}
}

func TestSlabInvalidOperations(t *testing.T) {
	s, _ := NewSlab[record](Config{PageSize: 1})
	a, g, _ := s.Allocate()
	for _, slot := range []Slot{0, a + 1} {
		if _, e := s.Read(slot); !errors.Is(e, ErrInvalidSlot) {
			t.Fatal(e)
		}
		if e := s.Write(slot, record{}); !errors.Is(e, ErrInvalidSlot) {
			t.Fatal(e)
		}
		if e := s.Recycle(slot); !errors.Is(e, ErrInvalidSlot) {
			t.Fatal(e)
		}
	}
	s.Recycle(a)
	if e := s.Recycle(a); !errors.Is(e, ErrInvalidSlot) {
		t.Fatal(e)
	}
	if s.IsLive(a, g) || s.IsLive(a, 0) {
		t.Fatal("dead or zero generation live")
	}
	s.Close()
	if _, e := s.Read(a); !errors.Is(e, ErrClosed) {
		t.Fatal(e)
	}
	if e := s.Write(a, record{}); !errors.Is(e, ErrClosed) {
		t.Fatal(e)
	}
	if e := s.Recycle(a); !errors.Is(e, ErrClosed) {
		t.Fatal(e)
	}
}

func BenchmarkSlabWarm(b *testing.B) {
	s, _ := NewSlab[record](Config{PageSize: 1024})
	a, _, _ := s.Allocate()
	s.Recycle(a)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a, _, _ := s.Allocate()
		s.Write(a, record{Value: uint64(i)})
		s.Read(a)
		s.Recycle(a)
	}
}

// FuzzArenaOps compares bounded byte operation streams with an independent
// live-value/token model. Input truncation only limits operation count.
func FuzzArenaOps(f *testing.F) {
	f.Add([]byte{0, 0, 0, 1, 2, 3, 0, 4, 0, 0, 1})
	f.Add([]byte{0, 1, 0, 1, 0, 1, 2, 0})
	f.Fuzz(func(t *testing.T, ops []byte) {
		if len(ops) > 512 {
			ops = ops[:512]
		}
		s, _ := NewSlab[record](Config{PageSize: 2, SlotLimit: 7})
		defer s.Close()
		live := map[Slot]Generation{}
		latest := map[Slot]Generation{}
		values := map[Slot]record{}
		for i, op := range ops {
			slot := Slot(op/5%7 + 1)
			switch op % 5 {
			case 0:
				a, g, e := s.Allocate()
				if e != nil {
					if len(live) != 7 || !errors.Is(e, ErrSlotLimit) {
						t.Fatal(i, e)
					}
					break
				}
				if _, ok := live[a]; ok || g <= latest[a] {
					t.Fatal("reused token", i, a, g)
				}
				live[a] = g
				latest[a] = g
				values[a] = record{}
			case 1:
				e := s.Recycle(slot)
				if _, ok := live[slot]; ok {
					if e != nil {
						t.Fatal(e)
					}
					delete(live, slot)
					delete(values, slot)
				} else if !errors.Is(e, ErrInvalidSlot) {
					t.Fatal(e)
				}
			case 2:
				s.TrimFreePages()
			case 3:
				s.Reset()
				clear(live)
				clear(values)
			case 4:
				v := record{Value: uint64(i)}
				e := s.Write(slot, v)
				if _, ok := live[slot]; ok {
					if e != nil {
						t.Fatal(e)
					}
					values[slot] = v
				} else if !errors.Is(e, ErrInvalidSlot) {
					t.Fatal(e)
				}
			}
			if s.Stats().Live != uint32(len(live)) {
				t.Fatal("live count", i)
			}
			for a := Slot(1); a <= 7; a++ {
				_, ok := live[a]
				if s.IsLive(a, latest[a]) != ok {
					t.Fatal("liveness", i, a)
				}
				if ok {
					v, e := s.Read(a)
					if e != nil || v != values[a] {
						t.Fatal("value", i, a, v, e)
					}
				}
			}
		}
	})
}
