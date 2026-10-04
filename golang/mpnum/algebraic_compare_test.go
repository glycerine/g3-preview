// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"testing"
)

type algebraicCompareControl struct {
	checks, cooperations uint32
	allow                bool
}

func algebraicCompareFixture(t *testing.T) (*AlgebraicRootCore, *UPolynomialManager, *algebraicCompareControl) {
	return algebraicCompareFixtureProfile(t, false)
}
func algebraicCompareFixtureProfile(t *testing.T, guardProfile bool) (*AlgebraicRootCore, *UPolynomialManager, *algebraicCompareControl) {
	t.Helper()
	var s *RationalService
	profile := memory.SmallAllocatorProfile{PointerBits: 64}
	if !guardProfile {
		s, _ = serviceNew(t)
	} else {
		budget := arena.NewBudget(1 << 25)
		runtime, e := memory.NewRuntime(memory.RuntimeConfig{Budget: budget, SizeTBits: 32})
		serviceMust(t, e)
		cfg := serviceConfig(budget)
		cfg.Integers.Memory = runtime
		profile = memory.SmallAllocatorProfile{PointerBits: 32, Debug: true}
		cfg.Integers.Profile = profile
		s, e = NewRationalService(cfg)
		serviceMust(t, e)
		t.Cleanup(func() {
			serviceMust(t, s.Close())
			serviceMust(t, runtime.Close())
			if budget.Used() != 0 {
				t.Fatal(budget.Used())
			}
		})
	}
	control := &algebraicCompareControl{allow: true}
	p, e := s.NewUPolynomialManager(UPolynomialConfig{Checkpoint: func() bool { control.checks++; return control.allow }, Cooperation: func() error { control.cooperations++; return nil }})
	serviceMust(t, e)
	primes, e := NewPrimeGenerator(s.manager.budget)
	serviceMust(t, e)
	f, e := NewUPolynomialFactorizationService(p, UPolynomialFactorizationConfig{Primes: primes})
	serviceMust(t, e)
	b, e := NewBinaryRationalManager(s.manager.z, BinaryRationalConfig{Diagnostics: s})
	serviceMust(t, e)
	bi, e := NewBinaryIntervalManager(b)
	serviceMust(t, e)
	a, e := memory.NewSmallAllocator(memory.SmallAllocatorConfig{Runtime: s.manager.z.memory, Profile: profile})
	serviceMust(t, e)
	c, e := NewAlgebraicRootCore(AlgebraicRootConfig{Rationals: s.manager, Polynomials: p, Factors: f, Binary: b, Intervals: bi, Allocator: a, Checkpoint: p.config.Checkpoint, Cooperation: p.config.Cooperation})
	serviceMust(t, e)
	t.Cleanup(func() {
		if control.allow && control.checks != control.cooperations {
			t.Error("source cooperation call pairing", *control)
		}
		serviceMust(t, c.Close())
		serviceMust(t, a.Close())
		bi.Close()
		serviceMust(t, b.CloseChecked())
		serviceMust(t, f.Close())
		serviceMust(t, primes.Close())
		serviceMust(t, p.Close())
	})
	return c, p, control
}
func algebraicCompareRoot(t *testing.T, c *AlgebraicRootCore, coeff []int32, ln int32, lk uint32, un int32, uk uint32, minimal bool) Algebraic {
	t.Helper()
	p := c.config.Polynomials
	poly, e := p.NewPolynomial()
	serviceMust(t, e)
	values := make([]Int, len(coeff))
	for i, n := range coeff {
		values[i] = NewInt(n)
	}
	serviceMust(t, p.SetInts(poly, values))
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	var l, u BinaryRational
	serviceMust(t, c.config.Binary.SetInt32Power(&l, ln, lk))
	serviceMust(t, c.config.Binary.SetInt32Power(&u, un, uk))
	var a Algebraic
	serviceMust(t, c.SetRoot(&a, v, &l, &u, minimal))
	serviceMust(t, c.config.Binary.Del(&l))
	serviceMust(t, c.config.Binary.Del(&u))
	serviceMust(t, p.DelPolynomial(&poly))
	return a
}

