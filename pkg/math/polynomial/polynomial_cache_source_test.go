package polynomial

import (
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestPolynomialCacheFactorStoppedBodyCleanupAndRetry(t *testing.T) {
	stopped := false
	for ordinal := int64(0); ordinal < 64 && !stopped; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c, e := NewPolynomialCache(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, c.Close()) })
			out, e := NewPolynomialRefVector(m.core)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, out.Close()) })
			p := polynomialQueryTerms(t, m.core, []int32{1, 2, 1}, [][]MonomialPower{nil, {{0, 1}}, {{0, 2}}})
			_, e = c.MkUnique(p)
			serviceMust(t, e)
			b := m.core.integer.Budget()
			b.FailAfter(ordinal)
			body := c.Factor(p, out)
			b.FailAfter(-1)
			if body == nil || m.factorCoreInvocation == 0 {
				return
			}
			stopped = true
			if e := c.Factor(p, out); !errors.Is(e, ErrPolynomialPending) {
				t.Fatal("incomplete factor entry permitted replay", e)
			}
			serviceMust(t, c.CleanupFactor())
			if c.pendingFactors != nil {
				t.Fatal("stopped factorization retained temporary factors")
			}
			serviceMust(t, c.Reset())
			serviceMust(t, c.Factor(p, out))
		})
	}
	if !stopped {
		t.Fatal("no real factor reservation refusal reached factorization")
	}
}

func TestPolynomialCachePSCStoppedBodyCleanupAndRetry(t *testing.T) {
	stopped := false
	for ordinal := int64(0); ordinal < 96 && !stopped; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c, e := NewPolynomialCache(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, c.Close()) })
			out, e := NewPolynomialRefVector(m.core)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, out.Close()) })
			p := polynomialQueryTerms(t, m.core, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
			q := polynomialQueryTerms(t, m.core, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
			_, e = c.MkUnique(p)
			serviceMust(t, e)
			_, e = c.MkUnique(q)
			serviceMust(t, e)
			b := m.core.integer.Budget()
			b.FailAfter(ordinal)
			body := c.PSCChain(p, q, 1, out)
			b.FailAfter(-1)
			if body == nil || m.core.pscLocal.invocation == 0 {
				return
			}
			stopped = true
			serviceMust(t, c.CleanupPSC())
			if m.core.pscReceipts.factory != 0 {
				t.Fatal("completed PSC parent left an owning factory receipt")
			}
			if e = c.PSCChain(p, q, 1, out); !errors.Is(e, ErrPolynomialPending) {
				t.Fatal("incomplete PSC entry permitted replay", e)
			}
			serviceMust(t, out.Close())
			serviceMust(t, c.Reset())
			retry, e := NewPolynomialRefVector(m.core)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, retry.Close()) })
			serviceMust(t, c.PSCChain(p, q, 1, retry))
		})
	}
	if !stopped {
		t.Fatal("no real PSC reservation refusal reached the source scope")
	}
}

