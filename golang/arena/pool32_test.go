package arena

import (
	"errors"
	"math"
	"math/rand"
	"reflect"
	"sync"
	"testing"
)

// This projects source object_allocator lifetime/counter assertions onto a
// concrete scalar policy. Source rational coefficient assertions remain pending
// a real rational-cell manager; these integers are not fake rational fixtures.
func TestPool32SourceLifetimeProjection(t *testing.T) {
	constructed, reset, destroyed := 0, 0, 0
	p, e := NewPool32(Pool32Config{PageSize: 2}, Pool32Policy{Construct: func() uint32 { constructed++; return 31 }, Reset: func(v uint32) uint32 { reset++; return v ^ 128 }, Destroy: func(uint32) { destroyed++ }})
	if e != nil {
		t.Fatal(e)
	}
	if p.Stats().Workers != 8 || p.Stats().Pages != 8 || constructed != 0 {
		t.Fatal(p.Stats(), constructed)
	}
	a, _ := p.Allocate()
	p.Write(a, 10)
	p.Allocate()
	if p.Stats().Pages != 9 {
		t.Fatal("no eager empty boundary page", p.Stats())
	}
	if e = p.Recycle(a); e != nil {
		t.Fatal(e)
	}
	b, _ := p.Allocate()
	if b.RegionWorker != a.RegionWorker || b.Slot != a.Slot || b.Generation <= a.Generation || constructed != 2 || reset != 1 || destroyed != 0 {
		t.Fatal(a, b, constructed, reset, destroyed)
	}
	if v, e := p.Read(b); e != nil || v != 10^128 {
		t.Fatal("retained reset state lost", v, e)
	}
	p.Recycle(b)
	p.Close()
	p.Close()
	if constructed != destroyed || reset != 2 {
		t.Fatal(constructed, reset, destroyed)
	}
}

func TestPool32DoNothingReset(t *testing.T) {
	p, _ := NewPool32(Pool32Config{Workers: 1, PageSize: 2}, Pool32Policy{})
	defer p.Close()
	a, _ := p.Allocate()
	p.Write(a, math.MaxUint32)
	p.Recycle(a)
	b, _ := p.Allocate()
	if v, e := p.Read(b); e != nil || v != math.MaxUint32 {
		t.Fatal("do-nothing recycle cleared constructed state", v, e)
	}
	if p.IsLive(a) {
		t.Fatal("old token revived")
	}
}

func TestPool32WorkerDestinationAndModes(t *testing.T) {
	p, _ := NewPool32(Pool32Config{Workers: 2, PageSize: 2}, Pool32Policy{})
	defer p.Close()
	w, e := p.Worker(1)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = w.Allocate(); !errors.Is(e, ErrPoolMode) {
		t.Fatal(e)
	}
	p.EnableConcurrent(true)
	a, e := w.Allocate()
	if e != nil {
		t.Fatal(e)
	}
	p.Write(a, 47)
	if e = p.RecycleWorker(0, a); e != nil {
		t.Fatal(e)
	}
	b, e := p.AllocateWorker(0)
	if e != nil || b.RegionWorker != 1 || b.Slot != a.Slot {
		t.Fatal("cross-worker free-list destination lost", a, b, e)
	}
	if v, _ := p.Read(b); v != 47 {
		t.Fatal(v)
	}
	if e = p.ReserveWorkers(3); !errors.Is(e, ErrPoolMode) {
		t.Fatal(e)
	}
	if e = p.Reset(); !errors.Is(e, ErrPoolMode) {
		t.Fatal(e)
	}
	if _, e = p.Allocate(); !errors.Is(e, ErrPoolMode) {
		t.Fatal(e)
	}
	if _, e = p.AllocateWorker(2); !errors.Is(e, ErrPoolWorker) {
		t.Fatal(e)
	}
	p.EnableConcurrent(false)
	if e = p.ReserveWorkers(3); e != nil {
		t.Fatal(e)
	}
	if e = p.Recycle(b); e != nil {
		t.Fatal(e)
	}
	p.EnableConcurrent(true)
	if _, e = w.Allocate(); e != nil {
		t.Fatal("worker wrapper invalidated across mode/reserve", e)
	}
}

