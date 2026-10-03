// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math/big"
	"reflect"
	"testing"
)

func algebraicBinaryOracle(t *testing.T, z *IntegerManager, b *BinaryRational) *big.Rat {
	t.Helper()
	n := roundPublicationDecode(t, z, &b.num)
	d := new(big.Int).Lsh(big.NewInt(1), uint(b.k))
	return new(big.Rat).SetFrac(n, d)
}
func TestAlgebraicIndependentRefinementAndRationalConversion(t *testing.T) {
	c, p := algebraicFixture(t)
	q := algebraicPolynomial(t, p, -2, 0, 1)
	v, e := p.wholeView(q)
	serviceMust(t, e)
	lo, hi := NewBinaryInt(1), NewBinaryInt(2)
	var a, b Algebraic
	serviceMust(t, c.SetRoot(&a, v, &lo, &hi, false))
	serviceMust(t, c.Copy(&b, &a))
	t.Cleanup(func() { serviceMust(t, c.Del(&b)); serviceMust(t, c.Del(&a)) })
	old := big.NewRat(1, 1)
	for i := 0; i < 12; i++ {
		serviceMust(t, c.Refine(&a))
		l, e := c.location(&a)
		serviceMust(t, e)
		r, e := c.load(l)
		serviceMust(t, e)
		lower, upper := algebraicBinaryOracle(t, c.config.Rationals.z, &r.interval.lower), algebraicBinaryOracle(t, c.config.Rationals.z, &r.interval.upper)
		two := big.NewRat(2, 1)
		if new(big.Rat).Mul(lower, lower).Cmp(two) >= 0 || new(big.Rat).Mul(upper, upper).Cmp(two) <= 0 {
			t.Fatal("independent sqrt2 bracket", lower, upper)
		}
		width := new(big.Rat).Sub(upper, lower)
		if width.Cmp(old) >= 0 {
			t.Fatal("source refinement failed to narrow")
		}
		old = width
	}
	eq, e := c.Compare(&a, &b)
	serviceMust(t, e)
	if eq != 0 {
		t.Fatal(eq)
	}
	rat, e := c.IsRational(&a)
	serviceMust(t, e)
	if rat {
		t.Fatal("nonminimal sqrt2 rational")
	}
	// Root of (2x-1)(x-2): exact rational remains encoded as a nonminimal
	// root until the real mutating predicate/refinement converts it.
	rp := algebraicPolynomial(t, p, 2, -5, 2)
	rv, e := p.wholeView(rp)
	serviceMust(t, e)
	lo = NewBinaryInt(0)
	hi = NewBinaryInt(1)
	var r Algebraic
	serviceMust(t, c.SetRoot(&r, rv, &lo, &hi, false))
	isRat, e := c.IsRational(&r)
	serviceMust(t, e)
	if !isRat {
		t.Fatal("rational root theorem")
	}
	rl, e := c.location(&r)
	serviceMust(t, e)
	if rl.kind != 0 {
		t.Fatal("predicate failed to publish basic conversion")
	}
	want := NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&want, 1, 2))
	d, e := c.CompareRational(&r, &want)
	serviceMust(t, e)
	if d != 0 {
		t.Fatal(d)
	}
	serviceMust(t, c.Del(&r))
	serviceMust(t, c.config.Rationals.Del(&want))
}
func TestAlgebraicPartialCopySourceDelGuard(t *testing.T) {
	found := false
	for fault := int64(0); fault < 16 && !found; fault++ {
		t.Run(big.NewInt(fault).String(), func(t *testing.T) {
			c, p := algebraicCellSourceFixture(t)
			poly, e := p.NewPolynomial()
			serviceMust(t, e)
			serviceMust(t, p.SetInts(poly, []Int{NewInt(-2), NewInt(0), NewInt(1)}))
			t.Cleanup(func() { serviceMust(t, p.DelPolynomial(&poly)) })
			v, e := p.wholeView(poly)
			serviceMust(t, e)
			lo, hi := NewBinaryInt(1), NewBinaryInt(2)
			var src, dst Algebraic
			serviceMust(t, c.SetRoot(&src, v, &lo, &hi, false))
			budget := c.config.Rationals.budget
			budget.FailAfter(fault)
			e = c.Copy(&dst, &src)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal(e)
			}
			if e != nil && dst.slot != 0 {
				l, e := c.location(&dst)
				serviceMust(t, e)
				r, e := c.load(l)
				serviceMust(t, e)
				if r.size > 0 && r.poly.slot != 0 {
					pd, e := p.descriptor(r.poly)
					serviceMust(t, e)
					if pd.state.backing == 0 {
						found = true
						algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, nil, nil, &src, &dst)
						before := budget.Used()
						if e = c.Del(&dst); !errors.Is(e, arena.ErrBounds) {
							t.Fatal("source null coefficient-array read must guard", e)
						}
						if budget.Used() != before {
							t.Fatal("guard changed real roots")
						}
						algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, nil, nil, &src, &dst)
					}
				}
			}
			budget.FailAfter(-1)
			if dst.slot != 0 {
				serviceMust(t, c.CleanupValue(&dst))
			}
			serviceMust(t, c.Del(&src))
			serviceMust(t, c.CleanupRetained())
		})
	}
	if !found {
		t.Fatal("did not reach actual published partial coefficient root")
	}
}
func TestAlgebraicSourceUnsignedMagnitude(t *testing.T) {
	c, _ := algebraicFixture(t)
	for _, tc := range []struct {
		l, u BinaryRational
		want int32
	}{{NewBinaryIntPower(3, 0x80000000), NewBinaryIntPower(7, 0x80000001), 2147483647}, {NewBinaryIntPower(-7, 0xffffffff), NewBinaryIntPower(-3, 0x80000000), -2}} {
		got, e := c.magnitude(&tc.l, &tc.u)
		serviceMust(t, e)
		if got != tc.want {
			t.Fatal(got, tc.want)
		}
	}
}