func TestPolynomialCachePSCResultReplacementStoppedDeletion(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.Close()) })
	p := polynomialQueryTerms(t, m.core, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
	q := polynomialQueryTerms(t, m.core, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	first, e := NewPolynomialRefVector(m.core)
	serviceMust(t, e)
	serviceMust(t, m.PSCChain(p, q, 1, first))
	n, e := first.Size()
	serviceMust(t, e)
	if n == 0 {
		t.Fatal("source PSC fixture produced no result")
	}
	canonical := make([]Polynomial, n)
	for i := uint32(0); i < n; i++ {
		value, e := first.Get(i)
		serviceMust(t, e)
		canonical[i], e = c.MkUnique(value)
		serviceMust(t, e)
	}
	serviceMust(t, first.Close())
	out, e := NewPolynomialRefVector(m.core)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	stop := errors.New("stop cached PSC result deletion")
	armed, fired := true, false
	h, e := NewPolynomialDeletionHandler(m.originalBudget, func(_ *PolynomialDeletionHandler, core *MultivariatePolynomialCore, value Polynomial) error {
		if armed && !core.pscLocal.active && value != canonical[0] && len(out.state.values) > 0 && out.state.values[0] == value {
			fired = true
			return stop
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, m.core.AddDeletionHandler(h))
	t.Cleanup(func() { serviceMust(t, m.core.RemoveDeletionHandler(h)); serviceMust(t, h.CloseChecked()) })
	e = c.PSCChain(p, q, 1, out)
	armed = false
	if !fired || !errors.Is(e, stop) {
		t.Fatal("published PSC result replacement did not reach deletion refusal", fired, e)
	}
	if !out.state.setActive || out.state.incoming != canonical[0] {
		t.Fatal("failed SET lost its acquired canonical reference")
	}
	info, e := m.core.Info(canonical[0])
	serviceMust(t, e)
	if info.RefCount != 2 {
		t.Fatal("failed SET did not retain its incoming reference", info.RefCount)
	}
	if len(c.psc) != 1 {
		t.Fatal("failed result replacement lost inserted cache entry")
	}
	serviceMust(t, c.CleanupPSC())
	serviceMust(t, out.Close())
	info, e = m.core.Info(canonical[0])
	serviceMust(t, e)
	if info.RefCount != 1 {
		t.Fatal("caller cleanup did not release incoming reference", info.RefCount)
	}
	serviceMust(t, c.Reset())
	retry, e := NewPolynomialRefVector(m.core)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, retry.Close()) })
	serviceMust(t, c.PSCChain(p, q, 1, retry))
}

