// Copyright (c) 2006 Microsoft Corporation
// Selected assertions translated from src/test/rational.cpp.
package mpnum

import (
	"errors"
	"math"
	"math/big"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func serviceConfig(b *arena.Budget) RationalServiceConfig {
	return RationalServiceConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}}
}
func serviceNew(t *testing.T) (*RationalService, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	s, e := NewRationalService(serviceConfig(b))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := s.Close(); e != nil {
			t.Error(e)
		}
		if b.Used() != 0 {
			t.Errorf("close retained %d", b.Used())
		}
	})
	return s, b
}
func serviceMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func serviceText(t *testing.T, s *RationalService, v *OwnedRational, want string) {
	t.Helper()
	got, e := s.String(v)
	if e != nil || got != want {
		t.Fatalf("got %q %v, want %q", got, e, want)
	}
}
func serviceValue(t *testing.T, s *RationalService, text string) OwnedRational {
	t.Helper()
	v, e := s.NewValue()
	serviceMust(t, e)
	serviceMust(t, s.SetString(&v, text))
	return v
}

func TestRationalServiceSourceConstructorAssertions(t *testing.T) {
	s, _ := serviceNew(t)
	r1, _ := s.NewValue()
	r2, _ := s.NewValue()
	r3, _ := s.NewValue()
	serviceMust(t, s.SetInt64(&r1, 1))
	serviceMust(t, s.SetFractionSigned(&r2, 1, 2))
	serviceMust(t, s.SetFractionSigned(&r3, 2, 4))
	// src/test/rational.cpp:30,31,74,119,120: exact selected active ENSUREs.
	eq, e := s.Equal(&r2, &r3)
	if e != nil || !eq {
		t.Fatal(eq, e)
	}
	eq, e = s.Equal(&r1, &r2)
	if e != nil || eq {
		t.Fatal(eq, e)
	}
	serviceMust(t, s.SetString(&r2, "100/20"))
	serviceMust(t, s.SetInt64(&r3, 5))
	eq, e = s.Equal(&r2, &r3)
	if e != nil || !eq {
		t.Fatal(eq, e)
	}
	serviceMust(t, s.SetString(&r1, "10000000000000000000000000000000000"))
	serviceMust(t, s.SetString(&r2, "10000000000000000000000000000000000/3"))
	serviceMust(t, s.SetString(&r3, "20000000000000000000000000000000000/6"))
	eq, e = s.Equal(&r2, &r3)
	if e != nil || !eq {
		t.Fatal(eq, e)
	}
	eq, e = s.Equal(&r1, &r2)
	if e != nil || eq {
		t.Fatal(eq, e)
	}
	for _, v := range []*OwnedRational{&r1, &r2, &r3} {
		serviceMust(t, s.Del(v))
	}
}

func TestRationalServiceConstantsSettersAndMoveHistory(t *testing.T) {
	s, _ := serviceNew(t)
	zero, e := s.Zero()
	serviceMust(t, e)
	one, e := s.One()
	serviceMust(t, e)
	minus, e := s.MinusOne()
	serviceMust(t, e)
	serviceText(t, s, &zero, "0")
	serviceText(t, s, &one, "1")
	serviceText(t, s, &minus, "-1")
	serviceMust(t, s.SetInt64(&one, 91))
	again, e := s.One()
	serviceMust(t, e)
	serviceText(t, s, &again, "1")
	for _, p := range [][2]int32{{5, -6}, {-5, -6}, {math.MinInt32, -1}, {1, math.MinInt32}, {math.MinInt32, math.MinInt32}} {
		serviceMust(t, s.SetFractionSigned(&zero, p[0], p[1]))
		want := big.NewRat(int64(p[0]), int64(p[1])).RatString()
		serviceText(t, s, &zero, want)
	}
	serviceMust(t, s.SetUint64(&one, math.MaxUint64))
	serviceText(t, s, &one, "18446744073709551615")
	serviceMust(t, s.SetFraction(&minus, math.MinInt64, math.MaxUint64))
	serviceText(t, s, &minus, "-9223372036854775808/18446744073709551615")
	originalNum, originalDen := minus.raw.num.val, minus.raw.den.val
	numRef, denRef := minus.raw.num.cell, minus.raw.den.cell
	serviceMust(t, s.Move(&zero, &minus))
	if zero.raw.num.cell != numRef || zero.raw.den.cell != denRef || minus.raw.num.cell != 0 || minus.raw.den.cell != 0 || minus.raw.num.val != originalNum || minus.raw.den.val != originalDen {
		t.Fatal("source move history/ownership changed")
	}
	serviceText(t, s, &minus, "-1")
	serviceMust(t, s.Swap(&zero, &one))
	serviceText(t, s, &zero, "18446744073709551615")
	for _, v := range []*OwnedRational{&zero, &one, &minus, &again} {
		serviceMust(t, s.Del(v))
	}
}

