// Source: src/math/polynomial/polynomial.cpp:3418-3430;
// src/util/mpz.cpp:730-817; src/util/mpz.h:393-407.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialContentBeforeAIActualGCDPrelude(t *testing.T) {
	contentBeforeAIActualPrelude(t, false)
}
func TestPolynomialICCPShortcutBeforeAIActualContent(t *testing.T) {
	contentBeforeAIActualPrelude(t, true)
}
func contentBeforeAIActualPrelude(t *testing.T, shortcut bool) {
	saw, success, retried := false, false, false
	for ordinal := int64(0); ordinal < 256 && !success; ordinal++ {
		full := polynomialPSCFullFixture(t)
		c := full.core
		var a, b, i Int
		serviceMust(t, c.integer.SetString(&a, "354224848179261915075"))
		serviceMust(t, c.integer.SetString(&b, "573147844013817084101"))
		serviceMust(t, c.integer.Mul2kInPlace(&a, 2048))
		serviceMust(t, c.integer.Mul2kInPlace(&b, 2048))
		x, e := c.monomials.MkVariable(0, 1)
		serviceMust(t, e)
		y, e := c.monomials.MkVariable(1, 1)
		serviceMust(t, e)
		p, e := c.MkTerms([]Int{a, b}, []Monomial{x, y})
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		ref, e := NewPolynomialRef(c)
		serviceMust(t, e)
		var receipt polynomialContentInvocationReceipt
		var outer polynomialICCPInvocationReceipt
		var pp *PolynomialRef
		if shortcut {
			pp, e = NewPolynomialRef(c)
			serviceMust(t, e)
		}
		c.integer.budget.FailAfter(ordinal)
		var callErr error
		if shortcut {
			callErr = full.iccpPartsWithReceipt(p, 9, &i, &ref.root, &pp.root, &outer)
			receipt = outer.shortcutContent
		} else {
			callErr = c.integerContentWithReceipt(p, &i, &ref.root, &receipt)
		}
		extra := ref.header
		if shortcut {
			extra += pp.header
			if len(full.iccpFrames) != 0 {
				t.Fatal("d0 shortcut invented filter/powers/ci frame")
			}
		}
		s := &c.contentLocal
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, ref.root.value}, []*Int{&a, &b, &i, &s.ai}, extra)
		if callErr != nil && !numericPhysicalReservationRefusal(callErr) {
			t.Fatal("actual Content refusal", ordinal, callErr)
		}
		if c.integer.gcdRaw.active && !s.aiConstructed && (polynomialNegNumericPending(&c.integer.gcdRaw.numeric) || c.integer.gcdRaw.values[0].cell != 0) {
			if !c.matchesContentInvocation(&receipt) || s.gcd.manager != c.integer || s.gcd.frame != &c.integer.gcdRaw || s.gcd.invocation != c.integer.gcdRaw.invocation {
				t.Fatal("actual before-ai GCD child missing", ordinal, callErr)
			}
			if s.aiEnded {
				t.Fatal("absent ai constructor received destructor credit")
			}
			saw = true
		}
		retry := !retried && callErr != nil && c.integer.gcdRaw.active && !s.aiConstructed && c.integer.tmp[0] != 0 && c.integer.tmp[1] != 0
		before := s.ai
		if shortcut {
			serviceMust(t, full.cleanupICCPInvocation(&outer, false))
		} else {
			serviceMust(t, c.cleanupContentInvocation(&receipt, false))
		}
		if !s.aiConstructed && s.ai != before {
			t.Fatal("unconstructed ai acquired source DT")
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, ref.root.value}, []*Int{&a, &b, &i, &s.ai}, extra)
		// The older caller i NUM is completed before unjoined RAW owning abort.
		if callErr != nil {
			serviceMust(t, c.integer.Del(&i))
		}
		if shortcut {
			serviceMust(t, full.cleanupICCPInvocation(&outer, true))
		} else {
			serviceMust(t, c.cleanupContentInvocation(&receipt, true))
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, ref.root.value}, []*Int{&a, &b, &i}, extra)
		c.integer.budget.FailAfter(-1)
		if retry {
			captured := callErr.Error()
			var later polynomialContentInvocationReceipt
			var laterOuter polynomialICCPInvocationReceipt
			if shortcut {
				serviceMust(t, full.iccpPartsWithReceipt(p, 9, &i, &ref.root, &pp.root, &laterOuter))
				later = laterOuter.shortcutContent
			} else {
				serviceMust(t, c.integerContentWithReceipt(p, &i, &ref.root, &later))
			}
			if later.frame != receipt.frame || later.invocation == receipt.invocation || callErr.Error() != captured || !numericPhysicalReservationRefusal(callErr) {
				t.Fatal("actual retry changed saved invocation/error", ordinal)
			}
			serviceMust(t, c.cleanupContentInvocation(&receipt, false))
			serviceMust(t, c.cleanupContentInvocation(&receipt, true))
			if shortcut {
				serviceMust(t, full.cleanupICCPInvocation(&outer, false))
				serviceMust(t, full.cleanupICCPInvocation(&outer, true))
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, ref.root.value, func() Polynomial {
				if pp != nil {
					return pp.root.value
				}
				return Polynomial{}
			}()}, []*Int{&a, &b, &i}, extra)
			retried = true
			t.Log("actual same-owner GCD prelude retry", ordinal)
		}
		if callErr == nil || retry {
			var expected Int
			serviceMust(t, c.integer.SetInt64(&expected, 1))
			serviceMust(t, c.integer.Mul2kInPlace(&expected, 2048))
			cmp, e := c.integer.Compare(&i, &expected)
			serviceMust(t, e)
			serviceMust(t, c.integer.Del(&expected))
			if cmp != 0 {
				t.Fatal("independent coprime Fibonacci content", ordinal)
			}
		}
		if callErr == nil {
			success = true
			t.Log("first actual Content success", ordinal)
		}
		if shortcut {
			serviceMust(t, pp.Close())
		}
		serviceMust(t, ref.Close())
		serviceMust(t, c.DecRef(p))
		serviceMust(t, c.integer.Del(&a))
		serviceMust(t, c.integer.Del(&b))
		serviceMust(t, c.integer.Del(&i))
	}
	if !saw || !success || !retried {
		t.Fatal("actual before-ai refusal/success horizon absent", saw, success)
	}
}