// All genuine reservation prefixes through first completion. The named census
// runs before fault disable/cleanup; the ordinary factor service must already
// have completed its lexical unwind or expose its retained scope explicitly.
func TestAlgebraicIsolationFailurePrefixes(t *testing.T) {
	success := false
	failures := 0
	for fault := int64(0); fault < 160 && !success; fault++ {
		t.Run(big.NewInt(fault).String(), func(t *testing.T) {
			c, p := algebraicFixture(t)
			poly := algebraicPolynomial(t, p, -2, 0, 1)
			v, e := p.wholeView(poly)
			serviceMust(t, e)
			roots, e := c.NewVector()
			serviceMust(t, e)
			b := c.config.Rationals.budget
			b.FailAfter(fault)
			e = c.IsolateRoots(v, roots)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("unexpected source error", e)
			}
			if c.config.Factors.scope != nil {
				t.Fatal("factor lexical root needs explicit nested census")
			}
			values := make([]*Algebraic, 0, roots.size)
			for i := uint32(0); i < roots.size; i++ {
				values = append(values, &roots.values[i])
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, nil, nil, values...)
			if e == nil {
				success = true
				if roots.Len() != 2 {
					t.Fatal("source sqrt2 count")
				}
			} else {
				failures++
			}
			b.FailAfter(-1)
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, roots.Close())
		})
	}
	if !success || failures == 0 {
		t.Fatal("full fault-prefix completion", failures, success)
	}
}
func TestAlgebraicSavedIntervalAndSquareFreePolicy(t *testing.T) {
	c, p := algebraicFixture(t)
	poly := algebraicPolynomial(t, p, -2, 0, 1)
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	lo, hi := NewBinaryInt(1), NewBinaryInt(2)
	var a Algebraic
	serviceMust(t, c.SetRoot(&a, v, &lo, &hi, false))
	c.minMagnitude = 32
	rat, e := c.IsRational(&a)
	serviceMust(t, e)
	if rat {
		t.Fatal("sqrt2")
	}
	l, e := c.location(&a)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	if algebraicBinaryOracle(t, c.config.Rationals.z, &r.interval.lower).Cmp(big.NewRat(1, 1)) != 0 || algebraicBinaryOracle(t, c.config.Rationals.z, &r.interval.upper).Cmp(big.NewRat(2, 1)) != 0 {
		t.Fatal("selected saved-interval restoration")
	}
	if r.flags&4 == 0 || c.predicate.active || c.localActive {
		t.Fatal("rational theorem result/cleanup")
	}
	serviceMust(t, c.Del(&a))
	c.factor = false
	roots, e := c.NewVector()
	serviceMust(t, e)
	serviceMust(t, c.IsolateRoots(v, roots))
	if roots.Len() != 2 {
		t.Fatal(roots.Len())
	}
	for i := uint32(0); i < roots.size; i++ {
		l, e := c.location(&roots.values[i])
		serviceMust(t, e)
		r, e := c.load(l)
		serviceMust(t, e)
		if r.flags&5 != 0 {
			t.Fatal("squarefree fallback falsely minimal")
		}
	}
	serviceMust(t, roots.Close())
}
func TestAlgebraicScalarCodecAndWarmCells(t *testing.T) {
	c, _ := algebraicCellSourceFixture(t)
	serviceMust(t, arena.ValidateRecordType(reflect.TypeOf(Algebraic{})))
	serviceMust(t, arena.ValidateRecordType(reflect.TypeOf(algebraicRecord{})))
	q := NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&q, 2, 3))
	var a Algebraic
	serviceMust(t, c.SetRational(&a, &q))
	serviceMust(t, c.Del(&a))
	before := c.config.Rationals.budget.Used()
	var problem error
	allocs := testing.AllocsPerRun(20, func() {
		if problem != nil {
			return
		}
		problem = c.SetRational(&a, &q)
		if problem == nil {
			problem = c.Del(&a)
		}
	})
	serviceMust(t, problem)
	if allocs != 0 || before != c.config.Rationals.budget.Used() {
		t.Fatal("warm source cell storage", allocs, before, c.config.Rationals.budget.Used())
	}
	serviceMust(t, c.config.Rationals.Del(&q))
}