func TestRationalServicePowerCacheSourceModel(t *testing.T) {
	s, _ := serviceNew(t)
	length, capacity := uint32(0), uint32(0)
	for _, k := range []uint32{0, 1, 2, 4, 3, 9, 31, 63, 127, 255, 128} {
		v, e := s.PowerOfTwo(k)
		serviceMust(t, e)
		want := new(big.Int).Lsh(big.NewInt(1), uint(k)).String()
		serviceText(t, s, &v, want)
		if k >= length {
			length = k + 2
		}
		for capacity < length {
			if capacity == 0 {
				capacity = 2
			} else {
				capacity = (3*capacity + 1) / 2
			}
		}
		stats, e := s.Stats()
		serviceMust(t, e)
		if stats.CacheLength != length || stats.CacheCapacity != capacity || stats.CacheBytes != uint64(capacity)*uint64(reflect.TypeOf(Rational{}).Size()) {
			t.Fatal(k, stats, length, capacity)
		}
		serviceMust(t, s.SetInt64(&v, 7))
		serviceMust(t, s.Del(&v))
		again, e := s.PowerOfTwo(k)
		serviceMust(t, e)
		serviceText(t, s, &again, want)
		serviceMust(t, s.Del(&again))
	}
	for i := uint32(0); i < uint32(len(s.cache)); i++ {
		got, e := s.manager.String(&s.cache[i])
		if e != nil || got != new(big.Int).Lsh(big.NewInt(1), uint(i)).String() {
			t.Fatal(i, got, e)
		}
	}
}

func TestRationalServiceProvenanceCopiesAndLifetimes(t *testing.T) {
	a, _ := serviceNew(t)
	b, _ := serviceNew(t)
	if a.OwnerID() == 0 || a.OwnerID() == b.OwnerID() {
		t.Fatal("owner collision")
	}
	x := serviceValue(t, a, "123456789012345678901234567890/73")
	y := serviceValue(t, b, "987654321098765432109876543210/83")
	// Separate managers deliberately reuse numeric slot numbers; service provenance
	// must reject even valid-looking small numerals and matching raw handles.
	baseline := b.manager.z.cells.Stats().Live
	if !errors.Is(b.Copy(&y, &x), ErrRationalServiceOwner) || !errors.Is(b.SetInt64(&x, 2), ErrRationalServiceOwner) || b.manager.z.cells.Stats().Live != baseline {
		t.Fatal("foreign mutation")
	}
	serviceMust(t, b.CopyFrom(&y, a, &x))
	serviceText(t, b, &y, "123456789012345678901234567890/73")
	if x.raw.num.cell == 0 || y.raw.num.cell == 0 {
		t.Fatal("not actually big")
	}
	serviceMust(t, a.SetInt64(&x, 3))
	serviceText(t, b, &y, "123456789012345678901234567890/73")
	serviceMust(t, a.Close())
	serviceText(t, b, &y, "123456789012345678901234567890/73")
	if !errors.Is(b.CopyFrom(&y, a, &x), arena.ErrClosed) {
		t.Fatal("closed source accepted")
	}
	// Reconstruct the copied cold identity without copying a live mutex.
	copyService := RationalService{self: b, manager: b.manager, ownerID: b.ownerID}
	if _, e := copyService.NewValue(); !errors.Is(e, ErrRationalServiceCopy) {
		t.Fatal(e)
	}
	if e := copyService.Close(); !errors.Is(e, ErrRationalServiceCopy) {
		t.Fatal(e)
	}
	serviceMust(t, b.Del(&y))
	if !errors.Is(b.Del(&y), ErrRationalServiceOwner) {
		t.Fatal("destroyed wrapper accepted")
	}
	var raw OwnedRational
	if !errors.Is(b.SetInt64(&raw, 1), ErrRationalServiceOwner) {
		t.Fatal("zero token accepted")
	}
}

