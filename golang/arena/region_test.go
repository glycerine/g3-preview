package arena

import (
	"errors"
	"math/rand"
	"testing"
)

func TestRegionNestedLifetime(t *testing.T) {
	r, _ := NewRegion[uint32](Config{PageSize: 2})
	defer r.Close(nil)
	base, _ := r.Allocate(1)
	outer, _ := r.Mark()
	a, _ := r.Allocate(2)
	inner, _ := r.Mark()
	b, _ := r.Allocate(3)
	if e := r.Rollback(outer, nil); !errors.Is(e, ErrMark) {
		t.Fatal(e)
	}
	if e := r.Rollback(inner, nil); e != nil {
		t.Fatal(e)
	}
	if r.IsLive(b) || !r.IsLive(a) || !r.IsLive(base) {
		t.Fatal("wrong rollback boundary")
	}
	if e := r.Rollback(inner, nil); !errors.Is(e, ErrMark) {
		t.Fatal(e)
	}
	if e := r.Rollback(outer, nil); e != nil {
		t.Fatal(e)
	}
	newer, _ := r.Allocate(7)
	if newer.Slot == a.Slot && newer.Generation <= a.Generation {
		t.Fatal("generation revived")
	}
	if x, e := r.Read(base); e != nil || x != 1 {
		t.Fatal(x, e)
	}
	foreign, _ := NewRegion[uint32](Config{})
	defer foreign.Close(nil)
	fm, _ := foreign.Mark()
	if e := r.Rollback(fm, nil); !errors.Is(e, ErrMark) {
		t.Fatal(e)
	}
}

func TestRegionCleanupBeforeRecycle(t *testing.T) {
	r, _ := NewRegion[uint32](Config{PageSize: 2})
	defer r.Close(nil)
	m, _ := r.Mark()
	refs := make([]RegionRef, 4)
	for i := range refs {
		refs[i], _ = r.Allocate(uint32(i + 1))
	}
	index := 3
	e := r.Rollback(m, func(ref RegionRef, value uint32) {
		if ref != refs[index] || value != uint32(index+1) {
			t.Fatal("wrong cleanup order", ref, value, index)
		}
		for _, x := range refs {
			if !r.IsLive(x) {
				t.Fatal("record recycled before all cleanup")
			}
		}
		if _, e := r.Allocate(9); !errors.Is(e, ErrReentrant) {
			t.Fatal(e)
		}
		if _, e := r.Mark(); !errors.Is(e, ErrReentrant) {
			t.Fatal(e)
		}
		if e := r.Reset(nil); !errors.Is(e, ErrReentrant) {
			t.Fatal(e)
		}
		if e := r.Write(ref, 88); !errors.Is(e, ErrReentrant) {
			t.Fatal(e)
		}
		for _, e := range []error{r.Close(nil), r.TrimFreePages(), r.PopScopes(0, nil), r.Rollback(m, nil)} {
			if !errors.Is(e, ErrReentrant) {
				t.Fatal(e)
			}
		}
		index--
	})
	if e != nil || index != -1 {
		t.Fatal(e, index)
	}
	for _, x := range refs {
		if r.IsLive(x) {
			t.Fatal("cleanup did not recycle")
		}
	}
}

func TestRegionPanicPreservesStorage(t *testing.T) {
	r, _ := NewRegion[uint32](Config{})
	defer r.Close(nil)
	m, _ := r.Mark()
	a, _ := r.Allocate(3)
	before := r.Stats()
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected callback panic")
			}
		}()
		r.Rollback(m, func(RegionRef, uint32) { panic("broken caller invariant") })
	}()
	if r.Stats() != before || !r.IsLive(a) {
		t.Fatal("panic changed storage")
	}
	// Caller cleanup side effects are not transactional. No repeated cleanup
	// guarantee follows a panic; nil explicitly means caller already cleaned.
	if e := r.Rollback(m, nil); e != nil {
		t.Fatal(e)
	}
}