func TestPool32DestructorOrderResetLedger(t *testing.T) {
	n := uint32(0)
	var order []uint32
	p, _ := NewPool32(Pool32Config{Workers: 1, PageSize: 2}, Pool32Policy{Construct: func() uint32 { n++; return n }, Destroy: func(v uint32) { order = append(order, v) }})
	refs := make([]Pool32Ref, 5)
	for i := range refs {
		refs[i], _ = p.Allocate()
	}
	p.Recycle(refs[1])
	if e := p.Reset(); e != nil {
		t.Fatal(e)
	}
	expected := []uint32{5, 1, 2, 3, 4}
	if len(order) != len(expected) {
		t.Fatal(order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Fatal("source last-partial-then-earlier-pages order", order)
		}
	}
	for _, a := range refs {
		if p.IsLive(a) {
			t.Fatal("reset kept token")
		}
	}
	b, _ := p.Allocate()
	if b.Slot != refs[0].Slot || b.Generation <= refs[0].Generation {
		t.Fatal(b, refs[0])
	}
	p.Close()
	if len(order) != 6 || order[5] != 6 {
		t.Fatal("reset recreated object not destroyed exactly once", order)
	}
}

func TestPool32RetirementAndDestructorPolicy(t *testing.T) {
	destroyed := 0
	p, _ := NewPool32(Pool32Config{Workers: 1, PageSize: 1, SlotLimit: 1}, Pool32Policy{Construct: func() uint32 { return 93 }, Destroy: func(v uint32) {
		if v != 93 {
			t.Fatal(v)
		}
		destroyed++
	}})
	a, _ := p.Allocate()
	p.workers[0].pages[0].meta[0].generation = math.MaxUint32
	a.Generation = math.MaxUint32
	if e := p.Recycle(a); e != nil {
		t.Fatal(e)
	}
	if destroyed != 0 {
		t.Fatal("retirement destroyed recycled constructed object")
	}
	if _, e := p.Allocate(); !errors.Is(e, ErrSlotLimit) {
		t.Fatal(e)
	}
	p.Reset()
	if destroyed != 1 {
		t.Fatal(destroyed)
	}
	if _, e := p.Allocate(); !errors.Is(e, ErrSlotLimit) {
		t.Fatal("reset revived retired token", e)
	}
	p.Close()
	if destroyed != 1 {
		t.Fatal("retired object destroyed twice", destroyed)
	}
	p, _ = NewPool32(Pool32Config{Workers: 1, PageSize: 1, DisableDestructors: true}, Pool32Policy{Destroy: func(uint32) { t.Fatal("disabled destructor called") }})
	p.Allocate()
	p.Reset()
	p.Allocate()
	p.Close()
}

func TestPool32PredictableFailuresPrecedeCallbacks(t *testing.T) {
	b := NewBudget(1 << 20)
	construct, reset, destroy := 0, 0, 0
	p, e := NewPool32(Pool32Config{Workers: 1, PageSize: 1, Budget: b}, Pool32Policy{Construct: func() uint32 { construct++; return 7 }, Reset: func(v uint32) uint32 { reset++; return v }, Destroy: func(uint32) { destroy++ }})
	if e != nil {
		t.Fatal(e)
	}
	b.FailAfter(0)
	before := p.Stats()
	if _, e = p.Allocate(); !errors.Is(e, ErrInjectedFailure) {
		t.Fatal(e)
	}
	if construct != 0 || p.Stats() != before {
		t.Fatal("boundary failure published state", construct, p.Stats())
	}
	b.FailAfter(-1)
	a, _ := p.Allocate()
	b.FailAfter(0)
	before = p.Stats()
	if e = p.Recycle(a); !errors.Is(e, ErrInjectedFailure) {
		t.Fatal(e)
	}
	if reset != 0 || !p.IsLive(a) || p.Stats() != before {
		t.Fatal("recycle failed after reset")
	}
	if e = p.Reset(); !errors.Is(e, ErrInjectedFailure) {
		t.Fatal(e)
	}
	if destroy != 0 || !p.IsLive(a) {
		t.Fatal("reset failure destroyed object")
	}
	p.Close()
	if construct != destroy || b.Used() != 0 {
		t.Fatal(construct, destroy, b.Used())
	}
	b.FailAfter(0)
	if p, e = NewPool32(Pool32Config{Workers: 2, Budget: b}, Pool32Policy{}); !errors.Is(e, ErrInjectedFailure) || p != nil || b.Used() != 0 {
		t.Fatal(p, e, b.Used())
	}
}

