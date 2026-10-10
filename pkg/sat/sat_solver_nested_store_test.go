// Concrete storage adapter checks for source nested literal vectors.
package sat

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func satNestedTestStore(t *testing.T) (*satSolverLiteralStore, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	s, e := newSATSolverLiteralStore(b)
	paramsMust(t, e)
	t.Cleanup(func() {
		paramsMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatalf("nested store retained %d", b.Used())
		}
	})
	return s, b
}

func TestSATSolverNativeLiteralGrowthCopyAndOracle(t *testing.T) {
	s, b := satNestedTestStore(t)
	other, ob := satNestedTestStore(t)
	r, e := s.NewList()
	paramsMust(t, e)
	before, foreign := b.Used(), ob.Used()
	b.FailAfter(0)
	ob.FailAfter(0)
	var want []Literal
	for i := uint32(0); i < 4097; i++ {
		paramsMust(t, s.Push(r, Literal(i)))
		want = append(want, Literal(i))
	}
	seed := uint32(19)
	for step := uint32(0); step < 4096; step++ {
		seed = 1664525*seed + 1013904223
		switch seed >> 30 {
		case 0:
			l := Literal(seed)
			paramsMust(t, s.Push(r, l))
			want = append(want, l)
		case 1:
			if len(want) > 0 {
				i := int(seed>>8) % len(want)
				l := Literal(seed)
				paramsMust(t, s.Write(r, uint32(i), l))
				want[i] = l
			}
		case 2:
			n := 0
			if len(want) > 0 {
				n = int(seed>>8) % (len(want) + 1)
			}
			paramsMust(t, s.Shrink(r, uint32(n)))
			want = want[:n]
		case 3:
			q, e := other.CopyList(s, r)
			paramsMust(t, e)
			for i, l := range want {
				x, e := other.Read(q, uint32(i))
				paramsMust(t, e)
				if x != l {
					t.Fatal("foreign copy oracle", step, i)
				}
			}
			if len(want) > 0 {
				paramsMust(t, other.Write(q, 0, want[0]^1))
			}
			paramsMust(t, other.Free(q))
		}
		n, e := s.Size(r)
		paramsMust(t, e)
		if int(n) != len(want) {
			t.Fatal(step, n, len(want))
		}
		for i, l := range want {
			x, e := s.Read(r, uint32(i))
			paramsMust(t, e)
			if x != l {
				t.Fatal("native literal oracle", step, i)
			}
		}
	}
	q, e := s.CloneList(r)
	paramsMust(t, e)
	paramsMust(t, s.Free(r))
	if _, e = s.Size(r); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	for i, l := range want {
		x, e := s.Read(q, uint32(i))
		paramsMust(t, e)
		if x != l {
			t.Fatal("clone borrowed source backing")
		}
	}
	for i := 0; i < 257; i++ {
		x, e := s.NewList()
		paramsMust(t, e)
		paramsMust(t, s.Push(x, Literal(i)))
	}
	b.FailAfter(-1)
	ob.FailAfter(-1)
	if b.Used() != before || ob.Used() != foreign {
		t.Fatal("native literal backing charged", b.Used(), ob.Used())
	}
}

func TestSATSolverNativeLiteralColdHeaderAndLifetime(t *testing.T) {
	b := arena.NewBudget(uint64(reflect.TypeOf(satSolverLiteralStore{}).Size()))
	b.FailAfter(0)
	s, e := newSATSolverLiteralStore(b)
	if s != nil || !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != 0 {
		t.Fatal(s, e, b.Used())
	}
	b.FailAfter(1)
	s, e = newSATSolverLiteralStore(b)
	paramsMust(t, e)
	r, e := s.NewList()
	paramsMust(t, e)
	for i := uint32(0); i < 257; i++ {
		paramsMust(t, s.Push(r, Literal(i)))
	}
	span, e := satSimplifierCapturedSpan(s, r)
	paramsMust(t, e)
	paramsMust(t, s.Shrink(r, 1))
	if _, e = span.literal(1); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("captured logical tail survived shrink", e)
	}
	paramsMust(t, span.setLiteral(0, 31))
	x, e := s.Read(r, 0)
	paramsMust(t, e)
	if x != 31 {
		t.Fatal("borrowed span write", x)
	}
	copyOwner := *s
	if e = copyOwner.Close(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	before := b.Used()
	if n := testing.AllocsPerRun(50, func() {
		if e := s.Push(r, 2); e != nil {
			panic(e)
		}
		if e := s.Shrink(r, 1); e != nil {
			panic(e)
		}
	}); n != 0 {
		t.Fatal("warm native operations", n)
	}
	if b.Used() != before {
		t.Fatal("warm backing charge")
	}
	paramsMust(t, s.Free(r))
	if _, e = span.literal(0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	q, e := s.NewList()
	paramsMust(t, e)
	if q.slot != r.slot || q.generation == r.generation {
		t.Fatal("native directory generation")
	}
	if _, e = s.Read(q, 0); satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	paramsMust(t, s.Close())
	paramsMust(t, s.Close())
	if _, e = span.literal(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("cold refund", b.Used())
	}
}
