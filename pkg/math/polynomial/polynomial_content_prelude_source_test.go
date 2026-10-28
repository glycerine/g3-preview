package polynomial

// Source: src/math/polynomial/polynomial.cpp:3418-3430;
// src/util/mpz.cpp:730-817; src/util/mpz.h:393-407.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
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
		var a, b, i unumeral.Int
		serviceMust(t, c.integer.SetString(&a, "354224848179261915075"))
		serviceMust(t, c.integer.SetString(&b, "573147844013817084101"))
		serviceMust(t, c.integer.Mul2kInPlace(&a, 2048))
		serviceMust(t, c.integer.Mul2kInPlace(&b, 2048))
		x, e := c.monomials.MkVariable(0, 1)
		serviceMust(t, e)
		y, e := c.monomials.MkVariable(1, 1)
		serviceMust(t, e)
		p, e := c.MkTerms([]unumeral.Int{a, b}, []Monomial{x, y})
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
		c.integer.Budget().FailAfter(ordinal)
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
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, ref.root.value}, []*unumeral.Int{&a, &b, &i, &s.ai}, extra)
		if callErr != nil && !numericPhysicalReservationRefusal(callErr) {
			t.Fatal("actual Content refusal", ordinal, callErr)
		}
		if c.integer.GcdRaw.Active && !s.aiConstructed && (unumeral.NumericAllocationPending(&c.integer.GcdRaw.Numeric) || c.integer.GcdRaw.Values[0].Cell != 0) {
			if !c.matchesContentInvocation(&receipt) || s.gcd.Manager != c.integer || s.gcd.Frame != &c.integer.GcdRaw || s.gcd.Invocation != c.integer.GcdRaw.Invocation {
				t.Fatal("actual before-ai GCD child missing", ordinal, callErr)
			}
			if s.aiEnded {
				t.Fatal("absent ai constructor received destructor credit")
			}
			saw = true
		}
		retry := !retried && callErr != nil && c.integer.GcdRaw.Active && !s.aiConstructed && c.integer.Tmp[0] != 0 && c.integer.Tmp[1] != 0
		before := s.ai
		if shortcut {
			serviceMust(t, full.cleanupICCPInvocation(&outer, false))
		} else {
			serviceMust(t, c.cleanupContentInvocation(&receipt, false))
		}
		if !s.aiConstructed && s.ai != before {
			t.Fatal("unconstructed ai acquired source DT")
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, ref.root.value}, []*unumeral.Int{&a, &b, &i, &s.ai}, extra)
		// The older caller i NUM is completed before unjoined RAW owning abort.
		if callErr != nil {
			serviceMust(t, c.integer.Del(&i))
		}
		if shortcut {
			serviceMust(t, full.cleanupICCPInvocation(&outer, true))
		} else {
			serviceMust(t, c.cleanupContentInvocation(&receipt, true))
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, ref.root.value}, []*unumeral.Int{&a, &b, &i}, extra)
		c.integer.Budget().FailAfter(-1)
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
			}()}, []*unumeral.Int{&a, &b, &i}, extra)
			retried = true
			t.Log("actual same-owner GCD prelude retry", ordinal)
		}
		if callErr == nil || retry {
			var expected unumeral.Int
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
	context := numericIntegerModularManager_contentAllocationContext(c.modular, &s.receipt)
	if (context.roots(z)) != (&s.numeric) {
		t.Fatal("actual guarded frame refused")
	}
	copied := *s
	forged := polynomialContentInvocationReceipt{c, &copied, 1}
	if (numericIntegerManager_contentAllocationRoots(z, &forged)) != nil {
		t.Fatal("copied frame acquired roots")
	}
	foreign := *c
	forged = polynomialContentInvocationReceipt{&foreign, &foreign.contentLocal, 1}
	if (numericIntegerManager_contentAllocationRoots(z, &forged)) != nil {
		t.Fatal("copied owner acquired roots")
	}
	c.callbackDepth++
	if (context.roots(z)) != nil {
		t.Fatal("callback acquired roots")
	}
	c.callbackDepth--
	s.executing = false
	serviceMust(t, c.cleanupContentInvocation(&r, false))
	serviceMust(t, c.cleanupContentInvocation(&r, true))
	*s = polynomialContentScope{invocation: 2, active: true, executing: true}
	current := polynomialContentInvocationReceipt{c, s, 2}
	s.receipt = current
	if (context.roots(z)) != nil {
		t.Fatal("old saved context retargeted")
	}
	if (numericIntegerManager_contentAllocationRoots(z, &current)) != (&s.numeric) {
		t.Fatal("current frame refused")
	}
	serviceMust(t, c.cleanupContentInvocation(&r, true))
	if !s.active || s.invocation != 2 {
		t.Fatal("stale receipt consumed current frame")
	}
	s.executing = false
	serviceMust(t, c.cleanupContentInvocation(&current, true))
}

