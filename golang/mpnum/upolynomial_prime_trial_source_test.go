// Source: src/math/polynomial/upolynomial_factorization.cpp:1097-1145;
// src/math/polynomial/upolynomial_factorization_int.h:42-52.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

func TestUPolynomialPrimeTrialLiteralRoutes(t *testing.T) {
	for _, tc := range []struct {
		prime   int32
		route   uint8
		factors uint32
	}{{2, 0, 0}, {3, 2, 0}, {7, 1, 2}} {
		q, s := mignotteDirectFixture(t)
		m := s.manager
		c, e := s.newSquareFreeScope()
		serviceMust(t, e)
		c.constructed = 1
		serviceMust(t, m.set(&c.fpp, []Int{NewInt(-2), NewInt(0), NewInt(1)}))
		c.constructed = 2
		serviceMust(t, m.modular.SetInt32(&c.p, tc.prime))
		c.zp, e = NewRelatedUPolynomialManager(m)
		serviceMust(t, e)
		c.constructed = 3
		serviceMust(t, c.zp.SetZP(&c.p))
		c.best, e = c.zp.NewFactors()
		serviceMust(t, e)
		c.constructed = 4
		c.constructed = 5
		serviceMust(t, m.modular.SetInt32(&c.bestP, 2))
		c.constructed = 6
		serviceMust(t, c.ds.init(c.best))
		out, e := m.NewFactors()
		serviceMust(t, e)
		trials := uint32(0)
		route, e := s.factorPrimeTrial(c.zp, &c.fpp, out, c.best, 1, &c.p, &c.bestP, &c.ds, &trials)
		serviceMust(t, e)
		if route != tc.route || c.best.count != tc.factors {
			t.Fatal("literal GF trial route/count", tc, route, c.best.count)
		}
		if tc.route == 2 {
			if out.count != 1 || out.factors[0].size != 3 {
				t.Fatal("irreducible source push", out.Stats())
			}
			for i, want := range []Int{NewInt(-2), NewInt(0), NewInt(1)} {
				got, e := m.read(m.stateView(&out.factors[0]), uint32(i))
				serviceMust(t, e)
				if *got != want {
					t.Fatal("original Z factor altered", i, *got, want)
				}
			}
		}
		if tc.route == 1 {
			seenPlus, seenMinus := false, false
			for i := uint32(0); i < c.best.count; i++ {
				v := &c.best.factors[i]
				if v.size != 2 {
					t.Fatal("independent GF7 linear factors", v.size)
				}
				a, x := c.zp.read(c.zp.stateView(v), 0)
				serviceMust(t, x)
				b, x := c.zp.read(c.zp.stateView(v), 1)
				serviceMust(t, x)
				if *b != NewInt(1) {
					t.Fatal("independent monic linear factor", *b)
				}
				switch *a {
				case NewInt(3):
					seenPlus = true
				case NewInt(-3):
					seenMinus = true
				default:
					t.Fatal("independent roots +/-3", *a)
				}
			}
			if !seenPlus || !seenMinus {
				t.Fatal("missing independent GF7 factor pair")
			}
		}
		if tc.route == 1 && (trials != 1 || c.bestP != NewInt(7)) {
			t.Fatal("actual trial/swap saved prime", trials, c.bestP)
		}
		if tc.route == 0 && trials != 0 {
			t.Fatal("continue updated trial", trials)
		}
		serviceMust(t, out.Close())
		primeTrialPhysicalAudit(t, q, s)
		serviceMust(t, s.CleanupRetained())
	}
}

