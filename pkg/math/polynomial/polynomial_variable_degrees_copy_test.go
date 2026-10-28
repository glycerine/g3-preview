package polynomial

// Source: src/math/polynomial/polynomial.h:62-69; src/math/polynomial/polynomial.cpp:66-82; src/util/vector.h:101-116,160-210,530-546.

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func degreeCopyRuntime(t *testing.T) (*umemory.Runtime, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	})
	return r, b
}
func degreeCopyOwner(t *testing.T, r *umemory.Runtime) *PolynomialVariableDegrees {
	t.Helper()
	d, e := NewPolynomialVariableDegrees(r)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, d.CloseChecked()) })
	return d
}

func TestPolynomialVariableDegreesSourceCopyLogicalTail(t *testing.T) {
	r, _ := degreeCopyRuntime(t)
	a := degreeCopyOwner(t, r)
	serviceMust(t, a.SetDegree(8, 17))
	serviceMust(t, a.resize(2))
	serviceMust(t, a.SetDegree(0, 3))
	c, e := NewPolynomialVariableDegreesCopy(a)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.CloseChecked()) })
	serviceMust(t, c.SetDegree(0, 9))
	v, e := a.Degree(0)
	serviceMust(t, e)
	if v != 3 {
		t.Fatal("copy shares backing", v)
	}
	serviceMust(t, a.resize(9))
	for i := uint32(2); i < 9; i++ {
		v, e = a.Degree(i)
		serviceMust(t, e)
		if v != 0 {
			t.Fatal("regrown tail was not zeroed", i, v)
		}
	}
	serviceMust(t, c.CopyFrom(c))
}

func TestPolynomialVariableDegreesSourceMoveAcrossCohorts(t *testing.T) {
	r1, _ := degreeCopyRuntime(t)
	r2, _ := degreeCopyRuntime(t)
	a := degreeCopyOwner(t, r1)
	d := degreeCopyOwner(t, r2)
	serviceMust(t, a.SetDegree(2, 11))
	serviceMust(t, d.MoveFrom(a))
	serviceMust(t, a.SetDegree(2, 29))
	v, e := d.Degree(2)
	serviceMust(t, e)
	if v != 11 || d.runtime != r2 {
		t.Fatal("move assignment must deep-copy into destination cohort", v)
	}
	c, e := NewPolynomialVariableDegreesMove(d)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.CloseChecked()) })
	n, e := d.size()
	serviceMust(t, e)
	if n != 0 || c.runtime != r2 {
		t.Fatal("move constructor did not transfer values", n)
	}
}

type degreeCopySink struct {
	out         strings.Builder
	calls, fail int
	onCall      func(int)
}

var degreeDisplayStop = errors.New("test stream action stop")

func (s *degreeCopySink) step() error {
	s.calls++
	if s.onCall != nil {
		s.onCall(s.calls)
	}
	if s.fail == s.calls {
		return degreeDisplayStop
	}
	return nil
}
func (s *degreeCopySink) AppendCString(v string) error {
	if e := s.step(); e != nil {
		return e
	}
	s.out.WriteString(v)
	return nil
}
func (s *degreeCopySink) AppendUint32(v uint32) error {
	if e := s.step(); e != nil {
		return e
	}
	fmt.Fprint(&s.out, v)
	return nil
}
func TestPolynomialVariableDegreesSourceDisplayOrder(t *testing.T) {
	r, _ := degreeCopyRuntime(t)
	a := degreeCopyOwner(t, r)
	serviceMust(t, a.SetDegree(2, 7))
	s := &degreeCopySink{}
	serviceMust(t, a.DisplayInto(s))
	if s.out.String() != "[x0^0,x1^0,x2^7]" {
		t.Fatal(s.out.String())
	}
	s = &degreeCopySink{fail: 4}
	if e := a.DisplayInto(s); !errors.Is(e, degreeDisplayStop) || s.out.String() != "[x0" {
		t.Fatal(e, s.out.String())
	}
	s = &degreeCopySink{onCall: func(n int) {
		if n == 4 {
			serviceMust(t, a.resize(0))
		}
	}}
	if e := a.DisplayInto(s); !errors.Is(e, ErrPolynomialAssertion) || s.out.String() != "[x0^" {
		t.Fatal(e, s.out.String())
	}
}

func TestPolynomialVariableDegreesNativeSliceIgnoresArenaFailure(t *testing.T) {
	r, b := degreeCopyRuntime(t)
	d := degreeCopyOwner(t, r)
	before := b.Used()
	b.FailAfter(0)
	serviceMust(t, d.SetDegree(4096, 23))
	b.FailAfter(-1)
	if b.Used() != before {
		t.Fatal("Go slice charged to arena", before, b.Used())
	}
	v, e := d.Degree(4096)
	serviceMust(t, e)
	if v != 23 {
		t.Fatal(v)
	}
}

func TestPolynomialVariableDegreesSourceSelfAndMoveReadMask(t *testing.T) {
	r, _ := degreeCopyRuntime(t)
	d := degreeCopyOwner(t, r)
	serviceMust(t, d.SetDegree(0, 13))
	serviceMust(t, d.CopyFrom(d))
	serviceMust(t, d.MoveFrom(d))
	shallow := *d
	if e := shallow.CloseChecked(); !errors.Is(e, ErrPolynomialOwner) {
		t.Fatal("cold Go copy is not a source move", e)
	}
	serviceMust(t, d.SourceFinalize())
	if _, e := d.Degree(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("source finalized read", e)
	}
}
