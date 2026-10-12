package lp

import (
	"errors"
	"io"
	"math"
	"math/rand/v2"
	"reflect"
	"testing"
)

func TestLPIntSetPrintEveryByteAndBorrowedState(t *testing.T) {
	for _, values := range [][]uint32{nil, {0}, {13, 2, 111, 7}} {
		s, err := NewIntSet(112)
		lpStaticMust(t, err)
		report := ""
		for _, j := range values {
			lpStaticMust(t, s.Insert(j))
		}
		good := &lpDriverFlushWriter{}
		lpStaticMust(t, s.Print(good))
		report = good.String()
		dense := append([]uint32(nil), s.Indices()...)
		positions := append([]int32(nil), s.positions...)
		for _, cause := range []error{nil, lpErrResidualWrite} {
			for n := 0; n < len(report); n++ {
				w := &larConstraintPrefixWriter{remaining: n, err: cause}
				err := s.Print(w)
				expected := cause
				if expected == nil {
					expected = io.ErrShortWrite
				}
				if !errors.Is(err, expected) || w.output.String() != report[:n] || !reflect.DeepEqual(s.Indices(), dense) || !reflect.DeepEqual(s.positions, positions) {
					t.Fatal(values, n, err, w.output.String())
				}
			}
		}
		w := &lpDriverFlushWriter{failAt: 1}
		if err = s.Print(w); !errors.Is(err, lpErrResidualWrite) || w.String() != report || w.flushes != 1 {
			t.Fatal(values, err, w.String())
		}
		if !reflect.DeepEqual(s.positions, positions) {
			t.Fatal("output changed inverse positions")
		}
		lpStaticMust(t, s.Clear())
	}
}

// A linked immutable snapshot model is independent of the four concrete
// slice-backed implementations. Saved T& references must still address the
// object's current value through every push and saturating pop.
func TestLPStackedScalarBorrowedValuesAndIndependentScopes(t *testing.T) {
	type values struct {
		u uint32
		i int32
		s Status
		p SimplexStrategy
	}
	type frame struct {
		value  values
		parent *frame
		depth  uint32
	}
	for seed := uint64(0); seed < 12; seed++ {
		r := rand.New(rand.NewPCG(seed+31, seed+71))
		u := NewStackedUint32(math.MaxUint32)
		i := NewStackedInt32(math.MinInt32)
		s := NewStackedStatus(StatusUnknown)
		p := NewStackedStrategy(StrategyUndecided)
		up, ip, sp, pp := &u.Value, &i.Value, &s.Value, &p.Value
		want := values{math.MaxUint32, math.MinInt32, StatusUnknown, StrategyUndecided}
		var history *frame
		for step := 0; step < 3000; step++ {
			switch r.IntN(4) {
			case 0, 1:
				want = values{r.Uint32(), int32(r.Uint32()), Status(r.IntN(14)), SimplexStrategy(r.IntN(4))}
				*up, *ip, *sp, *pp = want.u, want.i, want.s, want.p
			case 2:
				depth := uint32(1)
				if history != nil {
					depth += history.depth
				}
				history = &frame{want, history, depth}
				u.Push()
				i.Push()
				s.Push()
				p.Push()
			case 3:
				k := uint32(r.IntN(5))
				if step%19 == 0 {
					k = math.MaxUint32
				}
				u.Pop(k)
				i.Pop(k)
				s.Pop(k)
				p.Pop(k)
				for remaining := k; remaining > 0 && history != nil; remaining-- {
					want = history.value
					history = history.parent
				}
			}
			depth := uint32(0)
			if history != nil {
				depth = history.depth
			}
			if (values{u.Value, i.Value, s.Value, p.Value}) != want || u.StackSize() != depth || i.StackSize() != depth || s.StackSize() != depth || p.StackSize() != depth || up != &u.Value || ip != &i.Value || sp != &s.Value || pp != &p.Value {
				t.Fatal(seed, step, want, depth)
			}
		}
		u.Pop(math.MaxUint32)
		i.Pop(math.MaxUint32)
		s.Pop(math.MaxUint32)
		p.Pop(math.MaxUint32)
		for history != nil {
			want = history.value
			history = history.parent
		}
		if (values{*up, *ip, *sp, *pp}) != want {
			t.Fatal("borrowed reference after final pop", seed)
		}
	}
}

func TestLPStackedIntSetBorrowedViewAndFrameLifetime(t *testing.T) {
	var s StackedInt32Set
	s.Insert(math.MinInt32)
	s.Insert(math.MaxInt32)
	view := s.Values()
	s.Push()
	s.Insert(7)
	if _, ok := view[7]; !ok {
		t.Fatal("view was copied")
	}
	s.Push()
	s.Insert(-3)
	lpStaticMust(t, s.Pop(1))
	if _, ok := view[-3]; ok {
		t.Fatal("borrowed view lost rollback")
	}
	if _, ok := view[7]; !ok {
		t.Fatal("outer insertion lost")
	}
	lpStaticMust(t, s.Pop(math.MaxUint32))
	if len(view) != 2 || len(s.stack) != 0 {
		t.Fatal("oldest snapshot", view)
	}
	s.Insert(9)
	if _, ok := view[9]; !ok {
		t.Fatal("view detached after pop")
	}
	for _, d := range s.stack[:cap(s.stack)] {
		if d.inserted != nil || d.erased != nil || d.debugCopy != nil {
			t.Fatal("retained popped frame")
		}
	}
}