func TestPolynomialCacheFactorTemporaryStoppedDeletion(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.Close()) })
	p := polynomialQueryTerms(t, m.core, []int32{1, 2, 1}, [][]MonomialPower{nil, {{0, 1}}, {{0, 2}}})
	canonicalInput, e := c.MkUnique(p)
	serviceMust(t, e)
	seed, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	serviceMust(t, m.Factor(p, seed, DefaultUPolynomialFactorParams()))
	n, e := seed.Distinct()
	serviceMust(t, e)
	if n != 1 {
		t.Fatal("source square distinct factor count", n)
	}
	seedValue, e := seed.factor(0)
	serviceMust(t, e)
	if seedValue == p {
		t.Fatal("factorization reused input identity; no temporary factor to release")
	}
	canonicalFactor, e := c.MkUnique(seedValue)
	serviceMust(t, e)
	serviceMust(t, seed.CloseChecked())
	out, e := NewPolynomialRefVector(m.core)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	stop := errors.New("stop temporary factor deletion")
	armed, fired := true, false
	h, e := NewPolynomialDeletionHandler(m.originalBudget, func(_ *PolynomialDeletionHandler, core *MultivariatePolynomialCore, value Polynomial) error {
		if !armed || m.factorCoreExecuting() || value == canonicalFactor || c.factors[canonicalInput] == nil {
			return nil
		}
		same, err := core.Equal(value, canonicalFactor)
		if err != nil {
			return err
		}
		if same {
			fired = true
			return stop
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, m.core.AddDeletionHandler(h))
	t.Cleanup(func() { serviceMust(t, m.core.RemoveDeletionHandler(h)); serviceMust(t, h.CloseChecked()) })
	e = c.Factor(p, out)
	armed = false
	if !fired || !errors.Is(e, stop) || c.pendingFactors == nil {
		t.Fatal("temporary factor destructor did not retain its owner", fired, e, c.pendingFactors)
	}
	if c.pendingFactors.closed || len(c.pendingFactors.phases) != 1 || c.pendingFactors.phases[0] != factorLastDrop {
		t.Fatal("stopped temporary factor lost consumed final-decrement receipt")
	}
	if e := c.Factor(p, out); !errors.Is(e, ErrPolynomialPending) {
		t.Fatal("stopped temporary factor permitted cache reentry", e)
	}
	serviceMust(t, c.CleanupFactor())
	if c.pendingFactors != nil {
		t.Fatal("temporary factor owner survived owning cleanup")
	}
	info, e := m.core.Info(canonicalFactor)
	serviceMust(t, e)
	if info.RefCount != 2 {
		t.Fatal("cleanup changed cached and caller factor references", info.RefCount)
	}
	serviceMust(t, c.Reset())
	serviceMust(t, c.Factor(p, out))
}

func TestPolynomialCacheConstructorBudgetAndRefusal(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	b := m.core.integer.Budget()
	before := b.Used()
	b.FailAfter(0)
	_, e := NewPolynomialCache(m)
	b.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != before {
		t.Fatal("failed cache construction changed owner budget", e, b.Used(), before)
	}
	c, e := NewPolynomialCache(m)
	serviceMust(t, e)
	if b.Used() <= before {
		t.Fatal("cache owner was not charged")
	}
	serviceMust(t, c.Close())
	if b.Used() != before {
		t.Fatal("cache owner charge survived close", b.Used(), before)
	}
}

func TestPolynomialCacheCanonicalizationAndReset(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.Close()) })
	p := polynomialQueryTerms(t, m.core, []int32{3, 5, 7}, [][]MonomialPower{{{0, 2}}, {{1, 1}}, nil})
	q := polynomialQueryTerms(t, m.core, []int32{7, 3, 5}, [][]MonomialPower{nil, {{0, 2}}, {{1, 1}}})
	r := polynomialQueryTerms(t, m.core, []int32{7, 3, 6}, [][]MonomialPower{nil, {{0, 2}}, {{1, 1}}})
	first, e := c.MkUnique(q)
	serviceMust(t, e)
	if first != q {
		t.Fatal("first equal polynomial was not retained")
	}
	canonical, e := c.MkUnique(p)
	serviceMust(t, e)
	if canonical != q {
		t.Fatal("equal polynomial did not reuse first cache entry")
	}
	distinct, e := c.MkUnique(r)
	serviceMust(t, e)
	if distinct != r {
		t.Fatal("different polynomial reused an equal bucket entry")
	}
	for _, item := range []Polynomial{q, r} {
		info, e := m.core.Info(item)
		serviceMust(t, e)
		if info.RefCount != 2 {
			t.Fatal("cache did not own exactly one reference", info)
		}
	}
	again, e := c.MkUnique(q)
	serviceMust(t, e)
	if again != q {
		t.Fatal("cached ID did not return itself")
	}
	info, e := m.core.Info(q)
	serviceMust(t, e)
	if info.RefCount != 2 {
		t.Fatal("repeated lookup added a reference", info)
	}
	serviceMust(t, c.Reset())
	for _, item := range []Polynomial{q, r} {
		info, e := m.core.Info(item)
		serviceMust(t, e)
		if info.RefCount != 1 {
			t.Fatal("reset did not release the cached reference", info)
		}
	}
	first, e = c.MkUnique(p)
	serviceMust(t, e)
	if first != p {
		t.Fatal("reset retained the former representative")
	}
	canonical, e = c.MkUnique(q)
	serviceMust(t, e)
	if canonical != p {
		t.Fatal("new representative did not win after reset")
	}
	serviceMust(t, c.Close())
	if _, e := c.MkUnique(p); e != arena.ErrClosed {
		t.Fatal("closed cache accepted lookup", e)
	}
}