func TestRationalServiceLayoutIdentityAndExhaustion(t *testing.T) {
	serviceMust(t, arena.ValidateRecordType(reflect.TypeOf(OwnedRational{})))
	serviceMust(t, arena.ValidateRecordType(reflect.TypeOf(Rational{})))
	s, _ := serviceNew(t)
	id := s.OwnerID()
	v, _ := s.NewValue()
	serviceMust(t, s.Del(&v))
	serviceMust(t, s.Close())
	if s.OwnerID() != id {
		t.Fatal("identity changed")
	}
	old := atomic.LoadUint64(&rationalServiceNextOwnerID)
	atomic.StoreUint64(&rationalServiceNextOwnerID, math.MaxUint64)
	defer atomic.StoreUint64(&rationalServiceNextOwnerID, old)
	b := arena.NewBudget(1 << 20)
	_, e := NewRationalService(serviceConfig(b))
	if !errors.Is(e, arena.ErrBounds) || b.Used() != 0 {
		t.Fatal(e, b.Used())
	}
}

func TestRationalServiceConstructorBudgetUnwind(t *testing.T) {
	failures, successes := 0, 0
	for n := int64(0); n < 70; n++ {
		b := arena.NewBudget(1 << 24)
		b.FailAfter(n)
		s, e := NewRationalService(serviceConfig(b))
		if e != nil {
			failures++
			if !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != 0 {
				t.Fatal(n, e, b.Used())
			}
		} else {
			successes++
			serviceMust(t, s.Close())
			if b.Used() != 0 {
				t.Fatal(n, b.Used())
			}
		}
	}
	if failures == 0 || successes == 0 {
		t.Fatal(failures, successes)
	}
	b := arena.NewBudget(1 << 24)
	cfg := serviceConfig(b)
	cfg.Integers.Cells.PageSize = 3
	_, e := NewRationalService(cfg)
	if !errors.Is(e, arena.ErrConfig) || b.Used() != 0 {
		t.Fatal(e, b.Used())
	}
}

func TestRationalServiceCacheFailurePrefixes(t *testing.T) {
	failures, successes := 0, 0
	for n := int64(0); n < 160; n++ {
		s, b := serviceNew(t)
		b.FailAfter(n)
		v, e := s.PowerOfTwo(67)
		serviceAuditRationalRoots(t, s, &v.raw) // immediate pre-render/pre-Del graph
		b.FailAfter(-1)
		if e != nil {
			failures++
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(n, e)
			}
		} else {
			successes++
			serviceText(t, s, &v, new(big.Int).Lsh(big.NewInt(1), 67).String())
			serviceMust(t, s.Del(&v))
		}
		// Cache growth is a committed source prefix. Temporary curr/two/result values
		// must already be gone, before owner Close could mask any leaked numeral.
		// Source ensure_tmp can free its old buffer before failing: cached roots
		// are added to the ACTUAL remaining manager graph, not a fixed-six baseline.
		for i := range s.cache {
			got, err := s.manager.String(&s.cache[i])
			if err != nil || got != new(big.Int).Lsh(big.NewInt(1), uint(i)).String() {
				t.Fatal(n, i, got, err)
			}
		}
		serviceMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatal(n, b.Used())
		}
	}
	if failures == 0 || successes == 0 {
		t.Fatal(failures, successes)
	}
}

