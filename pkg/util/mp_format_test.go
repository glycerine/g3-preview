// Formatting expectations project src/util/mpn.cpp to_string and mpz/mpq display.
package util

import (
	"bytes"
	"errors"
	"math/big"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
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
		if m.RuntimeOwner().Stats().LiveBlocks != 0 {
			t.Fatal("source inline constructor allocated payload")
		}
		for i := range m.Division {
			if m.Division[i] != nil {
				t.Fatal("eager scratch", i)
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
	setIntegerOracle(t, m.Z, &a.Num, numerator)
	setIntegerOracle(t, m.Z, &a.Den, denominator)
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
	// Both numeral conversions reuse native scratch without Runtime events.
	beforeWarm := numericWarmProfileSnapshot(t, m.Z, m.Z.BudgetField)
	var formatError error
	allocations := testing.AllocsPerRun(10, func() { _, formatError = m.FormatInto(&a, dst[:len(want)]) })
	afterWarm := numericWarmProfileSnapshot(t, m.Z, m.Z.BudgetField)
	if formatError != nil || afterWarm.count != beforeWarm.count || allocations != 0 || m.Z.Natural.Formatting.Active || m.Z.Natural.Formatting.PendingField {
		t.Fatal("native reusable scratch", allocations, afterWarm.count-beforeWarm.count, formatError)
	}
	numericWarmProfileAudit(t, m.Z, m.Z.BudgetField, beforeWarm, allocations, false)
	if string(dst[:len(want)]) != want {
		t.Fatal("warm output value")
	}
	qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den})
	before := append([]byte(nil), dst...)
	_, e = m.FormatInto(&a, dst[:len(want)-1])
	if !errors.Is(e, ErrShortBuffer) || !bytes.Equal(before, dst) {
		t.Fatal(e)
	}
}

func TestFormattingNativeScratchAndLifetime(t *testing.T) {
	// Cold and warm formatting borrow numeral inputs and publish exact decimal
	// text while all Runtime reservations are disabled.
	for kind := 0; kind < 3; kind++ {
		for _, warm := range []bool{false, true} {
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
				if e = rm.Z.SetLimbs(&r.Num, a); e != nil {
					t.Fatal(e)
				}
				if e = rm.Z.Neg(&r.Num); e != nil {
					t.Fatal(e)
				}
				if kind == 1 {
					call = func() (int, error) { return rm.Z.FormatInto(&r.Num, dst) }
				} else {
					if e = rm.Z.SetLimbs(&r.Den, a[:129]); e != nil {
						t.Fatal(e)
					}
					call = func() (int, error) { return rm.FormatInto(&r, dst) }
				}
				nm = rm.Z.Natural
				closeOwner = rm.Close
			}
			if warm {
				if _, e := nm.FormatInto([]uint32{123}, dst[:3]); e != nil {
					t.Fatal(e)
				}
				copy(dst, before)
			}

			budget.FailAfter(0)
			n, e := call()
			if e != nil {
				t.Fatal(kind, warm, e)
			}
			want := oracle(a).String()
			if kind != 0 {
				want = "-" + want
			}
			if kind == 2 {
				want += "/" + oracle(a[:129]).String()
			}
			if string(dst[:n]) != want || dst[n] != '!' {
				t.Fatal(kind, warm, n, "decimal output")
			}
			if !reflect.DeepEqual(a, original) {
				t.Fatal("borrowed input changed")
			}
			budget.FailAfter(-1)
			closeOwner()
			if budget.Used() != 0 {
				t.Fatal(kind, warm, "format leak", budget.Used())
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
	parent, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	serviceMust(t, e)
	caller, e := parent.Allocate(4)
	serviceMust(t, e)
	serviceMust(t, parent.WriteUint32(caller, 0, 123))
	m, e := newNaturalManagerWithMemory(c, parent)
	serviceMust(t, e)
	if m.runtime != parent || m.ownsRuntime || m.budget != b {
		t.Fatal("nested ownership")
	}
	dst := make([]byte, 1000)
	_, e = m.FormatInto([]uint32{0xffffffff, 0xffffffff, 1}, dst)
	serviceMust(t, e)
	measured := parent.Stats().TotalBytes + m.HeaderBytes
	if measured != b.Used() {
		t.Fatal("complete owner graph", measured, b.Used())
	}
	if m.HeaderBytes != uint64(reflect.TypeOf(NaturalManager{}).Size()) {
		t.Fatal("natural header")
	}
	m.Close()
	// Ending the child leaves the caller's Runtime and allocation intact.
	if value, err := parent.ReadUint32(caller, 0); err != nil || value != 123 {
		t.Fatal("parent lifetime", value, err)
	}
	serviceMust(t, parent.Deallocate(&caller))
	serviceMust(t, parent.Close())
	if b.Used() != 0 {
		t.Fatal("complete refund", b.Used())
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
	if n, e := m.FormatInto([]uint32{123}, dst); e != nil || n != 3 || string(dst) != "123hanged" || b.Used() != baseline {
		t.Fatal("native text must work with only owner budget", n, e, string(dst), b.Used())
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
	if oracle(q).Cmp(wq) != 0 || oracle(r).Cmp(wr) != 0 || m.budget.Used() != before {
		t.Fatal("prepared requested capacities", naturalDivisionCapacities(m))
	}
	if m.Division[3] != nil {
		t.Fatal("unused add-back allocated")
	}
	baseline := m.RuntimeOwner().Stats()
	charge := m.budget.Used()
	state := m.RuntimeOwner().State()
	var failure error
	if allocations := testing.AllocsPerRun(100, func() { _, failure = m.Div(a, b, q, r) }); allocations != 0 || failure != nil {
		t.Fatal(allocations, failure)
	}
	if m.RuntimeOwner().State() != state || m.RuntimeOwner().Stats() != baseline || m.budget.Used() != charge {
		t.Fatal("native verification changed warmed physical graph")
	}

}

func TestOwnedTextNativeGrowthPreservesPrefix(t *testing.T) {
	m := naturalManager(t)
	before := m.budget.Used()
	runtimeBefore := m.RuntimeOwner().Stats()
	m.budget.FailAfter(0)
	defer m.budget.FailAfter(-1)
	serviceMust(t, m.reserveText(8))
	copy(m.text, "prefix")
	serviceMust(t, m.reserveText(8192))
	if string(m.text[:6]) != "prefix" || m.budget.Used() != before || m.RuntimeOwner().Stats() != runtimeBefore {
		t.Fatal("native assembly growth changed prefix or charged storage")
	}
	dst := bytes.Repeat([]byte{'?'}, 16)
	n, e := m.FormatInto([]uint32{123}, dst)
	if e != nil || n != 3 || string(dst[:n]) != "123" || dst[n] != '?' {
		t.Fatal("native output", n, e)
	}
	m.Close()
	if m.text != nil || m.budget.Used() != 0 {
		t.Fatal("assembly close retained memory")
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
	if e := rational.Z.SetLimbs(&r.Num, a); e != nil {
		t.Fatal(e)
	}
	r.Den = NewInt(7)
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
