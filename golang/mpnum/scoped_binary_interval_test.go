// Copyright (c) 2011 Microsoft Corporation. Concrete lexical-owner tests.
package mpnum

import (
	"bytes"
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"testing"
)

func TestBinaryIntervalFailurePrefixes(t *testing.T) {
	// Sweep actual reservations to success; persistent scratch belongs to owners.
	// Every sweep audits physical cells and Words before any rendering/teardown.
	for op := 0; op < 8; op++ {
		for alias := 0; alias < 3; alias++ {
			if op < 3 && alias != 0 {
				continue
			}
			success, failed := false, false
			for prefix := int64(0); prefix < 256; prefix++ {
				m := intervalManager(t)
				z := m.bounds.z
				b := z.budget
				var a, c, d BinaryInterval
				var large Int
				serviceMust(t, z.SetString(&large, "340282366920938463463374607431768211457"))
				serviceMust(t, m.SetInteger(&a, &large))
				serviceMust(t, m.SetInteger(&d, &large))
				n := NewInt(3)
				serviceMust(t, m.AddInteger(&a, &n, &a))
				serviceMust(t, m.AddInteger(&d, &n, &d))
				if op == 4 {
					serviceMust(t, m.Neg(&d))
				}
				intervalSet(t, m, &c, 7, 9)
				out := &c
				if alias == 1 {
					out = &a
				}
				if alias == 2 {
					out = &d
				}
				b.FailAfter(prefix)
				var e error
				switch op {
				case 0:
					e = m.Copy(out, &a)
				case 1:
					e = m.SetPoint(out, &a.lower)
				case 2:
					e = m.SetInteger(out, &large)
				case 3:
					e = m.Add(&a, &d, out)
				case 4:
					e = m.Sub(&a, &d, out)
				case 5:
					e = m.Mul(&a, &d, out)
				case 6:
					e = m.Power(&a, 3, out)
				case 7:
					e = m.AddInteger(&a, &large, out)
				}
				if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal(op, alias, prefix, e)
				}
				extra := BinaryInterval{lower: BinaryRational{num: large}} // Nonowning audit view.
				intervalAudit(t, m, &a, &c, &d, &extra)
				b.FailAfter(-1)
				if op < 3 && e != nil {
					wanted := &a.lower.num
					if op == 2 {
						wanted = &large
					}
					if c.lower.num.cell == z.failedRef {
						if c.lower.num.val != wanted.val || c.lower.k != 0 || c.upper.num != NewInt(9) {
							t.Fatal("lower allocation publication precedes upper copy", op, prefix, c)
						}
						if x := z.check(&c.lower.num); !errors.Is(x, arena.ErrInvalidSlot) {
							t.Fatal("failed lower read guard", x)
						}
					} else {
						lowerNew, x := z.Equal(&c.lower.num, wanted)
						serviceMust(t, x)
						if !lowerNew || c.upper.num.cell != z.failedRef || c.upper.num.val != wanted.val || c.upper.k != 0 {
							t.Fatal("completed lower precedes upper allocation publication", op, prefix, c)
						}
						if x := z.check(&c.upper.num); !errors.Is(x, arena.ErrInvalidSlot) {
							t.Fatal("failed upper read guard", x)
						}
					}
				}
				if e != nil {
					failed = true
				} else {
					success = true
				}
				serviceMust(t, m.Del(&a))
				serviceMust(t, m.Del(&c))
				serviceMust(t, m.Del(&d))
				serviceMust(t, z.Del(&large))
				m.Close()
				m.bounds.Close()
				z.Close()
				serviceMust(t, m.bounds.diagnostics.Close())
				if b.Used() != 0 {
					t.Fatal("closed graph", op, alias, prefix, b.Used())
				}
				if success {
					break
				}
			}
			if !success {
				t.Fatal("missing successful sweep", op, alias)
			}
			if !failed && alias == 0 {
				t.Fatal("fixture did not exercise refusal", op)
			}
		}
	}
	// Neg lower publication precedes failing INT_MIN negation on upper.
	m := intervalManager(t)
	a := BinaryInterval{lower: NewBinaryInt(-5), upper: NewBinaryInt(-2147483648)}
	m.bounds.z.budget.FailAfter(0)
	e := m.Neg(&a)
	m.bounds.z.budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || a.lower.num.val != 5 || a.upper.num.val != -2147483648 {
		t.Fatal("neg source prefix", e, a)
	}
	intervalAudit(t, m, &a)
	serviceMust(t, m.Del(&a))
	// Power publishes lower1 before the upper source debug assertion; release
	// executes upper Reset0. Both retain the same completed endpoint values.
	a = BinaryInterval{lower: NewBinaryInt(2), upper: NewBinaryInt(0)}
	var c BinaryInterval
	e = m.Power(&a, 0, &c)
	intervalAudit(t, m, &a, &c)
	if (sourceExtendedGCDDebug && !errors.Is(e, ErrInvalidPower)) || (!sourceExtendedGCDDebug && e != nil) || c.lower.num.val != 1 || c.upper.num.val != 0 {
		t.Fatal("power source prefix", e, c)
	}
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.Del(&c))
}

