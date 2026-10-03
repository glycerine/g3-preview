// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"testing"
)

func TestUPolynomialIsolationAllDispatchRoutesIndependent(t *testing.T) {
	m, q, budget := upolySourceOwner(t)
	b := upolyIsolationBinaryOwner(t, q, budget)
	roots, lo, hi := upolyIsolationVector(t, b), upolyIsolationVector(t, b), upolyIsolationVector(t, b)
	original := upolyIsolationProduct(t, m, [][]int32{{-1, 1}, {-2, 1}, {-3, 1}})
	scratch := upolySourceNew(t, m, nil)
	for route := 0; route < 5; route++ {
		if e := m.Copy(scratch, original); e != nil {
			t.Fatal(e)
		}
		for _, v := range []*mpnum.BinaryRationalVector{roots, lo, hi} {
			if e := v.Reset(); e != nil {
				t.Fatal(e)
			}
		}
		v := mustUPolyView(t, m, scratch)
		var e error
		switch route {
		case 0:
			e = m.DRSIsolateRoots(v, b, roots, lo, hi)
		case 1:
			e = m.DRSIsolateRootsWithBounds(v, 3, 3, b, roots, lo, hi)
		case 2:
			e = m.SturmIsolateRoots(v, b, roots, lo, hi)
		case 3:
			e = m.SturmIsolateRootsWithBounds(v, 3, 3, b, roots, lo, hi)
		case 4:
			e = m.SquareFreeIsolateRoots(v, b, roots, lo, hi)
		}
		if e != nil {
			t.Fatal(route, e)
		}
		c := upolyIsolationCounts{}
		upolyIsolationMatch(t, q, b, roots, lo, hi, []upolyIsolationRational{{1, 1}, {2, 1}, {3, 1}}, &c)
	}
}
func TestUPolynomialBinaryVectorOwnershipResizeReserveAndPlateau(t *testing.T) {
	m, q, budget := upolySourceOwner(t)
	b := upolyIsolationBinaryOwner(t, q, budget)
	v := upolyIsolationVector(t, b)
	a := mpnum.NewBinaryInt(7)
	if e := v.PushCopy(&a); e != nil {
		t.Fatal(e)
	}
	// vector::reserve actually calls resize when needed: logical length grows,
	// unlike Go's conventional capacity-only reserve.
	if e := v.Reserve(4); e != nil {
		t.Fatal(e)
	}
	if upolyIsolationSize(t, v) != 4 {
		t.Fatal("source reserve size")
	}
	for i := uint32(1); i < 4; i++ {
		value := upolyIsolationRead(t, b, v, i)
		zero, e := b.IsZero(&value)
		if e != nil || !zero {
			t.Fatal(i, e)
		}
		if e = b.Del(&value); e != nil {
			t.Fatal(e)
		}
	}
	if e := v.Set(3, &a); e != nil {
		t.Fatal(e)
	}
	if e := v.Shrink(1); e != nil {
		t.Fatal(e)
	}
	if e := v.Resize(3); e != nil {
		t.Fatal(e)
	}
	value := upolyIsolationRead(t, b, v, 0)
	numerator := mpnum.NewInt(7)
	same, e := b.EqualInteger(&value, &numerator)
	if e != nil || !same {
		t.Fatal(e)
	}
	if e = b.Del(&value); e != nil {
		t.Fatal(e)
	}
	before := budget.Used()
	budget.FailAfter(0)
	for i := 0; i < 50; i++ {
		if e = v.Reset(); e != nil {
			t.Fatal(e)
		}
		if e = v.PushCopy(&a); e != nil {
			t.Fatal(e)
		}
	}
	if budget.Used() != before {
		t.Fatal("warm physical plateau", before, budget.Used())
	}
	budget.FailAfter(-1)
	// Owning cold copy cannot duplicate array/numeral cleanup rights.
	copied := *v
	if e = copied.Reset(); e == nil {
		t.Fatal("copied owner accepted")
	}
	_ = m // real numerical owner is shared; no synthetic callbacks
}
func TestUPolynomialBinaryVectorConstructionAndPushFailurePrefixes(t *testing.T) {
	m, q, budget := upolySourceOwner(t)
	b := upolyIsolationBinaryOwner(t, q, budget)
	before := budget.Used()
	budget.FailAfter(0)
	v, e := b.NewVector()
	if !errors.Is(e, arena.ErrInjectedFailure) || v != nil || budget.Used() != before {
		t.Fatal("constructor prefix", v, e, before, budget.Used())
	}
	budget.FailAfter(-1)
	v = upolyIsolationVector(t, b)
	a := mpnum.NewBinaryInt(1)
	budget.FailAfter(0)
	e = v.PushCopy(&a)
	if !errors.Is(e, arena.ErrInjectedFailure) || upolyIsolationSize(t, v) != 0 {
		t.Fatal("push grow prefix", e)
	}
	budget.FailAfter(-1)
	if e = v.PushCopy(&a); e != nil {
		t.Fatal(e)
	}
	// Append-only lower-level isolation does not reset an existing exact root.
	p := upolySourceNew(t, m, []int32{1})
	lo, hi := upolyIsolationVector(t, b), upolyIsolationVector(t, b)
	if e = m.DRSIsolateRoots(mustUPolyView(t, m, p), b, v, lo, hi); e != nil {
		t.Fatal(e)
	}
	if upolyIsolationSize(t, v) != 1 {
		t.Fatal("unexpected core reset")
	}
}