// Every owner is supplied by an actual constructor/root in this test. The
// store directory is counted once even when the actual related core borrows it.
func primeTrialPhysicalAudit(t *testing.T, q *RationalManager, s *UPolynomialFactorizationService) {
	primeTrialPhysicalAuditExtra(t, q, s)
}
func primeTrialPhysicalAuditExtra(t *testing.T, q *RationalManager, s *UPolynomialFactorizationService, extra ...*Int) {
	primeTrialPhysicalAuditOwners(t, q, s, nil, extra...)
}
func primeTrialPhysicalAuditOwners(t *testing.T, q *RationalManager, s *UPolynomialFactorizationService, externalFactors []*UPolynomialFactors, extra ...*Int) {
	primeTrialPhysicalAuditAll(t, q, s, externalFactors, nil, extra...)
}
func primeTrialPhysicalAuditExternal(t *testing.T, q *RationalManager, s *UPolynomialFactorizationService, external []memory.Block, extra ...*Int) {
	primeTrialPhysicalAuditAll(t, q, s, nil, external, extra...)
}
func primeTrialPhysicalAuditAll(t *testing.T, q *RationalManager, s *UPolynomialFactorizationService, externalFactors []*UPolynomialFactors, external []memory.Block, extra ...*Int) {
	roots := append(extra, []*Int{&q.nTmp, &q.addTmp1, &q.addTmp2, &q.addMulTmp.num, &q.addMulTmp.den, &q.ltTmp1.num, &q.ltTmp1.den, &q.ltTmp2.num, &q.ltTmp2.den}...)
	n := q.headerBytes + integerActualOwnedBytes(q.z) + s.header + s.config.Primes.Stats().TotalBytes
	c := s.scope
	managers := []*UPolynomialManager{s.manager}
	if c != nil {
		n += c.header + uint64(cap(c.ds.words))*4 + uint64(cap(c.primeTrial.degrees.words))*4
		managers = append(managers, c.zp, c.zpe)
		roots = append(roots, &c.p, &c.bestP, &c.gcdTmp, &c.leading, &c.trialContent, &c.mignotte.norm, &c.mignotte.bound, &c.mignotte.tmp)
	}
	stores := map[*UPolynomialCoefficientStore]bool{}
	appendState := func(m *UPolynomialManager, v *upolyVectorState) {
		if v.backing != 0 {
			values := m.values(v)
			for i := uint32(0); i < v.initialized; i++ {
				roots = append(roots, &values[i])
			}
		}
	}
	for _, m := range managers {
		if m == nil {
			continue
		}
		n += m.header + m.modular.header + uint64(cap(m.pages))*uint64(reflect.TypeOf((*upolyDescriptorPage)(nil)).Size()) + uint64(len(m.pages))*uint64(reflect.TypeOf(upolyDescriptorPage{}).Size()) + uint64(cap(m.bindings))*uint64(reflect.TypeOf(upolyStoreBinding{}).Size())
		roots = append(roots, &m.modular.p, &m.modular.lower, &m.modular.upper, &m.modular.invTmp1, &m.modular.invTmp2, &m.modular.invTmp3, &m.modular.divTmp)
		for _, v := range []*upolyVectorState{&m.db, &m.dbab1, &m.dbab2, &m.tr, &m.push, &m.basic, &m.div1, &m.div2, &m.exactDiv, &m.gcd1, &m.gcd2, &m.cra, &m.mgcd[0], &m.mgcd[1], &m.mgcd[2], &m.mgcd[3], &m.mgcd[4], &m.mgcd[5], &m.sqf1, &m.sqf2, &m.pw} {
			appendState(m, v)
		}
		if !stores[m.store] {
			stores[m.store] = true
			n += m.store.header + uint64(cap(m.store.backings))*uint64(reflect.TypeOf(upolyBacking{}).Size())
			for _, b := range m.store.backings {
				if b.live {
					n += uint64(cap(b.values)) * uint64(reflect.TypeOf(Int{}).Size())
				}
			}
		}
	}
	if c != nil {
		roots = append(roots, &c.primeTrial.transport)
		for _, v := range []*upolyVectorState{&c.fpp, &c.trial, &c.quotient, &c.primeTrial.fzp} {
			appendState(s.manager, v)
		}
		for _, fs := range append(externalFactors, c.best, c.lifted, c.primeTrial.current) {
			if fs == nil {
				continue
			}
			n += fs.header + uint64(cap(fs.degrees.values))*4 + uint64(cap(fs.factors))*uint64(reflect.TypeOf(upolyVectorState{}).Size())
			roots = append(roots, &fs.constant)
			for i := uint32(0); i < fs.count; i++ {
				appendState(fs.manager, &fs.factors[i])
			}
		}
	}
	mignotteNamedIntegerPhysicalAudit(t, q.z, external, roots...)
	if n != q.budget.Used() {
		t.Fatal("actual named prime trial cold ownership", n, q.budget.Used())
	}
}

