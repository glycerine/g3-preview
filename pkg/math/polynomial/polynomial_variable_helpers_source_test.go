// Source: src/math/polynomial/polynomial.cpp:6179-6232,6307-6344.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func variableHelperArray(values []uint32) PolynomialVariablesView {
	return PolynomialVariablesView{values: values}
}
func TestPolynomialMaxDegreesIndependentFiniteModelAndRetainedVars(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{3, 5, 7, 11}, [][]MonomialPower{{{0, 3}, {1, 2}}, {{1, 5}}, {{2, 4}}, nil})
	d, e := NewPolynomialMaxDegrees(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, d.CloseChecked()) })
	serviceMust(t, d.Init(p))
	n, e := d.NumVariables()
	serviceMust(t, e)
	if n != 3 {
		t.Fatal(n)
	}
	for x, w := range []uint32{3, 5, 4, 0} {
		v, e := d.Degree(uint32(x))
		serviceMust(t, e)
		if v != w {
			t.Fatal(x, v, w)
		}
	}
	vars, e := d.Vars()
	serviceMust(t, e)
	seen := map[uint32]bool{}
	for i := uint32(0); i < n; i++ {
		x, e := vars.At(i)
		serviceMust(t, e)
		if x >= 3 || seen[x] {
			t.Fatal(x, seen)
		}
		seen[x] = true
	}
	serviceMust(t, d.Reset())
	n, e = d.NumVariables()
	serviceMust(t, e)
	if n != 0 {
		t.Fatal(n)
	}
	for x := uint32(0); x < 3; x++ {
		v, e := d.Degree(x)
		serviceMust(t, e)
		if v != 0 {
			t.Fatal(x, v)
		}
	}
	for i := uint32(0); i < 3; i++ {
		x, e := vars.At(i)
		serviceMust(t, e)
		if !seen[x] {
			t.Fatal("scalar retained tail", x)
		}
	}
}
func TestPolynomialVariablePositionsSentinelDuplicateSourceProfiles(t *testing.T) {
	polynomialAssertionActionsFixture(t, "TT")
	r, _ := degreeCopyRuntime(t)
	d, e := NewPolynomialVariablePositions(r)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, d.CloseChecked()) })
	xs := variableHelperArray([]uint32{2, 5, 2})
	e = d.Init(3, xs)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrPolynomialAssertion) {
			t.Fatal(e)
		}
	} else {
		serviceMust(t, e)
	}
	for _, x := range []uint32{0, 1, 3, 4, 6} {
		v, e := d.Position(x)
		serviceMust(t, e)
		if v != ^uint32(0) {
			t.Fatal(x, v)
		}
	}
	v, e := d.Position(2)
	serviceMust(t, e)
	want := uint32(2)
	if polynomialSourceDebug {
		want = 0
	}
	if v != want {
		t.Fatal(v, want)
	}
	v, e = d.Position(5)
	serviceMust(t, e)
	if v != 1 {
		t.Fatal(v)
	}
	e = d.Reset(3, xs)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrPolynomialAssertion) {
			t.Fatal(e)
		}
	} else {
		serviceMust(t, e)
	}
	for _, x := range []uint32{2, 5} {
		v, e := d.Position(x)
		serviceMust(t, e)
		if v != ^uint32(0) {
			t.Fatal(x, v)
		}
	}
}
func TestScopedPolynomialVariablePositionsBorrowedSliceAfterCallerGrowth(t *testing.T) {
	r, _ := degreeCopyRuntime(t)
	d, e := NewPolynomialVariablePositions(r)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, d.CloseChecked()) })
	variables := []uint32{1, 4}
	xs := variableHelperArray(variables)
	s, e := NewScopedPolynomialVariablePositions(d, 2, xs)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.CloseChecked()) })
	variables = append(variables, 7, 8, 9)
	serviceMust(t, s.SourceFinalize())
	for _, x := range []uint32{1, 4} {
		v, e := d.Position(x)
		serviceMust(t, e)
		if v != ^uint32(0) {
			t.Fatal(x, v)
		}
	}
	serviceMust(t, s.SourceFinalize())
}
func TestScopedPolynomialVariablePositionsFailedConstructorNoReset(t *testing.T) {
	polynomialThrowAssertionFixture(t)
	r, _ := degreeCopyRuntime(t)
	d, e := NewPolynomialVariablePositions(r)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, d.CloseChecked()) })
	xs := variableHelperArray([]uint32{1, 1})
	s, e := NewScopedPolynomialVariablePositions(d, 2, xs)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrPolynomialAssertion) || s != nil {
			t.Fatal(s, e)
		}
		v, e := d.Position(1)
		serviceMust(t, e)
		if v != 0 {
			t.Fatal("failed ctor reset", v)
		}
	} else {
		serviceMust(t, e)
		serviceMust(t, s.CloseChecked())
		v, e := d.Position(1)
		serviceMust(t, e)
		if v != ^uint32(0) {
			t.Fatal(v)
		}
	}
}