func TestRegionMultipleScopesAndReset(t *testing.T) {
	r, _ := NewRegion[uint32](Config{PageSize: 1})
	base, _ := r.Allocate(1)
	m, _ := r.Mark()
	a, _ := r.Allocate(2)
	r.Mark()
	b, _ := r.Allocate(3)
	if e := r.PopScopes(3, nil); !errors.Is(e, ErrMark) {
		t.Fatal(e)
	}
	if e := r.PopScopes(2, nil); e != nil {
		t.Fatal(e)
	}
	if !r.IsLive(base) || r.IsLive(a) || r.IsLive(b) {
		t.Fatal("multi pop")
	}
	if e := r.Rollback(m, nil); !errors.Is(e, ErrMark) {
		t.Fatal(e)
	}
	count := 0
	if e := r.Reset(func(ref RegionRef, value uint32) {
		if ref != base || value != 1 {
			t.Fatal(ref, value)
		}
		count++
	}); e != nil || count != 1 {
		t.Fatal(e, count)
	}
	r.TrimFreePages()
	if r.Stats().Storage.RecordBytes != 0 {
		t.Fatal(r.Stats())
	}
	r.Close(nil)
	r.Close(nil)
	if _, e := r.Allocate(8); !errors.Is(e, ErrClosed) {
		t.Fatal(e)
	}
}

func TestRegionBudgetFailure(t *testing.T) {
	b := NewBudget(1 << 20)
	r, _ := NewRegion[uint32](Config{PageSize: 1, Budget: b})
	b.FailAfter(0)
	before := r.Stats()
	if _, e := r.Mark(); !errors.Is(e, ErrInjectedFailure) {
		t.Fatal(e)
	}
	if r.Stats() != before {
		t.Fatal("mark failure changed state")
	}
	if _, e := r.Allocate(1); !errors.Is(e, ErrInjectedFailure) {
		t.Fatal(e)
	}
	if r.Stats() != before {
		t.Fatal("log failure changed state")
	}
	b.FailAfter(-1)
	a, _ := r.Allocate(7)
	r.Mark()
	before = r.Stats()
	b.FailAfter(0)
	if _, e := r.Mark(); !errors.Is(e, ErrInjectedFailure) {
		t.Fatal(e)
	}
	if r.Stats() != before || !r.IsLive(a) {
		t.Fatal("mark growth failure")
	}
	r.Close(nil)
	if b.Used() != 0 {
		t.Fatal("budget leak", b.Used())
	}
}

// Independent undo-state oracle for the executable reverse dispatch ordering
// in util/trail.h::undo_trail_stack. Entries are scalars, never stored closures.
func TestRegionUndoStateMachine(t *testing.T) {
	type undo struct{ Old uint32 }
	r, _ := NewRegion[undo](Config{PageSize: 2})
	defer r.Close(nil)
	var marks []RegionMark
	var checkpoints []uint32
	var state uint32
	rng := rand.New(rand.NewSource(719))
	cleanup := func(_ RegionRef, u undo) { state = u.Old }
	for i := 0; i < 3000; i++ {
		switch rng.Intn(4) {
		case 0:
			m, e := r.Mark()
			if e != nil {
				t.Fatal(e)
			}
			marks = append(marks, m)
			checkpoints = append(checkpoints, state)
		case 1:
			_, e := r.Allocate(undo{state})
			if e != nil {
				t.Fatal(e)
			}
			state = uint32(i + 1)
		case 2:
			if len(marks) > 0 {
				last := len(marks) - 1
				if e := r.Rollback(marks[last], cleanup); e != nil {
					t.Fatal(e)
				}
				if state != checkpoints[last] {
					t.Fatal("undo differs", i, state, checkpoints[last])
				}
				marks = marks[:last]
				checkpoints = checkpoints[:last]
			}
		case 3:
			r.TrimFreePages()
		}
	}
	if e := r.Reset(cleanup); e != nil || state != 0 {
		t.Fatal(e, state)
	}
}

func TestRegionWarmAllocations(t *testing.T) {
	r, _ := NewRegion[uint32](Config{PageSize: 2})
	m, _ := r.Mark()
	r.Allocate(2)
	r.Rollback(m, nil)
	n := testing.AllocsPerRun(1000, func() {
		m, e := r.Mark()
		if e != nil {
			panic(e)
		}
		_, e = r.Allocate(4)
		if e != nil {
			panic(e)
		}
		if e = r.Rollback(m, nil); e != nil {
			panic(e)
		}
	})
	if n != 0 {
		t.Fatal(n)
	}
	r.Close(nil)
}