func TestRationalServiceCopyFailureAndWarmReuse(t *testing.T) {
	s, b := serviceNew(t)
	src := serviceValue(t, s, "1234567890123456789012345678901234567890/99887766554433221100")
	dst := serviceValue(t, s, "7")
	// Occupy every free numeric cell so deep Copy requires actual cold growth.
	var fillers []OwnedRational
	for s.manager.z.cells.free != 0 {
		v, err := s.NewValue()
		serviceMust(t, err)
		serviceMust(t, s.SetUint64(&v, 4294967295))
		fillers = append(fillers, v)
	}
	b.FailAfter(0)
	e := s.Copy(&dst, &src)
	roots := []*Rational{&src.raw, &dst.raw}
	for i := range fillers {
		roots = append(roots, &fillers[i].raw)
	}
	serviceAuditRationalRoots(t, s, roots...)
	if !errors.Is(e, arena.ErrInjectedFailure) || dst.raw.num.cell != s.manager.z.failedRef || dst.raw.den != NewInt(1) || !errors.Is(s.manager.z.check(&dst.raw.num), arena.ErrInvalidSlot) {
		t.Fatal("sequential Copy numerator refusal", dst, e)
	}
	b.FailAfter(-1)
	sourceWant, ok := new(big.Rat).SetString("1234567890123456789012345678901234567890/99887766554433221100")
	if !ok {
		t.Fatal("invalid independent source fixture")
	}
	serviceText(t, s, &src, sourceWant.RatString())
	serviceMust(t, s.Del(&dst))
	dst, e = s.NewValue()
	serviceMust(t, e)
	for i := range fillers {
		serviceMust(t, s.Del(&fillers[i]))
	}
	serviceMust(t, s.Copy(&dst, &src))
	serviceMust(t, s.Del(&dst))
	dst, _ = s.NewValue()
	warm, e := s.PowerOfTwo(80)
	serviceMust(t, e)
	serviceMust(t, s.Del(&warm))
	warm, e = s.PowerOfTwo(80)
	serviceMust(t, e)
	serviceMust(t, s.Del(&warm))
	b.FailAfter(0)
	var failed error
	allocs := testing.AllocsPerRun(100, func() {
		v, e := s.PowerOfTwo(80)
		if e != nil {
			failed = e
			return
		}
		if e = s.Del(&v); e != nil {
			failed = e
		}
	})
	b.FailAfter(-1)
	if failed != nil || allocs != 0 {
		t.Fatal(failed, allocs)
	}
	serviceMust(t, s.Del(&src))
	serviceMust(t, s.Del(&dst))
}

func TestRationalServiceCacheBackingPeakAndBounds(t *testing.T) {
	s, b := serviceNew(t)
	v, e := s.PowerOfTwo(0)
	serviceMust(t, e)
	serviceMust(t, s.Del(&v))
	before := b.Used()
	stats, _ := s.Stats()
	if stats.CacheCapacity != 2 {
		t.Fatal(stats)
	}
	// Small powers need no numeral payload growth: precisely the entire new
	// five-element array must fit alongside the intermediate three-element array.
	bytes := uint64(reflect.TypeOf(Rational{}).Size()) * 6
	hold := b.Limit() - before - bytes + 1
	serviceMust(t, b.Reserve(hold))
	_, e = s.PowerOfTwo(2)
	if !errors.Is(e, arena.ErrBudget) || len(s.cache) != 3 || cap(s.cache) != 3 {
		t.Fatal(e, len(s.cache), cap(s.cache))
	}
	b.Release(1)
	v, e = s.PowerOfTwo(3)
	serviceMust(t, e)
	serviceMust(t, s.Del(&v))
	b.Release(hold - 1)
	if len(s.cache) != 5 {
		t.Fatal("inclusive extra entry", len(s.cache))
	}
	before = b.Used()
	stats, _ = s.Stats()
	_, e = s.PowerOfTwo(math.MaxUint32)
	if !errors.Is(e, arena.ErrBounds) || b.Used() != before {
		t.Fatal(e, b.Used(), before)
	}
	after, _ := s.Stats()
	if after != stats {
		t.Fatal(after, stats)
	}
}

