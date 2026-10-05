// Source: src/math/polynomial/polynomial.cpp:4076-4090.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestPolynomialGCDRecursiveLastVariableUsesEuclidWithoutReadingVariable(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	serviceMust(t, c.modular.SetZPUint64(11))
	// The last-variable branch reads size and idx, not vars[idx]. Preserve the
	// actual unavailable variable payload rather than substituting a host slice.
	vars := polynomialGCDVariableBuffer{count: 1, constructed: true, runtime: c.memory}
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, r.Close()) })
	serviceMust(t, full.modGCDRec(c.zero, p, 11, 0, &vars, &r.root))
	same, e := c.Equal(r.root.value, p)
	serviceMust(t, e)
	if !same {
		t.Fatal("recursive terminal Euclid zero shortcut")
	}
}

func TestPolynomialGCDRecursiveDenseIndependentCommonFactor(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	serviceMust(t, c.modular.SetZPUint64(11))
	full.useSparseGCD = false
	factor := polynomialQueryTerms(t, c, []int32{1, 1, 1}, [][]MonomialPower{{{0, 1}}, {{1, 1}}, nil})
	left := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{1, 1}}, nil})
	right := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{1, 1}}, nil})
	u, e := c.Mul(factor, left)
	serviceMust(t, e)
	ur, e := NewPolynomialRef(c)
	serviceMust(t, e)
	serviceMust(t, ur.root.assign(c, u))
	v, e := c.Mul(factor, right)
	serviceMust(t, e)
	vr, e := NewPolynomialRef(c)
	serviceMust(t, e)
	serviceMust(t, vr.root.assign(c, v))
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, r.Close()); serviceMust(t, ur.Close()); serviceMust(t, vr.Close()) })
	serviceMust(t, full.newton.SetOwner(c))
	serviceMust(t, full.skeletons.Push(nil))
	serviceMust(t, full.skeletons.Push(nil))
	var vars polynomialGCDVariableBuffer
	vars.construct(c.memory)
	serviceMust(t, vars.push(0))
	serviceMust(t, vars.push(1))
	serviceMust(t, full.modGCDRec(u, v, 11, 0, &vars, &r.root))
	same, e := c.Equal(r.root.value, factor)
	serviceMust(t, e)
	if !same {
		t.Fatal("actual recursive field GCD independent common factor")
	}
}

func TestPolynomialGCDRecursiveSavedSkeletonIndependentCommonFactor(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	serviceMust(t, c.modular.SetZPUint64(11))
	full.useSparseGCD = true
	factor := polynomialQueryTerms(t, c, []int32{1, 1, 1}, [][]MonomialPower{{{0, 1}}, {{1, 1}}, nil})
	left := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{1, 1}}, nil})
	right := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{1, 1}}, nil})
	u, e := c.Mul(factor, left)
	serviceMust(t, e)
	ur, e := NewPolynomialRef(c)
	serviceMust(t, e)
	serviceMust(t, ur.root.assign(c, u))
	v, e := c.Mul(factor, right)
	serviceMust(t, e)
	vr, e := NewPolynomialRef(c)
	serviceMust(t, e)
	serviceMust(t, vr.root.assign(c, v))
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, r.Close()); serviceMust(t, ur.Close()); serviceMust(t, vr.Close()) })
	serviceMust(t, full.newton.SetOwner(c))
	serviceMust(t, full.skeletons.Push(nil))
	serviceMust(t, full.skeletons.Push(nil))
	var vars polynomialGCDVariableBuffer
	vars.construct(c.memory)
	serviceMust(t, vars.push(0))
	serviceMust(t, vars.push(1))
	serviceMust(t, full.modGCDRec(u, v, 11, 0, &vars, &r.root))
	saved, e := full.skeletons.Get(0)
	serviceMust(t, e)
	if saved == nil {
		t.Fatal("source did not publish learned skeleton")
	}
	serviceMust(t, full.modGCDRec(u, v, 11, 0, &vars, &r.root))
	same, e := c.Equal(r.root.value, factor)
	serviceMust(t, e)
	if !same {
		t.Fatal("actual recursive field GCD independent common factor")
	}
}

