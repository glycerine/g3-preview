// Licensed under the MIT license.
// Selected registered algebraic.cpp inputs, through actual resource services.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/mpnum"
	"testing"
)

type algebraicSourceCohort struct {
	service                         *mpnum.RationalService
	polynomial                      *mpnum.UPolynomialManager
	core                            *mpnum.AlgebraicRootCore
	binary                          *mpnum.BinaryRationalManager
	budget                          *arena.Budget
	limit                           *utilReslimit
	algebraicTicks, polynomialTicks uint64
}

func algebraicSourceMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func algebraicSourceFixture(t *testing.T) *algebraicSourceCohort {
	t.Helper()
	b := arena.NewBudget(1 << 27)
	s, e := mpnum.NewRationalService(mpnum.RationalServiceConfig{Integers: mpnum.IntegerConfig{Cells: arena.Config{PageSize: 8, Budget: b}, Words: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 8, Budget: b}}}})
	algebraicSourceMust(t, e)
	limit, e := newUtilReslimit(b)
	algebraicSourceMust(t, e)
	cooperation, e := newUtilCooperation(b)
	algebraicSourceMust(t, e)
	f := &algebraicSourceCohort{service: s, budget: b, limit: limit}
	p, e := s.NewUPolynomialManager(mpnum.UPolynomialConfig{Checkpoint: func() bool { f.polynomialTicks++; return limit.inc() }, Cooperation: func() error { return cooperation.Cooperate("upolynomial") }})
	algebraicSourceMust(t, e)
	f.polynomial = p
	primes, e := mpnum.NewPrimeGenerator(b)
	algebraicSourceMust(t, e)
	factors, e := mpnum.NewUPolynomialFactorizationService(p, mpnum.UPolynomialFactorizationConfig{Primes: primes})
	algebraicSourceMust(t, e)
	binary, e := mpnum.NewBinaryRationalManager(p.IntegerManager(), mpnum.BinaryRationalConfig{Diagnostics: s})
	algebraicSourceMust(t, e)
	f.binary = binary
	intervals, e := mpnum.NewBinaryIntervalManager(binary)
	algebraicSourceMust(t, e)
	allocator, e := memory.NewSmallAllocator(memory.SmallAllocatorConfig{Runtime: s.RuntimeOwner(), Profile: memory.SmallAllocatorProfile{PointerBits: s.RuntimeOwner().SizeTBits(), Debug: satStorageSourceDebug}})
	algebraicSourceMust(t, e)
	core, e := s.NewAlgebraicRootCore(mpnum.AlgebraicRootConfig{Polynomials: p, Factors: factors, Binary: binary, Intervals: intervals, Allocator: allocator, Checkpoint: func() bool { f.algebraicTicks++; return limit.inc() }, Cooperation: func() error { return cooperation.Cooperate("algebraic") }})
	algebraicSourceMust(t, e)
	f.core = core
	t.Cleanup(func() {
		b.FailAfter(-1)
		for _, close := range []func() error{core.Close, allocator.Close, func() error { intervals.Close(); return nil }, binary.CloseChecked, factors.Close, primes.Close, p.Close, cooperation.Close, s.Close} {
			if e := close(); e != nil {
				t.Error(e)
			}
		}
		limit.close()
		if b.Used() != 0 {
			t.Error("remaining actual owner bytes", b.Used())
		}
	})
	return f
}
func algebraicSourceView(t *testing.T, p *mpnum.UPolynomialManager, v mpnum.UPolynomial) mpnum.UPolynomialView {
	t.Helper()
	n, e := p.Size(v)
	algebraicSourceMust(t, e)
	view, e := p.View(v, 0, n)
	algebraicSourceMust(t, e)
	return view
}
func TestAlgebraicRegisteredUnivariateRootCounts(t *testing.T) {
	f := algebraicSourceFixture(t)
	// algebraic.cpp57,65,91,107. The third row is independently expanded from
	// (x^5-x-1)*(x-1)*(x-2). Full tst1 arithmetic/printers remain separate.
	cases := []struct {
		coeff []int32
		count uint32
	}{{[]int32{-2, 3}, 1}, {[]int32{-2, 0, 1}, 2}, {[]int32{-2, 1, 2, -1, 0, 2, -3, 1}, 3}, {[]int32{6, 0, -5, 0, 1}, 4}}
	for _, x := range cases {
		p := upolySourceNew(t, f.polynomial, x.coeff)
		roots, e := f.core.NewVector()
		algebraicSourceMust(t, e)
		algebraicSourceMust(t, f.core.IsolateRoots(algebraicSourceView(t, f.polynomial, p), roots))
		if roots.Len() != x.count {
			t.Fatalf("%v: roots %d want %d", x.coeff, roots.Len(), x.count)
		}
		for i := uint32(1); i < roots.Len(); i++ {
			cmp, e := roots.Compare(i-1, i)
			algebraicSourceMust(t, e)
			if cmp >= 0 {
				t.Fatal("actual root order", i, cmp)
			}
		}
		algebraicSourceMust(t, roots.Close())
	}
	if f.polynomialTicks == 0 || f.limit.count() != f.polynomialTicks+f.algebraicTicks {
		t.Fatal("actual resource consumer", f.limit.count(), f.polynomialTicks, f.algebraicTicks)
	}
}
func TestAlgebraicRegisteredWilkinsonTwentyIntegerRoots(t *testing.T) {
	f := algebraicSourceFixture(t)
	// algebraic.cpp224-248: build the same twenty factors using actual dense
	// polynomial multiplication, then retain the count and all twenty IsInt ENSUREs.
	p := upolySourceNew(t, f.polynomial, []int32{-1, 1})
	factor := upolySourceNew(t, f.polynomial, []int32{0, 1})
	for i := int32(2); i <= 20; i++ {
		z := mpnum.NewInt(-i)
		algebraicSourceMust(t, f.polynomial.SetCoefficient(factor, 0, &z))
		algebraicSourceMust(t, f.polynomial.Mul(p, factor, p))
	}
	roots, e := f.core.NewVector()
	algebraicSourceMust(t, e)
	algebraicSourceMust(t, f.core.IsolateRoots(algebraicSourceView(t, f.polynomial, p), roots))
	if roots.Len() != 20 {
		t.Fatal("Wilkinson root count", roots.Len())
	}
	for i := uint32(0); i < 20; i++ {
		integer, e := roots.IsInt(i)
		algebraicSourceMust(t, e)
		if !integer {
			t.Fatal("Wilkinson integer root", i)
		}
	}
	algebraicSourceMust(t, roots.Close())
	if f.limit.count() == 0 || f.limit.count() != f.polynomialTicks+f.algebraicTicks {
		t.Fatal("actual Wilkinson resource calls")
	}
}
func TestAlgebraicRealResourceCancellationBeforeMinimalRefine(t *testing.T) {
	f := algebraicSourceFixture(t)
	p := upolySourceNew(t, f.polynomial, []int32{-2, 0, 1})
	q := upolySourceNew(t, f.polynomial, []int32{-3, 0, 1})
	lo, hi := mpnum.NewBinaryInt(1), mpnum.NewBinaryInt(2)
	var a, b mpnum.Algebraic
	algebraicSourceMust(t, f.core.SetRoot(&a, algebraicSourceView(t, f.polynomial, p), &lo, &hi, true))
	algebraicSourceMust(t, f.core.SetRoot(&b, algebraicSourceView(t, f.polynomial, q), &lo, &hi, true))
	t.Cleanup(func() { algebraicSourceMust(t, f.core.Del(&a)); algebraicSourceMust(t, f.core.Del(&b)) })
	before := f.limit.count()
	ticks := f.algebraicTicks
	f.limit.cancel()
	_, e := f.core.Compare(&a, &b)
	if e == nil {
		t.Fatal("actual canceled source checkpoint accepted")
	}
	if f.limit.count() != before+1 || f.algebraicTicks != ticks+1 {
		t.Fatal("minimal comparison cancellation order", before, f.limit.count(), ticks, f.algebraicTicks)
	}
	f.limit.resetCancel()
	cmp, e := f.core.Compare(&a, &b)
	algebraicSourceMust(t, e)
	if cmp != -1 {
		t.Fatal("sqrt2/sqrt3 after actual cancellation", cmp)
	}
}
