// Source: src/util/scoped_numeral.h, src/util/inf_rational.h.
package util

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// A failed rational copy constructor has no rational destructor. Its partial
// owning allocation is a Go obligation, after completed source stack unwind.
func TestConstructorPartialAbortWaitsForSourceUnwind(t *testing.T) {
	s, _ := serviceNew(t)
	partial, older := NewRational(), NewRational()
	if e := s.Manager.SetString(&partial, "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	if e := s.Manager.SetString(&older, "18446744073709551619"); e != nil {
		t.Fatal(e)
	}
	ref := partial.Num.Cell
	if e := s.abortLexicalRational(&partial); e != nil {
		t.Fatal(e)
	}
	state, _ := s.LexicalState()
	if state.Aborts != 1 || !state.Pending || !ValidNumericRef(s.Manager.Z.Cells, ref) {
		t.Fatal("partial copy aborted before completed outer source unwind", state)
	}
	serviceAuditRationalRoots(t, s, &older, &s.Lexical.Roots[0])
	if e := s.destroyLexicalRational(&older); e != nil {
		t.Fatal(e)
	}
	if !ValidNumericRef(s.Manager.Z.Cells, ref) {
		t.Fatal("completed source cleanup prematurely consumed partial ctor")
	}
	if e := s.CleanupLexical(); e != nil {
		t.Fatal(e)
	}
	serviceAuditRationalRoots(t, s)
}

func TestConstructorLexicalBoundaryAPI(t *testing.T) {
	s, _ := serviceNew(t)
	if e := s.beginLexical(); e != nil {
		t.Fatal(e)
	}
	if e := s.beginLexical(); e != nil {
		t.Fatal(e)
	}
	partial := NewRational()
	if e := s.Manager.SetString(&partial, "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	if e := s.abortLexicalRational(&partial); e != nil {
		t.Fatal(e)
	}
	var e error
	s.endLexicalError(&e)
	state, _ := s.LexicalState()
	if e != nil || state.Aborts != 1 {
		t.Fatal("inner boundary prematurely aborted", state, e)
	}
	s.endLexicalError(&e)
	state, _ = s.LexicalState()
	if e != nil || state.Pending {
		t.Fatal("outer boundary lost cleanup", state, e)
	}
	serviceAuditRationalRoots(t, s)
}

// C++11 rational(prvalue) selects move/elision. It must not make another BIG
// copy after the returned scalar expression is already an owning value.
func TestConstructorInfPowerUsesPrvalueMove(t *testing.T) {
	measure := func(manual bool) uint64 {
		b := arena.NewBudget(1 << 25)
		cfg := serviceConfig(b)
		cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
		s, e := NewRationalService(cfg)
		if e != nil {
			t.Fatal(e)
		}
		defer func() {
			if e := s.Close(); e != nil {
				t.Error(e)
			}
			if b.Used() != 0 {
				t.Error(b.Used())
			}
		}()
		v := s.infEmpty()
		if e = s.Manager.SetString(&v.First, "18446744073709551617/3"); e != nil {
			t.Fatal(e)
		}
		v.Second.Num = NewInt(1)
		before, e := s.Manager.Z.Memory.AllocationCount()
		if e != nil {
			t.Fatal(e)
		}
		out := s.infEmpty()
		if manual {
			two, half, expr, result := NewRational(), NewRational(), NewRational(), NewRational()
			if e = s.Manager.SetInt64(&two, 2); e != nil {
				t.Fatal(e)
			}
			if e = s.infRatExpr(&v.First, &two, &half, '/'); e != nil {
				t.Fatal(e)
			}
			if e = s.infRatExpr(&v.First, &half, &expr, '+'); e != nil {
				t.Fatal(e)
			}
			// expr is already the source value result; move/elision performs no
			// additional manager Copy before expt's completed default result.
			if e = s.Manager.Power(&expr, 2, &result); e != nil {
				t.Fatal(e)
			}
			if e = s.Manager.Copy(&out.First, &result); e != nil {
				t.Fatal(e)
			}
			for _, p := range []*Rational{&result, &expr, &half, &two} {
				if e = s.Manager.Del(p); e != nil {
					t.Fatal(e)
				}
			}
		} else {
			out, e = s.infPower(&v, 2, true)
			if e != nil {
				t.Fatal(e)
			}
		}
		after, e := s.Manager.Z.Memory.AllocationCount()
		if e != nil {
			t.Fatal(e)
		}
		serviceAuditRationalRoots(t, s, &v.First, &v.Second, &out.First, &out.Second)
		if got := roundPublicationDecode(t, s.Manager.Z, &out.First.Num).String(); got != "340282366920938463500268095579187314689" {
			t.Fatal(got)
		}
		if got := out.First.Den.Val; got != 4 {
			t.Fatal(got)
		}
		if e = s.InfDel(&out); e != nil {
			t.Fatal(e)
		}
		if e = s.InfDel(&v); e != nil {
			t.Fatal(e)
		}
		return after - before
	}
	want, got := measure(true), measure(false)
	if got != want {
		t.Fatalf("extra copy in prvalue move: source actions %d, facade %d", want, got)
	}
}