func TestAlgebraicCompareBasicAndRationalDispatch(t *testing.T) {
	c, _, control := algebraicCompareFixture(t)
	for n := int64(-3); n <= 3; n++ {
		for d := int64(1); d <= 3; d++ {
			q := NewRational()
			serviceMust(t, c.config.Rationals.SetFraction(&q, n, uint64(d)))
			var a, z Algebraic
			serviceMust(t, c.SetRational(&a, &q))
			got, e := c.CompareRational(&a, &q)
			serviceMust(t, e)
			if got != 0 {
				t.Fatal(n, d, got)
			}
			got, e = c.Compare(&a, &z)
			serviceMust(t, e)
			want := 0
			if n < 0 {
				want = -1
			}
			if n > 0 {
				want = 1
			}
			if got != want {
				t.Fatal(n, d, got, want)
			}
			got, e = c.Compare(&z, &a)
			serviceMust(t, e)
			if got != -want {
				t.Fatal(n, d, got)
			}
			serviceMust(t, c.Del(&a))
			serviceMust(t, c.config.Rationals.Del(&q))
		}
	}
	a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 3, 1, false)
	defer func() { serviceMust(t, c.Del(&a)) }()
	for _, tc := range []struct {
		n, d int64
		want int
	}{{1, 1, 1}, {3, 2, -1}, {7, 5, 1}, {10, 7, -1}} {
		q := NewRational()
		serviceMust(t, c.config.Rationals.SetFraction(&q, tc.n, uint64(tc.d)))
		got, e := c.CompareRational(&a, &q)
		serviceMust(t, e)
		if got != tc.want {
			t.Fatal(tc, got)
		}
		var b Algebraic
		serviceMust(t, c.SetRational(&b, &q))
		got, e = c.Compare(&b, &a)
		serviceMust(t, e)
		if got != -tc.want {
			t.Fatal(tc, got)
		}
		serviceMust(t, c.Del(&b))
		serviceMust(t, c.config.Rationals.Del(&q))
	}
	if control.checks != 0 || control.cooperations != 0 {
		t.Fatal("pure dispatch unexpectedly checkpointed", *control)
	}
}

// Source tst1 algebraic.cpp96/112 are printed comparison operations, not
// ENSUREs. These exact selected polynomial roots carry added finite expectations;
// the original multivariate expression-builder/print trace remains separate.
func TestAlgebraicCompareSourceOverlappingRootFixture(t *testing.T) {
	c, _, control := algebraicCompareFixture(t)
	c.minMagnitude = 0
	sqrt2 := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 3, 1, false)
	defer func() { serviceMust(t, c.Del(&sqrt2)) }()
	hidden := algebraicCompareRoot(t, c, []int32{6, 0, -5, 0, 1}, 1, 0, 3, 1, false)
	defer func() { serviceMust(t, c.Del(&hidden)) }()
	got, e := c.Compare(&sqrt2, &hidden)
	serviceMust(t, e)
	if got != 0 {
		t.Fatal("hidden sqrt2", got)
	}
	gauss := algebraicCompareRoot(t, c, []int32{-2, 1, 2, -1, 0, 2, -3, 1}, 9, 3, 5, 2, false)
	defer func() { serviceMust(t, c.Del(&gauss)) }()
	got, e = c.Compare(&sqrt2, &gauss)
	serviceMust(t, e)
	if got != 1 {
		t.Fatal("sqrt2 vs gauss", got)
	}
	got, e = c.Compare(&gauss, &sqrt2)
	serviceMust(t, e)
	if got != -1 {
		t.Fatal("gauss vs sqrt2", got)
	}
	if control.checks == 0 || control.checks != control.cooperations {
		t.Fatal("shared polynomial resource hooks", *control)
	}
}