// The saved tuple is authority only for its actual guarded invocation. Copied
// cold records and a later invocation at the same address are invalid Go-owner
// metadata; these checks do not claim a source allocation-failure frontier.
func TestPolynomialContentPreludeActualFrameAndSavedContextGuards(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c, z := full.core, full.core.integer
	s := &c.contentLocal
	t.Cleanup(func() { s.executing = false; serviceMust(t, c.CleanupIntegerContent()) })
	*s = polynomialContentScope{invocation: 1, active: true, executing: true}
	r := polynomialContentInvocationReceipt{c, s, 1}
	s.receipt = r
	context := c.modular.contentAllocationContext(&s.receipt)
	if context.roots(z) != &s.numeric {
		t.Fatal("actual guarded frame refused")
	}
	copied := *s
	forged := polynomialContentInvocationReceipt{c, &copied, 1}
	if z.contentAllocationRoots(&forged) != nil {
		t.Fatal("copied frame acquired roots")
	}
	foreign := *c
	forged = polynomialContentInvocationReceipt{&foreign, &foreign.contentLocal, 1}
	if z.contentAllocationRoots(&forged) != nil {
		t.Fatal("copied owner acquired roots")
	}
	c.callbackDepth++
	if context.roots(z) != nil {
		t.Fatal("callback acquired roots")
	}
	c.callbackDepth--
	s.executing = false
	serviceMust(t, c.cleanupContentInvocation(&r, false))
	serviceMust(t, c.cleanupContentInvocation(&r, true))
	*s = polynomialContentScope{invocation: 2, active: true, executing: true}
	current := polynomialContentInvocationReceipt{c, s, 2}
	s.receipt = current
	if context.roots(z) != nil || z.natural.divisionBufferReceipt(context, 0) != nil {
		t.Fatal("old saved context retargeted")
	}
	if z.contentAllocationRoots(&current) != &s.numeric {
		t.Fatal("current frame refused")
	}
	serviceMust(t, c.cleanupContentInvocation(&r, true))
	if !s.active || s.invocation != 2 {
		t.Fatal("stale receipt consumed current frame")
	}
	s.executing = false
	serviceMust(t, c.cleanupContentInvocation(&current, true))
}

