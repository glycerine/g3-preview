package arena

import (
	"errors"
	"math/rand"
	"reflect"
	"testing"
)

func bytesRegionTest(t *testing.T, page, alignment uint32, direct bool, b *Budget) *BytesRegion {
	t.Helper()
	r, e := NewBytesRegion(BytesRegionConfig{PageSize: page, Alignment: alignment, DirectOnly: direct, Descriptors: Config{PageSize: 2, Budget: b}})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { r.Close() })
	return r
}
func byteRegionAlloc(t *testing.T, r *BytesRegion, n uint32) BytesRegionRef {
	t.Helper()
	ref, e := r.Allocate(n)
	if e != nil {
		t.Fatal(e)
	}
	return ref
}
func byteRegionOwned(r *BytesRegion) uint64 {
	s := r.Stats()
	d := s.Descriptors
	return s.OwnerBytes + s.PayloadBytes + s.PageHeaderBytes + s.DirectoryBytes + s.LogBytes + s.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
}

func TestBytesRegionSourceDisplayMemStats(t *testing.T) {
	// region.cpp:113 counts only the active page chain, including big pages
	// and pages holding marks. Recycled default pages are not displayed.
	r := bytesRegionTest(t, 16, 8, false, nil)
	check := func(want string) {
		t.Helper()
		if got := r.DisplayMemStats(); got != want {
			t.Fatalf("diagnostic %q, want %q", got, want)
		}
	}
	check("num. pages:      1\n")
	before := r.DisplayMemStats()
	mark, e := r.PushScope() // three pointer widths: big mark and eager default
	if e != nil {
		t.Fatal(e)
	}
	check("num. pages:      3\n")
	byteRegionAlloc(t, r, 16) // dedicated big page and eager default
	check("num. pages:      5\n")
	if e = r.Rollback(mark); e != nil {
		t.Fatal(e)
	}
	check("num. pages:      1\n")
	if r.Stats().FreePages != 2 {
		t.Fatal("expected recycled defaults")
	}
	if before != "num. pages:      1\n" {
		t.Fatal("caller diagnostic changed")
	}
	byteRegionAlloc(t, r, 15)
	byteRegionAlloc(t, r, 1)
	check("num. pages:      2\n")
	if e = r.Reset(); e != nil {
		t.Fatal(e)
	}
	check("num. pages:      1\n")

	// region.cpp:23 debug counts allocations, including zero-size objects,
	// but neither scope bookkeeping nor cached capacities.
	d := bytesRegionTest(t, 16, 8, true, nil)
	if got := d.DisplayMemStats(); got != "num. objects:      0\n" {
		t.Fatal(got)
	}
	byteRegionAlloc(t, d, 0)
	m, e := d.PushScope()
	if e != nil {
		t.Fatal(e)
	}
	byteRegionAlloc(t, d, 100)
	if got := d.DisplayMemStats(); got != "num. objects:      2\n" {
		t.Fatal(got)
	}
	if e = d.Rollback(m); e != nil {
		t.Fatal(e)
	}
	if got := d.DisplayMemStats(); got != "num. objects:      1\n" {
		t.Fatal(got)
	}
	if e = d.Reset(); e != nil {
		t.Fatal(e)
	}
	if got := d.DisplayMemStats(); got != "num. objects:      0\n" {
		t.Fatal(got)
	}
}

