// Source: src/util/scoped_numeral.h, src/util/scoped_numeral_vector.h.
package util

import (
	"errors"
	"math/rand"
	"reflect"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestScopedNumeralSourceTstScoped(t *testing.T) {
	m, budget := integerManager(t)
	baseline := m.Cells.Stats().Live
	a, e := NewScopedInt(m)
	if e != nil {
		t.Fatal(e)
	}
	b, e := NewScopedInt(m)
	if e != nil {
		t.Fatal(e)
	}
	x, _ := a.Get()
	y, _ := b.Get()
	if e = m.SetInt64(x, 1000231); e != nil {
		t.Fatal(e)
	}
	if e = m.SetString(y, "102928187172727273"); e != nil {
		t.Fatal(e)
	}
	first, e := a.Display()
	if e != nil {
		t.Fatal(e)
	}
	second, e := b.Display()
	if e != nil {
		t.Fatal(e)
	}
	if e = b.MulAssign(x); e != nil {
		t.Fatal(e)
	}
	third, e := b.Display()
	if e != nil {
		t.Fatal(e)
	}
	if "a: "+first+"\nb: "+second+"\nb: "+third+"\n" != "a: 1000231\nb: 102928187172727273\nb: 102951963583964173000063\n" {
		t.Fatal(first, second, third)
	}
	a.Close()
	b.Close()
	if m.Cells.Stats().Live != baseline {
		t.Fatal("source destructors did not balance")
	}
	m.Close()
	if budget.Used() != 0 {
		t.Fatal(budget.Used())
	}
}

func testScopedNumeralManager(deleted *[]int) *ScopedNumeralManager {
	binary := func(operation func(int, int) int) func(*any, any, any) error {
		return func(out *any, left, right any) error {
			*out = operation(left.(int), right.(int))
			return nil
		}
	}
	return &ScopedNumeralManager{
		New:    func() any { return 0 },
		Delete: func(value *any) error { *deleted = append(*deleted, (*value).(int)); *value = 0; return nil },
		Set:    func(out *any, value any) error { *out = value.(int); return nil },
		SetInt: func(out *any, value int) error { *out = value; return nil },
		Reset:  func(out *any) error { *out = 0; return nil },
		Add:    binary(func(a, b int) int { return a + b }),
		Sub:    binary(func(a, b int) int { return a - b }),
		Mul:    binary(func(a, b int) int { return a * b }),
		Div:    binary(func(a, b int) int { return a / b }),
		Rem:    binary(func(a, b int) int { return a % b }),
		Neg:    func(value *any) error { *value = -(*value).(int); return nil },
		Abs: func(value *any) error {
			if number := (*value).(int); number < 0 {
				*value = -number
			}
			return nil
		},
		Compare: func(left, right any) (int, error) {
			a, b := left.(int), right.(int)
			if a < b {
				return -1, nil
			}
			if a > b {
				return 1, nil
			}
			return 0, nil
		},
		Sign: func(value any) (int, error) {
			number := value.(int)
			if number < 0 {
				return -1, nil
			}
			if number > 0 {
				return 1, nil
			}
			return 0, nil
		},
		Display: func(value any) (string, error) { return strconv.Itoa(value.(int)), nil },
	}
}

func TestGenericScopedNumeralCompleteOperations(t *testing.T) {
	var deleted []int
	manager := testScopedNumeralManager(&deleted)
	scope, err := NewScopedNumeral(manager)
	if err != nil {
		t.Fatal(err)
	}
	if err = scope.SetInt(-12); err != nil {
		t.Fatal(err)
	}
	abs, err := scope.AbsCopy()
	if err != nil {
		t.Fatal(err)
	}
	if text, _ := abs.Display(); text != "12" {
		t.Fatalf("abs = %s", text)
	}
	_ = abs.Close()

	for _, check := range []struct {
		name string
		want string
		op   func() error
	}{
		{"add", "13", func() error { _ = scope.SetInt(10); return scope.AddAssign(3) }},
		{"sub", "7", func() error { _ = scope.SetInt(10); return scope.SubAssign(3) }},
		{"mul", "30", func() error { _ = scope.SetInt(10); return scope.MulAssign(3) }},
		{"div", "3", func() error { _ = scope.SetInt(10); return scope.DivAssign(3) }},
		{"rem", "1", func() error { _ = scope.SetInt(10); return scope.RemAssign(3) }},
	} {
		if err = check.op(); err != nil {
			t.Fatal(check.name, err)
		}
		if text, _ := scope.Display(); text != check.want {
			t.Fatalf("%s = %s, want %s", check.name, text, check.want)
		}
	}
	_ = scope.SetInt(-5)
	if zero, _ := scope.IsZero(); zero {
		t.Fatal("negative value is zero")
	}
	if negative, _ := scope.IsNeg(); !negative {
		t.Fatal("negative predicate")
	}
	if nonpositive, _ := scope.IsNonpos(); !nonpositive {
		t.Fatal("nonpositive predicate")
	}
	if less, _ := scope.Less(0); !less {
		t.Fatal("comparison")
	}
	if equal, _ := scope.Equal(-5); !equal {
		t.Fatal("equality")
	}
	if unequal, _ := scope.NotEqual(0); !unequal {
		t.Fatal("inequality")
	}
	if greater, _ := scope.Greater(-6); !greater {
		t.Fatal("greater comparison")
	}
	if lessEqual, _ := scope.LessEqual(-5); !lessEqual {
		t.Fatal("less-equal comparison")
	}
	if greaterEqual, _ := scope.GreaterEqual(-5); !greaterEqual {
		t.Fatal("greater-equal comparison")
	}
	if positive, _ := scope.IsPos(); positive {
		t.Fatal("negative value is positive")
	}
	if nonnegative, _ := scope.IsNonneg(); nonnegative {
		t.Fatal("negative value is nonnegative")
	}
	if err = scope.Neg(); err != nil {
		t.Fatal(err)
	}
	result, err := scope.Add(7)
	if err != nil {
		t.Fatal(err)
	}
	if text, _ := result.Display(); text != "12" {
		t.Fatalf("copy arithmetic = %s", text)
	}
	if text, _ := scope.Display(); text != "5" {
		t.Fatalf("copy arithmetic changed input: %s", text)
	}
	for _, check := range []struct {
		name string
		want string
		make func() (*ScopedNumeral, error)
	}{
		{"sub", "3", func() (*ScopedNumeral, error) { return scope.Sub(2) }},
		{"mul", "10", func() (*ScopedNumeral, error) { return scope.Mul(2) }},
		{"div", "2", func() (*ScopedNumeral, error) { return scope.Div(2) }},
	} {
		copyResult, copyErr := check.make()
		if copyErr != nil {
			t.Fatal(check.name, copyErr)
		}
		if text, _ := copyResult.Display(); text != check.want {
			t.Fatalf("copy %s = %s, want %s", check.name, text, check.want)
		}
		_ = copyResult.Close()
	}
	_ = result.Close()
	_ = scope.Close()
	if !reflect.DeepEqual(deleted, []int{12, 3, 10, 2, 12, 5}) {
		t.Fatalf("deleted = %v", deleted)
	}
}

func TestGenericScopedNumeralCopyAssignmentAndSwap(t *testing.T) {
	var deletedA, deletedB []int
	managerA := testScopedNumeralManager(&deletedA)
	managerB := testScopedNumeralManager(&deletedB)
	left, _ := NewScopedNumeral(managerA)
	right, _ := NewScopedNumeral(managerB)
	_ = left.SetInt(4)
	_ = right.SetInt(9)
	if manager, _ := left.Manager(); manager != managerA {
		t.Fatal("manager identity")
	}
	if value, _ := left.Get(); (*value).(int) != 4 {
		t.Fatal("mutable get")
	}
	if err := left.Assign(right); err != nil {
		t.Fatal(err)
	}
	clone, err := left.Clone()
	if err != nil {
		t.Fatal(err)
	}
	if err = left.Swap(right); err != nil {
		t.Fatal(err)
	}
	value := any(13)
	if err = right.SwapValue(&value); err != nil {
		t.Fatal(err)
	}
	if left.num != 9 || right.num != 13 || value != 9 || clone.num != 9 {
		t.Fatalf("values = %v %v %v %v", left.num, right.num, value, clone.num)
	}
	_ = clone.Reset()
	_ = clone.Close()
	_ = left.Close()
	_ = right.Close()
}

func TestGenericScopedNumeralOwnerAndManagerFailures(t *testing.T) {
	var nilScope *ScopedNumeral
	if _, err := nilScope.Get(); !errors.Is(err, ErrScopedCopy) {
		t.Fatal("nil owner", err)
	}
	if scope, err := NewScopedNumeral(nil); scope != nil || !errors.Is(err, ErrScopedOwner) {
		t.Fatalf("nil manager = %#v, %v", scope, err)
	}
	incomplete := testScopedNumeralManager(new([]int))
	incomplete.Div = nil
	if scope, err := NewScopedNumeral(incomplete); scope != nil || !errors.Is(err, ErrScopedOwner) {
		t.Fatalf("incomplete manager = %#v, %v", scope, err)
	}
	var deleted []int
	scope, _ := NewScopedNumeral(testScopedNumeralManager(&deleted))
	copyOfScope := *scope
	if _, err := copyOfScope.Get(); !errors.Is(err, ErrScopedCopy) {
		t.Fatal("copied owner", err)
	}
	if err := scope.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := scope.Get(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed owner", err)
	}

	deleteFailure := errors.New("delete failure")
	failingManager := testScopedNumeralManager(new([]int))
	failingManager.Delete = func(*any) error { return deleteFailure }
	failing, _ := NewScopedNumeral(failingManager)
	if err := failing.Close(); !errors.Is(err, deleteFailure) || failing.closed {
		t.Fatal("failed delete published close", err)
	}
	failingManager.Delete = func(*any) error { return nil }
	if err := failing.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestScopedNumeralResizeFailurePrefixesAndBudget(t *testing.T) {
	for prefix := int64(0); prefix < 4; prefix++ {
		m, b := integerManager(t)
		base := b.Used()
		v, e := NewScopedIntVector(m)
		if e != nil {
			t.Fatal(e)
		}
		header := uint64(reflect.TypeOf(ScopedIntVector{}).Size())
		width := uint64(reflect.TypeOf(Int{}).Size())
		if b.Used() != base+header {
			t.Fatal("owner header charge")
		}
		b.FailAfter(prefix)
		e = v.Resize(5)
		wantCap := []uint32{0, 2, 3, 5}[prefix]
		if prefix < 3 {
			if !errors.Is(e, arena.ErrInjectedFailure) || v.Size() != 0 {
				t.Fatal("source retained growth prefix", prefix, e)
			}
		} else if e != nil || v.Size() != 5 {
			t.Fatal("completed resize", e)
		}
		if v.Capacity() != wantCap || b.Used() != base+header+uint64(wantCap)*width {
			t.Fatal("immediate backing charge", prefix, b.Used(), v.Capacity())
		}
		b.FailAfter(-1)
		v.Close()
		if b.Used() != base {
			t.Fatal("backing/header release")
		}
	}
	// Old+new backing is simultaneously charged at each source growth.
	m, b := integerManager(t)
	v, _ := NewScopedIntVector(m)
	v.Resize(2)
	before := b.Used()
	b.FailAfter(0)
	if e := v.Resize(3); !errors.Is(e, arena.ErrInjectedFailure) || v.Capacity() != 2 || b.Used() != before {
		t.Fatal("growth prepare", e)
	}
	b.FailAfter(-1)
	v.Close()
}

func TestScopedNumeralMoveAndAscendingCleanup(t *testing.T) {
	m, _ := integerManager(t)
	v, _ := NewScopedIntVector(m)
	defer v.Close()
	var x Int
	m.SetUint64(&x, 1<<63)
	v.Push(&x)
	v.Push(&x)
	old := v.data
	first := old[0].Cell
	second := old[1].Cell
	live := m.Cells.Stats().Live
	v.Push(&x)
	if old[0].Cell != 0 || old[1].Cell != 0 || old[0].Val != 1 || old[1].Val != 1 || v.data[0].Cell != first || v.data[1].Cell != second || m.Cells.Stats().Live != live+1 {
		t.Fatal("source element move ownership")
	}
	third := v.data[2].Cell
	v.Shrink(0)
	if m.Cells.Stats().Live != live-2 {
		t.Fatal("truncate immediate deletion")
	}
	// Source deletes ascending index. The allocator's LIFO recycle exposes the
	// last-deleted cell first; each freed payload was released exactly once.
	var a, b, c Int
	m.SetUint64(&a, 1<<63)
	m.SetUint64(&b, 1<<63)
	m.SetUint64(&c, 1<<63)
	if NumericSlot(a.Cell) != NumericSlot(third) || NumericSlot(b.Cell) != NumericSlot(second) || NumericSlot(c.Cell) != NumericSlot(first) {
		t.Fatal("ascending vector deletion order")
	}
	m.Del(&a)
	m.Del(&b)
	m.Del(&c)
	m.Del(&x)
}

func TestScopedNumeralScalarFailureAndAllThinOperations(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, b := rationalManager(t, synch)
		s, _ := NewScopedRational(m)
		defer s.Close()
		s.SetInt64(-17)
		three := NewRational()
		m.SetInt64(&three, 3)
		for _, f := range []func(*Rational) error{s.AddAssign, s.SubAssign, s.MulAssign, s.DivAssign, s.RemAssign} {
			if e := f(&three); e != nil {
				t.Fatal(e)
			}
		}
		if text, _ := s.Display(); text != "-2" {
			t.Fatal(text)
		} // exact rational / leaves -17, remainder -2
		s.SetInt64(3)
		for _, item := range []struct {
			f    func(*Rational) (bool, error)
			want bool
		}{{s.Equal, true}, {s.NotEqual, false}, {s.Less, false}, {s.Greater, false}, {s.LessEqual, true}, {s.GreaterEqual, true}} {
			v, e := item.f(&three)
			if e != nil || v != item.want {
				t.Fatal(v, e)
			}
		}
		for _, item := range []struct {
			f    func() (bool, error)
			want bool
		}{{s.IsZero, false}, {s.IsPos, true}, {s.IsNeg, false}, {s.IsNonpos, false}, {s.IsNonneg, true}} {
			v, e := item.f()
			if e != nil || v != item.want {
				t.Fatal(v, e)
			}
		}
		for _, f := range []func(*Rational) (*ScopedRational, error){s.Add, s.Sub, s.Mul, s.Div} {
			r, e := f(&three)
			if e != nil {
				t.Fatal(e)
			}
			r.Close()
		}
		abs, e := s.AbsCopy()
		if e != nil {
			t.Fatal(e)
		}
		abs.Neg()
		abs.Assign(s)
		abs.Swap(s)
		abs.SwapValue(&three)
		abs.Close()
		m.Del(&three)
		// Fill a genuinely cold allocation point rather than assuming a warmed
		// copy reserves. Header construction alone gives a deterministic boundary.
		live := m.Z.Cells.Stats().Live
		used := b.Used()
		b.FailAfter(0)
		if r, e := s.Clone(); r != nil || !errors.Is(e, arena.ErrInjectedFailure) || m.Z.Cells.Stats().Live != live || b.Used() != used {
			t.Fatal("clone header failure", r, e)
		}
		b.FailAfter(-1)
		fraction := NewRational()
		m.SetFraction(&fraction, 1, 2)
		e = s.RemAssign(&fraction)
		divPublicationAudit(t, m, &s.num, &fraction)
		if modularSourceDebug {
			if !errors.Is(e, ErrRationalNotInteger) {
				t.Fatal(e)
			}
			if text, _ := s.Display(); text != "3" {
				t.Fatal("source debug leaves scope unchanged", text)
			}
		} else {
			qopsMust(t, e)
			if text, _ := s.Display(); text != "0" {
				t.Fatal("release rem3 by raw numerator1", text)
			}
		}
		m.Del(&fraction)
		s.Close()
	}
}

func TestScopedNumeralWarmReuseAndLifetimeGuards(t *testing.T) {
	m, b := integerManager(t)
	v, _ := NewScopedIntVector(m)
	v.Resize(20)
	v.Reset()
	x := NewInt(7)
	s, _ := NewScopedInt(m)
	s.Set(&x)
	b.FailAfter(0)
	if n := testing.AllocsPerRun(100, func() {
		if e := v.Push(&x); e != nil {
			panic(e)
		}
		if e := v.Set(0, &x); e != nil {
			panic(e)
		}
		var out Int
		if e := v.Read(0, &out); e != nil {
			panic(e)
		}
		if e := v.Pop(); e != nil {
			panic(e)
		}
		if e := s.AddAssign(&x); e != nil {
			panic(e)
		}
		if e := s.Reset(); e != nil {
			panic(e)
		}
	}); n != 0 {
		t.Fatal("warm allocation", n)
	}
	b.FailAfter(-1)
	copy := *v
	if e := copy.Close(); !errors.Is(e, ErrScopedCopy) {
		t.Fatal(e)
	}
	if e := v.Pop(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if e := v.Shrink(1); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	var out Int
	if e := v.Read(1, &out); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	m.Close()
	if _, e := s.Get(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if e := s.Close(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("dead borrowed owner guard", e)
	}
	if e := v.Close(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("release independent cold storage after invalid manager lifetime", b.Used())
	}
	s.Close()
	v.Close()
}
func TestScopedNumeralScalarOwnershipAndOperators(t *testing.T) {
	m, _ := integerManager(t)
	a, e := NewScopedInt(m)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	a.SetInt64(-19)
	c, e := a.Clone()
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	v := NewInt(3)
	c.AddAssign(&v)
	c.SubAssign(&v)
	c.MulAssign(&v)
	c.DivAssign(&v)
	c.RemAssign(&v)
	if s, _ := c.Display(); s != "-1" {
		t.Fatal(s)
	}
	c.Assign(a)
	c.Neg()
	p, e := c.IsPos()
	if e != nil || !p {
		t.Fatal(p, e)
	}
	abs, e := a.AbsCopy()
	if e != nil {
		t.Fatal(e)
	}
	defer abs.Close()
	if s, _ := abs.Display(); s != "19" {
		t.Fatal(s)
	}
	y, e := a.Add(&v)
	if e != nil {
		t.Fatal(e)
	}
	defer y.Close()
	if s, _ := y.Display(); s != "-16" {
		t.Fatal(s)
	}
	if s, _ := a.Display(); s != "-19" {
		t.Fatal("arithmetic copy changed input")
	}
	c.Swap(a)
	if s, _ := a.Display(); s != "19" {
		t.Fatal(s)
	}
	a.SwapValue(&v)
	if s, _ := a.Display(); s != "3" || v.Val != 19 {
		t.Fatal(s, v)
	}
	a.Reset()
	if z, e := a.IsZero(); e != nil || !z {
		t.Fatal(z, e)
	}
	shallow := *a
	if _, e := shallow.Get(); !errors.Is(e, ErrScopedCopy) {
		t.Fatal("shallow owner copy", e)
	}
	other, _ := integerManager(t)
	foreign, _ := NewScopedInt(other)
	defer foreign.Close()
	if e := a.Assign(foreign); !errors.Is(e, ErrScopedOwner) {
		t.Fatal(e)
	}
	if arena.ValidateRecordType(reflect.TypeOf(Int{})) != nil || arena.ValidateRecordType(reflect.TypeOf(Rational{})) != nil {
		t.Fatal("hot numeral records must be pointer-free")
	}
}
func TestScopedNumeralVectorIndependentLifecycle(t *testing.T) {
	m, _ := integerManager(t)
	baseline := m.Cells.Stats().Live
	v, e := NewScopedIntVector(m)
	if e != nil {
		t.Fatal(e)
	}
	defer v.Close()
	rng := rand.New(rand.NewSource(1709))
	var model []int32
	capacity := uint32(0)
	grow := func(n uint32) {
		for capacity < n {
			if capacity == 0 {
				capacity = 2
			} else {
				capacity = (3*capacity + 1) / 2
			}
		}
	}
	for step := 0; step < 1000; step++ {
		switch rng.Intn(5) {
		case 0:
			x := NewInt(int32(rng.Intn(100) - 50))
			grow(uint32(len(model) + 1))
			if e := v.Push(&x); e != nil {
				t.Fatal(e)
			}
			model = append(model, x.Val)
		case 1:
			if len(model) > 0 {
				if e := v.Pop(); e != nil {
					t.Fatal(e)
				}
				model = model[:len(model)-1]
			}
		case 2:
			n := uint32(rng.Intn(20))
			grow(n)
			if e := v.Resize(n); e != nil {
				t.Fatal(e)
			}
			if int(n) <= len(model) {
				model = model[:n]
			} else {
				model = append(model, make([]int32, int(n)-len(model))...)
			}
		case 3:
			v.Reset()
			model = nil
		case 4:
			if len(model) > 0 {
				i := rng.Intn(len(model))
				x := NewInt(int32(rng.Intn(100)))
				v.Set(uint32(i), &x)
				model[i] = x.Val
			}
		}
		if v.Size() != uint32(len(model)) || v.Capacity() != capacity {
			t.Fatal("source size/capacity", step, v.Size(), v.Capacity(), len(model), capacity)
		}
		for i, w := range model {
			var out Int
			if e := v.Read(uint32(i), &out); e != nil || out.Val != w {
				t.Fatal(step, i, out, e)
			}
			m.Del(&out)
		}
	}
	v.Reset()
	if m.Cells.Stats().Live != baseline {
		t.Fatal("vector leak")
	}
}
func TestScopedRationalVectorAndScalarLifetimes(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		base := m.Z.Cells.Stats().Live
		a, e := NewScopedRational(m)
		if e != nil {
			t.Fatal(e)
		}
		q, _ := a.Get()
		if e = m.SetFraction(q, -9223372036854775807, 18446744073709551615); e != nil {
			t.Fatal(e)
		}
		copy, e := a.Clone()
		if e != nil {
			t.Fatal(e)
		}
		v, e := NewScopedRationalVector(m)
		if e != nil {
			t.Fatal(e)
		}
		v.Push(q)
		v.Push(q)
		if m.Z.Cells.Stats().Live <= base {
			t.Fatal("real rational payloads absent")
		}
		v.Resize(5)
		for i := uint32(2); i < 5; i++ {
			out := NewRational()
			v.Read(i, &out)
			if s := rationalString(t, m, &out); s != "0" {
				t.Fatal("source numeral(0) default", s)
			}
			m.Del(&out)
		}
		copy.Reset()
		out := NewRational()
		v.Read(0, &out)
		if s := rationalString(t, m, &out); s != "-9223372036854775807/18446744073709551615" {
			t.Fatal("deep push", s)
		}
		m.Del(&out)
		v.Shrink(1)
		v.Pop()
		v.Reset()
		a.Close()
		copy.Close()
		v.Close()
		if m.Z.Cells.Stats().Live != base {
			t.Fatal("rational destructor balance", synch, m.Z.Cells.Stats().Live, base)
		}
	}
}
func TestScopedNumeralHeadersFailuresAndSourcePushPrefix(t *testing.T) {
	// This action-timing fixture uses the actual direct source profile so the
	// copied big numeral requires physical preparation despite warmed pool slots.
	m, b := mpzPublicationDirectOwner(t)
	base := b.Used()
	b.FailAfter(0)
	if s, e := NewScopedInt(m); s != nil || !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != base {
		t.Fatal(s, e)
	}
	b.FailAfter(-1)
	v, e := NewScopedIntVector(m)
	if e != nil {
		t.Fatal(e)
	}
	v.Resize(2)
	v.Shrink(0)
	var big Int
	m.SetString(&big, "340282366920938463463374607431768211455")
	// Parsing can warm recyclable cells. Occupy every slot before injection so
	// the new deep-copy cell must actually reserve a page, rather than guessing
	// that the next allocation is cold.
	var fillers []Int
	for m.Cells.Stats().Live+m.Cells.Stats().Retired < m.Cells.Stats().Slots {
		var filler Int
		if e := m.SetUint64(&filler, 1<<63); e != nil {
			t.Fatal(e)
		}
		fillers = append(fillers, filler)
	}
	roots := []*Int{&big}
	for i := range fillers {
		roots = append(roots, &fillers[i])
	}
	live := m.Cells.Stats().Live
	wordsLive := m.Words.Stats().Live
	usedBefore := b.Used()
	b.FailAfter(0)
	if e = v.Push(&big); !errors.Is(e, arena.ErrInjectedFailure) || v.Size() != 1 || v.Capacity() != 2 {
		t.Fatal("source default appended before copy", e, v.Size())
	}
	integerPhysicalAudit(t, m, roots...)
	if m.Cells.Stats().Live != live {
		t.Fatal("failed push leaked a temporary cell")
	}
	if m.Words.Stats().Live != wordsLive || b.Used() != usedBefore {
		t.Fatal("failed push leaked live payload or backing charge")
	}
	b.FailAfter(-1)
	var out Int
	v.Read(0, &out)
	if out.Val != 0 || out.Cell != 0 {
		t.Fatal("failed push not default")
	}
	v.Close()
	m.Del(&big)
	for i := range fillers {
		m.Del(&fillers[i])
	}
	m.Close()
	if b.Used() != 0 {
		t.Fatal("header/backing cleanup")
	}
}

func TestScopedRationalVectorIndependentLifecycleAndFailures(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, b := rationalManager(t, synch)
		small := NewRational()
		big := NewRational()
		m.SetFraction(&small, -2, 3)
		m.SetFraction(&big, -9223372036854775807, 18446744073709551615)
		// Warm caller-output formatting workspace before fixed live-graph oracle.
		rationalString(t, m, &big)
		baseCells := m.Z.Cells.Stats().Live
		baseWords := m.Z.Words.Stats().Live
		v, e := NewScopedRationalVector(m)
		if e != nil {
			t.Fatal(e)
		}
		rng := rand.New(rand.NewSource(933))
		var model []uint8
		capacity := uint32(0)
		grow := func(n uint32) {
			for capacity < n {
				if capacity == 0 {
					capacity = 2
				} else {
					capacity = (3*capacity + 1) / 2
				}
			}
		}
		for step := 0; step < 400; step++ {
			switch rng.Intn(5) {
			case 0:
				code := uint8(rng.Intn(2) + 1)
				num := &small
				if code == 2 {
					num = &big
				}
				grow(uint32(len(model) + 1))
				if e := v.Push(num); e != nil {
					t.Fatal(e)
				}
				model = append(model, code)
			case 1:
				if len(model) > 0 {
					v.Pop()
					model = model[:len(model)-1]
				}
			case 2:
				n := uint32(rng.Intn(16))
				grow(n)
				if e := v.Resize(n); e != nil {
					t.Fatal(e)
				}
				if int(n) <= len(model) {
					model = model[:n]
				} else {
					model = append(model, make([]uint8, int(n)-len(model))...)
				}
			case 3:
				v.Reset()
				model = nil
			case 4:
				if len(model) > 0 {
					i := rng.Intn(len(model))
					v.Set(uint32(i), &big)
					model[i] = 2
				}
			}
			if v.Size() != uint32(len(model)) || v.Capacity() != capacity {
				t.Fatal("rational vector size/capacity")
			}
			bigCount := uint32(0)
			for i, code := range model {
				want := "0"
				if code == 1 {
					want = "-2/3"
				}
				if code == 2 {
					want = "-9223372036854775807/18446744073709551615"
					bigCount++
				}
				out := NewRational()
				if e := v.Read(uint32(i), &out); e != nil {
					t.Fatal(e)
				}
				if text := rationalString(t, m, &out); text != want {
					t.Fatal(step, i, text, want)
				}
				m.Del(&out)
			}
			if m.Z.Cells.Stats().Live != baseCells+bigCount*2 || m.Z.Words.Stats().Live != baseWords {
				t.Fatal("immediate rational cell/retained formatter lifecycle oracle", step, m.Z.Cells.Stats().Live, baseCells, bigCount)
			}
		}
		v.Reset()
		v.Resize(2)
		v.Shrink(0)
		// Occupy existing free cell slots so the deep copy is genuinely cold.
		var fillers []Int
		for m.Z.Cells.Stats().Live+m.Z.Cells.Stats().Retired < m.Z.Cells.Stats().Slots {
			var x Int
			m.Z.SetUint64(&x, 1<<63)
			fillers = append(fillers, x)
		}
		live := m.Z.Cells.Stats().Live
		payloads := m.Z.Words.Stats().Live
		b.FailAfter(0)
		if e := v.Push(&big); !numericPhysicalReservationRefusal(e) || v.Size() != 1 {
			t.Fatal("rational source append prefix", e)
		}
		roots := []*Int{&small.Num, &small.Den, &big.Num, &big.Den, &v.data[0].Num, &v.data[0].Den}
		for i := range fillers {
			roots = append(roots, &fillers[i])
		}
		qopsAuditLiveOwnership(t, m, roots)
		setterPairPrefix(t, m, &v.data[0], "0", "1", roundPublicationDecode(t, m.Z, &big.Num), roundPublicationDecode(t, m.Z, &big.Den), true)
		published := uint32(0)
		for _, n := range []*Int{&v.data[0].Num, &v.data[0].Den} {
			if n.Cell != 0 && !m.Z.IsFailureRef(n.Cell) {
				published++
			}
		}
		if m.Z.Cells.Stats().Live != live+published || m.Z.Words.Stats().Live != payloads {
			t.Fatal("actual source copy descriptors", m.Z.Cells.Stats().Live, live, published)
		}
		// The source object allocation can complete before descriptor refusal;
		// retained physical objects/metadata must remain fully charged and named.
		vectorCharge := uint64(reflect.TypeOf(ScopedRationalVector{}).Size()) + uint64(cap(v.data))*uint64(reflect.TypeOf(Rational{}).Size())
		if b.Used() != m.HeaderBytes+integerActualOwnedBytes(m.Z)+vectorCharge {
			t.Fatal("actual rational vector physical graph", b.Used(), integerActualOwnedBytes(m.Z), m.HeaderBytes, vectorCharge)
		}
		b.FailAfter(-1)
		qopsMust(t, m.Z.CleanupPendingAllocation()) // explicit unassigned-object cleanup, after immediate source audit
		for i := range fillers {
			m.Z.Del(&fillers[i])
		}
		beforeClose := b.Used()
		integerBefore := integerActualOwnedBytes(m.Z)
		coldCharge := uint64(reflect.TypeOf(ScopedRationalVector{}).Size()) + uint64(cap(v.data))*uint64(reflect.TypeOf(Rational{}).Size())
		if e := v.Close(); e != nil {
			t.Fatal("actual failed-component vector Close", e)
		}
		if beforeClose-b.Used() != coldCharge+integerBefore-integerActualOwnedBytes(m.Z) {
			t.Fatal("vector physical header/backing refund", beforeClose, b.Used(), coldCharge)
		}
		if m.Z.Cells.Stats().Live != baseCells || m.Z.Words.Stats().Live != baseWords {
			t.Fatal("rational close live graph")
		}
		m.Del(&small)
		m.Del(&big)
	}
}

func TestScopedNumeralDeepCloneFailureCleanupAndBackingPeak(t *testing.T) {
	failed, passed := 0, 0
	for prefix := int64(0); prefix < 24; prefix++ {
		m, b := rationalManager(t, true)
		s, _ := NewScopedRational(m)
		num, _ := s.Get()
		m.SetFraction(num, -9223372036854775807, 18446744073709551615)
		var fillers []Int
		for m.Z.Cells.Stats().Live+m.Z.Cells.Stats().Retired < m.Z.Cells.Stats().Slots {
			var x Int
			if e := m.Z.SetUint64(&x, 1<<63); e != nil {
				t.Fatal(e)
			}
			fillers = append(fillers, x)
		}
		cells := m.Z.Cells.Stats().Live
		words := m.Z.Words.Stats().Live
		b.FailAfter(prefix)
		copy, e := s.Clone()
		roots := []*Int{&num.Num, &num.Den}
		for i := range fillers {
			roots = append(roots, &fillers[i])
		}
		if copy != nil {
			roots = append(roots, &copy.num.Num, &copy.num.Den)
		}
		qopsAuditLiveOwnership(t, m, roots)
		ownerCharge := uint64(reflect.TypeOf(ScopedRational{}).Size())
		if copy != nil {
			ownerCharge *= 2
		}
		if b.Used() != m.HeaderBytes+integerActualOwnedBytes(m.Z)+ownerCharge {
			t.Fatal("actual clone owners and physical roots", prefix, b.Used(), ownerCharge)
		}
		if e != nil {
			failed++
			if !numericPhysicalReservationRefusal(e) || copy != nil || m.Z.Cells.Stats().Live != cells || m.Z.Words.Stats().Live != words {
				t.Fatal("clone immediate cleanup", prefix, e)
			}
		} else {
			passed++
			b.FailAfter(-1)
			if e := copy.Close(); e != nil {
				t.Fatal(e)
			}
			if m.Z.Cells.Stats().Live != cells || m.Z.Words.Stats().Live != words {
				t.Fatal("clone close immediate graph")
			}
		}
		b.FailAfter(-1)
		qopsMust(t, m.Z.CleanupPendingAllocation()) // explicit release after pre-cleanup audit
		for i := range fillers {
			m.Z.Del(&fillers[i])
		}
		s.Close()
		m.Close()
		if b.Used() != 0 {
			t.Fatal("clone header unwind", prefix, b.Used())
		}
	}
	if failed == 0 || passed == 0 {
		t.Fatal("clone sweep endpoints", failed, passed)
	}
	baseManager, baseBudget := integerManager(t)
	base := baseBudget.Used()
	baseManager.Close()
	header := uint64(reflect.TypeOf(ScopedIntVector{}).Size())
	width := uint64(reflect.TypeOf(Int{}).Size())
	// Caller-owned charged backing is reserved after manager construction:
	// construction has its own larger transient peaks, independent of this
	// vector's precise old+new boundary. The actual buffer keeps those separate.
	callerBacking := make([]byte, 1<<16)
	peak := base + uint64(len(callerBacking)) + header + 5*width
	for _, limit := range []uint64{peak - 1, peak} {
		b := arena.NewBudget(limit)
		m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
		if e != nil {
			t.Fatal(e)
		}
		if e := b.Reserve(uint64(len(callerBacking))); e != nil {
			t.Fatal(e)
		}
		v, e := NewScopedIntVector(m)
		if e != nil {
			t.Fatal(e)
		}
		if e = v.Resize(2); e != nil {
			t.Fatal(e)
		}
		before := b.Used()
		cellLive := m.Cells.Stats().Live
		wordLive := m.Words.Stats().Live
		e = v.Resize(3)
		if limit < peak {
			if !errors.Is(e, arena.ErrBudget) || v.Size() != 2 || v.Capacity() != 2 || b.Used() != before || m.Cells.Stats().Live != cellLive || m.Words.Stats().Live != wordLive {
				t.Fatal("backing peak failure", e)
			}
		} else if e != nil || v.Size() != 3 || v.Capacity() != 3 || b.Used() != base+uint64(len(callerBacking))+header+3*width {
			t.Fatal("exact backing peak success", e, b.Used())
		}
		v.Close()
		b.Release(uint64(len(callerBacking)))
		m.Close()
		if b.Used() != 0 {
			t.Fatal("peak lifetime")
		}
	}
	if _, e := scopedCapacity(int(uint64(^uint32(0))), uint64(reflect.TypeOf(Int{}).Size())); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("source capacity overflow", e)
	}
}
