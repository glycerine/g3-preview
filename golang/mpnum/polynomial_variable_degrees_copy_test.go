// Source: src/math/polynomial/polynomial.h:62-69; src/math/polynomial/polynomial.cpp:66-82; src/util/vector.h:101-116,160-210,530-546.
// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"strings"
	"testing"
)

func degreeCopyRuntime(t *testing.T) (*memory.Runtime, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	})
	return r, b
}
func degreeCopyOwner(t *testing.T, r *memory.Runtime) *PolynomialVariableDegrees {
	t.Helper()
	d, e := NewPolynomialVariableDegrees(r)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, d.CloseChecked()) })
	return d
}
func degreeCopyAudit(t *testing.T, r *memory.Runtime, b *arena.Budget, ds ...*PolynomialVariableDegrees) {
	t.Helper()
	h := uint64(0)
	n := uint32(0)
	for _, d := range ds {
		if d == nil {
			continue
		}
		h += d.header
		if d.raw != (memory.Block{}) && !d.rawDestroyed {
			n++
		}
		if d.pending != (memory.Block{}) {
			n++
		}
	}
	if r.Stats().LiveBlocks != n || r.Stats().TotalBytes+h != b.Used() {
		t.Fatal("named roots", r.Stats(), h, b.Used(), n)
	}
}
func TestPolynomialVariableDegreesSourceCopyLogicalTail(t *testing.T) {
	r, b := degreeCopyRuntime(t)
	a := degreeCopyOwner(t, r)
	serviceMust(t, a.SetDegree(8, 17))
	serviceMust(t, a.resize(2))
	serviceMust(t, a.SetDegree(0, 3))
	c, e := NewPolynomialVariableDegreesCopy(a)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.CloseChecked()) })
	degreeCopyAudit(t, r, b, a, c)
	ac, e := a.capacity()
	serviceMust(t, e)
	cc, e := c.capacity()
	serviceMust(t, e)
	if ac != cc || a.raw == c.raw {
		t.Fatal(ac, cc)
	}
	_, e = r.ReadUint32(c.raw, 40)
	if !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal("only logical slots copied", e)
	}
	serviceMust(t, c.SetDegree(0, 9))
	v, e := a.Degree(0)
	serviceMust(t, e)
	if v != 3 {
		t.Fatal(v)
	}
	serviceMust(t, c.CopyFrom(c))
	empty := degreeCopyOwner(t, r)
	serviceMust(t, c.CopyFrom(empty))
	if c.raw != (memory.Block{}) {
		t.Fatal(c.raw)
	}
	degreeCopyAudit(t, r, b, a, c, empty)
}
func TestPolynomialVariableDegreesSourceCopyFailurePhases(t *testing.T) {
	r, b := degreeCopyRuntime(t)
	a := degreeCopyOwner(t, r)
	serviceMust(t, a.SetDegree(8, 19))
	d := degreeCopyOwner(t, r)
	serviceMust(t, d.SetDegree(0, 5))
	old := d.raw
	b.FailAfter(0)
	e := d.CopyFrom(a)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal(e)
	}
	if d.raw != old || !d.rawDestroyed || !d.copyBroken {
		t.Fatal("old free before RHS refusal", d.raw, old)
	}
	degreeCopyAudit(t, r, b, a, d)
	if _, e = d.Degree(0); !errors.Is(e, memory.ErrRuntimeOwner) {
		t.Fatal(e)
	}
	if e = d.SourceFinalize(); !errors.Is(e, memory.ErrRuntimeOwner) {
		t.Fatal(e)
	}
	b.FailAfter(-1)
	serviceMust(t, d.CloseChecked())
	degreeCopyAudit(t, r, b, a, d)
	serviceMust(t, a.resize(1))
	serviceMust(t, r.WriteUint32(a.raw, 4, 10))
	c, e := NewPolynomialVariableDegreesCopy(a)
	if !errors.Is(e, memory.ErrRuntimeUninitialized) || c == nil {
		t.Fatal(c, e)
	}
	if c.constructed {
		t.Fatal("failed ctor completed")
	}
	degreeCopyAudit(t, r, b, a, c)
	raw := c.raw
	serviceMust(t, c.SourceFinalize())
	if c.raw != raw {
		t.Fatal("failed ctor destructor")
	}
	serviceMust(t, r.WriteUint32(a.raw, 4, 1))
	serviceMust(t, c.CloseChecked())
	degreeCopyAudit(t, r, b, a, c)
}
func TestPolynomialVariableDegreesSourceCopySizeTPromotion(t *testing.T) {
	r, b := degreeCopyRuntime(t)
	a := degreeCopyOwner(t, r)
	serviceMust(t, a.SetDegree(0, 7))
	serviceMust(t, r.WriteUint32(a.raw, 0, 0x40000000))
	d := degreeCopyOwner(t, r)
	before, e := r.AllocationCount()
	serviceMust(t, e)
	beforeState := r.State()
	b.FailAfter(0)
	e = d.CopyFrom(a)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal("SIZE_T request", e)
	}
	after, e := r.AllocationCount()
	serviceMust(t, e)
	if after != before+1 || r.State().GlobalSize-beforeState.GlobalSize != 4294967312 {
		t.Fatal("LP64 sizeof(unsigned)*capacity + 8, plus stored SIZE_T prefix", before, after, beforeState, r.State())
	}
	degreeCopyAudit(t, r, b, a, d)
	b.FailAfter(-1)
	serviceMust(t, r.WriteUint32(a.raw, 0, 2))
}
func TestPolynomialVariableDegreesSourceMoveAcrossCohorts(t *testing.T) {
	r1, b1 := degreeCopyRuntime(t)
	r2, b2 := degreeCopyRuntime(t)
	a := degreeCopyOwner(t, r1)
	d := degreeCopyOwner(t, r2)
	serviceMust(t, a.SetDegree(2, 11))
	serviceMust(t, d.SetDegree(0, 3))
	raw := a.raw
	// svector declares only operator=(const&). Implicit wrapper move assignment
	// therefore copies the member, whereas its explicit move ctor still steals.
	serviceMust(t, d.MoveFrom(a))
	if a.raw != raw || d.raw == raw || d.runtime != r2 {
		t.Fatal("source svector rvalue assignment is deep copy", a.raw, d.raw, d.runtime == r2)
	}
	degreeCopyAudit(t, r1, b1, a)
	degreeCopyAudit(t, r2, b2, d)
	serviceMust(t, d.MoveFrom(d))
	copiedRaw := d.raw
	c, e := NewPolynomialVariableDegreesMove(d)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.CloseChecked()) })
	if c.raw != copiedRaw || d.raw != (memory.Block{}) || c.runtime != r2 {
		t.Fatal("true member move constructor", c.raw, d.raw)
	}
	degreeCopyAudit(t, r2, b2, d, c)
	serviceMust(t, a.SetDegree(2, 29))
	v, e := c.Degree(2)
	serviceMust(t, e)
	if v != 11 {
		t.Fatal("copied backing independent", v)
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
	e := a.DisplayInto(s)
	if modularSourceDebug {
		if !errors.Is(e, ErrPolynomialAssertion) || s.out.String() != "[x0^" {
			t.Fatal(e, s.out.String())
		}
	} else {
		serviceMust(t, e)
		if s.out.String() != "[x0^0]" {
			t.Fatal(s.out.String())
		}
	}
}