func TestPolynomialCachePSCChainCanonicalKeyAndReplay(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.Close()) })
	out, e := NewPolynomialRefVector(m.core)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	p := polynomialQueryTerms(t, m.core, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
	q := polynomialQueryTerms(t, m.core, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	pCopy := polynomialQueryTerms(t, m.core, []int32{2, 1, 1, 1}, [][]MonomialPower{nil, {{1, 1}}, {{0, 1}, {1, 1}}, {{0, 1}, {1, 2}}})
	qCopy := polynomialQueryTerms(t, m.core, []int32{3, 1}, [][]MonomialPower{nil, {{0, 1}, {1, 1}}})
	serviceMust(t, c.PSCChain(p, q, 1, out))
	n, e := out.Size()
	serviceMust(t, e)
	if n == 0 {
		t.Fatal("source PSC fixture produced no result")
	}
	first := make([]Polynomial, n)
	for i := range first {
		first[i], e = out.Get(uint32(i))
		serviceMust(t, e)
	}
	invocation := m.core.pscLocal.invocation
	serviceMust(t, out.Reset())
	for _, value := range first {
		if _, e := m.core.Info(value); e != nil {
			t.Fatal("cache lost its result reference", e)
		}
	}
	serviceMust(t, c.PSCChain(pCopy, qCopy, 1, out))
	if m.core.pscLocal.invocation != invocation {
		t.Fatal("equal canonical pair recomputed PSC chain")
	}
	got, e := out.Size()
	serviceMust(t, e)
	if got != n {
		t.Fatal("cached PSC result length", got, n)
	}
	for i, want := range first {
		value, e := out.Get(uint32(i))
		serviceMust(t, e)
		if value != want {
			t.Fatal("cached PSC result changed identity", i)
		}
	}
	serviceMust(t, c.PSCChain(q, p, 1, out))
	if m.core.pscLocal.invocation == invocation {
		t.Fatal("reversed inputs reused an ordered PSC cache key")
	}
	invocation = m.core.pscLocal.invocation
	serviceMust(t, c.Reset())
	serviceMust(t, c.PSCChain(p, q, 1, out))
	if m.core.pscLocal.invocation == invocation {
		t.Fatal("reset retained PSC memoized entry")
	}
}

func TestPolynomialCacheFactorDistinctReplayAndReset(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.Close()) })
	out, e := NewPolynomialRefVector(m.core)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	p := polynomialQueryTerms(t, m.core, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	pCopy := polynomialQueryTerms(t, m.core, []int32{1, 1}, [][]MonomialPower{nil, {{0, 1}}})
	serviceMust(t, c.Factor(p, out))
	n, e := out.Size()
	serviceMust(t, e)
	if n != 1 {
		t.Fatal("source linear polynomial distinct factor count", n)
	}
	first, e := out.Get(0)
	serviceMust(t, e)
	invocation := m.factorCoreInvocation
	serviceMust(t, out.Reset())
	if _, e := m.core.Info(first); e != nil {
		t.Fatal("factor cache lost its result owner", e)
	}
	serviceMust(t, c.Factor(pCopy, out))
	if m.factorCoreInvocation != invocation {
		t.Fatal("equal canonical input recomputed factorization")
	}
	got, e := out.Get(0)
	serviceMust(t, e)
	if got != first {
		t.Fatal("factor cache replay changed canonical identity")
	}
	constant := polynomialQueryTerms(t, m.core, []int32{3}, [][]MonomialPower{nil})
	serviceMust(t, c.Factor(constant, out))
	n, e = out.Size()
	serviceMust(t, e)
	if n != 0 {
		t.Fatal("constant polynomial has distinct factors", n)
	}
	serviceMust(t, c.Reset())
	serviceMust(t, c.Factor(p, out))
	if m.factorCoreInvocation == invocation {
		t.Fatal("reset retained factor memoized entry")
	}
}

func TestPolynomialCacheFactorConstructorRefusalRetainsInsertedKey(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.Close()) })
	out, e := NewPolynomialRefVector(m.core)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	p := polynomialQueryTerms(t, m.core, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	_, e = c.MkUnique(p)
	serviceMust(t, e)
	b := m.core.integer.Budget()
	b.FailAfter(0)
	e = c.Factor(p, out)
	b.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("factor temporary constructor did not refuse", e)
	}
	if len(c.factors) != 1 || c.factors[p].complete {
		t.Fatal("source cache entry not inserted before factor constructor")
	}
	if e = c.Factor(p, out); !errors.Is(e, ErrPolynomialPending) {
		t.Fatal("incomplete factor entry replayed", e)
	}
	serviceMust(t, c.Reset())
	serviceMust(t, c.Factor(p, out))
}
