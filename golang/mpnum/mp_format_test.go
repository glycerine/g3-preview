// Copyright (c) Microsoft Corporation. All rights reserved.
// Formatting expectations project src/util/mpn.cpp to_string and mpz/mpq display.
package mpnum

import (
	"bytes"
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math/big"
	"reflect"
	"testing"
)

func TestNaturalOwnedConstructorAndClose(t *testing.T) {
	completed := false
	for fault := int64(0); fault < 128; fault++ {
		b := arena.NewBudget(1 << 22)
		b.FailAfter(fault)
		m, e := NewNaturalManager(arena.VectorConfig{PageSize: 32, Descriptors: arena.Config{PageSize: 2, Budget: b}})
		if e != nil {
			if m != nil || b.Used() != 0 {
				t.Fatal(fault, e, b.Used())
			}
			continue
		}
		if m.words.Stats().Live != 0 || m.RuntimeOwner().Stats().LiveBlocks != 0 {
			t.Fatal("source inline constructor allocated payload")
		}
		for i := range m.division {
			buffer := &m.division[i]
			if buffer.capacity() != 16 || buffer.size() != 0 || buffer.raw != (memory.Block{}) || buffer.pending != (memory.Block{}) || m.divScratch[i] != (arena.VectorRef{}) || m.divCapacity[i] != 16 {
				t.Fatal("actual source inline constructor", i, buffer)
			}
		}
		m.Close()
		m.Close()
		if b.Used() != 0 {
			t.Fatal("close", b.Used())
		}
		completed = true
		break
	}
	if !completed {
		t.Fatal("constructor sweep incomplete")
	}
}

func TestOwnedDecimalFormatAndPublication(t *testing.T) {
	m := naturalManager(t)
	cases := [][]uint32{{0}, {1}, {0xffffffff}, {0, 1}, {0, 0, 1}, {0xffffffff, 0xffffffff, 0xffffffff}, {1, 0, 0, 1}}
	for _, a := range cases {
		want := oracle(a).String()
		dst := bytes.Repeat([]byte{'?'}, len(want)+2)
		n, e := m.FormatInto(a, dst[:len(want)])
		if e != nil || n != len(want) || string(dst[:n]) != want || dst[n] != '?' {
			t.Fatal(a, n, e, string(dst))
		}
		before := append([]byte(nil), dst...)
		n, e = m.FormatInto(a, dst[:len(want)-1])
		if !errors.Is(e, ErrShortBuffer) || n != len(want) || !bytes.Equal(dst, before) {
			t.Fatal("short publication", n, e)
		}
	}
	dst := make([]byte, 200)
	if _, e := m.FormatInto(cases[len(cases)-1], dst); e != nil {
		t.Fatal(e)
	}
	var e error
	if n := testing.AllocsPerRun(100, func() { _, e = m.FormatInto(cases[len(cases)-1], dst) }); n != 0 || e != nil {
		t.Fatal("warm", n, e)
	}
	m.Close()
	if _, e := m.FormatInto([]uint32{1}, dst); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	var raw NaturalManager
	if _, e := raw.FormatInto([]uint32{1}, dst); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestIntegerRationalCombinedFormat(t *testing.T) {
	m, _ := rationalManager(t, false)
	a := NewRational()
	var e error
	// Independent powers exercise a numerator prefix retained while denominator
	// preparation grows the shared owned byte workspace beyond source inline1024.
	numerator := new(big.Int).Lsh(big.NewInt(1), 4000)
	denominator := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 4100), big.NewInt(1))
	setIntegerOracle(t, m.z, &a.num, numerator)
	setIntegerOracle(t, m.z, &a.den, denominator)
	want := numerator.String() + "/" + denominator.String()
	dst := make([]byte, len(want)+1)
	dst[len(want)] = '!'
	n, e := m.FormatInto(&a, dst[:len(want)])
	if e != nil || n != len(want) || string(dst[:n]) != want || dst[n] != '!' {
		t.Fatal(n, e)
	}
	s, e := m.String(&a)
	if e != nil || s != want {
		t.Fatal(e)
	}
	var formatError error
	if allocations := testing.AllocsPerRun(10, func() { _, formatError = m.FormatInto(&a, dst[:len(want)]) }); allocations != 0 || formatError != nil {
		t.Fatal(allocations, formatError)
	}
	before := append([]byte(nil), dst...)
	_, e = m.FormatInto(&a, dst[:len(want)-1])
	if !errors.Is(e, ErrShortBuffer) || !bytes.Equal(before, dst) {
		t.Fatal(e)
	}
}