func TestAlgebraicObservableCancellation(t *testing.T) {
	c, p, control := algebraicControlledFixture(t)
	q := algebraicPolynomial(t, p, -2, 0, 1)
	view, e := p.wholeView(q)
	serviceMust(t, e)
	lo, hi := NewBinaryInt(1), NewBinaryInt(2)
	var a Algebraic
	serviceMust(t, c.SetRoot(&a, view, &lo, &hi, false))
	t.Cleanup(func() { serviceMust(t, c.Del(&a)) })
	control.limit = control.visits
	before := control.cooperations
	if e = c.checkpoint(); !errors.Is(e, ErrUPolynomialCanceled) {
		t.Fatal("actual bounded checkpoint", e)
	}
	if control.refused != 1 || control.cooperations != before {
		t.Fatal("canceled checkpoint must not cooperate", control)
	}
	control.limit = control.visits + 1000000
	serviceMust(t, c.checkpoint())
	if control.cooperations == before {
		t.Fatal("resumed real source checkpoint missing")
	}
}
func TestAlgebraicPrivateMemberDestructionOrder(t *testing.T) {
	c, _ := algebraicFixture(t)
	z := c.config.Rationals.z
	serviceMust(t, c.ensureIsolation())
	var n Int
	serviceMust(t, z.SetUint64(&n, 1<<40))
	t.Cleanup(func() { serviceMust(t, z.Del(&n)) })
	b := NewBinaryInt(0)
	serviceMust(t, z.Copy(&b.num, &n))
	serviceMust(t, c.roots.PushCopy(&b))
	serviceMust(t, c.lowers.PushCopy(&b))
	serviceMust(t, c.uppers.PushCopy(&b))
	serviceMust(t, z.Del(&b.num))
	slots := []arena.Slot{numericSlot(c.roots.state.values[0].num.cell), numericSlot(c.lowers.state.values[0].num.cell), numericSlot(c.uppers.state.values[0].num.cell)}
	serviceMust(t, c.Close())
	// Reverse member destruction uppers, lowers, roots gives LIFO metadata
	// chain roots -> lowers -> uppers, independent of original slot values.
	head := z.cells.free
	for _, want := range slots {
		if head != want {
			t.Fatalf("reverse source members head=%d want=%d", head, want)
		}
		meta := z.cells.state(head)
		head = meta.next
	}
}

// A deliberately malformed requested extent exercises the real guarded cleanup
// continuation; it is not a fabricated valid allocation/free refusal.
func TestAlgebraicCloseStoppedSelectionBlocksOrdinaryUse(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	z := c.config.Rationals.z
	q := NewRational()
	serviceMust(t, z.SetUint64(&q.num, 1<<40))
	var a Algebraic
	serviceMust(t, c.SetTake(&a, &q))
	v, e := c.NewVector()
	serviceMust(t, e)
	serviceMust(t, v.pushTake(&a))
	c.rootSelection = v
	l, e := c.location(&v.values[0])
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	ref := r.value.num.cell
	cell := z.cell(ref)
	requested := cell.backing.requested
	cell.backing.requested = uint64(1) << 32
	z.store(ref, cell)
	if e = c.Close(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual cleanup guard", e)
	}
	if !c.closing || c.rootSelection != v || v.closed {
		t.Fatal("lost stopped owner state")
	}
	if e = c.SetTake(&a, &q); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("ordinary use after owning close", e)
	}
	algebraicCellSourcePhysical(t, c, nil, []*Int{&q.num, &q.den}, nil)
	cell = z.cell(ref)
	cell.backing.requested = requested
	z.store(ref, cell)
	serviceMust(t, c.Close())
}