func TestPool32IndependentStateMachine(t *testing.T) {
	fresh := uint32(0)
	destroyed := uint32(0)
	p, _ := NewPool32(Pool32Config{Workers: 2, PageSize: 2}, Pool32Policy{Construct: func() uint32 { fresh++; return fresh }, Reset: func(v uint32) uint32 { return v ^ 0x80000000 }, Destroy: func(uint32) { destroyed++ }})
	defer p.Close()
	p.EnableConcurrent(true)
	live := map[Pool32Ref]uint32{}
	physical := map[[2]uint32]uint32{}
	r := rand.New(rand.NewSource(762))
	for i := 0; i < 2000; i++ {
		switch r.Intn(4) {
		case 0, 1:
			before := fresh
			a, e := p.AllocateWorker(uint32(r.Intn(2)))
			if e != nil {
				t.Fatal(e)
			}
			key := [2]uint32{a.RegionWorker, uint32(a.Slot)}
			expected, exists := physical[key]
			if !exists {
				if fresh != before+1 {
					t.Fatal("missing constructor")
				}
				expected = fresh
			} else if fresh != before {
				t.Fatal("reconstructed recycled object")
			}
			v, e := p.Read(a)
			if e != nil || v != expected {
				t.Fatal(a, v, expected, e)
			}
			value := uint32(i)
			p.Write(a, value)
			live[a] = value
			physical[key] = value
		case 2:
			var selected Pool32Ref
			for a := range live {
				if selected.Slot == 0 || a.RegionWorker < selected.RegionWorker || (a.RegionWorker == selected.RegionWorker && a.Slot < selected.Slot) {
					selected = a
				}
			}
			if selected.Slot != 0 {
				a := selected
				value := live[a]
				if e := p.RecycleWorker(uint32(r.Intn(2)), a); e != nil {
					t.Fatal(e)
				}
				delete(live, a)
				physical[[2]uint32{a.RegionWorker, uint32(a.Slot)}] = value ^ 0x80000000
			}
		case 3:
			if i%31 == 0 {
				p.EnableConcurrent(false)
				if e := p.Reset(); e != nil {
					t.Fatal(e)
				}
				clear(live)
				clear(physical)
				p.EnableConcurrent(true)
			}
		}
		if p.Stats().Live != uint32(len(live)) {
			t.Fatal(i, p.Stats())
		}
		for a, x := range live {
			v, e := p.Read(a)
			if e != nil || v != x || !p.IsLive(a) {
				t.Fatal(i, a, v, x, e)
			}
		}
	}
	p.Close()
	if fresh != destroyed {
		t.Fatal("source active counter projection", fresh, destroyed)
	}
}

func TestPool32WarmReuse(t *testing.T) {
	p, _ := NewPool32(Pool32Config{Workers: 1, PageSize: 2}, Pool32Policy{})
	a, _ := p.Allocate()
	p.Recycle(a)
	n := testing.AllocsPerRun(1000, func() {
		a, e := p.Allocate()
		if e != nil {
			panic(e)
		}
		if e = p.Recycle(a); e != nil {
			panic(e)
		}
	})
	if n != 0 {
		t.Fatal(n)
	}
	p.Close()
}