func TestNaturalBorrowedDivisionFailureAtomic(t *testing.T) {
	a := make([]uint32, 40)
	b := make([]uint32, 19)
	for i := range a {
		a[i] = 0xfedcba98
	}
	for i := range b {
		b[i] = 0x12345678
	}
	b[len(b)-1] = 0x80000000
	completed := false
	for fault := int64(0); fault < 128; fault++ {
		budget := arena.NewBudget(1 << 22)
		m, e := NewNaturalManager(arena.VectorConfig{PageSize: 32, Descriptors: arena.Config{PageSize: 2, Budget: budget}})
		if e != nil {
			t.Fatal(e)
		}
		q, r := make([]uint32, len(a)-len(b)+1), make([]uint32, len(b))
		for i := range q {
			q[i] = 123
		}
		for i := range r {
			r[i] = 456
		}
		budget.FailAfter(fault)
		_, e = m.Div(a, b, q, r)
		budget.FailAfter(-1)
		if e != nil {
			for _, v := range q {
				if v != 123 {
					t.Fatal(fault, "quotient publication")
				}
			}
			for _, v := range r {
				if v != 456 {
					t.Fatal(fault, "remainder publication")
				}
			}
		} else {
			wq, wr := new(big.Int), new(big.Int)
			wq.QuoRem(oracle(a), oracle(b), wr)
			if oracle(q).Cmp(wq) != 0 || oracle(r).Cmp(wr) != 0 {
				t.Fatal("division oracle")
			}
			completed = true
		}
		m.Close()
		if budget.Used() != 0 {
			t.Fatal("scratch leak", fault, budget.Used())
		}
		if completed {
			break
		}
	}
	if !completed {
		t.Fatal("division sweep incomplete")
	}
}

func TestFormattingBudgetFailuresAndLifetime(t *testing.T) {
	// Every service reservation prefix is exercised both cold and after a small
	// format. Retained scratch on failure is owner-owned and fully released by
	// Close, while caller output and numeral inputs remain untouched.
	for kind := 0; kind < 3; kind++ {
		for _, warm := range []bool{false, true} {
			completed, failures := false, 0
			for fault := int64(0); fault < 128; fault++ {
				budget := arena.NewBudget(1 << 24)
				c := arena.VectorConfig{PageSize: 32, Descriptors: arena.Config{PageSize: 2, Budget: budget}}
				a := make([]uint32, 130)
				for i := range a {
					a[i] = 0x12345678
				}
				a[len(a)-1] = 1
				original := append([]uint32(nil), a...)
				dst := bytes.Repeat([]byte{'!'}, 3000)
				before := append([]byte(nil), dst...)
				var call func() (int, error)
				var closeOwner func()
				var nm *NaturalManager
				if kind == 0 {
					var e error
					nm, e = NewNaturalManager(c)
					if e != nil {
						t.Fatal(e)
					}
					call = func() (int, error) { return nm.FormatInto(a, dst) }
					closeOwner = nm.Close
				} else {
					rm, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: budget}, Words: c}})
					if e != nil {
						t.Fatal(e)
					}
					r := NewRational()
					if e = rm.z.SetLimbs(&r.num, a); e != nil {
						t.Fatal(e)
					}
					if e = rm.z.Neg(&r.num); e != nil {
						t.Fatal(e)
					}
					if kind == 1 {
						call = func() (int, error) { return rm.z.FormatInto(&r.num, dst) }
					} else {
						if e = rm.z.SetLimbs(&r.den, a[:129]); e != nil {
							t.Fatal(e)
						}
						call = func() (int, error) { return rm.FormatInto(&r, dst) }
					}
					nm = rm.z.natural
					closeOwner = rm.Close
				}
				if warm {
					if _, e := nm.FormatInto([]uint32{123}, dst[:3]); e != nil {
						t.Fatal(e)
					}
					copy(dst, before)
				}
				budget.FailAfter(fault)
				n, e := call()
				budget.FailAfter(-1)
				if e != nil {
					failures++
					if !errors.Is(e, arena.ErrInjectedFailure) || !bytes.Equal(before, dst) {
						t.Fatal(kind, warm, fault, n, e, "published")
					}
				} else {
					completed = true
					if n == 0 || dst[n] != '!' {
						t.Fatal("output length", n)
					}
				}
				if !reflect.DeepEqual(a, original) {
					t.Fatal("borrowed input changed")
				}
				closeOwner()
				if budget.Used() != 0 {
					t.Fatal(kind, warm, fault, "format leak", budget.Used())
				}
				if completed {
					break
				}
			}
			if !completed || failures == 0 {
				t.Fatal(kind, warm, "sweep incomplete", failures)
			}
		}
	}
	// Returned host strings remain valid after workspace replacement and Close.
	m := naturalManager(t)
	a := []uint32{0xffffffff, 0xffffffff}
	s, e := m.String(a)
	if e != nil {
		t.Fatal(e)
	}
	want := oracle(a).String()
	a[0] = 1
	changed, e := m.String(a)
	if e != nil || changed == s {
		t.Fatal(e, "borrow retained or output overwritten")
	}
	m.Close()
	if s != want {
		t.Fatal("host output lifetime")
	}
}

