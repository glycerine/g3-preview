// Source: src/math/polynomial/polynomial.cpp:2807-2952; src/test/polynomial.cpp:1651-1672. Licensed under the MIT license.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialNewtonSourceInterpolation(t *testing.T) {
	m := polynomialCellFixture(t)
	serviceMust(t, m.SetZPUint64(97))
	n, e := NewPolynomialNewtonInterpolator(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, n.Close()) }()
	// Exact dormant source samples: (-9)y-21, (-3)y+20, 5y-36.
	cs := [3]int32{-9, -3, 5}
	ks := [3]int32{-21, 20, -36}
	for i := 0; i < 3; i++ {
		as := [1]util.Int{util.NewInt(cs[i])}
		c := util.NewInt(ks[i])
		p, e := m.MkLinearTake(as[:], []uint32{1}, &c)
		serviceMust(t, e)
		polynomialCaller(t, m, p)
		a := util.NewInt(int32(i))
		serviceMust(t, n.Add(&a, p))
	}
	if newtonCount(t, n) != 3 {
		t.Fatal(newtonCount(t, n))
	}
	r, e := NewPolynomialRef(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, r.Close()) }()
	serviceMust(t, n.Mk(0, r))
	p := newtonRefGet(t, r)
	info, e := m.Info(p)
	serviceMust(t, e)
	// Independent coefficient map modulo97, centered: x²y+5xy+41x-9y-21.
	wants := map[[2]uint32]int64{{2, 1}: 1, {1, 1}: 5, {1, 0}: 41, {0, 1}: -9, {0, 0}: -21}
	if info.Size != uint32(len(wants)) {
		t.Fatal(info)
	}
	for i := uint32(0); i < info.Size; i++ {
		mm, e := m.TermMonomial(p, i)
		serviceMust(t, e)
		x, e := m.monomials.DegreeOf(mm, 0)
		serviceMust(t, e)
		y, e := m.monomials.DegreeOf(mm, 1)
		serviceMust(t, e)
		a, e := m.Coefficient(p, i)
		serviceMust(t, e)
		v, e := m.integer.Int64(&a)
		serviceMust(t, e)
		key := [2]uint32{x, y}
		want, ok := wants[key]
		if !ok || v != want {
			t.Fatal(key, v, want)
		}
		delete(wants, key)
	}
	serviceMust(t, n.Reset())
	if newtonCount(t, n) != 0 {
		t.Fatal("reset")
	}
}

func TestPolynomialNewtonReferenceCallbacksAndLazyVector(t *testing.T) {
	m := polynomialCellFixture(t)
	serviceMust(t, m.SetZPUint64(97))
	r, e := NewPolynomialRef(m)
	serviceMust(t, e)
	serviceMust(t, r.Assign(m.unit))
	info, e := m.Info(m.unit)
	serviceMust(t, e)
	if info.RefCount != 2 {
		t.Fatal(info)
	}
	serviceMust(t, r.Assign(m.unit))
	info, e = m.Info(m.unit)
	serviceMust(t, e)
	if info.RefCount != 2 {
		t.Fatal("self", info)
	}
	v, e := NewPolynomialNewtonVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	a, e := v.At(2)
	serviceMust(t, e)
	if newtonVectorCount(t, v) != 3 {
		t.Fatal(newtonVectorCount(t, v))
	}
	b, e := v.At(2)
	serviceMust(t, e)
	if a != b {
		t.Fatal("unstable child")
	}
	serviceMust(t, v.Flush())
	if newtonVectorCount(t, v) != 0 {
		t.Fatal("flush")
	}
	serviceMust(t, v.Close())
	serviceMust(t, r.Close())
}

func TestPolynomialNewtonWholeMethodAndWarmSingleSample(t *testing.T) {
	m := polynomialCellFixture(t)
	serviceMust(t, m.SetZPUint64(97))
	r, e := NewPolynomialRef(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, r.Close()) }()
	serviceMust(t, m.NewtonInterpolation(0, 0, []util.Int{util.NewInt(7)}, []Polynomial{m.unit}, r))
	if m.newtonLocal != nil || newtonRefGet(t, r) != m.unit {
		t.Fatal("source local was retained on success")
	}
	n, e := NewPolynomialNewtonInterpolator(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, n.Close()) }()
	input := util.NewInt(7)
	serviceMust(t, n.Add(&input, m.unit))
	serviceMust(t, n.Mk(0, r))
	serviceMust(t, n.Reset())
	before := numericWarmProfileSnapshot(t, m.integer, m.integer.Budget())
	allocs := testing.AllocsPerRun(20, func() {
		if e := n.Add(&input, m.unit); e != nil {
			panic(e)
		}
		if e := n.Mk(0, r); e != nil {
			panic(e)
		}
		if e := n.Reset(); e != nil {
			panic(e)
		}
	})
	numericWarmProfileAudit(t, m.integer, m.integer.Budget(), before, allocs, false)
	polynomialNewtonPhysicalAudit(t, m, []*PolynomialNewtonInterpolator{n}, []*PolynomialRef{r}, nil, nil, nil, nil)
}

