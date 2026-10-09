// Source: src/math/polynomial/polynomial.cpp:4076-4090.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/clibrand"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialGCDRecursiveTerminalEuclidIgnoresVariableValue(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	serviceMust(t, c.modular.SetZPUint64(11))
	// The terminal branch depends only on the variable count. A deliberately
	// invalid variable value must not affect its Euclidean result.
	vars := []uint32{^uint32(0)}
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, r.Close()) })
	serviceMust(t, full.modGCDRec(c.zero, p, 11, 0, vars, &r.root))
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
	var vars []uint32

	vars = append(vars, 0)
	vars = append(vars, 1)
	serviceMust(t, full.modGCDRec(u, v, 11, 0, vars, &r.root))
	same, e := c.Equal(r.root.value, factor)
	serviceMust(t, e)
	if !same {
		t.Fatal("actual recursive field GCD independent common factor")
	}
}

func TestPolynomialGCDRecursiveSavedSkeletonIndependentCommonFactor(t *testing.T) {
	// The independent success fixture owns its source-compatible process seed.
	clibrand.Srand(1)
	t.Cleanup(func() { clibrand.Srand(1) })
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
	var vars []uint32

	vars = append(vars, 0)
	vars = append(vars, 1)
	serviceMust(t, full.modGCDRec(u, v, 11, 0, vars, &r.root))
	saved, e := full.skeletons.Get(0)
	serviceMust(t, e)
	if saved == nil {
		t.Fatal("source did not publish learned skeleton")
	}
	serviceMust(t, full.modGCDRec(u, v, 11, 0, vars, &r.root))
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
	var vars []uint32

	vars = append(vars, 0)
	vars = append(vars, 1)
	serviceMust(t, full.modGCDRec(u, v, 11, 0, vars, &r.root))
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
	body := full.modGCDRec(newU, newV, 11, 0, vars, &r.root)
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
	var vars []uint32

	vars = append(vars, 0)
	vars = append(vars, 1)
	serviceMust(t, full.modGCDRec(u, v, 11, 0, vars, &r.root))
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
	var large util.Int
	limbs := make([]uint32, 8192)
	limbs[0], limbs[len(limbs)-1] = 17, 1
	serviceMust(t, c.integer.SetLimbs(&large, limbs))
	t.Cleanup(func() {
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, full.CleanupGCD())
		serviceMust(t, c.integer.Del(&large))
	})
	outer, e := full.beginGCDFunction()
	serviceMust(t, e)
	outer.constructPowers(0)
	outer.constructPowers(1)
	for i := uint32(0); i < 33; i++ {
		serviceMust(t, outer.powers[0].push(i, 1))
	}
	olderStorage := &outer.powers[0][0]
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
			if !f.active || !sp.constructed || sp.skeleton == nil || len(sp.outputs) == 0 {
				continue
			}
			a := sp.outputs[0]
			serviceMust(t, c.integer.Copy(&a, &c.integer.Two64))
			c.integer.Budget().FailAfter(0)
			copyError := c.integer.Copy(&a, &large)
			if !numericPhysicalReservationRefusal(copyError) || (a).Cell != c.integer.FailedRef {
				t.Fatal("actual Copy replacement guard", copyError)
			}
			sp.outputs[0] = a
			injected = true
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{factor, left, right, square, newU, newV, r.root.value}, []*util.Int{&large}, ur.header+vr.header+r.header+sr.header)
			c.integer.Budget().FailAfter(-1)
			return true
		}
		return true
	}
	body := full.modGCDRec(newU, newV, 11, 0, vars, &r.root)
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
	if stopped := full.finishGCDFunctionSource(outer); !errors.Is(stopped, arena.ErrInvalidSlot) || outer.depth != 2 || &outer.powers[0][0] != olderStorage {
		t.Fatal("older source buffers crossed stopped sparse destructor", stopped)
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{factor, left, right, square, newU, newV, r.root.value}, []*util.Int{&large}, ur.header+vr.header+r.header+sr.header)
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

func TestPolynomialGCDRecursiveRepeatedSparsePointThrowsAfterActualUnwind(t *testing.T) {
	// The independent success fixture owns its source-compatible process seed.
	clibrand.Srand(1)
	t.Cleanup(func() { clibrand.Srand(1) })
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
	var vars []uint32

	vars = append(vars, 0)
	vars = append(vars, 1)
	serviceMust(t, full.modGCDRec(u, v, 11, 0, vars, &r.root))
	saved, e := full.skeletons.Get(0)
	serviceMust(t, e)
	if saved == nil {
		t.Fatal("source did not publish learned skeleton")
	}
	// Seed2 starts with two literal selected-profile draws congruent to8 mod11.
	var stream clibrand.GLibc31
	stream.Seed(2)
	if stream.Next()%11 != 8 || stream.Next()%11 != 8 {
		t.Fatal("literal duplicate source sample fixture")
	}
	dense, e := full.newton.At(0)
	serviceMust(t, e)
	n, e := dense.Inputs().Size()
	serviceMust(t, e)
	for i := uint32(0); i < n; i++ {
		a, e := dense.Inputs().Get(i)
		serviceMust(t, e)
		if a == util.NewInt(-3) {
			t.Fatal("selected sparse point was already excluded by dense inputs")
		}
	}
	clibrand.Srand(2)
	body := full.modGCDRec(u, v, 11, 0, vars, &r.root)
	var failed *polynomialSparseMGCDException
	if !errors.As(body, &failed) {
		t.Fatal("literal repeated-point sparse exception absent", body)
	}
	// Direct mod_gcd_rec has no catch. Its actual caller may catch only after
	// the completed lexical unwind, which is checked before any owning ACK.
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{factor, left, right, ur.root.value, vr.root.value, r.root.value}, nil, ur.header+vr.header+r.header)
	for _, frame := range full.gcdFrames {
		if frame.active || frame.cleanupCause != nil {
			t.Fatal("literal sparse body exception retained source DT", frame)
		}
	}

	same, e := c.Equal(r.root.value, factor)
	serviceMust(t, e)
	if !same {
		t.Fatal("actual recursive field GCD independent common factor")
	}
}