func TestAlgebraicCompareIntervalsPolynomialAndCounterWrap(t *testing.T) {
	c, _, control := algebraicCompareFixture(t)
	a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 3, 1, false)
	b := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, -3, 1, -1, 0, false)
	same := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 5, 2, 3, 1, false)
	defer func() { serviceMust(t, c.Del(&same)); serviceMust(t, c.Del(&b)); serviceMust(t, c.Del(&a)) }()
	c.compareCheap = ^uint32(0)
	got, e := c.Compare(&a, &b)
	serviceMust(t, e)
	if got != 1 || c.compareCheap != 0 {
		t.Fatal(got, c.compareCheap)
	}
	c.comparePoly = ^uint32(0)
	got, e = c.Compare(&a, &same)
	serviceMust(t, e)
	if got != 0 || c.comparePoly != 0 {
		t.Fatal(got, c.comparePoly)
	}
	// Neither the disjoint nor equal-polynomial route reaches the real checkpoint.
	control.allow = false
	got, e = c.Compare(&a, &b)
	serviceMust(t, e)
	if got != 1 {
		t.Fatal(got)
	}
	got, e = c.Compare(&a, &same)
	serviceMust(t, e)
	if got != 0 {
		t.Fatal(got)
	}
	if control.checks != 0 || control.cooperations != 0 {
		t.Fatal("unreached source checkpoint", *control)
	}
}

func TestAlgebraicCompareMinimalAndMagnitudeRefinement(t *testing.T) {
	for _, minimal := range []bool{false, true} {
		c, _, control := algebraicCompareFixture(t)
		c.minMagnitude = 0
		a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, minimal)
		b := algebraicCompareRoot(t, c, []int32{-3, 0, 1}, 1, 0, 2, 0, minimal)
		control.checks = 0
		control.cooperations = 0
		got, e := c.Compare(&a, &b)
		serviceMust(t, e)
		if got != -1 || c.compareRefine != 1 || c.compareCheap != 1 || c.compareSturm != 0 {
			t.Fatal(minimal, got, c.compareRefine, c.compareCheap, c.compareSturm)
		}
		if minimal && control.checks != 1 || !minimal && control.checks != 0 || control.cooperations != control.checks {
			t.Fatal("exact source checkpoint branch", minimal, *control)
		}
		al, e := c.location(&a)
		serviceMust(t, e)
		ar, e := c.load(al)
		serviceMust(t, e)
		bl, e := c.location(&b)
		serviceMust(t, e)
		br, e := c.load(bl)
		serviceMust(t, e)
		separated, e := c.config.Binary.LessEqual(&ar.interval.upper, &br.interval.lower)
		serviceMust(t, e)
		if !separated {
			t.Fatal("comparison used stale pre-refinement bounds")
		}
		serviceMust(t, c.Del(&a))
		serviceMust(t, c.Del(&b))
	}
	c, _, control := algebraicCompareFixture(t)
	c.minMagnitude = -4
	a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, false)
	b := algebraicCompareRoot(t, c, []int32{-3, 0, 1}, 3, 1, 7, 2, false)
	got, e := c.Compare(&a, &b)
	serviceMust(t, e)
	if got != -1 || c.compareRefine != 3 || c.compareCheap != 1 || c.compareSturm != 0 {
		t.Fatal("refineN before interval test", got, c.compareRefine, c.compareCheap, c.compareSturm)
	}
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.Del(&b))
	if control.checks != 0 {
		t.Fatal("refineN checkpointed", *control)
	}
}