func TestPolynomialNewtonRealAddGrowthRefusalPrefixes(t *testing.T) {
	var failures int
	for fail := int64(0); fail < 256; fail++ {
		m := polynomialCellFixture(t)
		serviceMust(t, m.SetZPUint64(1000000007))
		n, e := NewPolynomialNewtonInterpolator(m)
		serviceMust(t, e)
		input := util.NewInt(0)
		serviceMust(t, n.Add(&input, m.unit))
		input = util.NewInt(2)
		m.integer.Budget().FailAfter(fail)
		e = n.Add(&input, m.unit)
		polynomialNewtonPhysicalAudit(t, m, []*PolynomialNewtonInterpolator{n}, nil, nil, nil, nil, nil, &input)
		m.integer.Budget().FailAfter(-1)
		if e != nil {
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal(fail, e)
			}
			failures++
		}
		serviceMust(t, n.Close())
		serviceMust(t, m.CleanupConstruction())
		serviceMust(t, m.monomials.CleanupCreation())
		serviceMust(t, m.integer.CleanupPendingAllocation())
		if e == nil {
			if failures == 0 {
				t.Fatal("no source failure prefix")
			}
			t.Log("refusals", failures, "first success", fail)
			return
		}
	}
	t.Fatal("never completed")
}

func TestPolynomialNewtonRefVectorSourceConsumedPrefix(t *testing.T) {
	m := polynomialCellFixture(t)
	a, e := m.MkVariable(0, 1)
	serviceMust(t, e)
	b, e := m.MkVariable(1, 1)
	serviceMust(t, e)
	v, e := NewPolynomialRefVector(m)
	serviceMust(t, e)
	serviceMust(t, v.Push(a))
	serviceMust(t, v.Push(b))
	serviceMust(t, m.writeWord(b, 0, 0)) // Explicit malformed RC fixture, source SASSERT refusal.
	e = v.Reset()
	if !polynomialSourceDebug {
		serviceMust(t, e)
		info, x := m.Info(b)
		serviceMust(t, x)
		if info.RefCount != ^uint32(0) {
			t.Fatal(info)
		}
		polynomialNewtonPhysicalAudit(t, m, nil, nil, []*PolynomialRefVector{v}, nil, []Polynomial{b}, nil)
		serviceMust(t, m.writeWord(b, 0, 1))
		serviceMust(t, m.DecRef(b))
		serviceMust(t, v.Close())
		return
	}
	if e != ErrPolynomialAssertion {
		t.Fatal(e)
	}
	if v.state.drain != 1 {
		t.Fatal("completed first DEC not recorded", v.state.drain)
	}
	if v.state.drainDec {
		t.Fatal("source DEBUG zero-RC refusal incorrectly acknowledged as consumed")
	}
	if _, e = m.Info(a); e != ErrPolynomialOwner {
		t.Fatal("first deletion was not complete", e)
	}
	info, e := m.Info(b)
	serviceMust(t, e)
	if info.RefCount != 0 {
		t.Fatal(info)
	}
	polynomialNewtonPhysicalAudit(t, m, nil, nil, []*PolynomialRefVector{v}, nil, nil, nil)
	serviceMust(t, m.writeWord(b, 0, 1)) // Restore only intentionally malformed counter for teardown.
	serviceMust(t, v.Close())
}

