package polynomial

// Source: algebraic_numbers.cpp:541-563,1015; upolynomial.cpp:962-981.

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicSquareFreeDefaultChildrenReservationClosure(t *testing.T) {
	succeeded, retained, sourceStop := false, false, false
	for prefix := int64(0); prefix < 8192 && !succeeded; prefix++ {
		if !t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, pm, _ := algebraicCompareFixtureProfile(t, true)
			c.factor = false
			a := algebraicTinyBinaryBoundsRoot(t, c)
			var out Algebraic
			z := c.config.Rationals.Integers()
			z.Budget().FailAfter(prefix)
			body := c.Mul(&a, &a, &out)
			z.Budget().FailAfter(-1)
			if body == nil {
				succeeded = true
			} else if !numericPhysicalReservationRefusal(body) {
				t.Fatalf("left actual allocator: %v sqf body=%v source=%v", body, pm.squareFreeCall.bodyCause, pm.squareFreeCall.sourceStop)
			}
			child := pm.squareFreeCallReceiptPending(&c.binary.squareFree)
			if child {
				retained = true
				if body == nil {
					t.Fatal("successful square-free left pending children")
				}
				if pm.squareFreeCall.sourceStop != nil {
					sourceStop = true
				}
				if e := c.Del(&a); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("pending square-free child admitted caller deletion", e)
				}
			}
			raw := append([]unumeral.IntegerRawInvocationReceipt(nil), pm.squareFreeCall.raw...)
			divides := append([]unumeral.IntegerDividesNumeralInvocationReceipt(nil), pm.squareFreeCall.divides...)
			if e := c.CleanupRetained(); e != nil {
				t.Fatalf("owning cleanup: %v body: %v squarefree source: %v", e, body, pm.squareFreeCall.sourceStop)
			}
			if c.binary.active || pm.squareFreeCallReceiptPending(&c.binary.squareFree) {
				t.Fatal("selector retained accepted square-free children")
			}
			for i := range raw {
				if z.MatchesRawInvocation(&raw[i]) {
					t.Fatal("actual square-free RAW child survived acknowledgement")
				}
			}
			for i := range divides {
				if z.MatchesDividesNumeralInvocation(&divides[i]) {
					t.Fatal("actual square-free divisibility child survived acknowledgement")
				}
			}
			if unumeral.NumericAllocationPending(&pm.squareFreeCall.numeric) || pm.modular.MatchesPNormalizeInvocation(&pm.squareFreeCall.pn) || pm.defaultNormalizationReceiptPending(&pm.squareFreeCall.normalization) {
				t.Fatal("actual square-free allocation or normalization child survived acknowledgement")
			}
			algebraicExpectNoPendingAllocation(t, z)
			if succeeded {
				want := unumeral.NewRational()
				serviceMust(t, c.config.Rationals.SetString(&want, "1/170141183460469231731687303715884105728"))
				cmp, e := c.CompareRational(&out, &want)
				serviceMust(t, e)
				if cmp != 0 {
					t.Fatal("independent exact square", cmp)
				}
				serviceMust(t, c.config.Rationals.Del(&want))
			}
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&a))
		}) {
			return
		}
	}
	if !succeeded || !retained || !sourceStop {
		t.Fatal("square-free sweep missed success, actual retained child or source stop", succeeded, retained, sourceStop)
	}
}

func TestAlgebraicSquareFreeActualTransportBorrow(t *testing.T) {
	var c *AlgebraicRootCore
	var pm *UPolynomialManager
	var callerNumeral *unumeral.Int
	var out UPolynomial
	armed, calls := false, 0
	transport := func(f umemory.Failure) error {
		if !armed {
			return f
		}
		calls++
		if !pm.squareFreeCall.executing {
			t.Error("transport left accepted square-free execution")
		}
		z := c.config.Rationals.Integers()
		before := *callerNumeral
		used, physical := z.Budget().Used(), z.RuntimeOwner().Stats()
		for _, call := range []func() error{
			func() error { return pm.DelPolynomial(&out) }, pm.Close, pm.SourceFinalize,
			func() error { return pm.cleanupSquareFreeCall(nil) }, z.CloseChecked,
			func() error { return z.Del(callerNumeral) },
			func() error { return pm.modular.SetZ() },
			func() error { return pm.modular.SetZP(nil) },
			func() error {
				return pm.modular.WithAllocationContext(unumeral.NumericAllocationContext{}.WithCaller(&pm.squareFreeCall.hooks)).SetZ()
			},
		} {
			if e := call(); !errors.Is(e, ErrUPolynomialCleanupPending) && !errors.Is(e, unumeral.ErrNumericPending) && !errors.Is(e, unumeral.ErrNumericOwner) {
				t.Error("callback acquired borrowed owner", e)
			}
		}
		if before != *callerNumeral || used != z.Budget().Used() || physical != z.RuntimeOwner().Stats() {
			t.Error("refused transport changed numeral or physical graph")
		}
		return f
	}
	var rt *umemory.Runtime
	c, pm, rt = algebraicNegTLSFixture(t, transport)
	a := algebraicTinyBinaryBoundsRoot(t, c)
	l, e := c.location(&a)
	serviceMust(t, e)
	record, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&record)
	serviceMust(t, e)
	callerNumeral, e = pm.read(v, 2)
	serviceMust(t, e)
	out, e = pm.NewPolynomial()
	serviceMust(t, e)
	held, e := rt.Allocate(uint64(int64(100000) - rt.State().LocalSize - 12))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	body := pm.squareFreeViewWithReceipt(v, out, nil)
	armed = false
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, rt.Deallocate(&held))
	if body == nil || calls == 0 || !pm.squareFreeCall.pending() {
		t.Fatal("actual square-free transport not retained", body, calls)
	}
	serviceMust(t, pm.cleanupSquareFreeCall(nil))
	serviceMust(t, pm.DelPolynomial(&out))
	serviceMust(t, c.Del(&a))
}