func TestPolynomialMaxDegreesFirstSeenNativeStorage(t *testing.T) {
	m := polynomialCellFixture(t)
	for x := uint32(3); x <= 10; x++ {
		_, e := m.monomials.MkVar()
		serviceMust(t, e)
	}
	p := polynomialQueryTerms(t, m, []int32{3, 5}, [][]MonomialPower{{{10, 4}}, {{0, 2}}})
	d, e := NewPolynomialMaxDegrees(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, d.CloseChecked()) })
	scope, e := NewScopedPolynomialMaxDegrees(d, p)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, scope.CloseChecked()) })
	n, e := d.NumVariables()
	serviceMust(t, e)
	if n != 2 {
		t.Fatal(n)
	}
	vars, e := d.Vars()
	serviceMust(t, e)
	first, e := vars.At(0)
	serviceMust(t, e)
	second, e := vars.At(1)
	serviceMust(t, e)
	if first != 10 || second != 0 {
		t.Fatal("first-seen order", first, second)
	}
	v, e := scope.Degree(10)
	serviceMust(t, e)
	if v != 4 {
		t.Fatal(v)
	}
}
func TestPolynomialVariablePositionsNativeSliceFill(t *testing.T) {
	r, b := degreeCopyRuntime(t)
	d, e := NewPolynomialVariablePositions(r)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, d.CloseChecked()) })
	xs := variableHelperArray([]uint32{19})
	before := b.Used()
	b.FailAfter(0)
	serviceMust(t, d.Init(1, xs))
	b.FailAfter(-1)
	if b.Used() != before {
		t.Fatal("position slice charged to arena", before, b.Used())
	}
	for x := uint32(0); x < 20; x++ {
		v, e := d.Position(x)
		serviceMust(t, e)
		want := ^uint32(0)
		if x == 19 {
			want = 0
		}
		if v != want {
			t.Fatal(x, v, want)
		}
	}
}
func TestPolynomialVariableHelperImplicitCopyMoveAndScopedCopies(t *testing.T) {
	polynomialThrowAssertionFixture(t)
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{7, 11}, [][]MonomialPower{{{0, 3}}, {{1, 4}}})
	a, e := NewPolynomialMaxDegrees(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, a.CloseChecked()) })
	serviceMust(t, a.Init(p))
	b, e := NewPolynomialMaxDegreesCopy(a)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, b.CloseChecked()) })
	serviceMust(t, b.Reset())
	v, e := a.Degree(1)
	serviceMust(t, e)
	if v != 4 {
		t.Fatal(v)
	}
	serviceMust(t, b.MoveFrom(a))
	serviceMust(t, b.maxDegree.SetDegree(1, 9))
	v, e = a.Degree(1)
	serviceMust(t, e)
	if v != 4 {
		t.Fatal("rvalue assignment shared member backing", v)
	}
	moved, e := NewPolynomialMaxDegreesMove(b)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, moved.CloseChecked()) })
	n, e := b.NumVariables()
	serviceMust(t, e)
	if n != 0 {
		t.Fatal(n)
	}
	polynomialVariablePhysical(t, m, []*PolynomialMaxDegrees{a, b, moved}, nil, nil, nil, nil, []Polynomial{p})
	// Scoped copies bind the SAME reference, run no init, and each source dtor
	// resets that same helper. The second empty max reset is source-defined.
	s, e := NewScopedPolynomialMaxDegrees(b, p)
	serviceMust(t, e)
	copyScope, e := NewScopedPolynomialMaxDegreesCopy(s)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.CloseChecked()); serviceMust(t, copyScope.CloseChecked()) })
	serviceMust(t, s.SourceFinalize())
	serviceMust(t, copyScope.SourceFinalize())
	r, _ := degreeCopyRuntime(t)
	positions, e := NewPolynomialVariablePositions(r)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, positions.CloseChecked()) })
	xs := variableHelperArray([]uint32{1, 4})
	sp, e := NewScopedPolynomialVariablePositions(positions, 2, xs)
	serviceMust(t, e)
	sp2, e := NewScopedPolynomialVariablePositionsMove(sp)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, sp.CloseChecked()); serviceMust(t, sp2.CloseChecked()) })
	serviceMust(t, sp.SourceFinalize())
	e = sp2.SourceFinalize()
	if polynomialSourceDebug {
		if !errors.Is(e, ErrPolynomialAssertion) {
			t.Fatal("reference copy repeats literal source reset guard", e)
		}
	} else {
		serviceMust(t, e)
	}
	c, e := NewPolynomialVariablePositionsCopy(positions)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.CloseChecked()) })
	serviceMust(t, c.MoveFrom(positions))
	want, e := positions.Position(1)
	serviceMust(t, e)
	c.positions.values[1] = 99
	v, e = positions.Position(1)
	serviceMust(t, e)
	if v != want {
		t.Fatal("member rvalue assignment shared backing", v)
	}
	cm, e := NewPolynomialVariablePositionsMove(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, cm.CloseChecked()) })
	if len(c.positions.values) != 0 || len(cm.positions.values) == 0 {
		t.Fatal("real move ctor")
	}
}

