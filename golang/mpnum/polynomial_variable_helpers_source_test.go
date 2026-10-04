// Source: src/math/polynomial/polynomial.cpp:6179-6232,6307-6344.
// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func variableHelperArray(t *testing.T, r *memory.Runtime, values []uint32) (PolynomialVariablesView, *memory.Block) {
	t.Helper()
	raw, e := r.Allocate(uint64(len(values)) * 4)
	serviceMust(t, e)
	for i, v := range values {
		serviceMust(t, r.WriteUint32(raw, uint64(i)*4, v))
	}
	t.Cleanup(func() { serviceMust(t, r.Deallocate(&raw)) })
	return BorrowPolynomialVariables(r, raw, 0), &raw
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
	r, _ := degreeCopyRuntime(t)
	d, e := NewPolynomialVariablePositions(r)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, d.CloseChecked()) })
	xs, _ := variableHelperArray(t, r, []uint32{2, 5, 2})
	e = d.Init(3, xs)
	if modularSourceDebug {
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
	if modularSourceDebug {
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
	if modularSourceDebug {
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
func TestScopedPolynomialVariablePositionsCapturedBackingLifetime(t *testing.T) {
	r, _ := degreeCopyRuntime(t)
	d, e := NewPolynomialVariablePositions(r)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, d.CloseChecked()) })
	xs, raw := variableHelperArray(t, r, []uint32{1, 4})
	s, e := NewScopedPolynomialVariablePositions(d, 2, xs)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.CloseChecked()) })
	serviceMust(t, r.Reallocate(raw, 24))
	e = s.SourceFinalize()
	if !errors.Is(e, memory.ErrRuntimeOwner) {
		t.Fatal("captured raw pointer after realloc", e)
	}
	for x, w := range map[uint32]uint32{1: 0, 4: 1} {
		v, e := d.Position(x)
		serviceMust(t, e)
		if v != w {
			t.Fatal(x, v, w)
		}
	}
	if e = s.SourceFinalize(); !errors.Is(e, memory.ErrRuntimeOwner) {
		t.Fatal(e)
	}
}
func TestScopedPolynomialVariablePositionsFailedConstructorNoReset(t *testing.T) {
	r, _ := degreeCopyRuntime(t)
	d, e := NewPolynomialVariablePositions(r)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, d.CloseChecked()) })
	xs, _ := variableHelperArray(t, r, []uint32{1, 1})
	s, e := NewScopedPolynomialVariablePositions(d, 2, xs)
	if modularSourceDebug {
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

func TestPolynomialMaxDegreesFirstSeenReservationPrefixesAndFailedScope(t *testing.T) {
	failures, success, sawFirstSeen := 0, false, false
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		done := false
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialCellFixture(t)
			for x := uint32(3); x <= 10; x++ {
				_, e := m.monomials.MkVar()
				serviceMust(t, e)
			}
			p := polynomialQueryTerms(t, m, []int32{3, 5}, [][]MonomialPower{{{10, 4}}, {{0, 2}}})
			d, e := NewPolynomialMaxDegrees(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, d.CloseChecked()) })
			b := m.integer.budget
			b.FailAfter(ordinal)
			scope, e := NewScopedPolynomialMaxDegrees(d, p)
			polynomialVariablePhysical(t, m, []*PolynomialMaxDegrees{d}, nil, []*ScopedPolynomialMaxDegrees{scope}, nil, nil, []Polynomial{p})
			n, x := d.NumVariables()
			serviceMust(t, x)
			mapSize, x := d.maxDegree.size()
			serviceMust(t, x)
			if e != nil {
				if !numericPhysicalReservationRefusal(e) || scope != nil {
					t.Fatal("actual scoped ctor refusal/no completed dtor", ordinal, e, scope)
				}
				failures++
				if n == 1 && mapSize == 0 {
					sawFirstSeen = true
				}
				if mapSize != 0 && mapSize != 11 {
					t.Fatal("all reserve growth before size", ordinal, mapSize)
				}
			} else {
				success = true
				done = true
				if n != 2 {
					t.Fatal(n)
				}
				v, e := scope.Degree(10)
				serviceMust(t, e)
				if v != 4 {
					t.Fatal(v)
				}
			}
			b.FailAfter(-1)
			if scope != nil {
				serviceMust(t, scope.CloseChecked())
			}
			polynomialVariablePhysical(t, m, []*PolynomialMaxDegrees{d}, nil, nil, nil, nil, []Polynomial{p})
		})
		if done {
			break
		}
	}
	if !success || failures == 0 || !sawFirstSeen {
		t.Fatal("source xs push precedes setx allocation", failures, success, sawFirstSeen)
	}
}
func TestPolynomialVariablePositionsEveryReservationKeepsOldSizeUntilFill(t *testing.T) {
	failures, success := 0, false
	for ordinal := int64(0); ordinal < 48; ordinal++ {
		done := false
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			r, b := degreeCopyRuntime(t)
			d, e := NewPolynomialVariablePositions(r)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, d.CloseChecked()) })
			xs, raw := variableHelperArray(t, r, []uint32{19})
			b.FailAfter(ordinal)
			e = d.Init(1, xs)
			roots := uint32(1)
			if d.positions.raw != (memory.Block{}) {
				roots++
			}
			if d.positions.pending != (memory.Block{}) {
				roots++
			}
			if r.Stats().LiveBlocks != roots || r.Stats().TotalBytes+d.header != b.Used() {
				t.Fatal("named position/input roots before fault disable", r.Stats(), d.header, b.Used(), *raw)
			}
			n, x := d.positions.size()
			serviceMust(t, x)
			if e != nil {
				if !numericPhysicalReservationRefusal(e) || n != 0 {
					t.Fatal(ordinal, e, n)
				}
				failures++
			} else {
				success = true
				done = true
				if n != 20 {
					t.Fatal(n)
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
			b.FailAfter(-1)
		})
		if done {
			break
		}
	}
	if !success || failures == 0 {
		t.Fatal(failures, success)
	}
}
func TestPolynomialVariableHelperImplicitCopyMoveAndScopedCopies(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{7, 11}, [][]MonomialPower{{{0, 3}}, {{1, 4}}})
	a, e := NewPolynomialMaxDegrees(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, a.CloseChecked()) })
	serviceMust(t, a.Init(p))
	b, e := NewPolynomialMaxDegreesCopy(a)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, b.CloseChecked()) })
	if a.variables.raw == b.variables.raw || a.maxDegree.raw == b.maxDegree.raw {
		t.Fatal("two actual deep-copied members")
	}
	serviceMust(t, b.Reset())
	v, e := a.Degree(1)
	serviceMust(t, e)
	if v != 4 {
		t.Fatal(v)
	}
	serviceMust(t, b.MoveFrom(a))
	if a.variables.raw == b.variables.raw {
		t.Fatal("rvalue assignment is member copy")
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
	xs, _ := variableHelperArray(t, r, []uint32{1, 4})
	sp, e := NewScopedPolynomialVariablePositions(positions, 2, xs)
	serviceMust(t, e)
	sp2, e := NewScopedPolynomialVariablePositionsMove(sp)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, sp.CloseChecked()); serviceMust(t, sp2.CloseChecked()) })
	serviceMust(t, sp.SourceFinalize())
	e = sp2.SourceFinalize()
	if modularSourceDebug {
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
	if c.positions.raw == positions.positions.raw {
		t.Fatal("member rvalue assignment copies")
	}
	cm, e := NewPolynomialVariablePositionsMove(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, cm.CloseChecked()) })
	if c.positions.raw != (memory.Block{}) || cm.positions.raw == (memory.Block{}) {
		t.Fatal("real move ctor")
	}
}

