package util

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func statFixture(t *testing.T, b *arena.Budget) (*Statistics, *arena.BytesRegion) {
	t.Helper()
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{})
	if e != nil {
		t.Fatal(e)
	}
	s, e := NewStatistics(StatisticsConfig{Keys: r, Budget: b})
	if e != nil {
		r.Close()
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close(); r.Close() })
	return s, r
}
func statKey(t *testing.T, r *arena.BytesRegion, v string) StatisticsKey {
	t.Helper()
	ref, e := r.Allocate(uint32(len(v) + 1))
	if e != nil {
		t.Fatal(e)
	}
	for i := range v {
		if e = r.Write(ref, uint32(i), v[i]); e != nil {
			t.Fatal(e)
		}
	}
	if e = r.Write(ref, uint32(len(v)), 0); e != nil {
		t.Fatal(e)
	}
	k, e := NewStatisticsKey(r, ref, 0)
	if e != nil {
		t.Fatal(e)
	}
	return k
}
func statUint(t *testing.T, s *Statistics, k StatisticsKey, v uint32) {
	t.Helper()
	if e := s.UpdateUint(k, v); e != nil {
		t.Fatal(e)
	}
}
func statDouble(t *testing.T, s *Statistics, k StatisticsKey, v float64) {
	t.Helper()
	if e := s.UpdateDouble(k, v); e != nil {
		t.Fatal(e)
	}
}

func TestStatisticsGoMemorySnapshotAndRuntime(t *testing.T) {
	const mebibyte = uint64(1024 * 1024)
	s, _ := statFixture(t, nil)
	if e := s.lock(); e != nil {
		t.Fatal(e)
	}
	e := s.appendMemoryStatistics(goMemoryStatistics{
		heapAlloc: mebibyte + mebibyte/2,
		heapSys:   2*mebibyte + mebibyte/4,
		mallocs:   uint64(^uint32(0)) + 3,
	})
	s.mu.Unlock()
	if e != nil {
		t.Fatal(e)
	}
	if s.Size() != 3 || s.IsUint(0) {
		t.Fatal("large allocation count must use the double domain")
	}
	wantKeys := []string{"max memory", "memory", "num allocs"}
	wantValues := []float64{2.25, 1.5, float64(uint64(^uint32(0)) + 3)}
	for i := range wantKeys {
		key, keyErr := s.Key(uint32(i))
		if keyErr != nil {
			t.Fatal(keyErr)
		}
		text, textErr := s.KeyText(key)
		value, valueErr := s.DoubleValue(uint32(i))
		if textErr != nil || valueErr != nil || text != wantKeys[i] || value != wantValues[i] {
			t.Fatalf("entry %d = %q %v, want %q %v (errors %v, %v)", i, text, value, wantKeys[i], wantValues[i], textErr, valueErr)
		}
	}

	if e = s.Reset(); e != nil {
		t.Fatal(e)
	}
	if e = s.lock(); e != nil {
		t.Fatal(e)
	}
	e = s.appendMemoryStatistics(goMemoryStatistics{heapAlloc: mebibyte, heapSys: 2 * mebibyte, mallocs: 7})
	s.mu.Unlock()
	if e != nil {
		t.Fatal(e)
	}
	if s.Size() != 3 || !s.IsUint(0) {
		t.Fatal("small allocation count must lead the uint domain")
	}
	key, e := s.Key(0)
	if e != nil {
		t.Fatal(e)
	}
	text, e := s.KeyText(key)
	if value, valueErr := s.UintValue(0); e != nil || valueErr != nil || text != "num allocs" || value != 7 {
		t.Fatal(text, value, e, valueErr)
	}

	if e = s.Reset(); e != nil {
		t.Fatal(e)
	}
	if e = GetMemoryStatistics(s); e != nil {
		t.Fatal(e)
	}
	if s.Size() != 3 {
		t.Fatalf("runtime memory statistic count = %d", s.Size())
	}
	seen := make(map[string]bool, 3)
	for i := uint32(0); i < s.Size(); i++ {
		key, keyErr := s.Key(i)
		if keyErr != nil {
			t.Fatal(keyErr)
		}
		text, textErr := s.KeyText(key)
		if textErr != nil {
			t.Fatal(textErr)
		}
		seen[text] = true
	}
	for _, name := range wantKeys {
		if !seen[name] {
			t.Fatalf("missing runtime memory statistic %q", name)
		}
	}
}