func TestRationalServiceConcurrentCacheAndClose(t *testing.T) {
	s, _ := serviceNew(t)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := uint32(0); k < 40; k++ {
				v, e := s.PowerOfTwo(k)
				if e != nil {
					t.Error(e)
					return
				}
				got, e := s.String(&v)
				if e != nil || got != new(big.Int).Lsh(big.NewInt(1), uint(k)).String() {
					t.Error(got, e)
				}
				if e = s.Del(&v); e != nil {
					t.Error(e)
				}
			}
		}()
	}
	wg.Wait()
	serviceMust(t, s.Close())
	if _, e := s.PowerOfTwo(1); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func serviceScratchRefs(s *RationalService) uint32 {
	var n uint32
	for _, r := range s.manager.z.natural.divScratch {
		if r != (arena.VectorRef{}) {
			n++
		}
	}
	for _, r := range s.manager.z.natural.formatScratch {
		if r != (arena.VectorRef{}) {
			n++
		}
	}
	return n
}

func TestRationalServiceCrossOwnerFailureAndSameOwnerCopy(t *testing.T) {
	source, _ := serviceNew(t)
	input := serviceValue(t, source, "123456789012345678901234567891/99887766554433221101")
	failures, successes := 0, 0
	for n := int64(0); n < 70; n++ {
		dest, b := serviceNew(t)
		out := serviceValue(t, dest, "3")
		cells, words := dest.manager.z.cells.Stats().Live, dest.manager.z.words.Stats().Live
		b.FailAfter(n)
		e := dest.CopyFrom(&out, source, &input)
		b.FailAfter(-1)
		if e != nil {
			failures++
			if !errors.Is(e, arena.ErrInjectedFailure) || dest.manager.z.cells.Stats().Live != cells || dest.manager.z.words.Stats().Live != words {
				t.Fatal(n, e, "immediate temporary ownership leak")
			}
			serviceText(t, dest, &out, "3")
		} else {
			successes++
			serviceText(t, dest, &out, "123456789012345678901234567891/99887766554433221101")
		}
		serviceMust(t, dest.Del(&out))
		serviceMust(t, dest.Close())
		if b.Used() != 0 {
			t.Fatal(n, b.Used())
		}
	}
	if failures == 0 || successes == 0 {
		t.Fatal(failures, successes)
	}
	copied, _ := source.NewValue()
	serviceMust(t, source.Copy(&copied, &input))
	if copied.raw.num.cell == input.raw.num.cell || copied.raw.den.cell == input.raw.den.cell {
		t.Fatal("shallow owning copy")
	}
	serviceMust(t, source.SetInt64(&input, 9))
	serviceText(t, source, &copied, "123456789012345678901234567891/99887766554433221101")
	serviceMust(t, source.CopyFrom(&copied, source, &copied))
	serviceMust(t, source.Move(&copied, &copied))
	serviceMust(t, source.Swap(&copied, &copied))
	serviceMust(t, source.Del(&input))
	serviceMust(t, source.Del(&copied))
}

func TestRationalServiceForeignSmallValuesAndInvalidGuards(t *testing.T) {
	a, ba := serviceNew(t)
	b, _ := serviceNew(t)
	foreign, _ := b.NewValue()
	local, _ := a.NewValue()
	serviceMust(t, a.SetInt64(&local, 17))
	used := ba.Used()
	for _, e := range []error{a.Del(&foreign), a.Move(&local, &foreign), a.Swap(&local, &foreign), a.SetFractionSigned(&foreign, 1, 2), a.CopyFrom(&local, a, &foreign)} {
		if !errors.Is(e, ErrRationalServiceOwner) {
			t.Fatal(e)
		}
	}
	if _, e := a.Equal(&local, &foreign); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal(e)
	}
	if _, e := a.String(&foreign); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal(e)
	}
	if e := a.SetFractionSigned(&local, 1, 0); !errors.Is(e, ErrDivideByZero) {
		t.Fatal(e)
	}
	if e := a.SetFraction(&local, 1, 0); !errors.Is(e, ErrDivideByZero) {
		t.Fatal(e)
	}
	if local.raw.num.val != 17 || !local.raw.IsSmall() {
		t.Fatal("guard changed value")
	}
	if ba.Used() != used {
		t.Fatal("guard changed charges", ba.Used(), used)
	}
	serviceText(t, a, &local, "17")
	serviceMust(t, a.Del(&local))
	serviceMust(t, b.Del(&foreign))
}