func TestPolynomialVariableHelperScopedReachedResetPrefix(t *testing.T) {
	r, b := degreeCopyRuntime(t)
	d, e := NewPolynomialVariablePositions(r)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, d.CloseChecked()) })
	xs, raw := variableHelperArray(t, r, []uint32{1, 4})
	s, e := NewScopedPolynomialVariablePositions(d, 2, xs)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.CloseChecked()) })
	// The caller's original mutable array remains borrowed by const pointer.
	// Changing its second value to UINT_MAX reaches a guarded source-undefined
	// destination index; the completed first reset must remain published.
	serviceMust(t, r.WriteUint32(*raw, 4, ^uint32(0)))
	e = s.SourceFinalize()
	want := error(arena.ErrBounds)
	if modularSourceDebug {
		want = ErrPolynomialAssertion
	}
	if !errors.Is(e, want) {
		t.Fatal(e, want)
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
	roots := uint32(2)
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
func TestPolynomialMaxDegreesCopyConstructorCompletedMemberUnwind(t *testing.T) {
	failures, success, sawUnwind := 0, false, false
	for ordinal := int64(0); ordinal < 24; ordinal++ {
		done := false
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialCellFixture(t)
			p := polynomialQueryTerms(t, m, []int32{3, 5}, [][]MonomialPower{{{0, 2}}, {{1, 4}}})
			a, e := NewPolynomialMaxDegrees(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, a.CloseChecked()) })
			serviceMust(t, a.Init(p))
			before, e := m.memory.AllocationCount()
			serviceMust(t, e)
			m.integer.budget.FailAfter(ordinal)
			c, e := NewPolynomialMaxDegreesCopy(a)
			polynomialVariablePhysical(t, m, []*PolynomialMaxDegrees{a, c}, nil, nil, nil, nil, []Polynomial{p})
			after, x := m.memory.AllocationCount()
			serviceMust(t, x)
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal(e)
				}
				failures++
				if after == before+2 && c == nil {
					sawUnwind = true
				}
				if c != nil {
					if c.constructed {
						t.Fatal("failed complete class lifetime")
					}
					if c.variables.constructed {
						t.Fatal("second member unexpectedly complete")
					}
					if c.maxDegree.constructed && c.maxDegree.raw != (memory.Block{}) {
						t.Fatal("completed first member not source-unwound", c.maxDegree.raw)
					}
				}
			} else {
				success = true
				done = true
				if after != before+2 {
					t.Fatal("two source allocation bodies", before, after)
				}
			}
			m.integer.budget.FailAfter(-1)
			if c != nil {
				serviceMust(t, c.CloseChecked())
			}
		})
		if done {
			break
		}
	}
	if !success || !sawUnwind || failures < 2 {
		t.Fatal(failures, success, sawUnwind)
	}
}
func TestPolynomialMaxDegreesCopyConstructorTLSIncompleteSecondRoot(t *testing.T) {
	b := arena.NewBudget(1 << 25)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true})
	serviceMust(t, e)
	// Real standalone class member-lifetime fixture: both initialized uint arrays
	// are created by the actual vector API. Their contents are not advertised as
	// a mathematical Init result, and no polynomial manager algorithm is replaced.
	a, e := newPolynomialMaxDegrees(nil, r, true)
	serviceMust(t, e)
	serviceMust(t, a.maxDegree.SetDegree(25000, 17))
	serviceMust(t, a.variables.SetDegree(25000, 9))
	cap, e := a.maxDegree.capacity()
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(uint64(r.State().GlobalSize)+uint64(cap)*4+16))
	c, e := NewPolynomialMaxDegreesCopy(a)
	var f memory.Failure
	if !errors.As(e, &f) || f.Kind != memory.FailureMemout || c == nil || c.constructed || !c.maxDegree.constructed || c.variables.constructed || c.variables.pending == (memory.Block{}) || c.maxDegree.raw != (memory.Block{}) {
		t.Fatal("completed first member unwind before partial second owning abort", e, c)
	}
	if r.Stats().LiveBlocks != 3 || r.Stats().TotalBytes+a.header+c.header != b.Used() {
		t.Fatal("actual caller2 + partial second1", r.Stats(), b.Used())
	}
	if _, e = c.NumVariables(); !errors.Is(e, ErrPolynomialPending) {
		t.Fatal("failed ctor carrier ordinary read", e)
	}
	pending := c.variables.pending
	serviceMust(t, c.SourceFinalize())
	if c.variables.pending != pending {
		t.Fatal("source dtor of uncompleted second")
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
	beforeStats, beforeState, beforeUsed := m.memory.Stats(), m.memory.State(), m.integer.budget.Used()
	before, e := m.memory.AllocationCount()
	serviceMust(t, e)
	m.integer.budget.FailAfter(0)
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
	if allocs != 0 || beforeStats != m.memory.Stats() || beforeState != m.memory.State() || beforeUsed != m.integer.budget.Used() || before != after {
		t.Fatal("warmed real retained two vectors", allocs)
	}
	m.integer.budget.FailAfter(-1)
	shallow := *d
	if e = shallow.CloseChecked(); !errors.Is(e, ErrPolynomialOwner) {
		t.Fatal("cold-copy owner guard", e)
	}
	serviceMust(t, d.SourceFinalize())
	if _, e = d.Degree(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestPolynomialMaxDegreesStoppedSourceDestructorAndOwningAbort(t *testing.T) {
	b := arena.NewBudget(1 << 25)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	d, e := newPolynomialMaxDegrees(nil, r, true)
	serviceMust(t, e)
	// Explicit source-member lifetime fixture in the guarded host/signed-overflow
	// domain. Real initialized arrays and public TLS history; no counter repair.
	serviceMust(t, d.maxDegree.SetDegree(25000, 17))
	serviceMust(t, d.variables.SetDegree(25000, 9))
	older := d.maxDegree.raw
	contextTLSNearMinimum(t, r)
	e = d.SourceFinalize()
	if !errors.Is(e, arena.ErrBounds) || d.variables.raw != (memory.Block{}) || d.maxDegree.raw != older || d.sourceStage != 0 {
		t.Fatal("younger physically consumed then source stop before older", e, d.sourceStage)
	}
	stats, state := r.Stats(), r.State()
	if stats.LiveBlocks != 1 || stats.TotalBytes+d.header != b.Used() {
		t.Fatal("exact remaining older root and cold owner", stats, b.Used())
	}
	if e = d.SourceFinalize(); !errors.Is(e, arena.ErrBounds) || r.Stats() != stats || r.State() != state || d.maxDegree.raw != older {
		t.Fatal("ordinary destruction must not acknowledge or replay", e)
	}
	if _, e = d.Degree(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("stopped source lifetime", e)
	}
	for i := 0; i < 4 && !d.closed; i++ {
		e = d.CloseChecked()
		if e != nil && !errors.Is(e, arena.ErrBounds) {
			t.Fatal("only reached real cleanup guard", e)
		}
	}
	if !d.closed || r.Stats().LiveBlocks != 0 || r.Stats().PhysicalBytes != 0 || r.Stats().TotalBytes != b.Used() {
		t.Fatal("owning abort drains actual roots", r.Stats(), b.Used())
	}
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("Runtime cold owner refund", b.Used())
	}
}