func TestPolynomialVariableDegreesCopyConstructorReturnedBlock(t *testing.T) {
	// Actual TLS allocation limit is checked after physical allocation/sync.
	// This tests a real returned owning raw block, not a fabricated refusal.
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true})
	serviceMust(t, e)
	a, e := NewPolynomialVariableDegrees(r)
	serviceMust(t, e)
	serviceMust(t, a.SetDegree(25000, 23))
	serviceMust(t, r.SetMaxSize(1))
	c, e := NewPolynomialVariableDegreesCopy(a)
	var failure memory.Failure
	if !errors.As(e, &failure) || failure.Kind != memory.FailureMemout || c == nil || c.pending == (memory.Block{}) || c.raw != (memory.Block{}) || c.constructed {
		t.Fatal("actual failed ctor allocation prefix", c, e)
	}
	degreeCopyAudit(t, r, b, a, c)
	pending := c.pending
	serviceMust(t, c.SourceFinalize())
	if c.pending != pending {
		t.Fatal("failed member ctor has no source destructor")
	}
	// A move may transfer only published m_data. The source-invisible pending
	// block remains named in the failed owner, never in the target.
	assigned, e := NewPolynomialVariableDegrees(r)
	serviceMust(t, e)
	e = assigned.CopyFrom(a)
	if !errors.As(e, &failure) || failure.Kind != memory.FailureMemout || assigned.pending == (memory.Block{}) || !assigned.constructed {
		t.Fatal("completed default object, failed source assignment", e)
	}
	assignedPending := assigned.pending
	moved, e := NewPolynomialVariableDegreesMove(assigned)
	serviceMust(t, e)
	if moved.raw != (memory.Block{}) || moved.pending != (memory.Block{}) || assigned.pending != assignedPending || c.pending != pending {
		t.Fatal("move stole unpublished allocation")
	}
	degreeCopyAudit(t, r, b, a, c, assigned, moved)
	serviceMust(t, moved.CloseChecked())
	serviceMust(t, assigned.CloseChecked())
	serviceMust(t, c.CloseChecked())
	serviceMust(t, a.CloseChecked())
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestPolynomialVariableDegreesSourceSelfAndMoveReadMask(t *testing.T) {
	r, b := degreeCopyRuntime(t)
	a := degreeCopyOwner(t, r)
	serviceMust(t, a.SetDegree(0, 13))
	d := degreeCopyOwner(t, r)
	// Honest incomplete scalar-vector header: capacity is initialized, SIZE is
	// not. svector<unsigned> destroy does not read SIZE; move assignment frees
	// this backing then steals source.m_data. Copy self also returns before SIZE.
	var e error
	d.raw, e = r.Allocate(16)
	serviceMust(t, e)
	serviceMust(t, r.WriteUint32(d.raw, 0, 2))
	serviceMust(t, d.CopyFrom(d))
	serviceMust(t, d.MoveFrom(d))
	degreeCopyAudit(t, r, b, a, d)
	serviceMust(t, d.MoveFrom(a))
	v, e := d.Degree(0)
	serviceMust(t, e)
	if v != 13 {
		t.Fatal(v)
	}
	shallow := *d
	if e = shallow.CloseChecked(); !errors.Is(e, ErrPolynomialOwner) {
		t.Fatal("cold Go copy is not source move", e)
	}
	serviceMust(t, d.SourceFinalize())
	if _, e = d.Degree(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("source finalized read", e)
	}
	degreeCopyAudit(t, r, b, a, d)
}
func TestPolynomialVariableDegreesCopyConstructorActualRefusalSweep(t *testing.T) {
	failures, success := 0, false
	for ordinal := int64(0); ordinal < 8; ordinal++ {
		reached := false
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			r, b := degreeCopyRuntime(t)
			a := degreeCopyOwner(t, r)
			serviceMust(t, a.SetDegree(6, 31))
			before, e := r.AllocationCount()
			serviceMust(t, e)
			b.FailAfter(ordinal)
			c, e := NewPolynomialVariableDegreesCopy(a)
			degreeCopyAudit(t, r, b, a, c)
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal(e)
				}
				failures++
				if c != nil && c.constructed {
					t.Fatal("failed constructor completed")
				}
			} else {
				success = true
				reached = true
				v, e := c.Degree(6)
				serviceMust(t, e)
				if v != 31 {
					t.Fatal(v)
				}
				after, e := r.AllocationCount()
				serviceMust(t, e)
				if after != before+1 {
					t.Fatal("one actual copy allocation", before, after)
				}
			}
			b.FailAfter(-1)
			if c != nil {
				serviceMust(t, c.CloseChecked())
			}
		})
		if reached {
			break
		}
	}
	if !success || failures < 2 {
		t.Fatal("cold header and actual body refusal", failures, success)
	}
}