func TestAlgebraicCompareRefinementBasicShortCircuit(t *testing.T) {
	c, _, control := algebraicCompareFixture(t)
	c.minMagnitude = 0
	a := algebraicCompareRoot(t, c, []int32{15, -16, 4}, 1, 0, 2, 0, false)
	b := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, false)
	bl, e := c.location(&b)
	serviceMust(t, e)
	before, e := c.load(bl)
	serviceMust(t, e)
	got, e := c.Compare(&a, &b)
	serviceMust(t, e)
	if got != 1 {
		t.Fatal(got)
	}
	al, e := c.location(&a)
	serviceMust(t, e)
	if al == nil || al.kind != 0 {
		t.Fatal("rational midpoint did not publish basic")
	}
	bl, e = c.location(&b)
	serviceMust(t, e)
	after, e := c.load(bl)
	serviceMust(t, e)
	if before.interval != after.interval || c.compareRefine != 0 {
		t.Fatal("source || must not refine B after A becomes basic", before.interval, after.interval, c.compareRefine)
	}
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.Del(&b))
	if control.checks != 0 {
		t.Fatal("midpoint short circuit checkpointed", *control)
	}
}

func TestAlgebraicCompareSturmTarskiIndependentOrder(t *testing.T) {
	c, _, control := algebraicCompareFixture(t)
	c.minMagnitude = 1
	a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, false)
	b := algebraicCompareRoot(t, c, []int32{-3, 0, 1}, 1, 0, 2, 0, false)
	got, e := c.Compare(&a, &b)
	serviceMust(t, e)
	if got != -1 || c.compareSturm != 1 || c.compareRefine != 0 {
		t.Fatal(got, c.compareSturm, c.compareRefine)
	}
	got, e = c.Compare(&b, &a)
	serviceMust(t, e)
	if got != 1 || c.compareSturm != 2 {
		t.Fatal(got, c.compareSturm)
	}
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.Del(&b))
	if control.checks == 0 || control.cooperations != control.checks {
		t.Fatal("real Sturm polynomial resource calls", *control)
	}
}

func TestAlgebraicCompareBasicRepresentationBeforeLess(t *testing.T) {
	c, _, control := algebraicCompareFixture(t)
	a, b := Algebraic{}, Algebraic{}
	raw := Rational{num: NewInt(2), den: NewInt(4)}
	canonical := Rational{num: NewInt(1), den: NewInt(2)}
	serviceMust(t, c.SetRational(&a, &raw))
	serviceMust(t, c.SetRational(&b, &canonical))
	// Source eq is representation equality. Mathematically equal unreduced
	// fractions compare as +1 because source eq fails and source lt is false.
	got, e := c.Compare(&a, &b)
	serviceMust(t, e)
	if got != 1 {
		t.Fatal("eq-before-lt source representation", got)
	}
	got, e = c.Compare(&b, &a)
	serviceMust(t, e)
	if got != 1 {
		t.Fatal("reverse raw representation", got)
	}
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.Del(&b))
	if control.checks != 0 {
		t.Fatal("basic representation checkpointed", *control)
	}
}

func TestAlgebraicCompareCheckpointBeforeRefinement(t *testing.T) {
	c, _, control := algebraicCompareFixture(t)
	a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	b := algebraicCompareRoot(t, c, []int32{-3, 0, 1}, 1, 0, 2, 0, true)
	al, e := c.location(&a)
	serviceMust(t, e)
	beforeA, e := c.load(al)
	serviceMust(t, e)
	bl, e := c.location(&b)
	serviceMust(t, e)
	beforeB, e := c.load(bl)
	serviceMust(t, e)
	control.allow = false
	_, e = c.Compare(&a, &b)
	if !errors.Is(e, ErrUPolynomialCanceled) {
		t.Fatal(e)
	}
	if control.checks != 1 || control.cooperations != 0 || c.compareRefine != 0 {
		t.Fatal("source checkpoint precedes refine/cooperate", *control, c.compareRefine)
	}
	al, e = c.location(&a)
	serviceMust(t, e)
	afterA, e := c.load(al)
	serviceMust(t, e)
	bl, e = c.location(&b)
	serviceMust(t, e)
	afterB, e := c.load(bl)
	serviceMust(t, e)
	if beforeA != afterA || beforeB != afterB {
		t.Fatal("canceled comparison altered cell")
	}
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.Del(&b))
}