func TestPolynomialGCDRecursiveActualSparseBodyExceptionAfterCompletedUnwind(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	serviceMust(t, c.modular.SetZPUint64(11))
	full.useSparseGCD = true
	factor := polynomialQueryTerms(t, c, []int32{1, 1, 1}, [][]MonomialPower{{{0, 1}}, {{1, 1}}, nil})
	left := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{1, 1}}, nil})
	right := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{1, 1}}, nil})
	u, e := c.Mul(factor, left)
	serviceMust(t, e)
	ur, e := NewPolynomialRef(c)
	serviceMust(t, e)
	serviceMust(t, ur.root.assign(c, u))
	v, e := c.Mul(factor, right)
	serviceMust(t, e)
	vr, e := NewPolynomialRef(c)
	serviceMust(t, e)
	serviceMust(t, vr.root.assign(c, v))
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, r.Close()); serviceMust(t, ur.Close()); serviceMust(t, vr.Close()) })
	serviceMust(t, full.newton.SetOwner(c))
	serviceMust(t, full.skeletons.Push(nil))
	serviceMust(t, full.skeletons.Push(nil))
	var vars polynomialGCDVariableBuffer
	vars.construct(c.memory)
	serviceMust(t, vars.push(0))
	serviceMust(t, vars.push(1))
	serviceMust(t, full.modGCDRec(u, v, 11, 0, &vars, &r.root))
	saved, e := full.skeletons.Get(0)
	serviceMust(t, e)
	if saved == nil {
		t.Fatal("source did not publish learned skeleton")
	}
	square, e := c.Mul(factor, factor)
	serviceMust(t, e)
	sr, e := NewPolynomialRef(c)
	serviceMust(t, e)
	serviceMust(t, sr.root.assign(c, square))
	t.Cleanup(func() { serviceMust(t, sr.Close()) })
	newU, e := c.Mul(square, left)
	serviceMust(t, e)
	serviceMust(t, ur.root.assign(c, newU))
	newV, e := c.Mul(square, right)
	serviceMust(t, e)
	serviceMust(t, vr.root.assign(c, newV))
	body := full.modGCDRec(newU, newV, 11, 0, &vars, &r.root)
	var failed *polynomialSparseMGCDException
	if !errors.As(body, &failed) {
		t.Fatal("actual saved sparse skeleton mismatch did not throw source exception", body)
	}
	for _, f := range full.gcdFrames {
		if f.active || f.cleanupCause != nil {
			t.Fatal("body exception did not complete actual source unwind")
		}
	}
	same, e := c.Equal(r.root.value, factor)
	serviceMust(t, e)
	if !same {
		t.Fatal("actual recursive field GCD independent common factor")
	}
}