func TestPolynomialContentPreludeNilParentBufferGuard(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	r := polynomialContentInvocationReceipt{}
	context := integerAllocationInvocationContext{content: &r, contentInvocation: 1}
	if full.core.integer.natural.divisionBufferReceipt(context, 0) != nil {
		t.Fatal("absent parent acquired buffer")
	}
	// Invalid Go metadata: even a matching embedded buffer cannot acquire
	// authority if its actual core's Integer owner is absent.
	c, z := full.core, full.core.integer
	s := &c.contentLocal
	*s = polynomialContentScope{invocation: 1, active: true, executing: true}
	r = polynomialContentInvocationReceipt{c, s, 1}
	s.buffers[0] = naturalPNBufferReceipt{owner: z.natural, content: r, invocation: 1, buffer: &z.natural.division[0]}
	defer func() { c.integer = z; s.executing = false; serviceMust(t, c.CleanupIntegerContent()) }()
	c.integer = nil
	if s.buffers[0].bound(false) {
		t.Fatal("absent Integer owner bound private buffer")
	}
}

// Signed-host TLS owner-primitive domain: actual GCD output precedes its RAW
// end DEL, and that reached child source stop precedes Content's ai constructor.
func TestPolynomialContentBeforeAIReachedGCDSourceStop(t *testing.T) {
	budget := arena.NewBudget(1 << 28)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	q, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Memory: rt, Cells: arena.Config{PageSize: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: budget, Memory: rt, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c, z := full.core, q.z
	for n := 0; n < 2; n++ {
		_, e = mm.MkVar()
		serviceMust(t, e)
	}
	var a, b, i, g Int
	words := make([]uint32, 15000)
	words[0] = 7
	words[len(words)-1] = 1
	serviceMust(t, z.SetLimbs(&a, words))
	words[0] = 8
	serviceMust(t, z.SetLimbs(&b, words))
	serviceMust(t, z.GCD(&a, &b, &g))
	serviceMust(t, z.Copy(&i, &a))
	x, e := mm.MkVariable(0, 1)
	serviceMust(t, e)
	y, e := mm.MkVariable(1, 1)
	serviceMust(t, e)
	p, e := c.MkTerms([]Int{a, b}, []Monomial{x, y})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	var primitive polynomialReference
	var receipt polynomialContentInvocationReceipt
	contextTLSNearMinimum(t, rt)
	lane, e := rt.Allocate(8)
	serviceMust(t, e)
	delta := int64(-50000) - rt.State().LocalSize
	if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual signed allocation guard", e)
	}
	returned := c.integerContentWithReceipt(p, &i, &primitive, &receipt)
	s := &c.contentLocal
	if !errors.Is(returned, arena.ErrBounds) || !z.gcdRaw.bodySucceeded || s.aiConstructed || s.aiEnded || i != NewInt(1) || s.gcd.frame != &z.gcdRaw {
		t.Fatal("actual child end DEL before ai absent", returned, z.gcdRaw.bodySucceeded, s.aiConstructed, s.aiEnded, i)
	}
	captured := returned.Error()
	cause := c.contentInvocationSourceCause(&receipt)
	if cause == nil || cause != z.rawInvocationSourceCause(&s.gcd) {
		t.Fatal("actual child source stop missing parent", cause)
	}
	polynomialImpPhysicalAuditNamedExtra(t, full, []Polynomial{p}, []*Int{&a, &b, &i, &g}, 0, []memory.Block{lane})
	serviceMust(t, c.cleanupContentInvocation(&receipt, false))
	if c.contentInvocationSourceCause(&receipt) != cause || s.aiConstructed || s.aiEnded {
		t.Fatal("false own phase invented ai/source completion")
	}
	recovery, e := rt.Allocate(2000000)
	serviceMust(t, e)
	serviceMust(t, c.cleanupContentInvocation(&receipt, true))
	if c.contentInvocationSourceCause(&receipt) != cause || z.rawInvocationSourceCause(&receipt.frame.gcd) != cause || returned.Error() != captured {
		t.Fatal("own acknowledgement changed source history")
	}
	polynomialImpPhysicalAuditNamedExtra(t, full, []Polynomial{p}, []*Int{&a, &b, &i, &g}, 0, []memory.Block{lane, recovery})
	var later polynomialContentInvocationReceipt
	serviceMust(t, c.integerContentWithReceipt(p, &i, &primitive, &later))
	if later.invocation == receipt.invocation || returned.Error() != captured || !errors.Is(returned, arena.ErrBounds) {
		t.Fatal("later source invocation changed old error")
	}
	serviceMust(t, primitive.close(c))
	serviceMust(t, c.DecRef(p))
	for _, v := range []*Int{&a, &b, &i, &g} {
		serviceMust(t, z.Del(v))
	}
	serviceMust(t, rt.Deallocate(&lane))
	serviceMust(t, full.CloseChecked())
	serviceMust(t, q.CloseChecked())
	e = rt.Deallocate(&recovery)
	if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (memory.Block{})) {
		t.Fatal("actual recovery consumption", e)
	}
	contextTLSFinish(t, rt, rt.Close)
	if budget.Used() != 0 {
		t.Fatal("explicit complete owner graph remains", budget.Used())
	}
}