// Enumerate actual named owner fields, never infer ordinary roots by scanning
// the numeral directory. The physical Integer oracle also checks real buffers.
func algebraicComparisonAudit(t *testing.T, c *AlgebraicRootCore, values ...*Algebraic) {
	t.Helper()
	algebraicCellSourcePhysical(t, c, nil, nil, nil, values...)
}

func TestAlgebraicCompareSequenceReservationPrefixes(t *testing.T) {
	failures, successes := 0, 0
	for prefix := int64(0); prefix < 256; prefix++ {
		c, p, control := algebraicCompareFixture(t)
		c.minMagnitude = 1
		a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, false)
		b := algebraicCompareRoot(t, c, []int32{-3, 0, 1}, 1, 0, 2, 0, false)
		al, e := c.location(&a)
		serviceMust(t, e)
		ar, e := c.load(al)
		serviceMust(t, e)
		bl, e := c.location(&b)
		serviceMust(t, e)
		br, e := c.load(bl)
		serviceMust(t, e)
		budget := c.config.Rationals.budget
		budget.FailAfter(prefix)
		got, e := c.Compare(&a, &b)
		algebraicComparisonAudit(t, c, &a, &b)
		if c.compareSequence != nil || p.liveSequences != 0 {
			t.Fatal("ordinary refusal discarded sequence owner", prefix, c.compareSequence, p.liveSequences)
		}
		if e != nil {
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal(prefix, e)
			}
			failures++
		} else {
			if got != -1 {
				t.Fatal(prefix, got)
			}
			successes++
		}
		if c.compareSturm != 1 || c.compareCheap != 0 || c.compareRefine != 0 {
			t.Fatal("source expensive phase before reservation", prefix, c.compareSturm, c.compareCheap, c.compareRefine)
		}
		if control.cooperations > control.checks {
			t.Fatal(*control)
		}
		al, se := c.location(&a)
		serviceMust(t, se)
		aa, se := c.load(al)
		serviceMust(t, se)
		bl, se = c.location(&b)
		serviceMust(t, se)
		bb, se := c.load(bl)
		serviceMust(t, se)
		if aa != ar || bb != br {
			t.Fatal("source comparison modified input roots", prefix)
		}
		budget.FailAfter(-1)
		serviceMust(t, c.config.Rationals.z.CleanupPendingAllocation())
		serviceMust(t, c.Del(&a))
		serviceMust(t, c.Del(&b))
		serviceMust(t, c.Close())
		if e == nil {
			break
		}
	}
	if failures == 0 || successes != 1 {
		t.Fatal(failures, successes)
	}
}