func TestBinaryIntervalFormattingAndWarmStorage(t *testing.T) {
	m := intervalManager(t)
	var a, b, c BinaryInterval
	defer m.Del(&a)
	defer m.Del(&b)
	defer m.Del(&c)
	lo, hi := NewBinaryIntPower(-123, 5), NewBinaryIntPower(456, 2)
	serviceMust(t, m.SetBounds(&a, &lo, &hi))
	intervalText(t, m, &a, "(-123/2^5, 456/2^2)")
	var dst [128]byte
	copy(dst[:], bytes.Repeat([]byte{'!'}, 128))
	before := dst
	if _, e := m.FormatInto(&a, dst[:2]); e == nil || dst != before {
		t.Fatal("short buffer publication", e)
	}
	intervalSet(t, m, &a, -2, 3)
	intervalSet(t, m, &b, -4, 5)
	for i := 0; i < 3; i++ {
		serviceMust(t, m.Mul(&a, &b, &c))
		serviceMust(t, m.Add(&a, &b, &c))
		serviceMust(t, m.Power(&a, 2, &c))
		_, e := m.FormatInto(&c, dst[:])
		serviceMust(t, e)
	}
	used := m.bounds.z.budget.Used()
	m.bounds.z.budget.FailAfter(0)
	alloc := testing.AllocsPerRun(30, func() {
		if e := m.Mul(&a, &b, &c); e != nil {
			panic(e)
		}
		if _, e := m.FormatInto(&c, dst[:]); e != nil {
			panic(e)
		}
	})
	m.bounds.z.budget.FailAfter(-1)
	if alloc != 0 || used != m.bounds.z.budget.Used() {
		t.Fatal("warm owner", alloc, used, m.bounds.z.budget.Used())
	}
	intervalAudit(t, m, &a, &b, &c)
	// Force formatter growth and ensure unpublished caller output on refusal.
	var n Int
	two := NewInt(2)
	serviceMust(t, m.bounds.z.Power(&two, 1000, &n))
	serviceMust(t, m.SetInteger(&a, &n))
	serviceMust(t, m.bounds.z.Del(&n))
	before = dst
	m.bounds.z.budget.FailAfter(0)
	_, e := m.FormatInto(&a, dst[:])
	m.bounds.z.budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || dst != before {
		t.Fatal("format growth publication", e)
	}
	intervalAudit(t, m, &a, &b, &c)
}

func TestBinaryIntervalPredicateShortCircuits(t *testing.T) {
	m := intervalManager(t)
	a := BinaryInterval{lower: NewBinaryIntPower(1, 300), upper: NewBinaryInt(2)}
	b := BinaryInterval{lower: NewBinaryInt(1), upper: NewBinaryInt(3)}
	m.bounds.z.budget.FailAfter(0)
	ok, e := m.Disjoint(&a, &b)
	m.bounds.z.budget.FailAfter(-1)
	serviceMust(t, e)
	if !ok {
		t.Fatal("first true disjoint")
	}
	// Directly demonstrate the omitted comparison would require growth.
	m.bounds.z.budget.FailAfter(0)
	_, e = m.bounds.LessEqual(&b.upper, &a.lower)
	m.bounds.z.budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("second comparison did not require storage", e)
	}
	intervalAudit(t, m, &a, &b)
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.Del(&b))
}