// Both refusals are deliberately malformed actual allocator metadata. The
// first recovery must consume completed outer source destruction BEFORE the
// earlier queued partial ctor abort, without replaying either freed member.
func TestConstructorCompletedUnwindPrecedesOwningAbort(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	cfg := serviceConfig(b)
	cfg.Integers.Memory = r
	cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 32}
	s, e := NewRationalService(cfg)
	if e != nil {
		t.Fatal(e)
	}
	z := s.Manager.Z
	partial, outer := NewRational(), NewRational()
	if e = s.Manager.SetString(&partial, "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	if e = s.Manager.SetString(&outer, "18446744073709551619"); e != nil {
		t.Fatal(e)
	}
	pr, or := partial.Num.Cell, outer.Num.Cell
	pc, oc := z.Cell(pr), z.Cell(or)
	pbad, obad := pc, oc
	pbad.Backing.Requested = uint64(math.MaxUint32) + 1
	obad.Backing.Requested = pbad.Backing.Requested
	z.Store(pr, pbad)
	z.Store(or, obad)
	defer func() {
		for _, x := range []struct {
			r NumericCellRef
			c NumericCell
		}{{pr, pc}, {or, oc}} {
			if ValidNumericRef(z.Cells, x.r) {
				c := z.Cell(x.r)
				c.Backing.Requested = x.c.Backing.Requested
				z.Store(x.r, c)
			}
		}
		if e := s.CleanupLexical(); e != nil {
			t.Error(e)
		}
		if e := s.Close(); e != nil {
			t.Error(e)
		}
		if e := r.Close(); e != nil {
			t.Error(e)
		}
		if b.Used() != 0 {
			t.Error(b.Used())
		}
	}()
	if e = s.beginLexical(); e != nil {
		t.Fatal(e)
	}
	if e = s.beginLexical(); e != nil {
		t.Fatal(e)
	}
	partialComplete := false
	s.endLexicalLocal(&partial, &partialComplete)
	e = ErrRationalFacadeDomain
	s.endLexicalError(&e)
	outerComplete := true
	s.endLexicalLocal(&outer, &outerComplete)
	s.endLexicalError(&e)
	state, _ := s.LexicalState()
	if state.Aborts != 1 || state.Completed != 1 || !errors.Is(state.Cause, arena.ErrBounds) || !errors.Is(state.OriginalCause, ErrRationalFacadeDomain) {
		t.Fatal(state, e)
	}
	serviceAuditRationalRoots(t, s, &s.Lexical.Roots[0], &s.Lexical.Roots[1])
	// Restore the source-completed object only. The remaining abort refuses.
	c := z.Cell(or)
	c.Backing.Requested = oc.Backing.Requested
	z.Store(or, c)
	if e = s.CleanupLexical(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if ValidNumericRef(z.Cells, or) || !ValidNumericRef(z.Cells, pr) {
		t.Fatal("wrong completed/partial ordering")
	}
	state, _ = s.LexicalState()
	if state.Completed != 0 || state.Aborts != 1 {
		t.Fatal(state)
	}
	serviceAuditRationalRoots(t, s, &s.Lexical.Roots[0])
	c = z.Cell(pr)
	c.Backing.Requested = pc.Backing.Requested
	z.Store(pr, c)
	if e = s.CleanupLexical(); e != nil {
		t.Fatal(e)
	}
	serviceAuditRationalRoots(t, s)
}