func TestBytesRegionSourceReleasePagePolicy(t *testing.T) {
	for _, alignment := range []uint32{4, 8} {
		t.Run(map[uint32]string{4: "pointer4", 8: "pointer8"}[alignment], func(t *testing.T) {
			r := bytesRegionTest(t, 8*alignment, alignment, false, nil)
			s := r.Stats()
			if s.ActivePages != 1 || s.DefaultPages != 1 || s.FreePages != 0 || s.PageHeaderBytes != uint64(alignment) || s.Cursor != 0 {
				t.Fatal("source eager first default", s)
			}
			a := byteRegionAlloc(t, r, 8*alignment-1)
			r.Write(a, 0, 87)
			if r.Stats().Cursor != 8*alignment {
				t.Fatal("fit checked before alignment")
			}
			z := byteRegionAlloc(t, r, 0)
			if !r.IsLive(z) || r.Stats().ActivePages != 2 || r.Stats().Cursor != 0 {
				t.Fatal("zero at strict endpoint requires default page")
			}
			if n, e := r.Size(z); e != nil || n != 0 {
				t.Fatal("zero live descriptor")
			}
			big := byteRegionAlloc(t, r, 8*alignment)
			s = r.Stats()
			if s.BigPages != 1 || s.DefaultPages != 3 || s.ActivePages != 4 || s.Cursor != 0 {
				t.Fatal("big dedicated then eager default", s)
			}
			if v, e := r.Read(a, 0); e != nil || v != 87 {
				t.Fatal("big page lost earlier payload")
			}
			r.Write(big, 8*alignment-1, 91)
			if e := r.Reset(); e != nil {
				t.Fatal(e)
			}
			s = r.Stats()
			if s.ActivePages != 1 || s.DefaultPages != 3 || s.FreePages != 2 || s.BigPages != 0 || s.Live != 0 || s.Cursor != 0 {
				t.Fatal("source reset default recycling", s)
			}
			if r.IsLive(a) || r.IsLive(big) || r.IsLive(z) {
				t.Fatal("reset stale ref")
			}
		})
	}
}
func TestBytesRegionMarksConsumeSourceStorage(t *testing.T) {
	r := bytesRegionTest(t, 32, 8, false, nil)
	outer, e := r.PushScope()
	if e != nil {
		t.Fatal(e)
	}
	if s := r.Stats(); s.Cursor != 24 || s.ActivePages != 1 || s.MarkPayloadBytes != 24 {
		t.Fatal("source mark consumes three pointer widths", s)
	}
	inner, e := r.PushScope()
	if e != nil {
		t.Fatal(e)
	}
	if s := r.Stats(); s.Cursor != 24 || s.ActivePages != 2 || s.MarkPayloadBytes != 48 {
		t.Fatal("mark crosses default page", s)
	}
	if e = r.Rollback(outer); !errors.Is(e, ErrMark) {
		t.Fatal("non-top mark accepted")
	}
	a := byteRegionAlloc(t, r, 1)
	r.Write(a, 0, 77)
	if e = r.Rollback(inner); e != nil {
		t.Fatal(e)
	}
	if r.IsLive(a) || r.Stats().Cursor != 24 || r.Stats().ActivePages != 1 || r.Stats().FreePages != 1 {
		t.Fatal("scope cursor/page restoration")
	}
	if e = r.Rollback(inner); !errors.Is(e, ErrMark) {
		t.Fatal("consumed mark")
	}
	if e = r.Rollback(outer); e != nil {
		t.Fatal(e)
	}
	if r.Stats().Cursor != 0 || r.Stats().Scopes != 0 {
		t.Fatal("outer cursor")
	}
	// Tiny default page smaller than mark: source allocates a big mark page,
	// followed immediately by another default page, then restores pre-mark page.
	tiny := bytesRegionTest(t, 16, 8, false, nil)
	mark, e := tiny.PushScope()
	if e != nil {
		t.Fatal(e)
	}
	if s := tiny.Stats(); s.BigPages != 1 || s.DefaultPages != 2 || s.ActivePages != 3 || s.Cursor != 0 {
		t.Fatal("big internal source mark", s)
	}
	if e = tiny.Rollback(mark); e != nil {
		t.Fatal(e)
	}
	if s := tiny.Stats(); s.BigPages != 0 || s.FreePages != 1 || s.ActivePages != 1 {
		t.Fatal("big mark rollback", s)
	}
}
func TestBytesRegionFreePageLIFOAndGenerationLedger(t *testing.T) {
	r := bytesRegionTest(t, 32, 8, false, nil)
	mark, _ := r.PushScope()
	first := byteRegionAlloc(t, r, 24)
	second := byteRegionAlloc(t, r, 24)
	firstD, _ := r.descriptors.Read(first.Slot)
	secondD, _ := r.descriptors.Read(second.Slot)
	if firstD.page == secondD.page {
		t.Fatal("expected separate default pages")
	}
	if e := r.Rollback(mark); e != nil {
		t.Fatal(e)
	}
	again := byteRegionAlloc(t, r, 31)
	aD, _ := r.descriptors.Read(again.Slot)
	if aD.page != 0 {
		t.Fatal("initial page should fit before alignment")
	}
	next := byteRegionAlloc(t, r, 1)
	nD, _ := r.descriptors.Read(next.Slot)
	if nD.page != firstD.page {
		t.Fatal("source recycle LIFO order")
	}
	if r.IsLive(first) || r.IsLive(second) {
		t.Fatal("page reuse resurrected descriptor")
	}
	r.descriptors.state(next.Slot).generation = ^Generation(0)
	next.Generation = ^Generation(0)
	if e := r.Reset(); e != nil {
		t.Fatal(e)
	}
	if r.descriptors.Stats().Retired != 1 {
		t.Fatal("generation exhaustion not retired")
	}
	later := byteRegionAlloc(t, r, 1)
	if later.Slot == next.Slot {
		t.Fatal("retired descriptor reused")
	}
	if e := ValidateRecordType(reflect.TypeOf(BytesRegionRef{})); e != nil {
		t.Fatal(e)
	}
	if e := ValidateRecordType(reflect.TypeOf(bytesRegionScope{})); e != nil {
		t.Fatal(e)
	}
}
func TestBytesRegionDebugDirectScopesAndZero(t *testing.T) {
	b := NewBudget(1 << 20)
	r := bytesRegionTest(t, 32, 8, true, b)
	if s := r.Stats(); s.ActivePages != 0 || s.DefaultPages != 0 || s.PayloadBytes != 0 {
		t.Fatal("debug ctor must have no page", s)
	}
	z := byteRegionAlloc(t, r, 0)
	a := byteRegionAlloc(t, r, 3)
	r.Write(a, 2, 88)
	mark, e := r.PushScope()
	if e != nil {
		t.Fatal(e)
	}
	if s := r.Stats(); s.DirectObjects != 2 || s.MarkPayloadBytes != 0 || s.PayloadBytes != 3 || s.PageHeaderBytes != 0 || s.Cursor != 0 {
		t.Fatal("source debug object/mark policy", s)
	}
	child := byteRegionAlloc(t, r, 40)
	r.Write(child, 0, 49)
	if e = r.Rollback(mark); e != nil {
		t.Fatal(e)
	}
	if r.IsLive(child) || !r.IsLive(a) || !r.IsLive(z) || r.Stats().DirectObjects != 2 {
		t.Fatal("debug prefix restoration")
	}
	if v, e := r.Read(a, 2); e != nil || v != 88 {
		t.Fatal("debug original payload")
	}
	before := r.Stats()
	if e = r.Reset(); e != nil {
		t.Fatal(e)
	}
	after := r.Stats()
	if after.DirectObjects != 0 || after.PayloadBytes != 0 || after.DirectoryBytes != before.DirectoryBytes || after.LogBytes != before.LogBytes || after.ScopeBytes != before.ScopeBytes {
		t.Fatal("debug reset retained vector capacity", after, before)
	}
	if b.Used() != byteRegionOwned(r) {
		t.Fatal("debug backing accounting")
	}
	r.Close()
	r.Close()
	if b.Used() != 0 {
		t.Fatal("debug close leak")
	}
}