func TestAlgebraicSquareFreeOldReceiptCannotAbortReplacement(t *testing.T) {
	reached := false
	for prefix := int64(0); prefix < 1024 && !reached; prefix++ {
		if !t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, pm, _ := algebraicCompareFixtureProfile(t, true)
			a := algebraicTinyBinaryBoundsRoot(t, c)
			l, e := c.location(&a)
			serviceMust(t, e)
			record, e := c.load(l)
			serviceMust(t, e)
			v, e := c.sourcePolynomial(&record)
			serviceMust(t, e)
			out, e := pm.NewPolynomial()
			serviceMust(t, e)
			var old, next upolySquareFreeCallReceipt
			serviceMust(t, pm.squareFreeViewWithReceipt(v, out, &old))
			z := c.config.Rationals.Integers()
			z.Budget().FailAfter(prefix)
			body := pm.squareFreeViewWithReceipt(v, out, &next)
			z.Budget().FailAfter(-1)
			if pm.squareFreeCallReceiptPending(&next) {
				reached = true
				if body == nil || old.invocation == next.invocation {
					t.Fatal("replacement did not issue a distinct stopped invocation", body)
				}
				before := pm.squareFreeCall
				used, physical := z.Budget().Used(), z.RuntimeOwner().Stats()
				serviceMust(t, pm.cleanupSquareFreeCall(&old))
				if !reflect.DeepEqual(before, pm.squareFreeCall) || used != z.Budget().Used() || physical != z.RuntimeOwner().Stats() {
					t.Fatal("old square-free receipt acquired replacement children")
				}
			}
			serviceMust(t, pm.cleanupSquareFreeCall(&next))
			serviceMust(t, pm.DelPolynomial(&out))
			serviceMust(t, c.Del(&a))
		}) {
			return
		}
	}
	if !reached {
		t.Fatal("no replacement square-free invocation retained")
	}
}

func TestAlgebraicSquareFreeCooperationBorrow(t *testing.T) {
	c, pm, _ := algebraicCompareFixtureProfile(t, true)
	a := algebraicTinyBinaryBoundsRoot(t, c)
	l, e := c.location(&a)
	serviceMust(t, e)
	record, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&record)
	serviceMust(t, e)
	out, e := pm.NewPolynomial()
	serviceMust(t, e)
	z := c.config.Rationals.Integers()
	var owned unumeral.Int
	serviceMust(t, z.Copy(&owned, &z.Two64))
	calls := 0
	original := pm.config.Cooperation
	pm.config.Cooperation = func() error {
		if pm.squareFreeCall.executing {
			calls++
			if pm.callbackDepth == 0 {
				t.Error("cooperation did not enter source callback span")
			}
			before := owned
			used, physical := z.Budget().Used(), z.RuntimeOwner().Stats()
			if e := z.Del(&owned); !errors.Is(e, unumeral.ErrNumericPending) {
				t.Error("cooperation released borrowed numerical storage", e)
			}
			if e := pm.SourceFinalize(); !errors.Is(e, ErrUPolynomialCleanupPending) {
				t.Error("cooperation finalized executing source", e)
			}
			if e := pm.CleanupSquareFree(); !errors.Is(e, ErrUPolynomialCleanupPending) {
				t.Error("cooperation acknowledged executing child", e)
			}
			if owned != before || used != z.Budget().Used() || physical != z.RuntimeOwner().Stats() {
				t.Error("cooperation refusals changed source state")
			}
		}
		return original()
	}
	serviceMust(t, pm.squareFreeViewWithReceipt(v, out, nil))
	if calls == 0 {
		t.Fatal("no actual square-free cooperation callback")
	}
	serviceMust(t, pm.CleanupSquareFree())
	serviceMust(t, z.Del(&owned))
	serviceMust(t, pm.DelPolynomial(&out))
	serviceMust(t, c.Del(&a))
}