// max8 is an actual charged owner array. Depth is stack-order metadata, not an
// allocation ticket. Every partial owning record remains independently named.
func TestConstructorEightRootsAndWarmBoundary(t *testing.T) {
	s, b := serviceNew(t)
	if e := s.beginLexical(); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 8; i++ {
		v := NewRational()
		if e := s.Manager.SetString(&v, "18446744073709551617"); e != nil {
			t.Fatal(e)
		}
		if e := s.abortLexicalRational(&v); e != nil {
			t.Fatal(e)
		}
	}
	st, _ := s.LexicalState()
	if st.Aborts != 8 || st.Completed != 0 {
		t.Fatal(st)
	}
	serviceAuditRationalRoots(t, s, &s.Lexical.Roots[0], &s.Lexical.Roots[1], &s.Lexical.Roots[2], &s.Lexical.Roots[3], &s.Lexical.Roots[4], &s.Lexical.Roots[5], &s.Lexical.Roots[6], &s.Lexical.Roots[7])
	var e error
	s.endLexicalError(&e)
	if e != nil {
		t.Fatal(e)
	}
	serviceAuditRationalRoots(t, s)
	before := numericWarmProfileSnapshot(t, s.Manager.Z, b)
	allocs := testing.AllocsPerRun(20, func() {
		if e := s.beginLexical(); e != nil {
			panic(e)
		}
		var e error
		s.endLexicalError(&e)
		if e != nil {
			panic(e)
		}
	})
	numericWarmProfileAudit(t, s.Manager.Z, b, before, allocs, false)
}

// Exercise the actual free-operator COPY constructor, rather than a synthetic
// callback: the first real Direct allocation refuses before compound action.
func TestConstructorExpressionFailedCopyPhase(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	cfg := serviceConfig(b)
	cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	s, e := NewRationalService(cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := s.Close(); e != nil {
			t.Error(e)
		}
		if b.Used() != 0 {
			t.Error(b.Used())
		}
	}()
	src := NewRational()
	if e = s.Manager.SetString(&src, "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	one, out := NewRational(), NewRational()
	one.Num = NewInt(1)
	if e = s.beginLexical(); e != nil {
		t.Fatal(e)
	}
	b.FailAfter(0)
	e = s.infRatExpr(&src, &one, &out, '+')
	state, _ := s.LexicalState()
	if !numericPhysicalReservationRefusal(e) || state.Aborts != 1 || state.Completed != 0 || s.Lexical.Roots[0].Num.Cell != s.Manager.Z.FailedNullRef {
		t.Fatal("failed copy destructor phase", e, state)
	}
	serviceAuditRationalRoots(t, s, &src, &out, &s.Lexical.Roots[0])
	b.FailAfter(-1)
	if next := s.Manager.Z.CleanupPendingAllocation(); next != nil {
		t.Fatal(next)
	}
	s.endLexicalError(&e)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal(e)
	}
	state, _ = s.LexicalState()
	if state.Pending {
		t.Fatal(state)
	}
	serviceAuditRationalRoots(t, s, &src, &out)
	if e = s.Manager.Del(&src); e != nil {
		t.Fatal(e)
	}
}