func TestUPolynomialBinaryVectorBigCopyFaultImmediateRoots(t *testing.T) {
	failures, successes := 0, 0
	for fault := int64(0); fault < 64; fault++ {
		m, q, budget := upolySourceOwner(t)
		b := upolyIsolationBinaryOwner(t, q, budget)
		v := upolyIsolationVector(t, b)
		var bigValue mpnum.Int
		limbs := make([]uint32, 90)
		limbs[0] = 37
		limbs[89] = 1
		if e := m.IntegerManager().SetLimbs(&bigValue, limbs); e != nil {
			t.Fatal(e)
		}
		a := mpnum.NewBinaryRational()
		if e := b.SetInteger(&a, &bigValue); e != nil {
			t.Fatal(e)
		}
		if e := m.IntegerManager().Del(&bigValue); e != nil {
			t.Fatal(e)
		}
		budget.FailAfter(fault)
		e := v.PushCopy(&a)
		graph, vec := m.Stats(), v.Stats()
		binary, e2 := b.Stats()
		if e2 != nil || graph.IntegerLiveCells != graph.IntegerCachedCells+graph.OwnedNumeralCells+vec.OwnedNumeralCells+binary.LiveScratchNumerators+1 || graph.IntegerWordsRefs != graph.IntegerLiveCells+graph.IntegerNaturalWordsRefs {
			t.Fatal("immediate binary output graph", fault, e, e2, graph, vec, binary)
		}
		budget.FailAfter(-1)
		if e != nil {
			failures++
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(fault, e)
			}
		} else {
			successes++
		}
		if e = v.Reset(); e != nil {
			t.Fatal(fault, e)
		}
		graph, vec = m.Stats(), v.Stats()
		binary, e2 = b.Stats()
		if e2 != nil || vec.OwnedNumeralCells != 0 || graph.IntegerLiveCells != graph.IntegerCachedCells+graph.OwnedNumeralCells+binary.LiveScratchNumerators+1 {
			t.Fatal("failed companion cleanup", fault, e2, graph, vec)
		}
		if e = b.Del(&a); e != nil {
			t.Fatal(e)
		}
		if successes != 0 {
			break
		}
	}
	if failures == 0 || successes == 0 {
		t.Fatal(failures, successes)
	}
}