func TestPolynomialNewtonConstructionAndLazyRefusals(t *testing.T) {
	for route := 0; route < 2; route++ {
		failures := 0
		for ordinal := int64(0); ordinal < 64; ordinal++ {
			m := polynomialCellFixture(t)
			base := m.integer.Budget().Used()
			if route == 0 {
				runtimeBefore := m.memory.Stats()
				m.integer.Budget().FailAfter(ordinal)
				n, e := NewPolynomialNewtonInterpolator(m)
				polynomialNewtonPhysicalAudit(t, m, []*PolynomialNewtonInterpolator{n}, nil, nil, nil, nil, nil)
				m.integer.Budget().FailAfter(-1)
				if e != nil {
					if !numericPhysicalReservationRefusal(e) {
						t.Fatal(route, ordinal, e)
					}
					failures++
				}
				if n != nil {
					serviceMust(t, n.Close())
				}
				serviceMust(t, m.integer.CleanupPendingAllocation())
				// Actual Runtime block-directory capacity survives child destruction.
				// Only its measured charged metadata may remain; every source root
				// and physical byte must return to the prior independently named graph.
				runtimeAfter := m.memory.Stats()
				polynomialNewtonPhysicalAudit(t, m, nil, nil, nil, nil, nil, nil)
				if runtimeAfter.OwnerBytes != runtimeBefore.OwnerBytes || runtimeAfter.PhysicalBytes != runtimeBefore.PhysicalBytes || runtimeAfter.InitializationBytes != runtimeBefore.InitializationBytes || runtimeAfter.LiveBlocks != runtimeBefore.LiveBlocks || runtimeAfter.DirectoryBytes < runtimeBefore.DirectoryBytes || m.integer.Budget().Used() != base+runtimeAfter.DirectoryBytes-runtimeBefore.DirectoryBytes {
					t.Fatal("constructor refund", ordinal, base, m.integer.Budget().Used(), runtimeBefore, runtimeAfter)
				}
				if e == nil {
					if failures == 0 {
						t.Fatal("missing failure")
					}
					t.Log("constructor failures", failures, "success", ordinal)
					break
				}
			} else {
				v, e := NewPolynomialNewtonVector(m)
				serviceMust(t, e)
				serviceMust(t, v.SetOwner(m))
				before := m.integer.Budget().Used()
				runtimeBefore := m.memory.Stats()
				rawBefore, re := m.memory.AllocationSize()
				serviceMust(t, re)
				m.integer.Budget().FailAfter(ordinal)
				_, e = v.At(3)
				polynomialNewtonPhysicalAudit(t, m, nil, nil, nil, []*PolynomialNewtonVector{v}, nil, nil)
				m.integer.Budget().FailAfter(-1)
				if e != nil {
					if !numericPhysicalReservationRefusal(e) {
						t.Fatal(route, ordinal, e)
					}
					failures++
				}
				serviceMust(t, v.Close())
				serviceMust(t, m.integer.CleanupPendingAllocation())
				// Real Runtime block-directory growth survives child destruction; it
				// is charged owner metadata, not a retained Newton allocation.
				runtimeAfter := m.memory.Stats()
				rawAfter, re := m.memory.AllocationSize()
				serviceMust(t, re)
				polynomialNewtonPhysicalAudit(t, m, nil, nil, nil, nil, nil, nil)
				if runtimeAfter.OwnerBytes != runtimeBefore.OwnerBytes || runtimeAfter.PhysicalBytes != runtimeBefore.PhysicalBytes || runtimeAfter.InitializationBytes != runtimeBefore.InitializationBytes || runtimeAfter.LiveBlocks != runtimeBefore.LiveBlocks || runtimeAfter.DirectoryBytes < runtimeBefore.DirectoryBytes || m.integer.Budget().Used() != base+runtimeAfter.DirectoryBytes-runtimeBefore.DirectoryBytes {
					t.Fatal("lazy refund", ordinal, before, base, m.integer.Budget().Used(), runtimeBefore, runtimeAfter, rawBefore, rawAfter)
				}
				if e == nil {
					if failures == 0 {
						t.Fatal("missing failure")
					}
					t.Log("lazy failures", failures, "success", ordinal)
					break
				}
			}
			if ordinal == 63 {
				t.Fatal("never succeeded", route)
			}
		}
	}
}

func TestPolynomialNewtonSourceModularAndDuplicateProfiles(t *testing.T) {
	m := polynomialCellFixture(t)
	n, e := NewPolynomialNewtonInterpolator(m)
	serviceMust(t, e)
	input := util.NewInt(0)
	e = n.Add(&input, m.unit)
	if polynomialSourceDebug {
		if e != ErrPolynomialAssertion {
			t.Fatal(e)
		}
		if newtonCount(t, n) != 0 {
			t.Fatal("assertion published")
		}
	} else {
		serviceMust(t, e)
		if newtonCount(t, n) != 1 {
			t.Fatal("release body absent")
		}
	}
	serviceMust(t, n.Close())
	serviceMust(t, m.SetZPUint64(97))
	n, e = NewPolynomialNewtonInterpolator(m)
	serviceMust(t, e)
	serviceMust(t, n.Add(&input, m.unit))
	e = n.Add(&input, m.unit)
	if polynomialSourceDebug {
		if e != ErrPolynomialAssertion {
			t.Fatal(e)
		}
		if newtonCount(t, n) != 1 {
			t.Fatal("duplicate assertion published")
		}
	} else {
		serviceMust(t, e)
		if newtonCount(t, n) != 2 {
			t.Fatal("release inverse continuation absent")
		}
	}
	polynomialNewtonPhysicalAudit(t, m, []*PolynomialNewtonInterpolator{n}, nil, nil, nil, nil, nil, &input)
	serviceMust(t, n.Close())
}