func intervalAudit(t *testing.T, m *BinaryIntervalManager, vs ...*BinaryInterval) {
	t.Helper()
	roots := []*BinaryRational{&m.mulCurr, &m.mulMax, &m.mulMin}
	for _, v := range vs {
		roots = append(roots, &v.lower, &v.upper)
	}
	binaryAudit(t, m.bounds, roots...)
}
func TestBinaryIntervalOwnerAndScope(t *testing.T) {
	m := intervalManager(t)
	b := m.bounds.z.budget
	baseline := b.Used()
	b.FailAfter(0)
	if _, e := NewBinaryIntervalManager(m.bounds); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if b.Used() != baseline {
		t.Fatal("constructor graph")
	}
	b.FailAfter(-1)
	b.FailAfter(0)
	if _, e := NewScopedBinaryInterval(m); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if b.Used() != baseline {
		t.Fatal("scope refused header")
	}
	b.FailAfter(-1)
	n, e := NewBinaryIntervalManager(m.bounds)
	serviceMust(t, e)
	if b.Used()-baseline != uint64(reflect.TypeOf(BinaryIntervalManager{}).Size()) {
		t.Fatal("header")
	}
	n.Close()
	n.Close()
	if b.Used() != baseline {
		t.Fatal("borrowed close")
	}
	serviceMust(t, arena.ValidateRecordType(reflect.TypeOf(BinaryInterval{})))
	s, e := NewScopedBinaryInterval(m)
	serviceMust(t, e)
	v, e := s.Get()
	serviceMust(t, e)
	intervalSet(t, m, v, -2, 3)
	copied := *s
	if _, e = copied.Get(); !errors.Is(e, ErrScopedCopy) {
		t.Fatal(e)
	}
	lo, e := s.Lower()
	serviceMust(t, e)
	if lo != &v.lower {
		t.Fatal("borrow")
	}
	hi, e := s.Upper()
	serviceMust(t, e)
	if hi != &v.upper {
		t.Fatal("borrow")
	}
	var a BinaryInterval
	serviceMust(t, s.SwapValue(&a))
	intervalText(t, m, &a, "(-2, 3)")
	serviceMust(t, m.Del(&a))
	serviceMust(t, s.Reset())
	var copiedBound BinaryRational
	serviceMust(t, s.LowerCopy(&copiedBound))
	serviceMust(t, s.UpperCopy(&copiedBound))
	serviceMust(t, m.bounds.Del(&copiedBound))
	manager, e := s.Manager()
	serviceMust(t, e)
	if manager != m {
		t.Fatal("scoped owner accessor")
	}
	var text [16]byte
	length, e := s.FormatInto(text[:])
	serviceMust(t, e)
	if string(text[:length]) != "(0, 0)" {
		t.Fatal("scoped formatter")
	}
	str, e := s.String()
	serviceMust(t, e)
	if str != "(0, 0)" {
		t.Fatal(str)
	}
	other, e := NewScopedBinaryInterval(m)
	serviceMust(t, e)
	ov, e := other.Get()
	serviceMust(t, e)
	intervalSet(t, m, ov, 2, 4)
	serviceMust(t, s.Swap(other))
	intervalText(t, m, v, "(2, 4)")
	serviceMust(t, other.Close())
	intervalAudit(t, m, v)
	serviceMust(t, s.Close())
	serviceMust(t, s.Close())
	if _, e = s.Get(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestBinaryIntervalRetainedProductsAndDeepCopies(t *testing.T) {
	m := intervalManager(t)
	var a, b, c BinaryInterval
	var n Int
	serviceMust(t, m.bounds.z.SetString(&n, "340282366920938463463374607431768211457"))
	serviceMust(t, m.SetInteger(&c, &n))
	serviceMust(t, m.bounds.z.Del(&n))
	oldLower, oldUpper := c.lower.num.cell, c.upper.num.cell
	intervalSet(t, m, &a, -2, 3)
	intervalSet(t, m, &b, -4, 5)
	serviceMust(t, m.Mul(&a, &b, &c))
	if m.mulMin.num.cell != oldLower || m.mulMax.num.cell != oldUpper {
		t.Fatal("source old outputs retained by Swap")
	}
	stats, e := m.Stats()
	serviceMust(t, e)
	if stats.LiveScratchNumerators != 2 || stats.HeaderBytes != uint64(reflect.TypeOf(BinaryIntervalManager{}).Size()) {
		t.Fatal(stats)
	}
	intervalAudit(t, m, &a, &b, &c)
	serviceMust(t, m.Copy(&a, &c))
	serviceMust(t, m.Swap(&a, &a))
	intervalText(t, m, &a, "(-12, 15)")
	serviceMust(t, m.bounds.z.SetString(&n, "680564733841876926926749214863536422913"))
	serviceMust(t, m.SetInteger(&a, &n))
	serviceMust(t, m.bounds.z.Del(&n))
	serviceMust(t, m.Copy(&b, &a))
	if a.lower.num.cell == b.lower.num.cell || a.upper.num.cell == b.upper.num.cell {
		t.Fatal("deep endpoint ownership")
	}
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.Del(&a))
	intervalText(t, m, &b, "(680564733841876926926749214863536422913, 680564733841876926926749214863536422913)")
	intervalAudit(t, m, &a, &b, &c)
	serviceMust(t, m.Del(&b))
	serviceMust(t, m.Del(&c))
	copied := *m
	if _, e := copied.Bounds(); !errors.Is(e, ErrBinaryOwner) {
		t.Fatal(e)
	}
	copied.Close()
	foreign, e := NewBinaryIntervalManager(m.bounds)
	serviceMust(t, e)
	s, e := NewScopedBinaryInterval(m)
	serviceMust(t, e)
	o, e := NewScopedBinaryInterval(foreign)
	serviceMust(t, e)
	if e = s.Swap(o); !errors.Is(e, ErrScopedOwner) {
		t.Fatal(e)
	}
	serviceMust(t, o.Close())
	serviceMust(t, s.Close())
	foreign.Close()
	m.Close()
	if _, e = m.Stats(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if _, e = NewBinaryIntervalManager(nil); !errors.Is(e, ErrBinaryOwner) {
		t.Fatal(e)
	}
}