// Independent source cursor/page model, not allocator metadata or helper calls.
type bytePageModel struct {
	size        uint32
	defaultPage bool
}
type byteRegionModel struct {
	alignment, page, cursor uint32
	active                  []bytePageModel
	free                    int
	scopes                  []byteModelScope
	direct                  bool
}
type byteModelScope struct {
	cursor             uint32
	activeLen, liveLen int
}

func (m *byteRegionModel) allocate(n uint32) {
	if m.direct {
		m.active = append(m.active, bytePageModel{size: n})
		return
	}
	round := func(n uint32) uint32 { return (n + m.alignment - 1) / m.alignment * m.alignment }
	if uint64(m.cursor)+uint64(n) < uint64(m.page) {
		m.cursor = round(m.cursor + n)
		return
	}
	addDefault := func() {
		if m.free > 0 {
			m.free--
		}
		m.active = append(m.active, bytePageModel{size: m.page, defaultPage: true})
	}
	if n < m.page {
		addDefault()
		m.cursor = round(n)
	} else {
		m.active = append(m.active, bytePageModel{size: n})
		addDefault()
		m.cursor = 0
	}
}
func (m *byteRegionModel) push(live int) {
	saved := byteModelScope{m.cursor, len(m.active), live}
	if !m.direct {
		m.allocate(3 * m.alignment)
	}
	m.scopes = append(m.scopes, saved)
}
func (m *byteRegionModel) pop() int {
	saved := m.scopes[len(m.scopes)-1]
	m.scopes = m.scopes[:len(m.scopes)-1]
	for _, p := range m.active[saved.activeLen:] {
		if p.defaultPage {
			m.free++
		}
	}
	m.active = m.active[:saved.activeLen]
	m.cursor = saved.cursor
	return saved.liveLen
}
func (m *byteRegionModel) reset() {
	for _, p := range m.active {
		if p.defaultPage {
			m.free++
		}
	}
	m.active = nil
	m.scopes = nil
	m.cursor = 0
	if !m.direct {
		m.free--
		m.active = append(m.active, bytePageModel{size: m.page, defaultPage: true})
	}
}
func TestBytesRegionIndependentByteAndPageModel(t *testing.T) {
	for _, direct := range []bool{false, true} {
		r := bytesRegionTest(t, 64, 8, direct, nil)
		m := byteRegionModel{page: 64, alignment: 8, direct: direct}
		if !direct {
			m.active = []bytePageModel{{size: 64, defaultPage: true}}
		}
		rng := rand.New(rand.NewSource(219))
		refs := []BytesRegionRef{}
		values := [][]byte{}
		dead := []BytesRegionRef{}
		for step := 0; step < 1200; step++ {
			switch rng.Intn(7) {
			case 0, 1, 2:
				n := uint32(rng.Intn(90))
				ref := byteRegionAlloc(t, r, n)
				data := make([]byte, n)
				for i := range data {
					data[i] = byte(rng.Intn(256))
					r.Write(ref, uint32(i), data[i])
				}
				refs = append(refs, ref)
				values = append(values, data)
				m.allocate(n)
			case 3:
				if _, e := r.PushScope(); e != nil {
					t.Fatal(e)
				}
				m.push(len(refs))
			case 4:
				if len(m.scopes) > 0 {
					if e := r.PopScope(); e != nil {
						t.Fatal(e)
					}
					keep := m.pop()
					dead = append(dead, refs[keep:]...)
					refs = refs[:keep]
					values = values[:keep]
				}
			case 5:
				if step%11 == 0 {
					r.Reset()
					m.reset()
					dead = append(dead, refs...)
					refs = nil
					values = nil
				}
			}
			var defaultCount, bigCount uint32
			var payload uint64
			for _, p := range m.active {
				payload += uint64(p.size)
				if p.defaultPage {
					defaultCount++
				} else if !direct {
					bigCount++
				}
			}
			payload += uint64(m.free) * uint64(m.page)
			s := r.Stats()
			active := uint32(len(m.active))
			if direct {
				active = 0
			}
			if s.ActivePages != active || s.FreePages != uint32(m.free) || s.DefaultPages != defaultCount+uint32(m.free) || s.BigPages != bigCount || s.Cursor != m.cursor || s.Live != uint32(len(refs)) || s.Scopes != uint32(len(m.scopes)) || s.PayloadBytes != payload {
				t.Fatalf("step%d direct%v page model mismatch %+v vs %+v", step, direct, s, m)
			}
			occupied := map[[2]uint32]bool{}
			for j, ref := range refs {
				if !r.IsLive(ref) {
					t.Fatal("lost live bytes")
				}
				d, _ := r.descriptors.Read(ref.Slot)
				for i, want := range values[j] {
					got, e := r.Read(ref, uint32(i))
					if e != nil || got != want {
						t.Fatal("byte oracle")
					}
					loc := [2]uint32{d.page, d.offset + uint32(i)}
					if occupied[loc] {
						t.Fatal("overlap")
					}
					occupied[loc] = true
				}
			}
			for _, ref := range dead {
				if r.IsLive(ref) {
					t.Fatal("stale ref resurrected")
				}
			}
		}
	}
}
func TestBytesRegionPredictableFailureCleanup(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for fail := int64(0); fail < 14; fail++ {
			b := NewBudget(1 << 20)
			r := bytesRegionTest(t, 32, 8, direct, b)
			old := byteRegionAlloc(t, r, 1)
			r.Write(old, 0, 77)
			before := r.Stats()
			b.FailAfter(fail)
			ref, e := r.Allocate(80)
			if e != nil {
				if !errors.Is(e, ErrInjectedFailure) {
					t.Fatal(e)
				}
				after := r.Stats()
				if after.Live != before.Live || after.ActivePages != before.ActivePages || after.Cursor != before.Cursor || after.FreePages != before.FreePages {
					t.Fatal("failed page plan published", fail, after, before)
				}
			} else {
				if !r.IsLive(ref) {
					t.Fatal("successful failure sweep allocation")
				}
			}
			if v, e := r.Read(old, 0); e != nil || v != 77 {
				t.Fatal("failure changed existing content")
			}
			if b.Used() != byteRegionOwned(r) {
				t.Fatal("unaccounted prepared/retained storage", fail, b.Used(), byteRegionOwned(r))
			}
			b.FailAfter(-1)
			r.Close()
			if b.Used() != 0 {
				t.Fatal("failure cleanup leaked")
			}
		}
	}
	for fail := int64(0); fail < 10; fail++ {
		b := NewBudget(1 << 20)
		b.FailAfter(fail)
		r, e := NewBytesRegion(BytesRegionConfig{PageSize: 16, Descriptors: Config{PageSize: 2, Budget: b}})
		if e != nil {
			if !errors.Is(e, ErrInjectedFailure) || r != nil || b.Used() != 0 {
				t.Fatal("constructor prefix leak", fail, e, b.Used())
			}
		} else {
			b.FailAfter(-1)
			r.Close()
			if b.Used() != 0 {
				t.Fatal("ctor close")
			}
		}
	}
}
func TestBytesRegionMarksFailureAndValidation(t *testing.T) {
	r := bytesRegionTest(t, 16, 8, false, nil)
	foreign := bytesRegionTest(t, 16, 8, false, nil)
	mark, _ := foreign.PushScope()
	if e := r.Rollback(mark); !errors.Is(e, ErrMark) {
		t.Fatal("foreign mark")
	}
	if e := r.PopScopes(0); e != nil {
		t.Fatal(e)
	}
	if e := r.PopScopes(1); !errors.Is(e, ErrMark) {
		t.Fatal("invalid depth")
	}
	for fail := int64(0); fail < 8; fail++ {
		b := NewBudget(1 << 20)
		a := bytesRegionTest(t, 16, 8, false, b)
		before := a.Stats()
		b.FailAfter(fail)
		_, e := a.PushScope()
		if e != nil {
			if !errors.Is(e, ErrInjectedFailure) {
				t.Fatal(e)
			}
			s := a.Stats()
			if s.Cursor != before.Cursor || s.Scopes != 0 || s.ActivePages != 1 {
				t.Fatal("failed mark changed logical state")
			}
		}
		b.FailAfter(-1)
		a.Close()
		if b.Used() != 0 {
			t.Fatal("mark failure leak")
		}
	}
	ref := byteRegionAlloc(t, r, 0)
	if _, e := r.Read(ref, 0); !errors.Is(e, ErrBounds) {
		t.Fatal("zero bounds")
	}
	r.serial = ^uint64(0)
	if _, e := r.PushScope(); !errors.Is(e, ErrSlotLimit) {
		t.Fatal("serial exhaustion")
	}
	r.Close()
	if _, e := r.Allocate(1); !errors.Is(e, ErrClosed) {
		t.Fatal("closed allocate")
	}
	for _, cfg := range []BytesRegionConfig{{Alignment: 3}, {Alignment: 8, PageSize: 7}, {Alignment: 4, PageSize: 6}, {Descriptors: Config{PageSize: 3}}} {
		b := NewBudget(1 << 20)
		cfg.Descriptors.Budget = b
		if _, e := NewBytesRegion(cfg); !errors.Is(e, ErrConfig) || b.Used() != 0 {
			t.Fatal("invalid configuration", e, b.Used())
		}
	}
}
func TestBytesRegionPlateauAndWarmAllocation(t *testing.T) {
	b := NewBudget(1 << 20)
	r := bytesRegionTest(t, 64, 8, false, b)
	cycle := func() {
		mark, e := r.PushScope()
		if e != nil {
			panic(e)
		}
		for _, n := range [...]uint32{40, 40, 80} {
			if _, e = r.Allocate(n); e != nil {
				panic(e)
			}
		}
		if e = r.Rollback(mark); e != nil {
			panic(e)
		}
	}
	for i := 0; i < 4; i++ {
		cycle()
	}
	baseline := b.Used()
	for i := 0; i < 100; i++ {
		cycle()
		if b.Used() != baseline {
			t.Fatal("bounded scope churn grew backing")
		}
	}
	// Warm cycle deliberately uses retained default pages only; big pages are
	// immediately destroyed by source policy and cannot promise zero allocation.
	warm := func() {
		m, e := r.PushScope()
		if e != nil {
			panic(e)
		}
		if _, e = r.Allocate(8); e != nil {
			panic(e)
		}
		if e = r.Rollback(m); e != nil {
			panic(e)
		}
	}
	warm()
	if n := testing.AllocsPerRun(1000, warm); n != 0 {
		t.Fatalf("warm default-page scope allocations%v", n)
	}
	r.Close()
	if b.Used() != 0 {
		t.Fatal("plateau close")
	}
}