func TestPolynomialNewtonLazyDirectoryPlateau(t *testing.T) {
	m := polynomialCellFixture(t)
	v, e := NewPolynomialNewtonVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	_, e = v.At(3)
	serviceMust(t, e)
	serviceMust(t, v.Flush())
	before := m.integer.Budget().Used()
	for i := 0; i < 4; i++ {
		_, e = v.At(3)
		serviceMust(t, e)
		serviceMust(t, v.Flush())
		if m.integer.Budget().Used() != before {
			t.Fatal("unused cold-directory slots grew", before, m.integer.Budget().Used())
		}
	}
	serviceMust(t, v.Close())
}

func newtonCount(t *testing.T, n *PolynomialNewtonInterpolator) uint32 {
	t.Helper()
	v, e := n.NumSamplePoints()
	serviceMust(t, e)
	return v
}
func newtonVectorCount(t *testing.T, n *PolynomialNewtonVector) uint32 {
	t.Helper()
	v, e := n.Size()
	serviceMust(t, e)
	return v
}
func newtonRefGet(t *testing.T, n *PolynomialRef) Polynomial {
	t.Helper()
	v, e := n.Get()
	serviceMust(t, e)
	return v
}

// Public TLS recipe reaches the guarded source signed-overflow domain. No
// private counters, callback errors, or allocation failures are fabricated.
func TestPolynomialNewtonStoppedReferenceBeforeOwningAbort(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	runtime, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	profile := util.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := util.NewRationalManager(util.RationalConfig{Integers: util.IntegerConfig{Memory: runtime, Profile: profile, Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: runtime, Profile: profile})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { t.Fatal("cleanup checkpoint"); return false }, Cooperation: func() error { t.Fatal("cleanup cooperation"); return nil }})
	serviceMust(t, e)
	n, e := NewPolynomialNewtonInterpolator(m)
	serviceMust(t, e)
	defer func() {
		contextTLSFinish(t, runtime, n.Close)
		contextTLSFinish(t, runtime, m.CloseChecked)
		before := runtime.Stats().LiveBlocks
		e := mm.DecManagerRef()
		if e != nil && (!errors.Is(e, arena.ErrBounds) || runtime.Stats().LiveBlocks >= before) {
			t.Fatal("source manager DEC", e)
		}
		contextTLSFinish(t, runtime, mm.CloseChecked)
		contextTLSFinish(t, runtime, q.CloseChecked)
		contextTLSFinish(t, runtime, runtime.Close)
	}()
	inputValue := util.NewInt(7)
	serviceMust(t, n.pushNumeral(&n.inputs, &inputValue))
	// Distinct raw core fields give the actual old polynomial a >100000-byte
	// physical extent. Duplicate terms are deliberately unsimplified raw input.
	const size = 4096
	as := make([]util.Int, size)
	ms := make([]Monomial, size)
	unit, e := mm.Unit()
	serviceMust(t, e)
	for i := range as {
		as[i] = util.NewInt(1)
		ms[i] = unit
		serviceMust(t, mm.IncRef(unit))
	}
	old, e := m.MkCoreTake(as, ms)
	serviceMust(t, e)
	small := util.NewInt(2)
	incoming, e := m.MkConstantTake(&small)
	serviceMust(t, e)
	other := util.NewInt(3)
	older, e := m.MkConstantTake(&other)
	serviceMust(t, e)
	n.scope.kind = 1
	n.scope.refsConstructed = true
	serviceMust(t, n.scope.auxPoly.assign(m, old))
	serviceMust(t, n.scope.temp.assign(m, older))
	// Prepare source recycle storage before the undefined-accounting boundary.
	warms := [8]Polynomial{}
	for i := range warms {
		a := util.NewInt(int32(4 + i))
		p, e := m.MkConstantTake(&a)
		serviceMust(t, e)
		serviceMust(t, m.IncRef(p))
		warms[i] = p
	}
	for _, p := range warms {
		serviceMust(t, m.DecRef(p))
	}
	for range warms {
		_, e = m.ids.Next()
		serviceMust(t, e)
	}
	mmwarm := [4]Monomial{}
	for i := range mmwarm {
		a, e := mm.MkMonomial([]MonomialPower{{uint32(i + 100), 1}})
		serviceMust(t, e)
		serviceMust(t, mm.IncRef(a))
		mmwarm[i] = a
	}
	for _, a := range mmwarm {
		serviceMust(t, mm.DecRef(a))
	}
	for range mmwarm {
		_, e = mm.ids.Next()
		serviceMust(t, e)
	}
	contextTLSNearMinimum(t, runtime)
	e = n.scope.auxPoly.assign(m, incoming)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual postphysical free", e)
	}
	e = n.finish(e)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("retained source cause", e)
	}
	// The completed outer interpolator may not owning-ack a stopped younger
	// source destructor. Explicit caller cleanup remains a later operation.
	readInput, readErr := n.Inputs().Get(0)
	if readErr != nil || readInput != inputValue {
		t.Fatal("stopped local poisoned source const inputs", readInput, readErr)
	}
	e = n.sourceClose()
	if !errors.Is(e, arena.ErrBounds) || n.closeStage != 0 {
		t.Fatal("source outer destructor advanced", e, n.closeStage)
	}
	if n.scope.auxPoly.value != old || n.scope.auxPoly.incoming != incoming || n.scope.temp.value != older {
		t.Fatal("owning acknowledgement/abort ran before stopped source unwind", n.scope)
	}
	loc, e := m.polynomialLocation(old)
	serviceMust(t, e)
	if !loc.released {
		t.Fatal("old raw free did not physically complete")
	}
	info, e := m.Info(incoming)
	serviceMust(t, e)
	if info.RefCount != 1 {
		t.Fatal("unjoined INC was consumed early", info)
	}
	info, e = m.Info(older)
	serviceMust(t, e)
	if info.RefCount != 1 {
		t.Fatal("older source destructor reached early", info)
	}
	polynomialNewtonPhysicalAudit(t, m, []*PolynomialNewtonInterpolator{n}, nil, nil, nil, nil, nil)
}