func TestPool32ResetAtGenerationLimit(t *testing.T) {
	destroyed := 0
	p, _ := NewPool32(Pool32Config{Workers: 1, PageSize: 1, SlotLimit: 1}, Pool32Policy{Destroy: func(uint32) { destroyed++ }})
	defer p.Close()
	p.Allocate()
	p.workers[0].pages[0].meta[0].generation = math.MaxUint32
	p.Reset()
	if _, e := p.Allocate(); !errors.Is(e, ErrSlotLimit) {
		t.Fatal("reset wrapped exhausted generation", e)
	}
	if destroyed != 1 {
		t.Fatal(destroyed)
	}
}

func TestPool32ConcurrentWorkerPartitions(t *testing.T) {
	p, _ := NewPool32(Pool32Config{Workers: 2, PageSize: 2}, Pool32Policy{})
	defer p.Close()
	p.EnableConcurrent(true)
	var wg sync.WaitGroup
	fail := make(chan error, 2)
	for worker := uint32(0); worker < 2; worker++ {
		wg.Add(1)
		go func(worker uint32) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				a, e := p.AllocateWorker(worker)
				if e != nil {
					fail <- e
					return
				}
				if e = p.Write(a, worker+19); e != nil {
					fail <- e
					return
				}
				x, e := p.Read(a)
				if e != nil {
					fail <- e
					return
				}
				if x != worker+19 {
					fail <- errors.New("worker data overlapped")
					return
				}
				if e = p.RecycleWorker(worker, a); e != nil {
					fail <- e
					return
				}
			}
		}(worker)
	}
	wg.Wait()
	close(fail)
	for e := range fail {
		t.Fatal(e)
	}
	if p.Stats().Live != 0 || p.Stats().Constructed != 2 {
		t.Fatal(p.Stats())
	}
}

func BenchmarkPool32Warm(b *testing.B) {
	p, _ := NewPool32(Pool32Config{Workers: 1}, Pool32Policy{})
	a, _ := p.Allocate()
	p.Recycle(a)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a, _ := p.Allocate()
		p.Recycle(a)
	}
}

func TestPool32SourceDefaultsAndRecordAudit(t *testing.T) {
	p, e := NewPool32(Pool32Config{}, Pool32Policy{})
	if e != nil {
		t.Fatal(e)
	}
	defer p.Close()
	s := p.Stats()
	if s.Workers != 8 || s.Pages != 8 || s.RecordBytes != 8*1024*4 || s.Constructed != 0 {
		t.Fatal(s)
	}
	for _, typ := range []reflect.Type{reflect.TypeOf(Pool32Ref{}), reflect.TypeOf(pool32Meta{}), reflect.TypeOf(uint32(0))} {
		if e := ValidateRecordType(typ); e != nil {
			t.Fatal(typ, e)
		}
	}
	if e := ValidateRecordType(reflect.TypeOf(Pool32Policy{})); !errors.Is(e, ErrPointerful) {
		t.Fatal("cold callback policy mistaken for hot record", e)
	}
}

func TestPool32ResetPeakBudget(t *testing.T) {
	probe, _ := NewPool32(Pool32Config{Workers: 1, PageSize: 2}, Pool32Policy{})
	s := probe.Stats()
	probe.Close()
	limit := s.RecordBytes + s.MetadataBytes + s.DirectoryBytes + s.FreeListBytes
	b := NewBudget(limit)
	destroyed := 0
	p, e := NewPool32(Pool32Config{Workers: 1, PageSize: 2, Budget: b}, Pool32Policy{Destroy: func(uint32) { destroyed++ }})
	if e != nil {
		t.Fatal(e)
	}
	a, _ := p.Allocate()
	before := p.Stats()
	if e = p.Reset(); !errors.Is(e, ErrBudget) {
		t.Fatal("reset omitted old+new page peak", e)
	}
	if p.Stats() != before || !p.IsLive(a) || destroyed != 0 {
		t.Fatal("peak failure changed constructed state")
	}
	if e = p.ReserveWorkers(2); !errors.Is(e, ErrBudget) {
		t.Fatal(e)
	}
	if p.Stats() != before {
		t.Fatal("failed reserve changed workers")
	}
	p.Close()
	if destroyed != 1 || b.Used() != 0 {
		t.Fatal(destroyed, b.Used())
	}
}