func TestPolynomialContentConstantActualSETAllocation(t *testing.T) {
	for _, field := range []bool{false, true} {
		t.Run(map[bool]string{false: "integer", true: "field"}[field], func(t *testing.T) {
			saw, success := false, false
			for prefix := int64(0); prefix < 128 && !success; prefix++ {
				full := polynomialPSCFullFixture(t)
				c, z := full.core, full.core.integer
				var a, i, modulus Int
				serviceMust(t, z.SetInt64(&a, 1))
				serviceMust(t, z.Mul2kInPlace(&a, 2048))
				p, e := c.MkConstantTakeCopy(&a)
				serviceMust(t, e)
				serviceMust(t, c.IncRef(p))
				if field {
					serviceMust(t, z.SetInt64(&modulus, 1))
					serviceMust(t, z.Mul2kInPlace(&modulus, 521))
					one := NewInt(1)
					serviceMust(t, z.Sub(&modulus, &one, &modulus))
					serviceMust(t, c.modular.SetZP(&modulus))
				}
				var result polynomialReference
				var r polynomialContentInvocationReceipt
				z.budget.FailAfter(prefix)
				returned := c.integerContentWithReceipt(p, &i, &result, &r)
				if returned != nil && !numericPhysicalReservationRefusal(returned) {
					t.Fatal("actual constant SET refusal", prefix, returned)
				}
				polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result.value, result.incoming}, []*Int{&a, &i, &modulus}, 0)
				if returned != nil && polynomialNegNumericPending(&c.contentLocal.numeric) {
					saw = true
				}
				if returned != nil {
					if !c.matchesContentInvocation(&r) || c.contentLocal.aiConstructed || c.contentLocal.gcd.frame != nil {
						t.Fatal("actual constant SET child omitted or invented NUM/GCD", prefix, r, c.contentLocal)
					}
					saw = true
				}
				serviceMust(t, c.cleanupContentInvocation(&r, false))
				if returned != nil {
					serviceMust(t, z.Del(&i))
				}
				serviceMust(t, c.cleanupContentInvocation(&r, true))
				polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result.value, result.incoming}, []*Int{&a, &i, &modulus}, 0)
				z.budget.FailAfter(-1)
				if returned == nil {
					success = true
					if result.value != c.unit {
						t.Fatal("source constant primitive must be unit")
					}
					t.Log("constant SET first actual success", field, prefix)
				}
				serviceMust(t, result.close(c))
				serviceMust(t, c.modular.SetZ())
				serviceMust(t, c.DecRef(p))
				serviceMust(t, z.Del(&a))
				serviceMust(t, z.Del(&i))
				serviceMust(t, z.Del(&modulus))
			}
			if !saw || !success {
				t.Fatal("actual constant SET acquisition/success absent", saw, success)
			}
		})
	}
}