func TestPolynomialNewtonMkGrowthAndCancellationPrefixes(t *testing.T) {
	failures := 0
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		m := polynomialCellFixture(t)
		serviceMust(t, m.SetZPUint64(97))
		n, e := NewPolynomialNewtonInterpolator(m)
		serviceMust(t, e)
		outputs := [3]Polynomial{}
		for i := range outputs {
			a := util.NewInt(int32(i*i + 1))
			p, e := m.MkConstantTake(&a)
			serviceMust(t, e)
			outputs[i] = p
			serviceMust(t, m.IncRef(p))
			input := util.NewInt(int32(i))
			serviceMust(t, n.Add(&input, p))
		}
		r, e := NewPolynomialRef(m)
		serviceMust(t, e)
		serviceMust(t, r.Assign(m.unit))
		m.integer.Budget().FailAfter(ordinal)
		e = n.Mk(0, r)
		polynomialNewtonPhysicalAudit(t, m, []*PolynomialNewtonInterpolator{n}, []*PolynomialRef{r}, nil, nil, outputs[:], nil)
		m.integer.Budget().FailAfter(-1)
		if e != nil {
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal(ordinal, e)
			}
			failures++
			if r.root.value != m.unit {
				t.Fatal("output changed before final assignment", ordinal, r.root)
			}
		}
		serviceMust(t, r.Close())
		serviceMust(t, n.Close())
		serviceMust(t, m.CleanupConstruction())
		serviceMust(t, m.monomials.CleanupCreation())
		serviceMust(t, m.integer.CleanupPendingAllocation())
		for _, p := range outputs {
			serviceMust(t, m.DecRef(p))
		}
		if e == nil {
			if failures == 0 {
				t.Fatal("missing actual Mk failures")
			}
			t.Log("Mk failures", failures, "success", ordinal)
			break
		}
		if ordinal == 255 {
			t.Fatal("never complete")
		}
	}
	m := polynomialCellFixture(t)
	serviceMust(t, m.SetZPUint64(97))
	n, e := NewPolynomialNewtonInterpolator(m)
	serviceMust(t, e)
	outputs := [3]Polynomial{}
	for i := int32(0); i < 3; i++ {
		a := util.NewInt(i*i + 1)
		p, e := m.MkConstantTake(&a)
		serviceMust(t, e)
		outputs[i] = p
		serviceMust(t, m.IncRef(p))
		input := util.NewInt(i)
		serviceMust(t, n.Add(&input, p))
	}
	r, e := NewPolynomialRef(m)
	serviceMust(t, e)
	serviceMust(t, r.Assign(m.unit))
	var visits uint32
	m.config.Checkpoint = func() bool { visits++; return false }
	m.config.Cooperation = func() error { t.Fatal("cooperate after cancel"); return nil }
	e = n.Mk(0, r)
	if e != ErrUPolynomialCanceled || visits != 1 || r.root.value != m.unit {
		t.Fatal(e, visits, r.root)
	}
	polynomialNewtonPhysicalAudit(t, m, []*PolynomialNewtonInterpolator{n}, []*PolynomialRef{r}, nil, nil, outputs[:], nil)
	serviceMust(t, r.Close())
	serviceMust(t, n.Close())
	for _, p := range outputs {
		serviceMust(t, m.DecRef(p))
	}
}