func TestIntegerSmallAndRawRationalFormatting(t *testing.T) {
	m, _ := integerManager(t)
	for _, v := range []int32{0, 1, -1, -2147483648, 2147483647} {
		a := NewInt(v)
		want := big.NewInt(int64(v)).String()
		dst := make([]byte, len(want))
		n, e := m.FormatInto(&a, dst)
		if e != nil || n != len(want) || string(dst) != want {
			t.Fatal(v, n, e)
		}
	}
	rm, _ := rationalManager(t, false)
	var raw Rational
	dst := []byte("unchanged")
	if _, e := rm.FormatInto(&raw, dst); !errors.Is(e, ErrUninitializedRational) || string(dst) != "unchanged" {
		t.Fatal(e)
	}
	rm.Close()
	valid := NewRational()
	if _, e := rm.FormatInto(&valid, dst); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestNaturalNestedOwnerAccounting(t *testing.T) {
	b := arena.NewBudget(1 << 23)
	c := arena.VectorConfig{PageSize: 32, Descriptors: arena.Config{PageSize: 2, Budget: b}}
	words, e := arena.NewWords(c)
	if e != nil {
		t.Fatal(e)
	}
	m, e := newNaturalManager(c, words, false)
	if e != nil {
		t.Fatal(e)
	}
	if m.ownsWords || m.words != words || m.budget != b {
		t.Fatal("nested ownership")
	}
	dst := make([]byte, 1000)
	if _, e = m.FormatInto([]uint32{0xffffffff, 0xffffffff, 1}, dst); e != nil {
		t.Fatal(e)
	}
	ws := words.Stats()
	bs := m.text.Stats()
	measured := m.headerBytes + ws.OwnerBytes + ws.PayloadBytes + ws.BlockMetadataBytes + ws.DirectoryBytes + ws.Descriptors.OwnerBytes + ws.Descriptors.RecordBytes + ws.Descriptors.MetadataBytes + ws.Descriptors.DirectoryBytes + bs.OwnerBytes + bs.PayloadBytes + bs.MetadataBytes + bs.DirectoryBytes
	if measured != b.Used() {
		t.Fatal("complete owner graph", measured, b.Used())
	}
	if m.headerBytes != uint64(reflect.TypeOf(NaturalManager{}).Size()) {
		t.Fatal("natural header")
	}
	m.Close()
	if words.Stats().Live != 0 {
		t.Fatal("child scratch not freed")
	}
	// Child close preserves the parent's Words service and its retained pool.
	ref, e := words.Allocate(1, 1)
	if e != nil {
		t.Fatal(e)
	}
	words.Free(ref)
	words.Close()
	if b.Used() != 0 {
		t.Fatal("parent close", b.Used())
	}
}

func TestNaturalDivisionWarmNoAllocations(t *testing.T) {
	m := naturalManager(t)
	a := []uint32{1, 0, 0x80000000}
	b := []uint32{2, 0, 0x80000000}
	q, r := make([]uint32, 1), make([]uint32, 3)
	var e error
	if _, e = m.Div(a, b, q, r); e != nil {
		t.Fatal(e)
	}
	if allocations := testing.AllocsPerRun(100, func() { _, e = m.Div(a, b, q, r) }); allocations != 0 || e != nil {
		t.Fatal(allocations, e)
	}
	m.Close()
	if _, e = m.Div(a, b, q, r); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestOwnedFormattingBudgetLimit(t *testing.T) {
	c := arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 8, Budget: arena.NewBudget(1 << 20)}}
	m, e := NewNaturalManager(c)
	if e != nil {
		t.Fatal(e)
	}
	baseline := c.Descriptors.Budget.Used()
	m.Close()
	b := arena.NewBudget(baseline)
	c.Descriptors.Budget = b
	m, e = NewNaturalManager(c)
	if e != nil {
		t.Fatal(e)
	}
	dst := []byte("unchanged")
	if _, e = m.FormatInto([]uint32{123}, dst); !errors.Is(e, arena.ErrBudget) || string(dst) != "unchanged" {
		t.Fatal(e)
	}
	m.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestNaturalPreparedAddBackStorageAdaptation(t *testing.T) {
	// src/util/mpn.cpp:341-344 grows ab only after subtraction borrows and
	// quotient is decremented. A power-of-two divisor needs no add-back here.
	m := naturalManager(t)
	a := make([]uint32, 40)
	b := make([]uint32, 19)
	for i := range a {
		a[i] = 0x12345678
	}
	b[18] = 0x80000000
	q, r := make([]uint32, 22), make([]uint32, 19)
	before := m.budget.Used()
	if _, e := m.Div(a, b, q, r); e != nil {
		t.Fatal(e)
	}
	wq, wr := new(big.Int), new(big.Int)
	wq.QuoRem(oracle(a), oracle(b), wr)
	if oracle(q).Cmp(wq) != 0 || oracle(r).Cmp(wr) != 0 || m.divCapacity != ([4]uint32{64, 32, 32, 16}) || m.budget.Used() <= before {
		t.Fatal("prepared requested capacities", m.divCapacity)
	}
	ab := &m.division[3]
	if ab.pos != 0 || ab.initialized != 0 || ab.raw != (memory.Block{}) || ab.pending != (memory.Block{}) {
		t.Fatal("unused ab must remain uninitialized inline", ab)
	}
	baseline := m.RuntimeOwner().Stats()
	charge := m.budget.Used()
	state := m.RuntimeOwner().State()
	if modularSourceDebug {
		// Each actual debug invocation constructs and destroys its lexical temp;
		// no universal zero-Go-allocation claim applies to these source mallocs.
		for iteration := 0; iteration < 10; iteration++ {
			previous := m.RuntimeOwner().State().GlobalCount
			if _, e := m.Div(a, b, q, r); e != nil {
				t.Fatal(e)
			}
			now := m.RuntimeOwner().State()
			if now.GlobalCount <= previous || now.GlobalSize != state.GlobalSize || now.LocalSize != state.LocalSize || m.RuntimeOwner().Stats() != baseline || m.budget.Used() != charge {
				t.Fatal("lexical temp physical destruction", state, now, baseline, m.RuntimeOwner().Stats())
			}
		}
	} else {
		var failure error
		if allocations := testing.AllocsPerRun(100, func() { _, failure = m.Div(a, b, q, r) }); allocations != 0 || failure != nil {
			t.Fatal(allocations, failure)
		}
		if m.RuntimeOwner().State() != state || m.RuntimeOwner().Stats() != baseline || m.budget.Used() != charge {
			t.Fatal("release warm division allocated actual storage")
		}
	}

}

func TestOwnedTextGrowthPeakAndRefusal(t *testing.T) {
	m := naturalManager(t)
	dst := bytes.Repeat([]byte{'?'}, 2000)
	if _, e := m.FormatInto([]uint32{123}, dst[:3]); e != nil {
		t.Fatal(e)
	}
	oldRef, oldCapacity := m.textRef, m.textCapacity
	// Leave less than the next complete2048-byte workspace reservation. The old
	// live1024 bytes remain charged; publication cannot consume its storage.
	otherOwner := m.budget.Limit() - m.budget.Used() - 2047
	if e := m.budget.Reserve(otherOwner); e != nil {
		t.Fatal(e)
	}
	a := make([]uint32, 130)
	a[len(a)-1] = 1
	before := append([]byte(nil), dst...)
	if _, e := m.FormatInto(a, dst); !errors.Is(e, arena.ErrBudget) || !bytes.Equal(dst, before) || m.textRef != oldRef || m.textCapacity != oldCapacity || !m.text.IsLive(oldRef) {
		t.Fatal("old-plus-new refusal", e)
	}
	m.budget.Release(otherOwner)
	if _, e := m.FormatInto(a, dst); e != nil || m.textCapacity != 2048 || m.text.IsLive(oldRef) {
		t.Fatal("growth replacement", e, m.textCapacity)
	}
	m.Close()
	if m.budget.Used() != 0 {
		t.Fatal("growth close", m.budget.Used())
	}
}

func TestNumericStringHostOwnershipBoundary(t *testing.T) {
	natural := naturalManager(t)
	a := []uint32{0xffffffff, 0xffffffff, 1}
	integer, _ := integerManager(t)
	var n Int
	if e := integer.SetLimbs(&n, a); e != nil {
		t.Fatal(e)
	}
	rational, _ := rationalManager(t, false)
	r := NewRational()
	if e := rational.z.SetLimbs(&r.num, a); e != nil {
		t.Fatal(e)
	}
	r.den = NewInt(7)
	for _, call := range []func() (string, error){func() (string, error) { return natural.String(a) }, func() (string, error) { return integer.String(&n) }, func() (string, error) { return rational.String(&r) }} {
		result, e := call()
		if e != nil {
			t.Fatal(e)
		}
		// Exactly one allocation is the final caller-owned Go string, including the
		// rational wrapper. There are no intermediate numerator/denominator strings.
		allocations := testing.AllocsPerRun(100, func() { result, e = call() })
		if allocations != 1 || e != nil || len(result) == 0 {
			t.Fatal("host output allocation", allocations, e)
		}
	}
}