func TestPolynomialGCDSparseCatchWaitsForGuardedActualDestructorStop(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	serviceMust(t, c.modular.SetZPUint64(11))
	full.useSparseGCD = true
	factor := polynomialQueryTerms(t, c, []int32{1, 1, 1}, [][]MonomialPower{{{0, 1}}, {{1, 1}}, nil})
	left := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{1, 1}}, nil})
	right := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{1, 1}}, nil})
	u, e := c.Mul(factor, left)
	serviceMust(t, e)
	ur, e := NewPolynomialRef(c)
	serviceMust(t, e)
	serviceMust(t, ur.root.assign(c, u))
	v, e := c.Mul(factor, right)
	serviceMust(t, e)
	vr, e := NewPolynomialRef(c)
	serviceMust(t, e)
	serviceMust(t, vr.root.assign(c, v))
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, r.Close()); serviceMust(t, ur.Close()); serviceMust(t, vr.Close()) })
	serviceMust(t, full.newton.SetOwner(c))
	serviceMust(t, full.skeletons.Push(nil))
	serviceMust(t, full.skeletons.Push(nil))
	var vars polynomialGCDVariableBuffer
	vars.construct(c.memory)
	serviceMust(t, vars.push(0))
	serviceMust(t, vars.push(1))
	serviceMust(t, full.modGCDRec(u, v, 11, 0, &vars, &r.root))
	saved, e := full.skeletons.Get(0)
	serviceMust(t, e)
	if saved == nil {
		t.Fatal("source did not publish learned skeleton")
	}
	square, e := c.Mul(factor, factor)
	serviceMust(t, e)
	sr, e := NewPolynomialRef(c)
	serviceMust(t, e)
	serviceMust(t, sr.root.assign(c, square))
	t.Cleanup(func() { serviceMust(t, sr.Close()) })
	newU, e := c.Mul(square, left)
	serviceMust(t, e)
	serviceMust(t, ur.root.assign(c, newU))
	newV, e := c.Mul(square, right)
	serviceMust(t, e)
	serviceMust(t, vr.root.assign(c, newV))
	// Explicitly guarded/non-mathematical source-layout fixture. Actual
	// checkpoint mutates one reached sparse output with a real Copy-produced
	// dead token; no error callback fabricates a destructor refusal.
	var large Int
	limbs := make([]uint32, 8192)
	limbs[0], limbs[len(limbs)-1] = 17, 1
	serviceMust(t, c.integer.SetLimbs(&large, limbs))
	t.Cleanup(func() {
		c.integer.budget.FailAfter(-1)
		serviceMust(t, full.CleanupGCD())
		serviceMust(t, c.integer.Del(&large))
	})
	outer, e := full.beginGCDFunction()
	serviceMust(t, e)
	outer.constructBuffer(c, 0)
	outer.constructBuffer(c, 1)
	for i := uint32(0); i < 33; i++ {
		serviceMust(t, outer.powers[0].push(i, 1))
	}
	olderRaw := outer.powers[0].raw
	outerLeft := false
	defer func() {
		if !outerLeft {
			full.gcdDepth--
			c.pscLeave()
		}
	}()
	injected := false
	c.config.Checkpoint = func() bool {
		if injected {
			return true
		}
		for _, f := range full.gcdFrames {
			sp := &f.sparse
			if !f.active || !sp.constructed || sp.skeleton == nil || sp.outputs.initialized == 0 {
				continue
			}
			a, e := sp.outputs.coefficient(c, 0)
			serviceMust(t, e)
			serviceMust(t, c.integer.Copy(&a, &c.integer.two64))
			c.integer.budget.FailAfter(0)
			copyError := c.integer.Copy(&a, &large)
			if !numericPhysicalReservationRefusal(copyError) || a.cell != c.integer.failedRef {
				t.Fatal("actual Copy replacement guard", copyError)
			}
			serviceMust(t, sp.outputs.putCoefficient(c, 0, a))
			injected = true
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{factor, left, right, square, newU, newV, r.root.value}, []*Int{&large}, ur.header+vr.header+r.header+sr.header)
			c.integer.budget.FailAfter(-1)
			return true
		}
		return true
	}
	body := full.modGCDRec(newU, newV, 11, 0, &vars, &r.root)
	full.gcdDepth--
	c.pscLeave()
	outerLeft = true
	c.config.Checkpoint = func() bool { return true }
	if !injected {
		t.Fatal("actual source callback did not reach constructed sparse local")
	}
	if !errors.Is(body, arena.ErrInvalidSlot) {
		t.Fatal("actual output destructor guard absent", body)
	}
	if full.gcdMayCatchSparse(outer, body) {
		t.Fatal("source catch swallowed actual younger destructor stop", body)
	}
	if stopped := full.finishGCDFunctionSource(outer); !errors.Is(stopped, arena.ErrInvalidSlot) || outer.depth != 2 || outer.powers[0].raw != olderRaw {
		t.Fatal("older source buffers crossed stopped sparse destructor", stopped)
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{factor, left, right, square, newU, newV, r.root.value}, []*Int{&large}, ur.header+vr.header+r.header+sr.header)
	serviceMust(t, full.CleanupGCD())
	var failed *polynomialSparseMGCDException
	if !errors.As(body, &failed) {
		t.Fatal("actual saved sparse skeleton mismatch did not throw source exception", body)
	}

	same, e := c.Equal(r.root.value, factor)
	serviceMust(t, e)
	if !same {
		t.Fatal("actual recursive field GCD independent common factor")
	}
}