func TestPolynomialVariableHelperScopedReachedResetPrefix(t *testing.T) {
	polynomialThrowAssertionFixture(t)
	r, b := degreeCopyRuntime(t)
	d, e := NewPolynomialVariablePositions(r)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, d.CloseChecked()) })
	variables := []uint32{1, 4}
	xs := variableHelperArray(variables)
	s, e := NewScopedPolynomialVariablePositions(d, 2, xs)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.CloseChecked()) })
	// The caller's original mutable array remains borrowed by const pointer.
	// Changing its second value to UINT_MAX reaches a guarded source-undefined
	// destination index; the completed first reset must remain published.
	variables[1] = ^uint32(0)
	e = s.SourceFinalize()
	if e == nil {
		t.Fatal("malformed borrowed view was accepted")
	}
	v, e := d.Position(1)
	serviceMust(t, e)
	if v != ^uint32(0) {
		t.Fatal("first reset not published", v)
	}
	v, e = d.Position(4)
	serviceMust(t, e)
	if v != 1 {
		t.Fatal("unreached old variable untouched", v)
	}
	roots := uint32(0)
	if r.Stats().LiveBlocks != roots || r.Stats().TotalBytes+d.header+s.header != b.Used() {
		t.Fatal("source-stopped scope and borrowed/caller roots", r.Stats(), b.Used())
	}
	// Separate owning metadata abort acknowledges no second source reset and does
	// not mutate the borrowed array or helper. The caller still owns both.
	serviceMust(t, s.CloseChecked())
	v, e = d.Position(4)
	serviceMust(t, e)
	if v != 1 {
		t.Fatal(v)
	}
}
func TestPolynomialMaxDegreesCopyConstructorNativeMembers(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{3, 5}, [][]MonomialPower{{{0, 2}}, {{1, 4}}})
	a, e := NewPolynomialMaxDegrees(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, a.CloseChecked()) })
	serviceMust(t, a.Init(p))
	c, e := NewPolynomialMaxDegreesCopy(a)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.CloseChecked()) })
	serviceMust(t, c.maxDegree.SetDegree(1, 9))
	v, e := a.Degree(1)
	serviceMust(t, e)
	if v != 4 {
		t.Fatal("copied member shares backing", v)
	}
}
func TestPolynomialMaxDegreesCopyConstructorNativeMembersIgnoreTLSLimit(t *testing.T) {
	b := arena.NewBudget(1 << 25)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true})
	serviceMust(t, e)
	a, e := newPolynomialMaxDegrees(nil, r, true)
	serviceMust(t, e)
	serviceMust(t, a.maxDegree.SetDegree(250, 17))
	serviceMust(t, a.variables.SetDegree(250, 9))
	serviceMust(t, r.SetMaxSize(1))
	c, e := NewPolynomialMaxDegreesCopy(a)
	serviceMust(t, e)
	v, e := c.maxDegree.Degree(250)
	serviceMust(t, e)
	if v != 17 || r.Stats().LiveBlocks != 0 {
		t.Fatal("native member copy used Runtime backing", v, r.Stats())
	}
	serviceMust(t, c.CloseChecked())
	serviceMust(t, a.CloseChecked())
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestPolynomialVariableHelperWarmAndColdOwnerGuards(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{3, 5}, [][]MonomialPower{{{0, 2}}, {{1, 4}}})
	d, e := NewPolynomialMaxDegrees(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, d.CloseChecked()) })
	serviceMust(t, d.Init(p))
	serviceMust(t, d.Reset())
	beforeStats, beforeState, beforeUsed := m.memory.Stats(), m.memory.State(), m.integer.Budget().Used()
	before, e := m.memory.AllocationCount()
	serviceMust(t, e)
	m.integer.Budget().FailAfter(0)
	allocs := testing.AllocsPerRun(25, func() {
		if e := d.Init(p); e != nil {
			panic(e)
		}
		if v, e := d.Degree(1); e != nil || v != 4 {
			panic("degree")
		}
		if e := d.Reset(); e != nil {
			panic(e)
		}
	})
	polynomialVariablePhysical(t, m, []*PolynomialMaxDegrees{d}, nil, nil, nil, nil, []Polynomial{p})
	after, e := m.memory.AllocationCount()
	serviceMust(t, e)
	if allocs != 0 || beforeStats != m.memory.Stats() || beforeState != m.memory.State() || beforeUsed != m.integer.Budget().Used() || before != after {
		t.Fatal("warmed real retained two vectors", allocs)
	}
	m.integer.Budget().FailAfter(-1)
	shallow := *d
	if e = shallow.CloseChecked(); !errors.Is(e, ErrPolynomialOwner) {
		t.Fatal("cold-copy owner guard", e)
	}
	serviceMust(t, d.SourceFinalize())
	if _, e = d.Degree(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestPolynomialMaxDegreesSourceFinalizeNativeMembers(t *testing.T) {
	b := arena.NewBudget(1 << 25)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	d, e := newPolynomialMaxDegrees(nil, r, true)
	serviceMust(t, e)
	serviceMust(t, d.maxDegree.SetDegree(250, 17))
	serviceMust(t, d.variables.SetDegree(250, 9))
	serviceMust(t, r.SetMaxSize(1))
	serviceMust(t, d.SourceFinalize())
	if _, e = d.Degree(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("finalized source lifetime", e)
	}
	serviceMust(t, d.CloseChecked())
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("Runtime cold owner refund", b.Used())
	}
}
