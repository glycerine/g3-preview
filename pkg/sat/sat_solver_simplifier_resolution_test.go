package sat

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func satKernelOutput(t *testing.T, b *arena.Budget) (*satSolverLiteralStore, satSolverLiteralListRef) {
	t.Helper()
	s, e := newSATSolverLiteralStore(b)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, s.Close()) })
	r, e := s.NewList()
	satOK(t, e)
	return s, r
}
func satKernelOutputWant(t *testing.T, s *satSolverLiteralStore, r satSolverLiteralListRef, want []Literal) {
	t.Helper()
	n, e := s.Size(r)
	satOK(t, e)
	if n != uint32(len(want)) {
		t.Fatal(n, want)
	}
	for i, v := range want {
		got, e := s.Read(r, uint32(i))
		satOK(t, e)
		if got != v {
			t.Fatal(i, got, v)
		}
	}
}
func TestSATSimplifierResolutionWrapperAndRawDuplicateActions(t *testing.T) {
	k, a, _, b := satSimplifierKernelFixture(t, 4)
	out, r := satKernelOutput(t, b)
	c1 := satMk(t, a, satLs(0, 2, 4), false)
	c2 := satMk(t, a, satLs(1, 4, 6, 6), false)
	satOK(t, out.Push(r, 7))
	k.elimCounter = 20
	yes, e := k.resolve(newSATClauseWrapper(c1), newSATClauseWrapper(c2), 0, out, r)
	satOK(t, e)
	if !yes || k.elimCounter != 13 {
		t.Fatal(yes, k.elimCounter)
	}
	satKernelOutputWant(t, out, r, satLs(7, 2, 4, 6, 6))
	satOK(t, out.Shrink(r, 0))
	yes, e = k.resolve(newSATBinaryWrapper(0, 2), newSATBinaryWrapper(1, 4), 0, out, r)
	satOK(t, e)
	if !yes {
		t.Fatal(yes)
	}
	satKernelOutputWant(t, out, r, satLs(2, 4))
	satOK(t, out.Shrink(r, 0))
	yes, e = k.resolve(newSATClauseWrapper(c1), newSATBinaryWrapper(1, 3), 0, out, r)
	satOK(t, e)
	if yes {
		t.Fatal("tautology")
	}
	satKernelOutputWant(t, out, r, satLs(2, 4))
	for l := Literal(0); l < 8; l++ {
		yes, e := k.isMarked(l)
		if e != nil || yes {
			t.Fatal("normal cleanup", l, yes, e)
		}
	}
}
func TestSATSimplifierResolutionIndependentBoundedTruth(t *testing.T) {
	k, a, _, b := satSimplifierKernelFixture(t, 3)
	out, r := satKernelOutput(t, b)
	for left := uint32(0); left < 16; left++ {
		for right := uint32(0); right < 16; right++ {
			ls := []Literal{0}
			rs := []Literal{1}
			for i := uint32(0); i < 4; i++ {
				if left&(1<<i) != 0 {
					ls = append(ls, Literal(i+2))
				}
				if right&(1<<i) != 0 {
					rs = append(rs, Literal(i+2))
				}
			}
			c1 := satKernelRawClause(t, a, ls)
			c2 := satKernelRawClause(t, a, rs)
			satOK(t, out.Shrink(r, 0))
			yes, e := k.resolve(newSATClauseWrapper(c1), newSATClauseWrapper(c2), 0, out, r)
			satOK(t, e)
			// Complement crossing the two inputs determines source tautology status.
			want := true
			for i := uint32(0); i < 4; i++ {
				if left&(1<<i) != 0 && right&(1<<(i^1)) != 0 {
					want = false
				}
			}
			if yes != want {
				t.Fatal(left, right, yes, want)
			}
			if yes {
				for assignment := uint32(0); assignment < 8; assignment++ {
					eval := func(l Literal) bool { v := assignment&(1<<l.variable()) != 0; return v != l.sign() }
					premises := false
					for _, l := range ls {
						premises = premises || eval(l)
					}
					other := false
					for _, l := range rs {
						other = other || eval(l)
					}
					if premises && other {
						n, e := out.Size(r)
						satOK(t, e)
						satisfied := false
						for i := uint32(0); i < n; i++ {
							l, e := out.Read(r, i)
							satOK(t, e)
							satisfied = satisfied || eval(l)
						}
						if !satisfied {
							t.Fatal("independent resolution soundness", left, right, assignment)
						}
					}
				}
			}
			satOK(t, a.Delete(c1))
			satOK(t, a.Delete(c2))
		}
	}
}
func TestSATSimplifierNativeResolutionGrowthAndMarkCleanup(t *testing.T) {
	k, a, _, b := satSimplifierKernelFixture(t, 4)
	out, r := satKernelOutput(t, b)
	k.elimCounter = 10
	before := b.Used()
	b.FailAfter(0)
	yes, e := k.resolve(newSATBinaryWrapper(0, 2), newSATBinaryWrapper(1, 4), 0, out, r)
	satOK(t, e)
	if !yes || k.elimCounter != 6 || b.Used() != before {
		t.Fatal("native binary resolution", yes, k.elimCounter)
	}
	satKernelOutputWant(t, out, r, satLs(2, 4))
	marked, e := k.isMarked(2)
	satOK(t, e)
	if marked {
		t.Fatal("normal resolution kept first-input mark")
	}
	b.FailAfter(-1)
	satOK(t, out.Shrink(r, 0))
	c1 := satMk(t, a, satLs(0, 2, 4), false)
	c2 := satMk(t, a, satLs(1, 6, 6), false)
	before = b.Used()
	b.FailAfter(0)
	yes, e = k.resolve(newSATClauseWrapper(c1), newSATClauseWrapper(c2), 0, out, r)
	satOK(t, e)
	if !yes || k.elimCounter != 0 || b.Used() != before {
		t.Fatal("native clause resolution", yes, k.elimCounter)
	}
	satKernelOutputWant(t, out, r, satLs(2, 4, 6, 6))
	marked, e = k.isMarked(2)
	satOK(t, e)
	if marked {
		t.Fatal("first-input mark survived native completion")
	}
	b.FailAfter(-1)
}

func TestSATSimplifierResolutionDebugPivotAndWarmZeroAllocation(t *testing.T) {
	k, _, _, b := satSimplifierKernelFixture(t, 3)
	out, r := satKernelOutput(t, b)
	k.elimCounter = 10
	yes, e := k.resolve(newSATBinaryWrapper(2, 4), newSATBinaryWrapper(1, 4), 0, out, r)
	if satStorageSourceDebug {
		if e != ErrSATAssertion || k.elimCounter != 10 {
			t.Fatal(yes, e, k.elimCounter)
		}
	} else {
		satOK(t, e)
		if !yes {
			t.Fatal(yes)
		}
	}
	satOK(t, k.InitVisited(3))
	satOK(t, out.Shrink(r, 0))
	_, e = k.resolve(newSATBinaryWrapper(0, 2), newSATBinaryWrapper(1, 4), 0, out, r)
	satOK(t, e)
	b.FailAfter(0)
	before := b.Used()
	var failure error
	allocs := testing.AllocsPerRun(50, func() {
		if failure != nil {
			return
		}
		failure = out.Shrink(r, 0)
		if failure == nil {
			_, failure = k.resolve(newSATBinaryWrapper(0, 2), newSATBinaryWrapper(1, 4), 0, out, r)
		}
	})
	if failure != nil || allocs != 0 || b.Used() != before {
		t.Fatal(failure, allocs, b.Used(), before)
	}
	b.FailAfter(-1)
}