func TestStatisticsSourceFiveAssertionsAndRawDomains(t *testing.T) {
	s, r := statFixture(t, nil)
	k := statKey(t, r, "x")
	if e := s.UpdateUint(StatisticsKey{}, 0); e != nil {
		t.Fatal(e)
	}
	if e := s.UpdateDouble(StatisticsKey{}, math.Copysign(0, -1)); e != nil {
		t.Fatal(e)
	}
	statUint(t, s, k, 7)
	statDouble(t, s, k, 2.5)
	if s.Size() != 2 || !s.IsUint(0) || s.IsUint(1) || s.IsUint(99) {
		t.Fatal("raw domains")
	}
	if v, e := s.UintValue(0); v != 7 || e != nil {
		t.Fatal(v, e)
	}
	if v, e := s.DoubleValue(1); v != 2.5 || e != nil {
		t.Fatal(v, e)
	}
	for _, i := range []uint32{1, 2} {
		if _, e := s.UintValue(i); !errors.Is(e, arena.ErrBounds) {
			t.Fatal("source uint precondition", e)
		}
	}
	for _, i := range []uint32{0, 2} {
		if _, e := s.DoubleValue(i); !errors.Is(e, arena.ErrBounds) {
			t.Fatal("source double precondition", e)
		}
	}
	statUint(t, s, StatisticsKey{}, 1)
	if got, e := s.Key(1); got != (StatisticsKey{}) || e != nil {
		t.Fatal(e)
	}
	if _, e := s.DisplaySMT2(); !errors.Is(e, ErrStatisticsKey) {
		t.Fatal("source key!=0 assertion", e)
	}
	if e := arena.ValidateRecordType(reflect.TypeOf(StatisticsKey{})); e != nil {
		t.Fatal(e)
	}
}
func TestStatisticsRenderingSourceQuirksAndIEEE(t *testing.T) {
	s, r := statFixture(t, nil)
	a := statKey(t, r, ":a")
	ab := statKey(t, r, "a b")
	z := statKey(t, r, "zzz")
	statUint(t, s, a, math.MaxUint32)
	statUint(t, s, a, 2)
	statUint(t, s, ab, 4)
	statDouble(t, s, a, 9)
	statDouble(t, s, z, 1.125)
	got, e := s.Display()
	if e != nil || got != "a:   1\na:   1\na b: 4\nzzz: 1.12\n" {
		t.Fatal(got, e)
	}
	got, e = s.DisplaySMT2()
	if e != nil || got != "(:a  1\n :a  1\n :a-b 4\n :zzz 1.12)\n" {
		t.Fatal(got, e)
	}
	s.Reset()
	for _, v := range []float64{math.Inf(1), math.Inf(-1), math.NaN(), 1e16, 1, -1e16} {
		statDouble(t, s, z, v)
	}
	got, e = s.Display()
	if e != nil || (got != "zzz: -nan\n" && got != "zzz: nan\n") {
		t.Fatal(got, e)
	}
	s.Reset()
	statDouble(t, s, z, 1e16)
	statDouble(t, s, z, 1)
	statDouble(t, s, z, -1e16)
	got, e = s.Display()
	if e != nil || got != "zzz: 0.00\n" {
		t.Fatal(got, e)
	}
	s.Reset()
	got, e = s.DisplaySMT2()
	if got != "()\n" || e != nil {
		t.Fatal(got, e)
	}
}
func TestStatisticsBorrowedAliasesLifetimeAndNull(t *testing.T) {
	s, r := statFixture(t, nil)
	k := statKey(t, r, "abc\x00ignored")
	statUint(t, s, k, 1)
	r.Write(k.ref, k.offset, 'z')
	got, e := s.Display()
	if got != "zbc: 1\n" || e != nil {
		t.Fatal(got, e)
	}
	r2, e := arena.NewBytesRegion(arena.BytesRegionConfig{})
	if e != nil {
		t.Fatal(e)
	}
	defer r2.Close()
	f := statKey(t, r2, "foreign")
	statUint(t, s, f, 3)
	if q, e := s.Key(1); q != f || e != nil {
		t.Fatal(q, e)
	}
	if _, e = s.Display(); !errors.Is(e, ErrStatisticsKey) {
		t.Fatal(e)
	}
	s.Reset()
	statUint(t, s, k, 1)
	r.Reset()
	if _, e = s.Display(); e == nil {
		t.Fatal("stale key")
	}
	s.Reset()
	statDouble(t, s, StatisticsKey{}, 1)
	if v, e := s.DoubleValue(0); v != 1 || e != nil {
		t.Fatal(v, e)
	}
}
func TestStatisticsCopyAssignMoveCapacityAndPrefixes(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	s, r := statFixture(t, b)
	k := statKey(t, r, "x")
	for i := uint32(1); i < 9; i++ {
		statUint(t, s, k, i)
	}
	statDouble(t, s, k, 2)
	st := s.Stats()
	if st.UintCapacity != 8 || st.DoubleCapacity != 2 {
		t.Fatal(st)
	}
	c, e := s.Clone(nil)
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	if c.Stats().UintCapacity != 8 || c.Size() != 9 {
		t.Fatal(c.Stats())
	}
	s.Reset()
	if c.Size() != 9 {
		t.Fatal("shallowcopy")
	}
	if e = s.Assign(c); e != nil {
		t.Fatal(e)
	}
	d, e := s.MoveClone(nil)
	if e != nil {
		t.Fatal(e)
	}
	defer d.Close()
	if s.Size() != 0 || s.Stats().UintCapacity != 0 || d.Size() != 9 {
		t.Fatal("move ctor")
	}
	if e = s.MoveAssign(d); e != nil {
		t.Fatal(e)
	}
	if s.Size() != 9 || d.Size() != 9 {
		t.Fatal("svector rvalue assignment must copy")
	}
	s.Reset()
	if e = s.Copy(d); e != nil {
		t.Fatal(e)
	}
	if e = s.Copy(d); e != nil {
		t.Fatal(e)
	}
	if s.Size() != 18 {
		t.Fatal(s.Size())
	}
	if e = s.Copy(s); !errors.Is(e, ErrStatisticsSelfCopy) {
		t.Fatal(e)
	}
	s.Reset()
	b.FailAfter(0)
	if e = s.Assign(d); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if s.Size() != 0 || s.Stats().UintCapacity != 0 {
		t.Fatal("assignment destroys old before allocation", s.Stats())
	}
	b.FailAfter(-1)
}
func TestStatisticsFaultPrefixesAndImmediateAccounting(t *testing.T) {
	for fail := int64(0); fail < 22; fail++ {
		b := arena.NewBudget(1 << 24)
		s, r := statFixture(t, b)
		k := statKey(t, r, "x")
		for i := 0; i < 40; i++ {
			unique := statKey(t, r, string([]byte{byte(65 + i), byte(90 - i)}))
			statUint(t, s, unique, 1)
		}
		statDouble(t, s, k, 1)
		stream, e := NewNumericTextStream(NumericTextStreamConfig{})
		if e != nil {
			t.Fatal(e)
		}
		base := b.Used()
		b.FailAfter(fail)
		e = s.DisplayInto(stream)
		if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(e)
		}
		if b.Used() != base {
			t.Fatalf("scratch leak at %d: %d %d", fail, b.Used(), base)
		}
		b.FailAfter(-1)
		stream.Close()
		s.Close()
		if b.Used() != 0 {
			t.Fatal("owner leak", b.Used())
		}
		r.Close()
	}
	b := arena.NewBudget(1 << 20)
	s, r := statFixture(t, b)
	k := statKey(t, r, "x")
	base := b.Used()
	b.FailAfter(0)
	if e := s.UpdateUint(k, 1); !errors.Is(e, arena.ErrInjectedFailure) || s.Size() != 0 || b.Used() != base {
		t.Fatal(e)
	}
	b.FailAfter(-1)
	src, e := NewStatistics(StatisticsConfig{Keys: r})
	if e != nil {
		t.Fatal(e)
	}
	defer src.Close()
	for i := 0; i < 4; i++ {
		statUint(t, src, k, 1)
	}
	statDouble(t, src, k, 1)
	b.FailAfter(1)
	if e = s.Copy(src); !errors.Is(e, arena.ErrInjectedFailure) || s.Size() != 2 {
		t.Fatal("append prefix", s.Size(), e)
	}
	if b.Used() != base+2*uint64(reflect.TypeOf(statisticsUintContribution{}).Size()) {
		t.Fatal("immediate append roots", b.Used())
	}
	b.FailAfter(-1)
}
func TestStatisticsInternalNonterminationAndEmptyPrefix(t *testing.T) {
	s, r := statFixture(t, nil)
	empty := statKey(t, r, "")
	statUint(t, s, empty, 2)
	statDouble(t, s, empty, 3)
	out, e := s.DisplayInternal()
	if out != " 2\n 3\n" || e != nil {
		t.Fatal(out, e)
	}
	s.Reset()
	a := statKey(t, r, "a")
	statUint(t, s, a, 1)
	stream, e := NewNumericTextStream(NumericTextStreamConfig{})
	if e != nil {
		t.Fatal(e)
	}
	defer stream.Close()
	if e = s.DisplayInternalInto(stream); !errors.Is(e, ErrStatisticsNonterminating) {
		t.Fatal(e)
	}
	out, _ = stream.String()
	if out != "" {
		t.Fatal(out)
	}
	s.Reset()
	colon := statKey(t, r, ":")
	statUint(t, s, colon, 5)
	out, e = s.DisplayInternal()
	if out != " 5\n" || e != nil {
		t.Fatal(out, e)
	}
}
func TestStatisticsStreamPersistentFlagsAndBudgetPrefix(t *testing.T) {
	s, r := statFixture(t, nil)
	a := statKey(t, r, "a")
	statDouble(t, s, a, 2.5)
	stream, e := NewNumericTextStream(NumericTextStreamConfig{})
	if e != nil {
		t.Fatal(e)
	}
	defer stream.Close()
	stream.SetPrecision(9)
	stream.SetWidth(4)
	stream.SetFill('_')
	stream.SetShowPos(true)
	if e = s.DisplayInto(stream); e != nil {
		t.Fatal(e)
	}
	out, _ := stream.String()
	if out != "___a: +2.50\n" {
		t.Fatal(out)
	}
	state := stream.State()
	if state.FloatMode != NumericFixedFloat || state.Precision != 2 || state.Width != 0 || !state.ShowPos {
		t.Fatal(state)
	}
	budget := arena.NewBudget(1 << 20)
	limited, e := NewNumericTextStream(NumericTextStreamConfig{Budget: budget})
	if e != nil {
		t.Fatal(e)
	}
	defer limited.Close()
	if e = s.DisplayInto(limited); e != nil {
		t.Fatal(e)
	}
	limited.ResetText()
	limited.SetWidth(math.MaxInt64)
	budget.FailAfter(0)
	e = s.DisplayInto(limited)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	out, _ = limited.String()
	if len(out) == 0 || !strings.HasSuffix(out, " ") || limited.State().Width != 0 {
		t.Fatal("output prefix", len(out), limited.State())
	}
}
func TestStatisticsRealRlimitThresholdAndLiteralOwnership(t *testing.T) {
	s, r := statFixture(t, nil)
	l, e := NewReslimit(nil)
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	if e = GetRlimitStatistics(l, s); e != nil || s.Size() != 0 {
		t.Fatal(e)
	}
	l.IncOffset(math.MaxUint32)
	if e = GetRlimitStatistics(l, s); e != nil {
		t.Fatal(e)
	}
	if v, e := s.UintValue(0); v != math.MaxUint32 || e != nil {
		t.Fatal(v, e)
	}
	l.Inc()
	if e = GetRlimitStatistics(l, s); e != nil {
		t.Fatal(e)
	}
	if v, e := s.DoubleValue(1); v != float64(uint64(math.MaxUint32)+1) || e != nil {
		t.Fatal(v, e)
	}
	key, _ := s.Key(0)
	text, e := s.KeyText(key)
	if text != "rlimit count" || e != nil {
		t.Fatal(text, e)
	}
	s.Close()
	if !r.IsLive(key.ref) {
		t.Fatal("borrowed literal freed")
	}
}
func TestStatisticsHashMapGrowthAndIndependentRawModel(t *testing.T) {
	s, r := statFixture(t, nil)
	keys := make([]StatisticsKey, 40)
	for i := range keys {
		keys[i] = statKey(t, r, string([]byte{byte(128 + i), 'a'}))
	}
	var model []uint32
	seed := uint32(7)
	for i := 0; i < 500; i++ {
		seed = seed*1664525 + 1013904223
		if seed&31 == 0 {
			s.Reset()
			model = nil
		}
		v := seed >> 1
		statUint(t, s, keys[seed%40], v)
		model = append(model, v)
		if s.Size() != uint32(len(model)) {
			t.Fatal(i)
		}
		for j, w := range model {
			got, e := s.UintValue(uint32(j))
			if got != w || e != nil {
				t.Fatal(j, got, e)
			}
		}
	}
	m, e := newStatisticsMap(s, false)
	if e != nil {
		t.Fatal(e)
	}
	defer m.close()
	for i, k := range keys {
		if e = m.insert(k, uint32(i), 0); e != nil {
			t.Fatal(e)
		}
		h, e := s.keyHash(k)
		if e != nil || h != String(string([]byte{byte(128 + i), 'a'}), 17) {
			t.Fatal(h, e)
		}
	}
	if len(m.entries) != 64 {
		t.Fatal(len(m.entries))
	}
	for i, k := range keys {
		idx, ok, e := m.find(k)
		if e != nil || !ok || m.entries[idx].u != uint32(i) {
			t.Fatal(i, e)
		}
	}
}
func TestStatisticsWarmZeroAllocationAndClosedCopies(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	s, r := statFixture(t, b)
	k := statKey(t, r, "x")
	for i := 0; i < 10; i++ {
		statUint(t, s, k, 1)
		statDouble(t, s, k, 1)
	}
	s.Reset()
	a := testing.AllocsPerRun(100, func() {
		s.UpdateUint(k, 1)
		s.UpdateDouble(k, 2)
		s.UintValue(0)
		s.DoubleValue(1)
		s.keyHash(k)
		s.Reset()
	})
	if a != 0 {
		t.Fatal(a)
	}
	shallow := Statistics{self: s, keys: s.keys, budget: s.budget}
	if e := shallow.UpdateUint(k, 1); !errors.Is(e, ErrStatisticsOwner) {
		t.Fatal(e)
	}
	s.Close()
	if e := s.UpdateUint(k, 1); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	if _, e := NewStatistics(StatisticsConfig{}); !errors.Is(e, ErrStatisticsKey) {
		t.Fatal(e)
	}
}
func TestStatisticsCrossRegionCopyBindingAndFailures(t *testing.T) {
	b := arena.NewBudget(1 << 22)
	dst, _ := statFixture(t, b)
	src, r := statFixture(t, nil)
	k := statKey(t, r, "abc")
	statUint(t, src, k, 4)
	statDouble(t, src, k, 2)
	if e := dst.Assign(src); e != nil {
		t.Fatal(e)
	}
	if text, e := dst.Display(); text != "abc: 4\nabc: 4\n" || e != nil {
		t.Fatal(text, e)
	}
	r.Write(k.ref, 0, 'z')
	if text, e := dst.Display(); text != "zbc: 4\nzbc: 4\n" || e != nil {
		t.Fatal(text, e)
	}
	clone, e := dst.Clone(nil)
	if e != nil {
		t.Fatal(e)
	}
	defer clone.Close()
	if text, e := clone.Display(); text != "zbc: 4\nzbc: 4\n" || e != nil {
		t.Fatal(text, e)
	}
	dst.Reset()
	base := b.Used()
	b.FailAfter(0)
	if e = dst.BindKeyRegion(r); e != nil || b.Used() != base {
		t.Fatal("existing binding no reserve", e)
	}
	b.FailAfter(-1)
	src2, r2 := statFixture(t, nil)
	k2 := statKey(t, r2, "foreign")
	statUint(t, src2, k2, 2)
	b.FailAfter(0)
	if e = dst.Copy(src2); e != nil {
		t.Fatal("spare binding & vector capacity should need no reserve", e)
	}
	b.FailAfter(-1)
	dst.Reset()
	plain, _ := statFixture(t, nil)
	statUint(t, plain, k, 3)
	if _, e = plain.Display(); !errors.Is(e, ErrStatisticsKey) {
		t.Fatal(e)
	}
	if e = plain.BindKeyRegion(r); e != nil {
		t.Fatal(e)
	}
	if text, e := plain.Display(); text != "zbc: 3\n" || e != nil {
		t.Fatal(text, e)
	}
	r3, e := arena.NewBytesRegion(arena.BytesRegionConfig{})
	if e != nil {
		t.Fatal(e)
	}
	defer r3.Close()
	base = b.Used()
	b.FailAfter(0)
	if e = dst.BindKeyRegion(r3); !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != base {
		t.Fatal("directory old+new failure", e, b.Used(), base)
	}
	b.FailAfter(-1)
	if e = dst.BindKeyRegion(r3); e != nil {
		t.Fatal(e)
	}
	if dst.Stats().BindingCapacity != 3 {
		t.Fatal(dst.Stats())
	}
}
func TestStatisticsConstructorAssignmentMemberFailureAndPeak(t *testing.T) {
	zero := arena.NewBudget(0)
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{})
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	if s, e := NewStatistics(StatisticsConfig{Keys: r, Budget: zero}); s != nil || !errors.Is(e, arena.ErrBudget) || zero.Used() != 0 {
		t.Fatal(s, e)
	}
	src, e := NewStatistics(StatisticsConfig{Keys: r})
	if e != nil {
		t.Fatal(e)
	}
	defer src.Close()
	k := statKey(t, r, "x")
	statUint(t, src, k, 4)
	statDouble(t, src, k, 2)
	for f := int64(0); f < 3; f++ {
		b := arena.NewBudget(1 << 20)
		b.FailAfter(f)
		copy, e := src.Clone(b)
		if e != nil {
			if b.Used() != 0 {
				t.Fatal("ctor unwind", f, b.Used())
			}
		} else {
			copy.Close()
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
		}
	}
	b := arena.NewBudget(1 << 20)
	dst, e := NewStatistics(StatisticsConfig{Keys: r, Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	defer dst.Close()
	statUint(t, dst, k, 8)
	statDouble(t, dst, k, 9)
	base := b.Used()
	b.FailAfter(1)
	e = dst.Assign(src)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if v, e := dst.UintValue(0); v != 4 || e != nil {
		t.Fatal("uint assignment published", v, e)
	}
	if dst.Stats().DoubleSize != 0 || dst.Stats().DoubleCapacity != 0 {
		t.Fatal("double destroyed before failed copy", dst.Stats())
	}
	if b.Used() != base-2*uint64(reflect.TypeOf(statisticsDoubleContribution{}).Size()) {
		t.Fatal("immediate roots", b.Used(), base)
	}
	b.FailAfter(-1)
	dst.Reset()
	capBefore := dst.Stats().UintCapacity
	for i := uint32(0); i < capBefore; i++ {
		statUint(t, dst, k, 1)
	}
	peak := arena.NewBudget(uint64(reflect.TypeOf(Statistics{}).Size()) + uint64(3)*uint64(reflect.TypeOf(statisticsUintContribution{}).Size()))
	p, e := NewStatistics(StatisticsConfig{Keys: r, Budget: peak})
	if e != nil {
		t.Fatal(e)
	}
	defer p.Close()
	statUint(t, p, k, 1)
	statUint(t, p, k, 1)
	if e = p.UpdateUint(k, 1); !errors.Is(e, arena.ErrBudget) || p.Size() != 2 {
		t.Fatal("growth overlaps complete old and new", e)
	}
}
func TestStatisticsMapOverwriteGrowthIdentityAndInternalPrefix(t *testing.T) {
	s, r := statFixture(t, nil)
	m, e := newStatisticsMap(s, false)
	if e != nil {
		t.Fatal(e)
	}
	defer m.close()
	keys := make([]StatisticsKey, 7)
	for i := range keys {
		keys[i] = statKey(t, r, string(byte('a'+i)))
		if e = m.insert(keys[i], uint32(i+1), 0); e != nil {
			t.Fatal(e)
		}
	}
	if len(m.entries) != 8 {
		t.Fatal(len(m.entries))
	}
	equal := statKey(t, r, "a")
	if e = m.insert(equal, 99, 0); e != nil {
		t.Fatal(e)
	}
	if len(m.entries) != 16 {
		t.Fatal("grow before overwrite", len(m.entries))
	}
	idx, ok, e := m.find(keys[0])
	if e != nil || !ok || m.entries[idx].key != equal || m.entries[idx].u != 99 {
		t.Fatal("replace incoming key identity", idx, ok, e)
	}
	empty := ""
	if String(":", 17)&7 < String("", 17)&7 {
		empty = ":"
	}
	ek := statKey(t, r, empty)
	bucket := String(empty, 17) & 7
	var later string
	for c := byte(1); c < 127; c++ {
		v := string(c)
		if c != ':' && String(v, 17)&7 > bucket {
			later = v
			break
		}
	}
	if later == "" {
		t.Fatal("expected finite prefix fixture")
	}
	statUint(t, s, ek, 2)
	statUint(t, s, statKey(t, r, later), 3)
	stream, e := NewNumericTextStream(NumericTextStreamConfig{})
	if e != nil {
		t.Fatal(e)
	}
	defer stream.Close()
	if e = s.DisplayInternalInto(stream); !errors.Is(e, ErrStatisticsNonterminating) {
		t.Fatal(e)
	}
	got, _ := stream.String()
	if got != " 2\n" {
		t.Fatal("prior empty-key prefix", got)
	}
	s.Reset()
	statDouble(t, s, ek, 1.25)
	stream.ResetText()
	stream.SetPrecision(1)
	stream.SetFloatMode(NumericScientificFloat)
	if e = s.DisplayInternalInto(stream); e != nil {
		t.Fatal(e)
	}
	got, _ = stream.String()
	if got != " 1.2e+00\n" {
		t.Fatal(got)
	}
	if stream.State().FloatMode != NumericScientificFloat || stream.State().Precision != 1 {
		t.Fatal("internal must retain float flags")
	}
}
func TestStatisticsMergedExactIndependentOracleAndShadowState(t *testing.T) {
	s, r := statFixture(t, nil)
	keys := make([]StatisticsKey, 30)
	names := make([]string, 30)
	var expected [30]uint32
	for i := range keys {
		names[i] = string([]byte{'k', byte('A' + i)})
		keys[i] = statKey(t, r, names[i])
	}
	seed := uint32(93)
	for i := 0; i < 1000; i++ {
		seed = seed*1103515245 + 12345
		j := seed % 30
		v := seed ^ 0xf0000000
		statUint(t, s, keys[j], v)
		expected[j] += v
	}
	u, d, e := s.maps()
	if e != nil {
		t.Fatal(e)
	}
	defer u.close()
	defer d.close()
	for i, k := range keys {
		idx, ok, e := u.find(k)
		if e != nil || !ok || u.entries[idx].u != expected[i] {
			t.Fatal(i, expected[i], e)
		}
	}
	s.Reset()
	k := keys[0]
	statUint(t, s, k, 2)
	statDouble(t, s, k, math.NaN())
	out, e := NewNumericTextStream(NumericTextStreamConfig{})
	if e != nil {
		t.Fatal(e)
	}
	defer out.Close()
	out.SetPrecision(11)
	out.SetFloatMode(NumericScientificFloat)
	if e = s.DisplaySMT2Into(out); e != nil {
		t.Fatal(e)
	}
	text, _ := out.String()
	if text != "(:kA 2\n :kA 2)\n" {
		t.Fatal(text)
	}
	if out.State().Precision != 11 || out.State().FloatMode != NumericScientificFloat {
		t.Fatal("shadowed double must not mutate stream flags")
	}
}
func TestStatisticsMoveTransferFailuresAndCallerLiteralPrefixes(t *testing.T) {
	src, r := statFixture(t, nil)
	k := statKey(t, r, "x")
	statUint(t, src, k, 2)
	statDouble(t, src, k, 3)
	for f := int64(0); f < 2; f++ {
		b := arena.NewBudget(1 << 20)
		b.FailAfter(f)
		v, e := src.MoveClone(b)
		if !errors.Is(e, arena.ErrInjectedFailure) || v != nil || b.Used() != 0 || src.Size() != 2 {
			t.Fatal("move prepare failure", f, v, e, b.Used(), src.Size())
		}
	}
	b := arena.NewBudget(1 << 20)
	v, e := src.MoveClone(b)
	if e != nil {
		t.Fatal(e)
	}
	if src.Size() != 0 || v.Size() != 2 {
		t.Fatal("move ownership")
	}
	v.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	b2 := arena.NewBudget(1 << 20)
	s, r2 := statFixture(t, b2)
	l, e := NewReslimit(nil)
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	l.Inc()
	live := r2.Stats().Live
	before := b2.Used()
	b2.FailAfter(0)
	e = GetRlimitStatistics(l, s)
	if !errors.Is(e, arena.ErrInjectedFailure) || s.Size() != 0 || b2.Used() != before {
		t.Fatal(e, s.Size(), b2.Used())
	}
	if r2.Stats().Live != live+1 {
		t.Fatal("explicit caller key allocation prefix must remain owned", r2.Stats())
	}
	b2.FailAfter(-1)
	if e = GetRlimitStatistics(l, s); e != nil {
		t.Fatal(e)
	}
	if r2.Stats().Live != live+2 {
		t.Fatal("no hidden literal cache")
	}
}
func TestStatisticsSignedSpecialValuesAndBorrowedOutputBounds(t *testing.T) {
	s, r := statFixture(t, nil)
	k := statKey(t, r, "x")
	cases := []struct {
		value float64
		text  string
	}{{math.Inf(1), "x: inf\n"}, {math.Inf(-1), "x: -inf\n"}, {math.Float64frombits(0x7ff8000000000001), "x: nan\n"}, {math.Float64frombits(0xfff8000000000001), "x: -nan\n"}}
	for _, v := range cases {
		s.Reset()
		statDouble(t, s, k, v.value)
		text, e := s.Display()
		if text != v.text || e != nil {
			t.Fatal(text, v.text, e)
		}
		raw, e := s.DoubleValue(0)
		if e != nil || math.Float64bits(raw) != math.Float64bits(v.value) {
			t.Fatal("raw bits", raw, e)
		}
	}
	s.Reset()
	long := statKey(t, r, strings.Repeat("a", 200))
	statUint(t, s, long, 1)
	b := arena.NewBudget(1 << 20)
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	defer out.Close()
	out.SetWidth(7)
	b.FailAfter(0)
	if e = s.DisplayInto(out); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	text, _ := out.String()
	if text != "" || out.State().Width != 7 {
		t.Fatal("charged key staging fails before stream action", len(text), out.State())
	}
	b.FailAfter(-1)
	if e = s.DisplayInto(out); e != nil {
		t.Fatal(e)
	}
	text, _ = out.String()
	if text != strings.Repeat("a", 200)+": 1\n" {
		t.Fatal(len(text))
	}
	out.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	if e = s.DisplayInto(nil); !errors.Is(e, ErrNumericStreamOwner) {
		t.Fatal(e)
	}
	closed, e := NewNumericTextStream(NumericTextStreamConfig{})
	if e != nil {
		t.Fatal(e)
	}
	closed.Close()
	if e = s.DisplaySMT2Into(closed); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
func TestStatisticsBindingBackingHostBounds(t *testing.T) {
	stride := uint64(reflect.TypeOf(statisticsKeyBinding{}).Size())
	limit := uint64(math.MaxInt32)
	n := limit / stride
	if got, e := statisticsBackingBytes(n, stride, limit); got != n*stride || e != nil {
		t.Fatal(got, e)
	}
	if _, e := statisticsBackingBytes(n+1, stride, limit); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual 32bit byte size overflow", e)
	}
	if _, e := statisticsBackingBytes(math.MaxUint64, 16, math.MaxUint64); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("source element count bound", e)
	}
	for _, typ := range []reflect.Type{reflect.TypeOf(StatisticsKey{}), reflect.TypeOf(statisticsUintContribution{}), reflect.TypeOf(statisticsDoubleContribution{}), reflect.TypeOf(statisticsMapEntry{})} {
		if e := arena.ValidateRecordType(typ); e != nil {
			t.Fatal(typ, e)
		}
	}
}
func TestStatisticsClosedSecondaryRawCopyDoesNotReadKeys(t *testing.T) {
	src, _ := statFixture(t, nil)
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{})
	if e != nil {
		t.Fatal(e)
	}
	k := statKey(t, r, "secondary")
	if e = src.BindKeyRegion(r); e != nil {
		t.Fatal(e)
	}
	statUint(t, src, k, 7)
	r.Close()
	dst, _ := statFixture(t, nil)
	if e = dst.Copy(src); e != nil {
		t.Fatal("raw append must not inspect closed byte owner", e)
	}
	if got, e := dst.Key(0); got != k || e != nil {
		t.Fatal(got, e)
	}
	if e = dst.Assign(src); e != nil {
		t.Fatal("raw assignment", e)
	}
	clone, e := src.Clone(nil)
	if e != nil {
		t.Fatal("raw copy constructor", e)
	}
	defer clone.Close()
	for _, s := range []*Statistics{src, dst, clone} {
		if got, e := s.UintValue(0); got != 7 || e != nil {
			t.Fatal(got, e)
		}
		if _, e = s.Display(); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("later actual key read must fail closed", e)
		}
	}
	if e = dst.BindKeyRegion(r); !errors.Is(e, ErrStatisticsKey) {
		t.Fatal("public bind requires live owner", e)
	}
}
func TestStatisticsClosedPrimaryCloneMoveRetainsRawIdentity(t *testing.T) {
	s, r := statFixture(t, nil)
	k := statKey(t, r, "primary")
	for i := 0; i < 3; i++ {
		statUint(t, s, k, 4)
	}
	statDouble(t, s, k, 2)
	capacity := s.Stats()
	r.Close()
	if _, e := NewStatistics(StatisticsConfig{Keys: r}); !errors.Is(e, ErrStatisticsKey) {
		t.Fatal("public ctor live guard", e)
	}
	b := arena.NewBudget(1 << 20)
	clone, e := s.Clone(b)
	if e != nil {
		t.Fatal("internal clone must not dereference primary", e)
	}
	if clone.Stats().UintCapacity != capacity.UintCapacity || clone.Stats().DoubleCapacity != capacity.DoubleCapacity {
		t.Fatal(clone.Stats(), capacity)
	}
	for _, owner := range []*Statistics{s, clone} {
		if raw, e := owner.Key(0); raw != k || e != nil {
			t.Fatal(raw, e)
		}
		if value, e := owner.UintValue(0); value != 4 || e != nil {
			t.Fatal(value, e)
		}
		if _, e = owner.Display(); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("read guard", e)
		}
	}
	clone.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	b.FailAfter(1)
	if moved, e := s.MoveClone(b); moved != nil || !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != 0 || s.Size() != 4 {
		t.Fatal("closed-owner move failure roots", moved, e, b.Used())
	}
	b.FailAfter(-1)
	moved, e := s.MoveClone(b)
	if e != nil {
		t.Fatal(e)
	}
	if s.Size() != 0 || moved.Size() != 4 || moved.Stats().UintCapacity != capacity.UintCapacity {
		t.Fatal("move", moved.Stats())
	}
	if _, e = moved.DisplaySMT2(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	moved.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