// The checked source verification temporary is a younger lexical child of
// constant SET/PN. Signed-host TLS stop is qualified owner-primitive behavior.
func TestPolynomialContentConstantPNSourceStopBeforeAbsentAI(t *testing.T) {
	if !modularSourceDebug {
		return
	}
	budget := arena.NewBudget(1 << 26)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(budget).Integers
	cfg.Memory = rt
	q, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c, z := full.core, q.z
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	var a, warm, out Int
	digits := make([]uint32, 16)
	digits[0] = 17
	digits[15] = 1
	serviceMust(t, z.SetLimbs(&a, digits))
	p, e := c.MkConstantTakeCopy(&a)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	serviceMust(t, c.modular.SetZPUint64(251))
	serviceMust(t, z.Rem(&a, &c.modular.p, &warm))
	var result polynomialReference
	var r polynomialContentInvocationReceipt
	external, e := rt.Allocate(200000)
	serviceMust(t, e)
	contextTLSNearMinimum(t, rt)
	e = rt.Deallocate(&external)
	if !errors.Is(e, arena.ErrBounds) || external != (memory.Block{}) {
		t.Fatal("actual signed release guard", e, external)
	}
	returned := c.integerContentWithReceipt(p, &out, &result, &r)
	if !errors.Is(returned, arena.ErrBounds) || out.IsSmall() || !c.matchesContentInvocation(&r) || r.frame.aiConstructed || r.frame.aiEnded || r.frame.gcd.frame != nil {
		t.Fatal("actual constant PN source stop/absent NUM", returned, out, r)
	}
	child := r.frame.normalize.frame.verification
	cause := c.contentInvocationSourceCause(&r)
	if cause == nil || cause != c.modular.pNormalizeInvocationSourceCause(&r.frame.normalize) || !z.natural.matchesPNVerification(&child) || !child.frame.constructed || child.frame.ended || z.natural.verification.raw != (memory.Block{}) {
		t.Fatal("actual younger DT history", cause, child)
	}
	captured := returned.Error()
	caller := out
	polynomialImpPhysicalAuditNamedRaw(t, full, []Polynomial{p}, []*Int{&a, &warm, &out}, 0, nil)
	used, state, stats := budget.Used(), rt.State(), rt.Stats()
	if again := c.finishIntegerContent(false); again != cause || out != caller || budget.Used() != used || rt.State() != state || rt.Stats() != stats {
		t.Fatal("source suffix retried child release/advanced caller", again)
	}
	serviceMust(t, c.cleanupContentInvocation(&r, false))
	if c.contentInvocationSourceCause(&r) != cause || out != caller || r.frame.aiConstructed || r.frame.aiEnded || child.frame.sourceCause != cause {
		t.Fatal("false owning phase changed source/caller history")
	}
	recovery, e := rt.Allocate(400000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, func() error { return z.Del(&out) })
	serviceMust(t, c.cleanupContentInvocation(&r, true))
	if c.contentInvocationSourceCause(&r) != cause || c.modular.pNormalizeInvocationSourceCause(&r.frame.normalize) != cause || returned.Error() != captured {
		t.Fatal("true owning phase changed source error")
	}
	var later polynomialContentInvocationReceipt
	serviceMust(t, c.integerContentWithReceipt(p, &out, &result, &later))
	power := int64(1)
	for n := 0; n < 480; n++ {
		power = power * 2 % 251
	}
	want := (power + 17) % 251
	if want > 125 {
		want -= 251
	}
	if out != NewInt(int32(want)) || result.value != c.unit || later.invocation == r.invocation || returned.Error() != captured || !errors.Is(returned, arena.ErrBounds) {
		t.Fatal("later constant source result/history", out, want, returned)
	}
	serviceMust(t, result.close(c))
	serviceMust(t, c.modular.SetZ())
	contextTLSFinish(t, rt, func() error { return c.DecRef(p) })
	for _, v := range []*Int{&a, &warm, &out} {
		contextTLSFinish(t, rt, func() error { return z.Del(v) })
	}
	contextTLSFinish(t, rt, full.CloseChecked)
	contextTLSFinish(t, rt, q.CloseChecked)
	e = rt.Deallocate(&recovery)
	if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (memory.Block{})) {
		t.Fatal("actual recovery consumption", e)
	}
	contextTLSFinish(t, rt, rt.Close)
	if budget.Used() != 0 {
		t.Fatal("actual complete named owner graph", budget.Used())
	}
}

