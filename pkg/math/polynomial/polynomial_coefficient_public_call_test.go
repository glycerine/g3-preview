package polynomial

import (
	"errors"
	"reflect"
	"testing"
)

func TestPolynomialCoefficientOwningCallTwoBuffersAndAlias(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{3, 5, 7}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{1, 1}}, nil})
	lc, err := NewPolynomialRef(c)
	serviceMust(t, err)
	defer func() { serviceMust(t, lc.Close()) }()
	reduct, err := NewPolynomialRef(c)
	serviceMust(t, err)
	defer func() { serviceMust(t, reduct.Close()) }()
	serviceMust(t, lc.Assign(p))
	var call PolynomialCoefficientCall
	serviceMust(t, c.CoeffWithReductInto(p, 1, 2, lc, reduct, &call))
	result, err := lc.Get()
	serviceMust(t, err)
	polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{1, 0, 0}: 3})
	rest, err := reduct.Get()
	serviceMust(t, err)
	polynomialQueryExpect(t, c, rest, map[[3]uint32]int64{{0, 1, 0}: 5, {}: 7})
	serviceMust(t, call.CleanupRetained())
	if call.active || c.coefficientInvocation.bufferActive || c.factoryInvocation.active {
		t.Fatal("completed owning boundary retained child")
	}
	// Actual input aliases the reduct output on this second call.
	serviceMust(t, c.CoeffWithReductInto(rest, 1, 1, lc, reduct, &call))
	result, err = lc.Get()
	serviceMust(t, err)
	polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{}: 5})
	rest, err = reduct.Get()
	serviceMust(t, err)
	polynomialQueryExpect(t, c, rest, map[[3]uint32]int64{{}: 7})
	serviceMust(t, call.CleanupRetained())
	copied := call
	if !errors.Is(copied.CleanupRetained(), ErrPolynomialOwner) {
		t.Fatal("copied public call acquired authority")
	}
}
func TestPolynomialCoefficientOwningCallAllocationPrefixes(t *testing.T) {
	success, failures := false, 0
	for stop := int64(0); stop < 128 && !success; stop++ {
		m := polynomialPSCFullFixture(t)
		c := m.core
		p := polynomialQueryTerms(t, c, []int32{2, 3, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{1, 2}}, nil})
		lc, err := NewPolynomialRef(c)
		serviceMust(t, err)
		reduct, err := NewPolynomialRef(c)
		serviceMust(t, err)
		var call PolynomialCoefficientCall
		c.integer.Budget().FailAfter(stop)
		err = c.CoeffWithReductInto(p, 0, 1, lc, reduct, &call)
		c.integer.Budget().FailAfter(-1)
		if err == nil {
			success = true
		} else {
			failures++
			if !numericPhysicalReservationRefusal(err) {
				t.Fatal(stop, err)
			}
		}
		serviceMust(t, call.FinishSource())
		serviceMust(t, reduct.Close())
		serviceMust(t, lc.Close())
		serviceMust(t, call.CleanupRetained())
		polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
		if call.active || c.coefficientInvocation.active || c.coefficientInvocation.bufferActive {
			t.Fatal("accepted call retained coefficient children", stop)
		}
	}
	if !success || failures < 2 {
		t.Fatal("cold two-buffer horizon not covered", success, failures)
	}
	t.Log("consecutive failures", failures)
}
func TestPolynomialCoefficientOwningCallPreservesOlderInvocation(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	lc, err := NewPolynomialRef(c)
	serviceMust(t, err)
	defer func() { serviceMust(t, lc.Close()) }()
	reduct, err := NewPolynomialRef(c)
	serviceMust(t, err)
	defer func() { serviceMust(t, reduct.Close()) }()
	var old polynomialCoefficientInvocationReceipt
	var appendReceipt polynomialCheapAppendInvocationReceipt
	var factory polynomialFactoryInvocationReceipt
	value, err := c.coeffCoreWithReceipts(p, 0, 1, nil, &old, &appendReceipt, &factory)
	serviceMust(t, err)
	before, fb, ab := c.coefficientInvocation, c.factoryInvocation, c.cheap.appendScope
	var rejected PolynomialCoefficientCall
	err = c.CoeffWithReductInto(p, 0, 1, lc, reduct, &rejected)
	if !errors.Is(err, ErrPolynomialPending) {
		t.Fatal("older untransferred source factory not guarded", err)
	}
	serviceMust(t, rejected.CleanupRetained())
	if c.coefficientInvocation != before || c.factoryInvocation != fb || !reflect.DeepEqual(c.cheap.appendScope, ab) {
		t.Fatal("refused public call adopted older source invocation")
	}
	serviceMust(t, c.IncRef(value))
	serviceMust(t, c.transferFactoryResult(&factory, value))
	serviceMust(t, c.cleanupCoefficientInvocation(&old, true))
	serviceMust(t, c.DecRef(value))
}

func TestPolynomialReferenceSourceFinalizeDoesNotRetryStoppedAssignment(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	old, err := c.MkVariable(0, 5)
	serviceMust(t, err)
	incoming := polynomialQueryTerms(t, c, []int32{2}, [][]MonomialPower{nil})
	ref, err := NewPolynomialRef(c)
	serviceMust(t, err)
	serviceMust(t, ref.Assign(old))
	stop := errors.New("stop actual old reference deletion")
	calls := 0
	h, err := NewPolynomialDeletionHandler(c.integer.Budget(), func(_ *PolynomialDeletionHandler, _ *MultivariatePolynomialCore, p Polynomial) error {
		if p == old {
			calls++
			return stop
		}
		return nil
	})
	serviceMust(t, err)
	serviceMust(t, c.AddDeletionHandler(h))
	t.Cleanup(func() { serviceMust(t, c.RemoveDeletionHandler(h)); serviceMust(t, h.CloseChecked()) })
	err = ref.Assign(incoming)
	if !errors.Is(err, stop) || !ref.root.decremented || !ref.root.assigning || ref.root.incoming != incoming {
		t.Fatal("actual stopped old DEC not established", err)
	}
	before := ref.root
	complete, err := ref.SourceFinalize()
	if complete || !errors.Is(err, stop) || ref.root != before || calls != 1 {
		t.Fatal("source destructor retried owning assignment", complete, err, calls)
	}
	info, err := c.Info(incoming)
	serviceMust(t, err)
	if info.RefCount != 2 {
		t.Fatal("lexical destructor dropped unpublished incoming lease", info.RefCount)
	}
	serviceMust(t, ref.Close())
	info, err = c.Info(incoming)
	serviceMust(t, err)
	if info.RefCount != 1 || calls != 1 {
		t.Fatal("owning close replayed old DEC or lost incoming", info.RefCount, calls)
	}
}