func TestUPolynomialIsolationRemoveZeroExistingPublicRegression(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{0, 0, 6, -3})
	out := upolySourceNew(t, m, []int32{1, 1, 1})
	if e := m.RemoveZeroRoots(mustUPolyView(t, m, p), out); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, out, []int64{6, -3})
	if e := m.SetInts(out, []mpnum.Int{mpnum.NewInt(1), mpnum.NewInt(1), mpnum.NewInt(1)}); e != nil {
		t.Fatal(e)
	}
	e := m.RemoveZeroRoots(mustUPolyView(t, m, p).WithCount(5), out)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("late source raw read", e)
	}
	upolyGCDExpect(t, m, out, []int64{6, -3, 1})
	if e = m.RemoveZeroRoots(mustUPolyView(t, m, p), p); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, p, []int64{6, -3})
}

func TestUPolynomialIsolationDefinedOutputAliases(t *testing.T) {
	m, q, budget := upolySourceOwner(t)
	b := upolyIsolationBinaryOwner(t, q, budget)
	p := upolySourceNew(t, m, []int32{-2, 0, 1})
	for _, f := range []struct {
		sturm  bool
		shared []int64
		upper  []int64
	}{{false, []int64{0, -4}, []int64{4, 0}}, {true, []int64{0, 0, 0, 4}, nil}} {
		shared, other := upolyIsolationVector(t, b), upolyIsolationVector(t, b)
		var e error
		if f.sturm {
			e = m.SturmIsolateRoots(mustUPolyView(t, m, p), b, other, shared, shared)
		} else {
			e = m.DRSIsolateRoots(mustUPolyView(t, m, p), b, shared, shared, other)
		}
		if e != nil {
			t.Fatal(f.sturm, e)
		}
		for _, x := range []struct {
			v    *mpnum.BinaryRationalVector
			want []int64
		}{{shared, f.shared}, {other, f.upper}} {
			if upolyIsolationSize(t, x.v) != uint32(len(x.want)) {
				t.Fatal(f.sturm, x.want, x.v.Stats())
			}
			for i, n := range x.want {
				a := upolyIsolationRead(t, b, x.v, uint32(i))
				want := mpnum.NewInt(int32(n))
				same, e := b.EqualInteger(&a, &want)
				if e != nil || !same {
					t.Fatal(f.sturm, i, n, e)
				}
				if e = b.Del(&a); e != nil {
					t.Fatal(e)
				}
			}
		}
	}
}
func TestUPolynomialIsolationBoundedFaultAndCancelImmediateGraph(t *testing.T) {
	for route := 0; route < 2; route++ {
		failures, successes := 0, 0
		for fault := int64(0); fault < 256; fault++ {
			m, q, budget := upolySourceOwner(t)
			b := upolyIsolationBinaryOwner(t, q, budget)
			r, l, u := upolyIsolationVector(t, b), upolyIsolationVector(t, b), upolyIsolationVector(t, b)
			p := upolyIsolationProduct(t, m, [][]int32{{-1, 4}, {-3, 4}})
			budget.FailAfter(fault)
			var e error
			if route == 0 {
				e = m.DRSIsolateRootsWithBounds(mustUPolyView(t, m, p), 128, 128, b, r, l, u)
			} else {
				e = m.SturmIsolateRootsWithBounds(mustUPolyView(t, m, p), 128, 128, b, r, l, u)
			}
			graph := m.Stats()
			scratch, e2 := b.Stats()
			roots := r.Stats().OwnedNumeralCells + l.Stats().OwnedNumeralCells + u.Stats().OwnedNumeralCells
			if e2 != nil || graph.IntegerLiveCells != graph.IntegerCachedCells+graph.OwnedNumeralCells+scratch.LiveScratchNumerators+roots || graph.IntegerWordsRefs != graph.IntegerLiveCells+graph.IntegerNaturalWordsRefs {
				t.Fatal("immediate isolation graph", route, fault, e, e2, graph, scratch, roots)
			}
			budget.FailAfter(-1)
			if e != nil {
				failures++
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal(route, fault, e)
				}
			} else {
				successes++
				counts := upolyIsolationCounts{}
				upolyIsolationMatch(t, q, b, r, l, u, []upolyIsolationRational{{1, 4}, {3, 4}}, &counts)
			}
			for _, v := range []*mpnum.BinaryRationalVector{r, l, u} {
				if e = v.Reset(); e != nil {
					t.Fatal(route, fault, e)
				}
			}
			graph = m.Stats()
			scratch, e2 = b.Stats()
			if e2 != nil || graph.IntegerLiveCells != graph.IntegerCachedCells+graph.OwnedNumeralCells+scratch.LiveScratchNumerators {
				t.Fatal("lost local/frame", route, fault, e2, graph)
			}
			if successes != 0 {
				break
			}
		}
		if failures == 0 || successes == 0 {
			t.Fatal(route, failures, successes)
		}
		t.Log("real source refusal sweep", route, failures, successes)
	}
	m, q, budget, limit, _ := upolySourceOwnerServices(t)
	bqm := upolyIsolationBinaryOwner(t, q, budget)
	p := upolySourceNew(t, m, []int32{-2, 0, 1})
	a, b := mpnum.NewBinaryInt(1), mpnum.NewBinaryInt(2)
	limit.incCancel()
	steps := limit.steps
	_, e := m.RefineToPrecision(mustUPolyView(t, m, p), bqm, &a, &b, 40)
	graph := m.Stats()
	scratch, e2 := bqm.Stats()
	if !errors.Is(e, mpnum.ErrUPolynomialCanceled) || e2 != nil || limit.steps != steps+1 || graph.IntegerLiveCells != graph.IntegerCachedCells+graph.OwnedNumeralCells+scratch.LiveScratchNumerators {
		t.Fatal("source pre-width cancel", e, e2, steps, limit.steps, graph)
	}
	one, two := mpnum.NewInt(1), mpnum.NewInt(2)
	left, e := bqm.EqualInteger(&a, &one)
	if e != nil {
		t.Fatal(e)
	}
	right, e := bqm.EqualInteger(&b, &two)
	if e != nil || !left || !right {
		t.Fatal("cancel changed endpoints", left, right, e)
	}
	if e = bqm.Del(&a); e != nil {
		t.Fatal(e)
	}
	if e = bqm.Del(&b); e != nil {
		t.Fatal(e)
	}
}