func TestPolynomialNewtonSignedSourceCountGuardBeforeUnreadVector(t *testing.T) {
	m := polynomialCellFixture(t)
	serviceMust(t, m.SetZPUint64(97))
	n, e := NewPolynomialNewtonInterpolator(m)
	serviceMust(t, e)
	input := util.NewInt(0)
	serviceMust(t, n.Add(&input, m.unit))
	r, e := NewPolynomialRef(m)
	serviceMust(t, e)
	oldCount, e := n.inputs.size(m)
	serviceMust(t, e)
	oldVS := n.vs.values.raw
	dead, e := m.memory.Allocate(8)
	serviceMust(t, e)
	stale := dead
	serviceMust(t, m.memory.Deallocate(&dead))
	serviceMust(t, m.memory.WriteUint32(n.inputs.raw, 4, 0x80000000))
	n.vs.values.raw = stale
	e = n.Mk(0, r)
	// int num=unsigned_count yields INT_MIN, then num-1 is source signed UB.
	// The explicit reached Go domain guard precedes all m_vs reads and c ctor.
	if !errors.Is(e, arena.ErrBounds) || errors.Is(e, util.ErrRuntimeOwner) {
		t.Error("wrong reached source guard/read order", e)
	}
	n.vs.values.raw = oldVS
	serviceMust(t, m.memory.WriteUint32(n.inputs.raw, 4, oldCount))
	serviceMust(t, n.Close())
	serviceMust(t, r.Close())
}

func TestPolynomialNewtonVectorSourceOwnerBinding(t *testing.T) {
	m := polynomialCellFixture(t)
	v, e := NewPolynomialNewtonVector(m)
	serviceMust(t, e)
	if v.owner != nil {
		t.Fatal("source default constructor bound owner")
	}
	_, e = v.At(0)
	if polynomialSourceDebug {
		if e != ErrPolynomialAssertion {
			t.Fatal(e)
		}
	} else {
		if e != ErrPolynomialOwner {
			t.Fatal(e)
		}
	}
	serviceMust(t, v.SetOwner(m))
	serviceMust(t, v.SetOwner(m))
	_, e = v.At(0)
	serviceMust(t, e)
	serviceMust(t, v.Close())
}

// A bounded malformed source size reaches a real field-access guard after
// the incoming reference's DEC has consumed its count. Restore only that
// deliberate header corruption; retry must acknowledge, not repeat the DEC.
func TestPolynomialNewtonRefVectorPendingAbortRetry(t *testing.T) {
	m := polynomialCellFixture(t)
	p, e := m.MkVariable(0, 1)
	serviceMust(t, e)
	v, e := NewPolynomialRefVector(m)
	serviceMust(t, e)
	m.integer.Budget().FailAfter(0)
	e = v.Push(p)
	polynomialNewtonPhysicalAudit(t, m, nil, nil, []*PolynomialRefVector{v}, nil, nil, nil)
	m.integer.Budget().FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) || v.state.incoming != p {
		t.Fatal(e, v.state)
	}
	serviceMust(t, m.integer.CleanupPendingAllocation())
	serviceMust(t, m.writeWord(p, 8, 2))
	e = v.Close()
	t.Log("reached malformed field guard", e)
	if e != arena.ErrBounds || !v.state.incomingDec || v.state.incoming != p {
		t.Fatal(e, v.state)
	}
	loc, x := m.polynomialLocation(p)
	serviceMust(t, x)
	if loc.deleteStage != 1 || loc.deleteIndex != 0 {
		t.Fatal("consumed DEC prefix", loc)
	}
	polynomialNewtonPhysicalAudit(t, m, nil, nil, []*PolynomialRefVector{v}, nil, nil, nil)
	serviceMust(t, m.writeWord(p, 8, 1))
	serviceMust(t, v.Close())
}

