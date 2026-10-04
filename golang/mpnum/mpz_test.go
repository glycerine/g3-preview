// Licensed under the MIT license.
// Source assertions: src/test/mpz.cpp, especially bug3. Added bounded
// lifecycle and exact arithmetic tests cover the Go ownership adaptation.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"math/big"
	"math/rand"
	"reflect"
	"runtime"
	"testing"
)

func integerManager(t *testing.T) (*IntegerManager, *arena.Budget) {
	t.Helper()
	budget := arena.NewBudget(1 << 20)
	m, err := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: budget}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		m.Close()
		if budget.Used() != 0 {
			t.Errorf("close retained %d bytes", budget.Used())
		}
	})
	return m, budget
}
func integerString(t *testing.T, m *IntegerManager, n *Int) string {
	t.Helper()
	s, e := m.String(n)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func TestIntegerSourceBug3(t *testing.T) {
	m, _ := integerManager(t)
	var v1, v2 Int
	if err := m.SetInt64(&v1, math.MinInt32); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt64(&v2, math.MaxInt32); err != nil {
		t.Fatal(err)
	}
	one := NewInt(1)
	if err := m.Add(&v2, &one, &v2); err != nil {
		t.Fatal(err)
	}
	if err := m.Neg(&v1); err != nil {
		t.Fatal(err)
	}
	cmp, err := m.Compare(&v1, &v2)
	if err != nil || cmp != 0 {
		t.Fatal(cmp, err)
	}
	if integerString(t, m, &v1) != "2147483648" {
		t.Fatal("INT_MIN negation")
	}
}

func TestIntegerSourceIntMinSubtraction(t *testing.T) {
	// src/test/mpz.cpp::tst_int_min_bug, actively called by tst_mpz().
	m, _ := integerManager(t)
	small := NewInt(math.MinInt32)
	var big, out Int
	if e := m.SetUint64(&big, math.MaxUint64); e != nil {
		t.Fatal(e)
	}
	if e := m.Sub(&big, &small, &out); e != nil {
		t.Fatal(e)
	}
	if got := integerString(t, m, &out); got != "18446744075857035263" {
		t.Fatal(got)
	}
}

func TestIntegerConstructorOwnership(t *testing.T) {
	m, _ := integerManager(t)
	if m.cells.Stats().Live != 6 || m.words.Stats().Live != 0 {
		t.Fatal("six actual source objects; four actual inline Natural buffers have no allocated payload")
	}
	var objects [6]memory.ObjectRef
	for i, ref := range []numericCellRef{m.tmp[0], m.tmp[1], m.arg[0], m.arg[1], m.intMin.cell, m.two64.cell} {
		c := m.cell(ref)
		if c.capacity != 6 || c.payload != (arena.VectorRef{}) {
			t.Fatal("numeric payload duplicated or source capacity changed")
		}
		object, e := m.cellObject(ref)
		if e != nil {
			t.Fatal(e)
		}
		objects[i] = object
		for _, other := range objects[:i] {
			if other == object {
				t.Fatal("source object aliased")
			}
		}
		extent, e := m.allocator.ObjectExtent(object)
		if e != nil || extent != 32 {
			t.Fatal("actual source extent", extent, e)
		}
		raw, offset, e := m.allocator.RawLocation(object)
		if e != nil {
			t.Fatal(e)
		}
		capacity, e := m.memory.ReadUint32(raw, offset+4)
		if e != nil || capacity != 6 {
			t.Fatal("source capacity header", capacity, e)
		}
	}
	physical := m.memory.Stats()
	count, e := m.memory.AllocationCount()
	if e != nil {
		t.Fatal(e)
	}
	header := uint64(m.memory.SizeTBits() / 8)
	if modularSourceDebug && runtime.GOOS != "windows" {
		if physical.LiveBlocks != 6 || physical.PhysicalBytes != 6*(32+header) || count != 6 {
			t.Fatal("source debug direct topology", physical, count)
		}
	} else {
		if physical.LiveBlocks != 1 || physical.PhysicalBytes != 8192+header || count != 1 {
			t.Fatal("source pooled topology", physical, count)
		}
	}
	if integerString(t, m, &m.intMin) != "2147483648" || integerString(t, m, &m.two64) != "18446744073709551616" {
		t.Fatal("constructor constants")
	}
	// Concrete record audit includes the widened checked reference layout.
	for _, recordType := range []reflect.Type{reflect.TypeOf(Int{}), reflect.TypeOf(numericCell{})} {
		if e := arena.ValidateRecordType(recordType); e != nil {
			t.Fatal(e)
		}
	}
	if e := arena.ValidateRecordType(reflect.TypeOf(struct{ Pointer *int }{})); !errors.Is(e, arena.ErrPointerful) {
		t.Fatal("pointerful record accepted", e)
	}
}
func TestIntegerSetConversionBoundaries(t *testing.T) {
	m, _ := integerManager(t)
	var n Int
	for _, v := range []int64{0, 1, -1, math.MinInt32, math.MaxInt32, int64(math.MaxInt32) + 1, int64(math.MinInt32) - 1, math.MaxInt64, math.MinInt64} {
		if err := m.SetInt64(&n, v); err != nil {
			t.Fatal(err)
		}
		if integerString(t, m, &n) != big.NewInt(v).String() {
			t.Fatal(v)
		}
		got, e := m.Int64(&n)
		if e != nil || got != v {
			t.Fatal(got, e, v)
		}
	}
	for _, v := range []uint64{0, math.MaxInt32, math.MaxUint32, math.MaxUint64} {
		if err := m.SetUint64(&n, v); err != nil {
			t.Fatal(err)
		}
		got, e := m.Uint64(&n)
		if e != nil || got != v {
			t.Fatal(got, e, v)
		}
		if v == math.MaxInt32 && n.IsSmall() {
			t.Fatal("source uint64 strict < INT_MAX threshold")
		}
	}
	if err := m.SetLimbs(&n, []uint32{0, 0, 1, 0}); err != nil {
		t.Fatal(err)
	}
	if integerString(t, m, &n) != "18446744073709551616" {
		t.Fatal("two64")
	}
	if ok, e := m.IsUint64(&n); e != nil || ok {
		t.Fatal(ok, e)
	}
	i, ie := m.Int64(&n)
	u, ue := m.Uint64(&n)
	integerPhysicalAudit(t, m, &n)
	if modularSourceDebug {
		if !errors.Is(ie, ErrIntegerAssertion) || !errors.Is(ue, ErrIntegerAssertion) {
			t.Fatal("source oversized getter assertions", ie, ue)
		}
	} else if ie != nil || ue != nil || i != 0 || u != 0 {
		t.Fatal("source low two words of 2^64", i, u, ie, ue)
	}
}
func TestIntegerOwnership(t *testing.T) {
	m, _ := integerManager(t)
	var a, b, c Int
	if e := m.SetUint64(&a, math.MaxUint64); e != nil {
		t.Fatal(e)
	}
	if e := m.Copy(&a, &a); e != nil {
		t.Fatal(e)
	}
	if e := m.Copy(&b, &a); e != nil {
		t.Fatal(e)
	}
	if a.cell == b.cell {
		t.Fatal("copy reused owned cell")
	}
	if e := m.Neg(&a); e != nil {
		t.Fatal(e)
	}
	if integerString(t, m, &b) != "18446744073709551615" {
		t.Fatal("copy alias")
	}
	if e := m.Move(&c, &a); e != nil {
		t.Fatal(e)
	}
	if !a.IsSmall() || a.val != -1 {
		t.Fatal("source moved-from state retains sign")
	}
	if e := m.Abs(&c); e != nil {
		t.Fatal(e)
	}
	if e := m.Swap(&b, &c); e != nil {
		t.Fatal(e)
	}
	if e := m.Del(&b); e != nil {
		t.Fatal(e)
	}
	if !b.IsSmall() || b.val != 1 {
		t.Fatal("source Del preserves sign")
	}
	if e := m.Del(&b); e != nil {
		t.Fatal(e)
	}
	if e := m.Reset(&b); e != nil || b.val != 0 {
		t.Fatal(e)
	}
	if e := m.Reset(&c); e != nil {
		t.Fatal(e)
	}
}
func TestIntegerArithmeticOracle(t *testing.T) {
	m, _ := integerManager(t)
	r := rand.New(rand.NewSource(0x4d505a))
	var a, b, out Int
	for trial := 0; trial < 300; trial++ {
		x, y := int64(r.Uint64()), int64(r.Uint64())
		for _, sub := range []bool{false, true} {
			for alias := 0; alias < 3; alias++ {
				if e := m.SetInt64(&a, x); e != nil {
					t.Fatal(e)
				}
				if e := m.SetInt64(&b, y); e != nil {
					t.Fatal(e)
				}
				dst := &out
				if alias == 1 {
					dst = &a
				}
				if alias == 2 {
					dst = &b
				}
				var e error
				expected := new(big.Int)
				if sub {
					e = m.Sub(&a, &b, dst)
					expected.Sub(big.NewInt(x), big.NewInt(y))
				} else {
					e = m.Add(&a, &b, dst)
					expected.Add(big.NewInt(x), big.NewInt(y))
				}
				if e != nil || integerString(t, m, dst) != expected.String() {
					t.Fatalf("trial %d alias %d sub %v: %v", trial, alias, sub, e)
				}
			}
		}
		m.SetInt64(&a, x)
		m.SetInt64(&b, y)
		cmp, e := m.Compare(&a, &b)
		if e != nil || cmp != big.NewInt(x).Cmp(big.NewInt(y)) {
			t.Fatal(cmp, e)
		}
	}
}
func TestIntegerFailureCleanup(t *testing.T) {
	for after := int64(0); after < 20; after++ {
		budget := arena.NewBudget(1 << 20)
		budget.FailAfter(after)
		m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 1, Budget: budget}, Words: arena.VectorConfig{PageSize: 8, Descriptors: arena.Config{PageSize: 1, Budget: budget}}})
		if e == nil {
			m.Close()
		} else {
			// Audit the actual retained constructor owner before explicit cleanup.
			// Source member unwind can leave direct allocations alive.
			constructionFailureOwningCleanup(t, e, budget)
		}
		if budget.Used() != 0 {
			t.Fatalf("constructor failure %d leaked %d", after, budget.Used())
		}
	}
	m, budget := integerManager(t)
	a := NewInt(17)
	budget.FailAfter(0)
	// Force enough words to need a new capacity class and descriptor/cell pages.
	before := m.cells.Stats().Live
	err := m.SetLimbs(&a, []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17})
	if !numericPhysicalReservationRefusal(err) {
		t.Fatal(err)
	}
	budget.FailAfter(-1)
	if a.val != 1 || a.cell != m.failedRef || m.cells.Stats().Live != before {
		t.Fatal("source digit setter sign/failure token or roots", a)
	}
	if _, e := m.Int64(&a); e != arena.ErrInvalidSlot {
		t.Fatal("failed source pointer read", e)
	}
	m.Close()
	if _, e := m.String(&a); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestIntegerFailurePublication(t *testing.T) {
	for after := int64(0); after < 6; after++ {
		m, budget := integerManager(t)
		a := NewInt(17)
		before := m.cells.Stats().Live
		beforeWords := m.words.Stats().Live
		budget.FailAfter(after)
		digits := make([]uint32, 33)
		digits[len(digits)-1] = 1
		err := m.SetLimbs(&a, digits)
		budget.FailAfter(-1)
		if err != nil {
			if a.val != 1 || a.cell != m.failedRef || m.cells.Stats().Live != before || m.words.Stats().Live != beforeWords {
				t.Fatalf("source allocation failure %d sign/token/roots", after)
			}
			if _, e := m.Int64(&a); e != arena.ErrInvalidSlot {
				t.Fatal("invalid source publication read", e)
			}
		}
	}
	m, budget := mpzPublicationDirectOwner(t)
	out := NewInt(17)
	one := NewInt(1)
	budget.FailAfter(0)
	// mpz.cpp:274–279 publishes the big result before requesting missing
	// scratch. Failure leaves a valid output and an empty manager scratch.
	err := m.Add(&m.two64, &one, &out)
	if !numericPhysicalReservationRefusal(err) || out.cell == 0 || out.val != 1 || m.tmp[0] != 0 || m.cell(out.cell).size != 3 || m.digit(out.cell, 0) != 1 || m.digit(out.cell, 2) != 1 {
		t.Fatal("source Add publication prefix", err)
	}
	if e := m.Add(&m.two64, &one, &out); e != arena.ErrInvalidSlot {
		t.Fatal("missing scratch guard", e)
	}
	budget.FailAfter(-1)
	if integerString(t, m, &out) != "18446744073709551617" {
		t.Fatal("published result mathematics")
	}
	m.Close()
	// Retain the original successful mathematical assertion on a fresh owner;
	// native source does not provide a retry contract for its dangling scratch.
	m, _ = integerManager(t)
	out = NewInt(17)
	if err = m.Add(&m.two64, &one, &out); err != nil || integerString(t, m, &out) != "18446744073709551617" {
		t.Fatal("source successful Add", err)
	}
}