func TestRationalServiceCacheRelocationMoveAndDestruction(t *testing.T) {
	s, _ := serviceNew(t)
	v, e := s.PowerOfTwo(31)
	serviceMust(t, e)
	serviceMust(t, s.Del(&v))
	old := s.cache
	refs := make([]numericCellRef, len(old))
	vals := make([]int32, len(old))
	for i := range old {
		refs[i] = old[i].num.cell
		vals[i] = old[i].num.val
	}
	// Force source expansion by requesting the first index past current capacity.
	// The old backing is retained only by this cold test observer.
	v, e = s.PowerOfTwo(uint32(cap(old)))
	serviceMust(t, e)
	serviceMust(t, s.Del(&v))
	for i := range old {
		if old[i].num.cell != 0 || old[i].den.cell != 0 || old[i].num.val != vals[i] || s.cache[i].num.cell != refs[i] {
			t.Fatal("relocation deep-copied/recycled source handles or lost moved-from history", i)
		}
	}
	// Isolate source ascending cache destructor order through real LIFO slot
	// reuse. Constants and scratch retain their own cells; no callback is added.
	saved := make([]numericCellRef, 0)
	for i := range s.cache {
		if s.cache[i].num.cell != 0 {
			saved = append(saved, s.cache[i].num.cell)
		}
	}
	s.mu.Lock()
	s.destroyCache()
	s.mu.Unlock()
	if s.cache != nil || s.cacheBytes != 0 {
		t.Fatal("production cache destructor retained backing")
	}
	var reused []OwnedRational
	for i := len(saved) - 1; i >= 0; i-- {
		x, _ := s.NewValue()
		serviceMust(t, s.SetUint64(&x, math.MaxUint32))
		if numericSlot(x.raw.num.cell) != numericSlot(saved[i]) {
			t.Fatal("cache destructor order", i, x.raw.num.cell, saved[i])
		}
		reused = append(reused, x)
	}
	for i := range reused {
		serviceMust(t, s.Del(&reused[i]))
	}
}

func TestRationalServiceSourceUnsignedConstructorBoundary(t *testing.T) {
	s, _ := serviceNew(t)
	v, _ := s.NewValue()
	baseline := s.manager.z.cells.Stats().Live
	for _, n := range []uint32{math.MaxInt32 - 1, math.MaxInt32, math.MaxInt32 + 1, math.MaxUint32} {
		// Existing big numerator AND denominator must be destroyed when constructor
		// setter publishes an inline numerator and resets the denominator.
		serviceMust(t, s.SetString(&v, "123456789012345678901/99887766554433221101"))
		serviceMust(t, s.SetUnsigned(&v, n))
		wantSmall := n <= math.MaxInt32
		if v.raw.num.IsSmall() != wantSmall || !v.raw.den.IsSmall() || v.raw.den.val != 1 {
			t.Fatal(n, v.raw.num, v.raw.den)
		}
		expected := baseline
		if !wantSmall {
			expected++
		}
		if s.manager.z.cells.Stats().Live != expected {
			t.Fatal("big old output not released", n, s.manager.z.cells.Stats().Live, expected)
		}
		serviceText(t, s, &v, new(big.Int).SetUint64(uint64(n)).String())
	}
	// The typed64 source overload deliberately has the stricter '< INT_MAX'
	// condition: it must remain separate from the unsigned32 constructor.
	serviceMust(t, s.SetUint64(&v, math.MaxInt32))
	if v.raw.num.IsSmall() {
		t.Fatal("uint64 overload lost source shape")
	}
	serviceMust(t, s.SetUnsigned(&v, math.MaxInt32))
	if !v.raw.num.IsSmall() {
		t.Fatal("unsigned32 overload lost inclusive boundary")
	}
	serviceMust(t, s.Del(&v))
}