// Signed-host TLS owner-primitive domain: actual GCD output precedes its RAW
// end DEL, and that reached child source stop precedes Content's ai constructor.
func TestPolynomialContentBeforeAIReachedGCDSourceStop(t *testing.T) {
	budget := arena.NewBudget(1 << 28)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Memory: rt, Cells: arena.Config{PageSize: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: budget, Memory: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c, z := full.core, q.Integers()
	for n := 0; n < 2; n++ {
		_, e = mm.MkVar()
		serviceMust(t, e)
	}
	var a, b, i, g unumeral.Int
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
	p, e := c.MkTerms([]unumeral.Int{a, b}, []Monomial{x, y})
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
	if !errors.Is(returned, arena.ErrBounds) || !z.GcdRaw.BodySucceeded || s.aiConstructed || s.aiEnded || i != unumeral.NewInt(1) || s.gcd.Frame != &z.GcdRaw {
		t.Fatal("actual child end DEL before ai absent", returned, z.GcdRaw.BodySucceeded, s.aiConstructed, s.aiEnded, i)
	}
	captured := returned.Error()
	cause := c.contentInvocationSourceCause(&receipt)
	if cause == nil || cause != z.RawInvocationSourceCause(&s.gcd) {
		t.Fatal("actual child source stop missing parent", cause)
	}
	polynomialImpPhysicalAuditNamedExtra(t, full, []Polynomial{p}, []*unumeral.Int{&a, &b, &i, &g}, 0, []umemory.Block{lane})
	serviceMust(t, c.cleanupContentInvocation(&receipt, false))
	if c.contentInvocationSourceCause(&receipt) != cause || s.aiConstructed || s.aiEnded {
		t.Fatal("false own phase invented ai/source completion")
	}
	recovery, e := rt.Allocate(2000000)
	serviceMust(t, e)
	serviceMust(t, c.cleanupContentInvocation(&receipt, true))
	if c.contentInvocationSourceCause(&receipt) != cause || z.RawInvocationSourceCause(&receipt.frame.gcd) != cause || returned.Error() != captured {
		t.Fatal("own acknowledgement changed source history")
	}
	polynomialImpPhysicalAuditNamedExtra(t, full, []Polynomial{p}, []*unumeral.Int{&a, &b, &i, &g}, 0, []umemory.Block{lane, recovery})
	var later polynomialContentInvocationReceipt
	serviceMust(t, c.integerContentWithReceipt(p, &i, &primitive, &later))
	if later.invocation == receipt.invocation || returned.Error() != captured || !errors.Is(returned, arena.ErrBounds) {
		t.Fatal("later source invocation changed old error")
	}
	serviceMust(t, primitive.close(c))
	serviceMust(t, c.DecRef(p))
	for _, v := range []*unumeral.Int{&a, &b, &i, &g} {
		serviceMust(t, z.Del(v))
	}
	serviceMust(t, rt.Deallocate(&lane))
	serviceMust(t, full.CloseChecked())
	serviceMust(t, q.CloseChecked())
	e = rt.Deallocate(&recovery)
	if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (umemory.Block{})) {
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
				var a, i, modulus unumeral.Int
				serviceMust(t, z.SetInt64(&a, 1))
				serviceMust(t, z.Mul2kInPlace(&a, 2048))
				p, e := c.MkConstantTakeCopy(&a)
				serviceMust(t, e)
				serviceMust(t, c.IncRef(p))
				if field {
					serviceMust(t, z.SetInt64(&modulus, 1))
					serviceMust(t, z.Mul2kInPlace(&modulus, 521))
					one := unumeral.NewInt(1)
					serviceMust(t, z.Sub(&modulus, &one, &modulus))
					serviceMust(t, c.modular.SetZP(&modulus))
				}
				var result polynomialReference
				var r polynomialContentInvocationReceipt
				z.Budget().FailAfter(prefix)
				returned := c.integerContentWithReceipt(p, &i, &result, &r)
				if returned != nil && !numericPhysicalReservationRefusal(returned) {
					t.Fatal("actual constant SET refusal", prefix, returned)
				}
				polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result.value, result.incoming}, []*unumeral.Int{&a, &i, &modulus}, 0)
				if returned != nil && polynomialNegNumericPending(&c.contentLocal.numeric) {
					saw = true
				}
				if returned != nil {
					if !c.matchesContentInvocation(&r) || c.contentLocal.aiConstructed || c.contentLocal.gcd.Frame != nil {
						t.Fatal("actual constant SET child omitted or invented NUM/GCD", prefix, r, c.contentLocal)
					}
					saw = true
				}
				serviceMust(t, c.cleanupContentInvocation(&r, false))
				if returned != nil {
					serviceMust(t, z.Del(&i))
				}
				serviceMust(t, c.cleanupContentInvocation(&r, true))
				polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result.value, result.incoming}, []*unumeral.Int{&a, &i, &modulus}, 0)
				z.Budget().FailAfter(-1)
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