func TestIntegerMultiLimbOracleAndSourceBug1(t *testing.T) {
	m, _ := integerManager(t)
	r := rand.New(rand.NewSource(0x43454c4c))
	var a, b, out, copy Int
	// Added independent coverage: 200 signed magnitude cases, at most 32 limbs.
	for trial := 0; trial < 200; trial++ {
		aa, bb := make([]uint32, 1+r.Intn(32)), make([]uint32, 1+r.Intn(32))
		for i := range aa {
			aa[i] = r.Uint32()
		}
		for i := range bb {
			bb[i] = r.Uint32()
		}
		av, bv := oracle(aa), oracle(bb)
		if e := m.SetLimbs(&a, aa); e != nil {
			t.Fatal(e)
		}
		if e := m.SetLimbs(&b, bb); e != nil {
			t.Fatal(e)
		}
		if trial%2 == 0 {
			if e := m.Neg(&a); e != nil {
				t.Fatal(e)
			}
			av.Neg(av)
		}
		if trial%3 == 0 {
			if e := m.Neg(&b); e != nil {
				t.Fatal(e)
			}
			bv.Neg(bv)
		}
		if e := m.Copy(&copy, &a); e != nil || integerString(t, m, &copy) != av.String() {
			t.Fatal("copy", e)
		}
		if e := m.Add(&a, &b, &out); e != nil || integerString(t, m, &out) != new(big.Int).Add(av, bv).String() {
			t.Fatal("add", trial, e)
		}
		if e := m.Sub(&a, &b, &out); e != nil || integerString(t, m, &out) != new(big.Int).Sub(av, bv).String() {
			t.Fatal("sub", trial, e)
		}
		cmp, e := m.Compare(&a, &b)
		if e != nil || cmp != av.Cmp(bv) {
			t.Fatal("compare", trial, e)
		}
	}
	// src/test/mpz.cpp::bug1 exact input, full manager self-subtraction. Decimal
	// fixture parsing remains independent test setup until source set(string).
	expected, _ := new(big.Int).SetString("1002043949858757875676767675747473", 10)
	if e := m.SetLimbs(&a, wantLimbs(expected, 4)); e != nil {
		t.Fatal(e)
	}
	if e := m.Sub(&a, &a, &out); e != nil || !out.IsSmall() || integerString(t, m, &out) != "0" {
		t.Fatal("source bug1", e)
	}
}