func TestUPolynomialIsolationZeroResetPublicationBeforeRefusal(t *testing.T) {
	m, q, budget := upolySourceOwner(t)
	b := upolyIsolationBinaryOwner(t, q, budget)
	r, l, u := upolyIsolationVector(t, b), upolyIsolationVector(t, b), upolyIsolationVector(t, b)
	one := mpnum.NewBinaryInt(1)
	for _, v := range []*mpnum.BinaryRationalVector{r, l, u} {
		if e := v.PushCopy(&one); e != nil {
			t.Fatal(e)
		}
	}
	p := upolySourceNew(t, m, []int32{0, -2, 0, 1})
	budget.FailAfter(0)
	e := m.SquareFreeIsolateRoots(mustUPolyView(t, m, p), b, r, l, u)
	graph := m.Stats()
	scratch, e2 := b.Stats()
	if !errors.Is(e, arena.ErrInjectedFailure) || e2 != nil || graph.IntegerLiveCells != graph.IntegerCachedCells+graph.OwnedNumeralCells+scratch.LiveScratchNumerators+r.Stats().OwnedNumeralCells+l.Stats().OwnedNumeralCells+u.Stats().OwnedNumeralCells || upolyIsolationSize(t, r) != 1 || upolyIsolationSize(t, l) != 0 || upolyIsolationSize(t, u) != 0 {
		t.Fatal("reset roots/lower/upper and publish zero before nz backing refusal", e, e2, graph, r.Stats(), l.Stats(), u.Stats())
	}
	budget.FailAfter(-1)
	value := upolyIsolationRead(t, b, r, 0)
	zero, e := b.IsZero(&value)
	if e != nil || !zero {
		t.Fatal(e, zero)
	}
	if e = b.Del(&value); e != nil {
		t.Fatal(e)
	}
}