func TestPolynomialContentFieldActualBIGInverseChildren(t *testing.T) {
	saw, success := false, false
	for prefix := int64(0); prefix < 128 && !success; prefix++ {
		full := polynomialPSCFullFixture(t)
		c, z := full.core, full.core.integer
		var modulus, i Int
		serviceMust(t, z.SetInt64(&modulus, 1))
		serviceMust(t, z.Mul2kInPlace(&modulus, 521))
		one := NewInt(1)
		serviceMust(t, z.Sub(&modulus, &one, &modulus))
		serviceMust(t, c.modular.SetZP(&modulus))
		p := polynomialQueryTerms(t, c, []int32{2, 4}, [][]MonomialPower{{{0, 1}}, {{1, 1}}})
		var result polynomialReference
		var r polynomialContentInvocationReceipt
		z.budget.FailAfter(prefix)
		returned := c.integerContentWithReceipt(p, &i, &result, &r)
		if returned != nil && !numericPhysicalReservationRefusal(returned) {
			t.Fatal("actual BIG inverse Content refusal", prefix, returned)
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result.value, result.incoming}, []*Int{&modulus, &i}, 0)
		if returned != nil && z.extendedRaw.active {
			if !r.frame.aiConstructed || r.frame.extended.manager != z || r.frame.extended.frame != &z.extendedRaw || r.frame.extended.invocation != z.extendedRaw.invocation {
				t.Fatal("actual younger six RAW inverse omitted", prefix, r.frame)
			}
			saw = true
		}
		serviceMust(t, c.cleanupContentInvocation(&r, false))
		if returned != nil {
			serviceMust(t, z.Del(&i))
		}
		serviceMust(t, c.cleanupContentInvocation(&r, true))
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result.value, result.incoming}, []*Int{&modulus, &i}, 0)
		z.budget.FailAfter(-1)
		if returned == nil {
			if i != NewInt(2) {
				t.Fatal("source raw integer GCD in finite field", i)
			}
			polynomialQueryExpect(t, c, result.value, map[[3]uint32]int64{{1, 0, 0}: 1, {0, 1, 0}: 2})
			success = true
			t.Log("actual BIG inverse first success", prefix)
		}
		serviceMust(t, result.close(c))
		serviceMust(t, c.modular.SetZ())
		serviceMust(t, z.Del(&modulus))
		serviceMust(t, z.Del(&i))
	}
	if !saw || !success {
		t.Fatal("actual BIG inverse child prefix/success absent", saw, success)
	}
}
