package polynomial

// Source: src/math/polynomial/polynomial.cpp:5751-5901; src/util/ref_vector.h.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestPolynomialPSCSameInputsResetSwapAndRepeatedOutput(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
	q := polynomialQueryTerms(t, m, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	S, e := NewPolynomialRefVector(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, S.Close()) })
	serviceMust(t, S.Push(p))
	serviceMust(t, S.Push(q))
	serviceMust(t, m.PSCChain(p, q, 1, S))
	n, e := S.Size()
	serviceMust(t, e)
	if n != 1 {
		t.Fatal(n)
	}
	first, e := S.Get(0)
	serviceMust(t, e)
	polynomialQueryExpect(t, m, first, map[[3]uint32]int64{{1, 0, 0}: 6, {2, 0, 0}: -1})
	serviceMust(t, m.PSCChain(q, p, 1, S))
	first, e = S.Get(0)
	serviceMust(t, e)
	polynomialQueryExpect(t, m, first, map[[3]uint32]int64{{1, 0, 0}: 6, {2, 0, 0}: -1})
	serviceMust(t, m.PSCChain(p, p, 1, S))
	n, e = S.Size()
	serviceMust(t, e)
	first, e = S.Get(0)
	serviceMust(t, e)
	if n != 1 || first != m.zero {
		t.Fatal("literal equal-input prem0 and empty-chain zero push", n, first)
	}
	polynomialPSCPhysicalVectors(t, m, nil, []*PolynomialRefVector{S}, []Polynomial{p, q}, nil)
}

func TestPolynomialPSCEveryCheckpointCancellationFrontier(t *testing.T) {
	for cancel := uint32(1); cancel <= 28; cancel++ {
		m := polynomialCellFixture(t)
		p := polynomialQueryTerms(t, m, []int32{1, 2, 3, 4}, [][]MonomialPower{{{0, 4}}, {{0, 2}}, {{0, 1}}, nil})
		q := polynomialQueryTerms(t, m, []int32{4, 4, 3}, [][]MonomialPower{{{0, 3}}, {{0, 1}}, nil})
		S, e := NewPolynomialRefVector(m)
		serviceMust(t, e)
		serviceMust(t, S.Push(m.unit))
		var visits, cooperate uint32
		m.config.Checkpoint = func() bool { visits++; return visits != cancel }
		m.config.Cooperation = func() error { cooperate++; return nil }
		e = m.PSCChain(p, q, 0, S)
		if !errors.Is(e, ErrUPolynomialCanceled) || visits != cancel || cooperate != cancel-1 {
			t.Fatal("source checkpoint->cooperation frontier", cancel, e, visits, cooperate)
		}
		polynomialPSCPhysicalVectors(t, m, nil, []*PolynomialRefVector{S}, []Polynomial{p, q}, nil)
		n, z := S.Size()
		serviceMust(t, z)
		values := [3]int64{16, 132, 17109}
		if n > 3 {
			t.Fatal(n)
		}
		for i := uint32(0); i < n; i++ {
			p, z := S.Get(i)
			serviceMust(t, z)
			polynomialQueryExpect(t, m, p, map[[3]uint32]int64{{}: values[i]})
		}
		serviceMust(t, m.CleanupPSC())
		serviceMust(t, S.Close())
	}
}

func TestPolynomialPSCWarmActualSourceProfile(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
	q := polynomialQueryTerms(t, m, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	S, e := NewPolynomialRefVector(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, S.Close()) })
	run := func() {
		if e := m.PSCChain(p, q, 1, S); e != nil {
			panic(e)
		}
	}
	for i := 0; i < 20; i++ {
		run()
	}
	before := numericWarmProfileSnapshot(t, m.integer, m.integer.Budget())
	mono := m.monomials.allocator.Stats()
	for i := 0; i < 20; i++ {
		run()
	}
	numericWarmRuntimeAudit(t, m.integer, m.integer.Budget(), before, mono.DirectObjects != 0)
	if m.monomials.allocator.Stats() != mono {
		t.Fatal("actual source allocator retained graph changed", mono, m.monomials.allocator.Stats())
	}
	polynomialPSCPhysicalVectors(t, m, nil, []*PolynomialRefVector{S}, []Polynomial{p, q}, nil)
}

func TestPolynomialPSCFullOwnerPendingLocalClosePrefix(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	b := &m.pscBuffer
	serviceMust(t, b.construct(m))
	serviceMust(t, b.push(m, m.unit))
	defer func() { serviceMust(t, b.finalize(m, true)) }()
	before := m.integer.Budget().Used()
	e := full.CloseChecked()
	if !errors.Is(e, ErrPolynomialPending) || full.closing || m.closing || m.integer.Budget().Used() != before {
		t.Errorf("actual H local preguard before caller/member mutation: %v", e)
	}
	polynomialImpPhysicalAudit(t, full)
	serviceMust(t, b.finalize(m, false))
}

func TestPolynomialPSCSourceAssertionBeforeOutputReset(t *testing.T) {
	polynomialThrowAssertionFixture(t)
	m := polynomialCellFixture(t)
	S, e := NewPolynomialRefVector(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, S.Close()) }()
	serviceMust(t, S.Push(m.unit))
	e = m.PSCChain(m.unit, m.unit, 0, S)
	n, z := S.Size()
	serviceMust(t, z)
	a, z := S.Get(0)
	serviceMust(t, z)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrPolynomialAssertion) || n != 1 || a != m.unit {
			t.Fatal("source SASSERT precedes reset", e, n, a)
		}
	} else {
		if e != nil || n != 1 || a != m.zero {
			t.Fatal("release literal constant prem0 and empty-chain zero", e, n, a)
		}
	}
	polynomialPSCPhysicalVectors(t, m, nil, []*PolynomialRefVector{S}, nil, nil)
}

func TestPolynomialPSCOutputColdGuardBeforeOperandRead(t *testing.T) {
	m := polynomialCellFixture(t)
	S, e := NewPolynomialRefVector(m)
	serviceMust(t, e)
	serviceMust(t, S.Close())
	unread := Polynomial{owner: m.origin, slot: ^uint32(0), generation: 1}
	if e = m.PSCChain(unread, unread, 0, S); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("actual output lifetime guard before degree reads", e)
	}
	other := polynomialCellFixture(t)
	T, e := NewPolynomialRefVector(other)
	serviceMust(t, e)
	defer func() { serviceMust(t, T.Close()) }()
	if e = m.PSCChain(unread, unread, 0, T); !errors.Is(e, ErrPolynomialOwner) {
		t.Fatal("actual output cohort guard before degree reads", e)
	}
}