func TestRationalServiceSourceSignedFractionShape(t *testing.T) {
	s, _ := serviceNew(t)
	v, _ := s.NewValue()
	baseline := s.manager.z.cells.Stats().Live
	for _, p := range [][2]int32{{1, math.MaxInt32}, {-1, -math.MaxInt32}, {math.MaxInt32, 1}, {math.MaxInt32, 2}, {math.MaxInt32 - 1, math.MaxInt32}} {
		serviceMust(t, s.SetString(&v, "123456789012345678901/99887766554433221101"))
		serviceMust(t, s.SetFractionSigned(&v, p[0], p[1]))
		if !v.raw.num.IsSmall() || !v.raw.den.IsSmall() {
			t.Fatal("signed fraction constructor changed finite source representation", p, v.raw.num, v.raw.den)
		}
		if s.manager.z.cells.Stats().Live != baseline {
			t.Fatal("old big output not released", p, s.manager.z.cells.Stats().Live, baseline)
		}
		serviceText(t, s, &v, big.NewRat(int64(p[0]), int64(p[1])).RatString())
	}
	// The int64/uint64 pair has the source strict unsigned64 denominator path.
	serviceMust(t, s.SetFraction(&v, 1, math.MaxInt32))
	if v.raw.den.IsSmall() {
		t.Fatal("typed64 fraction overload lost representation")
	}
	serviceMust(t, s.SetFractionSigned(&v, 1, math.MaxInt32))
	if !v.raw.den.IsSmall() {
		t.Fatal("signed32 fraction overload lost inclusive boundary")
	}
	serviceMust(t, s.Del(&v))
	failures, successes := 0, 0
	for n := int64(0); n < 24; n++ {
		owner, b := serviceNew(t)
		out, _ := owner.NewValue()
		serviceMust(t, owner.SetInt64(&out, 3))
		cells, words := owner.manager.z.cells.Stats().Live, owner.manager.z.words.Stats().Live
		b.FailAfter(n)
		err := owner.SetFractionSigned(&out, math.MinInt32, -1)
		b.FailAfter(-1)
		if err != nil {
			failures++
			if !errors.Is(err, arena.ErrInjectedFailure) || owner.manager.z.cells.Stats().Live != cells || owner.manager.z.words.Stats().Live != words {
				t.Fatal("signed pair temporary leak before teardown", n, err)
			}
			if out.raw.num.val != 3 || !out.raw.IsSmall() {
				t.Fatal("failed staged pair published", out)
			}
		} else {
			successes++
			serviceText(t, owner, &out, "2147483648")
		}
		serviceMust(t, owner.Del(&out))
		serviceMust(t, owner.Close())
		if b.Used() != 0 {
			t.Fatal(n, b.Used())
		}
	}
	if failures == 0 || successes == 0 {
		t.Fatal(failures, successes)
	}
}

// Audit actual live roots before formatting or teardown. Missing source TMP
// buffers and the reserved permanently-dead failure token are not live cells;
// qopsAuditLiveOwnership validates the manager's actual physical ownership.
func serviceAuditRationalRoots(t *testing.T, s *RationalService, values ...*Rational) {
	t.Helper()
	roots := []*Int{}
	add := func(q *Rational) { roots = append(roots, &q.num, &q.den) }
	for _, q := range []*Rational{&s.zero, &s.one, &s.minusOne, &s.infZero.first, &s.infZero.second, &s.infOne.first, &s.infOne.second, &s.infMinusOne.first, &s.infMinusOne.second, &s.infIntZero.first, &s.infIntOne.first, &s.infIntMinusOne.first} {
		add(q)
	}
	for i := range s.cache {
		add(&s.cache[i])
	}
	for _, q := range values {
		if q != nil {
			add(q)
		}
	}
	qopsAuditLiveOwnership(t, s.manager, roots)
}