func TestAlgebraicIsolationOwnsStoppedSquareFreeChild(t *testing.T) {
	succeeded, retained, numeric := false, false, false
	for prefix := int64(0); prefix < 1024 && !succeeded; prefix++ {
		if !t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, pm, _ := algebraicCompareFixtureProfile(t, true)
			c.factor = false
			a := algebraicTinyBinaryBoundsRoot(t, c)
			l, e := c.location(&a)
			serviceMust(t, e)
			record, e := c.load(l)
			serviceMust(t, e)
			v, e := c.sourcePolynomial(&record)
			serviceMust(t, e)
			roots, e := c.NewVector()
			serviceMust(t, e)
			z := c.config.Rationals.Integers()
			z.Budget().FailAfter(prefix)
			body := c.IsolateRoots(v, roots)
			z.Budget().FailAfter(-1)
			child := pm.squareFreeCallReceiptPending(&c.isolationSquareFree)
			if unumeral.NumericAllocationPending(&c.isolationNumeric.roots) {
				numeric = true
				if c.isolationNumeric.owner != c || c.isolationNumeric.hooks.Policy != &c.isolationNumeric {
					t.Fatal("stopped isolation allocation lost concrete caller identity")
				}
			}
			if body == nil {
				succeeded = true
			}
			if child {
				retained = true
				if body == nil {
					t.Fatal("isolation succeeded with a pending square-free child")
				}
				before := a
				if e := c.Del(&a); !errors.Is(e, ErrAlgebraicPending) || a != before {
					t.Fatal("stopped isolation child admitted caller deletion", e)
				}
			}
			if e := c.CleanupRetained(); e != nil {
				t.Fatalf("isolation caller cleanup: %v operation: %v", e, body)
			}
			if pm.squareFreeCallReceiptPending(&c.isolationSquareFree) {
				t.Fatal("isolation retained accepted square-free child")
			}
			algebraicExpectNoPendingAllocation(t, z)
			if c.isolationNumeric.active || unumeral.NumericAllocationPending(&c.isolationNumeric.roots) {
				t.Fatal("isolation caller cleanup retained its numeric root")
			}
			if succeeded && len(roots.values) != 2 {
				t.Fatal("quadratic isolation lost real roots", len(roots.values))
			}
			serviceMust(t, roots.Close())
			serviceMust(t, c.Del(&a))
		}) {
			return
		}
	}
	if !succeeded || !retained || !numeric {
		t.Fatal("isolation did not reach success and retained numeric/square-free children", succeeded, numeric, retained)
	}
}

func TestAlgebraicIsolationFactorCooperationProtectsCore(t *testing.T) {
	for _, factor := range []bool{false, true} {
		t.Run(fmt.Sprint(factor), func(t *testing.T) {
			c, pm, _ := algebraicCompareFixtureProfile(t, true)
			c.factor = factor
			a := algebraicTinyBinaryBoundsRoot(t, c)
			l, e := c.location(&a)
			serviceMust(t, e)
			record, e := c.load(l)
			serviceMust(t, e)
			v, e := c.sourcePolynomial(&record)
			serviceMust(t, e)
			roots, e := c.NewVector()
			serviceMust(t, e)
			z := c.config.Rationals.Integers()
			calls := 0
			original := pm.config.Cooperation
			pm.config.Cooperation = func() error {
				if c.factorExecuting {
					calls++
					if pm.callbackDepth == 0 || !c.univariateIsolationRunning {
						t.Error("factor cooperation lost its source callback span")
					}
					before, callers := a, c.callers
					used, physical := z.Budget().Used(), z.RuntimeOwner().Stats()
					for _, call := range []func() error{c.Close, c.CleanupRetained, func() error { return c.Del(&a) }, func() error { return c.Neg(&a) }} {
						if e := call(); !errors.Is(e, ErrAlgebraicPending) {
							t.Error("cooperation acquired algebraic owner", e)
						}
					}
					if a != before || c.callers != callers || used != z.Budget().Used() || physical != z.RuntimeOwner().Stats() {
						t.Error("refused algebraic callback changed source state")
					}
				}
				return original()
			}
			serviceMust(t, c.IsolateRoots(v, roots))
			if calls == 0 || len(roots.values) != 2 {
				t.Fatal("isolation did not reach actual square-free cooperation", calls, len(roots.values))
			}
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, roots.Close())
			serviceMust(t, c.Del(&a))
		})
	}
}
