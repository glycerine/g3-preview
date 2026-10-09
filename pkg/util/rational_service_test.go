// Selected assertions translated from src/test/rational.cpp.
package util

import (
	"errors"
	"math"
	"math/big"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
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
	originalNum, originalDen := minus.Raw.Num.Val, minus.Raw.Den.Val
	numRef, denRef := minus.Raw.Num.Cell, minus.Raw.Den.Cell
	serviceMust(t, s.Move(&zero, &minus))
	if zero.Raw.Num.Cell != numRef || zero.Raw.Den.Cell != denRef || minus.Raw.Num.Cell != 0 || minus.Raw.Den.Cell != 0 || minus.Raw.Num.Val != originalNum || minus.Raw.Den.Val != originalDen {
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
	length := uint32(0)
	for _, k := range []uint32{0, 1, 2, 4, 3, 9, 31, 63, 127, 255, 128} {
		v, e := s.PowerOfTwo(k)
		serviceMust(t, e)
		want := new(big.Int).Lsh(big.NewInt(1), uint(k)).String()
		serviceText(t, s, &v, want)
		if k >= length {
			length = k + 2
		}
		stats, e := s.Stats()
		serviceMust(t, e)
		if stats.CacheLength != length || stats.CacheCapacity < length {
			t.Fatal(k, stats, length)
		}
		serviceMust(t, s.SetInt64(&v, 7))
		serviceMust(t, s.Del(&v))
		again, e := s.PowerOfTwo(k)
		serviceMust(t, e)
		serviceText(t, s, &again, want)
		serviceMust(t, s.Del(&again))
	}
	for i := uint32(0); i < uint32(len(s.Cache)); i++ {
		got, e := s.Manager.String(&s.Cache[i])
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
	baseline := b.Manager.Z.Cells.Stats().Live
	if !errors.Is(b.Copy(&y, &x), ErrRationalServiceOwner) || !errors.Is(b.SetInt64(&x, 2), ErrRationalServiceOwner) || b.Manager.Z.Cells.Stats().Live != baseline {
		t.Fatal("foreign mutation")
	}
	serviceMust(t, b.CopyFrom(&y, a, &x))
	serviceText(t, b, &y, "123456789012345678901234567890/73")
	if x.Raw.Num.Cell == 0 || y.Raw.Num.Cell == 0 {
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
	copyService := RationalService{self: b, Manager: b.Manager, ownerID: b.ownerID}
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
			constructionFailureOwningCleanup(t, e, b)
			if !numericPhysicalReservationRefusal(e) || b.Used() != 0 {
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
		serviceAuditRationalRoots(t, s, &v.Raw) // immediate pre-render/pre-Del graph
		b.FailAfter(-1)
		if e != nil {
			failures++
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal(n, e)
			}
		} else {
			successes++
			serviceText(t, s, &v, new(big.Int).Lsh(big.NewInt(1), 67).String())
			serviceMust(t, s.Del(&v))
		}
		// Temporary curr/two/result values must already be gone before owner Close
		// could mask any leaked numeral. Cached roots remain part of the actual graph.
		for i := range s.Cache {
			got, err := s.Manager.String(&s.Cache[i])
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
	for s.Manager.Z.Cells.Free != 0 {
		v, err := s.NewValue()
		serviceMust(t, err)
		serviceMust(t, s.SetUint64(&v, 4294967295))
		fillers = append(fillers, v)
	}
	b.FailAfter(0)
	e := s.Copy(&dst, &src)
	roots := []*Rational{&src.Raw, &dst.Raw}
	for i := range fillers {
		roots = append(roots, &fillers[i].Raw)
	}
	serviceAuditRationalRoots(t, s, roots...)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal("Copy category", dst, e)
	}
	setterPairPrefix(t, s.Manager, &dst.Raw, "7", "1", roundPublicationDecode(t, s.Manager.Z, &src.Raw.Num), roundPublicationDecode(t, s.Manager.Z, &src.Raw.Den), true)
	b.FailAfter(-1)
	serviceMust(t, s.Manager.Z.CleanupPendingAllocation()) // explicit release after immediate named audit
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
	beforeWarm := numericWarmProfileSnapshot(t, s.Manager.Z, b)
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
	if failed != nil {
		t.Fatal(failed)
	}
	numericWarmProfileAudit(t, s.Manager.Z, b, beforeWarm, allocs, s.Manager.Z.Allocator.Stats().DirectObjects > 0)
	serviceMust(t, s.Del(&src))
	serviceMust(t, s.Del(&dst))
}

func TestRationalServiceCacheNativeStorageAndBounds(t *testing.T) {
	s, b := serviceNew(t)
	v, e := s.PowerOfTwo(0)
	serviceMust(t, e)
	serviceMust(t, s.Del(&v))
	before := b.Used()
	v, e = s.PowerOfTwo(3)
	serviceMust(t, e)
	serviceMust(t, s.Del(&v))
	if len(s.Cache) != 5 {
		t.Fatal("inclusive extra entry", len(s.Cache))
	}
	if b.Used() != before {
		t.Fatal("Go cache backing charged to numeric arena", b.Used(), before)
	}
	before = b.Used()
	stats, _ := s.Stats()
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
	for _, r := range s.Manager.Z.Natural.divScratch {
		if r != (arena.VectorRef{}) {
			n++
		}
	}
	for _, r := range s.Manager.Z.Natural.FormatScratch {
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
		cells, words := dest.Manager.Z.Cells.Stats().Live, dest.Manager.Z.Words.Stats().Live
		b.FailAfter(n)
		e := dest.CopyFrom(&out, source, &input)
		b.FailAfter(-1)
		if e != nil {
			failures++
			if !numericPhysicalReservationRefusal(e) || dest.Manager.Z.Cells.Stats().Live != cells || dest.Manager.Z.Words.Stats().Live != words {
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
	if copied.Raw.Num.Cell == input.Raw.Num.Cell || copied.Raw.Den.Cell == input.Raw.Den.Cell {
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
	for _, signed := range []bool{true, false} {
		var e error
		if signed {
			e = a.SetFractionSigned(&local, 1, 0)
		} else {
			e = a.SetFraction(&local, 1, 0)
		}
		serviceAuditRationalRoots(t, a, &local.Raw)
		if modularSourceDebug {
			if !errors.Is(e, ErrIntegerAssertion) || local.Raw != (Rational{Num: NewInt(17), Den: NewInt(1)}) {
				t.Fatal("debug source zero DEN assertion", local, e)
			}
		} else {
			if e != nil || local.Raw != (Rational{Num: NewInt(1), Den: NewInt(0)}) {
				t.Fatal("finite source invalid denominator publication", local, e)
			}
			if _, e = a.String(&local); !errors.Is(e, ErrUninitializedRational) {
				t.Fatal(e)
			}
			// Explicit cleanup action; no successful arithmetic retry on invalid fields.
			serviceMust(t, a.Del(&local))
			local, _ = a.NewValue()
			serviceMust(t, a.SetInt64(&local, 17))
		}
	}
	if ba.Used() != used {
		t.Fatal("inline branch changed charges", ba.Used(), used)
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
	old := s.Cache
	refs := make([]NumericCellRef, len(old))
	vals := make([]int32, len(old))
	for i := range old {
		refs[i] = old[i].Num.Cell
		vals[i] = old[i].Num.Val
	}
	// Force native slice expansion by requesting the first index past capacity.
	v, e = s.PowerOfTwo(uint32(cap(old)))
	serviceMust(t, e)
	serviceMust(t, s.Del(&v))
	for i := range old {
		if old[i].Num.Val != vals[i] || s.Cache[i].Num.Cell != refs[i] {
			t.Fatal("slice relocation changed owned numeric identity", i)
		}
	}
	// Isolate source ascending cache destructor order through real LIFO slot
	// reuse. Constants and scratch retain their own cells; no callback is added.
	saved := make([]NumericCellRef, 0)
	for i := range s.Cache {
		if s.Cache[i].Num.Cell != 0 {
			saved = append(saved, s.Cache[i].Num.Cell)
		}
	}
	s.mu.Lock()
	s.destroyCache()
	s.mu.Unlock()
	if s.Cache != nil {
		t.Fatal("production cache destructor retained backing")
	}
	var reused []OwnedRational
	for i := len(saved) - 1; i >= 0; i-- {
		x, _ := s.NewValue()
		serviceMust(t, s.SetUint64(&x, math.MaxUint32))
		if NumericSlot(x.Raw.Num.Cell) != NumericSlot(saved[i]) {
			t.Fatal("cache destructor order", i, x.Raw.Num.Cell, saved[i])
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
	baseline := s.Manager.Z.Cells.Stats().Live
	for _, n := range []uint32{math.MaxInt32 - 1, math.MaxInt32, math.MaxInt32 + 1, math.MaxUint32} {
		// Existing big numerator AND denominator must be destroyed when constructor
		// setter publishes an inline numerator and resets the denominator.
		serviceMust(t, s.SetString(&v, "123456789012345678901/99887766554433221101"))
		serviceMust(t, s.SetUnsigned(&v, n))
		wantSmall := n <= math.MaxInt32
		if v.Raw.Num.IsSmall() != wantSmall || !v.Raw.Den.IsSmall() || v.Raw.Den.Val != 1 {
			t.Fatal(n, v.Raw.Num, v.Raw.Den)
		}
		expected := baseline
		if !wantSmall {
			expected++
		}
		if s.Manager.Z.Cells.Stats().Live != expected {
			t.Fatal("big old output not released", n, s.Manager.Z.Cells.Stats().Live, expected)
		}
		serviceText(t, s, &v, new(big.Int).SetUint64(uint64(n)).String())
	}
	// The typed64 source overload deliberately has the stricter '< INT_MAX'
	// condition: it must remain separate from the unsigned32 constructor.
	serviceMust(t, s.SetUint64(&v, math.MaxInt32))
	if v.Raw.Num.IsSmall() {
		t.Fatal("uint64 overload lost source shape")
	}
	serviceMust(t, s.SetUnsigned(&v, math.MaxInt32))
	if !v.Raw.Num.IsSmall() {
		t.Fatal("unsigned32 overload lost inclusive boundary")
	}
	serviceMust(t, s.Del(&v))
}

func TestRationalServiceSourceSignedFractionShape(t *testing.T) {
	s, _ := serviceNew(t)
	v, _ := s.NewValue()
	baseline := s.Manager.Z.Cells.Stats().Live
	for _, p := range [][2]int32{{1, math.MaxInt32}, {-1, -math.MaxInt32}, {math.MaxInt32, 1}, {math.MaxInt32, 2}, {math.MaxInt32 - 1, math.MaxInt32}} {
		serviceMust(t, s.SetString(&v, "123456789012345678901/99887766554433221101"))
		serviceMust(t, s.SetFractionSigned(&v, p[0], p[1]))
		if !v.Raw.Num.IsSmall() || !v.Raw.Den.IsSmall() {
			t.Fatal("signed fraction constructor changed finite source representation", p, v.Raw.Num, v.Raw.Den)
		}
		if s.Manager.Z.Cells.Stats().Live != baseline {
			t.Fatal("old big output not released", p, s.Manager.Z.Cells.Stats().Live, baseline)
		}
		serviceText(t, s, &v, big.NewRat(int64(p[0]), int64(p[1])).RatString())
	}
	// The int64/uint64 pair has the source strict unsigned64 denominator path.
	serviceMust(t, s.SetFraction(&v, 1, math.MaxInt32))
	if v.Raw.Den.IsSmall() {
		t.Fatal("typed64 fraction overload lost representation")
	}
	serviceMust(t, s.SetFractionSigned(&v, 1, math.MaxInt32))
	if !v.Raw.Den.IsSmall() {
		t.Fatal("signed32 fraction overload lost inclusive boundary")
	}
	serviceMust(t, s.Del(&v))
	failures, successes := 0, 0
	for n := int64(0); n < 24; n++ {
		owner, b := serviceNew(t)
		out, _ := owner.NewValue()
		serviceMust(t, owner.SetInt64(&out, 3))
		b.FailAfter(n)
		err := owner.SetFractionSigned(&out, math.MinInt32, -1)
		serviceAuditRationalRoots(t, owner, &out.Raw)
		if err != nil {
			failures++
			if !numericPhysicalReservationRefusal(err) {
				t.Fatal("signed pair temporary leak before teardown", n, err)
			}
			setterPairPrefix(t, owner.Manager, &out.Raw, "3", "1", big.NewInt(2147483648), big.NewInt(1), false)
		} else {
			successes++
			b.FailAfter(-1)
			serviceText(t, owner, &out, "2147483648")
		}
		b.FailAfter(-1)
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
	add := func(q *Rational) { roots = append(roots, &q.Num, &q.Den) }
	for _, q := range []*Rational{&s.ZeroField, &s.OneField, &s.MinusOneField, &s.InfZeroField.First, &s.InfZeroField.Second, &s.InfOneField.First, &s.InfOneField.Second, &s.InfMinusOneField.First, &s.InfMinusOneField.Second, &s.InfIntZeroField.First, &s.InfIntOneField.First, &s.InfIntMinusOneField.First} {
		add(q)
	}
	for i := range s.Cache {
		add(&s.Cache[i])
	}
	for _, q := range values {
		if q != nil {
			add(q)
		}
	}
	qopsAuditLiveOwnership(t, s.Manager, roots)
}

// Failure-route checks read physical published components without allocating
// the source decimal formatter's scratch after a failed arithmetic action.
func serviceIndependentText(t *testing.T, s *RationalService, a *OwnedRational, want string) {
	t.Helper()
	expected, ok := new(big.Rat).SetString(want)
	if !ok {
		t.Fatal(want)
	}
	n, d := roundPublicationDecode(t, s.Manager.Z, &a.Raw.Num), roundPublicationDecode(t, s.Manager.Z, &a.Raw.Den)
	if n.Cmp(expected.Num()) != 0 || d.Cmp(expected.Denom()) != 0 {
		t.Fatal("source raw components changed", n, d, expected)
	}
}