func TestBytesRegionBudgetPeaksAndResetNoReserve(t *testing.T) {
	setup := func(r *BytesRegion) BytesRegionRef {
		old := byteRegionAlloc(t, r, 1)
		r.Write(old, 0, 77)
		m, e := r.PushScope()
		if e != nil {
			t.Fatal(e)
		}
		byteRegionAlloc(t, r, 0)
		if e = r.Rollback(m); e != nil {
			t.Fatal(e)
		}
		return old
	}
	probe := bytesRegionTest(t, 64, 8, false, nil)
	setup(probe)
	byteRegionAlloc(t, probe, 64)
	limit := byteRegionOwned(probe)
	b := NewBudget(limit)
	r := bytesRegionTest(t, 64, 8, false, b)
	old := setup(r)
	before := r.Stats()
	if _, e := r.Allocate(64); !errors.Is(e, ErrBudget) {
		t.Fatal("missing old+new page directory and eager default peak", e)
	}
	after := r.Stats()
	if after.Live != before.Live || after.Cursor != before.Cursor || after.ActivePages != 1 || after.BigPages != 0 {
		t.Fatal("budget failure published page plan")
	}
	if v, e := r.Read(old, 0); e != nil || v != 77 {
		t.Fatal("budget failure lost live payload")
	}
	if b.Used() != byteRegionOwned(r) {
		t.Fatal("peak failure retained uncharged preparation")
	}
	b.limit += 2 * uint64(reflect.TypeOf(bytesRegionPage{}).Size())
	byteRegionAlloc(t, r, 64)
	b.FailAfter(0)
	if e := r.Reset(); e != nil {
		t.Fatal("source reset must reuse existing default", e)
	}
	if s := r.Stats(); s.ActivePages != 1 || s.FreePages != 1 || s.BigPages != 0 || s.Live != 0 || s.Scopes != 0 || s.Cursor != 0 {
		t.Fatal("source reset lifecycle", s)
	}
	if b.remaining != 0 {
		t.Fatal("reset made reservation")
	}
	if e := r.Reset(); e != nil {
		t.Fatal("repeated reset reserved")
	}
	r.Close()
	if b.Used() != 0 {
		t.Fatal("peak/reset close leaked")
	}
	debugBudget := NewBudget(1 << 20)
	debug := bytesRegionTest(t, 64, 8, true, debugBudget)
	byteRegionAlloc(t, debug, 40)
	debug.PushScope()
	byteRegionAlloc(t, debug, 0)
	debugBudget.FailAfter(0)
	if e := debug.Reset(); e != nil || debugBudget.remaining != 0 {
		t.Fatal("debug reset reserved", e)
	}
	debug.Close()
	if debugBudget.Used() != 0 {
		t.Fatal("debug reset balance")
	}
}
func TestBytesRegionBulkScopeLifetime(t *testing.T) {
	for _, direct := range []bool{false, true} {
		r := bytesRegionTest(t, 32, 8, direct, nil)
		outer, e := r.PushScope()
		if e != nil {
			t.Fatal(e)
		}
		first := byteRegionAlloc(t, r, 1)
		r.Write(first, 0, 77)
		r.PushScope()
		second := byteRegionAlloc(t, r, 1)
		r.PushScope()
		third := byteRegionAlloc(t, r, 64)
		if e = r.PopScopes(2); e != nil {
			t.Fatal(e)
		}
		if !r.IsLive(first) || r.IsLive(second) || r.IsLive(third) || r.Stats().Scopes != 1 {
			t.Fatal("bulk source pop lifetime")
		}
		if v, e := r.Read(first, 0); e != nil || v != 77 {
			t.Fatal("outer bytes after bulk pop")
		}
		if e = r.Rollback(outer); e != nil || r.IsLive(first) {
			t.Fatal("outer rollback after bulk pop", e)
		}
	}
}