func TestIntegerSmallAndConversionSemantics(t *testing.T) {
	m, _ := integerManager(t)
	zero := NewInt(0)
	one := NewInt(1)
	negative := NewInt(-1)
	var out Int
	if e := m.Add(&zero, &one, &out); e != nil || integerString(t, m, &out) != "1" {
		t.Fatal(e)
	}
	if e := m.Sub(&one, &one, &one); e != nil || one.val != 0 {
		t.Fatal(e)
	}
	if e := m.Copy(&out, &zero); e != nil || out.val != 0 {
		t.Fatal(e)
	}
	if e := m.Move(&out, &out); e != nil {
		t.Fatal(e)
	}
	if e := m.Abs(&negative); e != nil || negative.val != 1 {
		t.Fatal(e)
	}
	negative = NewInt(math.MinInt32)
	if e := m.Abs(&negative); e != nil || integerString(t, m, &negative) != "2147483648" {
		t.Fatal(e)
	}
	negative = NewInt(-7)
	if ok, e := m.IsUint64(&negative); e != nil || ok {
		t.Fatal(ok, e)
	}
	if got, e := m.Uint64(&negative); e != nil || got != math.MaxUint64-6 {
		t.Fatal(got, e)
	}
	m.SetInt64(&negative, -(1 << 40))
	if got, e := m.Uint64(&negative); e != nil || got != 1<<40 {
		t.Fatal("source big accessor returns absolute", got, e)
	}
	if e := m.Neg(&negative); e != nil {
		t.Fatal(e)
	}
	if e := m.SetLimbs(&out, []uint32{0, 0}); e != nil || out.val != 0 {
		t.Fatal(e)
	}
	if e := m.SetLimbs(&out, []uint32{5, 0}); e != nil || out.val != 5 {
		t.Fatal(e)
	}
	if e := m.SetLimbs(&out, []uint32{math.MaxUint32}); e != nil || integerString(t, m, &out) != "4294967295" {
		t.Fatal(e)
	}
	if _, e := m.String(nil); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	if _, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{Budget: arena.NewBudget(100)}}); !errors.Is(e, arena.ErrConfig) {
		t.Fatal(e)
	}
}
