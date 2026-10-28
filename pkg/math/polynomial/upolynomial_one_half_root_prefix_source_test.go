package polynomial

// Source: src/math/polynomial/upolynomial.cpp:1298-1321.

import (
	"fmt"
	"math"
	"reflect"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestUPolynomialHasOneHalfRootEveryActualReservationPrefix(t *testing.T) {
	success, acquired := false, false
	for prefix := int64(0); prefix < 64 && !success; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, p := algebraicFixture(t)
			z := c.config.Rationals.Integers()
			poly := algebraicPolynomial(t, p, 1, math.MaxInt32, 1)
			constant := algebraicPolynomial(t, p, -7)
			v, e := p.wholeView(poly)
			serviceMust(t, e)
			cv, e := p.wholeView(constant)
			serviceMust(t, e)

			baseline := z.Budget().Used() - integerActualOwnedBytes(z) - p.oneHalfRootHistoryBytes
			z.Budget().FailAfter(prefix)
			result, body := p.HasOneHalfRoot(v)
			q := &p.oneHalfRoot
			r := p.oneHalfRootReceipt()
			context := q.context()
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal(body)
			}
			extra := []*unumeral.Int{&q.values[0], &q.values[1]}
			algebraicCellSourcePhysical(t, c, []UPolynomial{poly, constant}, extra, nil)
			if want := baseline + integerActualOwnedBytes(z) + p.oneHalfRootHistoryBytes; z.Budget().Used() != want {
				t.Fatal("before-lift cold graph", z.Budget().Used(), want)
			}
			if q.numeric.Cell.Object != (umemory.ObjectRef{}) || polynomialNegNumericPending(&q.numeric) {
				acquired = true
			}
			if q.history != nil {
				copiedOwner := *p
				copiedOwner.self = &copiedOwner
				copiedOwner.oneHalfRoot.owner = &copiedOwner
				copiedReceipt := copiedOwner.oneHalfRootReceipt()
				if e := copiedOwner.cleanupOneHalfRootWithReceipt(&copiedReceipt, true); e != ErrUPolynomialOwner {
					t.Fatal("copied owner obtained exact allocating scope cleanup", e)
				}
			}
			before := *q
			z.Budget().FailAfter(0)
			if got, e := p.HasOneHalfRoot(UPolynomialView{}); got != true || e != nil {
				t.Fatal("erased zero", got, e)
			}
			if got, e := p.HasOneHalfRoot(cv); got != false || e != nil {
				t.Fatal("erased constant", got, e)
			}
			if !reflect.DeepEqual(before, *q) {
				t.Fatal("erased calls changed actual older invocation")
			}
			z.Budget().FailAfter(-1)
			serviceMust(t, p.cleanupOneHalfRootWithReceipt(&r, false))
			serviceMust(t, p.cleanupOneHalfRootWithReceipt(&r, true))
			if q.history != nil {
				old := *q.history
				// A real later inline two-coefficient body reuses the actual embedded frame.
				linear := algebraicPolynomial(t, p, -1, 2)
				lv, e := p.wholeView(linear)
				serviceMust(t, e)
				z.Budget().FailAfter(0)
				if yes, e := p.HasOneHalfRoot(lv); !yes || e != nil {
					t.Fatal("later inline scaled Horner", yes, e)
				}
				z.Budget().FailAfter(-1)
				if *r.history != old {
					t.Fatal("later body changed charged first diagnostic/child tuples")
				}
				current := p.oneHalfRoot
				serviceMust(t, p.cleanupOneHalfRootWithReceipt(&r, true))
				if !reflect.DeepEqual(current, p.oneHalfRoot) {
					t.Fatal("old ACK touched later invocation")
				}
				setOneHalfRootCaller(z, p)
				if (context.roots(z)) != nil {
					t.Fatal("saved context retargeted actual reused member")
				}
				setOneHalfRootCaller(z, nil)
			}
			if body == nil {
				success = true
				if result {
					t.Fatal("independent nonroot scaled Horner", result)
				}
			}
		})
	}
	if !success || !acquired {
		t.Fatal("actual campaign did not reach success/pending leaf", success, acquired)
	}
}

func TestUPolynomialHasOneHalfRootIndependentExactCases(t *testing.T) {
	_, p := algebraicFixture(t)
	z := p.config.Integer
	for _, tc := range []struct {
		a    []int32
		want bool
	}{{nil, true}, {[]int32{9}, false}, {[]int32{-1, 2}, true}, {[]int32{1, -4, 4}, true}, {[]int32{1, -3, 2}, true}, {[]int32{-2, 0, 1}, false}, {[]int32{1, 0, 4}, false}} {
		poly := algebraicPolynomial(t, p, tc.a...)
		v, e := p.wholeView(poly)
		serviceMust(t, e)
		z.Budget().FailAfter(0)
		yes, e := p.HasOneHalfRoot(v)
		z.Budget().FailAfter(-1)
		if e != nil || yes != tc.want {
			t.Fatal(tc, yes, e)
		}
		serviceMust(t, p.CleanupOneHalfRoot())
	}
}

func TestUPolynomialHasOneHalfRootFieldPNActualPrefixes(t *testing.T) {
	success, reached := false, false
	for prefix := int64(0); prefix < 128 && !success; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, p := algebraicFixture(t)
			z := p.config.Integer
			serviceMust(t, p.modular.SetZPUint64(17))
			poly, e := p.NewPolynomial()
			serviceMust(t, e)
			values := make([]unumeral.Int, 40)
			for i := range values {
				values[i] = unumeral.NewInt(1)
			}
			serviceMust(t, p.SetInts(poly, values))
			v, e := p.wholeView(poly)
			serviceMust(t, e)
			baseline := z.Budget().Used() - integerActualOwnedBytes(z) - p.oneHalfRootHistoryBytes
			z.Budget().FailAfter(prefix)
			got, body := p.HasOneHalfRoot(v)
			q := &p.oneHalfRoot
			r := p.oneHalfRootReceipt()
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal(body)
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, []*unumeral.Int{&q.values[0], &q.values[1]}, nil)
			if want := baseline + integerActualOwnedBytes(z) + p.oneHalfRootHistoryBytes; want != z.Budget().Used() {
				t.Fatal("actual field cold beforelift", want, z.Budget().Used())
			}
			if q.normalize.Manager != nil {
				reached = true
			}
			z.Budget().FailAfter(-1)
			serviceMust(t, p.cleanupOneHalfRootWithReceipt(&r, false))
			serviceMust(t, p.DelPolynomial(&poly))
			serviceMust(t, p.cleanupOneHalfRootWithReceipt(&r, true))
			if body == nil {
				success = true
				if !got {
					t.Fatal("independent40-term scaled sum modulo17 should vanish")
				}
			}
		})
	}
	if !success || !reached {
		t.Fatal("actual field PN/success horizon absent", success, reached)
	}
}