func TestPolynomialVariableDegreesCopySizeT32WrappedPhysicalPrefix(t *testing.T) {
	// Explicit malformed-capacity finite source action: native SIZE_T32 wraps the
	// allocation expression to eight bytes, then publishes CAP/SIZE before any
	// element access. No huge allocation or valid-source geometry claim is made.
	b := arena.NewBudget(1 << 20)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 32})
	serviceMust(t, e)
	a, e := NewPolynomialVariableDegrees(r)
	serviceMust(t, e)
	serviceMust(t, a.SetDegree(0, 7))
	serviceMust(t, a.resize(0))
	serviceMust(t, r.WriteUint32(a.raw, 0, 0x40000000))
	c, e := NewPolynomialVariableDegreesCopy(a)
	serviceMust(t, e)
	degreeCopyAudit(t, r, b, a, c)
	cap, e := c.capacity()
	serviceMust(t, e)
	n, e := c.size()
	serviceMust(t, e)
	if cap != 0x40000000 || n != 0 {
		t.Fatal(cap, n)
	}
	// The first endangered payload access occurs after resize publishes SIZE1.
	e = c.SetDegree(0, 9)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	n, e = c.size()
	serviceMust(t, e)
	if n != 1 {
		t.Fatal("published source size survives physical guard", n)
	}
	serviceMust(t, c.CloseChecked())
	serviceMust(t, r.WriteUint32(a.raw, 0, 2))
	serviceMust(t, a.CloseChecked())
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

type degreeWarmSink struct {
	data [128]byte
	n    uint32
}