func TestRegionGenerationAcrossResetTrim(t *testing.T) {
	r, _ := NewRegion[uint64](Config{PageSize: 1, SlotLimit: 1})
	defer r.Close(nil)
	m, _ := r.Mark()
	a, _ := r.Allocate(11)
	r.Reset(nil)
	r.TrimFreePages()
	b, e := r.Allocate(22)
	if e != nil || b.Slot != a.Slot || b.Generation <= a.Generation || r.IsLive(a) {
		t.Fatal(a, b, e)
	}
	if _, e = r.Read(a); !errors.Is(e, ErrInvalidSlot) {
		t.Fatal(e)
	}
	if e = r.Write(a, 5); !errors.Is(e, ErrInvalidSlot) {
		t.Fatal(e)
	}
	if e = r.Rollback(m, nil); !errors.Is(e, ErrMark) {
		t.Fatal(e)
	}
}

func TestRegionBudgetPeakLogs(t *testing.T) {
	probe, _ := NewRegion[uint32](Config{PageSize: 2})
	probe.Allocate(1)
	one := probe.Stats()
	probe.Allocate(2)
	two := probe.Stats()
	probe.Close(nil)
	total := two.Storage.RecordBytes + two.Storage.MetadataBytes + two.Storage.DirectoryBytes + two.LogBytes + two.MarkBytes
	budget := NewBudget(total)
	r, _ := NewRegion[uint32](Config{PageSize: 2, Budget: budget})
	defer r.Close(nil)
	r.Allocate(1)
	if _, e := r.Allocate(2); !errors.Is(e, ErrBudget) {
		t.Fatal("missing log copy peak", e)
	}
	if r.Stats() != one {
		t.Fatal("failed log growth changed state", r.Stats(), one)
	}
}

func TestRegionMarkPeakAndExhaustion(t *testing.T) {
	probe, _ := NewRegion[uint32](Config{})
	probe.Mark()
	one := probe.Stats()
	probe.Mark()
	two := probe.Stats()
	probe.Close(nil)
	budget := NewBudget(two.MarkBytes)
	r, _ := NewRegion[uint32](Config{Budget: budget})
	defer r.Close(nil)
	r.Mark()
	if _, e := r.Mark(); !errors.Is(e, ErrBudget) {
		t.Fatal("missing mark copy peak", e)
	}
	if r.Stats() != one {
		t.Fatal("failed mark changed state")
	}
	r.serial = ^uint64(0)
	if _, e := r.Mark(); !errors.Is(e, ErrSlotLimit) {
		t.Fatal(e)
	}
	r.Reset(nil)
	if _, e := r.Mark(); !errors.Is(e, ErrSlotLimit) {
		t.Fatal("serial exhaustion revived", e)
	}
}

func BenchmarkRegionWarm(b *testing.B) {
	r, _ := NewRegion[uint32](Config{})
	m, _ := r.Mark()
	r.Allocate(2)
	r.Rollback(m, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m, _ := r.Mark()
		r.Allocate(uint32(i))
		r.Rollback(m, nil)
	}
}

// Independent scalar undo oracle, bounded to 512 operations. Permanent seed
// cases are ordinary tests too; mutation never needs C++ or external fixtures.
func FuzzRegionTrailOps(f *testing.F) {
	f.Add([]byte{0, 1, 1, 0, 1, 2, 2, 3})
	f.Add([]byte{1, 0, 0, 2, 1, 3, 2})
	f.Fuzz(func(t *testing.T, ops []byte) {
		if len(ops) > 512 {
			ops = ops[:512]
		}
		r, _ := NewRegion[uint32](Config{PageSize: 2})
		defer r.Close(nil)
		var marks []RegionMark
		var states []uint32
		state := uint32(0)
		cleanup := func(_ RegionRef, old uint32) { state = old }
		for i, op := range ops {
			switch op % 4 {
			case 0:
				m, e := r.Mark()
				if e != nil {
					t.Fatal(e)
				}
				marks = append(marks, m)
				states = append(states, state)
			case 1:
				if _, e := r.Allocate(state); e != nil {
					t.Fatal(e)
				}
				state = uint32(i + 1)
			case 2:
				if len(marks) > 0 {
					last := len(marks) - 1
					if e := r.Rollback(marks[last], cleanup); e != nil {
						t.Fatal(e)
					}
					if state != states[last] {
						t.Fatal("state", i, state, states[last])
					}
					marks = marks[:last]
					states = states[:last]
				}
			case 3:
				if e := r.Reset(cleanup); e != nil || state != 0 {
					t.Fatal("reset", e, state)
				}
				marks = nil
				states = nil
			}
		}
	})
}