// The helper has a RAW p_i, not a scoped numeral. A failed SET or vector
// publication must retain its actual transport until explicit owning abort.
func TestUPolynomialPrimeTrialToZPRawEveryPrefix(t *testing.T) {
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		q, s := mignotteDirectFixture(t)
		m := s.manager
		c, e := s.newSquareFreeScope()
		serviceMust(t, e)
		c.constructed = 1
		serviceMust(t, m.set(&c.fpp, []Int{NewInt(1), q.z.two64, NewInt(1)}))
		c.constructed = 2
		serviceMust(t, m.modular.SetInt32(&c.p, 7))
		c.zp, e = NewRelatedUPolynomialManager(m)
		serviceMust(t, e)
		c.constructed = 3
		serviceMust(t, c.zp.SetZP(&c.p))
		r, e := s.beginPrimeTrialInvocation(c.zp)
		serviceMust(t, e)
		r.frame.constructed = 1
		q.budget.FailAfter(ordinal)
		e = c.zp.toZPCopyWithPrimeReceipt(m.stateView(&c.fpp), &r.frame.fzp, &r)
		r.frame.executing = false
		primeTrialPhysicalAudit(t, q, s)
		if e != nil && r.frame.transportBound && r.frame.fzp.size > 0 {
			first, x := c.zp.read(c.zp.stateView(&r.frame.fzp), 0)
			serviceMust(t, x)
			if *first != NewInt(1) {
				t.Fatal("completed prefix lost before owning", *first)
			}
		}
		if e == nil && r.frame.transportBound {
			t.Fatal("successful move retained RAW authority")
		}
		rawBefore, boundBefore := r.frame.transport, r.frame.transportBound
		serviceMust(t, s.cleanupPrimeTrialInvocation(&r, false))
		if r.frame.transport != rawBefore || r.frame.transportBound != boundBefore {
			t.Fatal("source-completed phase consumed unjoined RAW")
		}
		primeTrialPhysicalAudit(t, q, s)
		q.budget.FailAfter(-1)
		serviceMust(t, s.cleanupPrimeTrialInvocation(&r, true))
		primeTrialPhysicalAudit(t, q, s)
		serviceMust(t, s.CleanupRetained())
		if e == nil {
			t.Log("first actual completed prefix", ordinal)
			return
		}
	}
	t.Fatal("no actual completed source prefix")
}

// Actual destroyed-old/refused-new Copy qualifies the checked owner-primitive
// guard domain. It does not claim this dangling constant is mathematical trial
// failure. The younger current_fs source DT must preserve older f_pp_zp.
func TestUPolynomialPrimeTrialCurrentDestructorStopsOlderVector(t *testing.T) {
	q, s := mignotteDirectFixture(t)
	m := s.manager
	c, e := s.newSquareFreeScope()
	serviceMust(t, e)
	c.constructed = 1
	serviceMust(t, m.set(&c.fpp, []Int{NewInt(1), NewInt(1)}))
	c.constructed = 2
	serviceMust(t, m.modular.SetInt32(&c.p, 7))
	c.zp, e = NewRelatedUPolynomialManager(m)
	serviceMust(t, e)
	c.constructed = 3
	serviceMust(t, c.zp.SetZP(&c.p))
	r, e := s.beginPrimeTrialInvocation(c.zp)
	serviceMust(t, e)
	f := r.frame
	f.constructed = 1
	serviceMust(t, c.zp.toZPCopyWithPrimeReceipt(m.stateView(&c.fpp), &f.fzp, &r))
	f.current, e = c.zp.NewFactors()
	serviceMust(t, e)
	f.constructed = 2
	serviceMust(t, q.z.Copy(&f.current.constant, &q.z.two64))
	var large Int
	words := make([]uint32, 80)
	words[79] = 1
	serviceMust(t, q.z.SetLimbs(&large, words))
	old := f.fzp
	q.budget.FailAfter(0)
	body := q.z.Copy(&f.current.constant, &large)
	if !numericPhysicalReservationRefusal(body) || f.current.constant.cell != q.z.failedRef {
		t.Fatal("actual destroyed old", body)
	}
	f.executing = false
	stop := s.finishPrimeTrialSource(&r, body)
	if !errors.Is(stop, body) || !errors.Is(stop, arena.ErrInvalidSlot) || f.fzp != old || f.current == nil {
		t.Fatal("younger current DT drained older fzp", stop)
	}
	// Immediate graph includes the real caller-owned larger input.
	primeTrialPhysicalAuditExtra(t, q, s, &large)
	q.budget.FailAfter(-1)
	serviceMust(t, s.cleanupPrimeTrialInvocation(&r, false))
	serviceMust(t, s.cleanupPrimeTrialInvocation(&r, true))
	if !errors.Is(stop, body) || f.sourceCause == nil {
		t.Fatal("owning erased immutable history")
	}
	if s.primeTrialInvocationSourceCause(&r) != f.sourceCause {
		t.Fatal("inactive exact history unavailable")
	}
	if repeated := s.finishPrimeTrialSource(&r, nil); !errors.Is(repeated, body) || !errors.Is(repeated, arena.ErrInvalidSlot) {
		t.Fatal("source retry erased first history", repeated)
	}
	next, x := s.beginPrimeTrialInvocation(c.zp)
	serviceMust(t, x)
	next.frame.constructed = 1
	serviceMust(t, c.zp.toZPCopyWithPrimeReceipt(m.stateView(&c.fpp), &next.frame.fzp, &next))
	next.frame.executing = false
	serviceMust(t, s.finishPrimeTrialSource(&next, nil))
	if !errors.Is(stop, body) || !errors.Is(stop, arena.ErrInvalidSlot) {
		t.Fatal("actual new invocation mutated earlier returned diagnostic", stop)
	}
	serviceMust(t, q.z.Del(&large))
	primeTrialPhysicalAudit(t, q, s)
	serviceMust(t, s.CleanupRetained())
}