func TestPolynomialNewtonIndependentBoundedInterpolants(t *testing.T) {
	// Known polynomials supply independent sample values; expected coefficients
	// do not use the interpolation implementation or its inverse recurrence.
	coefficients := [5]int64{-5, 3, -2, 7, 1}
	for degree := 0; degree < 5; degree++ {
		m := polynomialCellFixture(t)
		serviceMust(t, m.SetZPUint64(97))
		inputs := make([]util.Int, degree+1)
		outputs := make([]Polynomial, degree+1)
		for i := range inputs {
			x := int64(i - 2)
			v := int64(0)
			for k := degree; k >= 0; k-- {
				v = (v*x + coefficients[k]) % 97
			}
			if v > 48 {
				v -= 97
			}
			if v < -48 {
				v += 97
			}
			inputs[i] = util.NewInt(int32(x))
			a := util.NewInt(int32(v))
			p, e := m.MkConstantTake(&a)
			serviceMust(t, e)
			outputs[i] = p
			serviceMust(t, m.IncRef(p))
		}
		r, e := NewPolynomialRef(m)
		serviceMust(t, e)
		serviceMust(t, m.NewtonInterpolation(0, uint32(degree), inputs, outputs, r))
		p := newtonRefGet(t, r)
		info, e := m.Info(p)
		serviceMust(t, e)
		if info.Size != uint32(degree+1) {
			t.Fatal(degree, info)
		}
		seen := [5]bool{}
		for i := uint32(0); i < info.Size; i++ {
			term, e := m.TermMonomial(p, i)
			serviceMust(t, e)
			power, e := m.monomials.DegreeOf(term, 0)
			serviceMust(t, e)
			a, e := m.Coefficient(p, i)
			serviceMust(t, e)
			v, e := m.integer.Int64(&a)
			serviceMust(t, e)
			if power > uint32(degree) || seen[power] || v != coefficients[power] {
				t.Fatal(degree, power, v)
			}
			seen[power] = true
		}
		polynomialNewtonPhysicalAudit(t, m, nil, []*PolynomialRef{r}, nil, nil, outputs, nil)
		serviceMust(t, r.Close())
		for _, p := range outputs {
			serviceMust(t, m.DecRef(p))
		}
	}
}

func TestPolynomialNewtonWholeSourcePushBeforeOwningAbort(t *testing.T) {
	m := polynomialCellFixture(t)
	serviceMust(t, m.SetZPUint64(97))
	r, e := NewPolynomialRef(m)
	serviceMust(t, e)
	input := util.NewInt(7)
	ordinal := int64(3)
	if polynomialSourceDebug {
		ordinal = 4
	} // Direct cold fixture grows the actual Runtime directory first.
	m.integer.Budget().FailAfter(ordinal) // ctor owner/invs and inputs precede real vs backing.
	e = m.NewtonInterpolation(0, 0, []util.Int{input}, []Polynomial{m.unit}, r)
	polynomialNewtonPhysicalAudit(t, m, []*PolynomialNewtonInterpolator{m.newtonLocal}, []*PolynomialRef{r}, nil, nil, nil, nil)
	m.integer.Budget().FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal(e)
	}
	n := m.newtonLocal
	if n == nil || n.vs.incoming != m.unit {
		t.Fatal("unjoined INC erased by ordinary outer destructor", n)
	}
	info, e := m.Info(m.unit)
	serviceMust(t, e)
	if info.RefCount != 2 || n.closeStage != 3 || n.header == 0 || n.closed {
		t.Fatal("source versus owning prefix", info, n)
	}
	if n.inputs.raw != (util.Block{}) || n.invs.raw != (util.Block{}) || n.vs.values.raw != (util.Block{}) {
		t.Fatal("completed source members not destroyed", n)
	}
	if e = m.CloseChecked(); e != ErrPolynomialPending {
		t.Fatal("pending local owner not retained", e)
	}
	serviceMust(t, m.CleanupNewtonInterpolation())
	serviceMust(t, m.integer.CleanupPendingAllocation())
	info, e = m.Info(m.unit)
	serviceMust(t, e)
	if info.RefCount != 1 {
		t.Fatal(info)
	}
	serviceMust(t, r.Close())
}

func TestPolynomialNewtonRefVectorSourceResetStop(t *testing.T) {
	m := polynomialCellFixture(t)
	p, e := m.MkVariable(0, 1)
	serviceMust(t, e)
	v, e := NewPolynomialRefVector(m)
	serviceMust(t, e)
	serviceMust(t, v.Push(p))
	serviceMust(t, m.writeWord(p, 8, 2))
	e = v.Reset()
	if e != arena.ErrBounds {
		t.Fatal(e)
	}
	polynomialNewtonPhysicalAudit(t, m, nil, nil, []*PolynomialRefVector{v}, nil, nil, nil)
	serviceMust(t, m.writeWord(p, 8, 1))
	if second := v.Reset(); second != e {
		t.Fatal("source Reset owning-acknowledged stopped DEC", second, e)
	}
	serviceMust(t, v.Close())
}