func TestAlgebraicCompareSequenceStoppedDestructorRetainsOwner(t *testing.T) {
	c, p, control := algebraicCompareFixtureProfile(t, true)
	c.minMagnitude = 1
	z := c.config.Rationals.z
	var k, n Int
	serviceMust(t, z.SetLimbs(&k, []uint32{1, 0, 1}))
	serviceMust(t, z.Mul2k(&k, 1, &n))
	serviceMust(t, z.Neg(&n))
	serviceMust(t, z.Dec(&n))
	poly, e := p.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, p.SetInts(poly, []Int{n, NewInt(0), k}))
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	var lower, upper BinaryRational
	serviceMust(t, c.config.Binary.SetInt32(&lower, 1))
	serviceMust(t, c.config.Binary.SetInt32(&upper, 2))
	var a Algebraic
	serviceMust(t, c.SetRoot(&a, v, &lower, &upper, false))
	serviceMust(t, c.config.Binary.Del(&lower))
	serviceMust(t, c.config.Binary.Del(&upper))
	serviceMust(t, p.DelPolynomial(&poly))
	serviceMust(t, z.Del(&k))
	serviceMust(t, z.Del(&n))
	b := algebraicCompareRoot(t, c, []int32{-3, 0, 1}, 1, 0, 2, 0, false)
	var stopped numericCellRef
	var requested uint64
	// Explicit malformed ILP32 allocation metadata, not a valid source malloc
	// failure: invalidate ONLY the actual completed sequence copy at a reached
	// real multiplication checkpoint. The input root's different owner stays valid.
	hook := func() bool {
		control.checks++
		seq := c.compareSequence
		if seq == nil || seq.state.coefficients.size == 0 {
			t.Fatal("checkpoint before copied sequence")
		}
		value := &p.values(&seq.state.coefficients)[0]
		stopped = value.cell
		if stopped == 0 {
			t.Fatal("expected actual copied BIG coefficient")
		}
		cell := z.cell(stopped)
		requested = cell.backing.requested
		cell.backing.requested = uint64(math.MaxUint32) + 1
		z.store(stopped, cell)
		return false
	}
	p.config.Checkpoint = hook
	c.config.Checkpoint = hook
	_, e = c.Compare(&a, &b)
	if !errors.Is(e, ErrUPolynomialCanceled) || !errors.Is(e, arena.ErrBounds) {
		t.Fatal("original plus actual cleanup refusal", e)
	}
	if c.compareSequence == nil || p.liveSequences != 1 {
		t.Fatal("lost actual stopped sequence owner")
	}
	if control.checks != 1 || control.cooperations != 0 {
		t.Fatal("reached source checkpoint count", *control)
	}
	algebraicComparisonAudit(t, c, &a, &b)
	if cell := z.cell(stopped); cell.released {
		t.Fatal("malformed requested-size guard must precede physical release")
	}
	if e = c.CleanupRetained(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("cleanup bypassed stopped coefficient", e)
	}
	algebraicComparisonAudit(t, c, &a, &b)
	q := Rational{num: NewInt(1), den: NewInt(1)}
	got, e := c.CompareRational(&a, &q)
	serviceMust(t, e)
	if got != 1 {
		t.Fatal("pure query poisoned by owning sequence", got)
	}
	if _, e = c.Compare(&a, &b); !errors.Is(e, ErrAlgebraicPending) {
		t.Fatal("sequence scope reused while stopped", e)
	}
	// Restore only deliberately invalid layout bytes; no numerical fields,
	// allocation counters, source input or arithmetic scratch are repaired.
	cell := z.cell(stopped)
	cell.backing.requested = requested
	z.store(stopped, cell)
	serviceMust(t, c.CleanupRetained())
	if c.compareSequence != nil || p.liveSequences != 0 {
		t.Fatal("stopped sequence not cleaned")
	}
	algebraicComparisonAudit(t, c, &a, &b)
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.Del(&b))
	// This test's hook intentionally refused cooperation once; no hidden model.
	control.allow = false
}

func TestAlgebraicCompareCheapWarmAllocationFree(t *testing.T) {
	c, _, control := algebraicCompareFixture(t)
	a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 3, 1, false)
	b := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, -3, 1, -1, 0, false)
	got, e := c.Compare(&a, &b)
	serviceMust(t, e)
	if got != 1 {
		t.Fatal(got)
	}
	budget := c.config.Rationals.budget
	before := numericWarmProfileSnapshot(t, c.config.Rationals.z, budget)
	budget.FailAfter(0)
	allocations := testing.AllocsPerRun(100, func() {
		got, e := c.Compare(&a, &b)
		if e != nil || got != 1 {
			panic("warmed cheap comparison")
		}
	})
	algebraicComparisonAudit(t, c, &a, &b)
	numericWarmProfileAudit(t, c.config.Rationals.z, budget, before, allocations, false)
	if control.checks != 0 || control.cooperations != 0 {
		t.Fatal("cheap comparison reached resource hooks", *control)
	}
	budget.FailAfter(-1)
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.Del(&b))
}