func TestUPolynomialPrimeTrialEveryActualReservation(t *testing.T) {
	for ordinal := int64(0); ordinal < 2048; ordinal++ {
		q, s := mignotteDirectFixture(t)
		m := s.manager
		c, e := s.newSquareFreeScope()
		serviceMust(t, e)
		c.constructed = 1
		serviceMust(t, m.set(&c.fpp, []Int{NewInt(-2), NewInt(0), NewInt(1)}))
		c.constructed = 2
		serviceMust(t, m.modular.SetInt32(&c.p, 7))
		c.zp, e = NewRelatedUPolynomialManager(m)
		serviceMust(t, e)
		c.constructed = 3
		serviceMust(t, c.zp.SetZP(&c.p))
		c.best, e = c.zp.NewFactors()
		serviceMust(t, e)
		c.constructed = 4
		c.constructed = 5
		serviceMust(t, m.modular.SetInt32(&c.bestP, 2))
		c.constructed = 6
		serviceMust(t, c.ds.init(c.best))
		out, e := m.NewFactors()
		serviceMust(t, e)
		trials := uint32(0)
		q.budget.FailAfter(ordinal)
		route, body := s.factorPrimeTrial(c.zp, &c.fpp, out, c.best, 1, &c.p, &c.bestP, &c.ds, &trials)
		primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{out})
		q.budget.FailAfter(-1)
		serviceMust(t, out.Close())
		serviceMust(t, s.CleanupRetained())
		if body == nil {
			if route != 1 || trials != 1 {
				t.Fatal(route, trials)
			}
			t.Log("first actual full trial success", ordinal)
			return
		}
	}
	t.Fatal("no actual whole trial completion")
}

func TestUPolynomialPrimeTrialCapturedContextRejectsReuseCopiesAndForeign(t *testing.T) {
	q, s := mignotteDirectFixture(t)
	m := s.manager
	c, e := s.newSquareFreeScope()
	serviceMust(t, e)
	c.zp, e = NewRelatedUPolynomialManager(m)
	serviceMust(t, e)
	c.constructed = 3
	r, e := s.beginPrimeTrialInvocation(c.zp)
	serviceMust(t, e)
	r.frame.constructed = 1
	r.frame.transportBound = true
	context := c.zp.modular.primeTrialAllocationContext(&r)
	if context.roots(q.z) != &r.frame.numeric {
		t.Fatal("actual embedded context refused")
	}
	copied := *r.frame
	bad := r
	bad.frame = &copied
	if (integerAllocationInvocationContext{primeTrial: &bad, primeTrialInvocation: bad.invocation}).roots(q.z) != nil {
		t.Fatal("copied frame acquired")
	}
	c.zp.callbackDepth++
	if context.roots(q.z) != nil {
		t.Fatal("callback acquired")
	}
	c.zp.callbackDepth--
	foreign, e := NewRelatedUPolynomialManager(m)
	serviceMust(t, e)
	if foreign.modular.primeTrialAllocationContext(&r).roots(q.z) != nil {
		t.Fatal("foreign receiver acquired")
	}
	serviceMust(t, foreign.Close())
	r.frame.executing = false
	serviceMust(t, s.cleanupPrimeTrialInvocation(&r, false))
	serviceMust(t, s.cleanupPrimeTrialInvocation(&r, true))
	next, e := s.beginPrimeTrialInvocation(c.zp)
	serviceMust(t, e)
	next.frame.constructed = 1
	next.frame.transportBound = true
	// The context retained the old actual receipt address and immutable nonce.
	r = next
	if context.roots(q.z) != nil {
		t.Fatal("reused receipt pointer retargeted old context")
	}
	next.frame.executing = false
	serviceMust(t, s.cleanupPrimeTrialInvocation(&next, false))
	serviceMust(t, s.cleanupPrimeTrialInvocation(&next, true))
	serviceMust(t, s.CleanupRetained())
}