func TestPolynomialNewtonLazySourceFlushBeforeOwningAbort(t *testing.T) {
	m := polynomialCellFixture(t)
	serviceMust(t, m.SetZPUint64(97))
	v, e := NewPolynomialNewtonVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	child, e := v.At(0)
	serviceMust(t, e)
	input := util.NewInt(7)
	m.integer.Budget().FailAfter(1)
	e = child.Add(&input, m.unit)
	polynomialNewtonPhysicalAudit(t, m, nil, nil, nil, []*PolynomialNewtonVector{v}, nil, nil)
	m.integer.Budget().FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) || child.vs.incoming != m.unit {
		t.Fatal(e, child)
	}
	serviceMust(t, v.Flush())
	info, e := m.Info(m.unit)
	serviceMust(t, e)
	if info.RefCount != 2 || child.closed || v.children[0] != child || newtonVectorCount(t, v) != 0 {
		t.Fatal("source Flush owning-aborted unjoined reference", info, child, v)
	}
	polynomialNewtonPhysicalAudit(t, m, nil, nil, nil, []*PolynomialNewtonVector{v}, nil, nil)
	if _, e = v.At(0); e != ErrPolynomialPending {
		t.Fatal("retained child overwritten", e)
	}
	serviceMust(t, v.Close())
	serviceMust(t, m.integer.CleanupPendingAllocation())
	info, e = m.Info(m.unit)
	serviceMust(t, e)
	if info.RefCount != 1 {
		t.Fatal(info)
	}
}

// TLS's returned block is a Go failed-constructor abort root, not a numeral.
func TestPolynomialNewtonWholeFailedConstructorRetainedTLSBlock(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	profile := util.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := util.NewRationalManager(util.RationalConfig{Integers: util.IntegerConfig{Memory: rt, Profile: profile, Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: rt, Profile: profile})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { t.Fatal("constructor checkpoint"); return false }, Cooperation: func() error { t.Fatal("constructor cooperation"); return nil }})
	serviceMust(t, e)
	var prep util.Block
	defer func() {
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, m.CleanupNewtonInterpolation())
		serviceMust(t, rt.Deallocate(&prep))
		serviceMust(t, m.CloseChecked())
		serviceMust(t, mm.DecManagerRef())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, rt.Close())
	}()
	serviceMust(t, m.SetZPUint64(97))
	// Prime the lane using an actual independently owned block. No counter repair.
	local := rt.State().LocalSize
	if local < 0 || local >= 99972 {
		t.Fatal("initial lane", rt.State())
	}
	prep, e = rt.Allocate(uint64(99980 - local - 8))
	serviceMust(t, e)
	if rt.State().LocalSize != 99980 {
		t.Fatal(rt.State())
	}
	before := rt.Stats()
	serviceMust(t, rt.SetMaxSize(1))
	e = m.NewtonInterpolation(0, 0, nil, nil, nil)
	var failure util.Failure
	if !errors.As(e, &failure) || failure.Kind != util.FailureMemout || !rt.State().OutOfMemory {
		t.Fatal("TLS refusal", e, rt.State())
	}
	n := m.newtonLocal
	if n == nil {
		t.Fatal("failed constructor automatically aborted actual returned block")
	}
	if n.invs.pending == (util.Block{}) || n.invs.raw != (util.Block{}) || n.closeStage != 3 || n.closed || n.header == 0 {
		t.Fatal("source/abort phases", n)
	}
	if rt.Stats().LiveBlocks != before.LiveBlocks+1 || rt.Stats().PhysicalBytes != before.PhysicalBytes+40 {
		t.Fatal(before, rt.Stats())
	}
	polynomialNewtonPhysicalWithRaw(t, m, []*PolynomialNewtonInterpolator{n}, nil, nil, nil, nil, nil, []util.Block{prep})
	// Lift only the explicit fixture limit AFTER immediate root/cause inspection.
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, m.CleanupNewtonInterpolation())
	if m.newtonLocal != nil || rt.Stats().LiveBlocks != before.LiveBlocks {
		t.Fatal("abort", rt.Stats())
	}
	polynomialNewtonPhysicalWithRaw(t, m, nil, nil, nil, nil, nil, nil, []util.Block{prep})
}
