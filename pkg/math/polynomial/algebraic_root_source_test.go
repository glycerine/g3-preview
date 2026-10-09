// Selected registered algebraic.cpp inputs, through actual resource services.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/cooperate"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

type algebraicSourceCohort struct {
	service                         *util.RationalService
	polynomial                      *UPolynomialManager
	core                            *AlgebraicRootCore
	binary                          *util.BinaryRationalManager
	budget                          *arena.Budget
	limit                           *control.Reslimit
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
	s, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Cells: arena.Config{PageSize: 8, Budget: b}, Words: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 8, Budget: b}}}})
	algebraicSourceMust(t, e)
	limit, e := control.NewReslimit(b)
	algebraicSourceMust(t, e)
	cooperation, e := cooperate.NewCooperation(b)
	algebraicSourceMust(t, e)
	f := &algebraicSourceCohort{service: s, budget: b, limit: limit}
	p, e := NewUPolynomialManagerFromRationalService(s, UPolynomialConfig{Checkpoint: func() bool { f.polynomialTicks++; return limit.Inc() }, Cooperation: func() error { return cooperation.Cooperate("upolynomial") }})
	algebraicSourceMust(t, e)
	f.polynomial = p
	primes, e := primegen.NewGenerator(b)
	algebraicSourceMust(t, e)
	factors, e := NewUPolynomialFactorizationService(p, UPolynomialFactorizationConfig{Primes: primes})
	algebraicSourceMust(t, e)
	binary, e := util.NewBinaryRationalManager(p.IntegerManager(), util.BinaryRationalConfig{Diagnostics: s})
	algebraicSourceMust(t, e)
	f.binary = binary
	intervals, e := util.NewBinaryIntervalManager(binary)
	algebraicSourceMust(t, e)
	allocator, e := util.NewSmallAllocator(util.SmallAllocatorConfig{Runtime: s.RuntimeOwner(), Profile: util.SmallAllocatorProfile{PointerBits: s.RuntimeOwner().SizeTBits(), Debug: polynomialSourceDebug}})
	algebraicSourceMust(t, e)
	core, e := numericRationalService_NewAlgebraicRootCore(s, AlgebraicRootConfig{Polynomials: p, Factors: factors, Binary: binary, Intervals: intervals, Allocator: allocator, Checkpoint: func() bool { f.algebraicTicks++; return limit.Inc() }, Cooperation: func() error { return cooperation.Cooperate("algebraic") }})
	algebraicSourceMust(t, e)
	f.core = core
	t.Cleanup(func() {
		b.FailAfter(-1)
		for _, close := range []func() error{core.Close, allocator.Close, func() error { intervals.Close(); return nil }, binary.CloseChecked, factors.Close, primes.Close, p.Close, cooperation.Close, s.Close} {
			if e := close(); e != nil {
				t.Error(e)
			}
		}
		limit.Close()
		if b.Used() != 0 {
			t.Error("remaining actual owner bytes", b.Used())
		}
	})
	return f
}
func algebraicSourceView(t *testing.T, p *UPolynomialManager, v UPolynomial) UPolynomialView {
	t.Helper()
	n, e := p.Size(v)
	algebraicSourceMust(t, e)
	view, e := p.View(v, 0, n)
	algebraicSourceMust(t, e)
	return view
}
func TestAlgebraicRegisteredUnivariateRootCounts(t *testing.T) {
	f := algebraicSourceFixture(t)
	// algebraic.cpp:57,65,91,107. The third row is independently expanded from
	// (x^5-x-1)*(x-1)*(x-2). Full tst1 arithmetic/printers remain separate.
	cases := []struct {
		coeff []int32
		count uint32
	}{{[]int32{-2, 3}, 1}, {[]int32{-2, 0, 1}, 2}, {[]int32{-2, 1, 2, -1, 0, 2, -3, 1}, 3}, {[]int32{6, 0, -5, 0, 1}, 4}}
	for _, x := range cases {
		p := upolySourceNew(t, f.polynomial, x.coeff)
		roots, e := (f.core).NewVector()
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
	if f.polynomialTicks == 0 || f.limit.Count() != f.polynomialTicks+f.algebraicTicks {
		t.Fatal("actual resource consumer", f.limit.Count(), f.polynomialTicks, f.algebraicTicks)
	}
}
func TestAlgebraicRegisteredWilkinsonTwentyIntegerRoots(t *testing.T) {
	f := algebraicSourceFixture(t)
	// algebraic.cpp:224-248: build the same twenty factors using actual dense
	// polynomial multiplication, then retain the count and all twenty IsInt ENSUREs.
	p := upolySourceNew(t, f.polynomial, []int32{-1, 1})
	factor := upolySourceNew(t, f.polynomial, []int32{0, 1})
	for i := int32(2); i <= 20; i++ {
		z := util.NewInt(-i)
		algebraicSourceMust(t, f.polynomial.SetCoefficient(factor, 0, &z))
		algebraicSourceMust(t, f.polynomial.Mul(p, factor, p))
	}
	roots, e := (f.core).NewVector()
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
	if f.limit.Count() == 0 || f.limit.Count() != f.polynomialTicks+f.algebraicTicks {
		t.Fatal("actual Wilkinson resource calls")
	}
}
func TestAlgebraicRealResourceCancellationBeforeMinimalRefine(t *testing.T) {
	f := algebraicSourceFixture(t)
	p := upolySourceNew(t, f.polynomial, []int32{-2, 0, 1})
	q := upolySourceNew(t, f.polynomial, []int32{-3, 0, 1})
	lo, hi := util.NewBinaryInt(1), util.NewBinaryInt(2)
	var a, b Algebraic
	algebraicSourceMust(t, f.core.SetRoot(&a, algebraicSourceView(t, f.polynomial, p), &lo, &hi, true))
	algebraicSourceMust(t, f.core.SetRoot(&b, algebraicSourceView(t, f.polynomial, q), &lo, &hi, true))
	t.Cleanup(func() { algebraicSourceMust(t, f.core.Del(&a)); algebraicSourceMust(t, f.core.Del(&b)) })
	before := f.limit.Count()
	ticks := f.algebraicTicks
	f.limit.Cancel()
	_, e := f.core.Compare(&a, &b)
	if e == nil {
		t.Fatal("actual canceled source checkpoint accepted")
	}
	if f.limit.Count() != before+1 || f.algebraicTicks != ticks+1 {
		t.Fatal("minimal comparison cancellation order", before, f.limit.Count(), ticks, f.algebraicTicks)
	}
	f.limit.ResetCancel()
	cmp, e := f.core.Compare(&a, &b)
	algebraicSourceMust(t, e)
	if cmp != -1 {
		t.Fatal("sqrt2/sqrt3 after actual cancellation", cmp)
	}
}