func TestPolynomialContentFieldActualBIGInverseChildren(t *testing.T) {
	saw, success := false, false
	for prefix := int64(0); prefix < 128 && !success; prefix++ {
		full := polynomialPSCFullFixture(t)
		c, z := full.core, full.core.integer
		var modulus, i unumeral.Int
		serviceMust(t, z.SetInt64(&modulus, 1))
		serviceMust(t, z.Mul2kInPlace(&modulus, 521))
		one := unumeral.NewInt(1)
		serviceMust(t, z.Sub(&modulus, &one, &modulus))
		serviceMust(t, c.modular.SetZP(&modulus))
		p := polynomialQueryTerms(t, c, []int32{2, 4}, [][]MonomialPower{{{0, 1}}, {{1, 1}}})
		var result polynomialReference
		var r polynomialContentInvocationReceipt
		z.Budget().FailAfter(prefix)
		returned := c.integerContentWithReceipt(p, &i, &result, &r)
		if returned != nil && !numericPhysicalReservationRefusal(returned) {
			t.Fatal("actual BIG inverse Content refusal", prefix, returned)
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result.value, result.incoming}, []*unumeral.Int{&modulus, &i}, 0)
		if returned != nil && z.ExtendedRaw.Active {
			if !r.frame.aiConstructed || r.frame.extended.Manager != z || r.frame.extended.Frame != &z.ExtendedRaw || r.frame.extended.Invocation != z.ExtendedRaw.Invocation {
				t.Fatal("actual younger six RAW inverse omitted", prefix, r.frame)
			}
			saw = true
		}
		serviceMust(t, c.cleanupContentInvocation(&r, false))
		if returned != nil {
			serviceMust(t, z.Del(&i))
		}
		serviceMust(t, c.cleanupContentInvocation(&r, true))
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result.value, result.incoming}, []*unumeral.Int{&modulus, &i}, 0)
		z.Budget().FailAfter(-1)
		if returned == nil {
			if i != unumeral.NewInt(2) {
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