func (s *degreeWarmSink) AppendCString(v string) error {
	for i := 0; i < len(v); i++ {
		s.data[s.n] = v[i]
		s.n++
	}
	return nil
}
func (s *degreeWarmSink) AppendUint32(v uint32) error {
	var digits [10]byte
	i := len(digits)
	for {
		i--
		digits[i] = byte('0' + v%10)
		v /= 10
		if v == 0 {
			break
		}
	}
	for ; i < len(digits); i++ {
		s.data[s.n] = digits[i]
		s.n++
	}
	return nil
}
func TestPolynomialVariableDegreesBorrowedDisplayWarmOwnership(t *testing.T) {
	r, b := degreeCopyRuntime(t)
	d := degreeCopyOwner(t, r)
	serviceMust(t, d.SetDegree(2, 17))
	sink := &degreeWarmSink{}
	serviceMust(t, d.DisplayInto(sink))
	stats, state, used := r.Stats(), r.State(), b.Used()
	count, e := r.AllocationCount()
	serviceMust(t, e)
	b.FailAfter(0)
	allocs := testing.AllocsPerRun(25, func() {
		sink.n = 0
		if e := d.DisplayInto(sink); e != nil {
			panic(e)
		}
		if v, e := d.Degree(2); e != nil || v != 17 {
			panic("source degree read")
		}
	})
	degreeCopyAudit(t, r, b, d)
	after, e := r.AllocationCount()
	serviceMust(t, e)
	if allocs != 0 || stats != r.Stats() || state != r.State() || used != b.Used() || count != after {
		t.Fatal("borrowed stream no source allocation/no hidden owner", allocs, stats, r.Stats(), state, r.State(), used, b.Used(), count, after)
	}
	b.FailAfter(-1)
}

func TestPolynomialVariableDegreesEmptyAssignmentRefusalIsRetryable(t *testing.T) {
	r, b := degreeCopyRuntime(t)
	a := degreeCopyOwner(t, r)
	serviceMust(t, a.SetDegree(1, 29))
	d := degreeCopyOwner(t, r)
	b.FailAfter(0)
	e := d.CopyFrom(a)
	if !numericPhysicalReservationRefusal(e) || d.raw != (memory.Block{}) || d.pending != (memory.Block{}) {
		t.Fatal("source empty m_data unchanged", e, d.raw, d.pending)
	}
	degreeCopyAudit(t, r, b, a, d)
	b.FailAfter(-1)
	serviceMust(t, d.CopyFrom(a))
	v, e := d.Degree(1)
	serviceMust(t, e)
	if v != 29 {
		t.Fatal(v)
	}
	degreeCopyAudit(t, r, b, a, d)
}

func TestPolynomialVariableDegreesFailedConstructorCarrierRejectsSourceAPI(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true})
	serviceMust(t, e)
	a, e := NewPolynomialVariableDegrees(r)
	serviceMust(t, e)
	serviceMust(t, a.SetDegree(25000, 23))
	serviceMust(t, r.SetMaxSize(1))
	c, e := NewPolynomialVariableDegreesCopy(a)
	if c == nil || c.constructed || !numericPhysicalReservationRefusal(e) || c.pending == (memory.Block{}) {
		t.Fatal("actual failed constructor owning carrier", c, e)
	}
	t.Cleanup(func() {
		serviceMust(t, c.CloseChecked())
		serviceMust(t, a.CloseChecked())
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	})
	beforeStats, beforeState, beforeUsed := r.Stats(), r.State(), b.Used()
	if _, e = c.Degree(0); !errors.Is(e, ErrPolynomialPending) {
		t.Fatal("unconstructed carrier must not return ordinary source value", e)
	}
	sink := &degreeCopySink{}
	if e = c.DisplayInto(sink); !errors.Is(e, ErrPolynomialPending) || sink.calls != 0 {
		t.Fatal("no ostream action before lifetime guard", e, sink.calls)
	}
	if e = c.SetDegree(0, 4); !errors.Is(e, ErrPolynomialPending) {
		t.Fatal(e)
	}
	if moved, e := NewPolynomialVariableDegreesMove(c); !errors.Is(e, ErrPolynomialPending) || moved != nil {
		t.Fatal("no completed move from failed ctor carrier", moved, e)
	}
	if copied, e := NewPolynomialVariableDegreesCopy(c); !errors.Is(e, ErrPolynomialPending) || copied != nil {
		t.Fatal("no copy reservation from failed ctor carrier", copied, e)
	}
	old := a.raw
	if e = a.CopyFrom(c); !errors.Is(e, ErrPolynomialPending) || a.raw != old {
		t.Fatal("reject source before destination destructor", e, a.raw, old)
	}
	if e = a.MoveFrom(c); !errors.Is(e, ErrPolynomialPending) || a.raw != old {
		t.Fatal("rvalue member copy has same lifetime guard", e)
	}
	if beforeStats != r.Stats() || beforeState != r.State() || beforeUsed != b.Used() {
		t.Fatal("rejected ordinary operations changed actual graph/accounting", beforeStats, r.Stats(), beforeState, r.State(), beforeUsed, b.Used())
	}
	degreeCopyAudit(t, r, b, a, c)
	pending := c.pending
	serviceMust(t, c.SourceFinalize())
	if c.pending != pending {
		t.Fatal("failed member ctor still has no source destructor")
	}
}