// The generic source helper accepts a true large positive modulus. The trial
// loop's uint32 prime selection cannot itself produce this modulus, so this
// proves reached helper RAW publication, not ordinary large-prime trial input.
func TestUPolynomialPrimeToZPCompletedPrefixAndPublishedRawPushFailure(t *testing.T) {
	found := false
	for ordinal := int64(0); ordinal < 1024; ordinal++ {
		q, s := mignotteDirectFixture(t)
		m := s.manager
		c, e := s.newSquareFreeScope()
		serviceMust(t, e)
		c.constructed = 1
		serviceMust(t, m.set(&c.fpp, []Int{NewInt(1), NewInt(0), q.z.two64}))
		c.constructed = 2
		serviceMust(t, q.z.SetLimbs(&c.p, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
		c.zp, e = NewRelatedUPolynomialManager(m)
		serviceMust(t, e)
		c.constructed = 3
		serviceMust(t, c.zp.SetZP(&c.p))
		r, e := s.beginPrimeTrialInvocation(c.zp)
		serviceMust(t, e)
		r.frame.constructed = 1
		q.budget.FailAfter(ordinal)
		body := c.zp.toZPCopyWithPrimeReceipt(m.stateView(&c.fpp), &r.frame.fzp, &r)
		r.frame.executing = false
		primeTrialPhysicalAudit(t, q, s)
		if body != nil && r.frame.fzp.size == 2 && r.frame.transportBound && validNumericRef(q.z.cells, r.frame.transport.cell) {
			same, x := q.z.Equal(&r.frame.transport, &q.z.two64)
			serviceMust(t, x)
			if !same {
				t.Fatal("actual published RAW value", r.frame.transport)
			}
			found = true
			raw := r.frame.transport
			serviceMust(t, s.cleanupPrimeTrialInvocation(&r, false))
			if r.frame.transport != raw || !r.frame.transportBound {
				t.Fatal("source vector lifetime consumed RAW p_i")
			}
			primeTrialPhysicalAudit(t, q, s)
		}
		q.budget.FailAfter(-1)
		serviceMust(t, s.cleanupPrimeTrialInvocation(&r, false))
		serviceMust(t, s.cleanupPrimeTrialInvocation(&r, true))
		primeTrialPhysicalAudit(t, q, s)
		serviceMust(t, s.CleanupRetained())
		if body == nil {
			if !found {
				t.Fatal("no actual BIG RAW push refusal before success", ordinal)
			}
			t.Log("completed helper success", ordinal)
			return
		}
	}
	t.Fatal("no actual helper completion")
}

func TestUPolynomialPrimeTrialEnclosingOwnContinuationEveryHelperPrefix(t *testing.T) {
	for ordinal := int64(0); ordinal < 512; ordinal++ {
		q, s := mignotteDirectFixture(t)
		m := s.manager
		c, e := s.newSquareFreeScope()
		serviceMust(t, e)
		c.constructed = 1
		serviceMust(t, m.set(&c.fpp, []Int{NewInt(1), q.z.two64, NewInt(1)}))
		c.constructed = 2
		serviceMust(t, m.modular.SetInt32(&c.p, 7))
		c.zp, e = NewRelatedUPolynomialManager(m)
		serviceMust(t, e)
		c.constructed = 3
		serviceMust(t, c.zp.SetZP(&c.p))
		r, e := s.beginPrimeTrialInvocation(c.zp)
		serviceMust(t, e)
		r.frame.constructed = 1
		q.budget.FailAfter(ordinal)
		body := c.zp.toZPCopyWithPrimeReceipt(m.stateView(&c.fpp), &r.frame.fzp, &r)
		r.frame.executing = false
		_ = s.finishPrimeTrialSource(&r, body)
		primeTrialPhysicalAudit(t, q, s)
		q.budget.FailAfter(-1)
		if e = s.CleanupRetained(); e != nil {
			t.Fatal("actual enclosing owning continuation", ordinal, e, r.frame.normalize, r.frame.numeric, c.constructed)
		}
		if body == nil {
			t.Log("first source helper completion", ordinal)
			return
		}
	}
	t.Fatal("no helper completion")
}