func TestBytesRegionOwnerIDCollisionAndLifetime(t *testing.T) {
	a := bytesRegionTest(t, 64, 8, false, nil)
	b := bytesRegionTest(t, 64, 8, false, nil)
	ar := byteRegionAlloc(t, a, 1)
	br := byteRegionAlloc(t, b, 1)
	if ar != br {
		t.Fatal("fixture should deliberately collide scalar allocation handles")
	}
	aid, bid := a.OwnerID(), b.OwnerID()
	if aid == 0 || bid == 0 || aid == bid {
		t.Fatal("foreign regions share scalar origin")
	}
	mark, e := a.PushScope()
	if e != nil {
		t.Fatal(e)
	}
	byteRegionAlloc(t, a, 64)
	if e = a.Rollback(mark); e != nil {
		t.Fatal(e)
	}
	if a.OwnerID() != aid {
		t.Fatal("rollback changed origin")
	}
	a.Reset()
	if a.OwnerID() != aid {
		t.Fatal("reset changed origin")
	}
	a.Close()
	b.Close()
	if a.OwnerID() != aid || b.OwnerID() != bid {
		t.Fatal("close changed origin")
	}
}
func TestBytesRegionOwnerIDExhaustion(t *testing.T) {
	// No parallel tests touch this process-local namespace while it is replaced.
	// Restore only this cold test controller after every temporary owner closes.
	previous := bytesRegionNextOwnerID.Load()
	defer bytesRegionNextOwnerID.Store(previous)
	bytesRegionNextOwnerID.Store(^uint64(0) - 1)
	b := NewBudget(1 << 20)
	r, e := NewBytesRegion(BytesRegionConfig{PageSize: 64, Descriptors: Config{PageSize: 2, Budget: b}})
	if e != nil {
		t.Fatal(e)
	}
	if r.OwnerID() != ^uint64(0) {
		r.Close()
		t.Fatal("last owner ID")
	}
	r.Close()
	if r, e = NewBytesRegion(BytesRegionConfig{PageSize: 64, Descriptors: Config{PageSize: 2, Budget: b}}); !errors.Is(e, ErrSlotLimit) || r != nil || b.Used() != 0 {
		t.Fatal("owner namespace wrapped/published", r, e, b.Used())
	}
	if bytesRegionNextOwnerID.Load() != ^uint64(0) {
		t.Fatal("exhaustion changed namespace")
	}
}
func TestBytesRegionSourceDefaultAndAlignmentConstants(t *testing.T) {
	for _, alignment := range []uint32{4, 8} {
		r := bytesRegionTest(t, 0, alignment, false, nil)
		s := r.Stats()
		if s.PayloadBytes != uint64(8192-alignment) || s.PageHeaderBytes != uint64(alignment) || s.ActivePages != 1 || s.DefaultPages != 1 {
			t.Fatal("source default page/header constants", alignment, s)
		}
	}
}
